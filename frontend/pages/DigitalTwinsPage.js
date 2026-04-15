(() => {
	const root = window.MimirApp = window.MimirApp || {};
	const pages = root.pages = root.pages || {};
	const { apiCall, deriveStorageConfigLabel, notify, confirmAction } = root.lib;
	const { ProjectContext } = root.context;
	const { Button, FormField, Modal, Table, Tabs } = root.components.primitives;

	const operatorOptions = [
		{ value: 'gt', label: 'Greater Than' },
		{ value: 'gte', label: 'Greater Than or Equal' },
		{ value: 'lt', label: 'Less Than' },
		{ value: 'lte', label: 'Less Than or Equal' },
		{ value: 'eq', label: 'Equals' },
		{ value: 'ne', label: 'Not Equal' },
	];

	const approvalModeOptions = [
		{ value: 'automatic', label: 'Automatic export' },
		{ value: 'manual', label: 'Manual approval required' },
	];

	const workspaceTabs = ['Control Center', 'Knowledge Graph', 'Automation', 'Scenarios'];

	function getApprovalBadgeClass(status) {
		return status === 'approved' || status === 'not_required'
			? 'status-active'
			: status === 'rejected'
			? 'status-failed'
			: 'status-pending';
	}

	function getExecutionBadgeClass(status) {
		return status === 'queued'
			? 'status-active'
			: status === 'failed' || status === 'rejected'
			? 'status-failed'
			: status === 'pending_approval'
			? 'status-pending'
			: 'status-idle';
	}

	function createTwinPath(projectId, twinId = '', extra = {}) {
		const params = new URLSearchParams();
		if (projectId) params.set('project_id', projectId);
		Object.entries(extra || {}).forEach(([key, value]) => {
			if (value === undefined || value === null || value === '') return;
			params.set(key, String(value));
		});
		const suffix = twinId || '';
		const query = params.toString();
		return `/api/digital-twins${suffix}${query ? `?${query}` : ''}`;
	}

	function createEmptyActionForm() {
		return {
			name: '',
			description: '',
			enabled: true,
			model_id: '',
			entity_type: '',
			attribute: '',
			operator: 'gt',
			threshold: '0',
			pipeline_id: '',
			approval_mode: 'automatic',
			parameters: '{"alert_severity":"high"}',
		};
	}

	function createEmptyAutomationForm() {
		return {
			name: '',
			description: '',
			enabled: true,
			trigger_type: 'pipeline_completed',
			trigger_config: '{"pipeline_types":["ingestion"]}',
		};
	}

	function MetricCards({ cards }) {
		return (
			<div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(180px, 1fr))', gap: '12px', marginBottom: '20px' }}>
				{cards.map(card => (
					<div key={card.label} className="card" style={{ background: 'linear-gradient(180deg, rgba(255,255,255,0.03), transparent)' }}>
						<div className="section-panel-copy" style={{ marginBottom: '8px', fontSize: '0.82rem' }}>{card.label}</div>
						<div style={{ fontSize: '1.8rem', fontWeight: 700, color: card.tone }}>{card.value}</div>
					</div>
				))}
			</div>
		);
	}

	function JsonPreview({ value, empty }) {
		if (!value) return <div className="section-panel-copy">{empty || 'No data.'}</div>;
		return <div className="json-display"><pre>{JSON.stringify(value, null, 2)}</pre></div>;
	}

	function ControlPanel({ twin, latestRun, latestAlert, pendingApprovals, sourceLabels, ontology, mlModels, exportPipelines }) {
		return (
			<div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(280px, 1fr))', gap: '16px', marginBottom: '20px' }}>
				<div className="section-panel section-panel--neutral">
					<div className="section-panel-header">
						<div>
							<h3 className="section-panel-title">Source and Schema</h3>
							<p className="section-panel-copy">Storage sources feed the twin. The ontology defines entity types and relationships.</p>
						</div>
					</div>
					<div><strong>Status:</strong> {twin.status}</div>
					<div><strong>Ontology:</strong> {ontology?.name || twin.ontology_id}</div>
					<div><strong>Configured sources:</strong> {sourceLabels.length || 0}</div>
					{sourceLabels.length ? (
						<ul style={{ margin: '10px 0 0', paddingLeft: '1.2rem' }}>
							{sourceLabels.map(label => <li key={label}>{label}</li>)}
						</ul>
					) : <p className="section-panel-copy" style={{ marginTop: '10px' }}>No storage sources wired yet. Add project storage configs and attach them in the twin definition before running sync.</p>}
				</div>
				<div className="section-panel section-panel--neutral">
					<div className="section-panel-header">
						<div>
							<h3 className="section-panel-title">Operational Flow</h3>
							<p className="section-panel-copy">Queue Source Sync refreshes entity state. Process Twin then runs insight generation, alert evaluation, and export actions.</p>
						</div>
					</div>
					<div><strong>Last sync:</strong> {twin.last_sync_at ? new Date(twin.last_sync_at).toLocaleString() : 'Never'}</div>
					<div><strong>Latest processing:</strong> {latestRun ? `${latestRun.status} · ${new Date(latestRun.requested_at).toLocaleString()}` : 'No runs yet'}</div>
					<div><strong>Latest alert:</strong> {latestAlert ? `${latestAlert.severity} · ${latestAlert.title}` : 'No alerts yet'}</div>
					<div><strong>Pending approvals:</strong> {pendingApprovals}</div>
				</div>
				<div className="section-panel section-panel--neutral">
					<div className="section-panel-header">
						<div>
							<h3 className="section-panel-title">Connected Intelligence</h3>
							<p className="section-panel-copy">Models score the graph. Actions map alert conditions to output pipelines. Automations trigger processing from upstream pipeline activity.</p>
						</div>
					</div>
					<div><strong>Trained ML models:</strong> {mlModels.length}</div>
					<div><strong>Export pipelines:</strong> {exportPipelines.length}</div>
					<div><strong>Prediction cache TTL:</strong> {twin.config?.prediction_cache_ttl || '—'} seconds</div>
					<div><strong>Predictions enabled:</strong> {twin.config?.enable_predictions ? 'Yes' : 'No'}</div>
				</div>
			</div>
		);
	}

	function TwinWorkspaceHeader({ twin, onBack, onProcess, onSync, onNewAutomation, onNewAction, onQuery }) {
		return (
			<div style={{ marginBottom: '20px' }}>
				<Button label="← Back to List" onClick={onBack} variant="secondary" />
				<h3 style={{ color: 'var(--accent)', marginTop: '16px', marginBottom: '8px' }}>{twin.name}</h3>
				<p className="section-panel-copy" style={{ margin: 0 }}>{twin.description || 'Ontology-grounded operational workspace for this project.'}</p>
				<div className="inline-actions" style={{ marginTop: '12px', flexWrap: 'wrap' }}>
					<Button label="Process Twin" onClick={onProcess} variant="secondary" />
					<Button label="Queue Source Sync" onClick={onSync} variant="secondary" />
					<Button label="+ New Automation" onClick={onNewAutomation} variant="secondary" />
					<Button label="+ New Action" onClick={onNewAction} variant="secondary" />
					<Button label="Query Twin" onClick={onQuery} variant="secondary" />
				</div>
			</div>
		);
	}

	pages.DigitalTwinsPage = function DigitalTwinsPage() {
		const { activeProject, projects } = React.useContext(ProjectContext);
		const [twins, setTwins] = React.useState([]);
		const [loading, setLoading] = React.useState(true);
		const [showModal, setShowModal] = React.useState(false);
		const [showActionModal, setShowActionModal] = React.useState(false);
		const [showAutomationModal, setShowAutomationModal] = React.useState(false);
		const [showQueryModal, setShowQueryModal] = React.useState(false);
		const [selectedTab, setSelectedTab] = React.useState('Control Center');
		const [selectedTwin, setSelectedTwin] = React.useState(null);
		const [entities, setEntities] = React.useState([]);
		const [scenarios, setScenarios] = React.useState([]);
		const [actions, setActions] = React.useState([]);
		const [insights, setInsights] = React.useState([]);
		const [alerts, setAlerts] = React.useState([]);
		const [runs, setRuns] = React.useState([]);
		const [automations, setAutomations] = React.useState([]);
		const [queryResult, setQueryResult] = React.useState(null);
		const [twinsError, setTwinsError] = React.useState('');
		const [detailsError, setDetailsError] = React.useState('');
		const [ontologies, setOntologies] = React.useState([]);
		const [mlModels, setMlModels] = React.useState([]);
		const [pipelines, setPipelines] = React.useState([]);
		const [storageConfigs, setStorageConfigs] = React.useState([]);
		const [formData, setFormData] = React.useState({ name: '', project_id: '', ontology_id: '', description: '', storage_ids: [] });
		const [actionForm, setActionForm] = React.useState(createEmptyActionForm());
		const [automationForm, setAutomationForm] = React.useState(createEmptyAutomationForm());
		const [queryForm, setQueryForm] = React.useState({ query: '' });

		const resetActionForm = React.useCallback(() => setActionForm(createEmptyActionForm()), []);
		const resetAutomationForm = React.useCallback(() => setAutomationForm(createEmptyAutomationForm()), []);

		const loadProjectAssets = React.useCallback(async (projectId) => {
			if (!projectId) {
				setOntologies([]);
				setMlModels([]);
				setPipelines([]);
				setStorageConfigs([]);
				return;
			}
			const [ontologyData, mlModelData, pipelineData, storageData] = await Promise.all([
				apiCall(`/api/ontologies?project_id=${projectId}`).catch(() => []),
				apiCall(`/api/ml-models?project_id=${projectId}`).catch(() => []),
				apiCall(`/api/pipelines?project_id=${projectId}`).catch(() => []),
				apiCall(`/api/storage/configs?project_id=${projectId}`).catch(() => []),
			]);
			setOntologies(ontologyData || []);
			setMlModels(mlModelData || []);
			setPipelines(pipelineData || []);
			setStorageConfigs(storageData || []);
		}, []);

		React.useEffect(() => {
			if (!activeProject) {
				setFormData({ name: '', project_id: '', ontology_id: '', description: '', storage_ids: [] });
				setSelectedTwin(null);
				loadProjectAssets('');
				return;
			}
			setFormData(prev => ({ ...prev, project_id: activeProject.id, ontology_id: '', storage_ids: [] }));
			loadProjectAssets(activeProject.id);
		}, [activeProject, loadProjectAssets]);

		const loadTwins = React.useCallback(async () => {
			const projectId = activeProject?.id || '';
			if (!projectId) {
				setTwins([]);
				setTwinsError('');
				setLoading(false);
				return;
			}
			setLoading(true);
			setTwinsError('');
			try {
				const data = await apiCall(createTwinPath(projectId));
				setTwins(data || []);
			} catch (error) {
				setTwins([]);
				setTwinsError(error.message || 'Failed to load digital twins.');
			} finally {
				setLoading(false);
			}
		}, [activeProject?.id]);

		const loadTwinDetails = React.useCallback(async (twin) => {
			const projectId = activeProject?.id;
			if (!projectId || !twin?.id) return;
			setDetailsError('');
			try {
				const [entitiesData, scenariosData, actionsData, insightsData, alertsData, runsData, automationsData] = await Promise.all([
					apiCall(createTwinPath(projectId, `/${twin.id}/entities`)),
					apiCall(createTwinPath(projectId, `/${twin.id}/scenarios`)),
					apiCall(createTwinPath(projectId, `/${twin.id}/actions`)),
					apiCall(`/api/insights?project_id=${projectId}`).catch(() => []),
					apiCall(createTwinPath(projectId, `/${twin.id}/alerts`, { limit: 100 })),
					apiCall(createTwinPath(projectId, `/${twin.id}/runs`, { limit: 50 })),
					apiCall(createTwinPath(projectId, `/${twin.id}/automations`)),
				]);
				setEntities(entitiesData || []);
				setScenarios(scenariosData || []);
				setActions(actionsData || []);
				setInsights(insightsData || []);
				setAlerts(alertsData || []);
				setRuns(runsData || []);
				setAutomations(automationsData || []);
			} catch (error) {
				setDetailsError(error.message || 'Failed to load digital twin details.');
			}
		}, [activeProject?.id]);

		React.useEffect(() => {
			loadTwins();
		}, [loadTwins]);

		React.useEffect(() => {
			if (!selectedTwin) return;
			setSelectedTab('Control Center');
			loadTwinDetails(selectedTwin);
		}, [selectedTwin, loadTwinDetails]);

		const handleSubmit = async (e) => {
			e.preventDefault();
			try {
				await apiCall('/api/digital-twins', {
					method: 'POST',
					body: JSON.stringify({
						name: formData.name,
						project_id: formData.project_id,
						ontology_id: formData.ontology_id,
						description: formData.description,
						config: {
							storage_ids: formData.storage_ids,
							enable_predictions: true,
							prediction_cache_ttl: 1800,
						},
					}),
				});
				setShowModal(false);
				setFormData({ name: '', project_id: activeProject?.id || '', ontology_id: '', description: '', storage_ids: [] });
				notify({ tone: 'success', message: 'Digital twin created.' });
				loadTwins();
			} catch (error) {
				notify({ tone: 'error', message: `Failed to create digital twin: ${error.message}` });
			}
		};

		const handleDelete = async (id) => {
			const confirmed = await confirmAction({
				title: 'Delete digital twin',
				message: 'Delete this twin and its materialized graph, history, scenarios, predictions, actions, alerts, and automations?',
				confirmLabel: 'Delete twin',
				variant: 'danger',
			});
			if (!confirmed) return;
			try {
				await apiCall(createTwinPath(activeProject?.id || '', `/${id}`), { method: 'DELETE' });
				if (selectedTwin?.id === id) setSelectedTwin(null);
				notify({ tone: 'success', message: 'Digital twin deleted.' });
				loadTwins();
			} catch (error) {
				notify({ tone: 'error', message: `Failed to delete digital twin: ${error.message}` });
			}
		};

		const handleSync = async (id) => {
			try {
				const result = await apiCall(createTwinPath(activeProject?.id || '', `/${id}/sync`), { method: 'POST', body: JSON.stringify({}) });
				notify({ tone: 'success', message: result?.message || `Digital twin sync queued (${result?.work_task_id || 'work task created'})` });
				loadTwins();
				if (selectedTwin?.id === id) loadTwinDetails(selectedTwin);
			} catch (error) {
				notify({ tone: 'error', message: `Failed to sync digital twin: ${error.message}` });
			}
		};

		const handleProcessTwin = async (id) => {
			try {
				const result = await apiCall(createTwinPath(activeProject?.id || '', `/${id}/runs`), { method: 'POST', body: JSON.stringify({}) });
				notify({ tone: 'success', message: `Twin processing queued (${result?.id || 'run created'})` });
				loadTwins();
				if (selectedTwin?.id === id) loadTwinDetails(selectedTwin);
			} catch (error) {
				notify({ tone: 'error', message: `Failed to process digital twin: ${error.message}` });
			}
		};

		const handleCreateAction = async (e) => {
			e.preventDefault();
			try {
				const parameters = actionForm.parameters.trim() ? JSON.parse(actionForm.parameters) : {};
				let threshold;
				try { threshold = JSON.parse(actionForm.threshold); } catch { threshold = actionForm.threshold; }
				await apiCall(createTwinPath(activeProject?.id || '', `/${selectedTwin.id}/actions`), {
					method: 'POST',
					body: JSON.stringify({
						name: actionForm.name,
						description: actionForm.description,
						enabled: actionForm.enabled,
						condition: {
							operator: actionForm.operator,
							threshold,
							model_id: actionForm.model_id || undefined,
							entity_type: actionForm.entity_type || undefined,
							attribute: actionForm.attribute || undefined,
						},
						trigger: {
							pipeline_id: actionForm.pipeline_id,
							approval_mode: actionForm.approval_mode,
							parameters,
						},
					}),
				});
				setShowActionModal(false);
				resetActionForm();
				notify({ tone: 'success', message: 'Digital twin action created.' });
				loadTwinDetails(selectedTwin);
			} catch (error) {
				notify({ tone: 'error', message: `Failed to create action: ${error.message}` });
			}
		};

		const handleCreateAutomation = async (e) => {
			e.preventDefault();
			try {
				const triggerConfig = automationForm.trigger_config.trim() ? JSON.parse(automationForm.trigger_config) : {};
				await apiCall(createTwinPath(activeProject?.id || '', `/${selectedTwin.id}/automations`), {
					method: 'POST',
					body: JSON.stringify({
						name: automationForm.name,
						description: automationForm.description,
						enabled: automationForm.enabled,
						trigger_type: automationForm.trigger_type,
						trigger_config: triggerConfig,
						action_type: 'process_twin',
					}),
				});
				setShowAutomationModal(false);
				resetAutomationForm();
				notify({ tone: 'success', message: 'Twin automation created.' });
				loadTwinDetails(selectedTwin);
			} catch (error) {
				notify({ tone: 'error', message: `Failed to create automation: ${error.message}` });
			}
		};

		const handleDeleteAction = async (actionId) => {
			const confirmed = await confirmAction({
				title: 'Delete digital twin action',
				message: 'Delete this action? Future matching alerts will stop using it.',
				confirmLabel: 'Delete action',
				variant: 'danger',
			});
			if (!confirmed) return;
			try {
				await apiCall(createTwinPath(activeProject?.id || '', `/${selectedTwin.id}/actions/${actionId}`), { method: 'DELETE' });
				notify({ tone: 'success', message: 'Action deleted.' });
				loadTwinDetails(selectedTwin);
			} catch (error) {
				notify({ tone: 'error', message: `Failed to delete action: ${error.message}` });
			}
		};

		const handleDeleteAutomation = async (automationId) => {
			const confirmed = await confirmAction({
				title: 'Delete twin automation',
				message: 'Delete this automation? Automatic processing from that trigger will stop.',
				confirmLabel: 'Delete automation',
				variant: 'danger',
			});
			if (!confirmed) return;
			try {
				await apiCall(createTwinPath(activeProject?.id || '', `/${selectedTwin.id}/automations/${automationId}`), { method: 'DELETE' });
				notify({ tone: 'success', message: 'Automation deleted.' });
				loadTwinDetails(selectedTwin);
			} catch (error) {
				notify({ tone: 'error', message: `Failed to delete automation: ${error.message}` });
			}
		};

		const handleReviewAlert = async (alertId, decision) => {
			const note = decision === 'approve' ? 'Approved from Digital Twin workspace' : 'Rejected from Digital Twin workspace';
			try {
				await apiCall(createTwinPath(activeProject?.id || '', `/${selectedTwin.id}/alerts/${alertId}/approval`), {
					method: 'POST',
					body: JSON.stringify({ decision, actor: 'frontend', note }),
				});
				notify({ tone: 'success', message: decision === 'approve' ? 'Alert action approved.' : 'Alert action rejected.' });
				loadTwinDetails(selectedTwin);
			} catch (error) {
				notify({ tone: 'error', message: `Failed to ${decision} alert action: ${error.message}` });
			}
		};

		const handleQuery = async (e) => {
			e.preventDefault();
			try {
				const result = await apiCall(createTwinPath(activeProject?.id || '', `/${selectedTwin.id}/query`), { method: 'POST', body: JSON.stringify({ query: queryForm.query }) });
				setQueryResult(result);
			} catch (error) {
				notify({ tone: 'error', message: `Failed to execute query: ${error.message}` });
			}
		};

		const projectOptions = projects.map(p => ({ value: p.id, label: p.name }));
		const ontologyOptions = ontologies.map(o => ({ value: o.id, label: o.name }));
		const storageOptions = storageConfigs.map(c => ({ value: c.id, label: deriveStorageConfigLabel(c) }));
		const outputPipelineOptions = pipelines.filter(p => p.type === 'output').map(p => ({ value: p.id, label: p.name }));
		const activeTwinOntology = ontologies.find(o => o.id === selectedTwin?.ontology_id);
		const activeTwinModels = mlModels.filter(model => model.ontology_id === selectedTwin?.ontology_id || model.project_id === selectedTwin?.project_id);
		const sourceLabels = (selectedTwin?.config?.storage_ids || []).map(id => {
			const config = storageConfigs.find(item => item.id === id);
			return config ? deriveStorageConfigLabel(config) : id;
		});
		const pendingApprovals = alerts.filter(alert => alert.approval_status === 'pending').length;
		const latestRun = runs[0];
		const latestAlert = alerts[0];

		const twinColumns = [
			{ key: 'name', label: 'Twin' },
			{ key: 'ontology_id', label: 'Ontology', render: row => ontologies.find(item => item.id === row.ontology_id)?.name || row.ontology_id },
			{ key: 'storage_ids', label: 'Sources', render: row => (row.config?.storage_ids || []).length },
			{ key: 'status', label: 'Status', render: row => <span className={`status-badge status-${row.status}`}>{row.status}</span> },
			{ key: 'last_sync_at', label: 'Last Sync', render: row => row.last_sync_at ? new Date(row.last_sync_at).toLocaleString() : 'Never' },
		];

		const entityColumns = [
			{ key: 'id', label: 'Entity ID' },
			{ key: 'type', label: 'Type' },
			{ key: 'attributes', label: 'Attributes', render: row => <pre style={{ margin: 0, fontSize: '0.75rem', whiteSpace: 'pre-wrap' }}>{JSON.stringify(row.attributes || {}, null, 2)}</pre> },
			{ key: 'updated_at', label: 'Updated', render: row => new Date(row.updated_at).toLocaleString() },
		];
		const insightColumns = [
			{ key: 'type', label: 'Type' },
			{ key: 'severity', label: 'Severity', render: row => <span className={`status-badge status-${row.severity}`}>{row.severity}</span> },
			{ key: 'confidence', label: 'Confidence', render: row => Number(row.confidence || 0).toFixed(2) },
			{ key: 'explanation', label: 'Explanation', render: row => row.explanation || '—' },
			{ key: 'created_at', label: 'Created', render: row => new Date(row.created_at).toLocaleString() },
		];
		const alertColumns = [
			{ key: 'severity', label: 'Severity', render: row => <span className={`status-badge status-${row.severity}`}>{row.severity}</span> },
			{ key: 'approval_status', label: 'Approval', render: row => <span className={`status-badge ${getApprovalBadgeClass(row.approval_status)}`}>{row.approval_status || 'not_required'}</span> },
			{ key: 'execution_status', label: 'Execution', render: row => <span className={`status-badge ${getExecutionBadgeClass(row.execution_status)}`}>{row.execution_status || 'not_applicable'}</span> },
			{ key: 'title', label: 'Title' },
			{ key: 'requested_export_pipeline_id', label: 'Requested Pipeline', render: row => row.requested_export_pipeline_id || '—' },
			{ key: 'created_at', label: 'Created', render: row => new Date(row.created_at).toLocaleString() },
		];
		const runColumns = [
			{ key: 'status', label: 'Status', render: row => <span className={`status-badge status-${row.status}`}>{row.status}</span> },
			{ key: 'trigger_type', label: 'Trigger' },
			{ key: 'requested_at', label: 'Requested', render: row => new Date(row.requested_at).toLocaleString() },
			{ key: 'completed_at', label: 'Completed', render: row => row.completed_at ? new Date(row.completed_at).toLocaleString() : '—' },
			{ key: 'insight_count', label: 'Insights', render: row => row.metrics?.insight_count ?? '—' },
			{ key: 'alert_count', label: 'Alerts', render: row => row.metrics?.alert_count ?? '—' },
		];
		const automationColumns = [
			{ key: 'name', label: 'Automation' },
			{ key: 'enabled', label: 'Enabled', render: row => row.enabled ? 'Yes' : 'No' },
			{ key: 'trigger_type', label: 'Trigger' },
			{ key: 'updated_at', label: 'Updated', render: row => new Date(row.updated_at).toLocaleString() },
		];
		const actionColumns = [
			{ key: 'name', label: 'Action' },
			{ key: 'enabled', label: 'Enabled', render: row => row.enabled ? 'Yes' : 'No' },
			{ key: 'condition', label: 'Condition', render: row => {
				const condition = row.condition || {};
				const scope = condition.attribute ? `${condition.entity_type || 'Any entity'}.${condition.attribute}` : (condition.model_id ? `Model ${condition.model_id}` : 'Condition');
				return `${scope} ${condition.operator || ''} ${JSON.stringify(condition.threshold)}`;
			} },
			{ key: 'pipeline_id', label: 'Pipeline', render: row => row.trigger?.pipeline_id || '—' },
			{ key: 'approval_mode', label: 'Execution', render: row => row.trigger?.approval_mode === 'manual' ? 'Manual approval' : 'Automatic' },
		];
		const scenarioColumns = [
			{ key: 'name', label: 'Scenario' },
			{ key: 'description', label: 'Description' },
			{ key: 'base_state', label: 'Base State' },
			{ key: 'created_at', label: 'Created', render: row => new Date(row.created_at).toLocaleDateString() },
		];

		const summaryCards = [
			{ label: 'Entities', value: entities.length, tone: 'var(--accent)' },
			{ label: 'Insights', value: insights.length, tone: '#4cc9f0' },
			{ label: 'Alert Events', value: alerts.length, tone: '#ff7b72' },
			{ label: 'Pending Approvals', value: pendingApprovals, tone: '#34d399' },
		];

		const renderWorkspace = () => {
			switch (selectedTab) {
			case 'Control Center':
				return (
					<>
						<ControlPanel
							twin={selectedTwin}
							latestRun={latestRun}
							latestAlert={latestAlert}
							pendingApprovals={pendingApprovals}
							sourceLabels={sourceLabels}
							ontology={activeTwinOntology}
							mlModels={activeTwinModels}
							exportPipelines={outputPipelineOptions}
						/>
						<div style={{ display: 'grid', gridTemplateColumns: '1.2fr 1fr', gap: '16px', marginBottom: '20px' }}>
							<div className="section-panel section-panel--neutral">
								<div className="section-panel-header"><div><h3 className="section-panel-title">Recent Processing Runs</h3><p className="section-panel-copy">Track the sync → insights → alerts workflow.</p></div></div>
								<Table columns={runColumns} data={runs.slice(0, 5)} emptyState="No processing runs yet." />
							</div>
							<div className="section-panel section-panel--neutral">
							<div className="section-panel section-panel--neutral">
								<div className="section-panel-header"><div><h3 className="section-panel-title">Recent Alerts</h3><p className="section-panel-copy">Manual approvals and automatic export queue outcomes are both surfaced here before downstream pipelines run.</p></div></div>
								<Table columns={alertColumns} data={alerts.slice(0, 5)} emptyState="No alert events yet." actions={(row) => row.approval_status === 'pending' ? <><Button label="Approve" onClick={() => handleReviewAlert(row.id, 'approve')} variant="secondary" /><Button label="Reject" onClick={() => handleReviewAlert(row.id, 'reject')} variant="danger" /></> : null} />
							</div>
						</div>
						<div className="section-panel section-panel--neutral">
							<div className="section-panel-header"><div><h3 className="section-panel-title">Generated Insights</h3><p className="section-panel-copy">Project-level insights tied to the same sources feeding this twin.</p></div></div>
							<Table columns={insightColumns} data={insights.slice(0, 10)} emptyState="No insights have been generated for this twin yet." />
						</div>
					</>
				);
			case 'Knowledge Graph':
				return (
					<>
						<div className="section-panel section-panel--neutral" style={{ marginBottom: '20px' }}>
							<div className="section-panel-header"><div><h3 className="section-panel-title">Materialized Entity Graph</h3><p className="section-panel-copy">Entities are synchronized from storage through ontology-driven shaping and relationship wiring.</p></div></div>
							<Table columns={entityColumns} data={entities} emptyState="No entities are currently materialized in this twin." />
						</div>
						<div className="section-panel section-panel--neutral">
							<div className="section-panel-header"><div><h3 className="section-panel-title">Query Result</h3><p className="section-panel-copy">Use Query Twin to inspect graph slices and verify ontology mappings.</p></div></div>
							<JsonPreview value={queryResult} empty="No query executed yet." />
						</div>
					</>
				);
			case 'Automation':
				return (
					<div style={{ display: 'grid', gridTemplateColumns: '1fr', gap: '16px' }}>
						<div className="section-panel section-panel--neutral">
							<div className="section-panel-header"><div><h3 className="section-panel-title">Alert Actions</h3><p className="section-panel-copy">Map twin conditions and ML model outputs to export pipelines.</p></div></div>
							<Table columns={actionColumns} data={actions} emptyState="No actions are configured for this twin." actions={(row) => <Button label="Delete" onClick={() => handleDeleteAction(row.id)} variant="danger" />} />
						</div>
						<div className="section-panel section-panel--neutral">
							<div className="section-panel-header"><div><h3 className="section-panel-title">Processing Automations</h3><p className="section-panel-copy">Trigger twin processing from upstream ingestion or manual lifecycle events.</p></div></div>
							<Table columns={automationColumns} data={automations} emptyState="No automations are configured for this twin." actions={(row) => <Button label="Delete" onClick={() => handleDeleteAutomation(row.id)} variant="danger" />} />
						</div>
					</div>
				);
			case 'Scenarios':
				return (
					<div style={{ display: 'grid', gridTemplateColumns: '1fr 320px', gap: '16px' }}>
						<div className="section-panel section-panel--neutral">
							<div className="section-panel-header"><div><h3 className="section-panel-title">What-if Scenarios</h3><p className="section-panel-copy">Scenario predictions reuse the project’s trained ML models against the twin’s current graph.</p></div></div>
							<Table columns={scenarioColumns} data={scenarios} emptyState="No scenarios have been created for this twin." />
						</div>
						<div className="section-panel section-panel--neutral">
							<div className="section-panel-header"><div><h3 className="section-panel-title">Connected Assets</h3><p className="section-panel-copy">Upstream and downstream dependencies affecting this twin.</p></div></div>
							<div className="form-group"><label>Ontology</label><div className="field-static">{activeTwinOntology?.name || selectedTwin.ontology_id}</div></div>
							<div className="form-group"><label>ML Models</label><div className="field-static">{activeTwinModels.length}</div></div>
							<div className="form-group"><label>Export Pipelines</label><div className="field-static">{outputPipelineOptions.length}</div></div>
							<div className="form-group"><label>Storage Sources</label><div className="field-static">{sourceLabels.length}</div></div>
						</div>
					</div>
				);
			default:
				return null;
			}
		};

		return (
			<div className="content-section">
				<div className="section-header">
					<h2>Digital Twins</h2>
					<Button label="+ New Digital Twin" onClick={() => setShowModal(true)} />
				</div>

				{activeProject ? <div className="page-notice"><strong>Project scope:</strong> digital twins orchestrate storage ingestion, ontology shaping, ML-backed predictions, and pipeline-driven actions for {activeProject.name}.</div> : null}
				{twinsError ? <div className="error-message">{twinsError}</div> : null}

				{selectedTwin ? (
					<>
						<TwinWorkspaceHeader
							twin={selectedTwin}
							onBack={() => setSelectedTwin(null)}
							onProcess={() => handleProcessTwin(selectedTwin.id)}
							onSync={() => handleSync(selectedTwin.id)}
							onNewAutomation={() => setShowAutomationModal(true)}
							onNewAction={() => setShowActionModal(true)}
							onQuery={() => setShowQueryModal(true)}
						/>
						{detailsError ? <div className="error-message">{detailsError}</div> : null}
						<MetricCards cards={summaryCards} />
						<Tabs tabs={workspaceTabs} activeTab={selectedTab} onTabChange={setSelectedTab} />
						{renderWorkspace()}
					</>
				) : (
					loading ? <div className="loading">Loading digital twins...</div> : <Table columns={twinColumns} data={twins} emptyState={activeProject ? 'No digital twins exist for this project yet.' : 'Select a project to inspect digital twins.'} actions={(row) => <><Button label="Workspace" onClick={() => setSelectedTwin(row)} variant="secondary" /><Button label="Process" onClick={() => handleProcessTwin(row.id)} variant="secondary" /><Button label="Sync" onClick={() => handleSync(row.id)} variant="secondary" /><Button label="Delete" onClick={() => handleDelete(row.id)} variant="danger" /></>} />
				)}

				<Modal open={showModal} onClose={() => setShowModal(false)} title="Create New Digital Twin">
					<form onSubmit={handleSubmit}>
						<div className="form-grid">
							<FormField label="Twin Name" value={formData.name} onChange={(v) => setFormData({ ...formData, name: v })} required />
							<FormField label="Project" type="select" value={formData.project_id} onChange={(v) => setFormData({ ...formData, project_id: v, ontology_id: '', storage_ids: [] })} options={projectOptions} required disabled />
						</div>
						<div className="form-grid">
							<FormField label="Ontology" type="select" value={formData.ontology_id} onChange={(v) => setFormData({ ...formData, ontology_id: v })} options={ontologyOptions} required />
						</div>
						<div className="form-group">
							<label>Storage Sources</label>
							{storageOptions.length ? (
								<div style={{ display: 'grid', gap: '8px' }}>
									{storageOptions.map(option => {
										const checked = formData.storage_ids.includes(option.value);
										return (
											<label key={option.value} className="checkbox-row">
												<input
													type="checkbox"
													checked={checked}
													onChange={event => setFormData({
														...formData,
														storage_ids: event.target.checked
															? [...formData.storage_ids, option.value]
															: formData.storage_ids.filter(id => id !== option.value),
													})}
												/>
												{option.label}
											</label>
										);
									})}
								</div>
							) : <div className="section-panel-copy">No storage configs exist for this project yet.</div>}
						</div>
						<FormField label="Description" type="textarea" value={formData.description} onChange={(v) => setFormData({ ...formData, description: v })} />
						<Button type="submit" label="Create Digital Twin" />
					</form>
				</Modal>

				<Modal open={showAutomationModal} onClose={() => { setShowAutomationModal(false); resetAutomationForm(); }} title="Create Twin Automation">
					<form onSubmit={handleCreateAutomation}>
						<div className="form-grid">
							<FormField label="Automation Name" value={automationForm.name} onChange={(v) => setAutomationForm({ ...automationForm, name: v })} required />
							<FormField label="Trigger Type" type="select" value={automationForm.trigger_type} onChange={(v) => setAutomationForm({ ...automationForm, trigger_type: v })} options={[{ value: 'pipeline_completed', label: 'Pipeline Completed' }, { value: 'manual', label: 'Manual' }]} required />
						</div>
						<FormField label="Description" type="textarea" value={automationForm.description} onChange={(v) => setAutomationForm({ ...automationForm, description: v })} />
						<FormField label="Trigger Config (JSON)" type="textarea" value={automationForm.trigger_config} onChange={(v) => setAutomationForm({ ...automationForm, trigger_config: v })} placeholder='{"pipeline_types":["ingestion"]}' />
						<label className="checkbox-row"><input type="checkbox" checked={automationForm.enabled} onChange={e => setAutomationForm({ ...automationForm, enabled: e.target.checked })} />Enabled</label>
						<Button type="submit" label="Create Twin Automation" />
					</form>
				</Modal>

				<Modal open={showActionModal} onClose={() => { setShowActionModal(false); resetActionForm(); }} title="Create New Action">
					<form onSubmit={handleCreateAction}>
						<div className="form-grid">
							<FormField label="Action Name" value={actionForm.name} onChange={(v) => setActionForm({ ...actionForm, name: v })} required />
							<FormField label="Trigger Pipeline" type="select" value={actionForm.pipeline_id} onChange={(v) => setActionForm({ ...actionForm, pipeline_id: v })} options={outputPipelineOptions} required />
						</div>
						<FormField label="Description" type="textarea" value={actionForm.description} onChange={(v) => setActionForm({ ...actionForm, description: v })} />
						<div className="form-grid">
							<FormField label="Model (optional)" type="select" value={actionForm.model_id} onChange={(v) => setActionForm({ ...actionForm, model_id: v })} options={activeTwinModels.map(model => ({ value: model.id, label: model.name }))} />
							<FormField label="Entity Type (optional)" value={actionForm.entity_type} onChange={(v) => setActionForm({ ...actionForm, entity_type: v })} placeholder="e.g. Product" />
						</div>
						<div className="form-grid">
							<FormField label="Attribute (optional)" value={actionForm.attribute} onChange={(v) => setActionForm({ ...actionForm, attribute: v })} placeholder="e.g. stock_level" />
							<FormField label="Operator" type="select" value={actionForm.operator} onChange={(v) => setActionForm({ ...actionForm, operator: v })} options={operatorOptions} required />
						</div>
						<div className="form-grid">
							<FormField label="Threshold" value={actionForm.threshold} onChange={(v) => setActionForm({ ...actionForm, threshold: v })} placeholder="0" required />
							<FormField label="Execution Mode" type="select" value={actionForm.approval_mode} onChange={(v) => setActionForm({ ...actionForm, approval_mode: v })} options={approvalModeOptions} required />
						</div>
						<FormField label="Trigger Parameters (JSON)" type="textarea" value={actionForm.parameters} onChange={(v) => setActionForm({ ...actionForm, parameters: v })} placeholder='{"alert_severity":"high","alert_category":"stockout"}' />
						<label className="checkbox-row"><input type="checkbox" checked={actionForm.enabled} onChange={e => setActionForm({ ...actionForm, enabled: e.target.checked })} />Enabled</label>
						<Button type="submit" label="Create Action" />
					</form>
				</Modal>

				<Modal open={showQueryModal} onClose={() => setShowQueryModal(false)} title="Query Digital Twin">
					<form onSubmit={handleQuery}>
						<FormField label="SPARQL SELECT Query" type="textarea" value={queryForm.query} onChange={(v) => setQueryForm({ ...queryForm, query: v })} placeholder="SELECT ?entity ?type WHERE { ?entity a ?type } LIMIT 25" required />
						<Button type="submit" label="Execute Query" />
					</form>
					<JsonPreview value={queryResult} empty="No query executed yet." />
				</Modal>
			</div>
		);
	};
})();

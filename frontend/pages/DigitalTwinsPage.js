(() => {
	const root = window.MimirApp = window.MimirApp || {};
	const pages = root.pages = root.pages || {};
	const { apiCall, notify, confirmAction } = root.lib;
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

	function getApprovalBadgeClass(status) {
		return status === 'approved' || status === 'not_required'
			? 'status-active'
			: status === 'rejected'
			? 'status-failed'
			: 'status-pending';
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

	function TwinWorkspaceHeader({ twin, onBack, onProcess, onSync, onNewAutomation, onNewAction, onQuery }) {
		return (
			<div style={{ marginBottom: '20px' }}>
				<Button label="← Back to List" onClick={onBack} variant="secondary" />
				<h3 style={{ color: 'var(--accent)', marginTop: '16px', marginBottom: '8px' }}>{twin.name} - Twin Workspace</h3>
				<p className="section-panel-copy" style={{ margin: 0 }}>{twin.description || 'Ontology-grounded operational workspace for this project.'}</p>
				<div className="inline-actions" style={{ marginTop: '12px' }}>
					<Button label="Process Twin" onClick={onProcess} variant="secondary" />
					<Button label="Queue Source Sync" onClick={onSync} variant="secondary" />
					<Button label="+ New Automation" onClick={onNewAutomation} variant="secondary" />
					<Button label="+ New Action" onClick={onNewAction} variant="secondary" />
					<Button label="Query Twin" onClick={onQuery} variant="secondary" />
				</div>
			</div>
		);
	}

	function OverviewCards({ twin, latestRun, latestAlert, pendingApprovals }) {
		return (
			<div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(280px, 1fr))', gap: '16px', marginBottom: '20px' }}>
				<div className="card">
					<h3 className="section-panel-title">Current State</h3>
					<div><strong>Status:</strong> {twin.status}</div>
					<div><strong>Ontology:</strong> {twin.ontology_id}</div>
					<div><strong>Last Sync:</strong> {twin.last_sync_at ? new Date(twin.last_sync_at).toLocaleString() : 'Never'}</div>
					<div><strong>Configured Storage Sources:</strong> {twin.config?.storage_ids?.length || 0}</div>
					<div><strong>Pending Manual Approvals:</strong> {pendingApprovals}</div>
					<p className="section-panel-copy" style={{ marginTop: '10px' }}>Process Twin runs the full workflow: sync scoped sources, generate insights, evaluate alert events, and either trigger export actions automatically or queue them for approval. Queue Source Sync refreshes source-backed entities only.</p>
				</div>
				<div className="card">
					<h3 className="section-panel-title">Latest Processing Run</h3>
					{latestRun ? (
						<>
							<div><strong>Status:</strong> {latestRun.status}</div>
							<div><strong>Trigger:</strong> {latestRun.trigger_type}</div>
							<div><strong>Requested:</strong> {new Date(latestRun.requested_at).toLocaleString()}</div>
							<div><strong>Insights:</strong> {latestRun.metrics?.insight_count ?? 0}</div>
							<div><strong>Alerts:</strong> {latestRun.metrics?.alert_count ?? 0}</div>
							<div><strong>Pending Approvals:</strong> {latestRun.metrics?.pending_approval_count ?? 0}</div>
						</>
					) : <div className="section-panel-copy">No processing runs yet.</div>}
				</div>
				<div className="card">
					<h3 className="section-panel-title">Latest Alert Event</h3>
					{latestAlert ? (
						<>
							<div><strong>Severity:</strong> {latestAlert.severity}</div>
							<div><strong>Title:</strong> {latestAlert.title}</div>
							<div><strong>Approval:</strong> <span className={`status-badge ${getApprovalBadgeClass(latestAlert.approval_status)}`}>{latestAlert.approval_status || 'not_required'}</span></div>
							<div><strong>Requested Pipeline:</strong> {latestAlert.requested_export_pipeline_id || '—'}</div>
							<div><strong>Created:</strong> {new Date(latestAlert.created_at).toLocaleString()}</div>
						</>
					) : <div className="section-panel-copy">No alert events emitted yet.</div>}
				</div>
			</div>
		);
	}

	function QueryResultPanel({ queryResult }) {
		if (!queryResult) return null;
		return (
			<div style={{ marginTop: '16px' }}>
				<h3 style={{ color: 'var(--accent)' }}>Query Result</h3>
				<div className="json-display"><pre>{JSON.stringify(queryResult, null, 2)}</pre></div>
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
		const [selectedTab, setSelectedTab] = React.useState('Overview');
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
		const [formData, setFormData] = React.useState({ name: '', project_id: '', ontology_id: '', description: '' });
		const [actionForm, setActionForm] = React.useState(createEmptyActionForm());
		const [automationForm, setAutomationForm] = React.useState(createEmptyAutomationForm());
		const [queryForm, setQueryForm] = React.useState({ query: '' });

		const resetActionForm = React.useCallback(() => setActionForm(createEmptyActionForm()), []);
		const resetAutomationForm = React.useCallback(() => setAutomationForm(createEmptyAutomationForm()), []);

		React.useEffect(() => {
			if (!activeProject) {
				setOntologies([]);
				setMlModels([]);
				setPipelines([]);
				setFormData({ name: '', project_id: '', ontology_id: '', description: '' });
				return;
			}
			setFormData(prev => ({ ...prev, project_id: activeProject.id }));
			Promise.all([
				apiCall(`/api/ontologies?project_id=${activeProject.id}`).catch(() => []),
				apiCall(`/api/ml-models?project_id=${activeProject.id}`).catch(() => []),
				apiCall(`/api/pipelines?project_id=${activeProject.id}`).catch(() => []),
			]).then(([ontologyData, mlModelData, pipelineData]) => {
				setOntologies(ontologyData || []);
				setMlModels(mlModelData || []);
				setPipelines((pipelineData || []).filter(p => p.type === 'output' || p.type === 'ingestion'));
			});
		}, [activeProject]);

		const loadTwins = async () => {
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
				const data = await apiCall(`/api/digital-twins?project_id=${projectId}`);
				setTwins(data || []);
			} catch (error) {
				setTwins([]);
				setTwinsError(error.message || 'Failed to load digital twins.');
			} finally {
				setLoading(false);
			}
		};

		const loadTwinDetails = async (twinId) => {
			if (!activeProject?.id) return;
			setDetailsError('');
			try {
				const [entitiesData, scenariosData, actionsData, insightsData, alertsData, runsData, automationsData] = await Promise.all([
					apiCall(`/api/digital-twins/${twinId}/entities`),
					apiCall(`/api/digital-twins/${twinId}/scenarios`),
					apiCall(`/api/digital-twins/${twinId}/actions`),
					apiCall(`/api/insights?project_id=${activeProject.id}`),
					apiCall(`/api/digital-twins/${twinId}/alerts?limit=100`),
					apiCall(`/api/digital-twins/${twinId}/runs?limit=50`),
					apiCall(`/api/digital-twins/${twinId}/automations`),
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
		};

		React.useEffect(() => {
			loadTwins();
		}, [activeProject]);

		React.useEffect(() => {
			if (!selectedTwin) return;
			setSelectedTab('Overview');
			loadTwinDetails(selectedTwin.id);
		}, [selectedTwin, activeProject?.id]);

		const handleSubmit = async (e) => {
			e.preventDefault();
			try {
				await apiCall('/api/digital-twins', { method: 'POST', body: JSON.stringify(formData) });
				setShowModal(false);
				setFormData({ name: '', project_id: activeProject?.id || '', ontology_id: '', description: '' });
				notify({ tone: 'success', message: 'Digital twin created.' });
				loadTwins();
			} catch (error) {
				notify({ tone: 'error', message: `Failed to create digital twin: ${error.message}` });
			}
		};

		const handleDelete = async (id) => {
			const confirmed = await confirmAction({
				title: 'Delete digital twin',
				message: 'Delete this digital twin? Historical runs and alert records will no longer be reachable from the UI.',
				confirmLabel: 'Delete twin',
				variant: 'danger',
			});
			if (!confirmed) return;
			try {
				await apiCall(`/api/digital-twins/${id}`, { method: 'DELETE' });
				if (selectedTwin?.id === id) setSelectedTwin(null);
				notify({ tone: 'success', message: 'Digital twin deleted.' });
				loadTwins();
			} catch (error) {
				notify({ tone: 'error', message: `Failed to delete digital twin: ${error.message}` });
			}
		};

		const handleSync = async (id) => {
			try {
				const result = await apiCall(`/api/digital-twins/${id}/sync`, { method: 'POST', body: JSON.stringify({}) });
				notify({ tone: 'success', message: result?.message || `Digital twin sync queued (${result?.work_task_id || 'work task created'})` });
				loadTwins();
				if (selectedTwin?.id === id) loadTwinDetails(id);
			} catch (error) {
				notify({ tone: 'error', message: `Failed to sync digital twin: ${error.message}` });
			}
		};

		const handleProcessTwin = async (id) => {
			try {
				const result = await apiCall(`/api/digital-twins/${id}/runs`, { method: 'POST', body: JSON.stringify({}) });
				notify({ tone: 'success', message: `Twin processing queued (${result?.id || 'run created'})` });
				if (selectedTwin?.id === id) loadTwinDetails(id);
				loadTwins();
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
				await apiCall(`/api/digital-twins/${selectedTwin.id}/actions`, {
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
				loadTwinDetails(selectedTwin.id);
			} catch (error) {
				notify({ tone: 'error', message: `Failed to create action: ${error.message}` });
			}
		};

		const handleCreateAutomation = async (e) => {
			e.preventDefault();
			try {
				const triggerConfig = automationForm.trigger_config.trim() ? JSON.parse(automationForm.trigger_config) : {};
				await apiCall(`/api/digital-twins/${selectedTwin.id}/automations`, {
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
				loadTwinDetails(selectedTwin.id);
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
				await apiCall(`/api/digital-twins/${selectedTwin.id}/actions/${actionId}`, { method: 'DELETE' });
				notify({ tone: 'success', message: 'Action deleted.' });
				loadTwinDetails(selectedTwin.id);
			} catch (error) {
				notify({ tone: 'error', message: `Failed to delete action: ${error.message}` });
			}
		};

		const handleDeleteAutomation = async (automationId) => {
			const confirmed = await confirmAction({
				title: 'Delete twin automation',
				message: 'Delete this automation? Automatic twin processing from that trigger will stop.',
				confirmLabel: 'Delete automation',
				variant: 'danger',
			});
			if (!confirmed) return;
			try {
				await apiCall(`/api/digital-twins/${selectedTwin.id}/automations/${automationId}`, { method: 'DELETE' });
				notify({ tone: 'success', message: 'Automation deleted.' });
				loadTwinDetails(selectedTwin.id);
			} catch (error) {
				notify({ tone: 'error', message: `Failed to delete automation: ${error.message}` });
			}
		};

		const handleReviewAlert = async (alertId, decision) => {
			const note = decision === 'approve' ? 'Approved from Digital Twin workspace' : 'Rejected from Digital Twin workspace';
			try {
				await apiCall(`/api/digital-twins/${selectedTwin.id}/alerts/${alertId}/approval`, {
					method: 'POST',
					body: JSON.stringify({ decision, actor: 'frontend', note }),
				});
				notify({ tone: 'success', message: decision === 'approve' ? 'Alert action approved.' : 'Alert action rejected.' });
				loadTwinDetails(selectedTwin.id);
			} catch (error) {
				notify({ tone: 'error', message: `Failed to ${decision} alert action: ${error.message}` });
			}
		};

		const handleQuery = async (e) => {
			e.preventDefault();
			try {
				const result = await apiCall(`/api/digital-twins/${selectedTwin.id}/query`, { method: 'POST', body: JSON.stringify({ query: queryForm.query }) });
				setQueryResult(result);
			} catch (error) {
				notify({ tone: 'error', message: `Failed to execute query: ${error.message}` });
			}
		};

		const projectOptions = projects.map(p => ({ value: p.id, label: p.name }));
		const ontologyOptions = ontologies.map(o => ({ value: o.id, label: o.name }));
		const mlModelOptions = mlModels.map(m => ({ value: m.id, label: m.name }));
		const pipelineOptions = pipelines.filter(p => p.type === 'output').map(p => ({ value: p.id, label: p.name }));
		const pendingApprovals = alerts.filter(alert => alert.approval_status === 'pending').length;
		const latestRun = runs[0];
		const latestAlert = alerts[0];

		const twinColumns = [
			{ key: 'id', label: 'ID' },
			{ key: 'name', label: 'Name' },
			{ key: 'description', label: 'Description' },
			{ key: 'ontology_id', label: 'Ontology ID' },
			{ key: 'last_sync_at', label: 'Last Sync', render: (row) => row.last_sync_at ? new Date(row.last_sync_at).toLocaleString() : 'Never' },
			{ key: 'status', label: 'Status', render: (row) => <span className={`status-badge status-${row.status}`}>{row.status}</span> },
		];
		const entityColumns = [
			{ key: 'id', label: 'Entity ID' },
			{ key: 'type', label: 'Type' },
			{ key: 'attributes', label: 'Attributes', render: (row) => JSON.stringify(row.attributes || {}).substring(0, 80) + '...' },
			{ key: 'updated_at', label: 'Updated', render: (row) => new Date(row.updated_at).toLocaleString() },
		];
		const scenarioColumns = [
			{ key: 'id', label: 'Scenario ID' },
			{ key: 'name', label: 'Name' },
			{ key: 'description', label: 'Description' },
			{ key: 'created_at', label: 'Created', render: (row) => new Date(row.created_at).toLocaleDateString() },
		];
		const actionColumns = [
			{ key: 'id', label: 'Action ID' },
			{ key: 'name', label: 'Name' },
			{ key: 'enabled', label: 'Enabled', render: (row) => row.enabled ? 'Yes' : 'No' },
			{ key: 'condition', label: 'Condition', render: (row) => {
				const condition = row.condition || {};
				const scope = condition.attribute ? `${condition.entity_type || 'Any entity'}.${condition.attribute}` : (condition.model_id ? `Model ${condition.model_id}` : 'Condition');
				return `${scope} ${condition.operator || ''} ${JSON.stringify(condition.threshold)}`;
			} },
			{ key: 'pipeline_id', label: 'Trigger Pipeline', render: (row) => row.trigger?.pipeline_id || '—' },
			{ key: 'approval_mode', label: 'Execution', render: (row) => row.trigger?.approval_mode === 'manual' ? 'Manual approval' : 'Automatic' },
			{ key: 'created_at', label: 'Created', render: (row) => new Date(row.created_at).toLocaleDateString() },
		];
		const insightColumns = [
			{ key: 'type', label: 'Type' },
			{ key: 'severity', label: 'Severity', render: (row) => <span className={`status-badge status-${row.severity}`}>{row.severity}</span> },
			{ key: 'confidence', label: 'Confidence', render: (row) => Number(row.confidence || 0).toFixed(2) },
			{ key: 'explanation', label: 'Explanation', render: (row) => row.explanation || '—' },
			{ key: 'created_at', label: 'Created', render: (row) => new Date(row.created_at).toLocaleString() },
		];
		const alertColumns = [
			{ key: 'severity', label: 'Severity', render: (row) => <span className={`status-badge status-${row.severity}`}>{row.severity}</span> },
			{ key: 'approval_status', label: 'Approval', render: (row) => <span className={`status-badge ${getApprovalBadgeClass(row.approval_status)}`}>{row.approval_status || 'not_required'}</span> },
			{ key: 'category', label: 'Category' },
			{ key: 'title', label: 'Title' },
			{ key: 'message', label: 'Message' },
			{ key: 'requested_export_pipeline_id', label: 'Requested Pipeline', render: (row) => row.requested_export_pipeline_id || '—' },
			{ key: 'triggered_export_pipeline_id', label: 'Triggered Pipeline', render: (row) => row.triggered_export_pipeline_id || '—' },
			{ key: 'created_at', label: 'Created', render: (row) => new Date(row.created_at).toLocaleString() },
		];
		const runColumns = [
			{ key: 'status', label: 'Status', render: (row) => <span className={`status-badge status-${row.status}`}>{row.status}</span> },
			{ key: 'trigger_type', label: 'Trigger' },
			{ key: 'requested_at', label: 'Requested', render: (row) => new Date(row.requested_at).toLocaleString() },
			{ key: 'completed_at', label: 'Completed', render: (row) => row.completed_at ? new Date(row.completed_at).toLocaleString() : '—' },
			{ key: 'insight_count', label: 'Insights', render: (row) => row.metrics?.insight_count ?? '—' },
			{ key: 'alert_count', label: 'Alerts', render: (row) => row.metrics?.alert_count ?? '—' },
			{ key: 'pending_approval_count', label: 'Pending Approvals', render: (row) => row.metrics?.pending_approval_count ?? 0 },
			{ key: 'triggered_action_count', label: 'Triggered Actions', render: (row) => row.metrics?.triggered_action_count ?? '—' },
		];
		const automationColumns = [
			{ key: 'name', label: 'Name' },
			{ key: 'enabled', label: 'Enabled', render: (row) => row.enabled ? 'Yes' : 'No' },
			{ key: 'trigger_type', label: 'Trigger' },
			{ key: 'action_type', label: 'Action' },
			{ key: 'updated_at', label: 'Updated', render: (row) => new Date(row.updated_at).toLocaleString() },
		];

		const summaryCards = [
			{ label: 'Entities', value: entities.length, tone: 'var(--accent)' },
			{ label: 'Insights', value: insights.length, tone: '#4cc9f0' },
			{ label: 'Alert Events', value: alerts.length, tone: '#ff7b72' },
			{ label: 'Pending Approvals', value: pendingApprovals, tone: '#34d399' },
		];

		const renderWorkspaceTable = () => {
			switch (selectedTab) {
			case 'Insights':
				return <Table columns={insightColumns} data={insights} emptyState="No insights have been generated for this twin yet." />;
			case 'Alerts':
				return <Table columns={alertColumns} data={alerts} emptyState="No alert events have been emitted for this twin yet." actions={(row) => row.approval_status === 'pending' ? <><Button label="Approve" onClick={() => handleReviewAlert(row.id, 'approve')} variant="secondary" /><Button label="Reject" onClick={() => handleReviewAlert(row.id, 'reject')} variant="danger" /></> : null} />;
			case 'Automations':
				return <Table columns={automationColumns} data={automations} emptyState="No automations are configured for this twin." actions={(row) => <Button label="Delete" onClick={() => handleDeleteAutomation(row.id)} variant="danger" />} />;
			case 'Entities':
				return <Table columns={entityColumns} data={entities} emptyState="No entities are currently materialized in this twin." />;
			case 'Actions':
				return <Table columns={actionColumns} data={actions} emptyState="No actions are configured for this twin." actions={(row) => <Button label="Delete" onClick={() => handleDeleteAction(row.id)} variant="danger" />} />;
			case 'Scenarios':
				return <Table columns={scenarioColumns} data={scenarios} emptyState="No scenarios have been created for this twin." />;
			case 'Runs':
				return <Table columns={runColumns} data={runs} emptyState="No processing runs have been requested for this twin yet." />;
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
						{selectedTab === 'Overview' ? <OverviewCards twin={selectedTwin} latestRun={latestRun} latestAlert={latestAlert} pendingApprovals={pendingApprovals} /> : null}
						<Tabs tabs={['Overview', 'Insights', 'Alerts', 'Automations', 'Entities', 'Actions', 'Scenarios', 'Runs']} activeTab={selectedTab} onTabChange={setSelectedTab} />
						{renderWorkspaceTable()}
					</>
				) : (
					loading ? <div className="loading">Loading digital twins...</div> : <Table columns={twinColumns} data={twins} emptyState={activeProject ? 'No digital twins exist for this project yet.' : 'Select a project to inspect digital twins.'} actions={(row) => <><Button label="View" onClick={() => setSelectedTwin(row)} variant="secondary" /><Button label="Process" onClick={() => handleProcessTwin(row.id)} variant="secondary" /><Button label="Sync Sources" onClick={() => handleSync(row.id)} variant="secondary" /><Button label="Delete" onClick={() => handleDelete(row.id)} variant="danger" /></>} />
				)}

				<Modal open={showModal} onClose={() => setShowModal(false)} title="Create New Digital Twin">
					<form onSubmit={handleSubmit}>
						<div className="form-grid">
							<FormField label="Twin Name" value={formData.name} onChange={(v) => setFormData({ ...formData, name: v })} required />
							<FormField label="Project" type="select" value={formData.project_id} onChange={(v) => setFormData({ ...formData, project_id: v })} options={projectOptions} required />
						</div>
						<div className="form-grid">
							<FormField label="Ontology" type="select" value={formData.ontology_id} onChange={(v) => setFormData({ ...formData, ontology_id: v })} options={ontologyOptions} required />
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
							<FormField label="Trigger Pipeline" type="select" value={actionForm.pipeline_id} onChange={(v) => setActionForm({ ...actionForm, pipeline_id: v })} options={pipelineOptions} required />
						</div>
						<FormField label="Description" type="textarea" value={actionForm.description} onChange={(v) => setActionForm({ ...actionForm, description: v })} />
						<div className="form-grid">
							<FormField label="Model (optional)" type="select" value={actionForm.model_id} onChange={(v) => setActionForm({ ...actionForm, model_id: v })} options={mlModelOptions} />
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
					<QueryResultPanel queryResult={queryResult} />
				</Modal>
			</div>
		);
	};
})();

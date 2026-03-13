(() => {
	const root = window.MimirApp = window.MimirApp || {};
	const pages = root.pages = root.pages || {};
	const { apiCall } = root.lib;
	const { ProjectContext } = root.context;
	const { Button, FormField, Modal, Table, Tabs } = root.components.primitives;

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
		const [ontologies, setOntologies] = React.useState([]);
		const [mlModels, setMlModels] = React.useState([]);
		const [pipelines, setPipelines] = React.useState([]);
		const [formData, setFormData] = React.useState({
			name: '',
			project_id: '',
			ontology_id: '',
			description: '',
		});
		const [actionForm, setActionForm] = React.useState({
			name: '',
			description: '',
			enabled: true,
			model_id: '',
			entity_type: '',
			attribute: '',
			operator: 'gt',
			threshold: '0',
			pipeline_id: '',
			parameters: '{"alert_severity":"high"}',
		});
		const [automationForm, setAutomationForm] = React.useState({
			name: '',
			description: '',
			enabled: true,
			trigger_type: 'pipeline_completed',
			trigger_config: '{"pipeline_types":["ingestion"]}',
		});
		const [queryForm, setQueryForm] = React.useState({ query: '' });

		const resetActionForm = React.useCallback(() => {
			setActionForm({
				name: '',
				description: '',
				enabled: true,
				model_id: '',
				entity_type: '',
				attribute: '',
				operator: 'gt',
				threshold: '0',
				pipeline_id: '',
				parameters: '{"alert_severity":"high"}',
			});
		}, []);

		const resetAutomationForm = React.useCallback(() => {
			setAutomationForm({
				name: '',
				description: '',
				enabled: true,
				trigger_type: 'pipeline_completed',
				trigger_config: '{"pipeline_types":["ingestion"]}',
			});
		}, []);

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
				setLoading(false);
				return;
			}
			setLoading(true);
			try {
				const data = await apiCall(`/api/digital-twins?project_id=${projectId}`);
				setTwins(data || []);
			} catch (error) {
				console.error('Failed to load digital twins:', error);
				setTwins([]);
			}
			setLoading(false);
		};

		const loadTwinDetails = async (twinId) => {
			if (!activeProject?.id) return;
			try {
				const [entitiesData, scenariosData, actionsData, insightsData, alertsData, runsData, automationsData] = await Promise.all([
					apiCall(`/api/digital-twins/${twinId}/entities`).catch(() => []),
					apiCall(`/api/digital-twins/${twinId}/scenarios`).catch(() => []),
					apiCall(`/api/digital-twins/${twinId}/actions`).catch(() => []),
					apiCall(`/api/insights?project_id=${activeProject.id}`).catch(() => []),
					apiCall(`/api/digital-twins/${twinId}/alerts?limit=100`).catch(() => []),
					apiCall(`/api/digital-twins/${twinId}/runs?limit=50`).catch(() => []),
					apiCall(`/api/digital-twins/${twinId}/automations`).catch(() => []),
				]);
				setEntities(entitiesData || []);
				setScenarios(scenariosData || []);
				setActions(actionsData || []);
				setInsights(insightsData || []);
				setAlerts(alertsData || []);
				setRuns(runsData || []);
				setAutomations(automationsData || []);
			} catch (error) {
				console.error('Failed to load twin details:', error);
			}
		};

		React.useEffect(() => {
			loadTwins();
		}, [activeProject]);

		React.useEffect(() => {
			if (selectedTwin) {
				setSelectedTab('Overview');
				loadTwinDetails(selectedTwin.id);
			}
		}, [selectedTwin, activeProject?.id]);

		const handleSubmit = async (e) => {
			e.preventDefault();
			try {
				await apiCall('/api/digital-twins', {
					method: 'POST',
					body: JSON.stringify(formData),
				});
				setShowModal(false);
				setFormData({ name: '', project_id: activeProject?.id || '', ontology_id: '', description: '' });
				loadTwins();
			} catch (error) {
				alert('Failed to create digital twin: ' + error.message);
			}
		};

		const handleDelete = async (id) => {
			if (!confirm('Delete this digital twin?')) return;
			try {
				await apiCall(`/api/digital-twins/${id}`, { method: 'DELETE' });
				if (selectedTwin?.id === id) setSelectedTwin(null);
				loadTwins();
			} catch (error) {
				alert('Failed to delete digital twin: ' + error.message);
			}
		};

		const handleSync = async (id) => {
			try {
				const result = await apiCall(`/api/digital-twins/${id}/sync`, { method: 'POST', body: JSON.stringify({}) });
				alert(result?.message || `Digital twin sync queued (${result?.work_task_id || 'work task created'})`);
				loadTwins();
				if (selectedTwin?.id === id) loadTwinDetails(id);
			} catch (error) {
				alert('Failed to sync digital twin: ' + error.message);
			}
		};

		const handleProcessTwin = async (id) => {
			try {
				const result = await apiCall(`/api/digital-twins/${id}/runs`, { method: 'POST', body: JSON.stringify({}) });
				alert(`Twin processing queued (${result?.id || 'run created'})`);
				if (selectedTwin?.id === id) loadTwinDetails(id);
				loadTwins();
			} catch (error) {
				alert('Failed to process digital twin: ' + error.message);
			}
		};

		const handleCreateAction = async (e) => {
			e.preventDefault();
			try {
				let parameters = {};
				if (actionForm.parameters.trim()) parameters = JSON.parse(actionForm.parameters);
				let threshold;
				try { threshold = JSON.parse(actionForm.threshold); } catch { threshold = actionForm.threshold; }
				const data = {
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
						parameters,
					},
				};
				await apiCall(`/api/digital-twins/${selectedTwin.id}/actions`, { method: 'POST', body: JSON.stringify(data) });
				setShowActionModal(false);
				resetActionForm();
				loadTwinDetails(selectedTwin.id);
			} catch (error) {
				alert('Failed to create action: ' + error.message);
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
				loadTwinDetails(selectedTwin.id);
			} catch (error) {
				alert('Failed to create automation: ' + error.message);
			}
		};

		const handleDeleteAction = async (actionId) => {
			if (!confirm('Delete this action?')) return;
			try {
				await apiCall(`/api/digital-twins/${selectedTwin.id}/actions/${actionId}`, { method: 'DELETE' });
				loadTwinDetails(selectedTwin.id);
			} catch (error) {
				alert('Failed to delete action: ' + error.message);
			}
		};

		const handleDeleteAutomation = async (automationId) => {
			if (!confirm('Delete this automation?')) return;
			try {
				await apiCall(`/api/digital-twins/${selectedTwin.id}/automations/${automationId}`, { method: 'DELETE' });
				loadTwinDetails(selectedTwin.id);
			} catch (error) {
				alert('Failed to delete automation: ' + error.message);
			}
		};

		const handleQuery = async (e) => {
			e.preventDefault();
			try {
				const result = await apiCall(`/api/digital-twins/${selectedTwin.id}/query`, {
					method: 'POST',
					body: JSON.stringify({ query: queryForm.query }),
				});
				setQueryResult(result);
			} catch (error) {
				alert('Failed to execute query: ' + error.message);
			}
		};

		const projectOptions = projects.map(p => ({ value: p.id, label: p.name }));
		const ontologyOptions = ontologies.map(o => ({ value: o.id, label: o.name }));
		const mlModelOptions = mlModels.map(m => ({ value: m.id, label: m.name }));
		const pipelineOptions = pipelines.filter(p => p.type === 'output').map(p => ({ value: p.id, label: p.name }));
		const operatorOptions = [
			{ value: 'gt', label: 'Greater Than' },
			{ value: 'gte', label: 'Greater Than or Equal' },
			{ value: 'lt', label: 'Less Than' },
			{ value: 'lte', label: 'Less Than or Equal' },
			{ value: 'eq', label: 'Equals' },
			{ value: 'ne', label: 'Not Equal' },
		];

		const latestRun = runs[0];
		const latestAlert = alerts[0];
		const summaryCards = [
			{ label: 'Entities', value: entities.length, tone: 'var(--accent)' },
			{ label: 'Insights', value: insights.length, tone: '#4cc9f0' },
			{ label: 'Alert Events', value: alerts.length, tone: '#ff7b72' },
			{ label: 'Automations', value: automations.length, tone: '#7ee787' },
		];

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
			{ key: 'condition', label: 'Condition', render: (row) => { const condition = row.condition || {}; const scope = condition.attribute ? `${condition.entity_type || 'Any entity'}.${condition.attribute}` : (condition.model_id ? `Model ${condition.model_id}` : 'Condition'); return `${scope} ${condition.operator || ''} ${JSON.stringify(condition.threshold)}`; } },
			{ key: 'pipeline_id', label: 'Trigger Pipeline', render: (row) => row.trigger?.pipeline_id || '—' },
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
			{ key: 'category', label: 'Category' },
			{ key: 'title', label: 'Title' },
			{ key: 'message', label: 'Message' },
			{ key: 'triggered_export_pipeline_id', label: 'Export Pipeline', render: (row) => row.triggered_export_pipeline_id || '—' },
			{ key: 'created_at', label: 'Created', render: (row) => new Date(row.created_at).toLocaleString() },
		];
		const runColumns = [
			{ key: 'status', label: 'Status', render: (row) => <span className={`status-badge status-${row.status}`}>{row.status}</span> },
			{ key: 'trigger_type', label: 'Trigger' },
			{ key: 'requested_at', label: 'Requested', render: (row) => new Date(row.requested_at).toLocaleString() },
			{ key: 'completed_at', label: 'Completed', render: (row) => row.completed_at ? new Date(row.completed_at).toLocaleString() : '—' },
			{ key: 'insight_count', label: 'Insights', render: (row) => row.metrics?.insight_count ?? '—' },
			{ key: 'alert_count', label: 'Alerts', render: (row) => row.metrics?.alert_count ?? '—' },
			{ key: 'triggered_action_count', label: 'Triggered Actions', render: (row) => row.metrics?.triggered_action_count ?? '—' },
		];
		const automationColumns = [
			{ key: 'name', label: 'Name' },
			{ key: 'enabled', label: 'Enabled', render: (row) => row.enabled ? 'Yes' : 'No' },
			{ key: 'trigger_type', label: 'Trigger' },
			{ key: 'action_type', label: 'Action' },
			{ key: 'updated_at', label: 'Updated', render: (row) => new Date(row.updated_at).toLocaleString() },
		];

		return (
			<div className="content-section">
				<div className="section-header">
					<h2>Digital Twins</h2>
					<Button label="+ New Digital Twin" onClick={() => setShowModal(true)} />
				</div>

				{selectedTwin ? (
					<>
						<div style={{ marginBottom: '20px' }}>
							<Button label="← Back to List" onClick={() => setSelectedTwin(null)} variant="secondary" />
							<h3 style={{ color: 'var(--accent)', marginTop: '16px', marginBottom: '8px' }}>{selectedTwin.name} - Twin Workspace</h3>
							<p style={{ color: 'var(--text-secondary)', margin: 0 }}>{selectedTwin.description || 'Ontology-grounded operational workspace for this project.'}</p>
							<div style={{ display: 'flex', gap: '8px', marginTop: '12px', flexWrap: 'wrap' }}>
								<Button label="Process Twin" onClick={() => handleProcessTwin(selectedTwin.id)} variant="secondary" />
								<Button label="Queue Source Sync" onClick={() => handleSync(selectedTwin.id)} variant="secondary" />
								<Button label="+ New Automation" onClick={() => setShowAutomationModal(true)} variant="secondary" />
								<Button label="+ New Action" onClick={() => setShowActionModal(true)} variant="secondary" />
								<Button label="Query Twin" onClick={() => setShowQueryModal(true)} variant="secondary" />
							</div>
						</div>

						<div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(180px, 1fr))', gap: '12px', marginBottom: '20px' }}>
							{summaryCards.map(card => (
								<div key={card.label} style={{ padding: '16px', border: '1px solid var(--border)', borderRadius: '10px', background: 'linear-gradient(180deg, rgba(255,255,255,0.03), transparent)' }}>
									<div style={{ color: 'var(--text-secondary)', fontSize: '0.82rem', marginBottom: '8px' }}>{card.label}</div>
									<div style={{ fontSize: '1.8rem', fontWeight: 700, color: card.tone }}>{card.value}</div>
								</div>
							))}
						</div>

						{selectedTab === 'Overview' && (
							<div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(280px, 1fr))', gap: '16px', marginBottom: '20px' }}>
								<div style={{ padding: '16px', border: '1px solid var(--border)', borderRadius: '10px' }}>
									<h3 style={{ marginTop: 0 }}>Current State</h3>
									<div><strong>Status:</strong> {selectedTwin.status}</div>
									<div><strong>Ontology:</strong> {selectedTwin.ontology_id}</div>
									<div><strong>Last Sync:</strong> {selectedTwin.last_sync_at ? new Date(selectedTwin.last_sync_at).toLocaleString() : 'Never'}</div>
									<div><strong>Configured Storage Sources:</strong> {selectedTwin.config?.storage_ids?.length || 0}</div>
									<div style={{ marginTop: '10px', color: 'var(--text-secondary)', fontSize: '0.85rem' }}>Process Twin runs the full workflow: sync scoped sources, generate insights, evaluate alert events, and trigger configured export actions. Queue Source Sync refreshes source-backed entities only.</div>
								</div>
								<div style={{ padding: '16px', border: '1px solid var(--border)', borderRadius: '10px' }}>
									<h3 style={{ marginTop: 0 }}>Latest Processing Run</h3>
									{latestRun ? <>
										<div><strong>Status:</strong> {latestRun.status}</div>
										<div><strong>Trigger:</strong> {latestRun.trigger_type}</div>
										<div><strong>Requested:</strong> {new Date(latestRun.requested_at).toLocaleString()}</div>
										<div><strong>Insights:</strong> {latestRun.metrics?.insight_count ?? 0}</div>
										<div><strong>Alerts:</strong> {latestRun.metrics?.alert_count ?? 0}</div>
									</> : <div style={{ color: 'var(--text-secondary)' }}>No processing runs yet.</div>}
								</div>
								<div style={{ padding: '16px', border: '1px solid var(--border)', borderRadius: '10px' }}>
									<h3 style={{ marginTop: 0 }}>Latest Alert Event</h3>
									{latestAlert ? <>
										<div><strong>Severity:</strong> {latestAlert.severity}</div>
										<div><strong>Title:</strong> {latestAlert.title}</div>
										<div><strong>Created:</strong> {new Date(latestAlert.created_at).toLocaleString()}</div>
									</> : <div style={{ color: 'var(--text-secondary)' }}>No alert events emitted yet.</div>}
								</div>
							</div>
						)}

						<Tabs tabs={['Overview', 'Insights', 'Alerts', 'Automations', 'Entities', 'Actions', 'Scenarios', 'Runs']} activeTab={selectedTab} onTabChange={setSelectedTab} />

						{selectedTab === 'Insights' ? (
							<Table columns={insightColumns} data={insights} />
						) : selectedTab === 'Alerts' ? (
							<Table columns={alertColumns} data={alerts} />
						) : selectedTab === 'Automations' ? (
							<Table columns={automationColumns} data={automations} actions={(row) => <Button label="Delete" onClick={() => handleDeleteAutomation(row.id)} variant="danger" />} />
						) : selectedTab === 'Entities' ? (
							<Table columns={entityColumns} data={entities} />
						) : selectedTab === 'Actions' ? (
							<Table columns={actionColumns} data={actions} actions={(row) => <Button label="Delete" onClick={() => handleDeleteAction(row.id)} variant="danger" />} />
						) : selectedTab === 'Scenarios' ? (
							<Table columns={scenarioColumns} data={scenarios} />
						) : selectedTab === 'Runs' ? (
							<Table columns={runColumns} data={runs} />
						) : null}
					</>
				) : (
					<>
						{loading ? (
							<div className="loading">Loading digital twins...</div>
						) : (
							<Table columns={twinColumns} data={twins} actions={(row) => <>
								<Button label="View" onClick={() => setSelectedTwin(row)} variant="secondary" />
								<Button label="Process" onClick={() => handleProcessTwin(row.id)} variant="secondary" />
								<Button label="Sync Sources" onClick={() => handleSync(row.id)} variant="secondary" />
								<Button label="Delete" onClick={() => handleDelete(row.id)} variant="danger" />
							</>} />
						)}
					</>
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
						<label style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '16px' }}>
							<input type="checkbox" checked={automationForm.enabled} onChange={e => setAutomationForm({ ...automationForm, enabled: e.target.checked })} />
							Enabled
						</label>
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
						<FormField label="Threshold" value={actionForm.threshold} onChange={(v) => setActionForm({ ...actionForm, threshold: v })} placeholder="0" required />
						<FormField label="Trigger Parameters (JSON)" type="textarea" value={actionForm.parameters} onChange={(v) => setActionForm({ ...actionForm, parameters: v })} placeholder='{"alert_severity":"high","alert_category":"stockout"}' />
						<label style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '16px' }}>
							<input type="checkbox" checked={actionForm.enabled} onChange={e => setActionForm({ ...actionForm, enabled: e.target.checked })} />
							Enabled
						</label>
						<Button type="submit" label="Create Action" />
					</form>
				</Modal>

				<Modal open={showQueryModal} onClose={() => setShowQueryModal(false)} title="Query Digital Twin">
					<form onSubmit={handleQuery}>
						<FormField label="SPARQL SELECT Query" type="textarea" value={queryForm.query} onChange={(v) => setQueryForm({ ...queryForm, query: v })} placeholder="SELECT ?entity ?type WHERE { ?entity a ?type } LIMIT 25" required />
						<Button type="submit" label="Execute Query" />
					</form>
					{queryResult && <div style={{ marginTop: '16px' }}><h3 style={{ color: 'var(--accent)' }}>Query Result:</h3><div className="json-display"><pre>{JSON.stringify(queryResult, null, 2)}</pre></div></div>}
				</Modal>
			</div>
		);
	};
})();

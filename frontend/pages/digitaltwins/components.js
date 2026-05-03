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

	root.digitalTwins = Object.assign(root.digitalTwins || {}, {
		operatorOptions,
		approvalModeOptions,
		workspaceTabs,
		getApprovalBadgeClass,
		getExecutionBadgeClass,
		createTwinPath,
		createEmptyActionForm,
		createEmptyAutomationForm,
		MetricCards,
		JsonPreview,
		ControlPanel,
		TwinWorkspaceHeader,
	});
})();

(() => {
	const root = window.MimirApp = window.MimirApp || {};
	const pages = root.pages = root.pages || {};
	const { apiCall, deriveStorageConfigLabel, renderConfigPreview, notify, confirmAction } = root.lib;
	const { ProjectContext } = root.context;
	const { Button, FormField, Modal, Table } = root.components.primitives;

	pages.StoragePage = function StoragePage() {
		const { activeProject, projects } = React.useContext(ProjectContext);
		const [configs, setConfigs] = React.useState([]);
		const [loading, setLoading] = React.useState(true);
		const [loadError, setLoadError] = React.useState('');
		const [showModal, setShowModal] = React.useState(false);
		const [healthStatus, setHealthStatus] = React.useState({});
		const [formData, setFormData] = React.useState({ project_id: '', plugin_type: 'filesystem', config: '{}' });

		React.useEffect(() => {
			if (activeProject?.id) {
				setFormData(prev => ({ ...prev, project_id: activeProject.id }));
			}
		}, [activeProject?.id]);

		const loadConfigs = React.useCallback(async () => {
			setLoading(true);
			setLoadError('');
			try {
				const projectId = activeProject?.id || '';
				const data = await apiCall(`/api/storage/configs?project_id=${projectId}`);
				setConfigs(data || []);
			} catch (error) {
				setConfigs([]);
				setLoadError(error.message || 'Failed to load storage configurations.');
			} finally {
				setLoading(false);
			}
		}, [activeProject?.id]);

		React.useEffect(() => {
			loadConfigs();
		}, [loadConfigs]);

		const handleSubmit = async (e) => {
			e.preventDefault();
			try {
				await apiCall('/api/storage/configs', {
					method: 'POST',
					body: JSON.stringify({
						project_id: formData.project_id,
						plugin_type: formData.plugin_type,
						config: JSON.parse(formData.config),
					}),
				});
				setShowModal(false);
				setFormData({ project_id: activeProject?.id || '', plugin_type: 'filesystem', config: '{}' });
				notify({ tone: 'success', message: 'Storage configuration created.' });
				loadConfigs();
			} catch (error) {
				notify({ tone: 'error', message: `Failed to create storage config: ${error.message}` });
			}
		};

		const handleDelete = async (id) => {
			const confirmed = await confirmAction({
				title: 'Delete storage configuration',
				message: 'Delete this storage configuration permanently? Mimir will block deletion while pipelines, digital twins, or other persisted resources still reference it.',
				confirmLabel: 'Delete config',
				variant: 'danger',
			});
			if (!confirmed) return;
			try {
				await apiCall(`/api/storage/configs/${id}`, { method: 'DELETE' });
				notify({ tone: 'success', message: 'Storage configuration deleted.' });
				loadConfigs();
			} catch (error) {
				notify({ tone: 'error', message: `Failed to delete storage config: ${error.message}` });
			}
		};

		const handleCheckHealth = async (id) => {
			setHealthStatus(prev => ({ ...prev, [id]: 'checking' }));
			try {
				const result = await apiCall(`/api/storage/health?config_id=${id}`);
				const status = result.healthy ? 'ok' : 'error';
				setHealthStatus(prev => ({ ...prev, [id]: status }));
				notify({ tone: status === 'ok' ? 'success' : 'error', message: status === 'ok' ? 'Storage connection healthy.' : 'Storage connection failed health check.' });
			} catch (error) {
				setHealthStatus(prev => ({ ...prev, [id]: 'error' }));
				notify({ tone: 'error', message: `Storage health check failed: ${error.message}` });
			}
			window.setTimeout(() => setHealthStatus(prev => {
				const next = { ...prev };
				delete next[id];
				return next;
			}), 8000);
		};

		const projectOptions = projects.map(p => ({ value: p.id, label: p.name }));
		const columns = [
			{ key: 'id', label: 'ID' },
			{ key: 'label', label: 'Label', render: row => deriveStorageConfigLabel(row) },
			{ key: 'plugin_type', label: 'Plugin Type' },
			{ key: 'project_id', label: 'Project ID' },
			{ key: 'config', label: 'Config', render: row => <pre style={{ margin: 0, fontSize: '0.75rem', maxWidth: '320px', whiteSpace: 'pre-wrap' }}>{renderConfigPreview(row.config)}</pre> },
			{ key: 'created_at', label: 'Created', render: row => new Date(row.created_at).toLocaleDateString() },
		];

		return (
			<div className="content-section">
				<div className="section-header">
					<h2>Storage Configurations</h2>
					<Button label="+ New Storage Config" onClick={() => setShowModal(true)} />
				</div>
				{activeProject ? <div className="page-notice"><strong>Project scope:</strong> showing storage for {activeProject.name}.</div> : null}
				{loadError ? <div className="error-message">{loadError}</div> : null}

				{loading ? (
					<div className="loading">Loading storage configurations…</div>
				) : (
					<Table
						caption="Storage configurations"
						columns={columns}
						data={configs}
						emptyState={activeProject ? 'No storage configurations exist for this project yet.' : 'Select a project to inspect storage configurations.'}
						actions={(row) => (
							<>
								{healthStatus[row.id] === 'checking' ? <span className="status-badge status-pending">Checking</span> : null}
								{healthStatus[row.id] === 'ok' ? <span className="status-badge status-active">Connected</span> : null}
								{healthStatus[row.id] === 'error' ? <span className="status-badge status-failed">Failed</span> : null}
								<Button label="Test Connection" onClick={() => handleCheckHealth(row.id)} variant="secondary" />
								<Button label="Delete" onClick={() => handleDelete(row.id)} variant="danger" />
							</>
						)}
					/>
				)}

				<Modal open={showModal} onClose={() => setShowModal(false)} title="Create Storage Configuration">
					<form onSubmit={handleSubmit}>
						<div className="form-grid">
							<FormField label="Project" type="select" value={formData.project_id} onChange={(v) => setFormData({ ...formData, project_id: v })} options={projectOptions} required />
							<FormField label="Plugin Type" value={formData.plugin_type} onChange={(v) => setFormData({ ...formData, plugin_type: v })} placeholder="filesystem, s3, postgres, mongodb..." required />
						</div>
						<FormField label="Configuration (JSON)" type="textarea" value={formData.config} onChange={(v) => setFormData({ ...formData, config: v })} placeholder='{"path": "./data"}' required />
						<Button type="submit" label="Create Storage Config" />
					</form>
				</Modal>
			</div>
		);
	};
})();

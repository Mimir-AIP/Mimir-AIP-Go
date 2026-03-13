(() => {
	const root = window.MimirApp = window.MimirApp || {};
	const pages = root.pages = root.pages || {};
	const { apiCall, deriveStorageConfigLabel, renderConfigPreview } = root.lib;
	const { ProjectContext } = root.context;
	const { Button, FormField, Modal, Table } = root.components.primitives;

	pages.StoragePage = function StoragePage() {
		const { activeProject, projects } = React.useContext(ProjectContext);
		const [configs, setConfigs] = React.useState([]);
		const [loading, setLoading] = React.useState(true);
		const [showModal, setShowModal] = React.useState(false);
		const [healthStatus, setHealthStatus] = React.useState({});
		const [formData, setFormData] = React.useState({
			project_id: '',
			plugin_type: 'filesystem',
			config: '{}',
		});

		React.useEffect(() => {
			if (activeProject?.id) {
				setFormData(prev => ({ ...prev, project_id: activeProject.id }));
			}
		}, [activeProject?.id]);

		const loadConfigs = async () => {
			setLoading(true);
			try {
				const projectId = activeProject?.id || '';
				const data = await apiCall(`/api/storage/configs?project_id=${projectId}`);
				setConfigs(data || []);
			} catch (error) {
				console.error('Failed to load storage configs:', error);
				setConfigs([]);
			}
			setLoading(false);
		};

		React.useEffect(() => {
			loadConfigs();
		}, [activeProject?.id]);

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
				loadConfigs();
			} catch (error) {
				alert('Failed to create storage config: ' + error.message);
			}
		};

		const handleDelete = async (id) => {
			if (!confirm('Delete this storage config?')) return;
			try {
				await apiCall(`/api/storage/configs/${id}`, { method: 'DELETE' });
				loadConfigs();
			} catch (error) {
				alert('Failed to delete storage config: ' + error.message);
			}
		};

		const handleCheckHealth = async (id) => {
			setHealthStatus(prev => ({ ...prev, [id]: 'checking' }));
			try {
				const result = await apiCall(`/api/storage/health?config_id=${id}`);
				setHealthStatus(prev => ({ ...prev, [id]: result.healthy ? 'ok' : 'error' }));
			} catch {
				setHealthStatus(prev => ({ ...prev, [id]: 'error' }));
			}
			setTimeout(() => setHealthStatus(prev => {
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
			{
				key: 'config',
				label: 'Config',
				render: row => <pre style={{ margin: 0, fontSize: '0.75rem', maxWidth: '320px', whiteSpace: 'pre-wrap' }}>{renderConfigPreview(row.config)}</pre>
			},
			{ key: 'created_at', label: 'Created', render: row => new Date(row.created_at).toLocaleDateString() },
		];

		return (
			<div className="content-section">
				<div className="section-header">
					<h2>Storage Configurations</h2>
					<Button label="+ New Storage Config" onClick={() => setShowModal(true)} />
				</div>
				{activeProject && <div style={{ marginBottom: '12px', color: 'var(--text-secondary)' }}>Showing storage for <strong style={{ color: 'var(--accent)' }}>{activeProject.name}</strong>.</div>}

				{loading ? (
					<div className="loading">Loading storage configurations...</div>
				) : (
					<Table
						columns={columns}
						data={configs}
						actions={(row) => (
							<>
								{healthStatus[row.id] === 'checking' && <span style={{ color: 'var(--accent)' }}>Checking…</span>}
								{healthStatus[row.id] === 'ok' && <span style={{ color: '#22c55e' }}>✓ Connected</span>}
								{healthStatus[row.id] === 'error' && <span style={{ color: '#ef4444' }}>✗ Failed</span>}
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

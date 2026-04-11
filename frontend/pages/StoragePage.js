(() => {
	const root = window.MimirApp = window.MimirApp || {};
	const pages = root.pages = root.pages || {};
	const { apiCall, deriveStorageConfigLabel, renderConfigPreview, notify, confirmAction } = root.lib;
	const { ProjectContext } = root.context;
	const { Button, FormField, Modal, Table } = root.components.primitives;

	function emptyStorageForm(projectId = '') {
		return { project_id: projectId, plugin_type: 'filesystem', config: '{}', active: true };
	}

	pages.StoragePage = function StoragePage() {
		const { activeProject, projects } = React.useContext(ProjectContext);
		const [configs, setConfigs] = React.useState([]);
		const [loading, setLoading] = React.useState(true);
		const [loadError, setLoadError] = React.useState('');
		const [showModal, setShowModal] = React.useState(false);
		const [editingConfigId, setEditingConfigId] = React.useState('');
		const [showMetadataModal, setShowMetadataModal] = React.useState(false);
		const [selectedMetadata, setSelectedMetadata] = React.useState(null);
		const [metadataError, setMetadataError] = React.useState('');
		const [healthStatus, setHealthStatus] = React.useState({});
		const [ingestionHealth, setIngestionHealth] = React.useState(null);
		const [ingestionHealthLoading, setIngestionHealthLoading] = React.useState(false);
		const [formData, setFormData] = React.useState(emptyStorageForm());

		React.useEffect(() => {
			if (activeProject?.id) {
				setFormData(prev => ({ ...prev, project_id: prev.project_id || activeProject.id }));
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

		const loadIngestionHealth = React.useCallback(async () => {
			if (!activeProject?.id) {
				setIngestionHealth(null);
				return;
			}
			setIngestionHealthLoading(true);
			try {
				const report = await apiCall(`/api/storage/ingestion-health?project_id=${activeProject.id}`);
				setIngestionHealth(report);
			} catch (error) {
				setIngestionHealth({ status: 'critical', recommendations: [error.message || 'Failed to load ingestion health.'], sources: [] });
			} finally {
				setIngestionHealthLoading(false);
			}
		}, [activeProject?.id]);

		React.useEffect(() => {
			loadConfigs();
			loadIngestionHealth();
		}, [loadConfigs, loadIngestionHealth]);

		const openCreateModal = () => {
			setEditingConfigId('');
			setFormData(emptyStorageForm(activeProject?.id || ''));
			setShowModal(true);
		};

		const openEditModal = (config) => {
			setEditingConfigId(config.id);
			setFormData({
				project_id: config.project_id,
				plugin_type: config.plugin_type,
				config: JSON.stringify(config.config || {}, null, 2),
				active: config.active !== false,
			});
			setShowModal(true);
		};

		const handleSubmit = async (e) => {
			e.preventDefault();
			try {
				const payload = {
					project_id: formData.project_id,
					plugin_type: formData.plugin_type,
					config: JSON.parse(formData.config),
				};
				if (editingConfigId) {
					await apiCall(`/api/storage/configs/${editingConfigId}?project_id=${formData.project_id}`, {
						method: 'PUT',
						body: JSON.stringify({ config: payload.config, active: formData.active }),
					});
					notify({ tone: 'success', message: 'Storage configuration updated.' });
				} else {
					await apiCall('/api/storage/configs', { method: 'POST', body: JSON.stringify(payload) });
					notify({ tone: 'success', message: 'Storage configuration created.' });
				}
				setShowModal(false);
				setEditingConfigId('');
				setFormData(emptyStorageForm(activeProject?.id || ''));
				loadConfigs();
				loadIngestionHealth();
			} catch (error) {
				notify({ tone: 'error', message: `Failed to save storage config: ${error.message}` });
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
				await apiCall(`/api/storage/configs/${id}?project_id=${activeProject?.id || ''}`, { method: 'DELETE' });
				notify({ tone: 'success', message: 'Storage configuration deleted.' });
				loadConfigs();
				loadIngestionHealth();
			} catch (error) {
				notify({ tone: 'error', message: `Failed to delete storage config: ${error.message}` });
			}
		};

		const handleCheckHealth = async (id) => {
			setHealthStatus(prev => ({ ...prev, [id]: 'checking' }));
			try {
				const result = await apiCall(`/api/storage/health?config_id=${id}&project_id=${activeProject?.id || ''}`);
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

		const handleViewMetadata = async (id) => {
			setMetadataError('');
			setSelectedMetadata(null);
			setShowMetadataModal(true);
			try {
				const metadata = await apiCall(`/api/storage/metadata?config_id=${id}&project_id=${activeProject?.id || ''}`);
				setSelectedMetadata(metadata);
			} catch (error) {
				setMetadataError(error.message || 'Failed to load storage metadata.');
			}
		};

		const projectOptions = projects.map(p => ({ value: p.id, label: p.name }));
		const columns = [
			{ key: 'id', label: 'ID' },
			{ key: 'label', label: 'Label', render: row => deriveStorageConfigLabel(row) },
			{ key: 'plugin_type', label: 'Plugin Type' },
			{ key: 'active', label: 'Status', render: row => <span className={`status-badge ${row.active ? 'status-active' : 'status-inactive'}`}>{row.active ? 'Active' : 'Inactive'}</span> },
			{ key: 'config', label: 'Config', render: row => <pre style={{ margin: 0, fontSize: '0.75rem', maxWidth: '320px', whiteSpace: 'pre-wrap' }}>{renderConfigPreview(row.config)}</pre> },
			{ key: 'created_at', label: 'Created', render: row => new Date(row.created_at).toLocaleDateString() },
		];

		return (
			<div className="content-section">
				<div className="section-header">
					<h2>Storage Configurations</h2>
					<Button label="+ New Storage Config" onClick={openCreateModal} />
				</div>
				{activeProject ? <div className="page-notice"><strong>Project scope:</strong> showing storage for {activeProject.name}.</div> : null}
				{loadError ? <div className="error-message">{loadError}</div> : null}

				<div className="section-panel section-panel--neutral" style={{ marginBottom: '1rem' }}>
					<div className="section-panel-header">
						<div>
							<h3 className="section-panel-title">Ingestion Health</h3>
							<p className="section-panel-copy">Project-level freshness, completeness, and schema drift indicators across active storage sources.</p>
						</div>
					</div>
					{!activeProject ? (
						<div className="empty-state">Select a project to inspect ingestion health.</div>
					) : ingestionHealthLoading ? (
						<div className="loading">Computing ingestion health…</div>
					) : ingestionHealth ? (
						<>
							<div className="form-grid">
								<div className="form-group">
									<label>Overall Status</label>
									<div className={`field-static status-badge status-${ingestionHealth.status || 'inactive'}`}>{ingestionHealth.status || 'unknown'}</div>
								</div>
								<div className="form-group">
									<label>Overall Score</label>
									<div className="field-static">{ingestionHealth.overall_score ?? '—'}</div>
								</div>
							</div>
							{Array.isArray(ingestionHealth.sources) && ingestionHealth.sources.length ? (
								<div className="form-group">
									<label>Sources</label>
									<ul style={{ margin: 0, paddingLeft: '1.2rem' }}>
										{ingestionHealth.sources.map(source => (
											<li key={source.storage_id}>{source.plugin_type} · {source.storage_id} · {source.status} · score {source.overall_score}</li>
										))}
									</ul>
								</div>
							) : null}
							{Array.isArray(ingestionHealth.recommendations) && ingestionHealth.recommendations.length ? (
								<div className="form-group">
									<label>Recommendations</label>
									<ul style={{ margin: 0, paddingLeft: '1.2rem' }}>
										{ingestionHealth.recommendations.map((item, idx) => <li key={`${item}-${idx}`}>{item}</li>)}
									</ul>
								</div>
							) : null}
						</>
					) : null}
				</div>

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
								<Button label="Metadata" onClick={() => handleViewMetadata(row.id)} variant="secondary" />
								<Button label="Edit" onClick={() => openEditModal(row)} variant="secondary" />
								<Button label="Test Connection" onClick={() => handleCheckHealth(row.id)} variant="secondary" />
								<Button label="Delete" onClick={() => handleDelete(row.id)} variant="danger" />
							</>
						)}
					/>
				)}

				<Modal open={showModal} onClose={() => setShowModal(false)} title={editingConfigId ? 'Edit Storage Configuration' : 'Create Storage Configuration'}>
					<form onSubmit={handleSubmit}>
						<div className="form-grid">
							<FormField label="Project" type="select" value={formData.project_id} onChange={(v) => setFormData({ ...formData, project_id: v })} options={projectOptions} required disabled={Boolean(editingConfigId)} />
							<FormField label="Plugin Type" value={formData.plugin_type} onChange={(v) => setFormData({ ...formData, plugin_type: v })} placeholder="filesystem, s3, postgres, mongodb..." required disabled={Boolean(editingConfigId)} />
						</div>
						<FormField label="Configuration (JSON)" type="textarea" value={formData.config} onChange={(v) => setFormData({ ...formData, config: v })} placeholder='{"path": "./data"}' required />
						<label className="checkbox-row">
							<input type="checkbox" checked={formData.active} onChange={e => setFormData({ ...formData, active: e.target.checked })} />
							Active
						</label>
						<Button type="submit" label={editingConfigId ? 'Save Storage Config' : 'Create Storage Config'} />
					</form>
				</Modal>

				<Modal open={showMetadataModal} onClose={() => setShowMetadataModal(false)} title="Storage Metadata">
					{metadataError ? <div className="error-message">{metadataError}</div> : null}
					{selectedMetadata ? (
						<pre style={{ margin: 0, fontSize: '0.8rem', whiteSpace: 'pre-wrap' }}>{JSON.stringify(selectedMetadata, null, 2)}</pre>
					) : !metadataError ? (
						<div className="loading">Loading storage metadata…</div>
					) : null}
				</Modal>
			</div>
		);
	};
})();

(() => {
	const root = window.MimirApp = window.MimirApp || {};
	const pages = root.pages = root.pages || {};
	const { apiCall, deriveStorageConfigLabel, notify, confirmAction } = root.lib;
	const { ProjectContext } = root.context;
	const { useTaskWebSocket } = root.hooks;
	const { Button, FormField, Graph, Modal, Table } = root.components.primitives;

	pages.MLModelsPage = function MLModelsPage() {
		const { activeProject, projects } = React.useContext(ProjectContext);
		const [models, setModels] = React.useState([]);
		const [loading, setLoading] = React.useState(true);
		const [loadError, setLoadError] = React.useState('');
		const [showModal, setShowModal] = React.useState(false);
		const [showRecommendModal, setShowRecommendModal] = React.useState(false);
		const [showTrainModal, setShowTrainModal] = React.useState(false);
		const [trainingTarget, setTrainingTarget] = React.useState(null);
		const [trainStorageIds, setTrainStorageIds] = React.useState([]);
		const [availableStorageConfigs, setAvailableStorageConfigs] = React.useState([]);
		const [recommendForm, setRecommendForm] = React.useState({ project_id: '', ontology_id: '' });
		const [recommendOntologies, setRecommendOntologies] = React.useState([]);
		const [recommendResult, setRecommendResult] = React.useState(null);
		const [formData, setFormData] = React.useState({
			name: '',
			project_id: '',
			model_type: '',
			version: '1.0.0',
			config: '{}',
		});
		const [trainingMetrics, setTrainingMetrics] = React.useState({});

		React.useEffect(() => {
			if (!activeProject) return;
			setFormData(prev => ({ ...prev, project_id: activeProject.id }));
			setRecommendForm(prev => ({ ...prev, project_id: activeProject.id, ontology_id: '' }));
			apiCall(`/api/ontologies?project_id=${activeProject.id}`)
				.then(data => setRecommendOntologies(data || []))
				.catch(() => setRecommendOntologies([]));
		}, [activeProject]);

		const loadModels = React.useCallback(async () => {
			if (!activeProject?.id) {
				setModels([]);
				setLoading(false);
				setLoadError('');
				return;
			}
			setLoading(true);
			setLoadError('');
			try {
				const data = await apiCall(`/api/ml-models?project_id=${activeProject.id}`);
				setModels(data || []);
			} catch (error) {
				setLoadError(error.message || 'Failed to load ML models.');
				setModels([]);
			} finally {
				setLoading(false);
			}
		}, [activeProject?.id]);

		React.useEffect(() => {
			loadModels();
		}, [loadModels]);

		useTaskWebSocket(React.useCallback((task) => {
			if (task.type !== 'ml_training') return;
			if (activeProject?.id && task.project_id !== activeProject.id) return;
			const modelID = task.task_spec?.model_id;
			const metrics = task.task_spec?.parameters?.training_metrics;
			if (modelID && metrics) {
				setTrainingMetrics(prev => ({ ...prev, [modelID]: metrics }));
			}
			if (['queued', 'scheduled', 'spawned', 'executing', 'completed', 'failed', 'timeout', 'cancelled'].includes(task.status)) {
				loadModels();
			}
		}, [activeProject?.id, loadModels]));

		const handleSubmit = async (e) => {
			e.preventDefault();
			try {
				await apiCall('/api/ml-models', {
					method: 'POST',
					body: JSON.stringify({ ...formData, config: JSON.parse(formData.config) }),
				});
				setShowModal(false);
				setFormData({ name: '', project_id: activeProject?.id || '', model_type: '', version: '1.0.0', config: '{}' });
				notify({ tone: 'success', message: 'ML model created.' });
				loadModels();
			} catch (error) {
				notify({ tone: 'error', message: `Failed to create ML model: ${error.message}` });
			}
		};

		const handleDelete = async (id) => {
			const confirmed = await confirmAction({
				title: 'Delete ML model',
				message: 'Delete this ML model? Existing artifacts and references may stop working.',
				confirmLabel: 'Delete model',
				variant: 'danger',
			});
			if (!confirmed) return;
			try {
				await apiCall(`/api/ml-models/${id}`, { method: 'DELETE' });
				notify({ tone: 'success', message: 'ML model deleted.' });
				loadModels();
			} catch (error) {
				notify({ tone: 'error', message: `Failed to delete ML model: ${error.message}` });
			}
		};

		const openTrainModal = async (row) => {
			setTrainingTarget(row);
			setTrainStorageIds([]);
			try {
				const configs = await apiCall(`/api/storage/configs?project_id=${row.project_id}`);
				setAvailableStorageConfigs(configs || []);
			} catch (error) {
				setAvailableStorageConfigs([]);
				notify({ tone: 'error', message: `Failed to load storage configs: ${error.message}` });
			}
			setShowTrainModal(true);
		};

		const handleTrain = async () => {
			try {
				await apiCall('/api/ml-models/train', {
					method: 'POST',
					body: JSON.stringify({ model_id: trainingTarget.id, storage_ids: trainStorageIds }),
				});
				setShowTrainModal(false);
				notify({ tone: 'success', message: 'Training queued. Status will update from live task events.' });
				loadModels();
			} catch (error) {
				notify({ tone: 'error', message: `Failed to start training: ${error.message}` });
			}
		};

		const handleRecommend = async (e) => {
			e.preventDefault();
			try {
				const data = await apiCall('/api/ml-models/recommend', {
					method: 'POST',
					body: JSON.stringify({
						project_id: recommendForm.project_id,
						ontology_id: recommendForm.ontology_id,
					}),
				});
				setRecommendResult(data);
			} catch (error) {
				notify({ tone: 'error', message: `Failed to get recommendation: ${error.message}` });
			}
		};

		const onRecommendProjectChange = async (projectId) => {
			setRecommendForm(prev => ({ ...prev, project_id: projectId, ontology_id: '' }));
			setRecommendOntologies([]);
			if (!projectId) return;
			try {
				const data = await apiCall(`/api/ontologies?project_id=${projectId}`);
				setRecommendOntologies(data || []);
			} catch {
				setRecommendOntologies([]);
			}
		};

		const projectOptions = projects.map(p => ({ value: p.id, label: p.name }));
		const columns = [
			{ key: 'id', label: 'ID' },
			{ key: 'name', label: 'Name' },
			{ key: 'model_type', label: 'Type' },
			{ key: 'version', label: 'Version' },
			{ key: 'status', label: 'Status', render: (row) => <span className={`status-badge status-${row.status}`}>{row.status}</span> },
			{ key: 'project_id', label: 'Project ID' },
			{ key: 'created_at', label: 'Created', render: (row) => new Date(row.created_at).toLocaleDateString() },
		];

		return (
			<div className="content-section">
				<div className="section-header">
					<h2>ML Models</h2>
					<div className="inline-actions">
						<Button label="Recommend" onClick={() => { setRecommendResult(null); setRecommendForm({ project_id: activeProject?.id || '', ontology_id: '' }); setRecommendOntologies([]); setShowRecommendModal(true); }} variant="secondary" />
						<Button label="+ New Model" onClick={() => setShowModal(true)} />
					</div>
				</div>

				{loadError ? <div className="error-message">{loadError}</div> : null}

				{loading ? (
					<div className="loading">Loading ML models…</div>
				) : (
					<Table
						caption="Project ML models"
						columns={columns}
						data={models}
						emptyState={activeProject ? 'No ML models exist for this project yet.' : 'Select a project to inspect ML models.'}
						actions={(row) => (
							<>
								<Button label="Train" onClick={() => openTrainModal(row)} variant="secondary" />
								<Button label="Delete" onClick={() => handleDelete(row.id)} variant="danger" />
							</>
						)}
					/>
				)}

				{Object.keys(trainingMetrics).length > 0 && (
					<div className="section-panel section-panel--neutral">
						<div className="section-panel-header">
							<div>
								<h3 className="section-panel-title">Training Progress</h3>
								<p className="section-panel-copy">Live metrics are shown when workers emit training progress details.</p>
							</div>
						</div>
						{Object.entries(trainingMetrics).map(([modelID, metrics]) => {
							const epochs = metrics.epochs || [];
							const loss = metrics.loss || [];
							const accuracy = metrics.accuracy || [];
							if (epochs.length === 0) return null;
							const graphData = {
								labels: epochs,
								datasets: [
									{ label: 'Loss', data: loss, borderColor: '#ef4444', backgroundColor: 'rgba(239,68,68,0.1)', yAxisID: 'y' },
									{ label: 'Accuracy', data: accuracy, borderColor: '#22c55e', backgroundColor: 'rgba(34,197,94,0.1)', yAxisID: 'y1' },
								],
							};
							const graphOptions = {
								scales: {
									y: { type: 'linear', position: 'left', title: { display: true, text: 'Loss' } },
									y1: { type: 'linear', position: 'right', title: { display: true, text: 'Accuracy' }, grid: { drawOnChartArea: false } },
								},
							};
							return (
								<div key={modelID} className="card">
									<div className="section-panel-copy">Model: {modelID}</div>
									<Graph data={graphData} options={graphOptions} type="line" />
								</div>
							);
						})}
					</div>
				)}

				<Modal open={showTrainModal} onClose={() => setShowTrainModal(false)} title={`Train: ${trainingTarget?.name || ''}`}>
					<p className="modal-copy">Select storage configs to train on.</p>
					{availableStorageConfigs.length === 0 ? (
						<p className="section-panel-copy">No storage configs found for this project.</p>
					) : (
						availableStorageConfigs.map(cfg => (
							<label key={cfg.id} className="checkbox-row">
								<input
									type="checkbox"
									checked={trainStorageIds.includes(cfg.id)}
									onChange={e => setTrainStorageIds(prev => e.target.checked ? [...prev, cfg.id] : prev.filter(id => id !== cfg.id))}
								/>
								{deriveStorageConfigLabel(cfg)}
							</label>
						))
					)}
					<div className="inline-actions">
						<Button label="Start Training" onClick={handleTrain} disabled={!trainStorageIds.length} />
					</div>
				</Modal>

				<Modal open={showRecommendModal} onClose={() => setShowRecommendModal(false)} title="Recommend Model Type">
					<form onSubmit={handleRecommend}>
						<p className="modal-copy">Select your project and ontology — the backend analyses your data automatically.</p>
						<FormField label="Project" type="select" value={recommendForm.project_id} onChange={onRecommendProjectChange} options={projectOptions} required />
						<FormField label="Ontology" type="select" value={recommendForm.ontology_id} onChange={(v) => setRecommendForm(prev => ({ ...prev, ontology_id: v }))} options={recommendOntologies.map(o => ({ value: o.id, label: o.name }))} required />
						<Button type="submit" label="Get Recommendation" disabled={!recommendForm.project_id || !recommendForm.ontology_id} />
						{recommendResult ? (
							<div className="section-panel section-panel--neutral">
								<p className="section-panel-copy"><strong>Recommended:</strong> {recommendResult.recommended_type}</p>
								<p className="section-panel-copy"><strong>Confidence:</strong> {(recommendResult.confidence * 100).toFixed(0)}%</p>
								<p className="section-panel-copy"><strong>Reason:</strong> {recommendResult.reason}</p>
								<div className="inline-actions">
									<Button label="Use This" onClick={() => {
										setFormData(prev => ({ ...prev, model_type: recommendResult.recommended_type, project_id: recommendForm.project_id }));
										setShowRecommendModal(false);
										setShowModal(true);
									}} />
								</div>
							</div>
						) : null}
					</form>
				</Modal>

				<Modal open={showModal} onClose={() => setShowModal(false)} title="Create New ML Model">
					<form onSubmit={handleSubmit}>
						<div className="form-grid">
							<FormField label="Model Name" value={formData.name} onChange={(v) => setFormData({ ...formData, name: v })} required />
							<FormField label="Project" type="select" value={formData.project_id} onChange={(v) => setFormData({ ...formData, project_id: v })} options={projectOptions} required />
						</div>
						<div className="form-grid">
							<FormField label="Model Type" type="select" value={formData.model_type} onChange={(v) => setFormData({ ...formData, model_type: v })} options={['classification', 'regression', 'clustering', 'forecasting', 'anomaly_detection']} required />
							<FormField label="Version" value={formData.version} onChange={(v) => setFormData({ ...formData, version: v })} required />
						</div>
						<FormField label="Configuration (JSON)" type="textarea" value={formData.config} onChange={(v) => setFormData({ ...formData, config: v })} placeholder='{"hyperparameters": {}}' />
						<Button type="submit" label="Create Model" />
					</form>
				</Modal>
			</div>
		);
	};
})();

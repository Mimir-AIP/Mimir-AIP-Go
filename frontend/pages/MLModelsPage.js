(() => {
	const root = window.MimirApp = window.MimirApp || {};
	const pages = root.pages = root.pages || {};
	const { apiCall, deriveStorageConfigLabel, notify, confirmAction } = root.lib;
	const { ProjectContext } = root.context;
	const { useTaskWebSocket } = root.hooks;
	const { Button, Graph, Modal, Table } = root.components.primitives;
	const { FormField } = root.components.primitives;

	const MODEL_TYPE_OPTIONS = [
		{ value: 'decision_tree', label: 'Decision Tree' },
		{ value: 'random_forest', label: 'Random Forest' },
		{ value: 'regression', label: 'Regression' },
		{ value: 'neural_network', label: 'Neural Network' },
	];

	function emptyModelForm(projectId = '') {
		return {
			name: '',
			project_id: projectId,
			ontology_id: '',
			type: 'decision_tree',
			description: '',
			train_test_split: '0.8',
			random_seed: '42',
			max_iterations: '',
			learning_rate: '',
			batch_size: '',
			early_stopping_rounds: '',
			hyperparameters: '{}',
		};
	}

	function summarizeTaskState(status) {
		switch (status) {
		case 'queued':
		case 'scheduled':
		case 'spawned':
			return { label: 'Queued', className: 'status-pending' };
		case 'executing':
			return { label: 'Running', className: 'status-running' };
		case 'completed':
			return { label: 'Completed', className: 'status-completed' };
		case 'failed':
		case 'timeout':
		case 'cancelled':
			return { label: 'Failed', className: 'status-failed' };
		default:
			return null;
		}
	}

	function normalizeTrainingConfig(formData) {
		const cfg = {
			train_test_split: Number(formData.train_test_split || 0.8),
			random_seed: Number(formData.random_seed || 42),
		};
		if (formData.max_iterations) cfg.max_iterations = Number(formData.max_iterations);
		if (formData.learning_rate) cfg.learning_rate = Number(formData.learning_rate);
		if (formData.batch_size) cfg.batch_size = Number(formData.batch_size);
		if (formData.early_stopping_rounds) cfg.early_stopping_rounds = Number(formData.early_stopping_rounds);
		if (formData.hyperparameters && formData.hyperparameters.trim()) {
			cfg.hyperparameters = JSON.parse(formData.hyperparameters);
		}
		return cfg;
	}

	pages.MLModelsPage = function MLModelsPage() {
		const { activeProject, projects } = React.useContext(ProjectContext);
		const [models, setModels] = React.useState([]);
		const [loading, setLoading] = React.useState(true);
		const [loadError, setLoadError] = React.useState('');
		const [showModal, setShowModal] = React.useState(false);
		const [showRecommendModal, setShowRecommendModal] = React.useState(false);
		const [showTrainModal, setShowTrainModal] = React.useState(false);
		const [showDetailModal, setShowDetailModal] = React.useState(false);
		const [selectedModel, setSelectedModel] = React.useState(null);
		const [trainingTarget, setTrainingTarget] = React.useState(null);
		const [trainStorageIds, setTrainStorageIds] = React.useState([]);
		const [availableStorageConfigs, setAvailableStorageConfigs] = React.useState([]);
		const [availableOntologies, setAvailableOntologies] = React.useState([]);
		const [recommendForm, setRecommendForm] = React.useState({ project_id: '', ontology_id: '' });
		const [recommendOntologies, setRecommendOntologies] = React.useState([]);
		const [recommendResult, setRecommendResult] = React.useState(null);
		const [formData, setFormData] = React.useState(emptyModelForm());
		const [trainingMetrics, setTrainingMetrics] = React.useState({});
		const [taskState, setTaskState] = React.useState({});

		const loadOntologiesForProject = React.useCallback(async (projectId) => {
			if (!projectId) {
				setAvailableOntologies([]);
				return [];
			}
			try {
				const data = await apiCall(`/api/ontologies?project_id=${projectId}`);
				setAvailableOntologies(data || []);
				return data || [];
			} catch {
				setAvailableOntologies([]);
				return [];
			}
		}, []);

		React.useEffect(() => {
			if (!activeProject) return;
			setFormData(prev => ({ ...prev, project_id: activeProject.id, ontology_id: '' }));
			setRecommendForm({ project_id: activeProject.id, ontology_id: '' });
			loadOntologiesForProject(activeProject.id).then(data => setRecommendOntologies(data));
		}, [activeProject, loadOntologiesForProject]);

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
			if (!['ml_training', 'ml_inference'].includes(task.type)) return;
			if (activeProject?.id && task.project_id !== activeProject.id) return;
			const modelID = task.task_spec?.model_id || task.task_spec?.parameters?.model_id;
			if (modelID) {
				setTaskState(prev => ({ ...prev, [modelID]: task.status }));
				const metrics = task.task_spec?.parameters?.training_metrics;
				if (metrics) setTrainingMetrics(prev => ({ ...prev, [modelID]: metrics }));
			}
			if (['queued', 'scheduled', 'spawned', 'executing', 'completed', 'failed', 'timeout', 'cancelled'].includes(task.status)) {
				loadModels();
			}
		}, [activeProject?.id, loadModels]));

		const openCreateModal = async () => {
			setFormData(emptyModelForm(activeProject?.id || ''));
			await loadOntologiesForProject(activeProject?.id || '');
			setShowModal(true);
		};

		const handleSubmit = async (e) => {
			e.preventDefault();
			try {
				await apiCall('/api/ml-models', {
					method: 'POST',
					body: JSON.stringify({
						project_id: formData.project_id,
						ontology_id: formData.ontology_id,
						name: formData.name,
						description: formData.description,
						type: formData.type,
						training_config: normalizeTrainingConfig(formData),
					}),
				});
				setShowModal(false);
				setFormData(emptyModelForm(activeProject?.id || ''));
				notify({ tone: 'success', message: 'ML model created.' });
				loadModels();
			} catch (error) {
				notify({ tone: 'error', message: `Failed to create ML model: ${error.message}` });
			}
		};

		const handleDelete = async (id) => {
			const confirmed = await confirmAction({
				title: 'Delete ML model',
				message: 'Delete this ML model? Active training/inference work and digital twin references may block deletion.',
				confirmLabel: 'Delete model',
				variant: 'danger',
			});
			if (!confirmed) return;
			try {
				await apiCall(`/api/ml-models/${id}?project_id=${activeProject?.id || ''}`, { method: 'DELETE' });
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
			setRecommendForm({ project_id: projectId, ontology_id: '' });
			setRecommendResult(null);
			const data = await loadOntologiesForProject(projectId);
			setRecommendOntologies(data);
		};

		const openDetailModal = (row) => {
			setSelectedModel(row);
			setShowDetailModal(true);
		};

		const projectOptions = projects.map(p => ({ value: p.id, label: p.name }));
		const ontologyOptions = availableOntologies.map(o => ({ value: o.id, label: o.name }));
		const columns = [
			{ key: 'id', label: 'ID' },
			{ key: 'name', label: 'Name' },
			{ key: 'type', label: 'Type' },
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
						<Button label="+ New Model" onClick={openCreateModal} />
					</div>
				</div>

				{activeProject ? <div className="page-notice"><strong>Project scope:</strong> showing ML models for {activeProject.name}. Training and inference run asynchronously through work tasks.</div> : null}
				{loadError ? <div className="error-message">{loadError}</div> : null}

				{loading ? (
					<div className="loading">Loading ML models…</div>
				) : (
					<Table
						caption="Project ML models"
						columns={columns}
						data={models}
						emptyState={activeProject ? 'No ML models exist for this project yet.' : 'Select a project to inspect ML models.'}
						actions={(row) => {
							const task = summarizeTaskState(taskState[row.id]);
							return (
								<>
									{task ? <span className={`status-badge ${task.className}`}>{task.label}</span> : null}
									<Button label="Details" onClick={() => openDetailModal(row)} variant="secondary" />
									<Button label="Train" onClick={() => openTrainModal(row)} variant="secondary" />
									<Button label="Delete" onClick={() => handleDelete(row.id)} variant="danger" />
								</>
							);
						}}
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
						<p className="modal-copy">Select your project and ontology — the backend analyses ontology shape and project storage to score the available built-in model families.</p>
						<FormField label="Project" type="select" value={recommendForm.project_id} onChange={onRecommendProjectChange} options={projectOptions} required />
						<FormField label="Ontology" type="select" value={recommendForm.ontology_id} onChange={(v) => setRecommendForm(prev => ({ ...prev, ontology_id: v }))} options={recommendOntologies.map(o => ({ value: o.id, label: o.name }))} required />
						<Button type="submit" label="Get Recommendation" disabled={!recommendForm.project_id || !recommendForm.ontology_id} />
						{recommendResult ? (
							<div className="section-panel section-panel--neutral">
								<p className="section-panel-copy"><strong>Recommended:</strong> {recommendResult.recommended_type}</p>
								<p className="section-panel-copy"><strong>Score:</strong> {recommendResult.score}/100</p>
								<p className="section-panel-copy"><strong>Reasoning:</strong> {recommendResult.reasoning}</p>
								<div className="section-panel-copy"><strong>All Scores:</strong> {Object.entries(recommendResult.all_scores || {}).map(([type, score]) => `${type}: ${score}`).join(' · ') || '—'}</div>
								<div className="inline-actions">
									<Button label="Use This" onClick={async () => {
										const nextProjectId = recommendForm.project_id;
										const nextOntologies = await loadOntologiesForProject(nextProjectId);
										setFormData(prev => ({ ...emptyModelForm(nextProjectId), project_id: nextProjectId, ontology_id: recommendForm.ontology_id, type: recommendResult.recommended_type }));
										setAvailableOntologies(nextOntologies);
										setShowRecommendModal(false);
										setShowModal(true);
									}} />
								</div>
							</div>
						) : null}
					</form>
				</Modal>

				<Modal open={showDetailModal} onClose={() => setShowDetailModal(false)} title={`Model Details: ${selectedModel?.name || ''}`}>
					{selectedModel ? (
						<div className="section-panel section-panel--neutral">
							<div className="section-panel-copy"><strong>Type:</strong> {selectedModel.type}</div>
							<div className="section-panel-copy"><strong>Status:</strong> {selectedModel.status}</div>
							<div className="section-panel-copy"><strong>Ontology:</strong> {selectedModel.ontology_id}</div>
							<div className="section-panel-copy"><strong>Training Task:</strong> {selectedModel.training_task_id || '—'}</div>
							<div className="section-panel-copy"><strong>Artifact:</strong> {selectedModel.model_artifact_path || '—'}</div>
							<pre style={{ marginTop: '1rem', whiteSpace: 'pre-wrap', fontSize: '0.8rem' }}>{JSON.stringify({ training_config: selectedModel.training_config, performance_metrics: selectedModel.performance_metrics, metadata: selectedModel.metadata }, null, 2)}</pre>
						</div>
					) : null}
				</Modal>

				<Modal open={showModal} onClose={() => setShowModal(false)} title="Create New ML Model">
					<form onSubmit={handleSubmit}>
						<div className="form-grid">
							<FormField label="Model Name" value={formData.name} onChange={(v) => setFormData({ ...formData, name: v })} required />
							<FormField label="Project" type="select" value={formData.project_id} onChange={async (v) => { setFormData({ ...formData, project_id: v, ontology_id: '' }); await loadOntologiesForProject(v); }} options={projectOptions} required />
						</div>
						<div className="form-grid">
							<FormField label="Ontology" type="select" value={formData.ontology_id} onChange={(v) => setFormData({ ...formData, ontology_id: v })} options={ontologyOptions} required />
							<FormField label="Model Type" type="select" value={formData.type} onChange={(v) => setFormData({ ...formData, type: v })} options={MODEL_TYPE_OPTIONS} required />
						</div>
						<FormField label="Description" type="textarea" value={formData.description} onChange={(v) => setFormData({ ...formData, description: v })} />
						<div className="form-grid">
							<FormField label="Train/Test Split" type="number" value={formData.train_test_split} onChange={(v) => setFormData({ ...formData, train_test_split: v })} step="0.05" />
							<FormField label="Random Seed" type="number" value={formData.random_seed} onChange={(v) => setFormData({ ...formData, random_seed: v })} />
						</div>
						<div className="form-grid">
							<FormField label="Max Iterations" type="number" value={formData.max_iterations} onChange={(v) => setFormData({ ...formData, max_iterations: v })} />
							<FormField label="Learning Rate" type="number" value={formData.learning_rate} onChange={(v) => setFormData({ ...formData, learning_rate: v })} step="0.001" />
						</div>
						<div className="form-grid">
							<FormField label="Batch Size" type="number" value={formData.batch_size} onChange={(v) => setFormData({ ...formData, batch_size: v })} />
							<FormField label="Early Stopping Rounds" type="number" value={formData.early_stopping_rounds} onChange={(v) => setFormData({ ...formData, early_stopping_rounds: v })} />
						</div>
						<FormField label="Extra Hyperparameters (JSON)" type="textarea" value={formData.hyperparameters} onChange={(v) => setFormData({ ...formData, hyperparameters: v })} placeholder='{"max_depth": 5}' />
						<Button type="submit" label="Create Model" />
					</form>
				</Modal>
			</div>
		);
	};
})();

(() => {
	const root = window.MimirApp = window.MimirApp || {};
	const pages = root.pages = root.pages || {};
	const { apiCall, deriveStorageConfigLabel } = root.lib;
	const { ProjectContext } = root.context;
	const { useTaskWebSocket } = root.hooks;
	const { Button, FormField, Graph, Modal, Table } = root.components.primitives;

	pages.MLModelsPage = function MLModelsPage() {
		const { activeProject, projects } = React.useContext(ProjectContext);
		const [models, setModels] = React.useState([]);
		const [loading, setLoading] = React.useState(true);
		const [showModal, setShowModal] = React.useState(false);
		const [showRecommendModal, setShowRecommendModal] = React.useState(false);
		const [showTrainModal, setShowTrainModal] = React.useState(false);
		const [trainingTarget, setTrainingTarget] = React.useState(null);
		const [trainStorageIds, setTrainStorageIds] = React.useState([]);
		const [availableStorageConfigs, setAvailableStorageConfigs] = React.useState([]);
		const [recommendForm, setRecommendForm] = React.useState({
			project_id: '',
			ontology_id: '',
		});
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
			if (activeProject) {
				setFormData(prev => ({ ...prev, project_id: activeProject.id }));
				setRecommendForm(prev => ({ ...prev, project_id: activeProject.id, ontology_id: '' }));
				apiCall(`/api/ontologies?project_id=${activeProject.id}`)
					.then(data => setRecommendOntologies(data || []))
					.catch(() => {});
			}
		}, [activeProject]);

		useTaskWebSocket((task) => {
			if (task.type !== 'ml_training') return;
			const modelID = task.task_spec && task.task_spec.model_id;
			if (!modelID) return;
			const metrics = task.task_spec && task.task_spec.parameters && task.task_spec.parameters.training_metrics;
			if (!metrics) return;
			setTrainingMetrics((prev) => ({
				...prev,
				[modelID]: metrics,
			}));
			if (task.status === 'completed' || task.status === 'failed') {
				loadModels();
			}
		});

		const loadModels = async () => {
			setLoading(true);
			try {
				const projectId = activeProject?.id || '';
				const data = await apiCall(`/api/ml-models?project_id=${projectId}`);
				setModels(data || []);
			} catch (error) {
				console.error('Failed to load ML models:', error);
				setModels([]);
			}
			setLoading(false);
		};

		React.useEffect(() => {
			loadModels();
		}, [activeProject]);

		const handleSubmit = async (e) => {
			e.preventDefault();
			try {
				const data = {
					...formData,
					config: JSON.parse(formData.config),
				};
				await apiCall('/api/ml-models', {
					method: 'POST',
					body: JSON.stringify(data),
				});
				setShowModal(false);
				setFormData({ name: '', project_id: activeProject?.id || '', model_type: '', version: '1.0.0', config: '{}' });
				loadModels();
			} catch (error) {
				alert('Failed to create ML model: ' + error.message);
			}
		};

		const handleDelete = async (id) => {
			if (!confirm('Delete this ML model?')) return;
			try {
				await apiCall(`/api/ml-models/${id}`, { method: 'DELETE' });
				loadModels();
			} catch (error) {
				alert('Failed to delete ML model: ' + error.message);
			}
		};

		const openTrainModal = async (row) => {
			setTrainingTarget(row);
			try {
				const configs = await apiCall(`/api/storage/configs?project_id=${row.project_id}`);
				setAvailableStorageConfigs(configs || []);
			} catch {
				setAvailableStorageConfigs([]);
			}
			setTrainStorageIds([]);
			setShowTrainModal(true);
		};

		const handleTrain = async () => {
			try {
				await apiCall('/api/ml-models/train', {
					method: 'POST',
					body: JSON.stringify({ model_id: trainingTarget.id, storage_ids: trainStorageIds }),
				});
				setShowTrainModal(false);
			} catch (error) {
				alert('Failed to start training: ' + error.message);
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
				alert('Failed to get recommendation: ' + error.message);
			}
		};

		const onRecommendProjectChange = async (projectId) => {
			setRecommendForm(prev => ({ ...prev, project_id: projectId, ontology_id: '' }));
			setRecommendOntologies([]);
			if (projectId) {
				try {
					const data = await apiCall(`/api/ontologies?project_id=${projectId}`);
					setRecommendOntologies(data || []);
				} catch {
					setRecommendOntologies([]);
				}
			}
		};

		const projectOptions = projects.map(p => ({ value: p.id, label: p.name }));
		const columns = [
			{ key: 'id', label: 'ID' },
			{ key: 'name', label: 'Name' },
			{ key: 'model_type', label: 'Type' },
			{ key: 'version', label: 'Version' },
			{
				key: 'status',
				label: 'Status',
				render: (row) => <span className={`status-badge status-${row.status}`}>{row.status}</span>
			},
			{ key: 'project_id', label: 'Project ID' },
			{ key: 'created_at', label: 'Created', render: (row) => new Date(row.created_at).toLocaleDateString() },
		];

		return (
			<div className="content-section">
				<div className="section-header">
					<h2>ML Models</h2>
					<div style={{ display: 'flex', gap: '8px' }}>
						<Button label="Recommend" onClick={() => { setRecommendResult(null); setRecommendForm({ project_id: activeProject?.id || '', ontology_id: '' }); setRecommendOntologies([]); setShowRecommendModal(true); }} variant="secondary" />
						<Button label="+ New Model" onClick={() => setShowModal(true)} />
					</div>
				</div>

				{loading ? (
					<div className="loading">Loading ML models...</div>
				) : (
					<Table
						columns={columns}
						data={models}
						actions={(row) => (
							<>
								<Button label="Train" onClick={() => openTrainModal(row)} variant="secondary" />
								<Button label="Delete" onClick={() => handleDelete(row.id)} variant="danger" />
							</>
						)}
					/>
				)}

				{Object.keys(trainingMetrics).length > 0 && (
					<div style={{ marginTop: '24px' }}>
						<h3 style={{ color: 'var(--text-primary)', marginBottom: '12px' }}>Training Progress</h3>
						{Object.entries(trainingMetrics).map(([modelID, metrics]) => {
							const epochs = metrics.epochs || [];
							const loss = metrics.loss || [];
							const accuracy = metrics.accuracy || [];
							if (epochs.length === 0) return null;
							const graphData = {
								labels: epochs,
								datasets: [
									{
										label: 'Loss',
										data: loss,
										borderColor: '#ef4444',
										backgroundColor: 'rgba(239,68,68,0.1)',
										yAxisID: 'y',
									},
									{
										label: 'Accuracy',
										data: accuracy,
										borderColor: '#22c55e',
										backgroundColor: 'rgba(34,197,94,0.1)',
										yAxisID: 'y1',
									},
								],
							};
							const graphOptions = {
								scales: {
									y: { type: 'linear', position: 'left', title: { display: true, text: 'Loss' } },
									y1: { type: 'linear', position: 'right', title: { display: true, text: 'Accuracy' }, grid: { drawOnChartArea: false } },
								},
							};
							return (
								<div key={modelID} style={{ marginBottom: '16px', padding: '12px', background: 'var(--surface)', borderRadius: '6px' }}>
									<div style={{ fontSize: '12px', color: 'var(--text-secondary)', marginBottom: '8px' }}>Model: {modelID}</div>
									<Graph data={graphData} options={graphOptions} type="line" />
								</div>
							);
						})}
					</div>
				)}

				<Modal open={showTrainModal} onClose={() => setShowTrainModal(false)} title={`Train: ${trainingTarget?.name}`}>
					<p>Select storage configs to train on:</p>
					{availableStorageConfigs.length === 0 ? (
						<p style={{ color: 'var(--text-secondary)' }}>No storage configs found for this project.</p>
					) : (
						availableStorageConfigs.map(cfg => (
							<label key={cfg.id} style={{ display: 'block', marginBottom: '8px', cursor: 'pointer' }}>
								<input
									type="checkbox"
									checked={trainStorageIds.includes(cfg.id)}
									onChange={e => setTrainStorageIds(prev =>
										e.target.checked ? [...prev, cfg.id] : prev.filter(id => id !== cfg.id)
									)}
								/>
								{' '}{deriveStorageConfigLabel(cfg)}
							</label>
						))
					)}
					<div style={{ marginTop: '16px' }}>
						<Button label="Start Training" onClick={handleTrain} />
					</div>
				</Modal>

				<Modal open={showRecommendModal} onClose={() => setShowRecommendModal(false)} title="Recommend Model Type">
					<form onSubmit={handleRecommend}>
						<p style={{ color: 'var(--text-secondary)', fontSize: '0.875rem', marginBottom: '12px' }}>
							Select your project and ontology — the backend analyses your data automatically.
						</p>
						<FormField
							label="Project"
							type="select"
							value={recommendForm.project_id}
							onChange={onRecommendProjectChange}
							options={projectOptions}
							required
						/>
						<FormField
							label="Ontology"
							type="select"
							value={recommendForm.ontology_id}
							onChange={(v) => setRecommendForm(prev => ({ ...prev, ontology_id: v }))}
							options={recommendOntologies.map(o => ({ value: o.id, label: o.name }))}
							required
						/>
						<Button type="submit" label="Get Recommendation" disabled={!recommendForm.project_id || !recommendForm.ontology_id} />
						{recommendResult && (
							<div style={{ marginTop: '16px', padding: '12px', background: 'var(--surface)', borderRadius: '6px' }}>
								<strong>Recommended:</strong> {recommendResult.recommended_type}<br />
								<strong>Confidence:</strong> {(recommendResult.confidence * 100).toFixed(0)}%<br />
								<strong>Reason:</strong> {recommendResult.reason}
								<div style={{ marginTop: '12px' }}>
									<Button label="Use This" onClick={() => {
										setFormData(prev => ({
											...prev,
											model_type: recommendResult.recommended_type,
											project_id: recommendForm.project_id,
										}));
										setShowRecommendModal(false);
										setShowModal(true);
									}} />
								</div>
							</div>
						)}
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

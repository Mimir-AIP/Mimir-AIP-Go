(() => {
	const root = window.MimirApp = window.MimirApp || {};
	const pages = root.pages = root.pages || {};
	const { apiCall, deriveStorageConfigLabel } = root.lib;
	const { ProjectContext } = root.context;
	const { Button, FormField, Modal, Table } = root.components.primitives;

	pages.OntologiesPage = function OntologiesPage() {
		const { activeProject, projects } = React.useContext(ProjectContext);
		const [ontologies, setOntologies] = React.useState([]);
		const [loading, setLoading] = React.useState(true);
		const [showModal, setShowModal] = React.useState(false);
		const [showExtractionModal, setShowExtractionModal] = React.useState(false);
		const [storageConfigs, setStorageConfigs] = React.useState([]);
		const [formData, setFormData] = React.useState({
			name: '',
			project_id: '',
			schema: '{}',
		});
		const [extractionForm, setExtractionForm] = React.useState({
			project_id: '',
			storage_config_id: '',
			ontology_name: '',
		});

		React.useEffect(() => {
			if (activeProject) {
				setFormData(prev => ({ ...prev, project_id: activeProject.id }));
				setExtractionForm(prev => ({ ...prev, project_id: activeProject.id }));
			}
		}, [activeProject]);

		const loadOntologies = async () => {
			setLoading(true);
			try {
				const projectId = activeProject?.id || '';
				const data = await apiCall(`/api/ontologies?project_id=${projectId}`);
				setOntologies(data || []);
			} catch (error) {
				console.error('Failed to load ontologies:', error);
				setOntologies([]);
			}
			setLoading(false);
		};

		React.useEffect(() => {
			loadOntologies();
		}, [activeProject]);

		const openExtractionModal = async () => {
			if (activeProject?.id) {
				try {
					const configs = await apiCall(`/api/storage/configs?project_id=${activeProject.id}`);
					setStorageConfigs(configs || []);
				} catch {
					setStorageConfigs([]);
				}
			} else {
				setStorageConfigs([]);
			}
			setShowExtractionModal(true);
		};

		const handleSubmit = async (e) => {
			e.preventDefault();
			try {
				const data = {
					...formData,
					schema: JSON.parse(formData.schema),
				};
				await apiCall('/api/ontologies', {
					method: 'POST',
					body: JSON.stringify(data),
				});
				setShowModal(false);
				setFormData({ name: '', project_id: activeProject?.id || '', schema: '{}' });
				loadOntologies();
			} catch (error) {
				alert('Failed to create ontology: ' + error.message);
			}
		};

		const handleDelete = async (id) => {
			if (!confirm('Delete this ontology?')) return;
			try {
				await apiCall(`/api/ontologies/${id}`, { method: 'DELETE' });
				loadOntologies();
			} catch (error) {
				alert('Failed to delete ontology: ' + error.message);
			}
		};

		const handleApprove = async (id) => {
			try {
				await apiCall(`/api/ontologies/${id}`, { method: 'PUT', body: JSON.stringify({ status: 'active' }) });
				loadOntologies();
			} catch (error) {
				alert('Failed to approve ontology: ' + error.message);
			}
		};

		const handleReject = async (id) => {
			try {
				await apiCall(`/api/ontologies/${id}`, { method: 'PUT', body: JSON.stringify({ status: 'draft' }) });
				loadOntologies();
			} catch (error) {
				alert('Failed to reject ontology: ' + error.message);
			}
		};

		const handleExtraction = async (e) => {
			e.preventDefault();
			try {
				const result = await apiCall('/api/extraction/generate-ontology', {
					method: 'POST',
					body: JSON.stringify(extractionForm),
				});
				alert('Ontology extraction started!\n' + JSON.stringify(result, null, 2));
				setShowExtractionModal(false);
				setExtractionForm({ project_id: activeProject?.id || '', storage_config_id: '', ontology_name: '' });
				setTimeout(loadOntologies, 2000);
			} catch (error) {
				alert('Failed to start extraction: ' + error.message);
			}
		};

		const projectOptions = projects.map(p => ({ value: p.id, label: p.name }));
		const storageConfigOptions = storageConfigs.map(c => ({ value: c.id, label: deriveStorageConfigLabel(c) }));
		const columns = [
			{ key: 'id', label: 'ID' },
			{ key: 'name', label: 'Name' },
			{ key: 'project_id', label: 'Project ID' },
			{ key: 'version', label: 'Version' },
			{
				key: 'status',
				label: 'Status',
				render: (row) => <span className={`status-badge status-${row.status}`}>{row.status}</span>
			},
			{ key: 'created_at', label: 'Created', render: (row) => new Date(row.created_at).toLocaleDateString() },
		];

		return (
			<div className="content-section">
				<div className="section-header">
					<h2>Ontologies</h2>
					<div style={{ display: 'flex', gap: '8px' }}>
						<Button label="+ New Ontology" onClick={() => setShowModal(true)} />
						<Button label="Extract from Storage" onClick={openExtractionModal} variant="secondary" />
					</div>
				</div>

				{loading ? (
					<div className="loading">Loading ontologies...</div>
				) : (
					<Table
						columns={columns}
						data={ontologies}
						actions={(row) => (
							<>
								{row.status === 'needs_review' && <Button label="Approve" onClick={() => handleApprove(row.id)} variant="secondary" />}
								{row.status === 'needs_review' && <Button label="Reject" onClick={() => handleReject(row.id)} variant="secondary" />}
								{row.status === 'draft' && <Button label="Promote" onClick={() => handleApprove(row.id)} variant="secondary" />}
								<Button label="Delete" onClick={() => handleDelete(row.id)} variant="danger" />
							</>
						)}
					/>
				)}

				<Modal open={showModal} onClose={() => setShowModal(false)} title="Create New Ontology">
					<form onSubmit={handleSubmit}>
						<div className="form-grid">
							<FormField label="Ontology Name" value={formData.name} onChange={(v) => setFormData({ ...formData, name: v })} required />
							<FormField label="Project" type="select" value={formData.project_id} onChange={(v) => setFormData({ ...formData, project_id: v })} options={projectOptions} required />
						</div>
						<FormField label="Schema (JSON)" type="textarea" value={formData.schema} onChange={(v) => setFormData({ ...formData, schema: v })} placeholder='{"entities": [], "relationships": []}' required />
						<Button type="submit" label="Create Ontology" />
					</form>
				</Modal>

				<Modal open={showExtractionModal} onClose={() => setShowExtractionModal(false)} title="Extract Ontology from Storage">
					<form onSubmit={handleExtraction}>
						<div className="form-grid">
							<FormField label="Project" type="select" value={extractionForm.project_id} onChange={(v) => setExtractionForm({ ...extractionForm, project_id: v })} options={projectOptions} required />
							<FormField label="Storage Config" type="select" value={extractionForm.storage_config_id} onChange={(v) => setExtractionForm({ ...extractionForm, storage_config_id: v })} options={storageConfigOptions} required />
						</div>
						<FormField label="Ontology Name" value={extractionForm.ontology_name} onChange={(v) => setExtractionForm({ ...extractionForm, ontology_name: v })} placeholder="Extracted Ontology" required />
						<Button type="submit" label="Start Extraction" />
					</form>
				</Modal>
			</div>
		);
	};
})();

(() => {
	const root = window.MimirApp = window.MimirApp || {};
	const pages = root.pages = root.pages || {};
	const { apiCall, deriveStorageConfigLabel, notify, confirmAction } = root.lib;
	const { ProjectContext } = root.context;
	const { Button, FormField, Modal, Table } = root.components.primitives;

	pages.OntologiesPage = function OntologiesPage() {
		const { activeProject, projects } = React.useContext(ProjectContext);
		const [ontologies, setOntologies] = React.useState([]);
		const [loading, setLoading] = React.useState(true);
		const [loadError, setLoadError] = React.useState('');
		const [showModal, setShowModal] = React.useState(false);
		const [showExtractionModal, setShowExtractionModal] = React.useState(false);
		const [storageConfigs, setStorageConfigs] = React.useState([]);
		const [storageLoadError, setStorageLoadError] = React.useState('');
		const [formData, setFormData] = React.useState({ name: '', project_id: '', schema: '{}' });
		const [extractionForm, setExtractionForm] = React.useState({ project_id: '', storage_config_id: '', ontology_name: '' });

		React.useEffect(() => {
			if (activeProject) {
				setFormData(prev => ({ ...prev, project_id: activeProject.id }));
				setExtractionForm(prev => ({ ...prev, project_id: activeProject.id }));
			}
		}, [activeProject]);

		const loadOntologies = React.useCallback(async () => {
			setLoading(true);
			setLoadError('');
			try {
				const projectId = activeProject?.id || '';
				const data = await apiCall(`/api/ontologies?project_id=${projectId}`);
				setOntologies(data || []);
			} catch (error) {
				setOntologies([]);
				setLoadError(error.message || 'Failed to load ontologies.');
			} finally {
				setLoading(false);
			}
		}, [activeProject?.id]);

		React.useEffect(() => {
			loadOntologies();
		}, [loadOntologies]);

		const openExtractionModal = async () => {
			setStorageLoadError('');
			if (activeProject?.id) {
				try {
					const configs = await apiCall(`/api/storage/configs?project_id=${activeProject.id}`);
					setStorageConfigs(configs || []);
				} catch (error) {
					setStorageConfigs([]);
					setStorageLoadError(error.message || 'Failed to load storage configs.');
				}
			} else {
				setStorageConfigs([]);
			}
			setShowExtractionModal(true);
		};

		const handleSubmit = async (e) => {
			e.preventDefault();
			try {
				await apiCall('/api/ontologies', {
					method: 'POST',
					body: JSON.stringify({ ...formData, schema: JSON.parse(formData.schema) }),
				});
				setShowModal(false);
				setFormData({ name: '', project_id: activeProject?.id || '', schema: '{}' });
				notify({ tone: 'success', message: 'Ontology created.' });
				loadOntologies();
			} catch (error) {
				notify({ tone: 'error', message: `Failed to create ontology: ${error.message}` });
			}
		};

		const handleDelete = async (id) => {
			const confirmed = await confirmAction({
				title: 'Delete ontology',
				message: 'Delete this ontology? Linked workflows may stop validating against it.',
				confirmLabel: 'Delete ontology',
				variant: 'danger',
			});
			if (!confirmed) return;
			try {
				await apiCall(`/api/ontologies/${id}`, { method: 'DELETE' });
				notify({ tone: 'success', message: 'Ontology deleted.' });
				loadOntologies();
			} catch (error) {
				notify({ tone: 'error', message: `Failed to delete ontology: ${error.message}` });
			}
		};

		const handleApprove = async (id) => {
			try {
				await apiCall(`/api/ontologies/${id}`, { method: 'PUT', body: JSON.stringify({ status: 'active' }) });
				notify({ tone: 'success', message: 'Ontology promoted to active.' });
				loadOntologies();
			} catch (error) {
				notify({ tone: 'error', message: `Failed to approve ontology: ${error.message}` });
			}
		};

		const handleReject = async (id) => {
			try {
				await apiCall(`/api/ontologies/${id}`, { method: 'PUT', body: JSON.stringify({ status: 'draft' }) });
				notify({ tone: 'success', message: 'Ontology moved back to draft.' });
				loadOntologies();
			} catch (error) {
				notify({ tone: 'error', message: `Failed to reject ontology: ${error.message}` });
			}
		};

		const handleExtraction = async (e) => {
			e.preventDefault();
			try {
				const result = await apiCall('/api/extraction/generate-ontology', { method: 'POST', body: JSON.stringify(extractionForm) });
				notify({ tone: 'success', message: result?.message || 'Ontology extraction started. Refresh in a moment to see the generated ontology.' });
				setShowExtractionModal(false);
				setExtractionForm({ project_id: activeProject?.id || '', storage_config_id: '', ontology_name: '' });
				window.setTimeout(loadOntologies, 2000);
			} catch (error) {
				notify({ tone: 'error', message: `Failed to start extraction: ${error.message}` });
			}
		};

		const projectOptions = projects.map(p => ({ value: p.id, label: p.name }));
		const storageConfigOptions = storageConfigs.map(c => ({ value: c.id, label: deriveStorageConfigLabel(c) }));
		const columns = [
			{ key: 'id', label: 'ID' },
			{ key: 'name', label: 'Name' },
			{ key: 'project_id', label: 'Project ID' },
			{ key: 'version', label: 'Version' },
			{ key: 'status', label: 'Status', render: (row) => <span className={`status-badge status-${row.status}`}>{row.status}</span> },
			{ key: 'created_at', label: 'Created', render: (row) => new Date(row.created_at).toLocaleDateString() },
		];

		return (
			<div className="content-section">
				<div className="section-header">
					<h2>Ontologies</h2>
					<div className="inline-actions">
						<Button label="+ New Ontology" onClick={() => setShowModal(true)} />
						<Button label="Extract from Storage" onClick={openExtractionModal} variant="secondary" />
					</div>
				</div>

				{loadError ? <div className="error-message">{loadError}</div> : null}

				{loading ? (
					<div className="loading">Loading ontologies…</div>
				) : (
					<Table
						caption="Project ontologies"
						columns={columns}
						data={ontologies}
						emptyState={activeProject ? 'No ontologies exist for this project yet.' : 'Select a project to inspect ontologies.'}
						actions={(row) => (
							<>
								{row.status === 'needs_review' ? <Button label="Approve" onClick={() => handleApprove(row.id)} variant="secondary" /> : null}
								{row.status === 'needs_review' ? <Button label="Reject" onClick={() => handleReject(row.id)} variant="secondary" /> : null}
								{row.status === 'draft' ? <Button label="Promote" onClick={() => handleApprove(row.id)} variant="secondary" /> : null}
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
						{storageLoadError ? <div className="error-message">{storageLoadError}</div> : null}
						<div className="form-grid">
							<FormField label="Project" type="select" value={extractionForm.project_id} onChange={(v) => setExtractionForm({ ...extractionForm, project_id: v })} options={projectOptions} required />
							<FormField label="Storage Config" type="select" value={extractionForm.storage_config_id} onChange={(v) => setExtractionForm({ ...extractionForm, storage_config_id: v })} options={storageConfigOptions} required hint={storageConfigs.length ? undefined : 'No storage configs are currently available for this project.'} />
						</div>
						<FormField label="Ontology Name" value={extractionForm.ontology_name} onChange={(v) => setExtractionForm({ ...extractionForm, ontology_name: v })} placeholder="Extracted Ontology" required />
						<Button type="submit" label="Start Extraction" disabled={!storageConfigs.length} />
					</form>
				</Modal>
			</div>
		);
	};
})();

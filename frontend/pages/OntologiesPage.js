(() => {
	const root = window.MimirApp = window.MimirApp || {};
	const pages = root.pages = root.pages || {};
	const { apiCall, deriveStorageConfigLabel, notify, confirmAction } = root.lib;
	const { ProjectContext } = root.context;
	const { Button, FormField, Modal, Table } = root.components.primitives;

	function emptyOntologyForm(projectId = '') {
		return { name: '', project_id: projectId, description: '', version: '1.0', status: 'draft', content: '@prefix : <http://example.org/mimir#> .\n@prefix owl: <http://www.w3.org/2002/07/owl#> .\n\n:Entity a owl:Class .' };
	}

	pages.OntologiesPage = function OntologiesPage() {
		const { activeProject, projects } = React.useContext(ProjectContext);
		const [ontologies, setOntologies] = React.useState([]);
		const [loading, setLoading] = React.useState(true);
		const [loadError, setLoadError] = React.useState('');
		const [showModal, setShowModal] = React.useState(false);
		const [showExtractionModal, setShowExtractionModal] = React.useState(false);
		const [storageConfigs, setStorageConfigs] = React.useState([]);
		const [storageLoadError, setStorageLoadError] = React.useState('');
		const [formData, setFormData] = React.useState(emptyOntologyForm());
		const [extractionForm, setExtractionForm] = React.useState({ project_id: '', storage_ids: [], ontology_name: '', include_structured: true, include_unstructured: false });

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
					body: JSON.stringify({
						project_id: formData.project_id,
						name: formData.name,
						description: formData.description,
						version: formData.version,
						status: formData.status,
						content: formData.content,
					}),
				});
				setShowModal(false);
				setFormData(emptyOntologyForm(activeProject?.id || ''));
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
				await apiCall(`/api/ontologies/${id}?project_id=${activeProject?.id || ''}`, { method: 'DELETE' });
				notify({ tone: 'success', message: 'Ontology deleted.' });
				loadOntologies();
			} catch (error) {
				notify({ tone: 'error', message: `Failed to delete ontology: ${error.message}` });
			}
		};

		const handlePromote = async (id) => {
			try {
				await apiCall(`/api/ontologies/${id}?project_id=${activeProject?.id || ''}`, { method: 'PUT', body: JSON.stringify({ status: 'active' }) });
				notify({ tone: 'success', message: 'Ontology promoted to active.' });
				loadOntologies();
			} catch (error) {
				notify({ tone: 'error', message: `Failed to promote ontology: ${error.message}` });
			}
		};

		const handleExtraction = async (e) => {
			e.preventDefault();
			try {
				const result = await apiCall('/api/extraction/generate-ontology', {
					method: 'POST',
					body: JSON.stringify({
						project_id: extractionForm.project_id,
						storage_ids: extractionForm.storage_ids,
						ontology_name: extractionForm.ontology_name,
						include_structured: extractionForm.include_structured,
						include_unstructured: extractionForm.include_unstructured,
					}),
				});
				notify({ tone: 'success', message: result?.ontology?.name ? `Generated ontology: ${result.ontology.name}` : 'Ontology generation complete.' });
				setShowExtractionModal(false);
				setExtractionForm({ project_id: activeProject?.id || '', storage_ids: [], ontology_name: '', include_structured: true, include_unstructured: false });
				window.setTimeout(loadOntologies, 500);
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
						<Button label="Generate from Storage" onClick={openExtractionModal} variant="secondary" />
					</div>
				</div>

				{activeProject ? <div className="page-notice"><strong>Project scope:</strong> showing ontologies for {activeProject.name}.</div> : null}
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
								{row.status === 'draft' ? <Button label="Promote" onClick={() => handlePromote(row.id)} variant="secondary" /> : null}
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
						<div className="form-grid">
							<FormField label="Version" value={formData.version} onChange={(v) => setFormData({ ...formData, version: v })} />
							<FormField label="Status" type="select" value={formData.status} onChange={(v) => setFormData({ ...formData, status: v })} options={['draft', 'active', 'archived']} required />
						</div>
						<FormField label="Description" type="textarea" value={formData.description} onChange={(v) => setFormData({ ...formData, description: v })} />
						<FormField label="Ontology Content (OWL/Turtle)" type="textarea" value={formData.content} onChange={(v) => setFormData({ ...formData, content: v })} required rows={10} />
						<Button type="submit" label="Create Ontology" />
					</form>
				</Modal>

				<Modal open={showExtractionModal} onClose={() => setShowExtractionModal(false)} title="Generate Ontology from Storage">
					<form onSubmit={handleExtraction}>
						{storageLoadError ? <div className="error-message">{storageLoadError}</div> : null}
						<div className="form-grid">
							<FormField label="Project" type="select" value={extractionForm.project_id} onChange={(v) => setExtractionForm({ ...extractionForm, project_id: v })} options={projectOptions} required />
							<FormField label="Storage Config" type="select" value={extractionForm.storage_ids[0] || ''} onChange={(v) => setExtractionForm({ ...extractionForm, storage_ids: v ? [v] : [] })} options={storageConfigOptions} required hint={storageConfigs.length ? undefined : 'No storage configs are currently available for this project.'} />
						</div>
						<FormField label="Ontology Name" value={extractionForm.ontology_name} onChange={(v) => setExtractionForm({ ...extractionForm, ontology_name: v })} placeholder="Extracted Ontology" required />
						<label className="checkbox-row">
							<input type="checkbox" checked={extractionForm.include_structured} onChange={(e) => setExtractionForm({ ...extractionForm, include_structured: e.target.checked })} />
							Include structured extraction
						</label>
						<label className="checkbox-row">
							<input type="checkbox" checked={extractionForm.include_unstructured} onChange={(e) => setExtractionForm({ ...extractionForm, include_unstructured: e.target.checked })} />
							Include unstructured extraction
						</label>
						<Button type="submit" label="Generate Ontology" disabled={!storageConfigs.length} />
					</form>
				</Modal>
			</div>
		);
	};
})();

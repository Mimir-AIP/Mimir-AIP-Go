(() => {
	const root = window.MimirApp = window.MimirApp || {};
	const pages = root.pages = root.pages || {};
	const { apiCall, getProjectOnboardingMode, notify, confirmAction } = root.lib;
	const { ProjectContext } = root.context;
	const { Button, FormField, Modal, Table } = root.components.primitives;
	const { GuidedOnboardingPanel } = root.components.connectors;

	pages.ProjectsPage = function ProjectsPage() {
		const { projects, activeProject, setActiveProject, refreshProjects } = React.useContext(ProjectContext);
		const [showModal, setShowModal] = React.useState(false);
		const [savingMode, setSavingMode] = React.useState(false);
		const [formData, setFormData] = React.useState({ name: '', description: '', onboarding_mode: 'guided' });
		const [modeDraft, setModeDraft] = React.useState(getProjectOnboardingMode(activeProject));

		React.useEffect(() => {
			setModeDraft(getProjectOnboardingMode(activeProject));
		}, [activeProject?.id, activeProject?.settings?.onboarding_mode]);

		const savedMode = getProjectOnboardingMode(activeProject);
		const hasUnsavedModeChange = Boolean(activeProject?.id) && modeDraft !== savedMode;

		const handleSubmit = async (e) => {
			e.preventDefault();
			try {
				await apiCall('/api/projects', {
					method: 'POST',
					body: JSON.stringify({
						name: formData.name,
						description: formData.description,
						settings: { onboarding_mode: formData.onboarding_mode },
					}),
				});
				setShowModal(false);
				setFormData({ name: '', description: '', onboarding_mode: 'guided' });
				notify({ tone: 'success', message: 'Project created.' });
				await refreshProjects();
			} catch (error) {
				notify({ tone: 'error', message: `Failed to create project: ${error.message}` });
			}
		};

		const handleArchive = async (id) => {
			const confirmed = await confirmAction({
				title: 'Archive project',
				message: 'Archive this project? The record remains available for inspection, but it will be marked archived.',
				confirmLabel: 'Archive project',
				variant: 'danger',
			});
			if (!confirmed) return;
			try {
				await apiCall(`/api/projects/${id}/archive`, { method: 'POST' });
				notify({ tone: 'success', message: 'Project archived.' });
				await refreshProjects();
			} catch (error) {
				notify({ tone: 'error', message: `Failed to archive project: ${error.message}` });
			}
		};

		const handleDelete = async (id) => {
			const confirmed = await confirmAction({
				title: 'Delete project permanently',
				message: 'Delete this project permanently? Mimir will remove the project and its persisted project-owned resources. This cannot be undone.',
				confirmLabel: 'Delete project',
				variant: 'danger',
			});
			if (!confirmed) return;
			try {
				await apiCall(`/api/projects/${id}`, { method: 'DELETE' });
				notify({ tone: 'success', message: 'Project deleted.' });
				await refreshProjects();
			} catch (error) {
				notify({ tone: 'error', message: `Failed to delete project: ${error.message}` });
			}
		};

		const handleModeSave = async () => {
			if (!activeProject?.id || !hasUnsavedModeChange) return;
			setSavingMode(true);
			try {
				await apiCall(`/api/projects/${activeProject.id}`, {
					method: 'PUT',
					body: JSON.stringify({ settings: { ...activeProject.settings, onboarding_mode: modeDraft } }),
				});
				notify({ tone: 'success', message: `Onboarding mode saved as ${modeDraft}.` });
				await refreshProjects(activeProject.id);
			} catch (error) {
				notify({ tone: 'error', message: `Failed to update onboarding mode: ${error.message}` });
			} finally {
				setSavingMode(false);
			}
		};

		const columns = [
			{ key: 'id', label: 'ID' },
			{ key: 'name', label: 'Name' },
			{ key: 'description', label: 'Description' },
			{ key: 'onboarding_mode', label: 'Onboarding', render: row => getProjectOnboardingMode(row) },
			{ key: 'status', label: 'Status', render: (row) => <span className={`status-badge status-${row.status}`}>{row.status}</span> },
			{ key: 'created_at', label: 'Created', render: (row) => new Date(row.metadata?.created_at || row.created_at).toLocaleDateString() },
		];

		return (
			<div className="content-section">
				<div className="section-header">
					<h2>Projects</h2>
					<Button label="+ New Project" onClick={() => setShowModal(true)} />
				</div>

				<Table
					caption="Projects"
					columns={columns}
					data={projects}
					emptyState="No projects yet. Create one to start configuring the platform."
					actions={(row) => (
						<>
							{activeProject?.id === row.id ? (
								<span className="status-badge status-active">Active</span>
							) : (
								<Button label="Use" onClick={() => setActiveProject(row)} variant="secondary" />
							)}
							<Button label="Archive" onClick={() => handleArchive(row.id)} variant="secondary" />
							<Button label="Delete" onClick={() => handleDelete(row.id)} variant="danger" />
						</>
					)}
				/>

				<div className="section-panel">
					<div className="section-panel-header">
						<div>
							<h3 className="section-panel-title">Onboarding</h3>
							<p className="section-panel-copy">Choose between guided connector setup and the existing advanced manual workflow.</p>
						</div>
					</div>
					{activeProject ? (
						<>
							<div className="form-grid">
								<div className="form-group">
									<label>Active Project</label>
									<div className="field-static">{activeProject.name}</div>
								</div>
								<FormField
									label="Onboarding Mode"
									type="select"
									value={modeDraft}
									onChange={setModeDraft}
									options={[{ value: 'guided', label: 'Guided' }, { value: 'advanced', label: 'Advanced' }]}
									required
									hint="Changes apply only after you save."
								/>
							</div>
							{hasUnsavedModeChange ? (
								<div className="page-notice page-notice--warning">
									<strong>Unsaved change:</strong> the active mode is still {savedMode}. Save to switch the workspace.
								</div>
							) : null}
							<div className="inline-actions">
								<Button label={savingMode ? 'Saving…' : 'Save Onboarding Mode'} onClick={handleModeSave} variant="secondary" disabled={!hasUnsavedModeChange || savingMode} />
							</div>
							{savedMode === 'guided' ? (
								<GuidedOnboardingPanel project={activeProject} />
							) : (
								<div className="empty-state">Advanced mode keeps the manual pages available: Storage, Pipelines, Ontologies, ML Models, and Digital Twins.</div>
							)}
						</>
					) : (
						<div className="empty-state">Select a project to configure onboarding, insights, and reviews.</div>
					)}
				</div>

				<Modal open={showModal} onClose={() => setShowModal(false)} title="Create New Project">
					<form onSubmit={handleSubmit}>
						<FormField label="Project Name" value={formData.name} onChange={(v) => setFormData({ ...formData, name: v })} required />
						<FormField label="Description" type="textarea" value={formData.description} onChange={(v) => setFormData({ ...formData, description: v })} />
						<FormField
							label="Onboarding Mode"
							type="select"
							value={formData.onboarding_mode}
							onChange={(v) => setFormData({ ...formData, onboarding_mode: v })}
							options={[{ value: 'guided', label: 'Guided' }, { value: 'advanced', label: 'Advanced' }]}
							required
						/>
						<Button type="submit" label="Create Project" />
					</form>
				</Modal>
			</div>
		);
	};
})();

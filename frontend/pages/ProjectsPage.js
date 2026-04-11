(() => {
	const root = window.MimirApp = window.MimirApp || {};
	const pages = root.pages = root.pages || {};
	const { apiCall, notify, confirmAction } = root.lib;
	const { ProjectContext } = root.context;
	const { Button, FormField, Modal, Table } = root.components.primitives;

	pages.ProjectsPage = function ProjectsPage() {
		const { projects, activeProject, setActiveProject, refreshProjects } = React.useContext(ProjectContext);
		const [showModal, setShowModal] = React.useState(false);
		const [formData, setFormData] = React.useState({ name: '', description: '' });

		const handleSubmit = async (e) => {
			e.preventDefault();
			try {
				await apiCall('/api/projects', {
					method: 'POST',
					body: JSON.stringify({
						name: formData.name,
						description: formData.description,
					}),
				});
				setShowModal(false);
				setFormData({ name: '', description: '' });
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

		const columns = [
			{ key: 'id', label: 'ID' },
			{ key: 'name', label: 'Name' },
			{ key: 'description', label: 'Description' },
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
							<h3 className="section-panel-title">Active Workspace</h3>
							<p className="section-panel-copy">Projects define workspace scope. Configure storage, pipelines, ontologies, ML models, and digital twins through their dedicated pages.</p>
						</div>
					</div>
					{activeProject ? (
						<div className="form-grid">
							<div className="form-group">
								<label>Active Project</label>
								<div className="field-static">{activeProject.name}</div>
							</div>
							<div className="form-group">
								<label>Status</label>
								<div className="field-static">{activeProject.status}</div>
							</div>
						</div>
					) : (
						<div className="empty-state">Select a project to work on its resources.</div>
					)}
				</div>

				<Modal open={showModal} onClose={() => setShowModal(false)} title="Create New Project">
					<form onSubmit={handleSubmit}>
						<FormField label="Project Name" value={formData.name} onChange={(v) => setFormData({ ...formData, name: v })} required />
						<FormField label="Description" type="textarea" value={formData.description} onChange={(v) => setFormData({ ...formData, description: v })} />
						<Button type="submit" label="Create Project" />
					</form>
				</Modal>
			</div>
		);
	};
})();

(() => {
	const root = window.MimirApp = window.MimirApp || {};
	const pages = root.pages = root.pages || {};
	const { apiCall, getProjectOnboardingMode } = root.lib;
	const { ProjectContext } = root.context;
	const { Button, FormField, Modal, Table } = root.components.primitives;
	const { GuidedOnboardingPanel } = root.components.connectors;

	pages.ProjectsPage = function ProjectsPage() {
		const { projects, activeProject, setActiveProject, refreshProjects } = React.useContext(ProjectContext);
		const [showModal, setShowModal] = React.useState(false);
		const [formData, setFormData] = React.useState({
			name: '',
			description: '',
			onboarding_mode: 'guided',
		});
		const [modeDraft, setModeDraft] = React.useState(getProjectOnboardingMode(activeProject));

		React.useEffect(() => {
			setModeDraft(getProjectOnboardingMode(activeProject));
		}, [activeProject?.id, activeProject?.settings?.onboarding_mode]);

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
				await refreshProjects();
			} catch (error) {
				alert('Failed to create project: ' + error.message);
			}
		};

		const handleDelete = async (id) => {
			if (!confirm('Delete this project?')) return;
			try {
				await apiCall(`/api/projects/${id}`, { method: 'DELETE' });
				await refreshProjects();
			} catch (error) {
				alert('Failed to delete project: ' + error.message);
			}
		};

		const handleModeSave = async () => {
			if (!activeProject?.id) return;
			try {
				await apiCall(`/api/projects/${activeProject.id}`, {
					method: 'PUT',
					body: JSON.stringify({ settings: { ...activeProject.settings, onboarding_mode: modeDraft } }),
				});
				await refreshProjects(activeProject.id);
			} catch (error) {
				alert('Failed to update onboarding mode: ' + error.message);
			}
		};

		const columns = [
			{ key: 'id', label: 'ID' },
			{ key: 'name', label: 'Name' },
			{ key: 'description', label: 'Description' },
			{ key: 'onboarding_mode', label: 'Onboarding', render: row => getProjectOnboardingMode(row) },
			{
				key: 'status',
				label: 'Status',
				render: (row) => <span className={`status-badge status-${row.status}`}>{row.status}</span>
			},
			{
				key: 'created_at',
				label: 'Created',
				render: (row) => new Date(row.metadata?.created_at || row.created_at).toLocaleDateString()
			},
		];

		return (
			<div className="content-section">
				<div className="section-header">
					<h2>Projects</h2>
					<Button label="+ New Project" onClick={() => setShowModal(true)} />
				</div>

				<Table
					columns={columns}
					data={projects}
					actions={(row) => (
						<>
							{activeProject?.id === row.id ? (
								<span className="status-badge status-active">Active</span>
							) : (
								<Button label="Use" onClick={() => setActiveProject(row)} variant="secondary" />
							)}
							<Button label="Delete" onClick={() => handleDelete(row.id)} variant="danger" />
						</>
					)}
				/>

				<div style={{ marginTop: '24px', padding: '18px', border: '1px solid var(--border)', borderRadius: '8px', background: 'rgba(255,153,0,0.04)' }}>
					<div className="section-header" style={{ marginBottom: '12px' }}>
						<div>
							<h3 style={{ marginBottom: '4px' }}>Onboarding</h3>
							<div style={{ color: 'var(--text-secondary)' }}>Choose between guided connector setup and the existing advanced manual workflow.</div>
						</div>
					</div>
					{activeProject ? (
						<>
							<div className="form-grid">
								<div className="form-group"><label>Active Project</label><div style={{ padding: '10px 12px', border: '1px solid var(--border)', borderRadius: '4px', background: 'var(--surface)' }}>{activeProject.name}</div></div>
								<FormField
									label="Onboarding Mode"
									type="select"
									value={modeDraft}
									onChange={setModeDraft}
									options={[{ value: 'guided', label: 'guided' }, { value: 'advanced', label: 'advanced' }]}
									required
								/>
							</div>
							<div style={{ marginBottom: '16px' }}>
								<Button label="Save Onboarding Mode" onClick={handleModeSave} variant="secondary" />
							</div>
							{modeDraft === 'guided' ? (
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
							options={[{ value: 'guided', label: 'guided' }, { value: 'advanced', label: 'advanced' }]}
							required
						/>
						<Button type="submit" label="Create Project" />
					</form>
				</Modal>
			</div>
		);
	};
})();

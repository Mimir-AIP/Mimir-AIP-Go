(() => {
	const root = window.MimirApp = window.MimirApp || {};
	const pages = root.pages = root.pages || {};
	const { apiCall, notify, confirmAction } = root.lib;
	const { Button, FormField, Modal, Table } = root.components.primitives;

	pages.PluginsPage = function PluginsPage() {
		const [plugins, setPlugins] = React.useState([]);
		const [loading, setLoading] = React.useState(true);
		const [loadError, setLoadError] = React.useState('');
		const [showModal, setShowModal] = React.useState(false);
		const [formData, setFormData] = React.useState({ repository_url: '', git_ref: 'main' });

		const loadPlugins = React.useCallback(async () => {
			setLoading(true);
			setLoadError('');
			try {
				const data = await apiCall('/api/plugins');
				setPlugins(data || []);
			} catch (error) {
				setLoadError(error.message || 'Failed to load plugins.');
				setPlugins([]);
			} finally {
				setLoading(false);
			}
		}, []);

		React.useEffect(() => {
			loadPlugins();
		}, [loadPlugins]);

		const handleSubmit = async (e) => {
			e.preventDefault();
			try {
				await apiCall('/api/plugins', { method: 'POST', body: JSON.stringify(formData) });
				setShowModal(false);
				setFormData({ repository_url: '', git_ref: 'main' });
				notify({ tone: 'success', message: 'Plugin install queued.' });
				loadPlugins();
			} catch (error) {
				notify({ tone: 'error', message: `Failed to install plugin: ${error.message}` });
			}
		};

		const handleDelete = async (name) => {
			const confirmed = await confirmAction({
				title: 'Uninstall plugin',
				message: `Uninstall plugin "${name}"?`,
				confirmLabel: 'Uninstall plugin',
				variant: 'danger',
			});
			if (!confirmed) return;
			try {
				await apiCall(`/api/plugins/${name}`, { method: 'DELETE' });
				notify({ tone: 'success', message: 'Plugin uninstalled.' });
				loadPlugins();
			} catch (error) {
				notify({ tone: 'error', message: `Failed to uninstall plugin: ${error.message}` });
			}
		};

		const handleUpdate = async (name) => {
			try {
				await apiCall(`/api/plugins/${name}`, { method: 'PUT', body: JSON.stringify({}) });
				notify({ tone: 'success', message: `Plugin "${name}" updated.` });
				loadPlugins();
			} catch (error) {
				notify({ tone: 'error', message: `Failed to update plugin: ${error.message}` });
			}
		};

		const columns = [
			{ key: 'name', label: 'Name' },
			{ key: 'version', label: 'Version' },
			{ key: 'description', label: 'Description' },
			{ key: 'author', label: 'Author' },
			{ key: 'status', label: 'Status', render: (row) => <span className={`status-badge status-${row.status || 'active'}`}>{row.status || 'installed'}</span> },
		];

		return (
			<div className="content-section">
				<div className="section-header">
					<h2>Plugin Management</h2>
					<Button label="+ Install Plugin" onClick={() => setShowModal(true)} />
				</div>

				{loadError ? <div className="error-message">{loadError}</div> : null}

				{loading ? (
					<div className="loading">Loading plugins…</div>
				) : (
					<Table
						caption="Installed plugins"
						columns={columns}
						data={plugins}
						emptyState="No plugins are installed yet. Install one from a Git repository to extend the platform."
						actions={(row) => (
							<>
								<Button label="Update" onClick={() => handleUpdate(row.name)} variant="secondary" />
								<Button label="Uninstall" onClick={() => handleDelete(row.name)} variant="danger" />
							</>
						)}
					/>
				)}

				<Modal open={showModal} onClose={() => setShowModal(false)} title="Install Plugin">
					<form onSubmit={handleSubmit}>
						<FormField
							label="Git Repository URL"
							value={formData.repository_url}
							onChange={(v) => setFormData({ ...formData, repository_url: v })}
							placeholder="https://github.com/user/plugin.git"
							required
							hint="Use the Git URL for the plugin source repository."
						/>
						<FormField
							label="Version or Branch"
							value={formData.git_ref}
							onChange={(v) => setFormData({ ...formData, git_ref: v })}
							placeholder="main"
							hint="Pin to a branch, tag, or commit for repeatable installs."
						/>
						<Button type="submit" label="Install Plugin" />
					</form>
				</Modal>
			</div>
		);
	};
})();

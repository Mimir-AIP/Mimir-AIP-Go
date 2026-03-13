(() => {
	const root = window.MimirApp = window.MimirApp || {};
	const pages = root.pages = root.pages || {};
	const { apiCall } = root.lib;
	const { Button, FormField, Modal, Table } = root.components.primitives;

	pages.PluginsPage = function PluginsPage() {
		const [plugins, setPlugins] = React.useState([]);
		const [loading, setLoading] = React.useState(true);
		const [showModal, setShowModal] = React.useState(false);
		const [formData, setFormData] = React.useState({
			repository_url: '',
			git_ref: 'main',
		});

		const loadPlugins = async () => {
			setLoading(true);
			try {
				const data = await apiCall('/api/plugins');
				setPlugins(data || []);
			} catch (error) {
				console.error('Failed to load plugins:', error);
				setPlugins([]);
			}
			setLoading(false);
		};

		React.useEffect(() => {
			loadPlugins();
		}, []);

		const handleSubmit = async (e) => {
			e.preventDefault();
			try {
				await apiCall('/api/plugins', {
					method: 'POST',
					body: JSON.stringify(formData),
				});
				setShowModal(false);
				setFormData({ repository_url: '', git_ref: 'main' });
				loadPlugins();
			} catch (error) {
				alert('Failed to install plugin: ' + error.message);
			}
		};

		const handleDelete = async (name) => {
			if (!confirm(`Uninstall plugin "${name}"?`)) return;
			try {
				await apiCall(`/api/plugins/${name}`, { method: 'DELETE' });
				loadPlugins();
			} catch (error) {
				alert('Failed to uninstall plugin: ' + error.message);
			}
		};

		const handleUpdate = async (name) => {
			try {
				await apiCall(`/api/plugins/${name}`, { method: 'PUT', body: JSON.stringify({}) });
				alert('Plugin updated!');
				loadPlugins();
			} catch (error) {
				alert('Failed to update plugin: ' + error.message);
			}
		};

		const columns = [
			{ key: 'name', label: 'Name' },
			{ key: 'version', label: 'Version' },
			{ key: 'description', label: 'Description' },
			{ key: 'author', label: 'Author' },
			{
				key: 'status',
				label: 'Status',
				render: (row) => <span className={`status-badge status-${row.status || 'active'}`}>{row.status || 'installed'}</span>
			},
		];

		return (
			<div className="content-section">
				<div className="section-header">
					<h2>Plugin Management</h2>
					<Button label="+ Install Plugin" onClick={() => setShowModal(true)} />
				</div>

				{loading ? (
					<div className="loading">Loading plugins...</div>
				) : (
					<Table
						columns={columns}
						data={plugins}
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
						/>
						<FormField
							label="Version/Branch"
							value={formData.git_ref}
							onChange={(v) => setFormData({ ...formData, git_ref: v })}
							placeholder="main"
						/>
						<Button type="submit" label="Install Plugin" />
					</form>
				</Modal>
			</div>
		);
	};
})();

(() => {
	const root = window.MimirApp = window.MimirApp || {};
	const pages = root.pages = root.pages || {};
	const { apiCall, notify, confirmAction } = root.lib;
	const { Button, FormField, Modal, Table, Tabs } = root.components.primitives;

	const TAB_PIPELINE = 'Pipeline';
	const TAB_ML = 'ML Providers';
	const TAB_STORAGE = 'Storage';
	const TAB_LLM = 'LLM Providers';
	const TABS = [TAB_PIPELINE, TAB_ML, TAB_STORAGE, TAB_LLM];

	function emptyInstallForm() {
		return { repository_url: '', git_ref: 'main' };
	}

	function pluginActions(plugin) {
		return Array.isArray(plugin?.actions) ? plugin.actions : [];
	}

	function isMLPlugin(plugin) {
		return Boolean(plugin?.plugin_definition?.ml_provider);
	}

	function isPipelinePlugin(plugin) {
		return pluginActions(plugin).length > 0;
	}

	function statusBadge(status) {
		const value = status || 'active';
		return <span className={`status-badge status-${value}`}>{value}</span>;
	}

	function shortCommit(value) {
		return value ? String(value).slice(0, 12) : '—';
	}

	pages.PluginsPage = function PluginsPage() {
		const [activeTab, setActiveTab] = React.useState(TAB_PIPELINE);
		const [plugins, setPlugins] = React.useState([]);
		const [storagePlugins, setStoragePlugins] = React.useState([]);
		const [llmProviders, setLLMProviders] = React.useState([]);
		const [mlProviders, setMLProviders] = React.useState([]);
		const [loading, setLoading] = React.useState(true);
		const [loadError, setLoadError] = React.useState('');
		const [showModal, setShowModal] = React.useState(false);
		const [formData, setFormData] = React.useState(emptyInstallForm());

		const loadAll = React.useCallback(async () => {
			setLoading(true);
			setLoadError('');
			try {
				const [pluginData, storageData, llmData, mlData] = await Promise.all([
					apiCall('/api/plugins'),
					apiCall('/api/storage-plugins'),
					apiCall('/api/llm/providers'),
					apiCall('/api/ml-providers'),
				]);
				setPlugins(pluginData || []);
				setStoragePlugins(storageData || []);
				setLLMProviders(llmData || []);
				setMLProviders(mlData || []);
			} catch (error) {
				setLoadError(error.message || 'Failed to load plugin registry.');
				setPlugins([]);
				setStoragePlugins([]);
				setLLMProviders([]);
				setMLProviders([]);
			} finally {
				setLoading(false);
			}
		}, []);

		React.useEffect(() => {
			loadAll();
		}, [loadAll]);

		const openInstallModal = () => {
			setFormData(emptyInstallForm());
			setShowModal(true);
		};

		const installEndpoint = () => {
			if (activeTab === TAB_STORAGE) return '/api/storage-plugins';
			if (activeTab === TAB_LLM) return '/api/llm/providers';
			return '/api/plugins';
		};

		const installLabel = () => {
			if (activeTab === TAB_STORAGE) return 'Install Storage Plugin';
			if (activeTab === TAB_LLM) return 'Install LLM Provider';
			if (activeTab === TAB_ML) return 'Install ML Provider Plugin';
			return 'Install Pipeline Plugin';
		};

		const handleSubmit = async (e) => {
			e.preventDefault();
			try {
				await apiCall(installEndpoint(), { method: 'POST', body: JSON.stringify(formData) });
				setShowModal(false);
				setFormData(emptyInstallForm());
				notify({ tone: 'success', message: `${installLabel()} completed.` });
				loadAll();
			} catch (error) {
				notify({ tone: 'error', message: `Install failed: ${error.message}` });
			}
		};

		const handleDelete = async ({ name, endpoint, label }) => {
			const confirmed = await confirmAction({
				title: `Uninstall ${label}`,
				message: `Uninstall "${name}"? Running Go processes may retain already loaded plugin code until restart.`,
				confirmLabel: 'Uninstall',
				variant: 'danger',
			});
			if (!confirmed) return;
			try {
				await apiCall(`${endpoint}/${encodeURIComponent(name)}`, { method: 'DELETE' });
				notify({ tone: 'success', message: `${label} uninstalled.` });
				loadAll();
			} catch (error) {
				notify({ tone: 'error', message: `Uninstall failed: ${error.message}` });
			}
		};

		const handleUpdatePlugin = async (name) => {
			try {
				await apiCall(`/api/plugins/${encodeURIComponent(name)}`, { method: 'PUT', body: JSON.stringify({}) });
				notify({ tone: 'success', message: `Plugin "${name}" validated and updated.` });
				loadAll();
			} catch (error) {
				notify({ tone: 'error', message: `Update failed: ${error.message}` });
			}
		};

		const pipelinePlugins = plugins.filter(isPipelinePlugin);
		const mlPluginByProvider = new Map(
			plugins.filter(isMLPlugin).map(plugin => [plugin.plugin_definition?.ml_provider?.name || plugin.name, plugin])
		);
		const mlRows = mlProviders.map(provider => ({
			...provider,
			plugin: mlPluginByProvider.get(provider.name),
			source: provider.name === 'builtin' ? 'built-in' : (mlPluginByProvider.has(provider.name) ? 'plugin' : 'external'),
		}));

		const pluginColumns = [
			{ key: 'name', label: 'Name' },
			{ key: 'version', label: 'Version' },
			{ key: 'actions', label: 'Actions', render: row => pluginActions(row).map(action => action.name).join(', ') || '—' },
			{ key: 'git_commit_hash', label: 'Commit', render: row => shortCommit(row.git_commit_hash) },
			{ key: 'status', label: 'Status', render: row => statusBadge(row.status) },
		];
		const providerColumns = [
			{ key: 'name', label: 'Provider' },
			{ key: 'display_name', label: 'Display Name' },
			{ key: 'source', label: 'Source' },
			{ key: 'models', label: 'Models', render: row => (row.models || []).map(model => model.display_name || model.name).join(', ') || '—' },
			{ key: 'capabilities', label: 'Capabilities', render: row => (row.capabilities || []).join(', ') || '—' },
		];
		const externalColumns = [
			{ key: 'name', label: 'Name' },
			{ key: 'repository_url', label: 'Repository' },
			{ key: 'git_commit_hash', label: 'Commit', render: row => shortCommit(row.git_commit_hash) },
			{ key: 'status', label: 'Status', render: row => statusBadge(row.status) },
			{ key: 'error_message', label: 'Last Error' },
		];

		const renderTable = () => {
			if (loading) return <div className="loading">Loading plugin registry…</div>;
			if (activeTab === TAB_PIPELINE) {
				return <Table caption="Pipeline plugins" columns={pluginColumns} data={pipelinePlugins} emptyState="No pipeline plugins are installed." actions={(row) => <><Button label="Update" onClick={() => handleUpdatePlugin(row.name)} variant="secondary" /><Button label="Uninstall" onClick={() => handleDelete({ name: row.name, endpoint: '/api/plugins', label: 'plugin' })} variant="danger" /></>} />;
			}
			if (activeTab === TAB_ML) {
				return <Table caption="ML providers" columns={providerColumns} data={mlRows} emptyState="No ML providers are registered." actions={(row) => row.plugin ? <><Button label="Update" onClick={() => handleUpdatePlugin(row.plugin.name)} variant="secondary" /><Button label="Uninstall" onClick={() => handleDelete({ name: row.plugin.name, endpoint: '/api/plugins', label: 'ML provider plugin' })} variant="danger" /></> : <span className="status-badge status-active">managed</span>} />;
			}
			if (activeTab === TAB_STORAGE) {
				return <Table caption="Storage plugins" columns={externalColumns} data={storagePlugins} emptyState="No external storage plugins are installed." actions={(row) => <Button label="Uninstall" onClick={() => handleDelete({ name: row.name, endpoint: '/api/storage-plugins', label: 'storage plugin' })} variant="danger" />} />;
			}
			return <Table caption="LLM providers" columns={externalColumns} data={llmProviders} emptyState="No external LLM providers are installed." actions={(row) => <Button label="Uninstall" onClick={() => handleDelete({ name: row.name, endpoint: '/api/llm/providers', label: 'LLM provider' })} variant="danger" />} />;
		};

		return (
			<div className="content-section">
				<div className="section-header">
					<div>
						<h2>Plugin Management</h2>
						<p className="section-subtitle">Install and audit runtime extensions. Plugins are trusted Go code and cannot be unloaded from running processes.</p>
					</div>
					<Button label={`+ ${installLabel()}`} onClick={openInstallModal} />
				</div>

				<Tabs tabs={TABS} activeTab={activeTab} onTabChange={setActiveTab} />
				{loadError ? <div className="error-message">{loadError}</div> : null}
				<div className="page-notice"><strong>Runtime boundary:</strong> install validates against the orchestrator image; workers compile their own local artifact before task execution.</div>
				{renderTable()}

				<Modal open={showModal} onClose={() => setShowModal(false)} title={installLabel()}>
					<form onSubmit={handleSubmit}>
						<FormField
							label="Git Repository URL"
							value={formData.repository_url}
							onChange={(v) => setFormData({ ...formData, repository_url: v })}
							placeholder="https://github.com/user/plugin.git"
							required
							hint="Install only trusted repositories; plugin code runs with Mimir process privileges."
						/>
						<FormField
							label="Git Ref"
							value={formData.git_ref}
							onChange={(v) => setFormData({ ...formData, git_ref: v })}
							placeholder="main, v1.2.3, or commit SHA"
							hint="Use an immutable commit SHA for repeatable production installs."
						/>
						<Button type="submit" label={installLabel()} />
					</form>
				</Modal>
			</div>
		);
	};
})();

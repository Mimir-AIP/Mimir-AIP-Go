(() => {
	const root = window.MimirApp = window.MimirApp || {};
	root.components = root.components || {};
	root.pages = root.pages || {};
	root.lib = root.lib || {};
	root.context = root.context || {};
	root.hooks = root.hooks || {};

	const { apiCall } = root.lib;
	const { ProjectContext } = root.context;
	const { useProjectStateSummary } = root.hooks;
	const { Button, Modal } = root.components.primitives;
	const {
		ProjectsPage,
		PipelinesPage,
		OntologiesPage,
		MLModelsPage,
		DigitalTwinsPage,
		StoragePage,
		InsightsReviewPage,
		PluginsPage,
		WorkTasksPage,
	} = root.pages;

	function App() {
		const [currentPage, setCurrentPage] = React.useState('Projects');
		const [sidebarOpen, setSidebarOpen] = React.useState(false);
		const [projects, setProjects] = React.useState([]);
		const [activeProjectId, setActiveProjectId] = React.useState(undefined);
		const [notifications, setNotifications] = React.useState([]);
		const [confirmState, setConfirmState] = React.useState(null);
		const [bootError, setBootError] = React.useState('');

		const refreshProjects = React.useCallback(async (preferredProjectId) => {
			try {
				const data = await apiCall('/api/projects');
				const list = data || [];
				setProjects(list);
				setBootError('');
				setActiveProjectId(prev => {
					const target = preferredProjectId !== undefined ? preferredProjectId : prev;
					if (!list.length) return '';
					if (target === '') return '';
					if (target === undefined || target === null) return list[0].id;
					return list.some(project => project.id === target) ? target : list[0].id;
				});
				return list;
			} catch (error) {
				const message = error?.message || 'Failed to load projects';
				setBootError(message);
				throw error;
			}
		}, []);

		React.useEffect(() => {
			refreshProjects().catch(() => {});
		}, [refreshProjects]);

		React.useEffect(() => {
			const handleNotify = (event) => {
				const id = `${Date.now()}-${Math.random().toString(16).slice(2)}`;
				const notification = { id, duration: 4000, tone: 'info', ...event.detail };
				setNotifications(prev => [...prev, notification]);
				window.setTimeout(() => {
					setNotifications(prev => prev.filter(item => item.id !== id));
				}, notification.duration);
			};
			const handleConfirm = (event) => setConfirmState(event.detail || null);
			window.addEventListener('mimir:notify', handleNotify);
			window.addEventListener('mimir:confirm', handleConfirm);
			return () => {
				window.removeEventListener('mimir:notify', handleNotify);
				window.removeEventListener('mimir:confirm', handleConfirm);
			};
		}, []);

		const activeProject = projects.find(project => project.id === activeProjectId) || null;
		const setActiveProject = React.useCallback((project) => setActiveProjectId(project?.id || ''), []);
		const { summary, loading: summaryLoading, error: summaryError } = useProjectStateSummary(activeProject?.id || '');

		const pages = ['Projects', 'Pipelines', 'Ontologies', 'ML Models', 'Digital Twins', 'Storage', 'Insights & Review', 'Plugins', 'Work Queue'];
		const pageComponents = {
			Projects: ProjectsPage,
			Pipelines: PipelinesPage,
			Ontologies: OntologiesPage,
			'ML Models': MLModelsPage,
			'Digital Twins': DigitalTwinsPage,
			Storage: StoragePage,
			'Insights & Review': InsightsReviewPage,
			Plugins: PluginsPage,
			'Work Queue': WorkTasksPage,
		};

		const navigate = (page) => {
			setCurrentPage(page);
			setSidebarOpen(false);
		};

		const resolveConfirm = (confirmed) => {
			if (!confirmState?.id) return;
			window.dispatchEvent(new CustomEvent('mimir:confirm-result', { detail: { id: confirmState.id, confirmed } }));
			setConfirmState(null);
		};

		const PageComponent = pageComponents[currentPage] || ProjectsPage;
		const sectionState = summary?.sections || {};

		return (
			<div className="app-shell">
				<header className="app-topbar">
					<div className="topbar-brand">
						<button className="hamburger" onClick={() => setSidebarOpen(o => !o)} aria-label="Toggle navigation">
							<span/><span/><span/>
						</button>
						<span className="topbar-logo">◆</span>
						<span className="topbar-name">Mimir AIP</span>
					</div>
					<div className="topbar-meta">
						<span className="topbar-version">{activeProject ? `Project: ${activeProject.name}` : 'All projects'}</span>
					</div>
				</header>
				<div className="app-body">
					{sidebarOpen && <div className="sidebar-overlay" onClick={() => setSidebarOpen(false)} />}
					<aside className={`app-sidebar${sidebarOpen ? ' is-open' : ''}`}>
						<nav className="sidebar-nav" aria-label="Primary navigation">
							{pages.map(page => {
								const navState = sectionState[page] || {
									status: summaryError ? 'error' : 'inactive',
									detail: summaryError ? 'State unavailable' : activeProject ? (summaryLoading ? 'Refreshing status…' : 'Awaiting state snapshot') : 'Select a project',
								};
								return (
									<button
										key={page}
										className={`nav-item${currentPage === page ? ' active' : ''}`}
										onClick={() => navigate(page)}
										aria-current={currentPage === page ? 'page' : undefined}
										title={navState.detail || 'Idle'}
									>
										<span className={`nav-status-indicator status-${navState.status}${navState.pulse ? ' is-pulsing' : ''}`} aria-hidden="true" />
										<span className="nav-item-text">
											<span className="nav-item-label">{page}</span>
											<span className="nav-item-detail">{navState.detail || 'Idle'}</span>
										</span>
									</button>
								);
							})}
							<div className="sidebar-project-selector">
								<label htmlFor="active-project-select">Working Project</label>
								<select id="active-project-select" value={activeProjectId ?? ''} onChange={e => setActiveProjectId(e.target.value)}>
									<option value="">— All Projects —</option>
									{projects.map(project => <option key={project.id} value={project.id}>{project.name}</option>)}
								</select>
							</div>
						</nav>
					</aside>
					<main className="app-main">
						<div className="app-container">
							{bootError ? (
								<div className="panel-card" style={{ marginBottom: '1rem', border: '1px solid rgba(255,107,107,0.35)', background: 'rgba(96,18,18,0.45)' }}>
									<div style={{ display: 'flex', justifyContent: 'space-between', gap: '1rem', alignItems: 'center' }}>
										<div>
											<div style={{ fontWeight: 700, marginBottom: '0.35rem' }}>Frontend startup could not reach the API</div>
											<div style={{ opacity: 0.8, fontSize: '0.95rem' }}>{bootError}</div>
										</div>
										<Button label="Retry connection" onClick={() => refreshProjects().catch(() => {})} variant="secondary" />
									</div>
								</div>
							) : null}
							<ProjectContext.Provider value={{ activeProject, activeProjectId: activeProjectId || '', projects, setActiveProject, setActiveProjectId, refreshProjects }}>
								<PageComponent />
							</ProjectContext.Provider>
						</div>
					</main>
				</div>

				<div className="toast-stack" aria-live="polite" aria-atomic="true">
					{notifications.map((item) => (
						<div key={item.id} className={`toast toast-${item.tone || 'info'}`}>
							<div className="toast-message">{item.message}</div>
						</div>
					))}
				</div>

				<Modal
					open={Boolean(confirmState)}
					onClose={() => resolveConfirm(false)}
					title={confirmState?.title}
					footer={confirmState ? (
						<>
							<Button label={confirmState.cancelLabel || 'Cancel'} onClick={() => resolveConfirm(false)} variant="secondary" />
							<Button label={confirmState.confirmLabel || 'Confirm'} onClick={() => resolveConfirm(true)} variant={confirmState.variant || 'danger'} />
						</>
					) : null}
					hideDefaultFooter
				>
					<p className="modal-copy">{confirmState?.message}</p>
				</Modal>
			</div>
		);
	}

	const mountNode = document.getElementById('root');
	if (!mountNode) {
		throw new Error('Frontend root element #root is missing');
	}
	const rootNode = ReactDOM.createRoot(mountNode);
	rootNode.render(<App />);

})();
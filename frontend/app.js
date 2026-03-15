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

		const refreshProjects = React.useCallback(async (preferredProjectId) => {
			const data = await apiCall('/api/projects');
			const list = data || [];
			setProjects(list);
			setActiveProjectId(prev => {
				const target = preferredProjectId !== undefined ? preferredProjectId : prev;
				if (!list.length) return '';
				if (target === '') return '';
				if (target === undefined || target === null) return list[0].id;
				return list.some(project => project.id === target) ? target : list[0].id;
			});
			return list;
		}, []);

		React.useEffect(() => {
			refreshProjects().catch(() => {});
		}, [refreshProjects]);

		const activeProject = projects.find(project => project.id === activeProjectId) || null;
		const setActiveProject = React.useCallback((project) => setActiveProjectId(project?.id || ''), []);
		const { summary } = useProjectStateSummary(activeProject?.id || '');

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
						<nav className="sidebar-nav">
							{pages.map(page => {
								const navState = sectionState[page] || { status: 'inactive', detail: activeProject ? 'Awaiting state snapshot' : 'Select a project' };
								return (
									<button key={page} className={`nav-item${currentPage === page ? ' active' : ''}`} onClick={() => navigate(page)}>
										<span className={`nav-status-indicator status-${navState.status}${navState.pulse ? ' is-pulsing' : ''}`} aria-hidden="true" />
										<span className="nav-item-text">
											<span className="nav-item-label">{page}</span>
											<span className="nav-item-detail">{navState.detail || 'Idle'}</span>
										</span>
									</button>
								);
							})}
							<div className="sidebar-project-selector">
								<label>Working Project</label>
								<select value={activeProjectId ?? ''} onChange={e => setActiveProjectId(e.target.value)}>
									<option value="">— All Projects —</option>
									{projects.map(project => <option key={project.id} value={project.id}>{project.name}</option>)}
								</select>
							</div>
						</nav>
					</aside>
					<main className="app-main">
						<div className="app-container">
							<ProjectContext.Provider value={{ activeProject, activeProjectId: activeProjectId || '', projects, setActiveProject, setActiveProjectId, refreshProjects }}>
								<PageComponent />
							</ProjectContext.Provider>
						</div>
					</main>
				</div>
			</div>
		);
	}

	const rootNode = ReactDOM.createRoot(document.getElementById('root'));
	rootNode.render(<App />);
})();

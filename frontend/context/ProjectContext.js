(() => {
	const root = window.MimirApp = window.MimirApp || {};
	const context = root.context = root.context || {};

	context.ProjectContext = React.createContext({
		activeProject: null,
		activeProjectId: '',
		projects: [],
		setActiveProject: () => {},
		setActiveProjectId: () => {},
		refreshProjects: async () => [],
	});
})();

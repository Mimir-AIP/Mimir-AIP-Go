(() => {
	const root = window.MimirApp = window.MimirApp || {};
	const hooks = root.hooks = root.hooks || {};
	const { ProjectContext } = root.context;

	hooks.useProjectContext = function useProjectContext() {
		return React.useContext(ProjectContext);
	};
})();

(() => {
	const root = window.MimirApp = window.MimirApp || {};
	const hooks = root.hooks = root.hooks || {};
	const { apiCall } = root.lib;
	const { useTaskWebSocket } = root.hooks;

	hooks.useProjectStateSummary = function useProjectStateSummary(projectId) {
		const [summary, setSummary] = React.useState(null);
		const [loading, setLoading] = React.useState(false);

		const loadSummary = React.useCallback(async () => {
			if (!projectId) {
				setSummary(null);
				return;
			}
			setLoading(true);
			try {
				const data = await apiCall(`/api/projects/${projectId}/state-summary`);
				setSummary(data || null);
			} catch (error) {
				console.error('Failed to load project state summary:', error);
			} finally {
				setLoading(false);
			}
		}, [projectId]);

		React.useEffect(() => {
			loadSummary();
			if (!projectId) return;
			const intervalId = window.setInterval(loadSummary, 15000);
			return () => window.clearInterval(intervalId);
		}, [projectId, loadSummary]);

		useTaskWebSocket(() => {
			if (!projectId) return;
			loadSummary();
		});

		return { summary, loading, refresh: loadSummary };
	};
})();

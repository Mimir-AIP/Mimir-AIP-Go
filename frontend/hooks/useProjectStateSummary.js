(() => {
	const root = window.MimirApp = window.MimirApp || {};
	const hooks = root.hooks = root.hooks || {};
	const { apiCall } = root.lib;
	const { useTaskWebSocket } = root.hooks;

	hooks.useProjectStateSummary = function useProjectStateSummary(projectId) {
		const [summary, setSummary] = React.useState(null);
		const [loading, setLoading] = React.useState(false);
		const [error, setError] = React.useState('');
		const requestRef = React.useRef(0);

		const loadSummary = React.useCallback(async () => {
			requestRef.current += 1;
			const requestId = requestRef.current;
			if (!projectId) {
				setSummary(null);
				setError('');
				setLoading(false);
				return;
			}
			setLoading(true);
			setError('');
			try {
				const data = await apiCall(`/api/projects/${projectId}/state-summary`);
				if (requestRef.current !== requestId) return;
				setSummary(data || null);
			} catch (fetchError) {
				if (requestRef.current !== requestId) return;
				setSummary(null);
				setError(fetchError.message || 'Failed to load project state summary.');
			} finally {
				if (requestRef.current === requestId) {
					setLoading(false);
				}
			}
		}, [projectId]);

		React.useEffect(() => {
			loadSummary();
			if (!projectId) return undefined;
			const intervalId = window.setInterval(loadSummary, 15000);
			return () => window.clearInterval(intervalId);
		}, [projectId, loadSummary]);

		useTaskWebSocket(React.useCallback(() => {
			if (!projectId) return;
			loadSummary();
		}, [projectId, loadSummary]));

		return { summary, loading, error, refresh: loadSummary };
	};
})();

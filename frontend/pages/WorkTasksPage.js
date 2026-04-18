(() => {
	const root = window.MimirApp = window.MimirApp || {};
	const pages = root.pages = root.pages || {};
	const { apiCall } = root.lib;
	const { useTaskWebSocket } = root.hooks;
	const { Button, Table } = root.components.primitives;


	pages.WorkTasksPage = function WorkTasksPage() {
		const [tasks, setTasks] = React.useState([]);
		const [queueLength, setQueueLength] = React.useState(0);
		const [loading, setLoading] = React.useState(true);
		const [loadError, setLoadError] = React.useState('');

		const loadTasks = React.useCallback(async () => {
			setLoading(true);
			setLoadError('');
			try {
				const data = await apiCall('/api/worktasks');
				setTasks(data.tasks || []);
				setQueueLength(data.queue_length || 0);
			} catch (error) {
				setLoadError(error.message || 'Failed to load work tasks.');
				setTasks([]);
				setQueueLength(0);
			} finally {
				setLoading(false);
			}
		}, []);

		useTaskWebSocket(React.useCallback((updatedTask) => {
			setTasks((prev) => {
				const idx = prev.findIndex((task) => task.id === updatedTask.id);
				const previousTask = idx >= 0 ? prev[idx] : null;
				setQueueLength((current) => {
					const wasQueued = previousTask ? ['queued', 'scheduled'].includes(previousTask.status) : false;
					const isQueued = ['queued', 'scheduled'].includes(updatedTask.status);
					if (!previousTask) return isQueued ? current + 1 : current;
					if (wasQueued === isQueued) return current;
					return isQueued ? current + 1 : Math.max(0, current - 1);
				});
				if (idx >= 0) {
					const next = [...prev];
					next[idx] = updatedTask;
					return next;
				}
				return [updatedTask, ...prev];
			});
		}, []));

		React.useEffect(() => {
			loadTasks();
		}, [loadTasks]);

		const columns = [
			{ key: 'id', label: 'Task ID' },
			{ key: 'type', label: 'Type' },
			{ key: 'priority', label: 'Priority' },
			{ key: 'status', label: 'Status', render: (row) => <span className={`status-badge status-${row.status}`}>{row.status}</span> },
			{ key: 'project_id', label: 'Project ID' },
			{ key: 'submitted_at', label: 'Submitted', render: (row) => new Date(row.submitted_at || row.created_at).toLocaleString() },
			{ key: 'completed_at', label: 'Completed', render: (row) => row.completed_at ? new Date(row.completed_at).toLocaleString() : '—' },
		];

		return (
			<div className="content-section">
				<div className="section-header">
					<h2>Work Queue</h2>
					<div className="inline-actions">
						<span className="status-badge status-pending">{queueLength} queued</span>
						<Button label="Refresh" onClick={loadTasks} variant="secondary" />
					</div>
				</div>

				{loadError ? (
					<div className="error-message">{loadError}</div>
				) : null}

				{loading ? (
					<div className="loading">Loading work tasks…</div>
				) : (
					<Table caption="Work queue tasks" columns={columns} data={tasks} emptyState="No work tasks have been submitted yet." />
				)}
			</div>
		);
	};
})();

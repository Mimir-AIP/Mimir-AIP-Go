(() => {
	const root = window.MimirApp = window.MimirApp || {};
	const pages = root.pages = root.pages || {};
	const { apiCall } = root.lib;
	const { useTaskWebSocket } = root.hooks;
	const { Button, Table } = root.components.primitives;

	pages.WorkTasksPage = function WorkTasksPage() {
		const [tasks, setTasks] = React.useState([]);
		const [loading, setLoading] = React.useState(true);
		const [queueLength, setQueueLength] = React.useState(0);

		const loadTasks = async () => {
			setLoading(true);
			try {
				const data = await apiCall('/api/worktasks');
				setTasks(data.tasks || []);
				setQueueLength(data.queue_length || 0);
			} catch (error) {
				console.error('Failed to load work tasks:', error);
				setTasks([]);
			}
			setLoading(false);
		};

		useTaskWebSocket((updatedTask) => {
			setTasks((prev) => {
				const idx = prev.findIndex((t) => t.worktask_id === updatedTask.worktask_id);
				if (idx >= 0) {
					const next = [...prev];
					next[idx] = updatedTask;
					return next;
				}
				return [updatedTask, ...prev];
			});
		});

		React.useEffect(() => {
			loadTasks();
		}, []);

		const columns = [
			{ key: 'id', label: 'Task ID' },
			{ key: 'type', label: 'Type' },
			{ key: 'priority', label: 'Priority' },
			{
				key: 'status',
				label: 'Status',
				render: (row) => <span className={`status-badge status-${row.status}`}>{row.status}</span>
			},
			{ key: 'project_id', label: 'Project ID' },
			{ key: 'created_at', label: 'Created', render: (row) => new Date(row.created_at).toLocaleString() },
			{ key: 'updated_at', label: 'Updated', render: (row) => new Date(row.updated_at).toLocaleString() },
		];

		return (
			<div className="content-section">
				<div className="section-header">
					<h2>Work Queue</h2>
					<div style={{ display: 'flex', alignItems: 'center', gap: '16px' }}>
						<span style={{ color: 'var(--accent)', fontWeight: 'bold' }}>
							Queue Length: {queueLength}
						</span>
						<Button label="Refresh" onClick={loadTasks} variant="secondary" />
					</div>
				</div>

				{loading ? (
					<div className="loading">Loading work tasks...</div>
				) : (
					<Table columns={columns} data={tasks} />
				)}
			</div>
		);
	};
})();

(() => {
	const root = window.MimirApp = window.MimirApp || {};
	const pages = root.pages = root.pages || {};
	const { apiCall } = root.lib;
	const { ProjectContext } = root.context;
	const { useTaskWebSocket } = root.hooks;
	const { Button, FormField, Modal, Table, Tabs } = root.components.primitives;
	const { CronBuilder, StepBuilder } = root.components.pipelines;

	pages.PipelinesPage = function PipelinesPage() {
		const { activeProject, projects } = React.useContext(ProjectContext);
		const [pipelines, setPipelines] = React.useState([]);
		const [schedules, setSchedules] = React.useState([]);
		const [loading, setLoading] = React.useState(true);
		const [showPipelineModal, setShowPipelineModal] = React.useState(false);
		const [showScheduleModal, setShowScheduleModal] = React.useState(false);
		const [selectedTab, setSelectedTab] = React.useState('Pipelines');
		const [executionStatus, setExecutionStatus] = React.useState({});
		const [pipelineForm, setPipelineForm] = React.useState({
			name: '',
			description: '',
			project_id: '',
			type: 'ingestion',
			steps: '[]',
		});
		const [scheduleForm, setScheduleForm] = React.useState({
			name: '',
			pipelines: [],
			project_id: '',
			cron_schedule: '',
			enabled: true,
		});

		React.useEffect(() => {
			if (activeProject?.id) {
				setPipelineForm(prev => ({ ...prev, project_id: activeProject.id }));
				setScheduleForm(prev => ({ ...prev, project_id: activeProject.id }));
			}
		}, [activeProject?.id]);

		useTaskWebSocket((task) => {
			if (task.type !== 'pipeline_execution') return;
			const pid = task.task_spec && task.task_spec.pipeline_id;
			if (!pid) return;
			if (task.status === 'completed') setExecutionStatus(prev => ({ ...prev, [pid]: 'done' }));
			if (task.status === 'failed') setExecutionStatus(prev => ({ ...prev, [pid]: 'error' }));
		});

		const loadData = async () => {
			setLoading(true);
			try {
				const [pipelinesData, schedulesData] = await Promise.all([
					apiCall('/api/pipelines'),
					apiCall('/api/schedules'),
				]);
				setPipelines(pipelinesData || []);
				setSchedules(schedulesData || []);
			} catch (error) {
				console.error('Failed to load data:', error);
				setPipelines([]);
				setSchedules([]);
			}
			setLoading(false);
		};

		React.useEffect(() => {
			loadData();
		}, []);

		const filteredPipelines = activeProject?.id ? pipelines.filter(pipeline => pipeline.project_id === activeProject.id) : pipelines;
		const filteredSchedules = activeProject?.id ? schedules.filter(schedule => schedule.project_id === activeProject.id) : schedules;

		const handlePipelineSubmit = async (e) => {
			e.preventDefault();
			try {
				await apiCall('/api/pipelines', {
					method: 'POST',
					body: JSON.stringify({
						...pipelineForm,
						steps: JSON.parse(pipelineForm.steps),
					}),
				});
				setShowPipelineModal(false);
				setPipelineForm({ name: '', description: '', project_id: activeProject?.id || '', type: 'ingestion', steps: '[]' });
				loadData();
			} catch (error) {
				alert('Failed to create pipeline: ' + error.message);
			}
		};

		const handleScheduleSubmit = async (e) => {
			e.preventDefault();
			try {
				await apiCall('/api/schedules', {
					method: 'POST',
					body: JSON.stringify({
						...scheduleForm,
						pipelines: scheduleForm.pipelines.filter(Boolean),
					}),
				});
				setShowScheduleModal(false);
				setScheduleForm({ name: '', pipelines: [], project_id: activeProject?.id || '', cron_schedule: '', enabled: true });
				loadData();
			} catch (error) {
				alert('Failed to create schedule: ' + error.message);
			}
		};

		const handleDeletePipeline = async (id) => {
			if (!confirm('Delete this pipeline?')) return;
			try {
				await apiCall(`/api/pipelines/${id}`, { method: 'DELETE' });
				loadData();
			} catch (error) {
				alert('Failed to delete pipeline: ' + error.message);
			}
		};

		const handleDeleteSchedule = async (id) => {
			if (!confirm('Delete this schedule?')) return;
			try {
				await apiCall(`/api/schedules/${id}`, { method: 'DELETE' });
				loadData();
			} catch (error) {
				alert('Failed to delete schedule: ' + error.message);
			}
		};

		const handleExecutePipeline = async (id) => {
			setExecutionStatus(prev => ({ ...prev, [id]: 'running' }));
			try {
				await apiCall(`/api/pipelines/${id}/execute`, { method: 'POST', body: JSON.stringify({}) });
				setExecutionStatus(prev => ({ ...prev, [id]: 'done' }));
				setTimeout(() => setExecutionStatus(prev => {
					const next = { ...prev };
					delete next[id];
					return next;
				}), 5000);
			} catch {
				setExecutionStatus(prev => ({ ...prev, [id]: 'error' }));
			}
		};

		const projectOptions = projects.map(p => ({ value: p.id, label: p.name }));
		const schedulePipelineOptions = pipelines
			.filter(pipeline => !scheduleForm.project_id || pipeline.project_id === scheduleForm.project_id)
			.map(pipeline => ({ value: pipeline.id, label: pipeline.name }));

		const pipelineColumns = [
			{ key: 'id', label: 'ID' },
			{ key: 'name', label: 'Name' },
			{ key: 'description', label: 'Description' },
			{ key: 'project_id', label: 'Project ID' },
			{ key: 'created_at', label: 'Created', render: (row) => new Date(row.created_at).toLocaleDateString() },
		];

		const scheduleColumns = [
			{ key: 'id', label: 'ID' },
			{ key: 'name', label: 'Name' },
			{ key: 'pipelines', label: 'Pipelines', render: (row) => Array.isArray(row.pipelines) && row.pipelines.length > 0 ? row.pipelines.join(', ') : '-' },
			{ key: 'cron_schedule', label: 'Cron Schedule' },
			{
				key: 'enabled',
				label: 'Status',
				render: (row) => <span className={`status-badge ${row.enabled ? 'status-active' : 'status-inactive'}`}>{row.enabled ? 'Enabled' : 'Disabled'}</span>
			},
			{ key: 'last_run', label: 'Last Run', render: (row) => row.last_run ? new Date(row.last_run).toLocaleString() : 'Never' },
		];

		return (
			<div className="content-section">
				<div className="section-header">
					<h2>Pipelines & Schedules</h2>
					<div style={{ display: 'flex', gap: '8px' }}>
						<Button label="+ New Pipeline" onClick={() => setShowPipelineModal(true)} />
						<Button label="+ New Schedule" onClick={() => setShowScheduleModal(true)} variant="secondary" />
					</div>
				</div>
				{activeProject && (
					<div style={{ marginBottom: '12px', color: 'var(--text-secondary)' }}>Filtering to project: <strong style={{ color: 'var(--accent)' }}>{activeProject.name}</strong></div>
				)}

				<Tabs tabs={['Pipelines', 'Recurring Jobs']} activeTab={selectedTab} onTabChange={setSelectedTab} />

				{loading ? (
					<div className="loading">Loading...</div>
				) : selectedTab === 'Pipelines' ? (
					<Table
						columns={pipelineColumns}
						data={filteredPipelines}
						actions={(row) => (
							<>
								{executionStatus[row.id] === 'running'
									? <span className="status-badge status-pending">Running…</span>
									: executionStatus[row.id] === 'done'
									? <span className="status-badge status-active">Done ✓</span>
									: executionStatus[row.id] === 'error'
									? <span className="status-badge status-failed">Error</span>
									: <Button label="Execute" onClick={() => handleExecutePipeline(row.id)} variant="secondary" />
								}
								<Button label="Delete" onClick={() => handleDeletePipeline(row.id)} variant="danger" />
							</>
						)}
					/>
				) : (
					<Table
						columns={scheduleColumns}
						data={filteredSchedules}
						actions={(row) => <Button label="Delete" onClick={() => handleDeleteSchedule(row.id)} variant="danger" />}
					/>
				)}

				<Modal open={showPipelineModal} onClose={() => setShowPipelineModal(false)} title="Create New Pipeline">
					<form onSubmit={handlePipelineSubmit}>
						<div className="form-grid">
							<FormField label="Pipeline Name" value={pipelineForm.name} onChange={(v) => setPipelineForm({ ...pipelineForm, name: v })} required />
							<FormField label="Project" type="select" value={pipelineForm.project_id} onChange={(v) => setPipelineForm({ ...pipelineForm, project_id: v })} options={projectOptions} required />
						</div>
						<FormField label="Pipeline Type" type="select" value={pipelineForm.type} onChange={(v) => setPipelineForm({ ...pipelineForm, type: v })} options={['ingestion', 'processing', 'output']} required />
						<FormField label="Description" type="textarea" value={pipelineForm.description} onChange={(v) => setPipelineForm({ ...pipelineForm, description: v })} />
						<div className="form-group">
							<label>Steps</label>
							<StepBuilder value={pipelineForm.steps} onChange={v => setPipelineForm({ ...pipelineForm, steps: v })} />
						</div>
						<Button type="submit" label="Create Pipeline" />
					</form>
				</Modal>

				<Modal open={showScheduleModal} onClose={() => setShowScheduleModal(false)} title="Create Recurring Job">
					<form onSubmit={handleScheduleSubmit}>
						<div className="form-grid">
							<FormField label="Schedule Name" value={scheduleForm.name} onChange={(v) => setScheduleForm({ ...scheduleForm, name: v })} required />
							<FormField label="Project" type="select" value={scheduleForm.project_id} onChange={(v) => setScheduleForm({ ...scheduleForm, project_id: v, pipelines: [] })} options={projectOptions} required />
						</div>
						<div className="form-grid">
							<FormField
								label="Pipeline"
								type="select"
								value={scheduleForm.pipelines[0] || ''}
								onChange={(v) => setScheduleForm({ ...scheduleForm, pipelines: v ? [v] : [] })}
								options={schedulePipelineOptions}
								required
							/>
							<div className="form-group">
								<label>Schedule *</label>
								<CronBuilder value={scheduleForm.cron_schedule} onChange={v => setScheduleForm({ ...scheduleForm, cron_schedule: v })} />
							</div>
						</div>
						<label style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '16px' }}>
							<input type="checkbox" checked={scheduleForm.enabled} onChange={e => setScheduleForm({ ...scheduleForm, enabled: e.target.checked })} />
							Enabled
						</label>
						<Button type="submit" label="Create Schedule" />
					</form>
				</Modal>
			</div>
		);
	};
})();

(() => {
	const root = window.MimirApp = window.MimirApp || {};
	const pages = root.pages = root.pages || {};
	const { apiCall, notify, confirmAction } = root.lib;
	const { ProjectContext } = root.context;
	const { useTaskWebSocket } = root.hooks;
	const { Button, FormField, Modal, Table, Tabs } = root.components.primitives;
	const { CronBuilder, StepBuilder } = root.components.pipelines;

	function summarizeExecutionState(status) {
		switch (status) {
		case 'queued':
		case 'scheduled':
		case 'spawned':
			return { label: 'Queued', className: 'status-pending' };
		case 'executing':
			return { label: 'Running', className: 'status-running' };
		case 'completed':
			return { label: 'Completed', className: 'status-completed' };
		case 'failed':
		case 'timeout':
		case 'cancelled':
			return { label: 'Failed', className: 'status-failed' };
		default:
			return null;
		}
	}

	pages.PipelinesPage = function PipelinesPage() {
		const { activeProject, projects } = React.useContext(ProjectContext);
		const [pipelines, setPipelines] = React.useState([]);
		const [schedules, setSchedules] = React.useState([]);
		const [loading, setLoading] = React.useState(true);
		const [loadError, setLoadError] = React.useState('');
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

		const loadData = React.useCallback(async () => {
			setLoading(true);
			setLoadError('');
			try {
				const projectQuery = activeProject?.id ? `?project_id=${activeProject.id}` : '';
				const [pipelinesData, schedulesData] = await Promise.all([
					apiCall(`/api/pipelines${projectQuery}`),
					apiCall(`/api/schedules${projectQuery}`),
				]);
				setPipelines(pipelinesData || []);
				setSchedules(schedulesData || []);
			} catch (error) {
				setLoadError(error.message || 'Failed to load pipelines and schedules.');
				setPipelines([]);
				setSchedules([]);
			} finally {
				setLoading(false);
			}
		}, [activeProject?.id]);

		React.useEffect(() => {
			loadData();
		}, [loadData]);

		useTaskWebSocket(React.useCallback((task) => {
			if (task.type !== 'pipeline_execution') return;
			const pipelineId = task.task_spec?.pipeline_id;
			if (!pipelineId) return;
			if (activeProject?.id && task.project_id !== activeProject.id) return;
			setExecutionStatus(prev => ({ ...prev, [pipelineId]: task.status }));
			if (['completed', 'failed', 'timeout', 'cancelled'].includes(task.status)) {
				loadData();
			}
		}, [activeProject?.id, loadData]));

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
				notify({ tone: 'success', message: 'Pipeline created.' });
				loadData();
			} catch (error) {
				notify({ tone: 'error', message: `Failed to create pipeline: ${error.message}` });
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
				notify({ tone: 'success', message: 'Recurring job created.' });
				loadData();
			} catch (error) {
				notify({ tone: 'error', message: `Failed to create schedule: ${error.message}` });
			}
		};

		const handleDeletePipeline = async (id) => {
			const confirmed = await confirmAction({
				title: 'Delete pipeline',
				message: 'Delete this pipeline? Mimir will block deletion while schedules, automations, twin actions, or active work tasks still reference it.',
				confirmLabel: 'Delete pipeline',
				variant: 'danger',
			});
			if (!confirmed) return;
			try {
				await apiCall(`/api/pipelines/${id}`, { method: 'DELETE' });
				notify({ tone: 'success', message: 'Pipeline deleted.' });
				loadData();
			} catch (error) {
				notify({ tone: 'error', message: `Failed to delete pipeline: ${error.message}` });
			}
		};

		const handleDeleteSchedule = async (id) => {
			const confirmed = await confirmAction({
				title: 'Delete recurring job',
				message: 'Delete this recurring job? Future scheduled runs will stop.',
				confirmLabel: 'Delete job',
				variant: 'danger',
			});
			if (!confirmed) return;
			try {
				await apiCall(`/api/schedules/${id}`, { method: 'DELETE' });
				notify({ tone: 'success', message: 'Recurring job deleted.' });
				loadData();
			} catch (error) {
				notify({ tone: 'error', message: `Failed to delete schedule: ${error.message}` });
			}
		};

		const handleExecutePipeline = async (id) => {
			setExecutionStatus(prev => ({ ...prev, [id]: 'queued' }));
			try {
				const result = await apiCall(`/api/pipelines/${id}/execute`, { method: 'POST', body: JSON.stringify({}) });
				notify({ tone: 'success', message: result?.message || 'Pipeline run queued.' });
			} catch (error) {
				setExecutionStatus(prev => ({ ...prev, [id]: 'failed' }));
				notify({ tone: 'error', message: `Failed to queue pipeline run: ${error.message}` });
			}
		};

		const projectOptions = projects.map(p => ({ value: p.id, label: p.name }));
		const schedulePipelineOptions = pipelines
			.filter(pipeline => !scheduleForm.project_id || pipeline.project_id === scheduleForm.project_id)
			.map(pipeline => ({ value: pipeline.id, label: pipeline.name }));

		const renderPipelineActions = (row) => {
			const execution = summarizeExecutionState(executionStatus[row.id]);
			return (
				<>
					{execution ? <span className={`status-badge ${execution.className}`}>{execution.label}</span> : <Button label="Queue Run" onClick={() => handleExecutePipeline(row.id)} variant="secondary" />}
					<Button label="Delete" onClick={() => handleDeletePipeline(row.id)} variant="danger" />
				</>
			);
		};

		const renderScheduleActions = (row) => <Button label="Delete" onClick={() => handleDeleteSchedule(row.id)} variant="danger" />;


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
			{ key: 'pipelines', label: 'Pipelines', render: (row) => Array.isArray(row.pipelines) && row.pipelines.length > 0 ? row.pipelines.join(', ') : '—' },
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
					<div className="inline-actions">
						<Button label="+ New Pipeline" onClick={() => setShowPipelineModal(true)} />
						<Button label="+ New Schedule" onClick={() => setShowScheduleModal(true)} variant="secondary" />
					</div>
				</div>

				{activeProject ? (
					<div className="page-notice"><strong>Project scope:</strong> showing pipelines and schedules for {activeProject.name}. Execution is asynchronous and queues work tasks for workers.</div>
				) : null}
				{loadError ? <div className="error-message">{loadError}</div> : null}

				<Tabs tabs={['Pipelines', 'Recurring Jobs']} activeTab={selectedTab} onTabChange={setSelectedTab} />

				{loading ? (
					<div className="loading">Loading pipelines and schedules…</div>
				) : selectedTab === 'Pipelines' ? (
					<Table
						caption="Project pipelines"
						columns={pipelineColumns}
						data={pipelines}
						emptyState={activeProject ? 'No pipelines exist for this project yet.' : 'Select a project or create a pipeline to get started.'}
						actions={renderPipelineActions}
					/>
				) : (
					<Table
						caption="Recurring jobs"
						columns={scheduleColumns}
						data={schedules}
						emptyState={activeProject ? 'No recurring jobs exist for this project yet.' : 'Select a project or create a recurring job to see it here.'}
						actions={renderScheduleActions}
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
						<label className="checkbox-row">
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

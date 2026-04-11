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

	function buildTriggerSummary(triggerConfig) {
		if (!triggerConfig) return 'Manual enabled';
		const flags = [];
		flags.push(triggerConfig.allow_manual === false ? 'Manual disabled' : 'Manual enabled');
		if (triggerConfig.webhook) flags.push('Webhook enabled');
		return flags.join(' · ');
	}

	function emptyPipelineForm(projectId = '') {
		return {
			name: '',
			description: '',
			project_id: projectId,
			type: 'ingestion',
			steps: '[]',
			allow_manual: true,
			webhook: false,
			webhook_secret: '',
		};
	}

	pages.PipelinesPage = function PipelinesPage() {
		const { activeProject, projects } = React.useContext(ProjectContext);
		const [pipelines, setPipelines] = React.useState([]);
		const [schedules, setSchedules] = React.useState([]);
		const [loading, setLoading] = React.useState(true);
		const [loadError, setLoadError] = React.useState('');
		const [showPipelineModal, setShowPipelineModal] = React.useState(false);
		const [editingPipelineId, setEditingPipelineId] = React.useState('');
		const [showScheduleModal, setShowScheduleModal] = React.useState(false);
		const [showCheckpointModal, setShowCheckpointModal] = React.useState(false);
		const [selectedTab, setSelectedTab] = React.useState('Pipelines');
		const [executionStatus, setExecutionStatus] = React.useState({});
		const [pipelineForm, setPipelineForm] = React.useState(emptyPipelineForm());
		const [scheduleForm, setScheduleForm] = React.useState({
			name: '',
			pipelines: [],
			project_id: '',
			cron_schedule: '',
			enabled: true,
		});
		const [checkpointForm, setCheckpointForm] = React.useState({ pipeline_id: '', step_name: '', scope: '' });
		const [checkpointData, setCheckpointData] = React.useState(null);
		const [checkpointLoading, setCheckpointLoading] = React.useState(false);
		const [checkpointError, setCheckpointError] = React.useState('');

		React.useEffect(() => {
			if (activeProject?.id) {
				setPipelineForm(prev => ({ ...prev, project_id: prev.project_id || activeProject.id }));
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

		const openCreatePipelineModal = () => {
			setEditingPipelineId('');
			setPipelineForm(emptyPipelineForm(activeProject?.id || ''));
			setShowPipelineModal(true);
		};

		const openEditPipelineModal = (pipeline) => {
			setEditingPipelineId(pipeline.id);
			setPipelineForm({
				name: pipeline.name || '',
				description: pipeline.description || '',
				project_id: pipeline.project_id || activeProject?.id || '',
				type: pipeline.type || 'ingestion',
				steps: JSON.stringify(pipeline.steps || [], null, 2),
				allow_manual: pipeline.trigger_config?.allow_manual !== false,
				webhook: Boolean(pipeline.trigger_config?.webhook),
				webhook_secret: '',
			});
			setShowPipelineModal(true);
		};

		const handlePipelineSubmit = async (e) => {
			e.preventDefault();
			const body = {
				project_id: pipelineForm.project_id,
				name: pipelineForm.name,
				description: pipelineForm.description,
				type: pipelineForm.type,
				steps: JSON.parse(pipelineForm.steps),
				trigger_config: {
					allow_manual: pipelineForm.allow_manual,
					webhook: pipelineForm.webhook,
					secret: pipelineForm.webhook ? pipelineForm.webhook_secret : '',
				},
			};
			try {
				if (editingPipelineId) {
					await apiCall(`/api/pipelines/${editingPipelineId}`, {
						method: 'PUT',
						body: JSON.stringify({
							description: body.description,
							steps: body.steps,
							trigger_config: body.trigger_config,
						}),
					});
					notify({ tone: 'success', message: 'Pipeline updated.' });
				} else {
					await apiCall('/api/pipelines', { method: 'POST', body: JSON.stringify(body) });
					notify({ tone: 'success', message: 'Pipeline created.' });
				}
				setShowPipelineModal(false);
				setEditingPipelineId('');
				setPipelineForm(emptyPipelineForm(activeProject?.id || ''));
				loadData();
			} catch (error) {
				notify({ tone: 'error', message: `Failed to save pipeline: ${error.message}` });
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
				const result = await apiCall(`/api/pipelines/${id}/trigger`, { method: 'POST', body: JSON.stringify({ trigger_type: 'manual' }) });
				notify({ tone: 'success', message: result?.message || 'Pipeline run queued.' });
			} catch (error) {
				setExecutionStatus(prev => ({ ...prev, [id]: 'failed' }));
				notify({ tone: 'error', message: `Failed to queue pipeline run: ${error.message}` });
			}
		};

		const openCheckpointModal = (pipeline) => {
			setCheckpointForm({
				pipeline_id: pipeline.id,
				step_name: pipeline.steps?.[0]?.name || '',
				scope: '',
			});
			setCheckpointData(null);
			setCheckpointError('');
			setShowCheckpointModal(true);
		};

		const handleCheckpointLoad = async (e) => {
			e.preventDefault();
			if (!checkpointForm.pipeline_id || !checkpointForm.step_name) return;
			setCheckpointLoading(true);
			setCheckpointError('');
			setCheckpointData(null);
			try {
				const scopeQuery = checkpointForm.scope ? `&scope=${encodeURIComponent(checkpointForm.scope)}` : '';
				const result = await apiCall(`/api/pipelines/${checkpointForm.pipeline_id}/checkpoints?step_name=${encodeURIComponent(checkpointForm.step_name)}${scopeQuery}`);
				setCheckpointData(result);
			} catch (error) {
				setCheckpointError(error.message || 'Failed to load checkpoint.');
			}
			finally {
				setCheckpointLoading(false);
			}
		};

		const handleCheckpointReset = async () => {
			if (!checkpointData) return;
			try {
				const scopeQuery = checkpointForm.scope ? `&scope=${encodeURIComponent(checkpointForm.scope)}` : '';
				const result = await apiCall(`/api/pipelines/${checkpointForm.pipeline_id}/checkpoints?step_name=${encodeURIComponent(checkpointForm.step_name)}${scopeQuery}`, {
					method: 'PUT',
					body: JSON.stringify({ version: checkpointData.version, checkpoint: {} }),
				});
				setCheckpointData(result);
				notify({ tone: 'success', message: 'Checkpoint reset.' });
			} catch (error) {
				notify({ tone: 'error', message: `Failed to reset checkpoint: ${error.message}` });
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
					<Button label="Edit" onClick={() => openEditPipelineModal(row)} variant="secondary" />
					<Button label="Checkpoint" onClick={() => openCheckpointModal(row)} variant="secondary" />
					<Button label="Delete" onClick={() => handleDeletePipeline(row.id)} variant="danger" />
				</>
			);
		};

		const renderScheduleActions = (row) => <Button label="Delete" onClick={() => handleDeleteSchedule(row.id)} variant="danger" />;

		const pipelineColumns = [
			{ key: 'id', label: 'ID' },
			{ key: 'name', label: 'Name' },
			{ key: 'description', label: 'Description' },
			{ key: 'trigger_config', label: 'Triggers', render: (row) => buildTriggerSummary(row.trigger_config) },
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
						<Button label="+ New Pipeline" onClick={openCreatePipelineModal} />
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

				<Modal open={showPipelineModal} onClose={() => setShowPipelineModal(false)} title={editingPipelineId ? 'Edit Pipeline' : 'Create New Pipeline'}>
					<form onSubmit={handlePipelineSubmit}>
						<div className="form-grid">
							<FormField label="Pipeline Name" value={pipelineForm.name} onChange={(v) => setPipelineForm({ ...pipelineForm, name: v })} required disabled={Boolean(editingPipelineId)} />
							<FormField label="Project" type="select" value={pipelineForm.project_id} onChange={(v) => setPipelineForm({ ...pipelineForm, project_id: v })} options={projectOptions} required disabled={Boolean(editingPipelineId)} />
						</div>
						<FormField label="Pipeline Type" type="select" value={pipelineForm.type} onChange={(v) => setPipelineForm({ ...pipelineForm, type: v })} options={['ingestion', 'processing', 'output']} required disabled={Boolean(editingPipelineId)} />
						<FormField label="Description" type="textarea" value={pipelineForm.description} onChange={(v) => setPipelineForm({ ...pipelineForm, description: v })} />
						<div className="section-panel section-panel--neutral">
							<div className="section-panel-copy"><strong>Trigger Configuration</strong></div>
							<label className="checkbox-row">
								<input type="checkbox" checked={pipelineForm.allow_manual} onChange={e => setPipelineForm({ ...pipelineForm, allow_manual: e.target.checked })} />
								Allow manual trigger requests
							</label>
							<label className="checkbox-row">
								<input type="checkbox" checked={pipelineForm.webhook} onChange={e => setPipelineForm({ ...pipelineForm, webhook: e.target.checked })} />
								Enable authenticated webhook trigger
							</label>
							{pipelineForm.webhook ? (
								<FormField label="Webhook Secret" value={pipelineForm.webhook_secret} onChange={(v) => setPipelineForm({ ...pipelineForm, webhook_secret: v })} required={!editingPipelineId} hint={editingPipelineId ? 'Leave blank to keep the current secret. Provide a new value to rotate it.' : 'The API redacts this secret in pipeline responses.'} />
							) : null}
						</div>
						<div className="form-group">
							<label>Steps</label>
							<StepBuilder value={pipelineForm.steps} onChange={v => setPipelineForm({ ...pipelineForm, steps: v })} />
						</div>
						<Button type="submit" label={editingPipelineId ? 'Save Pipeline' : 'Create Pipeline'} />
					</form>
				</Modal>

				<Modal open={showCheckpointModal} onClose={() => setShowCheckpointModal(false)} title="Pipeline Checkpoint">
					<form onSubmit={handleCheckpointLoad}>
						<FormField label="Pipeline ID" value={checkpointForm.pipeline_id} onChange={(v) => setCheckpointForm({ ...checkpointForm, pipeline_id: v })} required />
						<FormField label="Step Name" value={checkpointForm.step_name} onChange={(v) => setCheckpointForm({ ...checkpointForm, step_name: v })} required hint="Checkpoint state is stored per pipeline step." />
						<FormField label="Scope" value={checkpointForm.scope} onChange={(v) => setCheckpointForm({ ...checkpointForm, scope: v })} hint="Optional checkpoint scope." />
						<div className="inline-actions">
							<Button type="submit" label={checkpointLoading ? 'Loading…' : 'Load Checkpoint'} disabled={checkpointLoading} />
							{checkpointData ? <Button label="Reset Checkpoint" onClick={handleCheckpointReset} variant="secondary" /> : null}
						</div>
					</form>
					{checkpointError ? <div className="error-message">{checkpointError}</div> : null}
					{checkpointData ? (
						<div className="section-panel section-panel--neutral">
							<div className="section-panel-copy"><strong>Version:</strong> {checkpointData.version}</div>
							<pre style={{ margin: '1rem 0 0', fontSize: '0.8rem', whiteSpace: 'pre-wrap' }}>{JSON.stringify(checkpointData.checkpoint || {}, null, 2)}</pre>
						</div>
					) : null}
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

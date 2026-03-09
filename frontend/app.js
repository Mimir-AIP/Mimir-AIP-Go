// API Configuration
const API_URL = window.location.origin.includes('localhost')
	? 'http://localhost:8080'
	: '';

// ============================================
// PRIMITIVE COMPONENTS (as defined in plan)
// ============================================

// 1. Tabs Component
function Tabs({ tabs, activeTab, onTabChange }) {
	return (
		<div className="tabs-container">
			{tabs.map(tab => (
				<button
					key={tab}
					className={`tab${tab === activeTab ? ' active' : ''}`}
					onClick={() => onTabChange(tab)}
				>
					{tab}
				</button>
			))}
		</div>
	);
}

// 2. Form Component (enhanced for multiple fields)
function FormField({ label, type = 'text', value, onChange, options, placeholder, required }) {
	return (
		<div className="form-group">
			<label>{label}{required && ' *'}</label>
			{type === 'select' ? (
				<select value={value} onChange={e => onChange(e.target.value)} required={required}>
					<option value="">Select...</option>
					{options?.map(opt => (
						<option key={opt.value || opt} value={opt.value || opt}>
							{opt.label || opt}
						</option>
					))}
				</select>
			) : type === 'textarea' ? (
				<textarea
					value={value}
					onChange={e => onChange(e.target.value)}
					placeholder={placeholder}
					required={required}
					rows="4"
				/>
			) : (
				<input
					type={type}
					value={value}
					onChange={e => onChange(e.target.value)}
					placeholder={placeholder}
					required={required}
				/>
			)}
		</div>
	);
}

// 3. Table Component
function Table({ columns, data, actions }) {
	if (!data || data.length === 0) {
		return <div className="empty-state">No data available</div>;
	}

	return (
		<table>
			<thead>
				<tr>
					{columns.map(col => (
						<th key={col.key || col}>{col.label || col}</th>
					))}
					{actions && <th>Actions</th>}
				</tr>
			</thead>
			<tbody>
				{data.map((row, i) => (
					<tr key={row.id || i}>
						{columns.map(col => {
							const key = col.key || col;
							const value = col.render ? col.render(row) : row[key];
							return <td key={key}>{value || '-'}</td>;
						})}
						{actions && (
							<td>
								<div style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
									{actions(row)}
								</div>
							</td>
						)}
					</tr>
				))}
			</tbody>
		</table>
	);
}

// 4. Button Component
function Button({ label, onClick, type = 'button', variant = 'primary', disabled }) {
	return (
		<button
			type={type}
			className={variant === 'secondary' ? 'secondary' : variant === 'danger' ? 'danger' : ''}
			style={{
				background: variant === 'primary' ? 'var(--accent)' : undefined,
				color: 'var(--text)',
				fontFamily: 'var(--font-family)',
				border: 'none',
				padding: '8px 16px',
				cursor: disabled ? 'not-allowed' : 'pointer',
				opacity: disabled ? 0.5 : 1,
			}}
			onClick={onClick}
			disabled={disabled}
		>
			{label}
		</button>
	);
}

// 5. Modal Component
function Modal({ open, onClose, title, children }) {
	if (!open) return null;
	return (
		<div className="modal-overlay" onClick={onClose}>
			<div className="modal-content" onClick={e => e.stopPropagation()}>
				{title && (
					<div className="modal-header">
						<h2>{title}</h2>
					</div>
				)}
				{children}
				<div className="modal-actions">
					<Button label="Close" onClick={onClose} variant="secondary" />
				</div>
			</div>
		</div>
	);
}

// 6. Graph Component (using Chart.js)
function Graph({ data, options, type = 'line' }) {
	const canvasRef = React.useRef(null);
	const chartRef = React.useRef(null);

	React.useEffect(() => {
		if (!canvasRef.current || !data) return;

		// Destroy existing chart
		if (chartRef.current) {
			chartRef.current.destroy();
		}

		// Create new chart
		const ctx = canvasRef.current.getContext('2d');
		chartRef.current = new Chart(ctx, {
			type,
			data,
			options: {
				responsive: true,
				maintainAspectRatio: true,
				...options,
				plugins: {
					legend: {
						labels: {
							color: 'var(--text)',
						}
					},
					...options?.plugins,
				},
				scales: {
					x: {
						ticks: { color: 'var(--text)' },
						grid: { color: 'rgba(255, 153, 0, 0.1)' },
					},
					y: {
						ticks: { color: 'var(--text)' },
						grid: { color: 'rgba(255, 153, 0, 0.1)' },
					},
					...options?.scales,
				},
			},
		});

		return () => {
			if (chartRef.current) {
				chartRef.current.destroy();
			}
		};
	}, [data, options, type]);

	return (
		<div className="graph-container">
			<canvas ref={canvasRef}></canvas>
		</div>
	);
}

// ============================================
// API UTILITIES
// ============================================

async function apiCall(endpoint, options = {}) {
	try {
		const response = await fetch(`${API_URL}${endpoint}`, {
			headers: {
				'Content-Type': 'application/json',
				...options.headers,
			},
			...options,
		});

		if (!response.ok) {
			const error = await response.text();
			throw new Error(error || `HTTP ${response.status}`);
		}

		if (response.status === 204) {
			return null;
		}

		const contentType = response.headers.get('content-type') || '';
		if (!contentType.includes('application/json')) {
			return await response.text();
		}

		return await response.json();
	} catch (error) {
		console.error('API Error:', error);
		throw error;
	}
}

// ============================================
// PROJECT CONTEXT
// ============================================

const ProjectContext = React.createContext({ activeProject: null, projects: [] });

// ============================================
// PROJECTS PAGE
// ============================================

function ProjectsPage() {
	const [projects, setProjects] = React.useState([]);
	const [loading, setLoading] = React.useState(true);
	const [showModal, setShowModal] = React.useState(false);
	const [formData, setFormData] = React.useState({
		name: '',
		description: '',
	});

	const loadProjects = async () => {
		setLoading(true);
		try {
			const data = await apiCall('/api/projects');
			setProjects(data || []);
		} catch (error) {
			console.error('Failed to load projects:', error);
		}
		setLoading(false);
	};

	React.useEffect(() => {
		loadProjects();
	}, []);

	const handleSubmit = async (e) => {
		e.preventDefault();
		try {
			await apiCall('/api/projects', {
				method: 'POST',
				body: JSON.stringify(formData),
			});
			setShowModal(false);
			setFormData({ name: '', description: '' });
			loadProjects();
		} catch (error) {
			alert('Failed to create project: ' + error.message);
		}
	};

	const handleDelete = async (id) => {
		if (!confirm('Delete this project?')) return;
		try {
			await apiCall(`/api/projects/${id}`, { method: 'DELETE' });
			loadProjects();
		} catch (error) {
			alert('Failed to delete project: ' + error.message);
		}
	};

	const columns = [
		{ key: 'id', label: 'ID' },
		{ key: 'name', label: 'Name' },
		{ key: 'description', label: 'Description' },
		{
			key: 'status',
			label: 'Status',
			render: (row) => <span className={`status-badge status-${row.status}`}>{row.status}</span>
		},
		{
			key: 'created_at',
			label: 'Created',
			render: (row) => new Date(row.metadata?.created_at || row.created_at).toLocaleDateString()
		},
	];

	return (
		<div className="content-section">
			<div className="section-header">
				<h2>Projects</h2>
				<Button label="+ New Project" onClick={() => setShowModal(true)} />
			</div>

			{loading ? (
				<div className="loading">Loading projects...</div>
			) : (
				<Table
					columns={columns}
					data={projects}
					actions={(row) => (
						<>
							<Button label="Delete" onClick={() => handleDelete(row.id)} variant="danger" />
						</>
					)}
				/>
			)}

			<Modal open={showModal} onClose={() => setShowModal(false)} title="Create New Project">
				<form onSubmit={handleSubmit}>
					<FormField
						label="Project Name"
						value={formData.name}
						onChange={(v) => setFormData({ ...formData, name: v })}
						required
					/>
					<FormField
						label="Description"
						type="textarea"
						value={formData.description}
						onChange={(v) => setFormData({ ...formData, description: v })}
					/>
					<Button type="submit" label="Create Project" />
				</form>
			</Modal>
		</div>
	);
}

// ============================================
// STEP BUILDER (for Pipelines)
// ============================================

// Renders typed parameter fields from a plugin action's ParameterSchema array.
// Falls back to a JSON textarea for complex/unknown types.
function StepParamFields({ paramSchemas, parameters, onUpdate }) {
	const safeParams = (typeof parameters === 'object' && parameters !== null && !Array.isArray(parameters))
		? parameters : {};

	const setParam = (name, val) => onUpdate({ ...safeParams, [name]: val });

	return (
		<div>
			{paramSchemas.map(p => {
				const v = safeParams[p.name] ?? '';
				const label = `${p.name}${p.required ? ' *' : ''}${p.description ? ` — ${p.description}` : ''}`;
				if (p.type === 'boolean') {
					return (
						<div key={p.name} className="form-group">
							<label style={{ display: 'flex', alignItems: 'center', gap: '8px', cursor: 'pointer' }}>
								<input type="checkbox" checked={!!v}
									onChange={e => setParam(p.name, e.target.checked)} />
								{label}
							</label>
						</div>
					);
				}
				if (p.type === 'number' || p.type === 'integer') {
					return (
						<div key={p.name} className="form-group">
							<label>{label}</label>
							<input type="number" value={v}
								onChange={e => setParam(p.name, Number(e.target.value))}
								style={{ width: '100%', padding: '6px 8px', background: 'var(--surface)', color: 'var(--text)', border: '1px solid var(--border)', borderRadius: '4px' }}
							/>
						</div>
					);
				}
				if (p.type === 'object' || p.type === 'array') {
					return (
						<div key={p.name} className="form-group">
							<label>{label} (JSON)</label>
							<textarea rows="3"
								value={typeof v === 'string' ? v : JSON.stringify(v, null, 2)}
								onChange={e => {
									try { setParam(p.name, JSON.parse(e.target.value)); }
									catch { setParam(p.name, e.target.value); }
								}}
								style={{ width: '100%', fontFamily: 'monospace', background: 'var(--surface)', color: 'var(--text)', border: '1px solid var(--border)', borderRadius: '4px', padding: '6px 8px' }}
							/>
						</div>
					);
				}
				// Default: string
				return (
					<div key={p.name} className="form-group">
						<label>{label}</label>
						<input type="text" value={v} onChange={e => setParam(p.name, e.target.value)}
							style={{ width: '100%', padding: '6px 8px', background: 'var(--surface)', color: 'var(--text)', border: '1px solid var(--border)', borderRadius: '4px' }}
						/>
					</div>
				);
			})}
		</div>
	);
}

function StepBuilder({ value, onChange }) {
	const [mode, setMode] = React.useState('visual');
	const [steps, setSteps] = React.useState(() => {
		try { return JSON.parse(value) || []; } catch { return []; }
	});
	const [plugins, setPlugins] = React.useState([]);

	React.useEffect(() => {
		apiCall('/api/plugins').then(data => setPlugins(data || [])).catch(() => {});
	}, []);

	// Keep raw JSON in sync when in visual mode
	React.useEffect(() => {
		if (mode === 'visual') onChange(JSON.stringify(steps, null, 2));
	}, [steps, mode]);

	const addStep = () => setSteps(prev => [...prev, { name: '', plugin: 'default', action: '', parameters: {} }]);
	const removeStep = (i) => setSteps(prev => prev.filter((_, idx) => idx !== i));
	const updateStep = (i, field, val) => setSteps(prev => prev.map((s, idx) => idx === i ? { ...s, [field]: val } : s));

	const builtinActions = ['http_request', 'parse_json', 'set_context', 'get_context', 'if_else', 'goto'];

	const getActions = (pluginName) => {
		if (pluginName === 'default' || pluginName === 'builtin') return builtinActions;
		const plugin = plugins.find(p => p.name === pluginName);
		return (plugin?.plugin_definition?.actions || []).map(a => a.name || a);
	};

	// Returns the ParameterSchema array for a given plugin+action, or null if unknown
	const getParamSchema = (pluginName, actionName) => {
		if (!actionName) return null;
		if (pluginName === 'default' || pluginName === 'builtin') return null; // built-ins have no schema
		const plugin = plugins.find(p => p.name === pluginName);
		const actionDef = (plugin?.plugin_definition?.actions || []).find(a => (a.name || a) === actionName);
		return actionDef?.parameters?.length > 0 ? actionDef.parameters : null;
	};

	return (
		<div>
			<div style={{ display: 'flex', gap: '8px', marginBottom: '8px' }}>
				<Button label="Visual Builder" onClick={() => {
					try { setSteps(JSON.parse(value) || []); } catch {}
					setMode('visual');
				}} variant={mode === 'visual' ? 'primary' : 'secondary'} />
				<Button label="Raw JSON" onClick={() => setMode('raw')} variant={mode === 'raw' ? 'primary' : 'secondary'} />
			</div>

			{mode === 'raw' ? (
				<textarea
					value={value}
					onChange={e => onChange(e.target.value)}
					rows="6"
					style={{
						width: '100%',
						fontFamily: 'monospace',
						background: 'var(--surface)',
						color: 'var(--text)',
						border: '1px solid var(--border)',
						borderRadius: '4px',
						padding: '8px',
						boxSizing: 'border-box',
					}}
				/>
			) : (
				<div>
					{steps.map((step, i) => {
						const paramSchema = getParamSchema(step.plugin, step.action);
						return (
							<div key={i} style={{
								background: 'var(--surface)',
								border: '1px solid var(--border)',
								borderRadius: '6px',
								padding: '12px',
								marginBottom: '8px',
							}}>
								<div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '8px' }}>
									<strong style={{ color: 'var(--accent)' }}>Step {i + 1}</strong>
									<Button label="Remove" onClick={() => removeStep(i)} variant="danger" />
								</div>
								<div className="form-grid">
									<FormField
										label="Step Name"
										value={step.name}
										onChange={v => updateStep(i, 'name', v)}
										placeholder="e.g. fetch-data"
									/>
									<FormField
										label="Plugin"
										type="select"
										value={step.plugin}
										onChange={v => updateStep(i, 'plugin', v)}
										options={[{ value: 'default', label: 'default (built-in)' }, ...plugins.map(p => ({ value: p.name, label: p.name }))]}
									/>
								</div>
								<FormField
									label="Action"
									type="select"
									value={step.action}
									onChange={v => updateStep(i, 'action', v)}
									options={getActions(step.plugin).map(a => ({ value: a, label: a }))}
								/>
								{paramSchema ? (
									<StepParamFields
										paramSchemas={paramSchema}
										parameters={step.parameters}
										onUpdate={v => updateStep(i, 'parameters', v)}
									/>
								) : (
									<FormField
										label="Parameters (JSON)"
										type="textarea"
										value={typeof step.parameters === 'string' ? step.parameters : JSON.stringify(step.parameters, null, 2)}
										onChange={v => {
											try { updateStep(i, 'parameters', JSON.parse(v)); }
											catch { updateStep(i, 'parameters', v); }
										}}
										placeholder='{"url": "https://..."}'
									/>
								)}
							</div>
						);
					})}
					<Button label="+ Add Step" onClick={addStep} variant="secondary" />
				</div>
			)}
		</div>
	);
}

// ============================================
// CRON BUILDER (for Schedules)
// ============================================

const CRON_PRESETS = [
	{ label: 'Every minute',       cron: '* * * * *'   },
	{ label: 'Every hour',         cron: '0 * * * *'   },
	{ label: 'Every day midnight', cron: '0 0 * * *'   },
	{ label: 'Every Monday',       cron: '0 0 * * 1'   },
	{ label: 'Every month (1st)',  cron: '0 0 1 * *'   },
];

function CronBuilder({ value, onChange }) {
	const [showCustom, setShowCustom] = React.useState(false);

	const applyPreset = (cron) => {
		onChange(cron);
		setShowCustom(false);
	};

	const activePreset = CRON_PRESETS.find(p => p.cron === value);

	return (
		<div>
			<div style={{ display: 'flex', flexWrap: 'wrap', gap: '6px', marginBottom: '10px' }}>
				{CRON_PRESETS.map(p => (
					<button key={p.cron} type="button" onClick={() => applyPreset(p.cron)} style={{
						padding: '4px 10px',
						fontSize: '0.8rem',
						background: value === p.cron ? 'var(--accent)' : 'var(--surface)',
						color: value === p.cron ? '#000' : 'var(--text)',
						border: '1px solid var(--border)',
						borderRadius: '4px',
						cursor: 'pointer',
					}}>
						{p.label}
					</button>
				))}
				<button type="button" onClick={() => setShowCustom(v => !v)} style={{
					padding: '4px 10px',
					fontSize: '0.8rem',
					background: showCustom ? 'var(--accent)' : 'var(--surface)',
					color: showCustom ? '#000' : 'var(--text)',
					border: '1px solid var(--border)',
					borderRadius: '4px',
					cursor: 'pointer',
				}}>
					Custom…
				</button>
			</div>

			{showCustom && (
				<input
					type="text"
					value={value}
					onChange={e => onChange(e.target.value)}
					placeholder="* * * * *  (min hour day month weekday)"
					style={{
						width: '100%',
						padding: '6px 8px',
						fontSize: '0.85rem',
						fontFamily: 'monospace',
						background: 'var(--surface)',
						color: 'var(--text)',
						border: '1px solid var(--border)',
						borderRadius: '4px',
						boxSizing: 'border-box',
						marginBottom: '10px',
					}}
				/>
			)}

			<div style={{ fontFamily: 'monospace', fontSize: '0.85rem', padding: '6px 8px', background: 'var(--surface)', border: '1px solid var(--border)', borderRadius: '4px', display: 'flex', alignItems: 'center', gap: '10px' }}>
				<span style={{ color: 'var(--accent)' }}>{value || '* * * * *'}</span>
				{activePreset && <span style={{ color: 'var(--text-secondary)', fontSize: '0.78rem' }}>({activePreset.label})</span>}
			</div>
		</div>
	);
}

// ============================================
// PIPELINES PAGE
// ============================================

function PipelinesPage() {
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
		pipeline_id: '',
		project_id: '',
		cron_expression: '',
		enabled: true,
	});

	// Default project_id when activeProject changes
	React.useEffect(() => {
		if (activeProject) {
			setPipelineForm(prev => ({ ...prev, project_id: activeProject.id }));
			setScheduleForm(prev => ({ ...prev, project_id: activeProject.id }));
		}
	}, [activeProject]);

	// Subscribe to WebSocket for pipeline execution status updates
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
		}
		setLoading(false);
	};

	React.useEffect(() => {
		loadData();
	}, []);

	const handlePipelineSubmit = async (e) => {
		e.preventDefault();
		try {
			const data = {
				...pipelineForm,
				steps: JSON.parse(pipelineForm.steps),
			};
			await apiCall('/api/pipelines', {
				method: 'POST',
				body: JSON.stringify(data),
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
				body: JSON.stringify(scheduleForm),
			});
			setShowScheduleModal(false);
			setScheduleForm({ name: '', pipeline_id: '', project_id: activeProject?.id || '', cron_expression: '', enabled: true });
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
				const n = { ...prev };
				delete n[id];
				return n;
			}), 5000);
		} catch (error) {
			setExecutionStatus(prev => ({ ...prev, [id]: 'error' }));
		}
	};

	const projectOptions = projects.map(p => ({ value: p.id, label: p.name }));
	const pipelineOptions = pipelines.map(p => ({ value: p.id, label: p.name }));

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
		{ key: 'pipeline_id', label: 'Pipeline ID' },
		{ key: 'cron_expression', label: 'Cron Expression' },
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

			<Tabs
				tabs={['Pipelines', 'Recurring Jobs']}
				activeTab={selectedTab}
				onTabChange={setSelectedTab}
			/>

			{loading ? (
				<div className="loading">Loading...</div>
			) : selectedTab === 'Pipelines' ? (
				<Table
					columns={pipelineColumns}
					data={pipelines}
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
					data={schedules}
					actions={(row) => (
						<>
							<Button label="Delete" onClick={() => handleDeleteSchedule(row.id)} variant="danger" />
						</>
					)}
				/>
			)}

			<Modal open={showPipelineModal} onClose={() => setShowPipelineModal(false)} title="Create New Pipeline">
				<form onSubmit={handlePipelineSubmit}>
					<div className="form-grid">
						<FormField
							label="Pipeline Name"
							value={pipelineForm.name}
							onChange={(v) => setPipelineForm({ ...pipelineForm, name: v })}
							required
						/>
						<FormField
							label="Project"
							type="select"
							value={pipelineForm.project_id}
							onChange={(v) => setPipelineForm({ ...pipelineForm, project_id: v })}
							options={projectOptions}
							required
						/>
					</div>
					<FormField
						label="Pipeline Type"
						type="select"
						value={pipelineForm.type}
						onChange={(v) => setPipelineForm({ ...pipelineForm, type: v })}
						options={['ingestion', 'processing', 'output']}
						required
					/>
					<FormField
						label="Description"
						type="textarea"
						value={pipelineForm.description}
						onChange={(v) => setPipelineForm({ ...pipelineForm, description: v })}
					/>
					<div className="form-group">
						<label>Steps</label>
						<StepBuilder
							value={pipelineForm.steps}
							onChange={v => setPipelineForm({ ...pipelineForm, steps: v })}
						/>
					</div>
					<Button type="submit" label="Create Pipeline" />
				</form>
			</Modal>

			<Modal open={showScheduleModal} onClose={() => setShowScheduleModal(false)} title="Create Recurring Job">
				<form onSubmit={handleScheduleSubmit}>
					<div className="form-grid">
						<FormField
							label="Schedule Name"
							value={scheduleForm.name}
							onChange={(v) => setScheduleForm({ ...scheduleForm, name: v })}
							required
						/>
						<FormField
							label="Pipeline"
							type="select"
							value={scheduleForm.pipeline_id}
							onChange={(v) => setScheduleForm({ ...scheduleForm, pipeline_id: v })}
							options={pipelineOptions}
							required
						/>
					</div>
					<div className="form-grid">
						<FormField
							label="Project"
							type="select"
							value={scheduleForm.project_id}
							onChange={(v) => setScheduleForm({ ...scheduleForm, project_id: v })}
							options={projectOptions}
							required
						/>
						<div className="form-group">
							<label>Schedule *</label>
							<CronBuilder
								value={scheduleForm.cron_expression}
								onChange={v => setScheduleForm({ ...scheduleForm, cron_expression: v })}
							/>
						</div>
					</div>
					<Button type="submit" label="Create Schedule" />
				</form>
			</Modal>
		</div>
	);
}

// ============================================
// ONTOLOGIES PAGE
// ============================================

function OntologiesPage() {
	const { activeProject, projects } = React.useContext(ProjectContext);
	const [ontologies, setOntologies] = React.useState([]);
	const [loading, setLoading] = React.useState(true);
	const [showModal, setShowModal] = React.useState(false);
	const [showExtractionModal, setShowExtractionModal] = React.useState(false);
	const [storageConfigs, setStorageConfigs] = React.useState([]);
	const [formData, setFormData] = React.useState({
		name: '',
		project_id: '',
		schema: '{}',
	});
	const [extractionForm, setExtractionForm] = React.useState({
		project_id: '',
		storage_config_id: '',
		ontology_name: '',
	});

	// Default project_id when activeProject changes
	React.useEffect(() => {
		if (activeProject) {
			setFormData(prev => ({ ...prev, project_id: activeProject.id }));
			setExtractionForm(prev => ({ ...prev, project_id: activeProject.id }));
		}
	}, [activeProject]);

	const loadOntologies = async () => {
		setLoading(true);
		try {
			const projectId = activeProject?.id || '';
			const data = await apiCall(`/api/ontologies?project_id=${projectId}`);
			setOntologies(data || []);
		} catch (error) {
			console.error('Failed to load ontologies:', error);
			setOntologies([]);
		}
		setLoading(false);
	};

	React.useEffect(() => {
		loadOntologies();
	}, [activeProject]);

	const openExtractionModal = async () => {
		if (activeProject?.id) {
			try {
				const configs = await apiCall(`/api/storage/configs?project_id=${activeProject.id}`);
				setStorageConfigs(configs || []);
			} catch {
				setStorageConfigs([]);
			}
		} else {
			setStorageConfigs([]);
		}
		setShowExtractionModal(true);
	};

	const handleSubmit = async (e) => {
		e.preventDefault();
		try {
			const data = {
				...formData,
				schema: JSON.parse(formData.schema),
			};
			await apiCall('/api/ontologies', {
				method: 'POST',
				body: JSON.stringify(data),
			});
			setShowModal(false);
			setFormData({ name: '', project_id: activeProject?.id || '', schema: '{}' });
			loadOntologies();
		} catch (error) {
			alert('Failed to create ontology: ' + error.message);
		}
	};

	const handleDelete = async (id) => {
		if (!confirm('Delete this ontology?')) return;
		try {
			await apiCall(`/api/ontologies/${id}`, { method: 'DELETE' });
			loadOntologies();
		} catch (error) {
			alert('Failed to delete ontology: ' + error.message);
		}
	};

	const handleApprove = async (id) => {
		try {
			await apiCall(`/api/ontologies/${id}`, { method: 'PUT', body: JSON.stringify({ status: 'active' }) });
			loadOntologies();
		} catch (error) {
			alert('Failed to approve ontology: ' + error.message);
		}
	};

	const handleReject = async (id) => {
		try {
			await apiCall(`/api/ontologies/${id}`, { method: 'PUT', body: JSON.stringify({ status: 'draft' }) });
			loadOntologies();
		} catch (error) {
			alert('Failed to reject ontology: ' + error.message);
		}
	};

	const handleExtraction = async (e) => {
		e.preventDefault();
		try {
			const result = await apiCall('/api/extraction/generate-ontology', {
				method: 'POST',
				body: JSON.stringify(extractionForm),
			});
			alert('Ontology extraction started!\n' + JSON.stringify(result, null, 2));
			setShowExtractionModal(false);
			setExtractionForm({ project_id: activeProject?.id || '', storage_config_id: '', ontology_name: '' });
			setTimeout(loadOntologies, 2000);
		} catch (error) {
			alert('Failed to start extraction: ' + error.message);
		}
	};

	const projectOptions = projects.map(p => ({ value: p.id, label: p.name }));
	const storageConfigOptions = storageConfigs.map(c => ({ value: c.id, label: c.name }));

	const columns = [
		{ key: 'id', label: 'ID' },
		{ key: 'name', label: 'Name' },
		{ key: 'project_id', label: 'Project ID' },
		{ key: 'version', label: 'Version' },
		{
			key: 'status',
			label: 'Status',
			render: (row) => <span className={`status-badge status-${row.status}`}>{row.status}</span>
		},
		{ key: 'created_at', label: 'Created', render: (row) => new Date(row.created_at).toLocaleDateString() },
	];

	return (
		<div className="content-section">
			<div className="section-header">
				<h2>Ontologies</h2>
				<div style={{ display: 'flex', gap: '8px' }}>
					<Button label="+ New Ontology" onClick={() => setShowModal(true)} />
					<Button label="Extract from Storage" onClick={openExtractionModal} variant="secondary" />
				</div>
			</div>

			{loading ? (
				<div className="loading">Loading ontologies...</div>
			) : (
				<Table
					columns={columns}
					data={ontologies}
					actions={(row) => (
						<>
							{row.status === 'needs_review' && <Button label="Approve" onClick={() => handleApprove(row.id)} variant="secondary" />}
							{row.status === 'needs_review' && <Button label="Reject" onClick={() => handleReject(row.id)} variant="secondary" />}
							{row.status === 'draft' && <Button label="Promote" onClick={() => handleApprove(row.id)} variant="secondary" />}
							<Button label="Delete" onClick={() => handleDelete(row.id)} variant="danger" />
						</>
					)}
				/>
			)}

			<Modal open={showModal} onClose={() => setShowModal(false)} title="Create New Ontology">
				<form onSubmit={handleSubmit}>
					<div className="form-grid">
						<FormField
							label="Ontology Name"
							value={formData.name}
							onChange={(v) => setFormData({ ...formData, name: v })}
							required
						/>
						<FormField
							label="Project"
							type="select"
							value={formData.project_id}
							onChange={(v) => setFormData({ ...formData, project_id: v })}
							options={projectOptions}
							required
						/>
					</div>
					<FormField
						label="Schema (JSON)"
						type="textarea"
						value={formData.schema}
						onChange={(v) => setFormData({ ...formData, schema: v })}
						placeholder='{"entities": [], "relationships": []}'
						required
					/>
					<Button type="submit" label="Create Ontology" />
				</form>
			</Modal>

			<Modal open={showExtractionModal} onClose={() => setShowExtractionModal(false)} title="Extract Ontology from Storage">
				<form onSubmit={handleExtraction}>
					<div className="form-grid">
						<FormField
							label="Project"
							type="select"
							value={extractionForm.project_id}
							onChange={(v) => setExtractionForm({ ...extractionForm, project_id: v })}
							options={projectOptions}
							required
						/>
						<FormField
							label="Storage Config"
							type="select"
							value={extractionForm.storage_config_id}
							onChange={(v) => setExtractionForm({ ...extractionForm, storage_config_id: v })}
							options={storageConfigOptions}
							required
						/>
					</div>
					<FormField
						label="Ontology Name"
						value={extractionForm.ontology_name}
						onChange={(v) => setExtractionForm({ ...extractionForm, ontology_name: v })}
						placeholder="Extracted Ontology"
						required
					/>
					<Button type="submit" label="Start Extraction" />
				</form>
			</Modal>
		</div>
	);
}

// ============================================
// ML MODELS PAGE
// ============================================

function MLModelsPage() {
	const { activeProject, projects } = React.useContext(ProjectContext);
	const [models, setModels] = React.useState([]);
	const [loading, setLoading] = React.useState(true);
	const [showModal, setShowModal] = React.useState(false);
	const [showRecommendModal, setShowRecommendModal] = React.useState(false);
	const [showTrainModal, setShowTrainModal] = React.useState(false);
	const [trainingTarget, setTrainingTarget] = React.useState(null);
	const [trainStorageIds, setTrainStorageIds] = React.useState([]);
	const [availableStorageConfigs, setAvailableStorageConfigs] = React.useState([]);
	const [recommendForm, setRecommendForm] = React.useState({
		project_id: '',
		ontology_id: '',
	});
	const [recommendOntologies, setRecommendOntologies] = React.useState([]);
	const [recommendResult, setRecommendResult] = React.useState(null);
	const [formData, setFormData] = React.useState({
		name: '',
		project_id: '',
		model_type: '',
		version: '1.0.0',
		config: '{}',
	});
	const [trainingMetrics, setTrainingMetrics] = React.useState({});

	// Default project_id when activeProject changes
	React.useEffect(() => {
		if (activeProject) {
			setFormData(prev => ({ ...prev, project_id: activeProject.id }));
			setRecommendForm(prev => ({ ...prev, project_id: activeProject.id, ontology_id: '' }));
			apiCall(`/api/ontologies?project_id=${activeProject.id}`)
				.then(data => setRecommendOntologies(data || []))
				.catch(() => {});
		}
	}, [activeProject]);

	// Subscribe to WS task updates for training progress
	useTaskWebSocket((task) => {
		if (task.type !== 'ml_training') return;
		const modelID = task.task_spec && task.task_spec.model_id;
		if (!modelID) return;
		const metrics = task.task_spec && task.task_spec.parameters && task.task_spec.parameters.training_metrics;
		if (!metrics) return;
		setTrainingMetrics((prev) => ({
			...prev,
			[modelID]: metrics,
		}));
		// Reload models when training completes
		if (task.status === 'completed' || task.status === 'failed') {
			loadModels();
		}
	});

	const loadModels = async () => {
		setLoading(true);
		try {
			const projectId = activeProject?.id || '';
			const data = await apiCall(`/api/ml-models?project_id=${projectId}`);
			setModels(data || []);
		} catch (error) {
			console.error('Failed to load ML models:', error);
			setModels([]);
		}
		setLoading(false);
	};

	React.useEffect(() => {
		loadModels();
	}, [activeProject]);

	const handleSubmit = async (e) => {
		e.preventDefault();
		try {
			const data = {
				...formData,
				config: JSON.parse(formData.config),
			};
			await apiCall('/api/ml-models', {
				method: 'POST',
				body: JSON.stringify(data),
			});
			setShowModal(false);
			setFormData({ name: '', project_id: activeProject?.id || '', model_type: '', version: '1.0.0', config: '{}' });
			loadModels();
		} catch (error) {
			alert('Failed to create ML model: ' + error.message);
		}
	};

	const handleDelete = async (id) => {
		if (!confirm('Delete this ML model?')) return;
		try {
			await apiCall(`/api/ml-models/${id}`, { method: 'DELETE' });
			loadModels();
		} catch (error) {
			alert('Failed to delete ML model: ' + error.message);
		}
	};

	const openTrainModal = async (row) => {
		setTrainingTarget(row);
		try {
			const configs = await apiCall(`/api/storage/configs?project_id=${row.project_id}`);
			setAvailableStorageConfigs(configs || []);
		} catch {
			setAvailableStorageConfigs([]);
		}
		setTrainStorageIds([]);
		setShowTrainModal(true);
	};

	const handleTrain = async () => {
		try {
			await apiCall('/api/ml-models/train', {
				method: 'POST',
				body: JSON.stringify({ model_id: trainingTarget.id, storage_ids: trainStorageIds }),
			});
			setShowTrainModal(false);
		} catch (error) {
			alert('Failed to start training: ' + error.message);
		}
	};

	const handleRecommend = async (e) => {
		e.preventDefault();
		try {
			const data = await apiCall('/api/ml-models/recommend', {
				method: 'POST',
				body: JSON.stringify({
					project_id: recommendForm.project_id,
					ontology_id: recommendForm.ontology_id,
				}),
			});
			setRecommendResult(data);
		} catch (error) {
			alert('Failed to get recommendation: ' + error.message);
		}
	};

	// Fetch ontologies for recommend modal when project changes
	const onRecommendProjectChange = async (projectId) => {
		setRecommendForm(prev => ({ ...prev, project_id: projectId, ontology_id: '' }));
		setRecommendOntologies([]);
		if (projectId) {
			try {
				const data = await apiCall(`/api/ontologies?project_id=${projectId}`);
				setRecommendOntologies(data || []);
			} catch {
				setRecommendOntologies([]);
			}
		}
	};

	const projectOptions = projects.map(p => ({ value: p.id, label: p.name }));

	const columns = [
		{ key: 'id', label: 'ID' },
		{ key: 'name', label: 'Name' },
		{ key: 'model_type', label: 'Type' },
		{ key: 'version', label: 'Version' },
		{
			key: 'status',
			label: 'Status',
			render: (row) => <span className={`status-badge status-${row.status}`}>{row.status}</span>
		},
		{ key: 'project_id', label: 'Project ID' },
		{ key: 'created_at', label: 'Created', render: (row) => new Date(row.created_at).toLocaleDateString() },
	];

	return (
		<div className="content-section">
			<div className="section-header">
				<h2>ML Models</h2>
				<div style={{ display: 'flex', gap: '8px' }}>
					<Button label="Recommend" onClick={() => { setRecommendResult(null); setRecommendForm({ project_id: activeProject?.id || '', ontology_id: '' }); setRecommendOntologies([]); setShowRecommendModal(true); }} variant="secondary" />
					<Button label="+ New Model" onClick={() => setShowModal(true)} />
				</div>
			</div>

			{loading ? (
				<div className="loading">Loading ML models...</div>
			) : (
				<Table
					columns={columns}
					data={models}
					actions={(row) => (
						<>
							<Button label="Train" onClick={() => openTrainModal(row)} variant="secondary" />
							<Button label="Delete" onClick={() => handleDelete(row.id)} variant="danger" />
						</>
					)}
				/>
			)}

			{Object.keys(trainingMetrics).length > 0 && (
				<div style={{ marginTop: '24px' }}>
					<h3 style={{ color: 'var(--text-primary)', marginBottom: '12px' }}>Training Progress</h3>
					{Object.entries(trainingMetrics).map(([modelID, metrics]) => {
						const epochs = metrics.epochs || [];
						const loss = metrics.loss || [];
						const accuracy = metrics.accuracy || [];
						if (epochs.length === 0) return null;
						const graphData = {
							labels: epochs,
							datasets: [
								{
									label: 'Loss',
									data: loss,
									borderColor: '#ef4444',
									backgroundColor: 'rgba(239,68,68,0.1)',
									yAxisID: 'y',
								},
								{
									label: 'Accuracy',
									data: accuracy,
									borderColor: '#22c55e',
									backgroundColor: 'rgba(34,197,94,0.1)',
									yAxisID: 'y1',
								},
							],
						};
						const graphOptions = {
							scales: {
								y: { type: 'linear', position: 'left', title: { display: true, text: 'Loss' } },
								y1: { type: 'linear', position: 'right', title: { display: true, text: 'Accuracy' }, grid: { drawOnChartArea: false } },
							},
						};
						return (
							<div key={modelID} style={{ marginBottom: '16px', padding: '12px', background: 'var(--surface)', borderRadius: '6px' }}>
								<div style={{ fontSize: '12px', color: 'var(--text-secondary)', marginBottom: '8px' }}>Model: {modelID}</div>
								<Graph data={graphData} options={graphOptions} type="line" />
							</div>
						);
					})}
				</div>
			)}

			<Modal open={showTrainModal} onClose={() => setShowTrainModal(false)} title={`Train: ${trainingTarget?.name}`}>
				<p>Select storage configs to train on:</p>
				{availableStorageConfigs.length === 0 ? (
					<p style={{ color: 'var(--text-secondary)' }}>No storage configs found for this project.</p>
				) : (
					availableStorageConfigs.map(cfg => (
						<label key={cfg.id} style={{ display: 'block', marginBottom: '8px', cursor: 'pointer' }}>
							<input
								type="checkbox"
								checked={trainStorageIds.includes(cfg.id)}
								onChange={e => setTrainStorageIds(prev =>
									e.target.checked ? [...prev, cfg.id] : prev.filter(id => id !== cfg.id)
								)}
							/>
							{' '}{cfg.name} ({cfg.storage_type})
						</label>
					))
				)}
				<div style={{ marginTop: '16px' }}>
					<Button label="Start Training" onClick={handleTrain} />
				</div>
			</Modal>

			<Modal open={showRecommendModal} onClose={() => setShowRecommendModal(false)} title="Recommend Model Type">
				<form onSubmit={handleRecommend}>
					<p style={{ color: 'var(--text-secondary)', fontSize: '0.875rem', marginBottom: '12px' }}>
						Select your project and ontology — the backend analyses your data automatically.
					</p>
					<FormField
						label="Project"
						type="select"
						value={recommendForm.project_id}
						onChange={onRecommendProjectChange}
						options={projectOptions}
						required
					/>
					<FormField
						label="Ontology"
						type="select"
						value={recommendForm.ontology_id}
						onChange={(v) => setRecommendForm(prev => ({ ...prev, ontology_id: v }))}
						options={recommendOntologies.map(o => ({ value: o.id, label: o.name }))}
						required
					/>
					<Button type="submit" label="Get Recommendation" disabled={!recommendForm.project_id || !recommendForm.ontology_id} />
					{recommendResult && (
						<div style={{ marginTop: '16px', padding: '12px', background: 'var(--surface)', borderRadius: '6px' }}>
							<strong>Recommended:</strong> {recommendResult.recommended_type}<br />
							<strong>Confidence:</strong> {(recommendResult.confidence * 100).toFixed(0)}%<br />
							<strong>Reason:</strong> {recommendResult.reason}
							<div style={{ marginTop: '12px' }}>
								<Button label="Use This" onClick={() => {
									setFormData(prev => ({
										...prev,
										model_type: recommendResult.recommended_type,
										project_id: recommendForm.project_id,
									}));
									setShowRecommendModal(false);
									setShowModal(true);
								}} />
							</div>
						</div>
					)}
				</form>
			</Modal>

			<Modal open={showModal} onClose={() => setShowModal(false)} title="Create New ML Model">
				<form onSubmit={handleSubmit}>
					<div className="form-grid">
						<FormField
							label="Model Name"
							value={formData.name}
							onChange={(v) => setFormData({ ...formData, name: v })}
							required
						/>
						<FormField
							label="Project"
							type="select"
							value={formData.project_id}
							onChange={(v) => setFormData({ ...formData, project_id: v })}
							options={projectOptions}
							required
						/>
					</div>
					<div className="form-grid">
						<FormField
							label="Model Type"
							type="select"
							value={formData.model_type}
							onChange={(v) => setFormData({ ...formData, model_type: v })}
							options={['classification', 'regression', 'clustering', 'forecasting', 'anomaly_detection']}
							required
						/>
						<FormField
							label="Version"
							value={formData.version}
							onChange={(v) => setFormData({ ...formData, version: v })}
							required
						/>
					</div>
					<FormField
						label="Configuration (JSON)"
						type="textarea"
						value={formData.config}
						onChange={(v) => setFormData({ ...formData, config: v })}
						placeholder='{"hyperparameters": {}}'
					/>
					<Button type="submit" label="Create Model" />
				</form>
			</Modal>
		</div>
	);
}

// ============================================
// DIGITAL TWINS PAGE
// ============================================

function DigitalTwinsPage() {
	const { activeProject, projects } = React.useContext(ProjectContext);
	const [twins, setTwins] = React.useState([]);
	const [loading, setLoading] = React.useState(true);
	const [showModal, setShowModal] = React.useState(false);
	const [showActionModal, setShowActionModal] = React.useState(false);
	const [showQueryModal, setShowQueryModal] = React.useState(false);
	const [selectedTab, setSelectedTab] = React.useState('Digital Twins');
	const [selectedTwin, setSelectedTwin] = React.useState(null);
	const [entities, setEntities] = React.useState([]);
	const [scenarios, setScenarios] = React.useState([]);
	const [actions, setActions] = React.useState([]);
	const [queryResult, setQueryResult] = React.useState(null);
	const [ontologies, setOntologies] = React.useState([]);
	const [mlModels, setMlModels] = React.useState([]);
	const [formData, setFormData] = React.useState({
		name: '',
		project_id: '',
		ontology_id: '',
		ml_model_id: '',
		description: '',
	});
	const [actionForm, setActionForm] = React.useState({
		name: '',
		action_type: '',
		parameters: '{}',
	});
	const [queryForm, setQueryForm] = React.useState({
		query: '',
	});

	// Default project_id and fetch related resources when activeProject changes
	React.useEffect(() => {
		if (activeProject) {
			setFormData(prev => ({ ...prev, project_id: activeProject.id }));
			apiCall(`/api/ontologies?project_id=${activeProject.id}`)
				.then(data => setOntologies(data || []))
				.catch(() => setOntologies([]));
			apiCall(`/api/ml-models?project_id=${activeProject.id}`)
				.then(data => setMlModels(data || []))
				.catch(() => setMlModels([]));
		}
	}, [activeProject]);

	const loadTwins = async () => {
		setLoading(true);
		try {
			const projectId = activeProject?.id || '';
			const data = await apiCall(`/api/digital-twins?project_id=${projectId}`);
			setTwins(data || []);
		} catch (error) {
			console.error('Failed to load digital twins:', error);
			setTwins([]);
		}
		setLoading(false);
	};

	const loadTwinDetails = async (twinId) => {
		try {
			const [entitiesData, scenariosData, actionsData] = await Promise.all([
				apiCall(`/api/digital-twins/${twinId}/entities`),
				apiCall(`/api/digital-twins/${twinId}/scenarios`),
				apiCall(`/api/digital-twins/${twinId}/actions`),
			]);
			setEntities(entitiesData || []);
			setScenarios(scenariosData || []);
			setActions(actionsData || []);
		} catch (error) {
			console.error('Failed to load twin details:', error);
		}
	};

	React.useEffect(() => {
		loadTwins();
	}, [activeProject]);

	React.useEffect(() => {
		if (selectedTwin) {
			loadTwinDetails(selectedTwin.id);
		}
	}, [selectedTwin]);

	const handleSubmit = async (e) => {
		e.preventDefault();
		try {
			await apiCall('/api/digital-twins', {
				method: 'POST',
				body: JSON.stringify(formData),
			});
			setShowModal(false);
			setFormData({ name: '', project_id: activeProject?.id || '', ontology_id: '', ml_model_id: '', description: '' });
			loadTwins();
		} catch (error) {
			alert('Failed to create digital twin: ' + error.message);
		}
	};

	const handleDelete = async (id) => {
		if (!confirm('Delete this digital twin?')) return;
		try {
			await apiCall(`/api/digital-twins/${id}`, { method: 'DELETE' });
			loadTwins();
		} catch (error) {
			alert('Failed to delete digital twin: ' + error.message);
		}
	};

	const handleSync = async (id) => {
		try {
			await apiCall(`/api/digital-twins/${id}/sync`, { method: 'POST', body: JSON.stringify({}) });
			alert('Sync started!');
		} catch (error) {
			alert('Failed to sync digital twin: ' + error.message);
		}
	};

	const handleCreateAction = async (e) => {
		e.preventDefault();
		try {
			const data = {
				...actionForm,
				parameters: JSON.parse(actionForm.parameters),
			};
			await apiCall(`/api/digital-twins/${selectedTwin.id}/actions`, {
				method: 'POST',
				body: JSON.stringify(data),
			});
			setShowActionModal(false);
			setActionForm({ name: '', action_type: '', parameters: '{}' });
			loadTwinDetails(selectedTwin.id);
		} catch (error) {
			alert('Failed to create action: ' + error.message);
		}
	};

	const handleDeleteAction = async (actionId) => {
		if (!confirm('Delete this action?')) return;
		try {
			await apiCall(`/api/digital-twins/${selectedTwin.id}/actions/${actionId}`, { method: 'DELETE' });
			loadTwinDetails(selectedTwin.id);
		} catch (error) {
			alert('Failed to delete action: ' + error.message);
		}
	};

	const handleQuery = async (e) => {
		e.preventDefault();
		try {
			const result = await apiCall(`/api/digital-twins/${selectedTwin.id}/query`, {
				method: 'POST',
				body: JSON.stringify({ query: queryForm.query }),
			});
			setQueryResult(result);
		} catch (error) {
			alert('Failed to execute query: ' + error.message);
		}
	};

	const projectOptions = projects.map(p => ({ value: p.id, label: p.name }));
	const ontologyOptions = ontologies.map(o => ({ value: o.id, label: o.name }));
	const mlModelOptions = mlModels.map(m => ({ value: m.id, label: m.name }));

	const twinColumns = [
		{ key: 'id', label: 'ID' },
		{ key: 'name', label: 'Name' },
		{ key: 'description', label: 'Description' },
		{ key: 'project_id', label: 'Project ID' },
		{ key: 'ontology_id', label: 'Ontology ID' },
		{ key: 'ml_model_id', label: 'ML Model ID' },
		{
			key: 'status',
			label: 'Status',
			render: (row) => <span className={`status-badge status-${row.status}`}>{row.status}</span>
		},
	];

	const entityColumns = [
		{ key: 'id', label: 'Entity ID' },
		{ key: 'type', label: 'Type' },
		{ key: 'properties', label: 'Properties', render: (row) => JSON.stringify(row.properties || {}).substring(0, 50) + '...' },
		{ key: 'updated_at', label: 'Updated', render: (row) => new Date(row.updated_at).toLocaleString() },
	];

	const scenarioColumns = [
		{ key: 'id', label: 'Scenario ID' },
		{ key: 'name', label: 'Name' },
		{ key: 'description', label: 'Description' },
		{ key: 'created_at', label: 'Created', render: (row) => new Date(row.created_at).toLocaleDateString() },
	];

	const actionColumns = [
		{ key: 'id', label: 'Action ID' },
		{ key: 'name', label: 'Name' },
		{ key: 'action_type', label: 'Type' },
		{ key: 'status', label: 'Status', render: (row) => <span className={`status-badge status-${row.status}`}>{row.status}</span> },
		{ key: 'created_at', label: 'Created', render: (row) => new Date(row.created_at).toLocaleDateString() },
	];

	return (
		<div className="content-section">
			<div className="section-header">
				<h2>Digital Twins</h2>
				<Button label="+ New Digital Twin" onClick={() => setShowModal(true)} />
			</div>

			{selectedTwin ? (
				<>
					<div style={{ marginBottom: '16px' }}>
						<Button label="← Back to List" onClick={() => setSelectedTwin(null)} variant="secondary" />
						<h3 style={{ color: 'var(--accent)', marginTop: '16px' }}>
							{selectedTwin.name} - Details
						</h3>
						<div style={{ display: 'flex', gap: '8px', marginTop: '8px' }}>
							<Button label="+ New Action" onClick={() => setShowActionModal(true)} variant="secondary" />
							<Button label="Query Twin" onClick={() => setShowQueryModal(true)} variant="secondary" />
						</div>
					</div>

					<Tabs
						tabs={['Entities', 'Scenarios', 'Actions']}
						activeTab={selectedTab}
						onTabChange={setSelectedTab}
					/>

					{selectedTab === 'Entities' ? (
						<Table columns={entityColumns} data={entities} />
					) : selectedTab === 'Scenarios' ? (
						<Table columns={scenarioColumns} data={scenarios} />
					) : (
						<Table
							columns={actionColumns}
							data={actions}
							actions={(row) => (
								<Button label="Delete" onClick={() => handleDeleteAction(row.id)} variant="danger" />
							)}
						/>
					)}
				</>
			) : (
				<>
					{loading ? (
						<div className="loading">Loading digital twins...</div>
					) : (
						<Table
							columns={twinColumns}
							data={twins}
							actions={(row) => (
								<>
									<Button label="View" onClick={() => setSelectedTwin(row)} variant="secondary" />
									<Button label="Sync" onClick={() => handleSync(row.id)} variant="secondary" />
									<Button label="Delete" onClick={() => handleDelete(row.id)} variant="danger" />
								</>
							)}
						/>
					)}
				</>
			)}

			<Modal open={showModal} onClose={() => setShowModal(false)} title="Create New Digital Twin">
				<form onSubmit={handleSubmit}>
					<div className="form-grid">
						<FormField
							label="Twin Name"
							value={formData.name}
							onChange={(v) => setFormData({ ...formData, name: v })}
							required
						/>
						<FormField
							label="Project"
							type="select"
							value={formData.project_id}
							onChange={(v) => setFormData({ ...formData, project_id: v })}
							options={projectOptions}
							required
						/>
					</div>
					<div className="form-grid">
						<FormField
							label="Ontology"
							type="select"
							value={formData.ontology_id}
							onChange={(v) => setFormData({ ...formData, ontology_id: v })}
							options={ontologyOptions}
							required
						/>
						<FormField
							label="ML Model"
							type="select"
							value={formData.ml_model_id}
							onChange={(v) => setFormData({ ...formData, ml_model_id: v })}
							options={mlModelOptions}
						/>
					</div>
					<FormField
						label="Description"
						type="textarea"
						value={formData.description}
						onChange={(v) => setFormData({ ...formData, description: v })}
					/>
					<Button type="submit" label="Create Digital Twin" />
				</form>
			</Modal>

			<Modal open={showActionModal} onClose={() => setShowActionModal(false)} title="Create New Action">
				<form onSubmit={handleCreateAction}>
					<div className="form-grid">
						<FormField
							label="Action Name"
							value={actionForm.name}
							onChange={(v) => setActionForm({ ...actionForm, name: v })}
							required
						/>
						<FormField
							label="Action Type"
							value={actionForm.action_type}
							onChange={(v) => setActionForm({ ...actionForm, action_type: v })}
							required
						/>
					</div>
					<FormField
						label="Parameters (JSON)"
						type="textarea"
						value={actionForm.parameters}
						onChange={(v) => setActionForm({ ...actionForm, parameters: v })}
						placeholder='{"key": "value"}'
					/>
					<Button type="submit" label="Create Action" />
				</form>
			</Modal>

			<Modal open={showQueryModal} onClose={() => setShowQueryModal(false)} title="Query Digital Twin">
				<form onSubmit={handleQuery}>
					<FormField
						label="Query"
						type="textarea"
						value={queryForm.query}
						onChange={(v) => setQueryForm({ ...queryForm, query: v })}
						placeholder="Enter your query..."
						required
					/>
					<Button type="submit" label="Execute Query" />
				</form>
				{queryResult && (
					<div style={{ marginTop: '16px' }}>
						<h3 style={{ color: 'var(--accent)' }}>Query Result:</h3>
						<div className="json-display">
							<pre>{JSON.stringify(queryResult, null, 2)}</pre>
						</div>
					</div>
				)}
			</Modal>
		</div>
	);
}

// ============================================
// STORAGE PAGE
// ============================================

function StoragePage() {
	const { activeProject, projects } = React.useContext(ProjectContext);
	const [configs, setConfigs] = React.useState([]);
	const [loading, setLoading] = React.useState(true);
	const [showModal, setShowModal] = React.useState(false);
	const [healthStatus, setHealthStatus] = React.useState({});
	const [formData, setFormData] = React.useState({
		name: '',
		project_id: '',
		storage_type: '',
		connection_string: '',
		config: '{}',
	});

	// Default project_id when activeProject changes
	React.useEffect(() => {
		if (activeProject) {
			setFormData(prev => ({ ...prev, project_id: activeProject.id }));
		}
	}, [activeProject]);

	const loadConfigs = async () => {
		setLoading(true);
		try {
			const projectId = activeProject?.id || '';
			const data = await apiCall(`/api/storage/configs?project_id=${projectId}`);
			setConfigs(data || []);
		} catch (error) {
			console.error('Failed to load storage configs:', error);
			setConfigs([]);
		}
		setLoading(false);
	};

	React.useEffect(() => {
		loadConfigs();
	}, [activeProject]);

	const handleSubmit = async (e) => {
		e.preventDefault();
		try {
			const data = {
				...formData,
				config: JSON.parse(formData.config),
			};
			await apiCall('/api/storage/configs', {
				method: 'POST',
				body: JSON.stringify(data),
			});
			setShowModal(false);
			setFormData({ name: '', project_id: activeProject?.id || '', storage_type: '', connection_string: '', config: '{}' });
			loadConfigs();
		} catch (error) {
			alert('Failed to create storage config: ' + error.message);
		}
	};

	const handleDelete = async (id) => {
		if (!confirm('Delete this storage config?')) return;
		try {
			await apiCall(`/api/storage/configs/${id}`, { method: 'DELETE' });
			loadConfigs();
		} catch (error) {
			alert('Failed to delete storage config: ' + error.message);
		}
	};

	const handleCheckHealth = async (id) => {
		setHealthStatus(prev => ({ ...prev, [id]: 'checking' }));
		try {
			const result = await apiCall(`/api/storage/health?config_id=${id}`);
			setHealthStatus(prev => ({ ...prev, [id]: result.healthy ? 'ok' : 'error' }));
		} catch {
			setHealthStatus(prev => ({ ...prev, [id]: 'error' }));
		}
		setTimeout(() => setHealthStatus(prev => {
			const n = { ...prev };
			delete n[id];
			return n;
		}), 8000);
	};

	const projectOptions = projects.map(p => ({ value: p.id, label: p.name }));

	const columns = [
		{ key: 'id', label: 'ID' },
		{ key: 'name', label: 'Name' },
		{ key: 'storage_type', label: 'Type' },
		{ key: 'project_id', label: 'Project ID' },
		{
			key: 'status',
			label: 'Status',
			render: (row) => <span className={`status-badge status-${row.status || 'active'}`}>{row.status || 'active'}</span>
		},
		{ key: 'created_at', label: 'Created', render: (row) => new Date(row.created_at).toLocaleDateString() },
	];

	return (
		<div className="content-section">
			<div className="section-header">
				<h2>Storage Configurations</h2>
				<Button label="+ New Storage Config" onClick={() => setShowModal(true)} />
			</div>

			{loading ? (
				<div className="loading">Loading storage configurations...</div>
			) : (
				<Table
					columns={columns}
					data={configs}
					actions={(row) => (
						<>
							{healthStatus[row.id] === 'checking' && <span style={{ color: 'var(--accent)' }}>Checking…</span>}
							{healthStatus[row.id] === 'ok' && <span style={{ color: '#22c55e' }}>✓ Connected</span>}
							{healthStatus[row.id] === 'error' && <span style={{ color: '#ef4444' }}>✗ Failed</span>}
							<Button label="Test Connection" onClick={() => handleCheckHealth(row.id)} variant="secondary" />
							<Button label="Delete" onClick={() => handleDelete(row.id)} variant="danger" />
						</>
					)}
				/>
			)}

			<Modal open={showModal} onClose={() => setShowModal(false)} title="Create Storage Configuration">
				<form onSubmit={handleSubmit}>
					<div className="form-grid">
						<FormField
							label="Config Name"
							value={formData.name}
							onChange={(v) => setFormData({ ...formData, name: v })}
							required
						/>
						<FormField
							label="Project"
							type="select"
							value={formData.project_id}
							onChange={(v) => setFormData({ ...formData, project_id: v })}
							options={projectOptions}
							required
						/>
					</div>
					<div className="form-grid">
						<FormField
							label="Storage Type"
							type="select"
							value={formData.storage_type}
							onChange={(v) => setFormData({ ...formData, storage_type: v })}
							options={['s3', 'gcs', 'azure', 'postgres', 'mongodb']}
							required
						/>
						<FormField
							label="Connection String"
							value={formData.connection_string}
							onChange={(v) => setFormData({ ...formData, connection_string: v })}
							placeholder="storage://..."
							required
						/>
					</div>
					<FormField
						label="Configuration (JSON)"
						type="textarea"
						value={formData.config}
						onChange={(v) => setFormData({ ...formData, config: v })}
						placeholder='{"region": "us-east-1"}'
					/>
					<Button type="submit" label="Create Storage Config" />
				</form>
			</Modal>
		</div>
	);
}

// ============================================
// PLUGINS PAGE
// ============================================

function PluginsPage() {
	const [plugins, setPlugins] = React.useState([]);
	const [loading, setLoading] = React.useState(true);
	const [showModal, setShowModal] = React.useState(false);
	const [formData, setFormData] = React.useState({
		repository_url: '',
		git_ref: 'main',
	});

	const loadPlugins = async () => {
		setLoading(true);
		try {
			const data = await apiCall('/api/plugins');
			setPlugins(data || []);
		} catch (error) {
			console.error('Failed to load plugins:', error);
			setPlugins([]);
		}
		setLoading(false);
	};

	React.useEffect(() => {
		loadPlugins();
	}, []);

	const handleSubmit = async (e) => {
		e.preventDefault();
		try {
			await apiCall('/api/plugins', {
				method: 'POST',
				body: JSON.stringify(formData),
			});
			setShowModal(false);
			setFormData({ repository_url: '', git_ref: 'main' });
			loadPlugins();
		} catch (error) {
			alert('Failed to install plugin: ' + error.message);
		}
	};

	const handleDelete = async (name) => {
		if (!confirm(`Uninstall plugin "${name}"?`)) return;
		try {
			await apiCall(`/api/plugins/${name}`, { method: 'DELETE' });
			loadPlugins();
		} catch (error) {
			alert('Failed to uninstall plugin: ' + error.message);
		}
	};

	const handleUpdate = async (name) => {
		try {
			await apiCall(`/api/plugins/${name}`, { method: 'PUT', body: JSON.stringify({}) });
			alert('Plugin updated!');
			loadPlugins();
		} catch (error) {
			alert('Failed to update plugin: ' + error.message);
		}
	};

	const columns = [
		{ key: 'name', label: 'Name' },
		{ key: 'version', label: 'Version' },
		{ key: 'description', label: 'Description' },
		{ key: 'author', label: 'Author' },
		{
			key: 'status',
			label: 'Status',
			render: (row) => <span className={`status-badge status-${row.status || 'active'}`}>{row.status || 'installed'}</span>
		},
	];

	return (
		<div className="content-section">
			<div className="section-header">
				<h2>Plugin Management</h2>
				<Button label="+ Install Plugin" onClick={() => setShowModal(true)} />
			</div>

			{loading ? (
				<div className="loading">Loading plugins...</div>
			) : (
				<Table
					columns={columns}
					data={plugins}
					actions={(row) => (
						<>
							<Button label="Update" onClick={() => handleUpdate(row.name)} variant="secondary" />
							<Button label="Uninstall" onClick={() => handleDelete(row.name)} variant="danger" />
						</>
					)}
				/>
			)}

			<Modal open={showModal} onClose={() => setShowModal(false)} title="Install Plugin">
				<form onSubmit={handleSubmit}>
					<FormField
						label="Git Repository URL"
						value={formData.repository_url}
						onChange={(v) => setFormData({ ...formData, repository_url: v })}
						placeholder="https://github.com/user/plugin.git"
						required
					/>
					<FormField
						label="Version/Branch"
						value={formData.git_ref}
						onChange={(v) => setFormData({ ...formData, git_ref: v })}
						placeholder="main"
					/>
					<Button type="submit" label="Install Plugin" />
				</form>
			</Modal>
		</div>
	);
}

// ============================================
// WORK TASKS PAGE
// ============================================

function useTaskWebSocket(onTaskUpdate) {
	React.useEffect(() => {
		const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
		const wsUrl = `${proto}//${window.location.host}/ws/tasks`;
		let ws;
		let reconnectTimer;

		function connect() {
			try {
				ws = new WebSocket(wsUrl);
				ws.onmessage = (event) => {
					try {
						const msg = JSON.parse(event.data);
						if (msg.event === 'task_update' && msg.task) {
							onTaskUpdate(msg.task);
						}
					} catch (e) {
						// ignore parse errors
					}
				};
				ws.onclose = () => {
					reconnectTimer = setTimeout(connect, 3000);
				};
				ws.onerror = () => {
					ws.close();
				};
			} catch (e) {
				reconnectTimer = setTimeout(connect, 3000);
			}
		}

		connect();
		return () => {
			clearTimeout(reconnectTimer);
			if (ws) ws.close();
		};
	}, []);
}

function WorkTasksPage() {
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

	// Replace polling with WebSocket for real-time updates
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
}

// ============================================
// MAIN APP
// ============================================

function App() {
	const [currentPage, setCurrentPage] = React.useState('Projects');
	const [sidebarOpen, setSidebarOpen] = React.useState(false);
	const [projects, setProjects] = React.useState([]);
	const [activeProject, setActiveProject] = React.useState(null);

	const pages = ['Projects', 'Pipelines', 'Ontologies', 'ML Models', 'Digital Twins', 'Storage', 'Plugins', 'Work Queue'];

	// Fetch projects on mount and set first project as active
	React.useEffect(() => {
		apiCall('/api/projects')
			.then(data => {
				const list = data || [];
				setProjects(list);
				if (list.length > 0 && !activeProject) {
					setActiveProject(list[0]);
				}
			})
			.catch(() => {});
	}, []);

	const navigate = (page) => {
		setCurrentPage(page);
		setSidebarOpen(false);
	};

	const renderPage = () => {
		switch (currentPage) {
			case 'Projects':
				return <ProjectsPage />;
			case 'Pipelines':
				return <PipelinesPage />;
			case 'Ontologies':
				return <OntologiesPage />;
			case 'ML Models':
				return <MLModelsPage />;
			case 'Digital Twins':
				return <DigitalTwinsPage />;
			case 'Storage':
				return <StoragePage />;
			case 'Plugins':
				return <PluginsPage />;
			case 'Work Queue':
				return <WorkTasksPage />;
			default:
				return <ProjectsPage />;
		}
	};

	return (
		<div className="app-shell">
			<header className="app-topbar">
				<div className="topbar-brand">
					<button
						className="hamburger"
						onClick={() => setSidebarOpen(o => !o)}
						aria-label="Toggle navigation"
					>
						<span/><span/><span/>
					</button>
					<span className="topbar-logo">◆</span>
					<span className="topbar-name">Mimir AIP</span>
				</div>
				<div className="topbar-meta">
					<span className="topbar-version">v0.1.0</span>
				</div>
			</header>
			<div className="app-body">
				{sidebarOpen && (
					<div className="sidebar-overlay" onClick={() => setSidebarOpen(false)} />
				)}
				<aside className={`app-sidebar${sidebarOpen ? ' is-open' : ''}`}>
					<nav className="sidebar-nav">
						{pages.map(page => (
							<button
								key={page}
								className={`nav-item${currentPage === page ? ' active' : ''}`}
								onClick={() => navigate(page)}
							>
								{page}
							</button>
						))}
						<div className="sidebar-project-selector">
							<label>Working Project</label>
							<select
								value={activeProject?.id || ''}
								onChange={e => setActiveProject(projects.find(p => p.id === e.target.value) || null)}
							>
								<option value="">— All Projects —</option>
								{projects.map(p => (
									<option key={p.id} value={p.id}>{p.name}</option>
								))}
							</select>
						</div>
					</nav>
				</aside>
				<main className="app-main">
					<div className="app-container">
						<ProjectContext.Provider value={{ activeProject, projects }}>
							{renderPage()}
						</ProjectContext.Provider>
					</div>
				</main>
			</div>
		</div>
	);
}

// Render the app
const root = ReactDOM.createRoot(document.getElementById('root'));
root.render(<App />);

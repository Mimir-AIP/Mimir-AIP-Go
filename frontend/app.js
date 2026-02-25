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
					className="tab"
					style={{
						fontWeight: tab === activeTab ? 'bold' : 'normal',
						background: tab === activeTab ? 'var(--accent)' : 'var(--background)',
						color: 'var(--text)',
						fontFamily: 'var(--font-family)',
						border: tab === activeTab ? 'none' : '1px solid var(--accent)',
						padding: '8px 16px',
						cursor: 'pointer',
					}}
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
								<div style={{ display: 'flex', gap: '8px' }}>
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

		return await response.json();
	} catch (error) {
		console.error('API Error:', error);
		throw error;
	}
}

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
		owner: '',
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
			setFormData({ name: '', description: '', owner: '' });
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
		{ key: 'owner', label: 'Owner' },
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
					<div className="form-grid">
						<FormField
							label="Project Name"
							value={formData.name}
							onChange={(v) => setFormData({ ...formData, name: v })}
							required
						/>
						<FormField
							label="Owner"
							value={formData.owner}
							onChange={(v) => setFormData({ ...formData, owner: v })}
							required
						/>
					</div>
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
// PIPELINES PAGE
// ============================================

function PipelinesPage() {
	const [pipelines, setPipelines] = React.useState([]);
	const [schedules, setSchedules] = React.useState([]);
	const [loading, setLoading] = React.useState(true);
	const [showPipelineModal, setShowPipelineModal] = React.useState(false);
	const [showScheduleModal, setShowScheduleModal] = React.useState(false);
	const [selectedTab, setSelectedTab] = React.useState('Pipelines');
	const [pipelineForm, setPipelineForm] = React.useState({
		name: '',
		description: '',
		project_id: '',
		steps: '[]',
	});
	const [scheduleForm, setScheduleForm] = React.useState({
		name: '',
		pipeline_id: '',
		project_id: '',
		cron_expression: '',
		enabled: true,
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
			setPipelineForm({ name: '', description: '', project_id: '', steps: '[]' });
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
			setScheduleForm({ name: '', pipeline_id: '', project_id: '', cron_expression: '', enabled: true });
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
		try {
			await apiCall(`/api/pipelines/${id}/execute`, { method: 'POST', body: JSON.stringify({}) });
			alert('Pipeline execution started!');
		} catch (error) {
			alert('Failed to execute pipeline: ' + error.message);
		}
	};

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
							<Button label="Execute" onClick={() => handleExecutePipeline(row.id)} variant="secondary" />
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
							label="Project ID"
							value={pipelineForm.project_id}
							onChange={(v) => setPipelineForm({ ...pipelineForm, project_id: v })}
							required
						/>
					</div>
					<FormField
						label="Description"
						type="textarea"
						value={pipelineForm.description}
						onChange={(v) => setPipelineForm({ ...pipelineForm, description: v })}
					/>
					<FormField
						label="Steps (JSON Array)"
						type="textarea"
						value={pipelineForm.steps}
						onChange={(v) => setPipelineForm({ ...pipelineForm, steps: v })}
						placeholder='[{"type": "extraction", "config": {}}]'
						required
					/>
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
							label="Pipeline ID"
							value={scheduleForm.pipeline_id}
							onChange={(v) => setScheduleForm({ ...scheduleForm, pipeline_id: v })}
							required
						/>
					</div>
					<div className="form-grid">
						<FormField
							label="Project ID"
							value={scheduleForm.project_id}
							onChange={(v) => setScheduleForm({ ...scheduleForm, project_id: v })}
							required
						/>
						<FormField
							label="Cron Expression"
							value={scheduleForm.cron_expression}
							onChange={(v) => setScheduleForm({ ...scheduleForm, cron_expression: v })}
							placeholder="0 0 * * *"
							required
						/>
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
	const [ontologies, setOntologies] = React.useState([]);
	const [loading, setLoading] = React.useState(true);
	const [showModal, setShowModal] = React.useState(false);
	const [showExtractionModal, setShowExtractionModal] = React.useState(false);
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

	const loadOntologies = async () => {
		setLoading(true);
		try {
			// Note: API requires project_id parameter, but we'll try to get all
			const data = await apiCall('/api/ontologies?project_id=');
			setOntologies(data || []);
		} catch (error) {
			console.error('Failed to load ontologies:', error);
			setOntologies([]);
		}
		setLoading(false);
	};

	React.useEffect(() => {
		loadOntologies();
	}, []);

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
			setFormData({ name: '', project_id: '', schema: '{}' });
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

	const handleExtraction = async (e) => {
		e.preventDefault();
		try {
			const result = await apiCall('/api/extraction/generate-ontology', {
				method: 'POST',
				body: JSON.stringify(extractionForm),
			});
			alert('Ontology extraction started!\n' + JSON.stringify(result, null, 2));
			setShowExtractionModal(false);
			setExtractionForm({ project_id: '', storage_config_id: '', ontology_name: '' });
			setTimeout(loadOntologies, 2000);
		} catch (error) {
			alert('Failed to start extraction: ' + error.message);
		}
	};

	const columns = [
		{ key: 'id', label: 'ID' },
		{ key: 'name', label: 'Name' },
		{ key: 'project_id', label: 'Project ID' },
		{ key: 'version', label: 'Version' },
		{ key: 'created_at', label: 'Created', render: (row) => new Date(row.created_at).toLocaleDateString() },
	];

	return (
		<div className="content-section">
			<div className="section-header">
				<h2>Ontologies</h2>
				<div style={{ display: 'flex', gap: '8px' }}>
					<Button label="+ New Ontology" onClick={() => setShowModal(true)} />
					<Button label="Extract from Storage" onClick={() => setShowExtractionModal(true)} variant="secondary" />
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
							label="Project ID"
							value={formData.project_id}
							onChange={(v) => setFormData({ ...formData, project_id: v })}
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
							label="Project ID"
							value={extractionForm.project_id}
							onChange={(v) => setExtractionForm({ ...extractionForm, project_id: v })}
							required
						/>
						<FormField
							label="Storage Config ID"
							value={extractionForm.storage_config_id}
							onChange={(v) => setExtractionForm({ ...extractionForm, storage_config_id: v })}
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
	const [models, setModels] = React.useState([]);
	const [loading, setLoading] = React.useState(true);
	const [showModal, setShowModal] = React.useState(false);
	const [formData, setFormData] = React.useState({
		name: '',
		project_id: '',
		model_type: '',
		version: '1.0.0',
		config: '{}',
	});

	const loadModels = async () => {
		setLoading(true);
		try {
			const data = await apiCall('/api/ml-models?project_id=');
			setModels(data || []);
		} catch (error) {
			console.error('Failed to load ML models:', error);
			setModels([]);
		}
		setLoading(false);
	};

	React.useEffect(() => {
		loadModels();
	}, []);

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
			setFormData({ name: '', project_id: '', model_type: '', version: '1.0.0', config: '{}' });
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

	const handleTrain = async (id) => {
		try {
			await apiCall('/api/ml-models/train', {
				method: 'POST',
				body: JSON.stringify({ model_id: id }),
			});
			alert('Training started!');
		} catch (error) {
			alert('Failed to start training: ' + error.message);
		}
	};

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
				<Button label="+ New Model" onClick={() => setShowModal(true)} />
			</div>

			{loading ? (
				<div className="loading">Loading ML models...</div>
			) : (
				<Table
					columns={columns}
					data={models}
					actions={(row) => (
						<>
							<Button label="Train" onClick={() => handleTrain(row.id)} variant="secondary" />
							<Button label="Delete" onClick={() => handleDelete(row.id)} variant="danger" />
						</>
					)}
				/>
			)}

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
							label="Project ID"
							value={formData.project_id}
							onChange={(v) => setFormData({ ...formData, project_id: v })}
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

	const loadTwins = async () => {
		setLoading(true);
		try {
			const data = await apiCall('/api/digital-twins?project_id=');
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
	}, []);

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
			setFormData({ name: '', project_id: '', ontology_id: '', ml_model_id: '', description: '' });
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
							label="Project ID"
							value={formData.project_id}
							onChange={(v) => setFormData({ ...formData, project_id: v })}
							required
						/>
					</div>
					<div className="form-grid">
						<FormField
							label="Ontology ID"
							value={formData.ontology_id}
							onChange={(v) => setFormData({ ...formData, ontology_id: v })}
							required
						/>
						<FormField
							label="ML Model ID"
							value={formData.ml_model_id}
							onChange={(v) => setFormData({ ...formData, ml_model_id: v })}
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
	const [configs, setConfigs] = React.useState([]);
	const [loading, setLoading] = React.useState(true);
	const [showModal, setShowModal] = React.useState(false);
	const [formData, setFormData] = React.useState({
		name: '',
		project_id: '',
		storage_type: '',
		connection_string: '',
		config: '{}',
	});

	const loadConfigs = async () => {
		setLoading(true);
		try {
			const data = await apiCall('/api/storage/configs?project_id=');
			setConfigs(data || []);
		} catch (error) {
			console.error('Failed to load storage configs:', error);
			setConfigs([]);
		}
		setLoading(false);
	};

	React.useEffect(() => {
		loadConfigs();
	}, []);

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
			setFormData({ name: '', project_id: '', storage_type: '', connection_string: '', config: '{}' });
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
		try {
			const result = await apiCall(`/api/storage/${id}/health`);
			alert(`Health Status: ${result.status || 'Unknown'}\n${JSON.stringify(result, null, 2)}`);
		} catch (error) {
			alert('Failed to check health: ' + error.message);
		}
	};

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
							<Button label="Health" onClick={() => handleCheckHealth(row.id)} variant="secondary" />
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
							label="Project ID"
							value={formData.project_id}
							onChange={(v) => setFormData({ ...formData, project_id: v })}
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
		git_url: '',
		version: 'main',
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
			setFormData({ git_url: '', version: 'main' });
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
			await apiCall(`/api/plugins/${name}`, { method: 'PUT' });
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
						value={formData.git_url}
						onChange={(v) => setFormData({ ...formData, git_url: v })}
						placeholder="https://github.com/user/plugin.git"
						required
					/>
					<FormField
						label="Version/Branch"
						value={formData.version}
						onChange={(v) => setFormData({ ...formData, version: v })}
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

	React.useEffect(() => {
		loadTasks();
		const interval = setInterval(loadTasks, 5000); // Refresh every 5 seconds
		return () => clearInterval(interval);
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

	const pages = ['Projects', 'Pipelines', 'Ontologies', 'ML Models', 'Digital Twins', 'Storage', 'Plugins', 'Work Queue'];

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
		<div className="app-container">
			<div className="app-header">
				<h1>Mimir AIP Orchestrator</h1>
			</div>

			<Tabs
				tabs={pages}
				activeTab={currentPage}
				onTabChange={setCurrentPage}
			/>

			{renderPage()}
		</div>
	);
}

// Render the app
const root = ReactDOM.createRoot(document.getElementById('root'));
root.render(<App />);

(() => {
	const root = window.MimirApp = window.MimirApp || {};
	const pipelines = (((root.components = root.components || {}).pipelines = root.components.pipelines || {}));
	const { apiCall } = root.lib;
	const { Button, FormField } = root.components.primitives;
	const { StepParamFields } = pipelines;

	pipelines.StepBuilder = function StepBuilder({ value, onChange }) {
		const [mode, setMode] = React.useState('visual');
		const [steps, setSteps] = React.useState(() => {
			try { return JSON.parse(value) || []; } catch { return []; }
		});
		const [plugins, setPlugins] = React.useState([]);

		React.useEffect(() => {
			apiCall('/api/plugins').then(data => setPlugins(data || [])).catch(() => {});
		}, []);

		React.useEffect(() => {
			if (mode === 'visual') onChange(JSON.stringify(steps, null, 2));
		}, [steps, mode]);

		const addStep = () => setSteps(prev => [...prev, { name: '', plugin: 'default', action: '', parameters: {} }]);
		const removeStep = (i) => setSteps(prev => prev.filter((_, idx) => idx !== i));
		const updateStep = (i, field, val) => setSteps(prev => prev.map((s, idx) => idx === i ? { ...s, [field]: val } : s));
		const builtinActions = ['http_request', 'poll_http_json', 'poll_rss', 'poll_sql_incremental', 'poll_csv_drop', 'ingest_csv', 'ingest_csv_url', 'query_sql', 'load_checkpoint', 'save_checkpoint', 'parse_json', 'if_else', 'set_context', 'get_context', 'goto', 'store_cir', 'store_cir_batch', 'send_email', 'send_webhook'];

		const getActions = (pluginName) => {
			if (pluginName === 'default' || pluginName === 'builtin') return builtinActions;
			const plugin = plugins.find(p => p.name === pluginName);
			return (plugin?.plugin_definition?.actions || []).map(a => a.name || a);
		};

		const getParamSchema = (pluginName, actionName) => {
			if (!actionName) return null;
			if (pluginName === 'default' || pluginName === 'builtin') return null;
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
	};
})();

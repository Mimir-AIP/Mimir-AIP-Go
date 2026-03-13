(() => {
	const root = window.MimirApp = window.MimirApp || {};
	const pipelines = (((root.components = root.components || {}).pipelines = root.components.pipelines || {}));

	pipelines.StepParamFields = function StepParamFields({ paramSchemas, parameters, onUpdate }) {
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
									<input type="checkbox" checked={!!v} onChange={e => setParam(p.name, e.target.checked)} />
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
	};
})();

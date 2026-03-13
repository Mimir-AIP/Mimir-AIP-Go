(() => {
	const root = window.MimirApp = window.MimirApp || {};
	const pipelines = (((root.components = root.components || {}).pipelines = root.components.pipelines || {}));

	const CRON_PRESETS = [
		{ label: 'Every minute', cron: '* * * * *' },
		{ label: 'Every hour', cron: '0 * * * *' },
		{ label: 'Every day midnight', cron: '0 0 * * *' },
		{ label: 'Every Monday', cron: '0 0 * * 1' },
		{ label: 'Every month (1st)', cron: '0 0 1 * *' },
	];

	pipelines.CronBuilder = function CronBuilder({ value, onChange }) {
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
	};
})();

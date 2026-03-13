(() => {
	const root = window.MimirApp = window.MimirApp || {};
	const connectors = (((root.components = root.components || {}).connectors = root.components.connectors || {}));
	const { FormField } = root.components.primitives;

	connectors.ConnectorFieldInput = function ConnectorFieldInput({ field, value, onChange }) {
		const inputValue = value ?? (field.type === 'boolean' ? false : '');
		if (field.type === 'boolean') {
			return (
				<div className="form-group">
					<label style={{ display: 'flex', alignItems: 'center', gap: '8px', cursor: 'pointer' }}>
						<input type="checkbox" checked={!!inputValue} onChange={e => onChange(e.target.checked)} />
						<span>{field.label}{field.required ? ' *' : ''}</span>
					</label>
					{field.description && <div style={{ color: 'var(--text-secondary)', fontSize: '0.85rem' }}>{field.description}</div>}
				</div>
			);
		}

		if (field.type === 'select') {
			return (
				<FormField
					label={field.label}
					type="select"
					value={inputValue}
					onChange={onChange}
					options={(field.options || []).map(opt => ({ value: opt.value, label: opt.label }))}
					required={field.required}
				/>
			);
		}

		if (field.type === 'number') {
			return (
				<div className="form-group">
					<label>{field.label}{field.required ? ' *' : ''}</label>
					<input
						type="number"
						value={inputValue}
						onChange={e => onChange(e.target.value === '' ? '' : Number(e.target.value))}
						required={field.required}
						style={{ width: '100%', padding: '8px', background: 'var(--surface)', color: 'var(--text)', border: '1px solid var(--border)', borderRadius: '4px' }}
					/>
					{field.description && <div style={{ color: 'var(--text-secondary)', fontSize: '0.85rem' }}>{field.description}</div>}
				</div>
			);
		}

		return (
			<FormField
				label={field.label}
				value={inputValue}
				onChange={onChange}
				placeholder={field.description || ''}
				required={field.required}
			/>
		);
	};
})();

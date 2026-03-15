(() => {
	const root = window.MimirApp = window.MimirApp || {};
	const primitives = (((root.components = root.components || {}).primitives = root.components.primitives || {}));

	primitives.FormField = function FormField({
		label,
		type = 'text',
		value,
		onChange,
		options,
		placeholder,
		required,
		hint,
		id,
		...rest
	}) {
		const reactId = React.useId();
		const fieldId = id || `field-${reactId}`;
		const hintId = hint ? `${fieldId}-hint` : undefined;
		const labelText = `${label}${required ? ' *' : ''}`;

		return (
			<div className="form-group">
				<label htmlFor={fieldId}>{labelText}</label>
				{type === 'select' ? (
					<select id={fieldId} value={value} onChange={e => onChange(e.target.value)} required={required} aria-describedby={hintId} {...rest}>
						<option value="">Select...</option>
						{options?.map(opt => {
							const option = typeof opt === 'string' ? { value: opt, label: opt } : opt;
							return <option key={option.value} value={option.value}>{option.label}</option>;
						})}
					</select>
				) : type === 'textarea' ? (
					<textarea
						id={fieldId}
						value={value}
						onChange={e => onChange(e.target.value)}
						placeholder={placeholder}
						required={required}
						rows="4"
						aria-describedby={hintId}
						{...rest}
					/>
				) : (
					<input
						id={fieldId}
						type={type}
						value={value}
						onChange={e => onChange(e.target.value)}
						placeholder={placeholder}
						required={required}
						aria-describedby={hintId}
						{...rest}
					/>
				)}
				{hint ? <div id={hintId} className="form-hint">{hint}</div> : null}
			</div>
		);
	};
})();

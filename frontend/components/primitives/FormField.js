(() => {
	const root = window.MimirApp = window.MimirApp || {};
	const primitives = (((root.components = root.components || {}).primitives = root.components.primitives || {}));

	primitives.FormField = function FormField({ label, type = 'text', value, onChange, options, placeholder, required }) {
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
	};
})();

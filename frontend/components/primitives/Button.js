(() => {
	const root = window.MimirApp = window.MimirApp || {};
	const primitives = (((root.components = root.components || {}).primitives = root.components.primitives || {}));

	primitives.Button = React.memo(function Button({
		label,
		onClick,
		type = 'button',
		variant = 'primary',
		disabled = false,
		className = '',
		children,
		...rest
	}) {
		const variantClass = variant === 'secondary' ? 'secondary' : variant === 'danger' ? 'danger' : '';
		return (
			<button
				type={type}
				className={[variantClass, className].filter(Boolean).join(' ')}
				onClick={onClick}
				disabled={disabled}
				{...rest}
			>
				{children || label}
			</button>
		);
	});
})();

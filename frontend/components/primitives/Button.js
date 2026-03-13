(() => {
	const root = window.MimirApp = window.MimirApp || {};
	const primitives = (((root.components = root.components || {}).primitives = root.components.primitives || {}));

	primitives.Button = function Button({ label, onClick, type = 'button', variant = 'primary', disabled }) {
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
	};
})();

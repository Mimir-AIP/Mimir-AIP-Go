(() => {
	const root = window.MimirApp = window.MimirApp || {};
	const primitives = (((root.components = root.components || {}).primitives = root.components.primitives || {}));
	const { Button } = primitives;

	primitives.Modal = function Modal({ open, onClose, title, children }) {
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
	};
})();

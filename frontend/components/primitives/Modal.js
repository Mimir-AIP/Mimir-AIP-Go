(() => {
	const root = window.MimirApp = window.MimirApp || {};
	const primitives = (((root.components = root.components || {}).primitives = root.components.primitives || {}));
	const { Button } = primitives;

	function getFocusableElements(container) {
		if (!container) return [];
		return Array.from(container.querySelectorAll('button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'))
			.filter((element) => !element.hasAttribute('disabled') && element.getAttribute('aria-hidden') !== 'true');
	}

	primitives.Modal = function Modal({ open, onClose, title, children, footer, hideDefaultFooter = false }) {
		const reactId = React.useId();
		const titleId = `modal-title-${reactId}`;
		const contentRef = React.useRef(null);

		React.useEffect(() => {
			if (!open) return undefined;
			const previousActiveElement = document.activeElement;
			const previousOverflow = document.body.style.overflow;
			document.body.style.overflow = 'hidden';
			const focusables = getFocusableElements(contentRef.current);
			(focusables[0] || contentRef.current)?.focus();

			const handleKeyDown = (event) => {
				if (event.key === 'Escape') {
					event.preventDefault();
					onClose?.();
					return;
				}
				if (event.key !== 'Tab') return;
				const items = getFocusableElements(contentRef.current);
				if (!items.length) {
					event.preventDefault();
					return;
				}
				const first = items[0];
				const last = items[items.length - 1];
				if (event.shiftKey && document.activeElement === first) {
					event.preventDefault();
					last.focus();
				} else if (!event.shiftKey && document.activeElement === last) {
					event.preventDefault();
					first.focus();
				}
			};

			document.addEventListener('keydown', handleKeyDown);
			return () => {
				document.removeEventListener('keydown', handleKeyDown);
				document.body.style.overflow = previousOverflow;
				if (previousActiveElement?.focus) previousActiveElement.focus();
			};
		}, [open, onClose]);

		if (!open) return null;

		return (
			<div className="modal-overlay" onClick={() => onClose?.()}>
				<div
					className="modal-content"
					ref={contentRef}
					onClick={e => e.stopPropagation()}
					role="dialog"
					aria-modal="true"
					aria-labelledby={title ? titleId : undefined}
					tabIndex={-1}
				>
					{title && (
						<div className="modal-header">
							<h2 id={titleId}>{title}</h2>
						</div>
					)}
					<div className="modal-body">{children}</div>
					{footer ? <div className="modal-actions">{footer}</div> : null}
					{!footer && !hideDefaultFooter ? (
						<div className="modal-actions">
							<Button label="Close" onClick={onClose} variant="secondary" />
						</div>
					) : null}
				</div>
			</div>
		);
	};
})();

(() => {
	const root = window.MimirApp = window.MimirApp || {};
	const lib = root.lib = root.lib || {};
	let confirmSequence = 0;

	function dispatch(name, detail) {
		window.dispatchEvent(new CustomEvent(name, { detail }));
	}

	lib.notify = function notify(input, defaults = {}) {
		const detail = typeof input === 'string'
			? { message: input, tone: defaults.tone || 'info', duration: defaults.duration }
			: { tone: 'info', ...defaults, ...(input || {}) };
		dispatch('mimir:notify', detail);
	};

	lib.confirmAction = function confirmAction(options = {}) {
		const id = `confirm-${++confirmSequence}`;
		return new Promise((resolve) => {
			const handleResult = (event) => {
				if (event.detail?.id !== id) return;
				window.removeEventListener('mimir:confirm-result', handleResult);
				resolve(Boolean(event.detail.confirmed));
			};
			window.addEventListener('mimir:confirm-result', handleResult);
			dispatch('mimir:confirm', {
				id,
				title: options.title || 'Confirm action',
				message: options.message || 'Are you sure you want to continue?',
				confirmLabel: options.confirmLabel || 'Confirm',
				cancelLabel: options.cancelLabel || 'Cancel',
				variant: options.variant || 'danger',
			});
		});
	};
})();

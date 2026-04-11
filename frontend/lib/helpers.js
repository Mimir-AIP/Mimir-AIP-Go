(() => {
	const root = window.MimirApp = window.MimirApp || {};
	const lib = root.lib = root.lib || {};


	function deriveStorageConfigLabel(config) {
		const details = config?.config || {};
		const candidate = details.path || details.table || details.bucket || details.url || details.database || details.container || details.topic;
		if (candidate) return `${config?.plugin_type || 'storage'}: ${candidate}`;
		return `${config?.plugin_type || 'storage'} · ${String(config?.id || 'new').slice(0, 8)}`;
	}

	function renderConfigPreview(value) {
		try {
			return JSON.stringify(value || {}, null, 2);
		} catch {
			return '{}';
		}
	}

	lib.deriveStorageConfigLabel = deriveStorageConfigLabel;
	lib.renderConfigPreview = renderConfigPreview;
})();

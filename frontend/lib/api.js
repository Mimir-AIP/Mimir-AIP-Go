(() => {
	const root = window.MimirApp = window.MimirApp || {};
	const lib = root.lib = root.lib || {};

	function normalizeBaseUrl(value) {
		if (!value || value === '/') return '';
		return String(value).replace(/\/$/, '');
	}

	function configuredApiBaseUrl() {
		const runtimeConfig = window.__MIMIR_RUNTIME_CONFIG__ || {};
		const metaBase = document.querySelector('meta[name="mimir-api-base"]')?.content || '';
		const storedBase = (() => {
			try {
				return window.localStorage.getItem('mimir:apiBaseUrl') || '';
			} catch {
				return '';
			}
		})();
		return normalizeBaseUrl(runtimeConfig.apiBaseUrl || metaBase || storedBase || '');
	}

	const API_URL = configuredApiBaseUrl();

	function buildUrl(endpoint) {
		if (/^https?:\/\//.test(endpoint)) return endpoint;
		return `${API_URL}${endpoint}`;
	}

	function resolveWebSocketUrl(pathname = '/ws/tasks') {
		const base = API_URL || window.location.origin;
		const url = new URL(base, window.location.origin);
		url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:';
		url.pathname = pathname;
		url.search = '';
		url.hash = '';
		return url.toString();
	}

	async function apiCall(endpoint, options = {}) {
		try {
			const response = await fetch(buildUrl(endpoint), {
				headers: {
					'Content-Type': 'application/json',
					...options.headers,
				},
				...options,
			});

			if (!response.ok) {
				const error = await response.text();
				throw new Error(error || `HTTP ${response.status}`);
			}

			if (response.status === 204) {
				return null;
			}

			const contentType = response.headers.get('content-type') || '';
			if (!contentType.includes('application/json')) {
				return await response.text();
			}

			return await response.json();
		} catch (error) {
			console.error('API Error:', error);
			throw error;
		}
	}

	lib.API_URL = API_URL;
	lib.WS_TASKS_URL = resolveWebSocketUrl('/ws/tasks');
	lib.apiCall = apiCall;
	lib.resolveWebSocketUrl = resolveWebSocketUrl;
})();

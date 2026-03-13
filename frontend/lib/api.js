(() => {
	const root = window.MimirApp = window.MimirApp || {};
	const lib = root.lib = root.lib || {};

	const API_URL = window.location.origin.includes('localhost')
		? 'http://localhost:8080'
		: '';

	async function apiCall(endpoint, options = {}) {
		try {
			const response = await fetch(`${API_URL}${endpoint}`, {
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
	lib.apiCall = apiCall;
})();

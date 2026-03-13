(() => {
	const root = window.MimirApp = window.MimirApp || {};
	const hooks = root.hooks = root.hooks || {};

	hooks.useTaskWebSocket = function useTaskWebSocket(onTaskUpdate) {
		React.useEffect(() => {
			const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
			const wsUrl = `${proto}//${window.location.host}/ws/tasks`;
			let ws;
			let reconnectTimer;

			function connect() {
				try {
					ws = new WebSocket(wsUrl);
					ws.onmessage = (event) => {
						try {
							const msg = JSON.parse(event.data);
							if (msg.event === 'task_update' && msg.task) {
								onTaskUpdate(msg.task);
							}
						} catch (e) {
							// ignore parse errors
						}
					};
					ws.onclose = () => {
						reconnectTimer = setTimeout(connect, 3000);
					};
					ws.onerror = () => {
						ws.close();
					};
				} catch (e) {
					reconnectTimer = setTimeout(connect, 3000);
				}
			}

			connect();
			return () => {
				clearTimeout(reconnectTimer);
				if (ws) ws.close();
			};
		}, []);
	};
})();

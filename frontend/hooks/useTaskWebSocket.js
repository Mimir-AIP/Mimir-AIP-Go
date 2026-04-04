(() => {
	const root = window.MimirApp = window.MimirApp || {};
	const hooks = root.hooks = root.hooks || {};

	hooks.useTaskWebSocket = function useTaskWebSocket(onTaskUpdate) {
		const callbackRef = React.useRef(onTaskUpdate);

		React.useEffect(() => {
			callbackRef.current = onTaskUpdate;
		}, [onTaskUpdate]);

		React.useEffect(() => {
			const wsUrl = root.lib?.WS_TASKS_URL || root.lib?.resolveWebSocketUrl?.('/ws/tasks');
			if (!wsUrl) return undefined;
			let ws = null;
			let reconnectTimer = null;
			let disposed = false;

			function scheduleReconnect() {
				if (disposed || reconnectTimer) return;
				reconnectTimer = window.setTimeout(() => {
					reconnectTimer = null;
					connect();
				}, 3000);
			}

			function connect() {
				if (disposed) return;
				try {
					ws = new WebSocket(wsUrl);
					ws.onmessage = (event) => {
						try {
							const msg = JSON.parse(event.data);
							if (msg.event === 'task_update' && msg.task && typeof callbackRef.current === 'function') {
								callbackRef.current(msg.task);
							}
						} catch {
							// Ignore malformed websocket payloads.
						}
					};
					ws.onclose = () => {
						ws = null;
						scheduleReconnect();
					};
					ws.onerror = () => {
						if (ws) ws.close();
					};
				} catch {
					scheduleReconnect();
				}
			}

			connect();
			return () => {
				disposed = true;
				if (reconnectTimer) window.clearTimeout(reconnectTimer);
				if (ws) ws.close();
			};
		}, []);
	};
})();

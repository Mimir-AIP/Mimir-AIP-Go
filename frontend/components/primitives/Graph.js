(() => {
	const root = window.MimirApp = window.MimirApp || {};
	const primitives = (((root.components = root.components || {}).primitives = root.components.primitives || {}));

	primitives.Graph = function Graph({ data, options, type = 'line' }) {
		const canvasRef = React.useRef(null);
		const chartRef = React.useRef(null);

		React.useEffect(() => {
			if (!canvasRef.current || !data) return;

			if (chartRef.current) {
				chartRef.current.destroy();
			}

			const ctx = canvasRef.current.getContext('2d');
			chartRef.current = new Chart(ctx, {
				type,
				data,
				options: {
					responsive: true,
					maintainAspectRatio: true,
					...options,
					plugins: {
						legend: {
							labels: {
								color: 'var(--text)',
							}
						},
						...options?.plugins,
					},
					scales: {
						x: {
							ticks: { color: 'var(--text)' },
							grid: { color: 'rgba(255, 153, 0, 0.1)' },
						},
						y: {
							ticks: { color: 'var(--text)' },
							grid: { color: 'rgba(255, 153, 0, 0.1)' },
						},
						...options?.scales,
					},
				},
			});

			return () => {
				if (chartRef.current) {
					chartRef.current.destroy();
				}
			};
		}, [data, options, type]);

		return (
			<div className="graph-container">
				<canvas ref={canvasRef}></canvas>
			</div>
		);
	};
})();

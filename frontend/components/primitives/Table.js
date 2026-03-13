(() => {
	const root = window.MimirApp = window.MimirApp || {};
	const primitives = (((root.components = root.components || {}).primitives = root.components.primitives || {}));

	primitives.Table = function Table({ columns, data, actions }) {
		if (!data || data.length === 0) {
			return <div className="empty-state">No data available</div>;
		}

		return (
			<table>
				<thead>
					<tr>
						{columns.map(col => (
							<th key={col.key || col}>{col.label || col}</th>
						))}
						{actions && <th>Actions</th>}
					</tr>
				</thead>
				<tbody>
					{data.map((row, i) => (
						<tr key={row.id || i}>
							{columns.map(col => {
								const key = col.key || col;
								const value = col.render ? col.render(row) : row[key];
								return <td key={key}>{value || '-'}</td>;
							})}
							{actions && (
								<td>
									<div style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
										{actions(row)}
									</div>
								</td>
							)}
						</tr>
					))}
				</tbody>
			</table>
		);
	};
})();

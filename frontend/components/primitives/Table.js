(() => {
	const root = window.MimirApp = window.MimirApp || {};
	const primitives = (((root.components = root.components || {}).primitives = root.components.primitives || {}));

	primitives.Table = React.memo(function Table({ columns, data, actions, caption, emptyState = 'No data available' }) {
		if (!data || data.length === 0) {
			return <div className="empty-state">{emptyState}</div>;
		}

		return (
			<div className="table-container">
				<table>
					{caption ? <caption className="sr-only">{caption}</caption> : null}
					<thead>
						<tr>
							{columns.map(col => (
								<th key={col.key || col} scope="col">{col.label || col}</th>
							))}
							{actions && <th scope="col">Actions</th>}
						</tr>
					</thead>
					<tbody>
						{data.map((row, i) => (
							<tr key={row.id || row.worktask_id || i}>
								{columns.map(col => {
									const key = col.key || col;
									const label = col.label || col;
									const value = col.render ? col.render(row) : row[key];
									const cellValue = value === undefined || value === null || value === '' ? '—' : value;
									const title = typeof cellValue === 'string' || typeof cellValue === 'number' ? String(cellValue) : undefined;
									return <td key={key} data-label={label} title={title}>{cellValue}</td>;
								})}
								{actions ? (
									<td data-label="Actions">
										<div className="table-action-group">{actions(row)}</div>
									</td>
								) : null}
							</tr>
						))}
					</tbody>
				</table>
			</div>
		);
	});
})();

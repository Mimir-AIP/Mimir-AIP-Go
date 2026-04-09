(() => {
	const root = window.MimirApp = window.MimirApp || {};
	const pages = root.pages = root.pages || {};
	const { apiCall, notify, confirmAction } = root.lib;
	const { Button } = root.components.primitives;

	pages.AdminSettingsPage = function AdminSettingsPage() {
		const [resetting, setResetting] = React.useState(false);

		const handleFactoryReset = async () => {
			const confirmed = await confirmAction({
				title: 'Factory reset Mimir',
				message: 'This deletes all persisted Mimir metadata, queued tasks, digital twin history, plugins, providers, and project configuration. External data stored in connected storage backends is not deleted. Continue?',
				confirmLabel: 'Factory reset',
				variant: 'danger',
			});
			if (!confirmed) return;

			setResetting(true);
			try {
				const result = await apiCall('/api/admin/settings/factory-reset', { method: 'POST', body: JSON.stringify({}) });
				notify({ tone: 'success', message: result?.message || 'Mimir metadata has been reset.' });
				window.setTimeout(() => window.location.reload(), 300);
			} catch (error) {
				notify({ tone: 'error', message: `Factory reset failed: ${error.message}` });
			} finally {
				setResetting(false);
			}
		};

		return (
			<div className="content-section">
				<div className="section-header">
					<h2>Admin Settings</h2>
				</div>

				<div className="page-notice page-notice--warning">
					<strong>Danger zone:</strong> these settings affect the whole Mimir instance, not just the active project.
				</div>

				<div className="card">
					<div className="card-header">
						<div>
							<div className="card-title">Factory Reset</div>
							<p className="section-panel-copy">Return Mimir to a clean metadata state by deleting all projects, pipelines, schedules, ontologies, ML models, digital twins, insights, automations, task history, and plugin/provider registrations.</p>
						</div>
					</div>
					<div className="json-display">
						<pre>{JSON.stringify({
							deletes: [
								'Projects and project settings',
								'Pipelines, schedules, work task history',
								'Ontologies, ML models, digital twins, sync history',
								'Installed plugin and external provider registrations'
							],
							preserves: [
								'External data already stored in connected databases, buckets, and other backends',
								'The running Mimir binary and deployment configuration'
							]
						}, null, 2)}</pre>
					</div>
					<div className="inline-actions" style={{ marginTop: '16px' }}>
						<Button label={resetting ? 'Resetting…' : 'Factory Reset Mimir'} onClick={handleFactoryReset} variant="danger" disabled={resetting} />
					</div>
				</div>
			</div>
		);
	};
})();

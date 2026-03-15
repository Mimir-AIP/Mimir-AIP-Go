(() => {
	const root = window.MimirApp = window.MimirApp || {};
	const connectors = (((root.components = root.components || {}).connectors = root.components.connectors || {}));
	const { apiCall, deriveStorageConfigLabel, notify } = root.lib;
	const { Button, FormField } = root.components.primitives;
	const { ConnectorFieldInput } = connectors;
	const { CronBuilder } = root.components.pipelines;

	connectors.GuidedOnboardingPanel = function GuidedOnboardingPanel({ project }) {
		const [loading, setLoading] = React.useState(false);
		const [loadError, setLoadError] = React.useState('');
		const [templates, setTemplates] = React.useState([]);
		const [storageConfigs, setStorageConfigs] = React.useState([]);
		const [selectedKind, setSelectedKind] = React.useState('');
		const [formData, setFormData] = React.useState({
			name: '',
			description: '',
			storage_id: '',
			source_config: {},
			create_schedule: false,
			schedule_name: '',
			cron_schedule: '0 * * * *',
			enabled: true,
		});

		const activeTemplate = templates.find(template => template.kind === selectedKind) || null;

		const loadGuidedOptions = React.useCallback(async () => {
			if (!project?.id) {
				setTemplates([]);
				setStorageConfigs([]);
				setSelectedKind('');
				setLoadError('');
				return;
			}
			setLoading(true);
			setLoadError('');
			try {
				const [connectorData, storageData] = await Promise.all([
					apiCall('/api/connectors'),
					apiCall(`/api/storage/configs?project_id=${project.id}`),
				]);
				const nextTemplates = connectorData || [];
				const nextStorageConfigs = storageData || [];
				setTemplates(nextTemplates);
				setStorageConfigs(nextStorageConfigs);
				setSelectedKind(prev => (prev && nextTemplates.some(template => template.kind === prev)) ? prev : (nextTemplates[0]?.kind || ''));
				setFormData(prev => ({
					...prev,
					storage_id: nextStorageConfigs.some(cfg => cfg.id === prev.storage_id) ? prev.storage_id : (nextStorageConfigs[0]?.id || ''),
				}));
			} catch (error) {
				setLoadError(error.message || 'Failed to load guided onboarding data.');
				setTemplates([]);
				setStorageConfigs([]);
			} finally {
				setLoading(false);
			}
		}, [project?.id]);

		React.useEffect(() => {
			loadGuidedOptions();
		}, [loadGuidedOptions]);

		React.useEffect(() => {
			if (!activeTemplate) return;
			setFormData(prev => {
				const nextSourceConfig = {};
				(activeTemplate.fields || []).forEach(field => {
					const defaultValue = field.default !== undefined ? field.default : (field.type === 'boolean' ? false : '');
					nextSourceConfig[field.name] = prev.source_config[field.name] !== undefined ? prev.source_config[field.name] : defaultValue;
				});
				return {
					...prev,
					source_config: nextSourceConfig,
					create_schedule: activeTemplate.supports_schedule ? prev.create_schedule : false,
				};
			});
		}, [activeTemplate?.kind]);

		const handleSubmit = async (e) => {
			e.preventDefault();
			if (!project?.id || !activeTemplate) return;
			try {
				const payload = {
					project_id: project.id,
					kind: activeTemplate.kind,
					name: formData.name,
					description: formData.description,
					storage_id: formData.storage_id,
					source_config: formData.source_config,
				};
				if (activeTemplate.supports_schedule && formData.create_schedule) {
					payload.schedule = {
						name: formData.schedule_name || `${formData.name} schedule`,
						cron_schedule: formData.cron_schedule,
						enabled: formData.enabled,
					};
				}
				const result = await apiCall('/api/connectors', { method: 'POST', body: JSON.stringify(payload) });
				notify({ tone: 'success', message: `Connector created: ${result?.pipeline?.name || formData.name}` });
				setFormData(prev => ({
					...prev,
					name: '',
					description: '',
					source_config: {},
					create_schedule: false,
					schedule_name: '',
					cron_schedule: '0 * * * *',
					enabled: true,
				}));
				await loadGuidedOptions();
			} catch (error) {
				notify({ tone: 'error', message: `Failed to create connector: ${error.message}` });
			}
		};

		if (!project?.id) {
			return <div className="empty-state">Choose a project to start guided onboarding.</div>;
		}
		if (loading) {
			return <div className="loading">Loading guided onboarding…</div>;
		}
		if (loadError) {
			return <div className="error-message">{loadError}</div>;
		}
		if (templates.length === 0) {
			return <div className="empty-state">No connector templates are available right now.</div>;
		}
		if (storageConfigs.length === 0) {
			return <div className="empty-state">Create a storage configuration first, then return here to materialize a connector.</div>;
		}

		return (
			<form onSubmit={handleSubmit}>
				<div className="form-grid">
					<FormField
						label="Connector Template"
						type="select"
						value={selectedKind}
						onChange={setSelectedKind}
						options={templates.map(template => ({ value: template.kind, label: `${template.label} (${template.category})` }))}
						required
					/>
					<FormField
						label="Destination Storage"
						type="select"
						value={formData.storage_id}
						onChange={v => setFormData({ ...formData, storage_id: v })}
						options={storageConfigs.map(cfg => ({ value: cfg.id, label: deriveStorageConfigLabel(cfg) }))}
						required
					/>
				</div>
				{activeTemplate ? (
					<div className="section-panel section-panel--neutral">
						<div className="section-panel-copy"><strong>{activeTemplate.label}</strong></div>
						<p className="section-panel-copy">{activeTemplate.description}</p>
						<p className="section-panel-copy">Creates a {activeTemplate.pipeline_type} pipeline{activeTemplate.supports_schedule ? ' with an optional recurring schedule.' : '.'}</p>
					</div>
				) : null}
				<div className="form-grid">
					<FormField label="Pipeline Name" value={formData.name} onChange={v => setFormData({ ...formData, name: v })} required />
					<FormField label="Description" value={formData.description} onChange={v => setFormData({ ...formData, description: v })} />
				</div>
				<div className="form-grid">
					{(activeTemplate?.fields || []).map(field => (
						<div key={field.name}>
							<ConnectorFieldInput
								field={field}
								value={formData.source_config[field.name]}
								onChange={value => setFormData(prev => ({ ...prev, source_config: { ...prev.source_config, [field.name]: value } }))}
							/>
						</div>
					))}
				</div>
				{activeTemplate?.supports_schedule ? (
					<div className="section-panel section-panel--neutral">
						<label className="checkbox-row">
							<input type="checkbox" checked={formData.create_schedule} onChange={e => setFormData({ ...formData, create_schedule: e.target.checked })} />
							Create a recurring schedule now
						</label>
						{formData.create_schedule ? (
							<>
								<div className="form-grid">
									<FormField label="Schedule Name" value={formData.schedule_name} onChange={v => setFormData({ ...formData, schedule_name: v })} placeholder={`${formData.name || activeTemplate.label} schedule`} />
									<div className="form-group">
										<label>Cron Schedule *</label>
										<CronBuilder value={formData.cron_schedule} onChange={v => setFormData({ ...formData, cron_schedule: v })} />
									</div>
								</div>
								<label className="checkbox-row">
									<input type="checkbox" checked={formData.enabled} onChange={e => setFormData({ ...formData, enabled: e.target.checked })} />
									Start enabled
								</label>
							</>
						) : null}
					</div>
				) : null}
				<div className="inline-actions">
					<Button type="submit" label="Create Connector Pipeline" />
				</div>
			</form>
		);
	};
})();

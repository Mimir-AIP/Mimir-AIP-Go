(() => {
	const root = window.MimirApp = window.MimirApp || {};
	const pages = root.pages = root.pages || {};
	const { apiCall, deriveStorageConfigLabel, renderConfigPreview } = root.lib;
	const { ProjectContext } = root.context;
	const { Button, FormField, Table } = root.components.primitives;

	pages.InsightsReviewPage = function InsightsReviewPage() {
		const { activeProject } = React.useContext(ProjectContext);
		const [storageConfigs, setStorageConfigs] = React.useState([]);
		const [selectedStorageIds, setSelectedStorageIds] = React.useState([]);
		const [insights, setInsights] = React.useState([]);
		const [reviews, setReviews] = React.useState([]);
		const [resolverMetrics, setResolverMetrics] = React.useState(null);
		const [severityFilter, setSeverityFilter] = React.useState('');
		const [minConfidence, setMinConfidence] = React.useState('');
		const [reviewStatus, setReviewStatus] = React.useState('pending');
		const [reviewer, setReviewer] = React.useState('');
		const [rationales, setRationales] = React.useState({});
		const [loading, setLoading] = React.useState(false);

		const loadStorageConfigs = React.useCallback(async () => {
			if (!activeProject?.id) {
				setStorageConfigs([]);
				setSelectedStorageIds([]);
				return;
			}
			const data = await apiCall(`/api/storage/configs?project_id=${activeProject.id}`);
			setStorageConfigs(data || []);
			setSelectedStorageIds(prev => (prev || []).filter(id => (data || []).some(cfg => cfg.id === id)));
		}, [activeProject?.id]);

		const loadInsights = React.useCallback(async () => {
			if (!activeProject?.id) {
				setInsights([]);
				return;
			}
			const params = new URLSearchParams({ project_id: activeProject.id });
			if (severityFilter) params.set('severity', severityFilter);
			if (minConfidence !== '') params.set('min_confidence', minConfidence);
			const data = await apiCall(`/api/insights?${params.toString()}`);
			setInsights(data || []);
		}, [activeProject?.id, severityFilter, minConfidence]);

		const loadReviews = React.useCallback(async () => {
			if (!activeProject?.id) {
				setReviews([]);
				return;
			}
			const params = new URLSearchParams({ project_id: activeProject.id });
			if (reviewStatus) params.set('status', reviewStatus);
			const data = await apiCall(`/api/reviews?${params.toString()}`);
			setReviews(data || []);
		}, [activeProject?.id, reviewStatus]);

		const loadMetrics = React.useCallback(async () => {
			if (!activeProject?.id) {
				setResolverMetrics(null);
				return;
			}
			const data = await apiCall(`/api/analysis/resolver/metrics?project_id=${activeProject.id}`);
			setResolverMetrics(data || null);
		}, [activeProject?.id]);

		React.useEffect(() => {
			if (!activeProject?.id) {
				setInsights([]);
				setReviews([]);
				setStorageConfigs([]);
				setResolverMetrics(null);
				setSelectedStorageIds([]);
				return;
			}
			setLoading(true);
			Promise.all([loadStorageConfigs(), loadInsights(), loadReviews(), loadMetrics()])
				.catch(error => console.error('Failed to load insights/review data:', error))
				.finally(() => setLoading(false));
		}, [activeProject?.id]);

		const handleGenerateInsights = async () => {
			if (!activeProject?.id) return;
			try {
				await apiCall('/api/insights', { method: 'POST', body: JSON.stringify({ project_id: activeProject.id }) });
				await loadInsights();
			} catch (error) {
				alert('Failed to generate insights: ' + error.message);
			}
		};

		const handleRunResolver = async () => {
			if (!activeProject?.id || selectedStorageIds.length < 2) return;
			try {
				const result = await apiCall('/api/analysis/resolver', {
					method: 'POST',
					body: JSON.stringify({ project_id: activeProject.id, storage_ids: selectedStorageIds }),
				});
				setResolverMetrics(result?.metrics || null);
				setReviews(result?.review_items || []);
				await loadMetrics();
			} catch (error) {
				alert('Failed to run resolver analysis: ' + error.message);
			}
		};

		const handleReviewDecision = async (itemId, decision) => {
			try {
				await apiCall(`/api/reviews/${itemId}/decision`, {
					method: 'POST',
					body: JSON.stringify({
						decision,
						rationale: rationales[itemId] || '',
						reviewer,
					}),
				});
				await Promise.all([loadReviews(), loadMetrics()]);
			} catch (error) {
				alert('Failed to submit review decision: ' + error.message);
			}
		};

		const insightColumns = [
			{ key: 'type', label: 'Type' },
			{ key: 'severity', label: 'Severity' },
			{ key: 'confidence', label: 'Confidence', render: row => Number(row.confidence || 0).toFixed(2) },
			{ key: 'status', label: 'Status' },
			{ key: 'explanation', label: 'Explanation' },
			{ key: 'suggested_action', label: 'Suggested Action' },
		];

		if (!activeProject?.id) {
			return (
				<div className="content-section">
					<div className="section-header"><h2>Insights & Review</h2></div>
					<div className="empty-state">Select a project before generating insights, running resolver analysis, or reviewing findings.</div>
				</div>
			);
		}

		return (
			<div className="content-section">
				<div className="section-header">
					<h2>Insights & Review</h2>
					<div style={{ display: 'flex', gap: '8px' }}>
						<Button label="Generate Insights" onClick={handleGenerateInsights} />
						<Button label="Refresh" onClick={() => Promise.all([loadInsights(), loadReviews(), loadMetrics(), loadStorageConfigs()])} variant="secondary" />
					</div>
				</div>
				<div style={{ color: 'var(--text-secondary)', marginBottom: '16px' }}>Project: <strong style={{ color: 'var(--accent)' }}>{activeProject.name}</strong></div>

				{loading ? <div className="loading">Loading insights and review queue...</div> : (
					<>
						<div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(220px, 1fr))', gap: '12px', marginBottom: '20px' }}>
							<div style={{ padding: '12px', border: '1px solid var(--border)', borderRadius: '8px', background: 'var(--surface)' }}>
								<div style={{ color: 'var(--text-secondary)', marginBottom: '6px' }}>High-confidence precision</div>
								<div style={{ fontSize: '1.5rem', color: 'var(--accent)' }}>{resolverMetrics ? Number(resolverMetrics.high_confidence_precision || 0).toFixed(2) : '0.00'}</div>
							</div>
							<div style={{ padding: '12px', border: '1px solid var(--border)', borderRadius: '8px', background: 'var(--surface)' }}>
								<div style={{ color: 'var(--text-secondary)', marginBottom: '6px' }}>Pending review</div>
								<div style={{ fontSize: '1.5rem', color: 'var(--accent)' }}>{resolverMetrics?.decision_counts?.pending || 0}</div>
							</div>
							<div style={{ padding: '12px', border: '1px solid var(--border)', borderRadius: '8px', background: 'var(--surface)' }}>
								<div style={{ color: 'var(--text-secondary)', marginBottom: '6px' }}>Accepted feedback</div>
								<div style={{ fontSize: '1.5rem', color: 'var(--accent)' }}>{resolverMetrics?.decision_counts?.accepted || 0}</div>
							</div>
							<div style={{ padding: '12px', border: '1px solid var(--border)', borderRadius: '8px', background: 'var(--surface)' }}>
								<div style={{ color: 'var(--text-secondary)', marginBottom: '6px' }}>Rejected feedback</div>
								<div style={{ fontSize: '1.5rem', color: 'var(--accent)' }}>{resolverMetrics?.decision_counts?.rejected || 0}</div>
							</div>
						</div>

						<div style={{ padding: '16px', border: '1px solid var(--border)', borderRadius: '8px', marginBottom: '20px', background: 'rgba(255,153,0,0.04)' }}>
							<h3 style={{ marginBottom: '8px' }}>Resolver Review Queue</h3>
							<div style={{ color: 'var(--text-secondary)', marginBottom: '12px' }}>Select at least two storage configs and run generic cross-source resolution.</div>
							<div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(220px, 1fr))', gap: '8px', marginBottom: '12px' }}>
								{storageConfigs.map(cfg => (
									<label key={cfg.id} style={{ display: 'flex', alignItems: 'center', gap: '8px', padding: '8px', border: '1px solid var(--border)', borderRadius: '6px', background: 'var(--surface)' }}>
										<input
											type="checkbox"
											checked={selectedStorageIds.includes(cfg.id)}
											onChange={e => setSelectedStorageIds(prev => e.target.checked ? [...prev, cfg.id] : prev.filter(id => id !== cfg.id))}
										/>
										<span>{deriveStorageConfigLabel(cfg)}</span>
									</label>
								))}
							</div>
							<div className="form-grid">
								<FormField label="Reviewer" value={reviewer} onChange={setReviewer} placeholder="analyst@team" />
								<div style={{ display: 'flex', alignItems: 'flex-end' }}>
									<Button label="Run Resolver Analysis" onClick={handleRunResolver} disabled={selectedStorageIds.length < 2} />
								</div>
							</div>
							{selectedStorageIds.length < 2 && <div style={{ color: 'var(--text-secondary)', marginTop: '8px' }}>Choose at least two storage configs to compare.</div>}
						</div>

						<div style={{ padding: '16px', border: '1px solid var(--border)', borderRadius: '8px', marginBottom: '20px' }}>
							<div className="section-header" style={{ marginBottom: '12px' }}>
								<h3 style={{ margin: 0 }}>Insights</h3>
								<div style={{ display: 'flex', gap: '8px' }}>
									<FormField label="Severity" type="select" value={severityFilter} onChange={setSeverityFilter} options={[{ value: '', label: 'all' }, 'low', 'medium', 'high', 'critical']} />
									<FormField label="Min Confidence" type="number" value={minConfidence} onChange={setMinConfidence} placeholder="0.5" />
									<Button label="Apply Filters" onClick={loadInsights} variant="secondary" />
								</div>
							</div>
							<Table columns={insightColumns} data={insights} />
						</div>

						<div style={{ padding: '16px', border: '1px solid var(--border)', borderRadius: '8px' }}>
							<div className="section-header" style={{ marginBottom: '12px' }}>
								<h3 style={{ margin: 0 }}>Review Queue</h3>
								<div style={{ display: 'flex', gap: '8px' }}>
									<FormField label="Status" type="select" value={reviewStatus} onChange={setReviewStatus} options={[{ value: '', label: 'all' }, 'pending', 'accepted', 'rejected', 'auto_accepted']} />
									<Button label="Reload Queue" onClick={loadReviews} variant="secondary" />
								</div>
							</div>
							{reviews.length === 0 ? (
								<div className="empty-state">No review items for the current filters.</div>
							) : (
								reviews.map(item => (
									<div key={item.id} style={{ padding: '12px', border: '1px solid var(--border)', borderRadius: '6px', marginBottom: '12px', background: 'var(--surface)' }}>
										<div style={{ display: 'flex', justifyContent: 'space-between', gap: '12px', marginBottom: '8px' }}>
											<div>
												<div style={{ color: 'var(--accent)', fontWeight: 'bold' }}>{item.finding_type}</div>
												<div style={{ color: 'var(--text-secondary)', fontSize: '0.85rem' }}>Status: {item.status} · Suggested: {item.suggested_decision} · Confidence: {Number(item.confidence || 0).toFixed(2)}</div>
											</div>
											<div className={`status-badge status-${item.status === 'accepted' || item.status === 'auto_accepted' ? 'active' : item.status === 'rejected' ? 'failed' : 'pending'}`}>{item.status}</div>
										</div>
										<div style={{ marginBottom: '8px' }}>{item.rationale}</div>
										<pre style={{ margin: '0 0 12px 0', fontSize: '0.75rem', whiteSpace: 'pre-wrap' }}>{renderConfigPreview(item.evidence || item.payload)}</pre>
										<FormField label="Decision rationale" type="textarea" value={rationales[item.id] || ''} onChange={value => setRationales(prev => ({ ...prev, [item.id]: value }))} placeholder="Why are you accepting or rejecting this link?" />
										<div style={{ display: 'flex', gap: '8px' }}>
											<Button label="Accept" onClick={() => handleReviewDecision(item.id, 'accept')} />
											<Button label="Reject" onClick={() => handleReviewDecision(item.id, 'reject')} variant="danger" />
										</div>
									</div>
								))
							)}
						</div>
					</>
				)}
			</div>
		);
	};
})();

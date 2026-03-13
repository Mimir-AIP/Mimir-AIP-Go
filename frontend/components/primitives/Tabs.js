(() => {
	const root = window.MimirApp = window.MimirApp || {};
	const primitives = (((root.components = root.components || {}).primitives = root.components.primitives || {}));

	primitives.Tabs = function Tabs({ tabs, activeTab, onTabChange }) {
		return (
			<div className="tabs-container">
				{tabs.map(tab => (
					<button
						key={tab}
						className={`tab${tab === activeTab ? ' active' : ''}`}
						onClick={() => onTabChange(tab)}
					>
						{tab}
					</button>
				))}
			</div>
		);
	};
})();

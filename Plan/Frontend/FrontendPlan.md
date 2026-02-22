# Frontend

## Overview
The frontend will be a simple static web application built using seven primitive components which will be detailed below, the frontend will use REST API calls to interact with the orchestrator backend. 
## Infrastructure
Frontend will be hosted on a simple static web server, similar to backend orchestrator server this will be in a container within the same cluster as the backend.

## Components

### Example Component Definitions & Usage

#### 1. Tabs
Component:
```jsx
function Tabs({ tabs, activeTab, onTabChange }) {
	return (
		<div style={{ display: 'flex', gap: '8px' }}>
			{tabs.map(tab => (
				<button
					key={tab}
					className="tab"
					style={{
						fontWeight: tab === activeTab ? 'bold' : 'normal',
						background: tab === activeTab ? 'var(--accent)' : 'var(--background)',
						color: 'var(--text)',
						fontFamily: 'var(--font-family)',
						border: 'none',
						padding: '8px 16px',
						cursor: 'pointer',
					}}
					onClick={() => onTabChange(tab)}
				>
					{tab}
				</button>
			))}
		</div>
	);
}
```
Usage:
```jsx
<Tabs
	tabs={["Home", "Settings", "Logs"]}
	activeTab={activeTab}
	onTabChange={setActiveTab}
/> 
```

#### 2. Forms
Component:
```jsx
function Form({ onSubmit }) {
	const [value, setValue] = React.useState('');
	return (
		<form
			onSubmit={e => { e.preventDefault(); onSubmit(value); }}
			style={{ fontFamily: 'var(--font-family)', color: 'var(--text)' }}
		>
			<input
				value={value}
				onChange={e => setValue(e.target.value)}
				style={{
					background: 'var(--background)',
					color: 'var(--text)',
					fontFamily: 'var(--font-family)',
					border: '1px solid var(--accent)',
					padding: '8px',
				}}
			/>
			<button type="submit" className="accent" style={{ marginLeft: 8 }}>
				Submit
			</button>
		</form>
	);
}
```
Usage:
```jsx
<Form onSubmit={val => alert('Submitted: ' + val)} />
```

#### 3. Tables
Component:
```jsx
function Table({ columns, data }) {
	return (
		<table style={{ width: '100%', fontFamily: 'var(--font-family)', color: 'var(--text)', background: 'var(--background)' }}>
			<thead>
				<tr>
					{columns.map(col => (
						<th key={col} style={{ borderBottom: '2px solid var(--accent)', padding: '8px' }}>{col}</th>
					))}
				</tr>
			</thead>
			<tbody>
				{data.map((row, i) => (
					<tr key={i}>
						{columns.map(col => (
							<td key={col} style={{ borderBottom: '1px solid var(--accent)', padding: '8px' }}>{row[col]}</td>
						))}
					</tr>
				))}
			</tbody>
		</table>
	);
}
```
Usage:
```jsx
<Table
	columns={["Name", "Value"]}
	data={[{ Name: 'A', Value: 1 }, { Name: 'B', Value: 2 }]}
/> 
```

#### 4. Buttons
Component:
```jsx
function Button({ label, onClick }) {
	return (
		<button
			className="accent"
			style={{
				background: 'var(--accent)',
				color: 'var(--text)',
				fontFamily: 'var(--font-family)',
				border: 'none',
				padding: '8px 16px',
				cursor: 'pointer',
			}}
			onClick={onClick}
		>
			{label}
		</button>
	);
}
```
Usage:
```jsx
<Button label="Click Me" onClick={() => alert('Clicked!')} />
```

#### 5. Modals
Component:
```jsx
function Modal({ open, onClose, children }) {
	if (!open) return null;
	return (
		<div style={{ position: 'fixed', top: 0, left: 0, right: 0, bottom: 0, background: '#0008', zIndex: 1000 }}>
			<div style={{ background: 'var(--background)', color: 'var(--text)', fontFamily: 'var(--font-family)', margin: '10% auto', padding: 20, width: 300, border: '2px solid var(--accent)', borderRadius: 8 }}>
				{children}
				<button
					className="accent"
					style={{ marginTop: 16 }}
					onClick={onClose}
				>
					Close
				</button>
			</div>
		</div>
	);
}
```
Usage:
```jsx
<Modal open={showModal} onClose={() => setShowModal(false)}>
	<p>Modal Content</p>
</Modal>
```

#### 6. Graphs
Component (using Chart.js via react-chartjs-2):
```jsx
import { Line } from 'react-chartjs-2';
function Graph({ data, options }) {
	return (
		<div style={{ background: 'var(--background)', color: 'var(--text)', fontFamily: 'var(--font-family)', padding: 16 }}>
			<Line data={data} options={options} />
		</div>
	);
}
```
Usage:
```jsx
<Graph
	data={{
		labels: ['Jan', 'Feb'],
		datasets: [{ label: 'Sales', data: [10, 20], backgroundColor: 'var(--accent)', borderColor: 'var(--accent)' }]
	}}
	options={{ responsive: true }}
/> 
```
...existing code...

## API Integration
The frontend will interact with the backend orchestrator through REST API calls. The frontend will make API calls to fetch data for display, submit user input, and perform actions such as starting or stopping services. The API endpoints will be defined in the backend orchestrator and the frontend will be designed to consume these endpoints effectively.

## Styling
The frontend will be styled using CSS, with a focus on simplicity and usability. We will ship a single colour scheme with the palette defined below, and ensure the frontend is responsive and works well on different screen sizes.

Palette:
```css
:root {
	--background: #1a2236; /* dark navy blue */
	--accent: #ff9900;    /* orange highlights */
	--text: #ffffff;      /* white text */
	--font-family: 'Google Sans Code', monospace;
}

body {
	background: var(--background);
	color: var(--text);
	font-family: var(--font-family);
}

.accent {
	color: var(--accent);
}

/* Example usage for buttons, tabs, etc. */
button, .tab {
	background: var(--accent);
	color: var(--text);
	font-family: var(--font-family);
}
```

Font:
Add this to your HTML `<head>`:
```html
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=Google+Sans+Code:ital,wght@0,300..800;1,300..800&display=swap" rel="stylesheet">
```
...existing code...

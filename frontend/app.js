// API Configuration
const API_URL = window.location.origin.includes('localhost') 
    ? 'http://localhost:8080' 
    : '/api';

// State
let jobs = [];
let systemMetrics = {
    queueLength: 0,
    activeWorkers: 0,
    jobsCompleted: 0,
    jobsFailed: 0
};

// Initialize app
document.addEventListener('DOMContentLoaded', () => {
    initializeApp();
    setupEventListeners();
    startPolling();
});

// Initialize application
function initializeApp() {
    checkSystemHealth();
    updateDashboard();
}

// Setup event listeners
function setupEventListeners() {
    const form = document.getElementById('jobSubmissionForm');
    form.addEventListener('submit', handleJobSubmission);
}

// Check system health
async function checkSystemHealth() {
    try {
        const response = await fetch(`${API_URL}/health`);
        const data = await response.json();
        
        const statusElement = document.getElementById('systemStatus');
        const statusDot = statusElement.querySelector('.status-dot');
        const statusText = statusElement.querySelector('.status-text');
        
        if (data.status === 'healthy') {
            statusDot.classList.add('online');
            statusText.textContent = 'Online';
        } else {
            statusDot.classList.add('offline');
            statusText.textContent = 'Offline';
        }
    } catch (error) {
        console.error('Health check failed:', error);
        const statusElement = document.getElementById('systemStatus');
        const statusDot = statusElement.querySelector('.status-dot');
        const statusText = statusElement.querySelector('.status-text');
        statusDot.classList.add('offline');
        statusText.textContent = 'Offline';
    }
}

// Update dashboard metrics
async function updateDashboard() {
    try {
        const response = await fetch(`${API_URL}/api/jobs`);
        const data = await response.json();
        
        document.getElementById('queueLength').textContent = data.queue_length || 0;
        
        // In a real implementation, these would come from the API
        // For now, we'll use placeholder values
        document.getElementById('activeWorkers').textContent = '0';
        document.getElementById('jobsCompleted').textContent = '0';
        document.getElementById('jobsFailed').textContent = '0';
    } catch (error) {
        console.error('Failed to update dashboard:', error);
    }
}

// Handle job submission
async function handleJobSubmission(event) {
    event.preventDefault();
    
    const formData = {
        type: document.getElementById('jobType').value,
        priority: parseInt(document.getElementById('priority').value),
        project_id: document.getElementById('projectId').value,
        task_spec: {
            pipeline_id: 'default-pipeline',
            project_id: document.getElementById('projectId').value,
            parameters: {}
        },
        resource_requirements: {
            cpu: document.getElementById('cpu').value,
            memory: document.getElementById('memory').value,
            gpu: false
        },
        data_access: {
            input_datasets: [],
            output_location: `s3://results/${Date.now()}/`,
            storage_credentials: ''
        }
    };

    try {
        const response = await fetch(`${API_URL}/api/jobs`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(formData)
        });

        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }

        const result = await response.json();
        
        showResultMessage('success', `Job submitted successfully! Job ID: ${result.job_id}`);
        
        // Reset form
        document.getElementById('jobSubmissionForm').reset();
        
        // Update dashboard
        setTimeout(updateDashboard, 500);
        
    } catch (error) {
        console.error('Job submission failed:', error);
        showResultMessage('error', `Failed to submit job: ${error.message}`);
    }
}

// Show result message
function showResultMessage(type, message) {
    const resultElement = document.getElementById('submitResult');
    resultElement.className = `result-message ${type}`;
    resultElement.textContent = message;
    
    setTimeout(() => {
        resultElement.className = 'result-message';
        resultElement.textContent = '';
    }, 5000);
}

// Start polling for updates
function startPolling() {
    // Update every 5 seconds
    setInterval(() => {
        checkSystemHealth();
        updateDashboard();
    }, 5000);
}

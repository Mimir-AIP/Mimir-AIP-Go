/**
 * @typedef {Object} Message
 * @property {string} type
 * @property {any} [data]
 * @property {string} [status]
 * @property {string} [message]
 * @property {number} [timestamp]
 * @property {any} [error]
 * @property {string} [details]
 */

/**
 * WebInterface frontend application
 * @class
 */
// Chart.js CDN
document.write('<script src="https://cdn.jsdelivr.net/npm/chart.js"></script>');

class WebInterfaceApp {
    // Pipeline message types
    static PIPELINE_MESSAGE_TYPES = {
        INPUT: 'pipeline_input',
        OUTPUT: 'pipeline_output',
        STATUS: 'pipeline_status',
        LLM_QUERY: 'llm_query',
        LLM_RESPONSE: 'llm_response',
        STREAM_CONTROL: 'stream_control',
        STREAM_STATUS: 'stream_status',
        VISUALIZATION_UPDATE: 'visualization_update'
    };

    /**
     * @constructor
     */
    constructor() {
        this.sectionElements = new Map();
        this.container = document.getElementById('content-container');
        this.pipelineContainer = document.getElementById('pipeline-visualizer-container');
        this.charts = new Map();
        
        // Initialize error container
        this.errorContainer = document.createElement('div');
        this.errorContainer.id = 'error-container';
        this.errorContainer.style.display = 'none';
        this.errorContainer.style.position = 'fixed';
        this.errorContainer.style.bottom = '20px';
        this.errorContainer.style.right = '20px';
        this.errorContainer.style.maxWidth = '400px';
        this.errorContainer.style.padding = '15px';
        this.errorContainer.style.backgroundColor = '#ffebee';
        this.errorContainer.style.border = '1px solid #ef9a9a';
        this.errorContainer.style.borderRadius = '4px';
        this.errorContainer.style.boxShadow = '0 2px 10px rgba(0,0,0,0.1)';
        this.errorContainer.style.zIndex = '1000';
        document.body.appendChild(this.errorContainer);

        // Initialize content handlers
        this.contentHandlers = {
            'text': this.handleTextContent.bind(this),
            'html': this.handleHtmlContent.bind(this),
            'javascript': this.handleJsContent.bind(this),
            'mixed': this.handleMixedContent.bind(this),
            'chart': this.handleChartContent.bind(this)
        };

        // Create pipeline visualization section
        this.pipelineSection = document.createElement('div');
        this.pipelineSection.id = 'pipeline-section';
        this.pipelineSection.className = 'dashboard-section';
        this.pipelineSection.innerHTML = `
            <div class="section-header">
                <h2>Pipeline Status</h2>
                <button id="toggle-pipeline" class="toggle-button">Show</button>
            </div>
            <div class="section-content" id="pipeline-content"></div>
        `;
        this.container.appendChild(this.pipelineSection);
        
        // Setup toggle button
        document.getElementById('toggle-pipeline').addEventListener('click', () => {
            const button = document.getElementById('toggle-pipeline');
            if (this.pipelineContainer.style.display === 'none') {
                this.pipelineContainer.style.display = 'block';
                button.textContent = 'Hide';
            } else {
                this.pipelineContainer.style.display = 'none';
                button.textContent = 'Show';
            }
        });

        this.initWebSocket();
    }


    initWebSocket() {
        this.socket = new WebSocket(`ws://${window.location.host}/ws`);
        this.socket.onmessage = (event) => this.handleMessage(event);
        this.socket.onerror = (error) => this.handleSocketError(error);
        this.socket.onclose = (event) => {
            if (event.wasClean) {
                console.log(`WebSocket closed cleanly, code=${event.code}, reason=${event.reason}`);
            } else {
                this.showError('Connection lost. Attempting to reconnect...');
                setTimeout(() => this.initWebSocket(), 5000);
            }
        };
    }

    sendPipelineInput(data) {
        if (this.socket.readyState === WebSocket.OPEN) {
            this.socket.send(JSON.stringify({
                type: WebInterfaceApp.PIPELINE_MESSAGE_TYPES.INPUT,
                data: data
            }));
        } else {
            this.showError('Connection not ready', 'Cannot send pipeline input');
        }
    }

    handleMessage(event) {
        try {
            const message = JSON.parse(event.data);
            
            if (message.error) {
                this.showError(message.error.message || 'An error occurred', message.error.details);
                return;
            }

            if (message.type === 'content_update') {
                this.renderContent(message.content);
            }
            else if (message.type === 'dashboard_update') {
                this.handleDashboardUpdate(message);
            }
            else if (message.type === 'pipeline_update') {
                this.handlePipelineUpdate(message);
            }
            else if (message.type === WebInterfaceApp.PIPELINE_MESSAGE_TYPES.OUTPUT) {
                this.handlePipelineOutput(message);
            }
            else if (message.type === WebInterfaceApp.PIPELINE_MESSAGE_TYPES.STATUS) {
                this.handlePipelineStatus(message);
            }
            else if (message.type === WebInterfaceApp.PIPELINE_MESSAGE_TYPES.LLM_RESPONSE) {
                this.handleLLMResponse(message);
            }
            else if (message.type === WebInterfaceApp.PIPELINE_MESSAGE_TYPES.STREAM_CONTROL) {
                this.handleStreamControl(message);
            }
            else if (message.type === WebInterfaceApp.PIPELINE_MESSAGE_TYPES.VISUALIZATION_UPDATE) {
                this.handlePipelineUpdate(message);
            }
            else if (message.type === 'error') {
                this.showError(message.message, message.details);
            }
        } catch (e) {
            this.showError('Failed to process message', e.message);
            console.error('Message handling error:', e);
        }
    }

    handleSocketError(error) {
        this.showError('WebSocket error', 'Connection error occurred');
        console.error('WebSocket error:', error);
    }

    showError(message, details = '') {
        this.errorContainer.style.display = 'block';
        this.errorContainer.innerHTML = `
            <div style="display: flex; justify-content: space-between; align-items: center;">
                <strong style="color: #c62828;">${message}</strong>
                <button onclick="document.getElementById('error-container').style.display='none'"
                        style="background: none; border: none; cursor: pointer; font-size: 1.2em;">×</button>
            </div>
            ${details ? `<div style="margin-top: 8px; font-size: 0.9em; color: #555;">${details}</div>` : ''}
        `;
        
        // Auto-hide after 10 seconds
        setTimeout(() => {
            this.errorContainer.style.display = 'none';
        }, 10000);
    }

    handleDashboardUpdate(message) {
        try {
            if (!message.sections || !Array.isArray(message.sections)) {
                throw new Error('Invalid dashboard update format');
            }

            message.sections.forEach(section => {
                if (!section.id) return;
                
                const existing = this.sectionElements.get(section.id);
                
                if (existing) {
                    // Update existing section
                    existing.innerHTML = this.renderSection(section);
                } else {
                    // Create new section
                    const element = document.createElement('div');
                    element.id = `section-${section.id}`;
                    element.className = 'dashboard-section';
                    element.innerHTML = this.renderSection(section);
                    this.container.appendChild(element);
                    this.sectionElements.set(section.id, element);
                }
                
                // Execute any JavaScript
                if (section.javascript) {
                    try {
                        new Function(section.javascript)();
                    } catch (e) {
                        this.showError('Section script error', e.message);
                        console.error('Error executing section JS:', e);
                    }
                }
            });
        } catch (e) {
            this.showError('Dashboard update failed', e.message);
            console.error('Dashboard update error:', e);
        }
    }

    renderSection(section) {
        return `
            <div class="section-header">
                <h2>${section.heading}</h2>
            </div>
            <div class="section-content">
                ${section.content}
            </div>
        `;
    }

    renderContent(contentBlocks) {
        this.container.innerHTML = '';
        contentBlocks.forEach(block => {
            const handler = this.contentHandlers[block.type] || this.contentHandlers['mixed'];
            handler(block);
        });
    }

    handleTextContent(block) {
        const div = document.createElement('div');
        div.className = 'content-block text-content';
        div.textContent = block.text;
        this.container.appendChild(div);
    }

    handleHtmlContent(block) {
        const div = document.createElement('div');
        div.className = 'content-block html-content';
        div.innerHTML = block.html;
        this.container.appendChild(div);
    }

    handleJsContent(block) {
        const script = document.createElement('script');
        script.textContent = block.javascript;
        this.container.appendChild(script);
    }

    handleMixedContent(block) {
        const div = document.createElement('div');
        div.className = 'content-block mixed-content';
        
        if (block.text) {
            const textElem = document.createElement('div');
            textElem.textContent = block.text;
            div.appendChild(textElem);
        }
        
        if (block.html) {
            const htmlElem = document.createElement('div');
            htmlElem.innerHTML = block.html;
            div.appendChild(htmlElem);
        }
        
        if (block.javascript) {
            const script = document.createElement('script');
            script.textContent = block.javascript;
            div.appendChild(script);
        }
        
        this.container.appendChild(div);
    }

    // Safe HTML sanitization helper
    sanitizeHtml(html) {
        const temp = document.createElement('div');
        temp.textContent = html;
        return temp.innerHTML;
    }
    
    // Chart content handler
    handleChartContent(block) {
        const container = document.createElement('div');
        container.className = 'chart-container';
        container.style.width = '100%';
        container.style.height = '400px';
        
        const canvas = document.createElement('canvas');
        container.appendChild(canvas);
        this.container.appendChild(container);
        
        // Destroy existing chart if present
        if (this.charts.has(block.id)) {
            this.charts.get(block.id).destroy();
        }
        
        // Create new chart
        const ctx = canvas.getContext('2d');
        const chart = new Chart(ctx, {
            type: block.chartType || 'bar',
            data: block.data || {
                labels: ['Example 1', 'Example 2', 'Example 3'],
                datasets: [{
                    label: 'Sample Data',
                    data: [12, 19, 3],
                    backgroundColor: [
                        'rgba(255, 99, 132, 0.2)',
                        'rgba(54, 162, 235, 0.2)',
                        'rgba(255, 206, 86, 0.2)'
                    ],
                    borderColor: [
                        'rgba(255, 99, 132, 1)',
                        'rgba(54, 162, 235, 1)',
                        'rgba(255, 206, 86, 1)'
                    ],
                    borderWidth: 1
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                scales: {
                    y: {
                        beginAtZero: true
                    }
                }
            }
        });
        
        this.charts.set(block.id, chart);
    }

    handlePipelineOutput(message) {
        // Create or update output section
        const outputSection = document.getElementById('pipeline-output-section') ||
            this.createPipelineSection('pipeline-output-section', 'Pipeline Output');
            
        const outputDiv = document.createElement('div');
        outputDiv.className = 'pipeline-output';
        outputDiv.textContent = JSON.stringify(message.data, null, 2);
        outputSection.querySelector('.section-content').appendChild(outputDiv);
    }

    handlePipelineStatus(message) {
        const statusSection = document.getElementById('pipeline-status-section') ||
            this.createPipelineSection('pipeline-status-section', 'Pipeline Status');
            
        const statusDiv = document.createElement('div');
        statusDiv.className = 'pipeline-status';
        statusDiv.innerHTML = `
            <div class="status-indicator ${message.status}"></div>
            <div class="status-message">${message.message}</div>
            <div class="status-timestamp">${new Date(message.timestamp * 1000).toLocaleString()}</div>
        `;
        statusSection.querySelector('.section-content').appendChild(statusDiv);
    }

    createPipelineSection(id, title) {
        const section = document.createElement('div');
        section.id = id;
        section.className = 'dashboard-section pipeline-section';
        section.innerHTML = `
            <div class="section-header">
                <h2>${title}</h2>
            </div>
            <div class="section-content"></div>
        `;
        this.container.appendChild(section);
        return section;
    }

    handlePipelineUpdate(message) {
        if (window.updatePipeline) {
            updatePipeline(message.tree, message.highlight_path);
        }
    }

    handleLLMResponse(message) {
        if (window.handleLLMResponse) {
            handleLLMResponse(message.data);
        }
    }

    handleStreamControl(message) {
        if (window.handleStreamControl) {
            handleStreamControl(message.command, message.data);
        }
    }
}

// Initialize app when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    window.app = new WebInterfaceApp();
});
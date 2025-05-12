/**
 * WebInterface frontend application
 */
class WebInterfaceApp {
    constructor() {
        this.socket = new WebSocket(`ws://${window.location.host}/ws`);
        this.socket.onmessage = (event) => this.handleMessage(event);
        this.container = document.getElementById('content-container');
        this.pipelineContainer = document.getElementById('pipeline-visualizer-container');
        
        // Initialize content types handlers
        this.contentHandlers = {
            'text': this.handleTextContent.bind(this),
            'html': this.handleHtmlContent.bind(this),
            'javascript': this.handleJsContent.bind(this),
            'mixed': this.handleMixedContent.bind(this)
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
    }

    constructor() {
        this.sectionElements = new Map();
        this.socket = new WebSocket(`ws://${window.location.host}/ws`);
        this.socket.onmessage = (event) => this.handleMessage(event);
        this.container = document.getElementById('content-container');
    }

    handleMessage(event) {
        const message = JSON.parse(event.data);
        if (message.type === 'content_update') {
            this.renderContent(message.content);
        }
        else if (message.type === 'dashboard_update') {
            this.handleDashboardUpdate(message);
        }
        else if (message.type === 'pipeline_update') {
            this.handlePipelineUpdate(message);
        }
    }

    handleDashboardUpdate(message) {
        message.sections.forEach(section => {
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
                    console.error('Error executing section JS:', e);
                }
            }
        });
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

    handlePipelineUpdate(message) {
        if (window.updatePipeline) {
            updatePipeline(message.tree, message.highlight_path);
        }
    }
}

// Initialize app when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    window.app = new WebInterfaceApp();
});
/**
 * WebInterface frontend application
 */
class WebInterfaceApp {
    constructor() {
        this.socket = new WebSocket(`ws://${window.location.host}/ws`);
        this.socket.onmessage = (event) => this.handleMessage(event);
        this.container = document.getElementById('content-container');
        
        // Initialize content types handlers
        this.contentHandlers = {
            'text': this.handleTextContent.bind(this),
            'html': this.handleHtmlContent.bind(this),
            'javascript': this.handleJsContent.bind(this),
            'mixed': this.handleMixedContent.bind(this)
        };
    }

    handleMessage(event) {
        const message = JSON.parse(event.data);
        if (message.type === 'content_update') {
            this.renderContent(message.content);
        }
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
}

// Initialize app when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    window.app = new WebInterfaceApp();
});
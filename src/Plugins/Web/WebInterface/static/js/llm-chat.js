/**
 * LLM Chat Component with Theme Support
 */
class LLMChat {
    constructor(containerId, contextData, themeConfig = {}) {
        this.container = document.getElementById(containerId);
        this.context = contextData;
        this.theme = themeConfig;
        this.chatHistory = [];
        this.initUI();
        this.applyTheme();
    }

    initUI() {
        this.container.innerHTML = `
            <div class="llm-chat" data-dashboard-integration="true">
                <div class="chat-header">
                    <h3>${this.theme.title || 'LLM Chat'}</h3>
                    <select class="context-select">
                        ${this.getContextOptions()}
                    </select>
                    <button class="export-btn">Export to Context</button>
                </div>
                <div class="context-viewer">
                    <pre class="context-data"></pre>
                </div>
                <div class="chat-messages"></div>
                <div class="chat-input">
                    <input type="text" placeholder="Ask about the context data...">
                    <button class="send-btn">Send</button>
                </div>
            </div>
        `;

        // Event listeners
        this.container.querySelector('.send-btn').addEventListener('click', () => this.sendMessage());
        this.container.querySelector('input').addEventListener('keypress', (e) => {
            if (e.key === 'Enter') this.sendMessage();
        });
        this.container.querySelector('.export-btn').addEventListener('click', () => this.exportToContext());
        this.container.querySelector('.context-select').addEventListener('change', (e) => this.updateContextView(e.target.value));
        
        this.updateContextView(Object.keys(this.context)[0]);
    }

    applyTheme() {
        if (this.theme.colors) {
            const root = document.documentElement;
            Object.entries(this.theme.colors).forEach(([key, value]) => {
                root.style.setProperty(`--${key}`, value);
            });
        }
    }

    getContextOptions() {
        return Object.keys(this.context)
            .map(key => `<option value="${key}">${key}</option>`)
            .join('');
    }

    updateContextView(key) {
        const data = this.context[key];
        const display = typeof data === 'object' 
            ? JSON.stringify(data, null, 2) 
            : String(data);
        
        this.container.querySelector('.context-data').textContent = display;
    }

    async sendMessage() {
        const input = this.container.querySelector('input');
        const message = input.value.trim();
        if (!message) return;

        this.addMessage('user', message);
        input.value = '';

        try {
            const selectedKey = this.container.querySelector('.context-select').value;
            const contextData = this.context[selectedKey];
            const response = await this.queryLLM(message, selectedKey, contextData);
            this.addMessage('assistant', response);
        } catch (error) {
            this.addMessage('error', `Error: ${error.message}`);
        }
    }

    async queryLLM(query, contextKey, contextData) {
        const contextStr = typeof contextData === 'object'
            ? JSON.stringify(contextData)
            : String(contextData);

        const prompt = `Context (${contextKey}):\n${contextStr}\n\nQuestion: ${query}`;

        // Send LLM query through pipeline
        const queryData = {
            type: 'llm_query',
            prompt: prompt,
            contextKey: selectedKey,
            contextData: contextData
        };
        
        window.app.sendPipelineInput(queryData);
        
        // Return a promise that will be resolved by the response handler
        return new Promise((resolve) => {
            this.pendingResponse = resolve;
        });
    }

    addMessage(role, content) {
        const messagesDiv = this.container.querySelector('.chat-messages');
        const messageDiv = document.createElement('div');
        messageDiv.className = `message ${role}`;
        messageDiv.innerHTML = `<strong>${role}:</strong> ${content}`;
        messagesDiv.appendChild(messageDiv);
        messagesDiv.scrollTop = messagesDiv.scrollHeight;
        
        this.chatHistory.push({role, content});
    }

    exportToContext() {
        const lastResponse = this.chatHistory
            .filter(m => m.role === 'assistant')
            .pop()?.content;

        if (!lastResponse) return;

        const exportData = {
            type: 'export_to_context',
            data: {
                key: `chat_${Date.now()}`,
                value: lastResponse
            }
        };
        
        // Check if we should also create a dashboard section
        if (this.container.dataset.dashboardIntegration === "true") {
            window.app.sendMessage('dashboard_update', {
                id: `chat_${Date.now()}`,
                action: "add",
                data: {
                    heading: "Chat Export",
                    content: lastResponse,
                    javascript: ""
                }
            });
        }
        
        window.app.sendPipelineInput(exportData);
    }
}

// Global handler for LLM responses from pipeline
window.handleLLMResponse = function(response) {
    const chat = window.currentChatInstance;
    if (chat && chat.pendingResponse) {
        chat.pendingResponse(response);
        chat.pendingResponse = null;
    }
};
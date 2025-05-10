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
            <div class="llm-chat">
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

        const response = await fetch('/llm-query', {
            method: 'POST',
            headers: {'Content-Type': 'application/json'},
            body: JSON.stringify({prompt})
        });

        if (!response.ok) throw new Error('LLM query failed');
        return await response.text();
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
        window.app.sendMessage(exportData.type, exportData.data);
    }
}
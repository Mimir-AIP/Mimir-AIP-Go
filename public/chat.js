document.addEventListener('DOMContentLoaded', function() {
    const chatInput = document.getElementById('chatInput');
    const sendMessageButton = document.getElementById('sendMessage');
    const chatMessages = document.getElementById('chatMessages');
    const llmProviderSelect = document.getElementById('llmProvider');
    const llmModelSelect = document.getElementById('llmModel');

    let availableProviders = {}; // To store providers and their models

    function appendMessage(sender, text) {
        const messageDiv = document.createElement('div');
        messageDiv.classList.add('message', sender);
        messageDiv.textContent = text;
        chatMessages.appendChild(messageDiv);
        chatMessages.scrollTop = chatMessages.scrollHeight; // Scroll to bottom
    }

    async function fetchLLMOptions() {
        try {
            const response = await fetch('/api/llm_options');
            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }
            availableProviders = await response.json();
            populateProviders();
        } catch (error) {
            console.error('Error fetching LLM options:', error);
            appendMessage('llm', `Error: Could not load LLM options. (${error.message})`);
        }
    }

    function populateProviders() {
        llmProviderSelect.innerHTML = ''; // Clear existing options
        for (const providerName in availableProviders) {
            const option = document.createElement('option');
            option.value = providerName;
            option.textContent = providerName;
            llmProviderSelect.appendChild(option);
        }
        populateModels(); // Populate models for the initially selected provider
    }

    function populateModels() {
        llmModelSelect.innerHTML = ''; // Clear existing options
        const selectedProvider = llmProviderSelect.value;
        const models = availableProviders[selectedProvider] || [];
        if (models.length === 0) {
            const option = document.createElement('option');
            option.value = '';
            option.textContent = 'No models available';
            llmModelSelect.appendChild(option);
            llmModelSelect.disabled = true;
        } else {
            llmModelSelect.disabled = false;
            models.forEach(model => {
                const option = document.createElement('option');
                option.value = model;
                option.textContent = model;
                llmModelSelect.appendChild(option);
            });
        }
    }

    async function sendMessage() {
        const message = chatInput.value.trim();
        if (message === '') return;

        const selectedProvider = llmProviderSelect.value;
        const selectedModel = llmModelSelect.value;

        if (!selectedProvider || !selectedModel) {
            appendMessage('llm', 'Error: Please select an LLM provider and model.');
            return;
        }

        appendMessage('user', message);
        chatInput.value = '';

        try {
            const response = await fetch('/api/chat', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    message: message,
                    provider: selectedProvider,
                    model: selectedModel
                }),
            });

            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            const data = await response.json();
            appendMessage('llm', data.response);
        } catch (error) {
            console.error('Error sending message to LLM:', error);
            appendMessage('llm', `Error: Could not get response from LLM. (${error.message})`);
        }
    }

    // Event Listeners
    sendMessageButton.addEventListener('click', sendMessage);
    chatInput.addEventListener('keypress', function(event) {
        if (event.key === 'Enter') {
            sendMessage();
        }
    });
    llmProviderSelect.addEventListener('change', populateModels);

    // Initial message from LLM and fetch options
    appendMessage('llm', 'Hello! How can I help you today?');
    fetchLLMOptions();
});
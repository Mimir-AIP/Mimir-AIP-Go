document.addEventListener('DOMContentLoaded', function() {
    const chatInput = document.getElementById('chatInput');
    const sendMessageButton = document.getElementById('sendMessage');
    const chatMessages = document.getElementById('chatMessages');
    const llmProviderSelect = document.getElementById('llmProvider');
    const llmModelSelect = document.getElementById('llmModel');

    let availableProviders = {}; // To store providers and their models
    let loadingMessage = null; // Reference to the loading message element
    let conversationHistory = []; // To maintain conversation context

    function appendMessage(sender, text) {
        const messageDiv = document.createElement('div');
        messageDiv.classList.add('message', sender);
        messageDiv.textContent = text;
        chatMessages.appendChild(messageDiv);
        chatMessages.scrollTop = chatMessages.scrollHeight; // Scroll to bottom
        return messageDiv; // Return the message element for reference
    }

    function showLoadingIndicator() {
        loadingMessage = appendMessage('system', 'Loading...');
    }

    function hideLoadingIndicator() {
        if (loadingMessage) {
            loadingMessage.remove();
            loadingMessage = null;
        }
    }

    async function fetchLLMOptions() {
        showLoadingIndicator();
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
        } finally {
            hideLoadingIndicator();
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

    async function sendMessageWithRetry(url, options, retries = 3) {
        for (let attempt = 1; attempt <= retries; attempt++) {
            try {
                const response = await fetch(url, options);
                if (!response.ok) {
                    throw new Error(`HTTP error! status: ${response.status}`);
                }
                return await response.json();
            } catch (error) {
                console.error(`Attempt ${attempt} failed:`, error);
                if (attempt === retries) {
                    throw error; // Re-throw after all retries
                }
                // Wait before retrying
                await new Promise(res => setTimeout(res, 1000 * attempt));
            }
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

        // Add user message to conversation history
        conversationHistory.push({ role: 'user', content: message });
        appendMessage('user', message);
        chatInput.value = '';

        showLoadingIndicator();
        try {
            const response = await sendMessageWithRetry('/api/chat', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    message: message, // Required field
                    provider: selectedProvider, // Required field
                    model: selectedModel, // Required field
                    messages: conversationHistory // Send conversation history
                }),
            });

            // Add LLM response to conversation history
            conversationHistory.push({ role: 'llm', content: response.response });
            appendMessage('llm', response.response);
        } catch (error) {
            console.error('Error sending message to LLM:', error);
            appendMessage('llm', `Error: Could not get response from LLM. (${error.message})`);
        } finally {
            hideLoadingIndicator();
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
    conversationHistory.push({ role: 'llm', content: 'Hello! How can I help you today?' });
    fetchLLMOptions();
});
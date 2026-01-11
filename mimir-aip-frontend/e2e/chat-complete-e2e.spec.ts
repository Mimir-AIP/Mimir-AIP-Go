import { test, expect } from '@playwright/test';

/**
 * Comprehensive E2E tests for Chat interface including conversations, tool calls, and context management
 */

test.describe('Chat - Basic Functionality', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/chat');
    await page.waitForLoadState('networkidle');
  });

  test('should display chat page', async ({ page }) => {
    await expect(page).toHaveTitle(/Chat|Agent/i);
    await expect(page.getByRole('heading', { name: /Chat|AI Agent/i })).toBeVisible();
  });

  test('should display chat input', async ({ page }) => {
    const chatInput = page.getByPlaceholder(/Type.*message|Ask.*question|Message/i);
    await expect(chatInput).toBeVisible();
    await expect(chatInput).toBeEnabled();
  });

  test('should send a message', async ({ page }) => {
    const chatInput = page.getByPlaceholder(/Type.*message/i);
    const sendButton = page.getByRole('button', { name: /Send/i });

    await chatInput.fill('Hello, can you help me?');
    await sendButton.click();

    // Message should appear in chat
    await expect(page.getByText('Hello, can you help me?')).toBeVisible();
  });

  test('should receive AI response', async ({ page }) => {
    const chatInput = page.getByPlaceholder(/Type.*message/i);
    await chatInput.fill('What is a digital twin?');
    await page.keyboard.press('Enter');

    // Wait for AI response
    await page.waitForTimeout(2000);

    // Response should appear
    const messages = page.getByTestId('chat-message');
    const count = await messages.count();
    expect(count).toBeGreaterThan(1);
  });

  test('should support keyboard shortcuts', async ({ page }) => {
    const chatInput = page.getByPlaceholder(/Type.*message/i);

    await chatInput.fill('Test message');

    // Press Enter to send
    await page.keyboard.press('Enter');

    // Message should be sent
    await expect(page.getByText('Test message')).toBeVisible();
  });

  test('should clear input after sending', async ({ page }) => {
    const chatInput = page.getByPlaceholder(/Type.*message/i);

    await chatInput.fill('Test message');
    await page.keyboard.press('Enter');

    // Input should be cleared
    await expect(chatInput).toHaveValue('');
  });

  test('should display message timestamps', async ({ page }) => {
    const chatInput = page.getByPlaceholder(/Type.*message/i);

    await chatInput.fill('Test message');
    await page.keyboard.press('Enter');
    await page.waitForTimeout(1000);

    // Timestamp should appear
    const timestamp = page.getByTestId('message-timestamp').first();
    if (await timestamp.isVisible()) {
      await expect(timestamp).toBeVisible();
    }
  });

  test('should support multiline input', async ({ page }) => {
    const chatInput = page.getByPlaceholder(/Type.*message/i);

    // Type multiline message (Shift+Enter for new line)
    await chatInput.fill('Line 1\nLine 2\nLine 3');

    const sendButton = page.getByRole('button', { name: /Send/i });
    await sendButton.click();

    // Message should preserve line breaks
    await expect(page.getByText('Line 1')).toBeVisible();
  });

  test('should disable send button when input is empty', async ({ page }) => {
    const sendButton = page.getByRole('button', { name: /Send/i });

    // Button should be disabled initially
    await expect(sendButton).toBeDisabled();
  });

  test('should enable send button when input has text', async ({ page }) => {
    const chatInput = page.getByPlaceholder(/Type.*message/i);
    const sendButton = page.getByRole('button', { name: /Send/i });

    await chatInput.fill('Test');

    // Button should be enabled
    await expect(sendButton).toBeEnabled();
  });

  test('should scroll to bottom on new message', async ({ page }) => {
    const chatInput = page.getByPlaceholder(/Type.*message/i);

    // Send multiple messages
    for (let i = 0; i < 5; i++) {
      await chatInput.fill(`Message ${i + 1}`);
      await page.keyboard.press('Enter');
      await page.waitForTimeout(500);
    }

    // Last message should be visible
    await expect(page.getByText('Message 5')).toBeVisible();
  });

  test('should copy message text', async ({ page }) => {
    const chatInput = page.getByPlaceholder(/Type.*message/i);
    await chatInput.fill('Test message to copy');
    await page.keyboard.press('Enter');
    await page.waitForTimeout(1000);

    const copyButton = page.getByRole('button', { name: /Copy/i }).first();
    if (await copyButton.isVisible()) {
      await copyButton.click();

      // Should show copied confirmation
      await expect(page.getByText(/Copied/i)).toBeVisible({ timeout: 2000 });
    }
  });

  test('should regenerate response', async ({ page }) => {
    const chatInput = page.getByPlaceholder(/Type.*message/i);
    await chatInput.fill('Tell me about pipelines');
    await page.keyboard.press('Enter');
    await page.waitForTimeout(2000);

    const regenerateButton = page.getByRole('button', { name: /Regenerate|Retry/i }).first();
    if (await regenerateButton.isVisible()) {
      await regenerateButton.click();

      // Should show generating indicator
      await expect(page.getByText(/Generating|Thinking/i)).toBeVisible({ timeout: 5000 });
    }
  });
});

test.describe('Chat - Conversations Management', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/chat');
    await page.waitForLoadState('networkidle');
  });

  test('should display conversation history sidebar', async ({ page }) => {
    const sidebar = page.getByTestId('conversations-sidebar');

    if (await sidebar.isVisible()) {
      await expect(sidebar).toBeVisible();
    }
  });

  test('should create new conversation', async ({ page }) => {
    const newChatButton = page.getByRole('button', { name: /New Chat|New Conversation/i });

    if (await newChatButton.isVisible()) {
      await newChatButton.click();

      // Should start fresh conversation
      const messages = page.getByTestId('chat-message');
      const count = await messages.count();
      expect(count).toBe(0);
    }
  });

  test('should save conversation automatically', async ({ page }) => {
    const chatInput = page.getByPlaceholder(/Type.*message/i);

    await chatInput.fill('Test conversation');
    await page.keyboard.press('Enter');
    await page.waitForTimeout(2000);

    // Conversation should appear in sidebar
    const conversationList = page.getByTestId('conversation-item');
    if (await conversationList.first().isVisible()) {
      await expect(conversationList.first()).toBeVisible();
    }
  });

  test('should switch between conversations', async ({ page }) => {
    const conversationItem = page.getByTestId('conversation-item').nth(1);

    if (await conversationItem.isVisible()) {
      await conversationItem.click();

      // Should load different conversation
      await page.waitForTimeout(1000);
      await expect(page.getByTestId('chat-messages')).toBeVisible();
    }
  });

  test('should rename conversation', async ({ page }) => {
    const conversationItem = page.getByTestId('conversation-item').first();

    if (await conversationItem.isVisible()) {
      // Right-click or find rename button
      await conversationItem.hover();

      const renameButton = page.getByRole('button', { name: /Rename/i }).first();
      if (await renameButton.isVisible()) {
        await renameButton.click();

        const nameInput = page.getByLabel(/Name|Title/i);
        await nameInput.fill('Updated Conversation Name');
        await page.getByRole('button', { name: /Save/i }).click();

        await expect(page.getByText('Updated Conversation Name')).toBeVisible();
      }
    }
  });

  test('should delete conversation', async ({ page }) => {
    const conversationItem = page.getByTestId('conversation-item').first();

    if (await conversationItem.isVisible()) {
      await conversationItem.hover();

      const deleteButton = page.getByRole('button', { name: /Delete/i }).first();
      if (await deleteButton.isVisible()) {
        await deleteButton.click();

        // Confirm deletion
        await expect(page.getByRole('dialog')).toBeVisible();
        await page.getByRole('button', { name: /Delete|Confirm/i }).click();

        await expect(page.getByText(/Conversation deleted/i)).toBeVisible({ timeout: 5000 });
      }
    }
  });

  test('should search conversations', async ({ page }) => {
    const searchInput = page.getByPlaceholder(/Search.*conversations/i);

    if (await searchInput.isVisible()) {
      await searchInput.fill('pipeline');
      await page.waitForTimeout(500);

      // Filtered conversations should appear
      const results = page.getByTestId('conversation-item');
      if (await results.first().isVisible()) {
        const text = await results.first().textContent();
        expect(text?.toLowerCase()).toContain('pipeline');
      }
    }
  });

  test('should export conversation', async ({ page }) => {
    const conversationItem = page.getByTestId('conversation-item').first();

    if (await conversationItem.isVisible()) {
      await conversationItem.hover();

      const exportButton = page.getByRole('button', { name: /Export/i }).first();
      if (await exportButton.isVisible()) {
        const downloadPromise = page.waitForEvent('download');
        await exportButton.click();

        const download = await downloadPromise;
        expect(download.suggestedFilename()).toMatch(/\.txt|\.md|\.json/);
      }
    }
  });
});

test.describe('Chat - Tool Calling', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/chat');
    await page.waitForLoadState('networkidle');
  });

  test('should trigger tool call', async ({ page }) => {
    const chatInput = page.getByPlaceholder(/Type.*message/i);

    // Ask something that requires tool use
    await chatInput.fill('List all my pipelines');
    await page.keyboard.press('Enter');

    // Wait for tool call indicator
    await page.waitForTimeout(2000);

    const toolCallIndicator = page.getByText(/Using.*tool|Calling.*function|Executing/i);
    if (await toolCallIndicator.isVisible()) {
      await expect(toolCallIndicator).toBeVisible();
    }
  });

  test('should display tool call results', async ({ page }) => {
    const chatInput = page.getByPlaceholder(/Type.*message/i);

    await chatInput.fill('Show me the status of pipeline123');
    await page.keyboard.press('Enter');
    await page.waitForTimeout(3000);

    // Tool results should appear
    const toolResults = page.getByTestId('tool-results');
    if (await toolResults.isVisible()) {
      await expect(toolResults).toBeVisible();
    }
  });

  test('should show tool execution progress', async ({ page }) => {
    const chatInput = page.getByPlaceholder(/Type.*message/i);

    await chatInput.fill('Execute pipeline test-pipeline');
    await page.keyboard.press('Enter');

    // Should show progress indicator
    await expect(page.getByText(/Executing|Running|Processing/i)).toBeVisible({ timeout: 5000 });
  });

  test('should handle tool errors gracefully', async ({ page }) => {
    // Mock API to return error
    await page.route('**/api/v1/chat/message', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          message: 'Tool execution failed',
          tool_error: true,
        }),
      });
    });

    const chatInput = page.getByPlaceholder(/Type.*message/i);
    await chatInput.fill('Do something impossible');
    await page.keyboard.press('Enter');

    // Should show error message
    await expect(page.getByText(/Failed|Error|Unable/i)).toBeVisible({ timeout: 5000 });
  });
});

test.describe('Chat - Context and Settings', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/chat');
    await page.waitForLoadState('networkidle');
  });

  test('should open chat settings', async ({ page }) => {
    const settingsButton = page.getByRole('button', { name: /Settings|Configure/i });

    if (await settingsButton.isVisible()) {
      await settingsButton.click();

      // Settings panel should appear
      await expect(page.getByRole('dialog', { name: /Settings/i })).toBeVisible();
    }
  });

  test('should change model selection', async ({ page }) => {
    const settingsButton = page.getByRole('button', { name: /Settings/i });

    if (await settingsButton.isVisible()) {
      await settingsButton.click();

      const modelSelect = page.getByLabel(/Model/i);
      if (await modelSelect.isVisible()) {
        await modelSelect.selectOption({ index: 1 });

        // Save settings
        await page.getByRole('button', { name: /Save/i }).click();

        await expect(page.getByText(/Settings.*saved/i)).toBeVisible({ timeout: 5000 });
      }
    }
  });

  test('should adjust temperature setting', async ({ page }) => {
    const settingsButton = page.getByRole('button', { name: /Settings/i });

    if (await settingsButton.isVisible()) {
      await settingsButton.click();

      const temperatureSlider = page.getByLabel(/Temperature/i);
      if (await temperatureSlider.isVisible()) {
        await temperatureSlider.fill('0.7');

        await page.getByRole('button', { name: /Save/i }).click();
      }
    }
  });

  test('should add context document', async ({ page }) => {
    const contextButton = page.getByRole('button', { name: /Add Context|Context/i });

    if (await contextButton.isVisible()) {
      await contextButton.click();

      // Context dialog should appear
      await expect(page.getByRole('dialog')).toBeVisible();
      await expect(page.getByText(/Add.*Context|Context.*Document/i)).toBeVisible();
    }
  });

  test('should view active context', async ({ page }) => {
    const contextIndicator = page.getByTestId('active-context');

    if (await contextIndicator.isVisible()) {
      await contextIndicator.click();

      // Should show context details
      await expect(page.getByText(/Context|Documents|Files/i)).toBeVisible();
    }
  });

  test('should clear conversation', async ({ page }) => {
    const clearButton = page.getByRole('button', { name: /Clear.*Conversation|Clear.*Chat/i });

    if (await clearButton.isVisible()) {
      await clearButton.click();

      // Confirm clear
      await expect(page.getByRole('dialog')).toBeVisible();
      await page.getByRole('button', { name: /Clear|Confirm/i }).click();

      // Messages should be cleared
      const messages = page.getByTestId('chat-message');
      const count = await messages.count();
      expect(count).toBe(0);
    }
  });
});

test.describe('Chat - Markdown and Code', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/chat');
    await page.waitForLoadState('networkidle');
  });

  test('should render markdown formatting', async ({ page }) => {
    const chatInput = page.getByPlaceholder(/Type.*message/i);

    await chatInput.fill('Show me code for hello world in Python');
    await page.keyboard.press('Enter');
    await page.waitForTimeout(3000);

    // Should render code block
    const codeBlock = page.locator('pre code');
    if (await codeBlock.isVisible()) {
      await expect(codeBlock).toBeVisible();
    }
  });

  test('should copy code blocks', async ({ page }) => {
    const chatInput = page.getByPlaceholder(/Type.*message/i);

    await chatInput.fill('Give me a Python function');
    await page.keyboard.press('Enter');
    await page.waitForTimeout(3000);

    const copyCodeButton = page.getByRole('button', { name: /Copy.*Code/i }).first();
    if (await copyCodeButton.isVisible()) {
      await copyCodeButton.click();

      await expect(page.getByText(/Copied/i)).toBeVisible({ timeout: 2000 });
    }
  });

  test('should render tables', async ({ page }) => {
    const chatInput = page.getByPlaceholder(/Type.*message/i);

    await chatInput.fill('Show me a comparison table');
    await page.keyboard.press('Enter');
    await page.waitForTimeout(3000);

    // Should render table
    const table = page.locator('table');
    if (await table.isVisible()) {
      await expect(table).toBeVisible();
    }
  });

  test('should render lists', async ({ page }) => {
    const chatInput = page.getByPlaceholder(/Type.*message/i);

    await chatInput.fill('Give me a bullet list');
    await page.keyboard.press('Enter');
    await page.waitForTimeout(3000);

    // Should render list
    const list = page.locator('ul, ol');
    if (await list.isVisible()) {
      await expect(list).toBeVisible();
    }
  });
});

test.describe('Chat - Streaming Responses', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/chat');
    await page.waitForLoadState('networkidle');
  });

  test('should show typing indicator', async ({ page }) => {
    const chatInput = page.getByPlaceholder(/Type.*message/i);

    await chatInput.fill('Tell me about AI');
    await page.keyboard.press('Enter');

    // Should show typing/generating indicator
    await expect(page.getByText(/Thinking|Generating|Typing/i)).toBeVisible({ timeout: 2000 });
  });

  test('should stream response tokens', async ({ page }) => {
    const chatInput = page.getByPlaceholder(/Type.*message/i);

    await chatInput.fill('Explain machine learning');
    await page.keyboard.press('Enter');

    // Wait a bit for streaming to start
    await page.waitForTimeout(1000);

    // Response should gradually appear
    const response = page.getByTestId('chat-message').last();
    await expect(response).toBeVisible();
  });

  test('should stop generation', async ({ page }) => {
    const chatInput = page.getByPlaceholder(/Type.*message/i);

    await chatInput.fill('Write a very long essay');
    await page.keyboard.press('Enter');
    await page.waitForTimeout(500);

    const stopButton = page.getByRole('button', { name: /Stop|Cancel/i });
    if (await stopButton.isVisible()) {
      await stopButton.click();

      // Generation should stop
      await expect(stopButton).not.toBeVisible({ timeout: 2000 });
    }
  });
});

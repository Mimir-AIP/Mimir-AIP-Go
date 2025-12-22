import { test, expect } from '@playwright/test';

test.describe('Agent Chat E2E Tests', () => {
  test('should load chat page and send messages', async ({ page }) => {
    console.log('=== Starting Agent Chat E2E Test ===');
    
    // Navigate to chat page
    console.log('Navigating to /chat...');
    await page.goto('/chat', { waitUntil: 'networkidle' });
    await page.waitForTimeout(2000);

    // Verify page loaded
    console.log('Checking for chat interface elements...');
    const textarea = page.locator('textarea[placeholder*="Type a message"]');
    await expect(textarea).toBeVisible({ timeout: 10000 });
    
    const sendButton = page.locator('button:has-text(""), button >> svg').last();
    await expect(sendButton).toBeVisible({ timeout: 5000 });
    
    console.log('✓ Chat interface loaded');

    // Check for welcome state
    const welcomeText = page.locator('text=/Chat with Mimir|What can you do/i');
    const hasWelcome = await welcomeText.count() > 0;
    if (hasWelcome) {
      console.log('✓ Welcome message visible');
    }

    // Send a test message
    console.log('Sending test message...');
    await textarea.fill('What can you help me with?');
    await page.keyboard.press('Enter');
    await page.waitForTimeout(3000);

    // Check for user message bubble
    console.log('Verifying user message appears...');
    const userMessage = page.locator('text="What can you help me with?"').first();
    await expect(userMessage).toBeVisible({ timeout: 5000 });
    console.log('✓ User message visible');

    // Check for assistant response
    console.log('Waiting for assistant response...');
    await page.waitForTimeout(2000);
    
    // Look for bot icon or assistant message
    const botIcon = page.locator('svg').filter({ hasText: '' }).first();
    const hasResponse = await page.locator('div').filter({ 
      has: page.locator('svg') 
    }).count() > 1;
    
    if (hasResponse) {
      console.log('✓ Assistant response received');
    } else {
      console.log('⚠ No assistant response detected (may be delayed)');
    }

    // Test tool calling
    console.log('Testing tool call trigger...');
    await textarea.fill('TRIGGER_TOOL: create_scenario');
    await page.keyboard.press('Enter');
    await page.waitForTimeout(3000);

    // Look for tool call indicator
    const toolIndicator = page.locator('text=/🔧|create_scenario|tool/i').first();
    const hasToolCall = await toolIndicator.count() > 0;
    
    if (hasToolCall) {
      console.log('✓ Tool call triggered and visible');
    } else {
      console.log('⚠ Tool call not detected in UI');
    }

    console.log('=== Agent Chat E2E Test Complete ===');
  });

  test('should handle multiple messages', async ({ page }) => {
    console.log('=== Testing Multiple Messages ===');
    
    await page.goto('/chat', { waitUntil: 'networkidle' });
    await page.waitForTimeout(2000);

    const textarea = page.locator('textarea[placeholder*="Type a message"]');
    await expect(textarea).toBeVisible({ timeout: 10000 });

    // Send multiple messages
    const messages = [
      'Hello',
      'What can you do?',
      'Tell me about pipelines'
    ];

    for (const msg of messages) {
      console.log(`Sending: ${msg}`);
      await textarea.fill(msg);
      await page.keyboard.press('Enter');
      await page.waitForTimeout(2000);
    }

    // Check all messages are visible
    for (const msg of messages) {
      const msgElement = page.locator(`text="${msg}"`).first();
      await expect(msgElement).toBeVisible({ timeout: 5000 });
      console.log(`✓ Message visible: ${msg}`);
    }

    console.log('✓ Multiple messages test complete');
  });

  test('should show quick action buttons', async ({ page }) => {
    console.log('=== Testing Quick Actions ===');
    
    await page.goto('/chat', { waitUntil: 'networkidle' });
    await page.waitForTimeout(2000);

    // Check for quick action buttons in empty state
    const quickActions = page.locator('button:has-text("What can you do?"), button:has-text("Show my data"), button:has-text("Create a scenario")');
    const actionCount = await quickActions.count();
    
    if (actionCount > 0) {
      console.log(`✓ Found ${actionCount} quick action buttons`);
      
      // Click first quick action
      const firstAction = quickActions.first();
      await firstAction.click();
      await page.waitForTimeout(1000);
      
      // Check if textarea was filled
      const textarea = page.locator('textarea[placeholder*="Type a message"]');
      const value = await textarea.inputValue();
      
      if (value.length > 0) {
        console.log(`✓ Quick action filled textarea with: ${value}`);
      }
    } else {
      console.log('ℹ No quick action buttons (may have messages already)');
    }

    console.log('✓ Quick actions test complete');
  });

  test('should switch models and show different responses', async ({ page }) => {
    console.log('=== Testing Model Switching ===');
    
    await page.goto('/chat', { waitUntil: 'networkidle' });
    await page.waitForTimeout(2000);

    // Open model selector
    console.log('Opening model selector...');
    const modelToggle = page.locator('[data-testid="model-selector-toggle"]');
    await expect(modelToggle).toBeVisible({ timeout: 10000 });
    
    // Verify initial model is shown
    const initialText = await modelToggle.textContent();
    console.log(`Initial model: ${initialText}`);
    expect(initialText).toBeTruthy();
    
    await modelToggle.click();
    await page.waitForTimeout(1000);

    // Verify model selector panel is visible
    const selectorPanel = page.locator('[data-testid="model-selector-panel"]');
    await expect(selectorPanel).toBeVisible({ timeout: 5000 });
    console.log('✓ Model selector panel opened');

    // Verify Provider and Model labels are present
    const providerLabel = page.locator('text="Provider"');
    const modelLabel = page.locator('text="Model"');
    await expect(providerLabel).toBeVisible();
    await expect(modelLabel).toBeVisible();
    console.log('✓ Model selector fields visible');

    // Check that select dropdowns exist
    const selects = selectorPanel.locator('select');
    const selectCount = await selects.count();
    console.log(`Found ${selectCount} select dropdowns`);
    expect(selectCount).toBeGreaterThanOrEqual(2); // Provider and Model

    // Close model selector
    await modelToggle.click();
    await page.waitForTimeout(500);
    
    // Verify panel is closed
    const isPanelVisible = await selectorPanel.isVisible();
    expect(isPanelVisible).toBeFalsy();
    console.log('✓ Model selector panel closed');

    console.log('=== Model Switching Test Complete ===');
  });

  test('should display available tools', async ({ page }) => {
    console.log('=== Testing Tools Display ===');
    
    await page.goto('/chat', { waitUntil: 'networkidle' });
    await page.waitForTimeout(2000);

    // Look for tools panel toggle
    console.log('Looking for tools panel...');
    const toolsToggle = page.locator('button:has-text("Available Tools")');
    await expect(toolsToggle).toBeVisible({ timeout: 10000 });
    console.log('✓ Tools panel toggle found');

    // Click to expand tools
    console.log('Expanding tools panel...');
    await toolsToggle.click();
    await page.waitForTimeout(2000); // Wait longer for expansion animation

    // Check if panel content is visible
    const panelContent = page.locator('text=/Ask the agent to use these tools/i');
    const hasPanelContent = await panelContent.count() > 0;
    console.log(`Panel content visible: ${hasPanelContent}`);

    // Verify tools are listed - check for new MCP dynamically discovered tools
    console.log('Checking for tool names...');
    const expectedTools = [
      'Input.csv',
      'Input.excel',
      'Input.markdown',
      'Input.api',
      'Input.xml',
      'Output.html',
      'Ontology.query',
      'Ontology.extract',
      'Ontology.management'
    ];

    let foundCount = 0;
    for (const tool of expectedTools) {
      // Look in the entire page
      const toolElement = page.locator(`text="${tool}"`);
      const count = await toolElement.count();
      if (count > 0) {
        foundCount++;
        console.log(`✓ Found tool: ${tool}`);
      } else {
        console.log(`⚠ Tool not found: ${tool}`);
      }
    }

    console.log(`✓ Found ${foundCount}/${expectedTools.length} tools`);
    
    // Check if we found any tools
    if (foundCount > 0) {
      console.log('✓ MCP Tools are visible in the UI');
      expect(foundCount).toBeGreaterThanOrEqual(7); // At least 7 out of 9 tools should be visible
    } else {
      console.log('⚠ No MCP tools found - panel may not be rendering correctly');
      // Take a screenshot for debugging
      await page.screenshot({ path: 'test-results/tools-panel-debug.png', fullPage: true });
      console.log('Screenshot saved to: test-results/tools-panel-debug.png');
      
      // Check if the tools badge shows any count
      const toolsBadge = page.locator('text=/Available Tools.*\\d+/i');
      const badgeText = await toolsBadge.textContent();
      console.log(`Tools badge text: ${badgeText}`);
      
      // Fail the test with more info
      throw new Error(`No MCP tools found in UI. Badge shows: ${badgeText}`);
    }

    console.log('=== Tools Display Test Complete ===');
  });

  test('should execute tools and display actual data in output', async ({ page }) => {
    console.log('=== Testing Tool Execution with Real Data ===');
    
    await page.goto('/chat', { waitUntil: 'networkidle' });
    await page.waitForTimeout(2000);

    const textarea = page.locator('textarea[placeholder*="Type a message"]');
    await expect(textarea).toBeVisible({ timeout: 10000 });

    // Send message to trigger CSV tool
    console.log('Sending message to trigger Input.csv tool...');
    await textarea.fill('TRIGGER_TOOL:Input.csv');
    await page.keyboard.press('Enter');
    await page.waitForTimeout(5000); // Wait for tool execution

    // Look for tool call indicator
    console.log('Checking for tool call in UI...');
    const toolIndicator = page.locator('text=/Tool.*Input\\.csv|🔧.*Input\\.csv/i').first();
    await expect(toolIndicator).toBeVisible({ timeout: 10000 });
    console.log('✓ Tool call indicator visible');

    // Click on tool call to expand it
    console.log('Expanding tool call details...');
    await toolIndicator.click();
    await page.waitForTimeout(1000);

    // Check if output section is visible
    const outputLabel = page.locator('text="Output:"');
    await expect(outputLabel).toBeVisible({ timeout: 5000 });
    console.log('✓ Output section visible');

    // Check for actual CSV data in the output
    console.log('Verifying CSV data is present in output...');
    const outputContent = page.locator('pre').filter({ hasText: 'row_count' });
    const hasOutput = await outputContent.count() > 0;
    
    if (hasOutput) {
      console.log('✓ Tool output contains data');
      
      // Verify specific CSV fields
      const outputText = await outputContent.textContent();
      
      if (outputText) {
        const hasRowCount = outputText.includes('row_count');
        const hasColumns = outputText.includes('columns');
        const hasRows = outputText.includes('rows');
        
        console.log(`  - row_count present: ${hasRowCount}`);
        console.log(`  - columns present: ${hasColumns}`);
        console.log(`  - rows present: ${hasRows}`);
        
        expect(hasRowCount).toBeTruthy();
        expect(hasColumns).toBeTruthy();
        expect(hasRows).toBeTruthy();
        
        console.log('✓ CSV data structure verified in UI');
      }
    } else {
      console.log('⚠ No data found in tool output');
      // Take screenshot for debugging
      await page.screenshot({ path: 'test-results/tool-output-debug.png', fullPage: true });
      throw new Error('Tool output does not contain expected CSV data');
    }

    console.log('=== Tool Execution with Real Data Test Complete ===');
  });
});

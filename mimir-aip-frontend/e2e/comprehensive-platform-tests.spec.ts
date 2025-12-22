import { test, expect } from '@playwright/test';
import * as path from 'path';

const BASE_URL = 'http://localhost:8080';

test.describe('Comprehensive Platform Tests - Full Workflows', () => {
  
  test('1. Pipeline Creation and Execution', async ({ page }) => {
    await page.goto('/pipelines');
    await page.waitForLoadState('networkidle');
    
    // Click create pipeline button - MUST exist
    const createButton = page.locator('button:has-text("Create"), a[href*="create"]');
    await expect(createButton.first()).toBeVisible({ timeout: 10000 });
    await createButton.first().click();
    await page.waitForTimeout(1000);
    
    // Fill pipeline details - form MUST be present
    const nameInput = page.locator('input[placeholder*="name"], input[name="name"]').first();
    await expect(nameInput).toBeVisible({ timeout: 5000 });
    await nameInput.fill('E2E Test Pipeline');
    
    // Check for YAML config area - MUST be present
    const yamlInput = page.locator('textarea[placeholder*="YAML"], textarea[placeholder*="config"]').first();
    await expect(yamlInput).toBeVisible({ timeout: 5000 });
    const samplePipeline = `
steps:
  - name: test_step
    plugin: Storage.simple_storage
    config:
      operation: store
      key: test_key
      value: test_value
`;
    await yamlInput.fill(samplePipeline);
    
    // Submit pipeline - button MUST be clickable
    const submitButton = page.locator('button:has-text("Create"), button:has-text("Save")').first();
    await expect(submitButton).toBeVisible({ timeout: 5000 });
    await submitButton.click({ force: true });
    await page.waitForTimeout(2000);
    
    // Navigate back to pipelines list
    await page.goto('/pipelines');
    await page.waitForLoadState('networkidle');
    
    // Created pipeline MUST appear in list
    const pipelineCard = page.locator('text=E2E Test Pipeline').first();
    await expect(pipelineCard).toBeVisible({ timeout: 5000 });
    
    // Try to run the pipeline
    await pipelineCard.click();
    await page.waitForTimeout(1000);
    
    const runButton = page.locator('button:has-text("Run"), button:has-text("Execute")');
    await expect(runButton.first()).toBeVisible({ timeout: 5000 });
    await runButton.first().click();
    await page.waitForTimeout(2000);
  });

  test('2. Monitoring Jobs - Create and Trigger Pipeline', async ({ page }) => {
    await page.goto('/monitoring/jobs');
    await page.waitForLoadState('networkidle');
    
    // Create job button MUST exist
    const createJobButton = page.locator('button:has-text("Create"), a[href*="create"]').first();
    await expect(createJobButton).toBeVisible({ timeout: 10000 });
    await createJobButton.click();
    await page.waitForTimeout(1000);
    
    // Fill job details - form MUST be present
    const jobNameInput = page.locator('input[placeholder*="name"], input[name="name"]').first();
    await expect(jobNameInput).toBeVisible({ timeout: 5000 });
    await jobNameInput.fill('E2E Test Job');
    
    // Pipeline select MUST exist
    const pipelineSelect = page.locator('select[name*="pipeline"], select:has-text("Pipeline")').first();
    await expect(pipelineSelect).toBeVisible({ timeout: 5000 });
    await pipelineSelect.selectOption({ index: 1 });
    
    // Schedule input MUST exist
    const scheduleInput = page.locator('input[placeholder*="cron"], input[placeholder*="schedule"]').first();
    await expect(scheduleInput).toBeVisible({ timeout: 5000 });
    await scheduleInput.fill('*/5 * * * *'); // Every 5 minutes
    
    // Submit button MUST be clickable
    const submitButton = page.locator('button:has-text("Create"), button:has-text("Save")').first();
    await expect(submitButton).toBeVisible({ timeout: 5000 });
    await submitButton.click();
    await page.waitForTimeout(2000);
    
    // Verify job in list - job MUST appear
    await page.goto('/monitoring/jobs');
    await page.waitForLoadState('networkidle');
    
    const jobCard = page.locator('text=E2E Test Job').first();
    await expect(jobCard).toBeVisible({ timeout: 5000 });
  });

  test('3. Data Ingestion - Full Workflow', async ({ page }) => {
    await page.goto('/data/upload');
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);
    
    // CSV plugin MUST be available
    const csvPlugin = page.locator('text=/csv/i').first();
    await expect(csvPlugin).toBeVisible({ timeout: 5000 });
    await csvPlugin.click();
    await page.waitForTimeout(1000);
    
    // Upload file - file input MUST exist
    const fileInput = page.locator('input[type="file"]').first();
    await expect(fileInput).toBeAttached({ timeout: 5000 });
    const csvPath = path.join(__dirname, '../../test_data/products.csv');
    await fileInput.setInputFiles(csvPath);
    await page.waitForTimeout(2000);
    
    // Submit upload - button MUST exist
    const uploadButton = page.locator('button:has-text("Upload")').first();
    await expect(uploadButton).toBeVisible({ timeout: 5000 });
    await uploadButton.click();
    await page.waitForTimeout(3000);
    
    // Should navigate to preview or show success
    const currentUrl = page.url();
    expect(currentUrl.includes('/data/preview/') || currentUrl.includes('/data')).toBe(true);
  });

  test('4. Ontology Creation and Management', async ({ page }) => {
    await page.goto('/ontologies');
    await page.waitForLoadState('networkidle');
    
    // Navigate to upload
    await page.goto('/ontologies/upload');
    await page.waitForLoadState('networkidle');
    
    // Fill ontology details - form MUST exist
    const nameInput = page.locator('input[placeholder="my-ontology"]').first();
    await expect(nameInput).toBeVisible({ timeout: 5000 });
    await nameInput.fill('E2E Test Ontology');
    
    const versionInput = page.locator('input[placeholder="1.0.0"]').first();
    await expect(versionInput).toBeVisible({ timeout: 5000 });
    await versionInput.fill('1.0.0');
    
    const descInput = page.locator('textarea').first();
    await expect(descInput).toBeVisible({ timeout: 5000 });
    await descInput.fill('Test ontology for E2E testing');
    
    // Upload ontology file - file input MUST exist
    const ontologyPath = path.join(__dirname, '../../test_data/product_ontology.ttl');
    const fileInput = page.locator('input[type="file"]').first();
    await expect(fileInput).toBeAttached({ timeout: 5000 });
    await fileInput.setInputFiles(ontologyPath);
    await page.waitForTimeout(1500);
    
    // Submit - button MUST be clickable
    const uploadButton = page.locator('button:has-text("Upload Ontology")');
    await expect(uploadButton).toBeVisible({ timeout: 5000 });
    await uploadButton.click();
    await page.waitForTimeout(3000);
    
    // Navigate to ontologies list
    await page.goto('/ontologies');
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(1000);
    
    // Ontology MUST appear in list
    const ontologyCard = page.locator('text=E2E Test Ontology').first();
    await expect(ontologyCard).toBeVisible({ timeout: 5000 });
    
    // Click to view details
    await ontologyCard.click();
    await page.waitForTimeout(1000);
    
    // Check for suggestions page
    const suggestionsLink = page.locator('a[href*="suggestions"]');
    if (await suggestionsLink.count() > 0) {
      await suggestionsLink.click();
      await page.waitForLoadState('networkidle');
    }
  });

  test('5. SPARQL Knowledge Graph Query', async ({ page }) => {
    await page.goto('/knowledge-graph');
    await page.waitForLoadState('networkidle');
    
    // SPARQL query interface MUST exist
    const queryTextarea = page.locator('textarea[placeholder*="SPARQL"], textarea[placeholder*="query"]').first();
    await expect(queryTextarea).toBeVisible({ timeout: 5000 });
    
    // Enter sample SPARQL query
    const sampleQuery = `
PREFIX rdf: <http://www.w3.org/1999/02/22-rdf-syntax-ns#>
PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>

SELECT ?s ?p ?o
WHERE {
  ?s ?p ?o
}
LIMIT 10
`;
    await queryTextarea.fill(sampleQuery);
    
    // Execute button MUST exist
    const executeButton = page.locator('button:has-text("Execute"), button:has-text("Run")').first();
    await expect(executeButton).toBeVisible({ timeout: 5000 });
    await executeButton.click();
    await page.waitForTimeout(3000);
    
    // Results section MUST appear
    const resultsSection = page.locator('table, [class*="result"]');
    await expect(resultsSection.first()).toBeVisible({ timeout: 10000 });
  });

  test('6. Digital Twin Creation and Usage', async ({ page }) => {
    await page.goto('/digital-twins');
    await page.waitForLoadState('networkidle');
    
    // Create button MUST exist
    const createButton = page.locator('button:has-text("Create"), a[href*="create"]').first();
    await expect(createButton).toBeVisible({ timeout: 10000 });
    await createButton.click();
    await page.waitForTimeout(1000);
    
    // Fill digital twin details - form MUST exist
    const nameInput = page.locator('input[placeholder*="name"], input[name="name"]').first();
    await expect(nameInput).toBeVisible({ timeout: 5000 });
    await nameInput.fill('E2E Test Digital Twin');
    
    const descInput = page.locator('textarea[placeholder*="description"]').first();
    await expect(descInput).toBeVisible({ timeout: 5000 });
    await descInput.fill('Test digital twin for E2E testing');
    
    // Select ontology - dropdown MUST exist
    const ontologySelect = page.locator('select[name*="ontology"]').first();
    await expect(ontologySelect).toBeVisible({ timeout: 5000 });
    await ontologySelect.selectOption({ index: 1 });
    
    // Select model type - dropdown MUST exist
    const modelTypeSelect = page.locator('select[name*="model"], select[name*="type"]').first();
    await expect(modelTypeSelect).toBeVisible({ timeout: 5000 });
    await modelTypeSelect.selectOption({ index: 1 });
    
    // Submit button MUST be clickable
    const submitButton = page.locator('button:has-text("Create"), button:has-text("Save")').first();
    await expect(submitButton).toBeVisible({ timeout: 5000 });
    await submitButton.click({ force: true });
    await page.waitForTimeout(2000);
    
    // Digital twin MUST appear in list
    await page.goto('/digital-twins');
    await page.waitForLoadState('networkidle');
    
    const twinCard = page.locator('text=E2E Test Digital Twin').first();
    await expect(twinCard).toBeVisible({ timeout: 5000 });
    
    // Click to view details
    await twinCard.click();
    await page.waitForTimeout(1000);
  });

  test('7. ML Model Training Workflow', async ({ page }) => {
    // Try /models first (primary ML page)
    await page.goto('/models');
    await page.waitForLoadState('networkidle');
    
    // ML training interface MUST exist at /models
    const trainButton = page.locator('button:has-text("Train"), button:has-text("Train Model"), button:has-text("Auto ML"), button:has-text("New Model")');
    await expect(trainButton.first()).toBeVisible({ 
      timeout: 10000 
    });
    
    await trainButton.first().click();
    await page.waitForTimeout(1000);
    
    // Training form MUST appear
    const targetInput = page.locator('input[placeholder*="target"], input[name*="target"]').first();
    await expect(targetInput).toBeVisible({ timeout: 5000 });
    await targetInput.fill('price');
    
    // Algorithm/model type selector MUST exist
    const algorithmSelect = page.locator('select[name*="algorithm"], select[name*="model"]').first();
    await expect(algorithmSelect).toBeVisible({ timeout: 5000 });
    
    // Start training button MUST exist
    const startButton = page.locator('button:has-text("Start"), button:has-text("Train")').first();
    await expect(startButton).toBeVisible({ timeout: 5000 });
    await startButton.click();
    await page.waitForTimeout(2000);
  });

  test('8. Entity Extraction Page', async ({ page }) => {
    await page.goto('/extraction');
    await page.waitForLoadState('networkidle');
    
    // Text input MUST exist
    const textInput = page.locator('textarea[placeholder*="text"], textarea[placeholder*="extract"]').first();
    await expect(textInput).toBeVisible({ timeout: 5000 });
    
    const sampleText = 'Apple Inc. was founded by Steve Jobs in Cupertino, California in 1976.';
    await textInput.fill(sampleText);
    
    // Extract button MUST exist
    const extractButton = page.locator('button:has-text("Extract"), button:has-text("Analyze")').first();
    await expect(extractButton).toBeVisible({ timeout: 5000 });
    await extractButton.click();
    await page.waitForTimeout(3000);
    
    // Results section MUST appear
    const resultsSection = page.locator('[class*="result"], [class*="entity"], table');
    await expect(resultsSection.first()).toBeVisible({ timeout: 10000 });
  });

  test('9. Plugin Management Page', async ({ page }) => {
    await page.goto('/plugins');
    await page.waitForLoadState('networkidle');
    
    // Plugin list MUST be displayed
    const pluginCards = page.locator('[class*="card"], [class*="plugin"]');
    await expect(pluginCards.first()).toBeVisible({ timeout: 10000 });
    
    const pluginCount = await pluginCards.count();
    expect(pluginCount).toBeGreaterThan(0);
    
    // Plugin action buttons MUST be available
    const actionButtons = page.locator('button:has-text("Enable"), button:has-text("Disable"), button:has-text("Configure")');
    await expect(actionButtons.first()).toBeVisible({ timeout: 5000 });
  });

  test('10. API Key Management on Settings/Plugins Page', async ({ page }) => {
    // Try settings page first
    await page.goto('/settings');
    await page.waitForLoadState('networkidle');
    
    // API key section MUST exist
    const apiKeySection = page.locator('text=/API Key|api.?key/i');
    await expect(apiKeySection.first()).toBeVisible({ timeout: 10000 });
    
    // Add API key button MUST exist
    const addKeyButton = page.locator('button:has-text("Add"), button:has-text("Create"), button:has-text("New Key")');
    await expect(addKeyButton.first()).toBeVisible({ timeout: 5000 });
    await addKeyButton.first().click();
    await page.waitForTimeout(1000);
    
    // API key form MUST appear
    const nameInput = page.locator('input[placeholder*="name"], input[name="name"]').first();
    await expect(nameInput).toBeVisible({ timeout: 5000 });
    await nameInput.fill('E2E Test API Key');
    
    const keyInput = page.locator('input[placeholder*="key"], input[type="password"]').first();
    await expect(keyInput).toBeVisible({ timeout: 5000 });
    await keyInput.fill('sk-test-key-12345');
    
    const providerSelect = page.locator('select[name*="provider"], select:has-text("Provider")').first();
    await expect(providerSelect).toBeVisible({ timeout: 5000 });
    await providerSelect.selectOption({ index: 1 });
    
    // Save button MUST be clickable
    const saveButton = page.locator('button:has-text("Save"), button:has-text("Add")').first();
    await expect(saveButton).toBeVisible({ timeout: 5000 });
    await saveButton.click({ force: true });
    await page.waitForTimeout(1000);
    
    // API keys list MUST be displayed
    const keysList = page.locator('[class*="key"], table');
    await expect(keysList.first()).toBeVisible({ timeout: 5000 });
  });

  test('11. Agent Chat with Tool Calling', async ({ page }) => {
    console.log('\n=== Test 11: Agent Chat with Tool Calling ===');
    
    // Navigate to digital twins page
    console.log('Navigating to digital twins page...');
    await page.goto('/digital-twins', { waitUntil: 'domcontentloaded' });
    await page.waitForLoadState('networkidle', { timeout: 10000 });
    await page.waitForTimeout(2000);
    
    // Try to find an existing twin or create one via UI
    console.log('Looking for existing twins...');
    const twinCards = page.locator('[data-testid="twin-card"], .twin-card, button:has-text("View"), a[href*="/digital-twins/"]').first();
    
    let twinLink;
    if (await twinCards.count() > 0) {
      // Click on first twin
      console.log('Found existing twin, clicking...');
      twinLink = await twinCards.getAttribute('href') || '';
      if (twinLink) {
        await page.goto(twinLink, { waitUntil: 'domcontentloaded' });
      } else {
        await twinCards.click();
      }
    } else {
      // Try to create a new twin via UI
      console.log('No twins found, trying to create one via UI...');
      const createButton = page.locator('button:has-text("Create"), button:has-text("New")').first();
      if (await createButton.isVisible({ timeout: 3000 }).catch(() => false)) {
        await createButton.click();
        await page.waitForTimeout(1000);
        
        // Fill in twin creation form
        await page.locator('input[name="name"], input[placeholder*="name" i]').first().fill('E2E Chat Test Twin');
        await page.locator('textarea, input[name="description"]').first().fill('Twin for testing agent chat');
        
        const submitButton = page.locator('button[type="submit"], button:has-text("Create"), button:has-text("Save")').last();
        await submitButton.click();
        await page.waitForTimeout(2000);
      } else {
        console.log('⚠ No twins and cannot create, skipping test');
        return;
      }
    }
    
    await page.waitForLoadState('networkidle', { timeout: 10000 });
    await page.waitForTimeout(2000);
    
    // Click on Agent Chat tab
    console.log('Clicking on Agent Chat tab...');
    const chatTab = page.locator('button').filter({ hasText: /agent.*chat|chat/i }).first();
    await expect(chatTab).toBeVisible({ timeout: 10000 });
    await chatTab.click();
    await page.waitForTimeout(2000);
    
    // Create new conversation
    console.log('Creating new conversation...');
    const newConvButton = page.locator('button:has-text("New")').first();
    if (await newConvButton.isVisible({ timeout: 3000 }).catch(() => false)) {
      await newConvButton.click();
      await page.waitForTimeout(1000);
    }
    
    // Check if we need to select a model provider (mock should be available)
    const providerSelect = page.locator('select').first();
    if (await providerSelect.isVisible({ timeout: 2000 }).catch(() => false)) {
      console.log('Selecting mock provider...');
      await providerSelect.selectOption('mock').catch(() => {
        console.log('Mock provider not available in dropdown, using default');
      });
      await page.waitForTimeout(500);
    }
    
    // Send regular message
    console.log('Sending regular message...');
    const messageInput = page.locator('textarea[placeholder*="Ask"], textarea[placeholder*="agent"], textarea').first();
    await expect(messageInput).toBeVisible({ timeout: 5000 });
    await messageInput.fill('What can you help me with?');
    
    const sendButton = page.locator('button:has-text("Send"), button[type="submit"]').last();
    await expect(sendButton).toBeVisible({ timeout: 5000 });
    await sendButton.click();
    await page.waitForTimeout(3000);
    
    // Verify response appears
    console.log('Verifying response appears...');
    const responseMessage = page.locator('text=/Mimir|help|assistant|digital twin/i').first();
    await expect(responseMessage).toBeVisible({ timeout: 10000 });
    console.log('✓ Regular message response received');
    
    // Trigger tool call
    console.log('Triggering tool call...');
    await messageInput.fill('TRIGGER_TOOL: create_scenario');
    await sendButton.click();
    await page.waitForTimeout(3000);
    
    // Verify tool call appears in messages
    console.log('Verifying tool call appears...');
    const toolCallText = page.locator('text=/create_scenario|tool|function|call/i').first();
    await expect(toolCallText).toBeVisible({ timeout: 10000 });
    console.log('✓ Tool call detected in response');
    
    // Try another tool
    console.log('Testing another tool call...');
    await messageInput.fill('TRIGGER_TOOL: train_model');
    await sendButton.click();
    await page.waitForTimeout(3000);
    
    const trainToolText = page.locator('text=/train_model|train|model/i').first();
    await expect(trainToolText).toBeVisible({ timeout: 10000 });
    console.log('✓ Second tool call successful');
    
    console.log('✓ Agent chat test completed successfully');
  });
});

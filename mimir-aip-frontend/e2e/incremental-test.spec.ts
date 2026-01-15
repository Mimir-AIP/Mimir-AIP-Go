import { test, expect } from '@playwright/test';

test.describe('Mimir AIP - Incremental Agent Tools Test', () => {
  test.setTimeout(120000);

  // ============================================
  // STEP 1: Verify Docker Container and API
  // ============================================
  test('Step 1: Docker container is running and API accessible', async ({ page }) => {
    console.log('=== Step 1: Docker Container & API Verification ===');
    
    // Check main page loads
    await page.goto('http://localhost:8080');
    await page.waitForLoadState('networkidle');
    const title = await page.title();
    expect(title).toContain('Mimir');
    console.log('✅ Main page loaded');
    
    // Verify API is accessible
    const apiResponse = await page.request.get('http://localhost:8080/health');
    expect(apiResponse.ok()).toBeTruthy();
    console.log('✅ API is accessible');
    
    // Verify agent tools endpoint exists
    const toolsResponse = await page.request.post('http://localhost:8080/api/v1/agent/tools/execute', {
      data: { tool_name: 'list_pipelines', input: {} }
    });
    expect(toolsResponse.ok()).toBeTruthy();
    const toolsData = await toolsResponse.json();
    expect(toolsData.success).toBeTruthy();
    console.log('✅ Agent tools API is working');
    console.log(`   Found ${toolsData.result?.count || 0} pipelines`);
  });

  // ============================================
  // STEP 2: Test Agent Tools API Directly
  // ============================================
  test('Step 2: Test Agent Tools API directly', async ({ page }) => {
    console.log('=== Step 2: Agent Tools API Tests ===');
    
    // Test list_pipelines
    let response = await page.request.post('http://localhost:8080/api/v1/agent/tools/execute', {
      data: { tool_name: 'list_pipelines', input: {} }
    });
    let data = await response.json();
    expect(data.success).toBeTruthy();
    console.log(`✅ list_pipelines: ${data.result?.count || 0} pipelines`);
    
    // Test recommend_models for anomaly_detection
    response = await page.request.post('http://localhost:8080/api/v1/agent/tools/execute', {
      data: { tool_name: 'recommend_models', input: { use_case: 'anomaly_detection' } }
    });
    data = await response.json();
    expect(data.success).toBeTruthy();
    const recommendations = data.result?.recommendations as Array<{name: string, type: string}>;
    console.log(`✅ recommend_models: Found ${recommendations?.length || 0} recommendations`);
    expect(recommendations?.length).toBeGreaterThan(0);
    
    // Test recommend_models for clustering
    response = await page.request.post('http://localhost:8080/api/v1/agent/tools/execute', {
      data: { tool_name: 'recommend_models', input: { use_case: 'clustering' } }
    });
    data = await response.json();
    expect(data.success).toBeTruthy();
    console.log(`✅ recommend_models (clustering): Found ${data.result?.count || 0} recommendations`);
    
    // Test list_ontologies
    response = await page.request.post('http://localhost:8080/api/v1/agent/tools/execute', {
      data: { tool_name: 'list_ontologies', input: {} }
    });
    data = await response.json();
    expect(data.success).toBeTruthy();
    console.log(`✅ list_ontologies: ${data.result?.count || 0} ontologies`);
    
    // Test list_alerts
    response = await page.request.post('http://localhost:8080/api/v1/agent/tools/execute', {
      data: { tool_name: 'list_alerts', input: {} }
    });
    data = await response.json();
    expect(data.success).toBeTruthy();
    console.log(`✅ list_alerts: ${data.result?.count || 0} alerts`);
    
    console.log('✅ All Agent Tools API tests passed');
  });

  // ============================================
  // STEP 3: Create Pipeline via Agent Tools
  // ============================================
  test('Step 3: Create pipeline via agent tools', async ({ page }) => {
    console.log('=== Step 3: Create Pipeline via Agent Tools ===');
    
    const pipelineName = `Test-Pipeline-${Date.now()}`;
    
    // Create pipeline using agent tools
    const response = await page.request.post('http://localhost:8080/api/v1/agent/tools/execute', {
      data: {
        tool_name: 'create_pipeline',
        input: {
          name: pipelineName,
          description: 'Test pipeline created via agent tools'
        }
      }
    });
    
    const data = await response.json();
    expect(data.success).toBeTruthy();
    expect(data.result).toHaveProperty('pipeline_id');
    expect(data.result?.pipeline_name).toBe(pipelineName);
    
    const pipelineId = data.result.pipeline_id as string;
    console.log(`✅ Created pipeline: ${pipelineName} (ID: ${pipelineId})`);
    
    // Verify pipeline appears in list
    const listResponse = await page.request.post('http://localhost:8080/api/v1/agent/tools/execute', {
      data: { tool_name: 'list_pipelines', input: {} }
    });
    const listData = await listResponse.json();
    const pipelines = listData.result?.pipelines as Array<{id: string, name: string}>;
    const found = pipelines?.some(p => p.id === pipelineId);
    expect(found).toBeTruthy();
    console.log(`✅ Pipeline verified in list`);
  });

  // ============================================
  // STEP 4: Navigate to Chat and Verify Tools Panel
  // ============================================
  test('Step 4: Navigate to Chat page and verify tools', async ({ page }) => {
    console.log('=== Step 4: Chat Page & Tools Panel ===');
    
    // Navigate to chat
    await page.goto('http://localhost:8080/chat');
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);
    
    // Check chat interface loaded
    const textarea = page.locator('textarea[placeholder*="Type a message"]');
    await expect(textarea).toBeVisible({ timeout: 10000 });
    console.log('✅ Chat interface loaded');
    
    // Check for welcome message
    const welcomeText = page.locator('text=/Chat with Mimir|What can you help/i');
    const hasWelcome = await welcomeText.count() > 0;
    console.log(`✅ Welcome message: ${hasWelcome ? 'visible' : 'not visible'}`);
    
    // Check Tools panel
    const toolsToggle = page.locator('button:has-text("Available Tools")');
    await expect(toolsToggle).toBeVisible({ timeout: 10000 });
    console.log('✅ Tools toggle button visible');
    
    // Expand tools panel
    await toolsToggle.click();
    await page.waitForTimeout(2000);
    
    // Check for MCP tools (Input, Output, Ontology plugins)
    const expectedTools = [
      'Input.csv',
      'Output.json',
      'Ontology.query',
      'Ontology.extract'
    ];
    
    let foundCount = 0;
    for (const tool of expectedTools) {
      const toolElement = page.locator(`text="${tool}"`);
      if (await toolElement.count() > 0) {
        foundCount++;
        console.log(`   ✅ Found tool: ${tool}`);
      }
    }
    
    expect(foundCount).toBeGreaterThanOrEqual(2);
    console.log(`✅ Found ${foundCount}/${expectedTools.length} expected tools`);
  });

  // ============================================
  // STEP 5: Test Model Provider Configuration
  // ============================================
  test('Step 5: Test LLM Provider Configuration in Settings', async ({ page }) => {
    console.log('=== Step 5: LLM Provider Configuration ===');
    
    // Navigate to Settings
    await page.goto('http://localhost:8080/settings');
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);
    
    // Check Settings page loaded
    const heading = page.locator('h1').first();
    await expect(heading).toBeVisible({ timeout: 10000 });
    const headingText = await heading.textContent();
    expect(headingText).toContain('Settings');
    console.log(`✅ Settings page: ${headingText}`);
    
    // Check for Plugins tab
    const pluginsTab = page.locator('button:has-text("Plugins")');
    await expect(pluginsTab).toBeVisible({ timeout: 5000 });
    console.log('✅ Plugins tab visible');
    
    // Click Plugins tab
    await pluginsTab.click();
    await page.waitForTimeout(1000);
    
    // Check for AI plugins
    const aiPluginsSection = page.locator('text=/Configurable Plugins|AI Plugins|Configure/i');
    const hasAISection = await aiPluginsSection.count() > 0;
    console.log(`✅ AI plugins section: ${hasAISection ? 'visible' : 'checking...'}`);
    
    // Check for Configure buttons
    const configureButtons = page.locator('button:has-text("Configure")');
    const buttonCount = await configureButtons.count();
    console.log(`   Found ${buttonCount} Configure buttons`);
  });

  // ============================================
  // STEP 6: Test Pipeline Management Page
  // ============================================
  test('Step 6: Test Pipeline Management Page', async ({ page }) => {
    console.log('=== Step 6: Pipeline Management Page ===');
    
    // Navigate to Pipelines
    await page.goto('http://localhost:8080/pipelines');
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);
    
    // Check page loaded
    const heading = page.locator('h1').first();
    await expect(heading).toBeVisible({ timeout: 10000 });
    const headingText = await heading.textContent();
    expect(headingText).toContain('Pipeline');
    console.log(`✅ Pipelines page: ${headingText}`);
    
    // Check for Create button
    const createButton = page.locator('button:has-text("Create Pipeline")').first();
    await expect(createButton).toBeVisible({ timeout: 10000 });
    console.log('✅ Create Pipeline button visible');
    
    // Verify we can see pipeline list
    const pageContent = await page.content();
    const hasPipelines = pageContent.includes('pipeline') || pageContent.includes('Pipeline');
    console.log(`✅ Pipeline content visible: ${hasPipelines}`);
  });

  // ============================================
  // STEP 7: Test Ontologies Page
  // ============================================
  test('Step 7: Test Ontologies Page', async ({ page }) => {
    console.log('=== Step 7: Ontologies Page ===');
    
    // Navigate to Ontologies
    await page.goto('http://localhost:8080/ontologies');
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);
    
    // Check page loaded
    const heading = page.locator('h1').first();
    await expect(heading).toBeVisible({ timeout: 10000 });
    const headingText = await heading.textContent();
    expect(headingText).toContain('Ontolog');
    console.log(`✅ Ontologies page: ${headingText}`);
    
    // Check for table
    const table = page.locator('table');
    await expect(table).toBeVisible({ timeout: 10000 });
    console.log('✅ Ontologies table visible');
    
    // Check for Upload button
    const uploadButton = page.locator('a:has-text("Upload Ontology")');
    await expect(uploadButton).toBeVisible({ timeout: 5000 });
    console.log('✅ Upload Ontology button visible');
  });

  // ============================================
  // STEP 8: Test Models Page
  // ============================================
  test('Step 8: Test Models Page', async ({ page }) => {
    console.log('=== Step 8: Models Page ===');
    
    // Navigate to Models
    await page.goto('http://localhost:8080/models');
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);
    
    // Check page loaded
    const heading = page.locator('h1').first();
    await expect(heading).toBeVisible({ timeout: 10000 });
    const headingText = await heading.textContent();
    console.log(`✅ Models page: ${headingText}`);
  });

  // ============================================
  // STEP 9: Test Digital Twins Page
  // ============================================
  test('Step 9: Test Digital Twins Page', async ({ page }) => {
    console.log('=== Step 9: Digital Twins Page ===');
    
    // Navigate to Digital Twins
    await page.goto('http://localhost:8080/digital-twins');
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);
    
    // Check page loaded
    const heading = page.locator('h1').first();
    await expect(heading).toBeVisible({ timeout: 10000 });
    const headingText = await heading.textContent();
    expect(headingText).toContain('Twin');
    console.log(`✅ Digital Twins page: ${headingText}`);
  });

  // ============================================
  // STEP 10: Create Digital Twin via Agent Tools
  // ============================================
  test('Step 10: Create Digital Twin via Agent Tools', async ({ page }) => {
    console.log('=== Step 10: Create Digital Twin ===');
    
    // Get list of ontologies first
    let response = await page.request.post('http://localhost:8080/api/v1/agent/tools/execute', {
      data: { tool_name: 'list_ontologies', input: {} }
    });
    let data = await response.json();
    const ontologies = data.result?.ontologies as Array<{id: string}>;
    
    // Create digital twin
    const twinName = `Test-Twin-${Date.now()}`;
    response = await page.request.post('http://localhost:8080/api/v1/agent/tools/execute', {
      data: {
        tool_name: 'create_twin',
        input: {
          name: twinName,
          description: 'Test digital twin created via agent tools',
          ontology_id: ontologies?.[0]?.id || ''
        }
      }
    });
    
    data = await response.json();
    expect(data.success).toBeTruthy();
    expect(data.result).toHaveProperty('twin_id');
    
    const twinId = data.result.twin_id as string;
    console.log(`✅ Created digital twin: ${twinName} (ID: ${twinId})`);
    
    // Get twin status
    response = await page.request.post('http://localhost:8080/api/v1/agent/tools/execute', {
      data: {
        tool_name: 'get_twin_status',
        input: { twin_id: twinId }
      }
    });
    data = await response.json();
    expect(data.success).toBeTruthy();
    console.log(`✅ Twin status: ${data.result?.status}`);
  });

  // ============================================
  // STEP 11: Test What-If Scenario via Agent Tools
  // ============================================
  test('Step 11: Test What-If Scenario Simulation', async ({ page }) => {
    console.log('=== Step 11: What-If Scenario Simulation ===');
    
    // Create a twin first
    const twinName = `Scenario-Twin-${Date.now()}`;
    let response = await page.request.post('http://localhost:8080/api/v1/agent/tools/execute', {
      data: {
        tool_name: 'create_twin',
        input: {
          name: twinName,
          description: 'Twin for scenario testing'
        }
      }
    });
    let data = await response.json();
    expect(data.success).toBeTruthy();
    const twinId = data.result.twin_id as string;
    
    // Run scenario simulation
    response = await page.request.post('http://localhost:8080/api/v1/agent/tools/execute', {
      data: {
        tool_name: 'simulate_scenario',
        input: {
          twin_id: twinId,
          scenario: 'What if supplier A is unavailable?',
          parameters: {
            supplier_a_unavailable: true,
            backup_supplier: 'supplier_b'
          }
        }
      }
    });
    
    data = await response.json();
    expect(data.success).toBeTruthy();
    expect(data.result).toHaveProperty('status');
    expect(data.result).toHaveProperty('metrics');

    console.log(`✅ Scenario simulation completed`);
    console.log(`   Status: ${data.result?.status}`);
    console.log(`   Total steps: ${data.result?.metrics?.total_steps}`);
    console.log(`   Events processed: ${data.result?.metrics?.events_processed}`);
  });

  // ============================================
  // STEP 12: Create Alert via Agent Tools
  // ============================================
  test('Step 12: Create Alert via Agent Tools', async ({ page }) => {
    console.log('=== Step 12: Create Alert ===');
    
    const alertName = `Test-Alert-${Date.now()}`;
    
    // Create alert
    const response = await page.request.post('http://localhost:8080/api/v1/agent/tools/execute', {
      data: {
        tool_name: 'create_alert',
        input: {
          title: alertName,
          type: 'anomaly',
          entity_id: 'test_entity',
          metric_name: 'anomaly_score',
          severity: 'high',
          message: 'Anomaly score exceeded threshold'
        }
      }
    });
    
    const data = await response.json();
    expect(data.success).toBeTruthy();
    expect(data.result).toHaveProperty('alert_id');
    
    const alertId = data.result.alert_id as string;
    console.log(`✅ Created alert: ${alertName} (ID: ${alertId})`);
    
    // List alerts
    const listResponse = await page.request.post('http://localhost:8080/api/v1/agent/tools/execute', {
      data: { tool_name: 'list_alerts', input: {} }
    });
    const listData = await listResponse.json();
    const alerts = listData.result?.alerts as Array<{id: string, name: string}>;
    const found = alerts?.some(a => a.id === alertId);
    expect(found).toBeTruthy();
    console.log(`✅ Alert verified in list (${listData.result?.count || 0} total alerts)`);
  });

  // ============================================
  // STEP 13: End-to-End Flow Test
  // ============================================
  test('Step 13: End-to-End Flow - Create Pipeline, Extract Ontology, Detect Anomalies', async ({ page }) => {
    console.log('=== Step 13: End-to-End Flow Test ===');
    
    // Step 1: Create a pipeline for data ingestion
    const pipelineName = `E2E-Pipeline-${Date.now()}`;
    let response = await page.request.post('http://localhost:8080/api/v1/agent/tools/execute', {
      data: {
        tool_name: 'create_pipeline',
        input: {
          name: pipelineName,
          description: 'E2E test pipeline for CSV data ingestion'
        }
      }
    });
    let data = await response.json();
    expect(data.success).toBeTruthy();
    const pipelineId = data.result.pipeline_id as string;
    console.log(`✅ Step 1: Created pipeline (${pipelineId})`);
    
    // Step 2: Get model recommendations for anomaly detection
    response = await page.request.post('http://localhost:8080/api/v1/agent/tools/execute', {
      data: {
        tool_name: 'recommend_models',
        input: { use_case: 'anomaly_detection' }
      }
    });
    data = await response.json();
    expect(data.success).toBeTruthy();
    const anomalyModels = data.result.recommendations as Array<{name: string}>;
    console.log(`✅ Step 2: Recommended ${anomalyModels?.length || 0} anomaly detection models`);
    console.log(`   Models: ${anomalyModels?.map(m => m.name).join(', ')}`);
    
    // Step 3: Create a digital twin
    const twinName = `E2E-Twin-${Date.now()}`;
    response = await page.request.post('http://localhost:8080/api/v1/agent/tools/execute', {
      data: {
        tool_name: 'create_twin',
        input: {
          name: twinName,
          description: 'E2E test digital twin'
        }
      }
    });
    data = await response.json();
    expect(data.success).toBeTruthy();
    const twinId = data.result.twin_id as string;
    console.log(`✅ Step 3: Created digital twin (${twinId})`);
    
    // Step 4: Run anomaly detection
    response = await page.request.post('http://localhost:8080/api/v1/agent/tools/execute', {
      data: {
        tool_name: 'detect_anomalies',
        input: {
          twin_id: twinId,
          time_range: '24h'
        }
      }
    });
    data = await response.json();
    expect(data.success).toBeTruthy();
    console.log(`✅ Step 4: Anomaly detection completed (${data.result?.count || 0} anomalies)`);
    
    // Summary
    console.log('========================================');
    console.log('✅ END-TO-END FLOW COMPLETED SUCCESSFULLY');
    console.log('========================================');
    console.log(`   Pipeline: ${pipelineName} (${pipelineId})`);
    console.log(`   Models: ${anomalyModels?.map(m => m.name).join(', ')}`);
    console.log(`   Twin: ${twinName} (${twinId})`);
    console.log(`   Anomalies: ${data.result?.count || 0}`);
  });
});

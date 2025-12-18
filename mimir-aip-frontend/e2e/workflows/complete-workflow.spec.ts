import { test, expect, Page } from '@playwright/test';
import { setupAuthenticatedPage, waitForToast, uploadFile, expectVisible, expectTextVisible, waitForAPIResponse } from '../helpers';

/**
 * Comprehensive E2E workflow test that covers:
 * 1. Job scheduling setup
 * 2. Data ingestion pipeline creation
 * 3. Ontology setup with ingestion pipeline
 * 4. ML model training
 * 5. Agent chat with mock provider
 */

test.describe('Complete User Workflow', () => {
  let page: Page;

  test.beforeEach(async ({ browser }) => {
    // Create a new page with authenticated context
    page = await browser.newPage();
    await setupAuthenticatedPage(page);
    
    // Mock common API endpoints
    await mockAPIEndpoints(page);
    
    await page.goto('/');
  });

  test.afterEach(async () => {
    await page.close();
  });

  test('complete workflow: scheduling → ingestion → ontology → training → chat', async () => {
    // =========================================================================
    // STEP 1: Create a scheduled job for data ingestion
    // =========================================================================
    test.step('Create scheduled data ingestion job', async () => {
      // Navigate to scheduler/jobs page
      await page.goto('/scheduler');
      await expectVisible(page, 'h1, h2');
      
      // Click create new job button
      await page.click('button:has-text("Create Job"), button:has-text("New Job"), a:has-text("Create Job")');
      
      // Fill job details
      await page.fill('input[name="name"], input[placeholder*="name" i]', 'E2E Test Data Ingestion');
      await page.fill('textarea[name="description"], textarea[placeholder*="description" i]', 'Automated E2E test job');
      
      // Set cron schedule (every hour)
      await page.fill('input[name="schedule"], input[placeholder*="schedule" i], input[placeholder*="cron" i]', '0 * * * *');
      
      // Set pipeline reference
      await page.fill('input[name="pipeline_id"], input[placeholder*="pipeline" i]', 'e2e-test-pipeline');
      
      // Submit job creation
      await page.click('button[type="submit"]:has-text("Create"), button:has-text("Save")');
      
      // Verify job was created
      await waitForToast(page, /created|success/i, 10000);
      await expectTextVisible(page, 'E2E Test Data Ingestion', 10000);
    });

    // =========================================================================
    // STEP 2: Create data ingestion pipeline
    // =========================================================================
    test.step('Create data ingestion pipeline', async () => {
      // Navigate to pipelines page
      await page.goto('/pipelines');
      await expectVisible(page, 'h1, h2');
      
      // Click create pipeline
      await page.click('button:has-text("Create Pipeline"), button:has-text("New Pipeline"), a[href*="create"]');
      
      // Fill pipeline YAML
      const pipelineYAML = `version: "1.0"
name: e2e-test-pipeline
description: E2E test data ingestion pipeline
steps:
  - name: ingest_csv
    plugin: Input.csv
    config:
      file_path: test_data.csv
      delimiter: ","
      has_header: true
    output: raw_data
  
  - name: transform_data
    plugin: Data_Processing.transformer
    config:
      operations:
        - type: normalize
          columns: ["value"]
    output: processed_data
  
  - name: store_data
    plugin: Storage.persistence
    config:
      table: test_dataset
      mode: append
`;
      
      await page.fill('textarea[name="yaml"], textarea[placeholder*="yaml" i], .monaco-editor textarea, textarea', pipelineYAML);
      
      // Save pipeline
      await page.click('button:has-text("Create"), button:has-text("Save")');
      
      // Verify pipeline created
      await waitForToast(page, /created|success/i, 10000);
    });

    // =========================================================================
    // STEP 3: Upload test data file
    // =========================================================================
    test.step('Upload test CSV data', async () => {
      // Navigate to data ingestion page
      await page.goto('/data-ingestion');
      
      // Create test CSV content
      const testCSV = `id,name,value,category
1,Item A,100,electronics
2,Item B,200,furniture
3,Item C,150,electronics
4,Item D,300,furniture
5,Item E,250,electronics`;
      
      // Find file input and upload
      const fileInput = page.locator('input[type="file"]');
      await uploadFile(page, 'input[type="file"]', 'test_data.csv', testCSV, 'text/csv');
      
      // Click upload/import button
      await page.click('button:has-text("Upload"), button:has-text("Import"), button:has-text("Ingest")');
      
      // Wait for upload success
      await waitForToast(page, /uploaded|success|imported/i, 15000);
    });

    // =========================================================================
    // STEP 4: Create ontology using the ingested data
    // =========================================================================
    test.step('Create ontology from ingested data', async () => {
      // Navigate to ontology page
      await page.goto('/ontology');
      await expectVisible(page, 'h1, h2');
      
      // Click create ontology
      await page.click('button:has-text("Create Ontology"), button:has-text("New Ontology"), a[href*="create"]');
      
      // Fill ontology details
      await page.fill('input[name="name"], input[placeholder*="name" i]', 'E2E Test Ontology');
      await page.fill('input[name="version"], input[placeholder*="version" i]', '1.0.0');
      await page.fill('textarea[name="description"], textarea[placeholder*="description" i]', 'Ontology generated from E2E test data');
      
      // Select turtle format
      await page.selectOption('select[name="format"], select', { value: 'turtle' });
      
      // Provide ontology content
      const ontologyContent = `@prefix : <http://test.example.org/e2e#> .
@prefix owl: <http://www.w3.org/2002/07/owl#> .
@prefix rdf: <http://www.w3.org/1999/02/22-rdf-syntax-ns#> .
@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .
@prefix xsd: <http://www.w3.org/2001/XMLSchema#> .

:E2ETestOntology a owl:Ontology ;
    rdfs:label "E2E Test Ontology" ;
    rdfs:comment "Generated from test dataset" .

:Item a owl:Class ;
    rdfs:label "Item" ;
    rdfs:comment "A catalog item" .

:Category a owl:Class ;
    rdfs:label "Category" ;
    rdfs:comment "Item category" .

:name a owl:DatatypeProperty ;
    rdfs:label "name" ;
    rdfs:domain :Item ;
    rdfs:range xsd:string .

:value a owl:DatatypeProperty ;
    rdfs:label "value" ;
    rdfs:domain :Item ;
    rdfs:range xsd:integer .

:belongsTo a owl:ObjectProperty ;
    rdfs:label "belongs to" ;
    rdfs:domain :Item ;
    rdfs:range :Category .
`;
      
      await page.fill('textarea[name="content"], textarea[name="ontology_data"], .monaco-editor textarea', ontologyContent);
      
      // Submit ontology creation
      await page.click('button[type="submit"]:has-text("Create"), button:has-text("Save")');
      
      // Verify ontology created
      await waitForToast(page, /created|success/i, 10000);
      await expectTextVisible(page, 'E2E Test Ontology', 10000);
    });

    // =========================================================================
    // STEP 5: Trigger schema inference (optional but good to test)
    // =========================================================================
    test.step('Run schema inference on data', async () => {
      // Navigate back to ontology page
      await page.goto('/ontology');
      
      // Click on the created ontology
      await page.click('text=E2E Test Ontology');
      
      // Look for schema inference button
      const inferButton = page.locator('button:has-text("Infer Schema"), button:has-text("Auto-detect"), button:has-text("Analyze")');
      
      if (await inferButton.isVisible()) {
        await inferButton.click();
        
        // Wait for inference to complete
        await waitForToast(page, /inference|detected|analyzed/i, 15000);
      }
    });

    // =========================================================================
    // STEP 6: Train ML model on the ingested data
    // =========================================================================
    test.step('Train ML model on ingested data', async () => {
      // Navigate to ML training page (could be on ontology page)
      await page.goto('/ontology');
      
      // Look for auto-training or ML training section
      const trainButton = page.locator('button:has-text("Train Model"), button:has-text("Auto-Train"), button:has-text("Start Training")');
      
      if (await trainButton.isVisible()) {
        await trainButton.click();
        
        // Fill training configuration
        await page.fill('input[name="model_name"], input[placeholder*="model name" i]', 'E2E Test Model');
        await page.fill('input[name="target_column"], input[placeholder*="target" i]', 'category');
        
        // Select model type (decision tree)
        const modelTypeSelect = page.locator('select[name="model_type"], select[name="algorithm"]');
        if (await modelTypeSelect.isVisible()) {
          await modelTypeSelect.selectOption({ value: 'decision_tree' });
        }
        
        // Start training
        await page.click('button[type="submit"]:has-text("Train"), button:has-text("Start Training")');
        
        // Wait for training to complete (this might take a while with optimizations)
        await waitForToast(page, /training|completed|success/i, 60000);
        await expectTextVisible(page, /model.*trained|training.*complete/i, 60000);
      } else {
        // Alternative: go directly to ML page if it exists
        await page.goto('/ml');
        
        // Upload dataset for training
        const mlFileInput = page.locator('input[type="file"]');
        if (await mlFileInput.isVisible()) {
          const trainingCSV = `feature1,feature2,feature3,label
1.5,2.3,3.1,class_a
2.1,3.4,1.9,class_b
3.2,1.8,4.2,class_a
1.9,4.1,2.3,class_b
2.8,2.9,3.5,class_a`;
          
          await uploadFile(page, 'input[type="file"]', 'training_data.csv', trainingCSV, 'text/csv');
          
          // Configure and start training
          await page.fill('input[name="model_name"]', 'E2E Test Model');
          await page.fill('input[name="target_column"]', 'label');
          await page.click('button:has-text("Train"), button:has-text("Start Training")');
          
          // Wait for completion
          await waitForToast(page, /training|completed|success/i, 60000);
        }
      }
    });

    // =========================================================================
    // STEP 7: Test agent chat with mock provider
    // =========================================================================
    test.step('Test agent chat with mock AI provider', async () => {
      // Navigate to agent chat page
      await page.goto('/agent');
      
      // Create new conversation with mock provider
      await page.click('button:has-text("New Chat"), button:has-text("New Conversation"), a:has-text("New Chat")');
      
      // Configure to use mock provider
      const providerSelect = page.locator('select[name="provider"], select[name="model_provider"]');
      if (await providerSelect.isVisible()) {
        await providerSelect.selectOption({ value: 'mock' });
      }
      
      // Set conversation title
      await page.fill('input[name="title"], input[placeholder*="title" i]', 'E2E Test Chat');
      
      // Create conversation
      await page.click('button:has-text("Create"), button:has-text("Start")');
      
      // Test 1: Ask about digital twins
      await page.fill('textarea[name="message"], textarea[placeholder*="message" i], textarea[placeholder*="type" i]', 'Can you help me create a new digital twin?');
      await page.click('button[type="submit"], button:has-text("Send")');
      
      // Verify mock response appears
      await expectTextVisible(page, /digital twin/i, 10000);
      await expectTextVisible(page, /system type|parameters/i, 10000);
      
      // Test 2: Ask about data/ML
      await page.fill('textarea[name="message"], textarea[placeholder*="message" i], textarea[placeholder*="type" i]', 'I need to train a model on my data');
      await page.click('button[type="submit"], button:has-text("Send")');
      
      // Verify mock response about ML
      await expectTextVisible(page, /machine learning|model|decision tree/i, 10000);
      
      // Test 3: Ask about pipelines
      await page.fill('textarea[name="message"], textarea[placeholder*="message" i], textarea[placeholder*="type" i]', 'How do I create a data pipeline?');
      await page.click('button[type="submit"], button:has-text("Send")');
      
      // Verify mock response about pipelines
      await expectTextVisible(page, /pipeline|source format|processing/i, 10000);
      
      // Test 4: General help
      await page.fill('textarea[name="message"], textarea[placeholder*="message" i], textarea[placeholder*="type" i]', 'What can you help me with?');
      await page.click('button[type="submit"], button:has-text("Send")');
      
      // Verify comprehensive help response
      await expectTextVisible(page, /Mimir|assistant|digital twins|pipelines|ML/i, 10000);
    });

    // =========================================================================
    // STEP 8: Verify all created resources
    // =========================================================================
    test.step('Verify all resources were created successfully', async () => {
      // Check scheduler job exists
      await page.goto('/scheduler');
      await expectTextVisible(page, 'E2E Test Data Ingestion', 10000);
      
      // Check pipeline exists
      await page.goto('/pipelines');
      await expectTextVisible(page, 'e2e-test-pipeline', 10000);
      
      // Check ontology exists
      await page.goto('/ontology');
      await expectTextVisible(page, 'E2E Test Ontology', 10000);
      
      // Check ML model exists (if there's a models page)
      const modelsPage = await page.goto('/ml/models').catch(() => null);
      if (modelsPage) {
        await expectTextVisible(page, 'E2E Test Model', 10000);
      }
      
      // Check agent conversation exists
      await page.goto('/agent');
      await expectTextVisible(page, 'E2E Test Chat', 10000);
    });
  });
});

/**
 * Helper function to mock all necessary API endpoints
 */
async function mockAPIEndpoints(page: Page) {
  // Mock scheduler job creation
  await page.route('**/api/v1/scheduler/jobs', async (route) => {
    if (route.request().method() === 'POST') {
      await route.fulfill({
        status: 201,
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          job_id: 'job-test-123',
          message: 'Job created successfully',
        }),
      });
    } else {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          jobs: [],
        }),
      });
    }
  });

  // Mock pipeline creation
  await page.route('**/api/v1/pipelines', async (route) => {
    if (route.request().method() === 'POST') {
      await route.fulfill({
        status: 201,
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          pipeline_id: 'pipeline-test-123',
          message: 'Pipeline created successfully',
        }),
      });
    } else {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          pipelines: [],
        }),
      });
    }
  });

  // Mock data ingestion/upload
  await page.route('**/api/v1/data/upload', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        dataset_id: 'dataset-test-123',
        rows_imported: 5,
        message: 'Data uploaded successfully',
      }),
    });
  });

  await page.route('**/api/v1/data/ingest', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        message: 'Data ingested successfully',
      }),
    });
  });

  // Mock ontology creation
  await page.route('**/api/v1/ontology', async (route) => {
    if (route.request().method() === 'POST') {
      await route.fulfill({
        status: 201,
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          ontology_id: 'ontology-test-123',
          message: 'Ontology created successfully',
        }),
      });
    } else {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [],
        }),
      });
    }
  });

  // Mock schema inference
  await page.route('**/api/v1/ontology/*/infer', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        inferred_schema: {
          classes: ['Item', 'Category'],
          properties: ['name', 'value', 'belongsTo'],
        },
        message: 'Schema inferred successfully',
      }),
    });
  });

  // Mock ML training
  await page.route('**/api/v1/ml/train', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        model_id: 'model-test-123',
        accuracy: 0.95,
        message: 'Model trained successfully',
      }),
    });
  });

  await page.route('**/api/v1/ml/auto-train', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        model_id: 'model-test-123',
        accuracy: 0.95,
        message: 'Model trained successfully',
      }),
    });
  });

  // Mock agent chat conversations
  await page.route('**/api/v1/agent/conversations', async (route) => {
    if (route.request().method() === 'POST') {
      await route.fulfill({
        status: 201,
        contentType: 'application/json',
        body: JSON.stringify({
          conversation_id: 'conv-test-123',
          conversation: {
            id: 'conv-test-123',
            title: 'E2E Test Chat',
            model_provider: 'mock',
            model_name: 'mock-gpt-4',
            created_at: new Date().toISOString(),
            updated_at: new Date().toISOString(),
          },
          message: 'Conversation created successfully',
        }),
      });
    } else {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          conversations: [],
        }),
      });
    }
  });

  // Mock agent chat messages (using our intelligent mock provider)
  await page.route('**/api/v1/agent/conversations/*/messages', async (route) => {
    const requestBody = await route.request().postDataJSON();
    const userMessage = requestBody?.message || '';
    
    // Generate context-aware response based on message content
    let assistantReply = 'I understand. How can I assist you further?';
    
    if (userMessage.toLowerCase().includes('digital twin')) {
      assistantReply = "I'll help you create a new digital twin. Please specify the system type and initial parameters.";
    } else if (userMessage.toLowerCase().includes('train') || userMessage.toLowerCase().includes('model')) {
      assistantReply = 'I can train a machine learning model on your data. I recommend starting with a decision tree classifier. Would you like me to proceed?';
    } else if (userMessage.toLowerCase().includes('pipeline')) {
      assistantReply = 'I can help you create a data pipeline. What\'s your source format and what processing do you need?';
    } else if (userMessage.toLowerCase().includes('help') || userMessage.toLowerCase().includes('what can')) {
      assistantReply = "I'm Mimir, your AI assistant. I can help with digital twins, data pipelines, ML training, ontology creation, and job scheduling. What would you like to start with?";
    }
    
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        conversation_id: 'conv-test-123',
        user_message: {
          id: Date.now(),
          conversation_id: 'conv-test-123',
          role: 'user',
          content: userMessage,
          created_at: new Date().toISOString(),
        },
        assistant_reply: {
          id: Date.now() + 1,
          conversation_id: 'conv-test-123',
          role: 'assistant',
          content: assistantReply,
          created_at: new Date().toISOString(),
        },
        tool_calls: [],
      }),
    });
  });

  // Mock models list
  await page.route('**/api/v1/ml/models', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        models: [],
      }),
    });
  });
}

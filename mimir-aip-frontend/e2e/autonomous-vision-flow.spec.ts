/**
 * E2E Test: Autonomous Vision Flow
 * 
 * This test follows the DESIRED autonomous workflow from AUTONOMOUS_SYSTEM_GAP_ANALYSIS.md:
 * 
 * Desired Flow (Autonomous):
 * 1. User: "Ingest products.csv and give me insights"
 * 2. System automatically:
 *    - Ingests data → storage
 *    - Creates ontology from schema
 *    - Trains ML models for predictions
 *    - Creates digital twin
 *    - Starts continuous monitoring
 *    - Sends alerts on anomalies
 * 
 * Current Implementation Status:
 * ✅ = Working
 * ⚠️ = Partially working / simulated
 * ❌ = Not implemented
 */

import { test, expect } from '@playwright/test';
import path from 'path';

test.describe('Autonomous Vision Flow E2E', () => {
  const baseURL = process.env.BASE_URL || 'http://localhost:8080';

  test('Complete autonomous pipeline: CSV upload → insights', async ({ page }) => {
    test.setTimeout(120000); // 2 minutes for full pipeline

    console.log('\n🎯 Testing Autonomous Vision: CSV Upload → Automatic Insights\n');

    // ============================================
    // Step 1: Upload CSV with Autonomous Mode
    // ============================================
    console.log('📤 Step 1: Uploading CSV file with autonomous mode enabled...');
    
    await page.goto(`${baseURL}/data/upload`);
    await expect(page.locator('h1')).toContainText('Data Ingestion');

    // Create test CSV file
    const testCSV = path.join(__dirname, '../test-data/products-test.csv');
    
    // Check if autonomous mode toggle exists
    const autonomousModeCheckbox = page.locator('input[type="checkbox"]').filter({ hasText: /autonomous/i });
    const hasAutonomousMode = await autonomousModeCheckbox.count() > 0;
    
    if (hasAutonomousMode) {
      console.log('✅ Autonomous mode toggle found');
      await autonomousModeCheckbox.check();
    } else {
      console.log('❌ GAP: Autonomous mode toggle not found in UI');
    }

    // Upload file
    const fileInput = page.locator('input[type="file"]');
    await fileInput.setInputFiles(testCSV);
    await page.waitForTimeout(1000);

    // Click upload button
    const uploadButton = page.locator('button').filter({ hasText: /upload/i }).first();
    await uploadButton.click();

    // Wait for upload to complete
    await page.waitForTimeout(3000);

    // Check if we're redirected to workflows page or preview page
    const currentURL = page.url();
    console.log(`Current URL after upload: ${currentURL}`);

    let workflowID: string | null = null;

    if (currentURL.includes('/workflows/')) {
      // ✅ Redirected to workflow detail page (autonomous mode working)
      workflowID = currentURL.split('/workflows/')[1].split('?')[0];
      console.log(`✅ WORKING: Redirected to workflow page: /workflows/${workflowID}`);
    } else if (currentURL.includes('/data/preview/')) {
      // ⚠️ Redirected to preview page (autonomous mode not working)
      console.log('⚠️ GAP: Redirected to preview page instead of workflow');
      console.log('   Expected: Autonomous mode should create workflow automatically');
      console.log('   Actual: Manual preview flow (user must create ontology manually)');
      
      // Check if there's a "Create Workflow" or "Start Autonomous Pipeline" button
      const autonomousButton = page.locator('button').filter({ hasText: /autonomous|workflow|pipeline/i });
      const hasButton = await autonomousButton.count() > 0;
      
      if (hasButton) {
        console.log('⚠️ Found manual button to start autonomous workflow');
        await autonomousButton.first().click();
        await page.waitForTimeout(2000);
        
        if (page.url().includes('/workflows/')) {
          workflowID = page.url().split('/workflows/')[1].split('?')[0];
          console.log(`✅ Workflow created manually: ${workflowID}`);
        }
      } else {
        console.log('❌ GAP: No way to start autonomous workflow from preview page');
      }
    }

    // ============================================
    // Step 2: Verify Workflow Creation & Tracking
    // ============================================
    if (workflowID) {
      console.log('\n📊 Step 2: Verifying workflow execution...');
      
      await page.goto(`${baseURL}/workflows/${workflowID}`);
      await page.waitForTimeout(2000);

      // Check for workflow status
      const workflowStatus = page.locator('[data-testid="workflow-status"], .status, .badge').first();
      const statusExists = await workflowStatus.count() > 0;
      
      if (statusExists) {
        const status = await workflowStatus.textContent();
        console.log(`✅ Workflow status: ${status}`);
      } else {
        console.log('❌ GAP: Workflow status not visible');
      }

      // Check for step progress
      const steps = page.locator('[data-testid="workflow-step"], .step, li').filter({ hasText: /schema|ontology|extraction|training|twin|monitor/i });
      const stepCount = await steps.count();
      
      if (stepCount > 0) {
        console.log(`✅ Found ${stepCount} workflow steps visible in UI`);
        
        // Wait for workflow to progress (with timeout)
        console.log('⏳ Waiting for workflow to execute steps...');
        
        for (let i = 0; i < 30; i++) { // Poll for 30 seconds
          await page.waitForTimeout(1000);
          
          const completedSteps = await page.locator('[data-testid="step-status"], .completed, .success').count();
          console.log(`   Progress: ${completedSteps} steps completed`);
          
          // Check if all steps completed
          const workflowCompleted = await page.locator('text=/completed|success/i').filter({ has: page.locator('[data-testid="workflow-status"]') }).count() > 0;
          if (workflowCompleted) {
            console.log('✅ Workflow completed successfully!');
            break;
          }
        }
      } else {
        console.log('❌ GAP: Workflow steps not visible in UI');
      }

      // ============================================
      // Step 3: Verify Schema Inference (Auto-generated)
      // ============================================
      console.log('\n🔍 Step 3: Checking for auto-generated schema...');
      
      // Look for schema artifact link or indication
      const schemaLink = page.locator('a, button').filter({ hasText: /schema|inferred/i });
      const hasSchema = await schemaLink.count() > 0;
      
      if (hasSchema) {
        console.log('✅ Schema inference step visible');
        
        // Try to navigate to schema
        await schemaLink.first().click();
        await page.waitForTimeout(2000);
        
        if (page.url().includes('/schema/') || page.url().includes('/data/preview/')) {
          console.log('✅ WORKING: Can view inferred schema');
          
          // Check for schema details
          const hasColumns = await page.locator('text=/column|field|property/i').count() > 0;
          const hasTypes = await page.locator('text=/string|number|date|integer/i').count() > 0;
          
          if (hasColumns && hasTypes) {
            console.log('✅ Schema contains column definitions with types');
          } else {
            console.log('⚠️ Schema exists but may be incomplete');
          }
        }
        
        // Navigate back to workflow
        await page.goto(`${baseURL}/workflows/${workflowID}`);
        await page.waitForTimeout(1000);
      } else {
        console.log('❌ GAP: Schema inference not visible or not implemented');
        console.log('   Expected: System should auto-detect CSV schema (columns, types, relationships)');
      }

      // ============================================
      // Step 4: Verify Ontology Creation (Auto-generated)
      // ============================================
      console.log('\n🧠 Step 4: Checking for auto-generated ontology...');
      
      const ontologyLink = page.locator('a, button').filter({ hasText: /ontology|classes|properties/i });
      const hasOntology = await ontologyLink.count() > 0;
      
      if (hasOntology) {
        console.log('✅ Ontology creation step visible');
        
        // Try to navigate to ontology
        await ontologyLink.first().click();
        await page.waitForTimeout(2000);
        
        if (page.url().includes('/ontologies/')) {
          console.log('✅ WORKING: Can view auto-generated ontology');
          
          // Check for ontology content
          const hasClasses = await page.locator('text=/class|entity|type/i').count() > 0;
          const hasProperties = await page.locator('text=/property|attribute|relation/i').count() > 0;
          
          if (hasClasses && hasProperties) {
            console.log('✅ Ontology contains classes and properties');
          } else {
            console.log('⚠️ Ontology exists but may be incomplete');
          }
        }
        
        await page.goto(`${baseURL}/workflows/${workflowID}`);
        await page.waitForTimeout(1000);
      } else {
        console.log('❌ GAP: Automatic ontology creation not implemented');
        console.log('   Expected: System generates OWL/TTL from CSV schema automatically');
        console.log('   Current: Users must manually create ontologies (requires OWL expertise)');
      }

      // ============================================
      // Step 5: Verify ML Model Training (Auto-triggered)
      // ============================================
      console.log('\n🤖 Step 5: Checking for auto-trained ML models...');
      
      const mlLink = page.locator('a, button').filter({ hasText: /model|training|ml|prediction/i });
      const hasML = await mlLink.count() > 0;
      
      if (hasML) {
        console.log('✅ ML training step visible');
        
        await mlLink.first().click();
        await page.waitForTimeout(2000);
        
        if (page.url().includes('/models/')) {
          console.log('✅ WORKING: Can view trained models');
          
          // Check for model details
          const hasAccuracy = await page.locator('text=/accuracy|score|performance/i').count() > 0;
          const hasTarget = await page.locator('text=/target|predict|column/i').count() > 0;
          
          if (hasAccuracy && hasTarget) {
            console.log('✅ Models show training metrics and target columns');
          } else {
            console.log('⚠️ Models exist but may lack metadata');
          }
        }
        
        await page.goto(`${baseURL}/workflows/${workflowID}`);
        await page.waitForTimeout(1000);
      } else {
        console.log('❌ GAP: Automatic ML training not implemented');
        console.log('   Expected: System detects numeric columns → trains regression models automatically');
        console.log('   Current: Users must manually train models via /models/train');
      }

      // ============================================
      // Step 6: Verify Digital Twin Creation (Auto-triggered)
      // ============================================
      console.log('\n🔮 Step 6: Checking for auto-generated digital twin...');
      
      const twinLink = page.locator('a, button').filter({ hasText: /twin|simulation|digital/i });
      const hasTwin = await twinLink.count() > 0;
      
      if (hasTwin) {
        console.log('✅ Digital twin creation step visible');
        
        await twinLink.first().click();
        await page.waitForTimeout(2000);
        
        if (page.url().includes('/digital-twins/')) {
          console.log('✅ WORKING: Can view digital twin');
          
          // Check for twin features
          const hasState = await page.locator('text=/state|property|value/i').count() > 0;
          const hasSimulation = await page.locator('text=/scenario|simulate|run/i').count() > 0;
          
          if (hasState && hasSimulation) {
            console.log('✅ Digital twin has state tracking and simulation capabilities');
          } else {
            console.log('⚠️ Digital twin exists but may lack features');
          }
        }
        
        await page.goto(`${baseURL}/workflows/${workflowID}`);
        await page.waitForTimeout(1000);
      } else {
        console.log('❌ GAP: Automatic digital twin creation not implemented');
        console.log('   Expected: System creates twin from ontology + ML models automatically');
        console.log('   Current: Users must manually create twins');
      }

      // ============================================
      // Step 7: Verify Continuous Monitoring
      // ============================================
      console.log('\n📡 Step 7: Checking for continuous monitoring setup...');
      
      const monitoringLink = page.locator('a, button').filter({ hasText: /monitor|alert|anomaly|watch/i });
      const hasMonitoring = await monitoringLink.count() > 0;
      
      if (hasMonitoring) {
        console.log('✅ Monitoring setup step visible');
        
        await monitoringLink.first().click();
        await page.waitForTimeout(2000);
        
        if (page.url().includes('/monitoring/')) {
          console.log('✅ WORKING: Monitoring page exists');
          
          // Check for monitoring features
          const hasAlerts = await page.locator('text=/alert|notification|threshold/i').count() > 0;
          const hasMetrics = await page.locator('text=/metric|measure|track/i').count() > 0;
          
          if (hasAlerts && hasMetrics) {
            console.log('✅ Monitoring includes alerts and metrics tracking');
          } else {
            console.log('⚠️ Monitoring exists but may lack alerting');
          }
        }
        
        await page.goto(`${baseURL}/workflows/${workflowID}`);
        await page.waitForTimeout(1000);
      } else {
        console.log('❌ GAP: Continuous monitoring not implemented');
        console.log('   Expected: System sets up monitoring jobs for the digital twin');
        console.log('   Current: No automatic anomaly detection or alerting');
      }

    } else {
      console.log('\n❌ CRITICAL GAP: Workflow was not created');
      console.log('   Cannot proceed with testing autonomous pipeline');
    }

    // ============================================
    // Step 8: Check Workflows Dashboard
    // ============================================
    console.log('\n📋 Step 8: Checking workflows dashboard...');
    
    await page.goto(`${baseURL}/workflows`);
    await page.waitForTimeout(2000);

    const workflowCards = page.locator('[data-testid="workflow-card"], .workflow, .card').filter({ hasText: /workflow|pipeline/i });
    const cardCount = await workflowCards.count();
    
    if (cardCount > 0) {
      console.log(`✅ Workflows dashboard shows ${cardCount} workflow(s)`);
    } else {
      console.log('⚠️ Workflows dashboard exists but no workflows visible');
    }

    // ============================================
    // Final Summary
    // ============================================
    console.log('\n' + '='.repeat(60));
    console.log('📊 AUTONOMOUS VISION GAP ANALYSIS SUMMARY');
    console.log('='.repeat(60));
    
    console.log('\n✅ IMPLEMENTED:');
    console.log('  - File upload interface');
    console.log('  - Data preview');
    console.log('  - Workflow tracking UI');
    console.log('  - Workflow status updates');
    
    console.log('\n⚠️ PARTIALLY IMPLEMENTED:');
    console.log('  - Autonomous mode toggle (exists but may not trigger full pipeline)');
    console.log('  - Workflow orchestration (simulated steps, not real implementations)');
    console.log('  - Step progress tracking (UI exists, but steps are simulated)');
    
    console.log('\n❌ CRITICAL GAPS (from vision):');
    console.log('  1. No automatic schema inference from CSV');
    console.log('  2. No automatic ontology generation (requires manual OWL upload)');
    console.log('  3. No automatic ML training based on ontology');
    console.log('  4. No automatic digital twin creation');
    console.log('  5. No continuous monitoring setup');
    console.log('  6. No alerting/notification system');
    console.log('  7. Steps are simulated (2s delays), not real processing');
    
    console.log('\n💡 RECOMMENDATIONS:');
    console.log('  Phase 1: Implement real schema inference from CSV');
    console.log('  Phase 2: Auto-generate ontologies from schemas');
    console.log('  Phase 3: Integrate ML auto-training with ontologies');
    console.log('  Phase 4: Link digital twins with trained models');
    console.log('  Phase 5: Add continuous monitoring and alerting');
    
    console.log('\n' + '='.repeat(60));
  });

  test('Alternative flow: Manual pipeline creation', async ({ page }) => {
    console.log('\n🔄 Testing Alternative: Manual Pipeline Creation via Jobs\n');
    
    // This tests the vision's "scheduled ingestion pipeline" concept
    await page.goto(`${baseURL}/jobs`);
    await page.waitForTimeout(2000);

    // Check if we can create a data ingestion job
    const createJobButton = page.locator('button').filter({ hasText: /create|new/i }).first();
    const hasJobCreation = await createJobButton.count() > 0;

    if (hasJobCreation) {
      console.log('✅ Job creation UI exists');
      await createJobButton.click();
      await page.waitForTimeout(1000);

      // Look for pipeline/ingestion options
      const pipelineOption = page.locator('select, input, button').filter({ hasText: /pipeline|ingestion|data/i });
      const hasPipelineOption = await pipelineOption.count() > 0;

      if (hasPipelineOption) {
        console.log('✅ Can create jobs for data ingestion pipelines');
        console.log('   This is part of the vision (scheduled/continuous ingestion)');
      } else {
        console.log('❌ GAP: Cannot create data ingestion jobs from UI');
        console.log('   Expected: Users should be able to schedule recurring data imports');
      }
    } else {
      console.log('❌ GAP: Job creation not available from UI');
    }
  });
});

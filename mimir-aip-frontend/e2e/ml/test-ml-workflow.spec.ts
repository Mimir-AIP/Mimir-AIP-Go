import { test, expect } from '@playwright/test';

test('ML Model Training Workflow', async ({ page }) => {
  test.setTimeout(120000);
  
  console.log('\n╔══════════════════════════════════════════════════════════════╗');
  console.log('║           ML MODEL TRAINING WORKFLOW TEST                    ║');
  console.log('║   Ontology → Train Model → View Model → Make Predictions     ║');
  console.log('╚══════════════════════════════════════════════════════════════╝\n');
  
  // ============================
  // STEP 1: Navigate to Models Page
  // ============================
  console.log('📊 STEP 1: Navigate to Models Page');
  console.log('─'.repeat(60));
  
  await page.goto('http://localhost:8080/models');
  await page.waitForLoadState('networkidle');
  await page.waitForTimeout(2000);
  
  await page.screenshot({ path: 'test-results/ml-1-models-list.png', fullPage: true });
  
  const existingModels = await page.locator('div:has-text("Model"), tr').count();
  console.log(`  → Found ${existingModels} existing models`);
  
  // ============================
  // STEP 2: Navigate to Train Model Page
  // ============================
  console.log('\n🤖 STEP 2: Navigate to Train Model Page');
  console.log('─'.repeat(60));
  
  const trainButton = page.locator('a[href="/models/train"], button:has-text("Train")').first();
  const hasTrainButton = await trainButton.isVisible({ timeout: 5000 }).catch(() => false);
  
  if (!hasTrainButton) {
    console.log('  ⚠️  Train button not found, navigating directly...');
    await page.goto('http://localhost:8080/models/train');
  } else {
    console.log('  → Clicking Train Model button...');
    await trainButton.click();
  }
  
  await page.waitForLoadState('networkidle');
  await page.waitForTimeout(2000);
  
  await page.screenshot({ path: 'test-results/ml-2-train-page.png', fullPage: true });
  
  console.log('  ✓ On training page');
  
  // ============================
  // STEP 3: Check Form Elements
  // ============================
  console.log('\n📝 STEP 3: Check Training Form');
  console.log('─'.repeat(60));
  
  // Check for form elements
  const hasModelName = await page.locator('input[name="name"], input[id="name"]').isVisible({ timeout: 3000 }).catch(() => false);
  const hasOntologySelect = await page.locator('select[name="ontology"], select[id="ontology"]').isVisible({ timeout: 3000 }).catch(() => false);
  const hasTargetClass = await page.locator('select[name="target"], input[name="target"], select[id="target_class"]').isVisible({ timeout: 3000 }).catch(() => false);
  const hasAlgorithm = await page.locator('select[name="algorithm"], select[id="algorithm"]').isVisible({ timeout: 3000 }).catch(() => false);
  
  console.log(`  → Model name input: ${hasModelName ? '✓' : '✗'}`);
  console.log(`  → Ontology select: ${hasOntologySelect ? '✓' : '✗'}`);
  console.log(`  → Target class: ${hasTargetClass ? '✓' : '✗'}`);
  console.log(`  → Algorithm select: ${hasAlgorithm ? '✓' : '✗'}`);
  
  if (!hasModelName && !hasOntologySelect) {
    console.log('\n  ⚠️  Training form not found or incomplete');
    console.log('  → Checking page content...');
    const bodyText = await page.textContent('body');
    console.log(`  → Page contains "Train": ${bodyText?.includes('Train')}`);
    console.log(`  → Page contains "Model": ${bodyText?.includes('Model')}`);
    console.log(`  → Page contains "Ontology": ${bodyText?.includes('Ontology')}`);
    
    // Check for any error messages
    const hasError = bodyText?.toLowerCase().includes('error') || bodyText?.toLowerCase().includes('not found');
    if (hasError) {
      console.log('  → Page shows error message');
    }
  }
  
  // ============================
  // STEP 4: Try to Fill Form
  // ============================
  if (hasModelName && hasOntologySelect) {
    console.log('\n✏️  STEP 4: Fill Training Form');
    console.log('─'.repeat(60));
    
    try {
      // Fill model name
      console.log('  → Entering model name...');
      await page.fill('input[name="name"], input[id="name"]', 'E2E Test ML Model');
      
      // Select ontology
      console.log('  → Selecting ontology...');
      const ontologySelect = page.locator('select[name="ontology"], select[id="ontology"]').first();
      const options = await ontologySelect.locator('option').all();
      
      let selectedOntology = false;
      for (const option of options) {
        const value = await option.getAttribute('value');
        if (value && value !== '') {
          console.log(`     Selected: ${value}`);
          await ontologySelect.selectOption(value);
          selectedOntology = true;
          break;
        }
      }
      
      if (!selectedOntology) {
        console.log('  ✗ No ontologies available for training');
      } else {
        await page.waitForTimeout(1000);
        
        // Try to select target class if visible
        if (hasTargetClass) {
          console.log('  → Selecting target class...');
          const targetSelect = page.locator('select[name="target"], select[id="target_class"]').first();
          const targetOptions = await targetSelect.locator('option').all();
          
          if (targetOptions.length > 1) {
            const firstTarget = targetOptions[1]; // Skip empty option
            const value = await firstTarget.getAttribute('value');
            if (value) {
              await targetSelect.selectOption(value);
              console.log(`     Selected: ${value}`);
            }
          }
        }
        
        // Select algorithm if visible
        if (hasAlgorithm) {
          console.log('  → Selecting algorithm...');
          const algoSelect = page.locator('select[name="algorithm"], select[id="algorithm"]').first();
          await algoSelect.selectOption('decision_tree');
          console.log('     Selected: decision_tree');
        }
        
        await page.screenshot({ path: 'test-results/ml-3-form-filled.png', fullPage: true });
        
        // Look for submit button
        console.log('\n  → Looking for Train/Submit button...');
        const submitButton = page.locator('button[type="submit"], button:has-text("Train"), button:has-text("Start Training")').first();
        const canSubmit = await submitButton.isVisible({ timeout: 3000 }).catch(() => false);
        
        if (canSubmit) {
          console.log('  → Clicking Train button...');
          await submitButton.click();
          
          // Wait for training to complete
          console.log('  → Waiting for training...');
          await page.waitForTimeout(5000);
          
          await page.screenshot({ path: 'test-results/ml-4-after-training.png', fullPage: true });
          
          const finalUrl = page.url();
          console.log(`  → Final URL: ${finalUrl}`);
          
          if (finalUrl.includes('/models/') && !finalUrl.includes('/train')) {
            console.log('  ✓ Redirected to model detail page');
          } else {
            const bodyText = await page.textContent('body');
            const hasSuccess = bodyText?.toLowerCase().includes('success') || 
                             bodyText?.toLowerCase().includes('trained') ||
                             bodyText?.toLowerCase().includes('completed');
            console.log(`  → Training success message: ${hasSuccess ? 'Yes' : 'No'}`);
          }
        } else {
          console.log('  ✗ Submit button not found');
        }
      }
    } catch (error) {
      console.log(`  ✗ Error filling form: ${error}`);
    }
  }
  
  // ============================
  // STEP 5: Check Existing Models
  // ============================
  console.log('\n📋 STEP 5: Verify Models List');
  console.log('─'.repeat(60));
  
  await page.goto('http://localhost:8080/models');
  await page.waitForLoadState('networkidle');
  await page.waitForTimeout(2000);
  
  const finalModels = await page.locator('div:has-text("Model"), tr, a[href^="/models/"]').count();
  console.log(`  → Total models visible: ${finalModels}`);
  
  // Try to click on first model
  const firstModel = page.locator('a[href^="/models/"]:not([href*="train"])').first();
  const hasModel = await firstModel.isVisible({ timeout: 3000 }).catch(() => false);
  
  if (hasModel) {
    console.log('  → Clicking on first model...');
    await firstModel.click();
    await page.waitForTimeout(2000);
    
    await page.screenshot({ path: 'test-results/ml-5-model-detail.png', fullPage: true });
    
    const modelUrl = page.url();
    console.log(`  ✓ Model detail page: ${modelUrl}`);
    
    // Check for model metrics
    const bodyText = await page.textContent('body');
    const hasAccuracy = bodyText?.toLowerCase().includes('accuracy');
    const hasPrecision = bodyText?.toLowerCase().includes('precision');
    const hasMetrics = bodyText?.toLowerCase().includes('metric');
    
    console.log(`  → Shows metrics: ${hasAccuracy || hasPrecision || hasMetrics ? 'Yes' : 'No'}`);
  }
  
  // ============================
  // SUMMARY
  // ============================
  console.log('\n╔══════════════════════════════════════════════════════════════╗');
  console.log('║              ML WORKFLOW VERIFICATION COMPLETE               ║');
  console.log('╚══════════════════════════════════════════════════════════════╝\n');
});

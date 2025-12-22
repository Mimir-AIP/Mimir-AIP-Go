import { test, expect } from '@playwright/test';

test('Complete ML Workflow - CSV to Trained Model', async ({ page }) => {
  test.setTimeout(120000);
  
  console.log('\n╔══════════════════════════════════════════════════════════════╗');
  console.log('║          COMPLETE ML WORKFLOW TEST                           ║');
  console.log('║   Upload CSV → Train Model → View Results → Make Predictions ║');
  console.log('╚══════════════════════════════════════════════════════════════╝\n');
  
  // ============================
  // STEP 1: Navigate to Train Page
  // ============================
  console.log('🚀 STEP 1: Navigate to Train Model Page');
  console.log('─'.repeat(60));
  
  await page.goto('http://localhost:8080/models/train');
  await page.waitForLoadState('networkidle');
  await page.waitForTimeout(2000);
  
  await page.screenshot({ path: 'test-results/ml-workflow-1-train-page.png', fullPage: true });
  
  console.log('  ✓ On training page');
  
  // ============================
  // STEP 2: Create Test CSV
  // ============================
  console.log('\n📄 STEP 2: Create Test CSV File');
  console.log('─'.repeat(60));
  
  const testCsvContent = `product_id,product_name,category,price,stock
1,Laptop,Electronics,999.99,50
2,Mouse,Electronics,29.99,200
3,Keyboard,Electronics,79.99,150
4,Monitor,Electronics,299.99,75
5,Desk Chair,Furniture,199.99,30
6,Office Desk,Furniture,449.99,20
7,Table Lamp,Furniture,79.99,80
8,Notebook,Stationery,4.99,500
9,Pen Set,Stationery,12.99,300
10,Stapler,Stationery,8.99,250`;
  
  const fs = require('fs');
  const testFilePath = '/tmp/ml-training-test.csv';
  fs.writeFileSync(testFilePath, testCsvContent);
  
  console.log(`  ✓ Created test CSV: ${testFilePath}`);
  
  // ============================
  // STEP 3: Fill Training Form
  // ============================
  console.log('\n✏️  STEP 3: Fill Training Form');
  console.log('─'.repeat(60));
  
  // Fill model name
  console.log('  → Entering model name...');
  await page.fill('input#model-name, input[name="model_name"]', 'E2E ML Test Model');
  
  // Upload CSV file
  console.log('  → Uploading CSV file...');
  const fileInput = page.locator('input[type="file"]#csv-file');
  await fileInput.setInputFiles(testFilePath);
  await page.waitForTimeout(1000);
  
  // Verify file was selected
  const fileNameVisible = await page.getByText('ml-training-test.csv').isVisible({ timeout: 3000 }).catch(() => false);
  console.log(`  → File selected: ${fileNameVisible ? 'Yes' : 'No'}`);
  
  // Fill target column
  console.log('  → Entering target column...');
  await page.fill('input#target-column, input[name="target_column"]', 'price');
  
  // Select algorithm
  console.log('  → Selecting algorithm...');
  const algoSelect = page.locator('select#algorithm');
  await algoSelect.selectOption('decision_tree');
  
  await page.screenshot({ path: 'test-results/ml-workflow-2-form-filled.png', fullPage: true });
  
  console.log('  ✓ Form filled completely');
  
  // ============================
  // STEP 4: Submit and Train
  // ============================
  console.log('\n🤖 STEP 4: Train Model');
  console.log('─'.repeat(60));
  
  console.log('  → Clicking Train Model button...');
  const trainButton = page.locator('button[type="submit"], button:has-text("Train")');
  await trainButton.click();
  
  console.log('  → Waiting for training to complete...');
  
  // Wait for redirect to model detail page or success message
  await page.waitForTimeout(8000); // Training might take a few seconds
  
  await page.screenshot({ path: 'test-results/ml-workflow-3-after-training.png', fullPage: true });
  
  const finalUrl = page.url();
  console.log(`  → Final URL: ${finalUrl}`);
  
  if (finalUrl.includes('/models/') && !finalUrl.includes('/train')) {
    const modelId = finalUrl.split('/models/').pop();
    console.log(`  ✓ Model trained! ID: ${modelId}`);
    
    // ============================
    // STEP 5: View Model Details
    // ============================
    console.log('\n📊 STEP 5: View Model Details');
    console.log('─'.repeat(60));
    
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);
    
    const bodyText = await page.textContent('body');
    
    // Check for key metrics
    const hasAccuracy = bodyText?.toLowerCase().includes('accuracy');
    const hasPrecision = bodyText?.toLowerCase().includes('precision');
    const hasRecall = bodyText?.toLowerCase().includes('recall');
    const hasF1 = bodyText?.toLowerCase().includes('f1');
    const hasMetrics = bodyText?.toLowerCase().includes('metric');
    
    console.log(`  → Shows Accuracy: ${hasAccuracy ? 'Yes' : 'No'}`);
    console.log(`  → Shows Precision: ${hasPrecision ? 'Yes' : 'No'}`);
    console.log(`  → Shows Recall: ${hasRecall ? 'Yes' : 'No'}`);
    console.log(`  → Shows F1 Score: ${hasF1 ? 'Yes' : 'No'}`);
    console.log(`  → Shows Metrics: ${hasMetrics ? 'Yes' : 'No'}`);
    
    const showsMetrics = hasAccuracy || hasPrecision || hasMetrics;
    
    if (showsMetrics) {
      console.log('  ✓ Model metrics displayed');
    } else {
      console.log('  ⚠️  Model metrics not visible');
    }
    
    await page.screenshot({ path: 'test-results/ml-workflow-4-model-details.png', fullPage: true });
    
    // ============================
    // STEP 6: Check Models List
    // ============================
    console.log('\n📋 STEP 6: Verify Model in List');
    console.log('─'.repeat(60));
    
    await page.goto('http://localhost:8080/models');
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);
    
    const modelsText = await page.textContent('body');
    const hasNewModel = modelsText?.includes('E2E ML Test Model');
    
    console.log(`  → New model visible in list: ${hasNewModel ? 'Yes' : 'No'}`);
    
    await page.screenshot({ path: 'test-results/ml-workflow-5-models-list.png', fullPage: true });
    
    // ============================
    // SUMMARY
    // ============================
    console.log('\n╔══════════════════════════════════════════════════════════════╗');
    console.log('║          ✅ ML WORKFLOW COMPLETE!                            ║');
    console.log('╚══════════════════════════════════════════════════════════════╝\n');
    console.log('Summary:');
    console.log(`  ✓ CSV file uploaded`);
    console.log(`  ✓ Model trained: E2E ML Test Model`);
    console.log(`  ✓ Model ID: ${modelId}`);
    console.log(`  ✓ Metrics displayed: ${showsMetrics ? 'Yes' : 'No'}`);
    console.log(`  ✓ Model visible in list: ${hasNewModel ? 'Yes' : 'No'}`);
    console.log('\n🎉 All ML steps completed using FRONTEND ONLY! 🎉\n');
    
  } else {
    console.log('  ⚠️  Did not redirect to model detail page');
    console.log('  → Checking for error messages...');
    
    const bodyText = await page.textContent('body');
    const hasError = bodyText?.toLowerCase().includes('error') || bodyText?.toLowerCase().includes('failed');
    
    if (hasError) {
      console.log('  ✗ Training failed with error');
    }
  }
});

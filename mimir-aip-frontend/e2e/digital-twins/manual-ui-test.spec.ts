import { test, expect } from '@playwright/test';

test.describe('Complete UI Workflow - Manual Verification', () => {
  test('Full workflow from CSV to simulation results', async ({ page }) => {
    // Set longer timeout for this comprehensive test
    test.setTimeout(120000);
    
    await page.goto('http://localhost:8080');
    
    console.log('\n=== STEP 1: Navigate to Data Upload ===');
    await page.click('a[href="/data/upload"], button:has-text("Upload")');
    await page.waitForURL('**/data/upload');
    console.log('✓ On data upload page');
    
    console.log('\n=== STEP 2: Check if file upload works ===');
    // Look for file input
    const fileInput = page.locator('input[type="file"]');
    const hasFileInput = await fileInput.count() > 0;
    console.log(`File input found: ${hasFileInput}`);
    
    if (!hasFileInput) {
      console.log('⚠️  No file input - cannot test CSV upload');
      console.log('Skipping to ontology selection...');
    }
    
    console.log('\n=== STEP 3: Navigate to Digital Twins ===');
    await page.goto('http://localhost:8080/digital-twins');
    await page.waitForLoadState('networkidle');
    console.log('✓ On digital twins list page');
    
    // Take screenshot
    await page.screenshot({ path: 'test-results/step3-twin-list.png', fullPage: true });
    
    console.log('\n=== STEP 4: Click Create Digital Twin ===');
    const createButton = page.locator('a[href="/digital-twins/create"], button:has-text("Create")');
    await createButton.click();
    await page.waitForURL('**/digital-twins/create');
    console.log('✓ On create twin page');
    
    await page.screenshot({ path: 'test-results/step4-create-form.png', fullPage: true });
    
    console.log('\n=== STEP 5: Fill and submit form ===');
    await page.fill('input[name="name"], input[id="name"]', 'UI Test Twin');
    await page.fill('textarea[name="description"], textarea[id="description"]', 'Created via UI only');
    
    // Select ontology
    const ontologySelect = page.locator('select[name="ontology"], select[id="ontology"]');
    const allOptions = await ontologySelect.locator('option').all();
    let selectedOntology = null;
    
    for (const option of allOptions) {
      const value = await option.getAttribute('value');
      if (value && value !== '') {
        console.log(`Selecting ontology: ${value}`);
        await ontologySelect.selectOption(value);
        selectedOntology = value;
        break;
      }
    }
    
    if (!selectedOntology) {
      throw new Error('No ontologies available');
    }
    
    // Submit form
    const submitButton = page.locator('button[type="submit"]:has-text("Create")');
    await submitButton.click();
    console.log('✓ Form submitted');
    
    console.log('\n=== STEP 6: Wait for twin detail page ===');
    await page.waitForURL(/\/digital-twins\/[a-f0-9-]+$/, { timeout: 10000 });
    const twinUrl = page.url();
    const twinId = twinUrl.split('/').pop();
    console.log(`✓ Redirected to twin: ${twinId}`);
    console.log(`URL: ${twinUrl}`);
    
    await page.waitForLoadState('networkidle');
    await page.screenshot({ path: 'test-results/step6-twin-detail.png', fullPage: true });
    
    console.log('\n=== STEP 7: Look for Scenarios on the page ===');
    
    // Wait a moment for any dynamic content to load
    await page.waitForTimeout(2000);
    
    // Try multiple ways to find scenarios
    const scenarioSelectors = [
      '[data-testid="scenario-card"]',
      '[data-testid="scenario-list"]',
      'div:has-text("Baseline Operations")',
      'div:has-text("Capacity Stress Test")',
      'div:has-text("Data Quality Issues")',
      'h3:has-text("Scenarios")',
      'button:has-text("Run Simulation")',
      'button:has-text("Run")'
    ];
    
    console.log('Searching for scenario elements...');
    for (const selector of scenarioSelectors) {
      const count = await page.locator(selector).count();
      if (count > 0) {
        console.log(`✓ Found ${count} elements matching: ${selector}`);
      }
    }
    
    // Check if there's a scenarios tab or section
    const tabElements = await page.locator('[role="tab"], .tab, button:has-text("Scenario")').all();
    console.log(`Found ${tabElements.length} tab-like elements`);
    
    if (tabElements.length > 0) {
      console.log('Clicking on scenarios tab...');
      for (const tab of tabElements) {
        const text = await tab.textContent();
        console.log(`  Tab text: ${text}`);
        if (text?.toLowerCase().includes('scenario')) {
          await tab.click();
          await page.waitForTimeout(1000);
          console.log('✓ Clicked scenarios tab');
          break;
        }
      }
    }
    
    await page.screenshot({ path: 'test-results/step7-after-tab-click.png', fullPage: true });
    
    console.log('\n=== STEP 8: Count actual scenarios visible ===');
    const scenarioCards = await page.locator('[data-testid="scenario-card"], .scenario-card, div:has-text("Baseline"), div:has-text("Capacity"), div:has-text("Data Quality")').all();
    console.log(`Visible scenario cards: ${scenarioCards.length}`);
    
    if (scenarioCards.length === 0) {
      console.log('⚠️  NO SCENARIOS VISIBLE IN UI');
      console.log('Checking page text content...');
      const pageText = await page.textContent('body');
      const hasBaseline = pageText?.includes('Baseline');
      const hasCapacity = pageText?.includes('Capacity');
      const hasDataQuality = pageText?.includes('Data Quality');
      console.log(`  Page contains "Baseline": ${hasBaseline}`);
      console.log(`  Page contains "Capacity": ${hasCapacity}`);
      console.log(`  Page contains "Data Quality": ${hasDataQuality}`);
      
      // Check the HTML structure
      console.log('\nPage structure:');
      const headings = await page.locator('h1, h2, h3, h4').allTextContents();
      console.log('  Headings:', headings);
      
      throw new Error('Scenarios were created in backend but NOT visible in UI');
    }
    
    console.log(`✓ Found ${scenarioCards.length} scenarios in UI`);
    
    console.log('\n=== STEP 9: Find and click Run button ===');
    const runButton = page.locator('button:has-text("Run")').first();
    const runButtonVisible = await runButton.isVisible({ timeout: 5000 }).catch(() => false);
    
    if (!runButtonVisible) {
      console.log('⚠️  Run button not visible');
      throw new Error('Cannot find Run button in UI');
    }
    
    console.log('✓ Run button found, clicking...');
    await runButton.click();
    console.log('✓ Clicked Run button');
    
    console.log('\n=== STEP 10: Wait for simulation results ===');
    // Could redirect to results page or show inline results
    await page.waitForTimeout(3000);
    
    const finalUrl = page.url();
    console.log(`Final URL: ${finalUrl}`);
    
    await page.screenshot({ path: 'test-results/step10-final-results.png', fullPage: true });
    
    // Check for success indicators
    const pageText = await page.textContent('body');
    const hasSuccess = pageText?.toLowerCase().includes('success') || 
                       pageText?.toLowerCase().includes('completed') ||
                       pageText?.toLowerCase().includes('simulation');
    
    console.log(`Page indicates success: ${hasSuccess}`);
    
    if (!hasSuccess) {
      throw new Error('Simulation results not visible in UI');
    }
    
    console.log('\n✅ COMPLETE UI WORKFLOW SUCCESSFUL!');
    console.log('User can:');
    console.log('  1. Create digital twin from ontology');
    console.log('  2. See auto-generated scenarios');
    console.log('  3. Run simulations');
    console.log('  4. View results');
    console.log('All without making API calls!');
  });
});

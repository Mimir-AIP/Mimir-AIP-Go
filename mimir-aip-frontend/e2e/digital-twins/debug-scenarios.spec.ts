import { test } from '@playwright/test';
import { setupAuthenticatedPage } from '../helpers';

test('Debug scenario loading', async ({ page }) => {
  await setupAuthenticatedPage(page);
  const twinId = '8f65f121-e1d9-4d92-8f5e-ad8f3aacdd94';
  
  // Capture console logs
  page.on('console', msg => {
    console.log(`[BROWSER ${msg.type()}]:`, msg.text());
  });
  
  // Capture network requests
  page.on('response', async response => {
    if (response.url().includes('/scenarios')) {
      console.log(`\n[NETWORK] Scenarios API Response:`);
      console.log(`  Status: ${response.status()}`);
      console.log(`  URL: ${response.url()}`);
      try {
        const json = await response.json();
        console.log(`  Body:`, JSON.stringify(json, null, 2));
      } catch (e) {
        console.log(`  Could not parse JSON`);
      }
    }
  });
  
  console.log(`\nNavigating to: http://localhost:8080/digital-twins/${twinId}`);
  await page.goto(`http://localhost:8080/digital-twins/${twinId}`);
  
  // Wait for page to load
  await page.waitForLoadState('networkidle');
  await page.waitForTimeout(2000);
  
  // Check if twin exists (404 page or error)
  const has404 = await page.locator('text=/not found|404/i').count() > 0;
  if (has404) {
    console.log('\nTwin not found - skipping test');
    test.skip();
    return;
  }
  
  // Click scenarios tab if it exists
  console.log('\nLooking for scenarios tab...');
  const scenariosTab = page.locator('button:has-text("Scenarios")');
  const tabExists = await scenariosTab.count() > 0;
  
  if (!tabExists) {
    console.log('Scenarios tab not found - feature may not be implemented');
    test.skip();
    return;
  }
  
  await scenariosTab.click();
  await page.waitForTimeout(1000);
  
  // Take screenshot
  await page.screenshot({ path: 'test-results/debug-scenarios-tab.png', fullPage: true });
  
  // Check what's rendered
  const scenarioCards = await page.locator('[data-testid="scenario-card"], .hover\\:shadow-md, div:has(button:has-text("Run"))').all();
  console.log(`\nScenario cards found: ${scenarioCards.length}`);
  
  // Check for "No Scenarios Yet" message
  const noScenariosText = await page.getByText('No Scenarios Yet').count();
  console.log(`"No Scenarios Yet" message visible: ${noScenariosText > 0}`);
  
  // Get page text
  const bodyText = await page.textContent('body');
  console.log('\nPage contains:');
  console.log(`  "Baseline": ${bodyText?.includes('Baseline')}`);
  console.log(`  "Capacity": ${bodyText?.includes('Capacity')}`);
  console.log(`  "Data Quality": ${bodyText?.includes('Data Quality')}`);
});

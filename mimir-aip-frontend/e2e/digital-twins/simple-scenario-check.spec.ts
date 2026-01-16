import { test, expect } from '@playwright/test';
import { setupAuthenticatedPage } from '../helpers';

test('Check if scenarios are visible for existing twin', async ({ page }) => {
  await setupAuthenticatedPage(page);
  const twinId = '8f65f121-e1d9-4d92-8f5e-ad8f3aacdd94';
  
  console.log(`\nNavigating to twin detail page: ${twinId}`);
  await page.goto(`http://localhost:8080/digital-twins/${twinId}`);
  await page.waitForLoadState('networkidle');
  
  // Take screenshot
  await page.screenshot({ path: 'test-results/twin-detail-full.png', fullPage: true });
  
  console.log('\nPage title:', await page.title());
  console.log('URL:', page.url());
  
  // Get all text content
  const bodyText = await page.textContent('body');
  
  console.log('\nSearching for scenario keywords...');
  console.log('  "Baseline": ', bodyText?.includes('Baseline'));
  console.log('  "Capacity": ', bodyText?.includes('Capacity'));
  console.log('  "Data Quality": ', bodyText?.includes('Data Quality'));
  console.log('  "Scenario": ', bodyText?.includes('Scenario'));
  
  // Try to find scenario elements
  const scenarioKeywords = ['Baseline Operations', 'Capacity Stress Test', 'Data Quality Issues'];
  
  for (const keyword of scenarioKeywords) {
    const elements = await page.getByText(keyword, { exact: false }).all();
    console.log(`\nElements containing "${keyword}": ${elements.length}`);
  }
  
  // Check for Run button
  const runButtons = await page.locator('button:has-text("Run")').all();
  console.log(`\nRun buttons found: ${runButtons.length}`);
  
  if (runButtons.length === 0) {
    console.log('\n❌ NO RUN BUTTONS - Scenarios not functional in UI');
  }
  
  // Get all headings
  const headings = await page.locator('h1, h2, h3, h4, h5').allTextContents();
  console.log('\nPage headings:', headings);
  
  // Final verdict
  const hasScenarios = bodyText?.includes('Baseline') || bodyText?.includes('Scenario');
  const hasRunButton = runButtons.length > 0;
  
  console.log('\n=== VERDICT ===');
  console.log(`Scenarios visible in UI: ${hasScenarios}`);
  console.log(`Run buttons available: ${hasRunButton}`);
  console.log(`Frontend workflow functional: ${hasScenarios && hasRunButton}`);
});

import { test, expect } from '@playwright/test';
import { setupAuthenticatedPage } from '../helpers';

test('Debug: Inspect scenario rendering on new twin', async ({ page }) => {
  test.setTimeout(90000);
  await setupAuthenticatedPage(page);
  
  console.log('\n=== Getting First Ontology ===');
  const ontologiesResponse = await page.request.get('http://localhost:8080/api/v1/ontologies');
  const ontologies = await ontologiesResponse.json();
  const firstOntology = ontologies[0];
  
  if (!firstOntology) {
    console.log('❌ No ontologies available');
    test.skip();
    return;
  }
  
  console.log(`Using ontology: ${firstOntology.id} (${firstOntology.name})`);
  
  console.log('\n=== Creating Test Twin via API ===');
  
  // Create twin via API
  const createResponse = await page.request.post('http://localhost:8080/api/v1/twin/create', {
    data: {
      name: 'Debug Scenario Rendering Test',
      description: 'Testing scenario card rendering',
      ontology_id: firstOntology.id,
      model_type: 'organization'
    }
  });
  
  const createData = await createResponse.json();
  console.log('Create response:', JSON.stringify(createData, null, 2));
  
  if (!createData.data?.twin_id) {
    console.log('❌ Failed to create twin');
    test.skip();
    return;
  }
  
  const twinId = createData.data.twin_id;
  console.log(`✅ Created twin: ${twinId}`);
  
  // Wait a moment for backend to process
  await page.waitForTimeout(1000);
  
  console.log('\n=== Checking Scenarios API ===');
  const scenariosResponse = await page.request.get(`http://localhost:8080/api/v1/twin/${twinId}/scenarios`);
  const scenariosData = await scenariosResponse.json();
  console.log('Scenarios API response:', JSON.stringify(scenariosData, null, 2));
  console.log(`Scenarios count: ${scenariosData.data?.count || 0}`);
  
  console.log('\n=== Navigating to Twin Detail Page ===');
  await page.goto(`http://localhost:8080/digital-twins/${twinId}`);
  await page.waitForLoadState('domcontentloaded');
  
  console.log('\n=== Clicking Scenarios Tab ===');
  const scenariosTab = page.getByRole('button', { name: /^Scenarios \(\d+\)$/ });
  await scenariosTab.click();
  await page.waitForTimeout(3000); // Wait for render
  
  console.log('\n=== Inspecting Page Content ===');
  const bodyText = await page.textContent('body');
  console.log('Body contains "Baseline":', bodyText?.includes('Baseline'));
  console.log('Body contains "Capacity":', bodyText?.includes('Capacity'));
  console.log('Body contains "Data Quality":', bodyText?.includes('Data Quality'));
  console.log('Body contains "No Scenarios Yet":', bodyText?.includes('No Scenarios Yet'));
  console.log('Body contains "Run":', bodyText?.includes('Run'));
  
  console.log('\n=== Trying Different Selectors ===');
  
  // Try the test's original selector
  const selector1 = await page.locator('div:has(button:has-text("Run"))').count();
  console.log('div:has(button:has-text("Run")):', selector1);
  
  // Try finding buttons directly
  const selector2 = await page.locator('button:has-text("Run")').count();
  console.log('button:has-text("Run"):', selector2);
  
  // Try finding Cards
  const selector3 = await page.locator('[class*="card"]').count();
  console.log('[class*="card"]:', selector3);
  
  // Try finding scenario names
  const selector4 = await page.getByText('Baseline Operations').count();
  console.log('getByText("Baseline Operations"):', selector4);
  
  console.log('\n=== Taking Screenshot ===');
  await page.screenshot({ path: 'test-results/debug-scenario-rendering.png', fullPage: true });
  
  console.log('\n=== Dumping DOM Structure ===');
  const scenariosTabContent = await page.locator('[role="tabpanel"], .space-y-6').first().innerHTML();
  console.log('Tab content HTML (first 1000 chars):', scenariosTabContent.substring(0, 1000));
  
  // Check network requests
  console.log('\n=== Checking Network Requests ===');
  page.on('response', async (response) => {
    if (response.url().includes('scenarios')) {
      console.log(`Response: ${response.url()} - Status: ${response.status()}`);
    }
  });
  
  // Reload scenarios by switching tabs
  await page.getByRole('button', { name: 'Overview' }).click();
  await page.waitForTimeout(500);
  await page.getByRole('button', { name: /^Scenarios \(\d+\)$/ }).click();
  await page.waitForTimeout(2000);
  
  console.log('\n=== Final Check ===');
  const finalCount = await page.locator('button:has-text("Run")').count();
  console.log(`Final "Run" button count: ${finalCount}`);
  
  if (finalCount === 0) {
    console.log('\n❌ ISSUE CONFIRMED: Scenarios exist in API but not rendered in UI');
    console.log('This is a frontend rendering bug, not a backend issue.');
  } else {
    console.log(`\n✅ Found ${finalCount} Run buttons - scenarios are rendering!`);
  }
});

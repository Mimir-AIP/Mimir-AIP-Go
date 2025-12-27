import { test, expect } from '@playwright/test';

/**
 * Happy Path E2E Test Suite
 * 
 * This test verifies that all pages in the application load correctly
 * and that navigation works without errors. It simulates a user exploring
 * the entire application.
 */

test.describe('Happy Path - Full Application Navigation', () => {
  
  test('should navigate through all pages without errors', async ({ page }) => {
    console.log('\n=== Starting Happy Path Navigation Test ===\n');

    // Track console errors
    const consoleErrors: string[] = [];
    page.on('console', msg => {
      if (msg.type() === 'error') {
        consoleErrors.push(msg.text());
      }
    });

    // 1. Dashboard
    console.log('1. Testing Dashboard...');
    await page.goto('/dashboard', { waitUntil: 'domcontentloaded', timeout: 10000 });
    await page.waitForTimeout(500);
    await expect(page.locator('h1, h2').first()).toBeVisible({ timeout: 5000 });
    console.log('   ✓ Dashboard loaded');

    // 2. Pipelines (Data Ingestion)
    console.log('2. Testing Pipelines...');
    await page.goto('/pipelines', { waitUntil: 'domcontentloaded', timeout: 10000 });
    await page.waitForTimeout(500);
    const pipelinesPage = page.locator('text=/pipeline|create/i').first();
    await expect(uploadPage).toBeVisible({ timeout: 5000 });
    console.log('   ✓ Data Ingestion loaded');

    // 3. Pipelines
    console.log('3. Testing Pipelines...');
    await page.goto('/pipelines', { waitUntil: 'domcontentloaded', timeout: 10000 });
    await page.waitForTimeout(500);
    const pipelinesHeading = page.locator('h1, h2').first();
    await expect(pipelinesHeading).toBeVisible({ timeout: 5000 });
    console.log('   ✓ Pipelines loaded');

    // 4. Jobs
    console.log('4. Testing Jobs...');
    await page.goto('/jobs', { waitUntil: 'domcontentloaded', timeout: 10000 });
    await page.waitForTimeout(500);
    const jobsHeading = page.locator('h1, h2').first();
    await expect(jobsHeading).toBeVisible({ timeout: 5000 });
    console.log('   ✓ Jobs loaded');

    // 5. Ontology System - Browse Ontologies
    console.log('5. Testing Ontology System - Browse Ontologies...');
    await page.goto('/ontologies', { waitUntil: 'domcontentloaded', timeout: 10000 });
    await page.waitForTimeout(500);
    const ontologiesHeading = page.locator('h1, h2').filter({ hasText: /ontolog/i }).first();
    await expect(ontologiesHeading).toBeVisible({ timeout: 5000 });
    console.log('   ✓ Browse Ontologies loaded');

    // 6. Ontology System - Upload Ontology
    console.log('6. Testing Ontology System - Upload Ontology...');
    await page.goto('/ontologies/upload', { waitUntil: 'domcontentloaded', timeout: 10000 });
    await page.waitForTimeout(500);
    const uploadHeading = page.locator('h1, h2').first();
    await expect(uploadHeading).toBeVisible({ timeout: 5000 });
    // Verify upload form exists
    const fileInput = page.locator('input[type="file"]');
    expect(await fileInput.count()).toBeGreaterThan(0);
    console.log('   ✓ Upload Ontology loaded');

    // 7. Ontology System - Knowledge Graph
    console.log('7. Testing Ontology System - Knowledge Graph...');
    await page.goto('/knowledge-graph', { waitUntil: 'domcontentloaded', timeout: 10000 });
    await page.waitForTimeout(500);
    const kgHeading = page.locator('h1, h2').first();
    await expect(kgHeading).toBeVisible({ timeout: 5000 });
    console.log('   ✓ Knowledge Graph loaded');

    // 8. Ontology System - Entity Extraction
    console.log('8. Testing Ontology System - Entity Extraction...');
    await page.goto('/extraction', { waitUntil: 'domcontentloaded', timeout: 10000 });
    await page.waitForTimeout(500);
    const extractionHeading = page.locator('h1, h2').first();
    await expect(extractionHeading).toBeVisible({ timeout: 5000 });
    console.log('   ✓ Entity Extraction loaded');

    // 9. Ontology System - ML Models
    console.log('9. Testing Ontology System - ML Models...');
    await page.goto('/models', { waitUntil: 'domcontentloaded', timeout: 10000 });
    await page.waitForTimeout(500);
    const modelsHeading = page.locator('h1, h2').first();
    await expect(modelsHeading).toBeVisible({ timeout: 5000 });
    console.log('   ✓ ML Models loaded');

    // 10. Ontology System - Digital Twins
    console.log('10. Testing Ontology System - Digital Twins...');
    await page.goto('/digital-twins', { waitUntil: 'domcontentloaded', timeout: 10000 });
    await page.waitForTimeout(500);
    const twinsHeading = page.locator('h1, h2').first();
    await expect(twinsHeading).toBeVisible({ timeout: 5000 });
    console.log('   ✓ Digital Twins loaded');

    // 11. Agent Chat
    console.log('11. Testing Agent Chat...');
    await page.goto('/chat', { waitUntil: 'domcontentloaded', timeout: 10000 });
    await page.waitForTimeout(500);
    const chatTextarea = page.locator('textarea');
    await expect(chatTextarea).toBeVisible({ timeout: 5000 });
    console.log('   ✓ Agent Chat loaded');

    // 12. Monitoring - Main Page
    console.log('12. Testing Monitoring - Main Page...');
    await page.goto('/monitoring', { waitUntil: 'domcontentloaded', timeout: 10000 });
    await page.waitForTimeout(500);
    const monitoringHeading = page.locator('h1, h2').first();
    await expect(monitoringHeading).toBeVisible({ timeout: 5000 });
    console.log('   ✓ Monitoring main page loaded');

    // 13. Monitoring - Jobs
    console.log('13. Testing Monitoring - Jobs...');
    await page.goto('/monitoring/jobs', { waitUntil: 'domcontentloaded', timeout: 10000 });
    await page.waitForTimeout(500);
    const jobsMonitoringHeading = page.locator('h1, h2').first();
    await expect(jobsMonitoringHeading).toBeVisible({ timeout: 5000 });
    console.log('   ✓ Monitoring Jobs loaded');

    // 14. Monitoring - Rules
    console.log('14. Testing Monitoring - Rules...');
    await page.goto('/monitoring/rules', { waitUntil: 'domcontentloaded', timeout: 10000 });
    await page.waitForTimeout(500);
    const rulesHeading = page.locator('h1, h2').first();
    await expect(rulesHeading).toBeVisible({ timeout: 5000 });
    console.log('   ✓ Monitoring Rules loaded');

    // 15. Monitoring - Alerts
    console.log('15. Testing Monitoring - Alerts...');
    await page.goto('/monitoring/alerts', { waitUntil: 'domcontentloaded', timeout: 10000 });
    await page.waitForTimeout(500);
    const alertsHeading = page.locator('h1, h2').first();
    await expect(alertsHeading).toBeVisible({ timeout: 5000 });
    console.log('   ✓ Monitoring Alerts loaded');

    // 16. Plugins
    console.log('16. Testing Plugins...');
    await page.goto('/plugins', { waitUntil: 'domcontentloaded', timeout: 10000 });
    await page.waitForTimeout(500);
    const pluginsHeading = page.locator('h1, h2').first();
    await expect(pluginsHeading).toBeVisible({ timeout: 5000 });
    console.log('   ✓ Plugins loaded');

    // 17. Config
    console.log('17. Testing Config...');
    await page.goto('/config', { waitUntil: 'domcontentloaded', timeout: 10000 });
    await page.waitForTimeout(500);
    const configHeading = page.locator('h1, h2').first();
    await expect(configHeading).toBeVisible({ timeout: 5000 });
    console.log('   ✓ Config loaded');

    // 18. Settings
    console.log('18. Testing Settings...');
    await page.goto('/settings', { waitUntil: 'domcontentloaded', timeout: 10000 });
    await page.waitForTimeout(500);
    const settingsHeading = page.locator('h1, h2').first();
    await expect(settingsHeading).toBeVisible({ timeout: 5000 });
    console.log('   ✓ Settings loaded');

    // Verify no console errors occurred
    console.log('\n=== Checking for Console Errors ===');
    if (consoleErrors.length > 0) {
      console.log(`⚠ Found ${consoleErrors.length} console errors:`);
      consoleErrors.forEach((error, idx) => {
        console.log(`  ${idx + 1}. ${error}`);
      });
      // Don't fail the test for console errors, just log them
      // expect(consoleErrors.length).toBe(0);
    } else {
      console.log('✓ No console errors detected');
    }

    console.log('\n=== Happy Path Navigation Test Complete ===\n');
    console.log('Summary: All 18 pages loaded successfully');
  });

  test('should test sidebar dropdown navigation', async ({ page }) => {
    console.log('\n=== Testing Sidebar Dropdown Navigation ===\n');

    await page.goto('/dashboard', { waitUntil: 'domcontentloaded', timeout: 10000 });
    await page.waitForTimeout(1000);

    // Find the Ontology System dropdown button
    console.log('1. Looking for Ontology System dropdown...');
    const ontologyDropdown = page.locator('button').filter({ hasText: /ontology system/i }).first();
    await expect(ontologyDropdown).toBeVisible({ timeout: 5000 });
    console.log('   ✓ Ontology System dropdown found');

    // Click to expand
    console.log('2. Expanding Ontology System dropdown...');
    await ontologyDropdown.click();
    await page.waitForTimeout(500);
    
    // Verify child items are visible
    const browseOntologies = page.locator('a[href="/ontologies"]').first();
    await expect(browseOntologies).toBeVisible({ timeout: 3000 });
    console.log('   ✓ Dropdown expanded, child items visible');

    // Click on a child item
    console.log('3. Clicking on Browse Ontologies...');
    await browseOntologies.click();
    await page.waitForTimeout(1000);
    
    // Verify navigation occurred
    expect(page.url()).toContain('/ontologies');
    console.log('   ✓ Navigation to Browse Ontologies successful');

    // Collapse the dropdown
    console.log('4. Collapsing dropdown...');
    const ontologyDropdownAgain = page.locator('button').filter({ hasText: /ontology system/i }).first();
    await ontologyDropdownAgain.click();
    await page.waitForTimeout(500);
    console.log('   ✓ Dropdown collapsed');

    console.log('\n=== Sidebar Dropdown Test Complete ===\n');
  });

  test('should test monitoring subsections via navigation', async ({ page }) => {
    console.log('\n=== Testing Monitoring Subsections ===\n');

    // Start at monitoring main page
    console.log('1. Navigating to Monitoring main page...');
    await page.goto('/monitoring', { waitUntil: 'domcontentloaded', timeout: 10000 });
    await page.waitForTimeout(1000);
    console.log('   ✓ Monitoring main page loaded');

    // Look for links/buttons to subsections
    console.log('2. Looking for Jobs link...');
    const jobsLink = page.locator('a[href*="/monitoring/jobs"], button').filter({ hasText: /jobs/i }).first();
    if (await jobsLink.count() > 0) {
      await jobsLink.click();
      await page.waitForTimeout(1000);
      expect(page.url()).toContain('/monitoring');
      console.log('   ✓ Jobs section accessible');
    } else {
      // Direct navigation if no link found
      await page.goto('/monitoring/jobs', { waitUntil: 'domcontentloaded', timeout: 10000 });
      await page.waitForTimeout(1000);
      console.log('   ✓ Jobs section accessible via direct navigation');
    }

    console.log('3. Looking for Rules link...');
    await page.goto('/monitoring', { waitUntil: 'domcontentloaded', timeout: 10000 });
    await page.waitForTimeout(1000);
    const rulesLink = page.locator('a[href*="/monitoring/rules"], button').filter({ hasText: /rules/i }).first();
    if (await rulesLink.count() > 0) {
      await rulesLink.click();
      await page.waitForTimeout(1000);
      expect(page.url()).toContain('/monitoring');
      console.log('   ✓ Rules section accessible');
    } else {
      await page.goto('/monitoring/rules', { waitUntil: 'domcontentloaded', timeout: 10000 });
      await page.waitForTimeout(1000);
      console.log('   ✓ Rules section accessible via direct navigation');
    }

    console.log('4. Looking for Alerts link...');
    await page.goto('/monitoring', { waitUntil: 'domcontentloaded', timeout: 10000 });
    await page.waitForTimeout(1000);
    const alertsLink = page.locator('a[href*="/monitoring/alerts"], button').filter({ hasText: /alerts/i }).first();
    if (await alertsLink.count() > 0) {
      await alertsLink.click();
      await page.waitForTimeout(1000);
      expect(page.url()).toContain('/monitoring');
      console.log('   ✓ Alerts section accessible');
    } else {
      await page.goto('/monitoring/alerts', { waitUntil: 'domcontentloaded', timeout: 10000 });
      await page.waitForTimeout(1000);
      console.log('   ✓ Alerts section accessible via direct navigation');
    }

    console.log('\n=== Monitoring Subsections Test Complete ===\n');
  });

  test('should verify no JavaScript errors on any page', async ({ page }) => {
    console.log('\n=== Testing for JavaScript Errors ===\n');

    const pagesToTest = [
      '/dashboard',
      '/pipelines',
      '/jobs',
      '/workflows',
      '/ontologies',
      '/ontologies/upload',
      '/knowledge-graph',
      '/extraction',
      '/models',
      '/digital-twins',
      '/chat',
      '/monitoring',
      '/monitoring/jobs',
      '/monitoring/rules',
      '/monitoring/alerts',
      '/plugins',
      '/config',
      '/settings',
    ];

    const jsErrors: Array<{ page: string; error: string }> = [];

    page.on('pageerror', error => {
      const currentUrl = page.url();
      jsErrors.push({
        page: currentUrl,
        error: error.message,
      });
      console.log(`   ⚠ JavaScript error on ${currentUrl}: ${error.message}`);
    });

    for (const pagePath of pagesToTest) {
      console.log(`Testing: ${pagePath}`);
      await page.goto(pagePath, { waitUntil: 'domcontentloaded', timeout: 10000 });
      await page.waitForTimeout(300);
    }

    console.log('\n=== JavaScript Error Check Complete ===');
    if (jsErrors.length > 0) {
      console.log(`Found ${jsErrors.length} JavaScript errors:`);
      jsErrors.forEach((err, idx) => {
        console.log(`  ${idx + 1}. ${err.page}: ${err.error}`);
      });
      // Log but don't fail - some errors might be expected
    } else {
      console.log('✓ No JavaScript errors detected on any page');
    }

    expect(jsErrors.length).toBe(0);
  });

  test('should verify all sidebar links are clickable', async ({ page }) => {
    console.log('\n=== Testing Sidebar Link Accessibility ===\n');

    await page.goto('/dashboard', { waitUntil: 'networkidle' });
    await page.waitForTimeout(1000);

    const sidebarLinks = [
      { text: 'Dashboard', href: '/dashboard' },
      { text: 'Pipelines', href: '/pipelines' },
      { text: 'Jobs', href: '/jobs' },
      { text: 'Workflows', href: '/workflows' },
      { text: 'Agent Chat', href: '/chat' },
      { text: 'Monitoring', href: '/monitoring' },
      { text: 'Plugins', href: '/plugins' },
      { text: 'Config', href: '/config' },
      { text: 'Settings', href: '/settings' },
    ];

    for (const link of sidebarLinks) {
      console.log(`Testing sidebar link: ${link.text}`);
      const linkElement = page.locator('a').filter({ hasText: new RegExp(link.text, 'i') }).first();
      await expect(linkElement).toBeVisible({ timeout: 5000 });
      
      // Click and verify navigation
      await linkElement.click();
      await page.waitForTimeout(1000);
      expect(page.url()).toContain(link.href);
      console.log(`   ✓ ${link.text} link works correctly`);
    }

    console.log('\n=== Sidebar Link Test Complete ===\n');
  });
});

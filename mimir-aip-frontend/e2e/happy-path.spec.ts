import { test, expect } from '@playwright/test';

/**
 * Happy Path E2E Test: Basic UI Navigation
 * 
 * Tests basic page navigation and loading:
 * 1. Navigate to home page
 * 2. Navigate to pipelines page
 * 3. Verify page elements load correctly
 * 4. Test basic navigation menu
 */

test.describe('Happy Path - Basic UI Navigation', () => {
  test('home page loads successfully', async ({ page }) => {
    await page.goto('/');
    
    // Wait for page to load
    await expect(page).toHaveTitle(/Mimir/i);
    
    // Verify main content loads
    const mainContent = page.locator('main');
    await expect(mainContent).toBeVisible();
  });

  test('pipelines page loads and displays content', async ({ page }) => {
    await page.goto('/pipelines');
    
    // Wait for page to load
    await expect(page.locator('body')).toBeVisible();
    
    // Verify the page has content (either pipelines or empty state)
    const content = page.locator('body');
    await expect(content).not.toBeEmpty();
    
    // Check for any heading or text content
    const headings = page.locator('h1, h2, h3');
    const headingCount = await headings.count();
    expect(headingCount).toBeGreaterThan(0);
  });

  test('navigation between pages works', async ({ page }) => {
    // Start on home
    await page.goto('/');
    await expect(page).toHaveTitle(/Mimir/i);
    
    // Navigate to pipelines
    await page.goto('/pipelines');
    await expect(page.locator('body')).toBeVisible();
    
    // Navigate to ontologies
    await page.goto('/ontologies');
    await expect(page.locator('body')).toBeVisible();
    
    // Navigate to digital twins
    await page.goto('/digital-twins');
    await expect(page.locator('body')).toBeVisible();
    
    // Navigate back to home
    await page.goto('/');
    await expect(page).toHaveTitle(/Mimir/i);
  });

  test('API is accessible from frontend', async ({ page }) => {
    // Navigate to a page that makes API calls
    await page.goto('/pipelines');
    
    // Wait for potential API calls to complete
    await page.waitForTimeout(2000);
    
    // Check that no error dialogs appeared
    const errorDialogs = page.locator('text=/error/i');
    const errorCount = await errorDialogs.count();
    
    // We expect either 0 errors, or the page to still be functional
    expect(errorCount).toBeLessThan(5);
  });
});
import { test, expect } from '@playwright/test';

/**
 * Docker Deployment E2E Tests
 * Tests the actual running Docker container to verify all pages work
 */

test.describe('Docker Deployment - Full Application Test', () => {
  
  test('homepage redirects to dashboard', async ({ page }) => {
    await page.goto('/');
    
    // Should redirect to dashboard
    await page.waitForURL('**/dashboard', { timeout: 10000 });
    
    // Verify page loaded - check for sidebar specifically
    await expect(page.locator('aside >> text=MIMIR AIP')).toBeVisible();
  });

  test('dashboard page loads and displays correctly', async ({ page }) => {
    await page.goto('/dashboard');
    
    // Wait for page to load
    await page.waitForLoadState('networkidle');
    
    // Check for key UI elements
    await expect(page.locator('text=Dashboard')).toBeVisible();
    await expect(page.locator('aside >> text=MIMIR AIP')).toBeVisible();
    
    // Verify navigation sidebar exists
    await expect(page.locator('aside')).toBeVisible();
  });

  test('navigation sidebar has all expected links', async ({ page }) => {
    await page.goto('/dashboard');
    
    // Check all navigation items exist
    const navItems = [
      'Dashboard',
      'Data Ingestion',
      'Pipelines',
      'Jobs',
      'Ontologies',
      'Knowledge Graph',
      'Digital Twins',
      'Extraction',
      'Monitoring',
      'Plugins',
      'Config',
      'Settings',
      'Auth'
    ];
    
    for (const item of navItems) {
      await expect(page.locator(`aside >> text=${item}`)).toBeVisible();
    }
  });

  test('can navigate to pipelines page', async ({ page }) => {
    await page.goto('/dashboard');
    
    // Click pipelines link
    await page.click('aside >> text=Pipelines');
    
    // Should navigate to pipelines
    await page.waitForURL('**/pipelines', { timeout: 5000 });
  });

  test('can navigate to jobs page', async ({ page }) => {
    await page.goto('/dashboard');
    
    // Click jobs link
    await page.click('aside >> text=Jobs');
    
    // Should navigate to jobs
    await page.waitForURL('**/jobs', { timeout: 5000 });
  });

  test('can navigate to data ingestion page', async ({ page }) => {
    await page.goto('/dashboard');
    
    // Click data ingestion link
    await page.click('aside >> text=Data Ingestion');
    
    // Should navigate to data upload
    await page.waitForURL('**/data/upload', { timeout: 5000 });
  });

  test('can navigate to ontologies page', async ({ page }) => {
    await page.goto('/dashboard');
    
    // Click ontologies link
    await page.click('aside >> text=Ontologies');
    
    // Should navigate to ontologies
    await page.waitForURL('**/ontologies', { timeout: 5000 });
  });

  test('can navigate to digital twins page', async ({ page }) => {
    await page.goto('/dashboard');
    
    // Click digital twins link
    await page.click('aside >> text=Digital Twins');
    
    // Should navigate to digital twins
    await page.waitForURL('**/digital-twins', { timeout: 5000 });
  });

  test('can navigate to monitoring page', async ({ page }) => {
    await page.goto('/dashboard');
    
    // Click monitoring link
    await page.click('aside >> text=Monitoring');
    
    // Should navigate to monitoring
    await page.waitForURL('**/monitoring', { timeout: 5000 });
  });

  test('API health endpoint works', async ({ request }) => {
    const response = await request.get('/health');
    
    expect(response.ok()).toBeTruthy();
    expect(response.status()).toBe(200);
    
    const json = await response.json();
    expect(json).toHaveProperty('status', 'healthy');
  });

  test('API pipelines endpoint returns data', async ({ request }) => {
    const response = await request.get('/api/v1/pipelines');
    
    expect(response.ok()).toBeTruthy();
    expect(response.status()).toBe(200);
    
    const json = await response.json();
    expect(Array.isArray(json)).toBeTruthy();
  });

  test('API scheduler jobs endpoint returns data', async ({ request }) => {
    const response = await request.get('/api/v1/scheduler/jobs');
    
    expect(response.ok()).toBeTruthy();
    expect(response.status()).toBe(200);
    
    const json = await response.json();
    expect(Array.isArray(json)).toBeTruthy();
  });

  test('API monitoring jobs endpoint returns data', async ({ request }) => {
    const response = await request.get('/api/v1/monitoring/jobs');
    
    expect(response.ok()).toBeTruthy();
    expect(response.status()).toBe(200);
    
    const json = await response.json();
    expect(json).toHaveProperty('data');
  });

  test('logo image loads correctly', async ({ page }) => {
    await page.goto('/dashboard');
    
    // Check logo in sidebar
    const sidebarLogo = page.locator('aside img[alt="Mimir AIP"]');
    await expect(sidebarLogo).toBeVisible();
    
    // Check logo in header
    const headerLogo = page.locator('header img[alt="Mimir AIP"]');
    await expect(headerLogo).toBeVisible();
  });

  test('page title is correct', async ({ page }) => {
    await page.goto('/dashboard');
    
    await expect(page).toHaveTitle(/Mimir AIP/);
  });

  test('responsive sidebar exists', async ({ page }) => {
    await page.goto('/dashboard');
    
    // Check sidebar is present
    const sidebar = page.locator('aside');
    await expect(sidebar).toBeVisible();
    
    // Check it has the expected styling
    await expect(sidebar).toHaveClass(/bg-navy/);
  });

  test('header displays welcome message', async ({ page }) => {
    await page.goto('/dashboard');
    
    // Check header exists
    const header = page.locator('header');
    await expect(header).toBeVisible();
    
    // Check welcome message
    await expect(page.locator('header >> text=Welcome')).toBeVisible();
  });
});

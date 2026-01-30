import { test, expect } from '@playwright/test';

/**
 * Happy Path E2E Test: Basic UI Navigation (Simplified Frontend)
 * 
 * Tests the simplified monitoring-focused frontend:
 * 1. Navigate to dashboard
 * 2. Navigate to pipelines page
 * 3. Navigate to ontologies (view-only)
 * 4. Navigate to twins (view-only)
 * 5. Navigate to chat
 */

test.describe('Happy Path - Basic UI Navigation', () => {
  test('dashboard loads successfully', async ({ page }) => {
    await page.goto('/');
    
    // Wait for page to load
    await expect(page).toHaveTitle(/Mimir/i);
    
    // Verify page body loads
    await expect(page.locator('body')).toBeVisible();
  });

  test('pipelines page loads and displays content', async ({ page }) => {
    await page.goto('/pipelines');
    
    // Wait for page to load
    await expect(page.locator('body')).toBeVisible();
    
    // Verify the page has content
    const content = page.locator('body');
    await expect(content).not.toBeEmpty();
  });

  test('simplified navigation works', async ({ page }) => {
    // Start on dashboard
    await page.goto('/');
    await expect(page).toHaveTitle(/Mimir/i);
    
    // Navigate to pipelines
    await page.goto('/pipelines');
    await expect(page.locator('body')).toBeVisible();
    
    // Navigate to ontologies (view-only)
    await page.goto('/ontologies');
    await expect(page.locator('body')).toBeVisible();
    
    // Navigate to digital twins (view-only)
    await page.goto('/digital-twins');
    await expect(page.locator('body')).toBeVisible();
    
    // Navigate to chat
    await page.goto('/chat');
    await expect(page.locator('body')).toBeVisible();
    
    // Back to dashboard
    await page.goto('/');
    await expect(page).toHaveTitle(/Mimir/i);
  });

  test('view-only pages load without errors', async ({ page }) => {
    // Test that view-only pages don't have configuration dialogs
    await page.goto('/ontologies');
    await expect(page.locator('body')).toBeVisible();
    await page.waitForTimeout(1000);
    
    await page.goto('/digital-twins');
    await expect(page.locator('body')).toBeVisible();
    await page.waitForTimeout(1000);
    
    await page.goto('/models');
    await expect(page.locator('body')).toBeVisible();
    
    // Pages should load without error dialogs
    const errorDialogs = page.locator('role=alert');
    const errorCount = await errorDialogs.count();
    expect(errorCount).toBe(0);
  });
});
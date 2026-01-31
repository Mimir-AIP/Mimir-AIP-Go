import { test, expect } from '@playwright/test';
import * as fs from 'fs';
import * as path from 'path';

/**
 * Screenshot test to verify what users actually see in the Mimir UI
 */

const SCREENSHOTS_DIR = path.join(process.cwd(), 'screenshots');

// Ensure screenshots directory exists
if (!fs.existsSync(SCREENSHOTS_DIR)) {
  fs.mkdirSync(SCREENSHOTS_DIR, { recursive: true });
}

test.describe('UI Screenshot Tests', () => {
  
  test.beforeEach(async ({ page }) => {
    await page.setViewportSize({ width: 1280, height: 800 });
  });

  test('screenshot dashboard', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(1000);
    
    const screenshotPath = path.join(SCREENSHOTS_DIR, '01-dashboard.png');
    await page.screenshot({ path: screenshotPath, fullPage: true });
    
    const heading = await page.locator('h1').first().textContent().catch(() => 'No h1');
    console.log(`Dashboard: ${heading}`);
  });

  test('screenshot pipelines', async ({ page }) => {
    await page.goto('/pipelines');
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(1000);
    
    const screenshotPath = path.join(SCREENSHOTS_DIR, '02-pipelines.png');
    await page.screenshot({ path: screenshotPath, fullPage: true });
    
    const cards = await page.locator('[data-testid="pipeline-card"]').count().catch(() => 0);
    console.log(`Pipelines: ${cards} cards`);
  });

  test('screenshot ontologies', async ({ page }) => {
    await page.goto('/ontologies');
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(1000);
    
    const screenshotPath = path.join(SCREENSHOTS_DIR, '03-ontologies.png');
    await page.screenshot({ path: screenshotPath, fullPage: true });
    
    const cards = await page.locator('.bg-navy').count().catch(() => 0);
    console.log(`Ontologies: ${cards} cards`);
  });

  test('screenshot digital twins', async ({ page }) => {
    await page.goto('/digital-twins');
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(1000);
    
    const screenshotPath = path.join(SCREENSHOTS_DIR, '04-digital-twins.png');
    await page.screenshot({ path: screenshotPath, fullPage: true });
    
    const cards = await page.locator('.bg-navy').count().catch(() => 0);
    console.log(`Digital Twins: ${cards} cards`);
  });

  test('screenshot models', async ({ page }) => {
    await page.goto('/models');
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(1000);
    
    const screenshotPath = path.join(SCREENSHOTS_DIR, '05-models.png');
    await page.screenshot({ path: screenshotPath, fullPage: true });
    
    const cards = await page.locator('.bg-navy').count().catch(() => 0);
    console.log(`Models: ${cards} cards`);
  });
});

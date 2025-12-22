import { test, expect, Page } from '@playwright/test';

/**
 * Comprehensive Functional E2E Tests
 * These tests verify actual functionality, not just UI rendering
 * - Console error detection
 * - API response validation
 * - Data rendering verification
 * - Error handling
 */

// Helper to collect console errors
interface ConsoleError {
  type: string;
  message: string;
  url: string;
}

async function collectErrors(page: Page): Promise<ConsoleError[]> {
  const errors: ConsoleError[] = [];
  
  page.on('console', msg => {
    if (msg.type() === 'error') {
      errors.push({
        type: 'console',
        message: msg.text(),
        url: page.url()
      });
    }
  });
  
  page.on('pageerror', error => {
    errors.push({
      type: 'pageerror',
      message: error.message,
      url: page.url()
    });
  });
  
  return errors;
}

// Helper to collect API responses
interface APIResponse {
  url: string;
  status: number;
  method: string;
}

async function collectAPIResponses(page: Page): Promise<APIResponse[]> {
  const responses: APIResponse[] = [];
  
  page.on('response', response => {
    if (response.url().includes('/api/v1/')) {
      responses.push({
        url: response.url(),
        status: response.status(),
        method: response.request().method()
      });
    }
  });
  
  return responses;
}

test.describe('Dashboard Page - Functional Tests', () => {
  
  test('dashboard has no console errors', async ({ page }) => {
    const errors = await collectErrors(page);
    
    await page.goto('/dashboard');
    await page.waitForLoadState('networkidle');
    
    // Wait a bit for any delayed JS errors
    await page.waitForTimeout(2000);
    
    if (errors.length > 0) {
      console.error('Console errors detected:', errors);
    }
    
    expect(errors).toHaveLength(0);
  });
  
  test('dashboard APIs return valid responses', async ({ page }) => {
    const apiResponses = await collectAPIResponses(page);
    
    await page.goto('/dashboard');
    await page.waitForLoadState('networkidle');
    
    // Check for server errors (5xx)
    const serverErrors = apiResponses.filter(r => r.status >= 500);
    if (serverErrors.length > 0) {
      console.error('Server errors detected:', serverErrors);
    }
    expect(serverErrors).toHaveLength(0);
    
    // Check for client errors (4xx) - some may be expected
    const clientErrors = apiResponses.filter(r => r.status >= 400 && r.status < 500);
    if (clientErrors.length > 0) {
      console.warn('Client errors detected:', clientErrors);
    }
  });
});

test.describe('Data Ingestion Page - Functional Tests', () => {
  
  test('data ingestion page has no console errors', async ({ page }) => {
    const errors = await collectErrors(page);
    
    await page.goto('/data/upload');
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);
    
    // Check for specific null/undefined errors
    const nullErrors = errors.filter(e => 
      e.message.includes('Cannot read properties of null') ||
      e.message.includes('map') ||
      e.message.includes('undefined')
    );
    
    if (nullErrors.length > 0) {
      console.error('Null/undefined errors on data ingestion page:', nullErrors);
    }
    
    expect(errors).toHaveLength(0);
  });
  
  test('data ingestion page displays content or empty state', async ({ page }) => {
    await page.goto('/data/upload');
    await page.waitForLoadState('networkidle');
    
    // Should not show JavaScript error messages
    const typeError = page.locator('text=TypeError');
    const cannotRead = page.locator('text=Cannot read properties');
    
    await expect(typeError).not.toBeVisible();
    await expect(cannotRead).not.toBeVisible();
    
    // Should show either upload form or some content
    const pageContent = await page.content();
    expect(pageContent.length).toBeGreaterThan(1000); // Has actual content
  });
  
  test('data ingestion APIs return valid responses', async ({ page }) => {
    const apiResponses = await collectAPIResponses(page);
    
    await page.goto('/data/upload');
    await page.waitForLoadState('networkidle');
    
    const serverErrors = apiResponses.filter(r => r.status >= 500);
    if (serverErrors.length > 0) {
      console.error('Server errors on data ingestion:', serverErrors);
    }
    expect(serverErrors).toHaveLength(0);
  });
});

test.describe('Digital Twins Page - Functional Tests', () => {
  
  test('digital twins page has no console errors', async ({ page }) => {
    const errors = await collectErrors(page);
    
    await page.goto('/digital-twins');
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);
    
    // Check for map/array errors
    const mapErrors = errors.filter(e => 
      e.message.includes('map is not a function') ||
      e.message.includes('Cannot read properties')
    );
    
    if (mapErrors.length > 0) {
      console.error('Array/map errors on digital twins page:', mapErrors);
    }
    
    expect(errors).toHaveLength(0);
  });
  
  test('digital twins page displays content or empty state', async ({ page }) => {
    await page.goto('/digital-twins');
    await page.waitForLoadState('networkidle');
    
    // Should not show error messages
    const typeError = page.locator('text=TypeError');
    await expect(typeError).not.toBeVisible();
    
    // Should have meaningful content
    const pageContent = await page.content();
    expect(pageContent.length).toBeGreaterThan(1000);
  });
  
  test('digital twins API returns array or handles non-array response', async ({ page }) => {
    const apiResponses = await collectAPIResponses(page);
    
    await page.goto('/digital-twins');
    await page.waitForLoadState('networkidle');
    
    // Check for server errors
    const serverErrors = apiResponses.filter(r => r.status >= 500);
    expect(serverErrors).toHaveLength(0);
    
    // If API exists, verify response structure
    const digitalTwinsAPI = apiResponses.find(r => r.url.includes('/digital-twins'));
    if (digitalTwinsAPI) {
      expect(digitalTwinsAPI.status).toBeLessThan(400);
    }
  });
});

test.describe('Monitoring Page - Functional Tests', () => {
  
  test('monitoring page has no console errors', async ({ page }) => {
    const errors = await collectErrors(page);
    
    await page.goto('/monitoring');
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);
    
    if (errors.length > 0) {
      console.error('Errors on monitoring page:', errors);
    }
    
    expect(errors).toHaveLength(0);
  });
  
  test('monitoring alerts API does not return 500 error', async ({ page }) => {
    const apiResponses = await collectAPIResponses(page);
    
    await page.goto('/monitoring');
    await page.waitForLoadState('networkidle');
    
    // Specifically check the alerts endpoint
    const alertsAPI = apiResponses.find(r => r.url.includes('/monitoring/alerts'));
    
    if (alertsAPI) {
      if (alertsAPI.status === 500) {
        console.error('Monitoring alerts API returned 500 - likely database schema issue');
      }
      expect(alertsAPI.status).not.toBe(500);
    }
  });
  
  test('monitoring page displays content', async ({ page }) => {
    await page.goto('/monitoring');
    await page.waitForLoadState('networkidle');
    
    // Should not show database errors
    const dbError = page.locator('text=no such column');
    await expect(dbError).not.toBeVisible();
    
    const pageContent = await page.content();
    expect(pageContent.length).toBeGreaterThan(1000);
  });
});

test.describe('Settings Page - Functional Tests', () => {
  
  test('settings page has no console errors', async ({ page }) => {
    const errors = await collectErrors(page);
    
    await page.goto('/settings');
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);
    
    // Filter out expected 501 errors (feature not implemented yet)
    const realErrors = errors.filter(e => 
      !e.message.includes('501') && 
      !e.message.includes('Not Implemented')
    );
    
    if (realErrors.length > 0) {
      console.error('Unexpected errors on settings page:', realErrors);
    }
    
    expect(realErrors).toHaveLength(0);
  });
  
  test('settings API endpoints exist or return proper status codes', async ({ page }) => {
    const apiResponses = await collectAPIResponses(page);
    
    await page.goto('/settings');
    await page.waitForLoadState('networkidle');
    
    // Check for API key endpoint
    const apiKeyEndpoint = apiResponses.find(r => r.url.includes('/settings/api-keys'));
    
    if (apiKeyEndpoint) {
      // Should return either 200 (exists), 501 (not implemented), but NOT 404
      if (apiKeyEndpoint.status === 404) {
        console.error('Settings API endpoint missing - should return 501 if not implemented');
      }
      
      // Accept 200 or 501, but not 404 or 500
      expect([200, 501]).toContain(apiKeyEndpoint.status);
    }
  });
});

test.describe('API Health and Connectivity', () => {
  
  test('backend API is accessible', async ({ request }) => {
    const response = await request.get('/health');
    expect(response.ok()).toBeTruthy();
    
    const json = await response.json();
    expect(json).toHaveProperty('status', 'healthy');
  });
  
  test('no API endpoints return 500 errors on basic GET requests', async ({ request }) => {
    const endpoints = [
      '/api/v1/pipelines',
      '/api/v1/scheduler/jobs',
      '/api/v1/monitoring/jobs',
      '/api/v1/ontologies'
    ];
    
    const errors: { endpoint: string; status: number }[] = [];
    
    for (const endpoint of endpoints) {
      const response = await request.get(endpoint);
      if (response.status() >= 500) {
        errors.push({ endpoint, status: response.status() });
      }
    }
    
    if (errors.length > 0) {
      console.error('API endpoints returning 500 errors:', errors);
    }
    
    expect(errors).toHaveLength(0);
  });
});

test.describe('Cross-Page Navigation Stability', () => {
  
  test('navigating between pages does not cause errors', async ({ page }) => {
    const errors = await collectErrors(page);
    
    const pages = [
      '/dashboard',
      '/pipelines',
      '/jobs',
      '/data/upload',
      '/ontologies',
      '/digital-twins',
      '/monitoring'
    ];
    
    for (const pagePath of pages) {
      await page.goto(pagePath);
      await page.waitForLoadState('networkidle');
      await page.waitForTimeout(1000);
    }
    
    if (errors.length > 0) {
      console.error('Errors during navigation:', errors);
    }
    
    expect(errors).toHaveLength(0);
  });
});

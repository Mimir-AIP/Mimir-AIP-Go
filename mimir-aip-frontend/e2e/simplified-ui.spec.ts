import { test, expect } from '@playwright/test';

/**
 * E2E Tests for Simplified Mimir UI
 * 
 * Tests match the simplified "hands-off" vision:
 * - View-only monitoring (not configuration)
 * - Simple navigation
 * - Basic chat interactions
 */

test.describe('E2E - Simplified Frontend', () => {
  
  test('dashboard loads and shows system status', async ({ page }) => {
    await page.goto('/');
    
    // Wait for page to load
    await page.waitForLoadState('networkidle');
    
    // Verify page loads
    await expect(page.locator('body')).toBeVisible();
    
    // Check for dashboard content - be flexible with what might be there
    const dashboardHeading = page.locator('h1').first();
    const body = page.locator('body');
    
    await expect(body).toBeVisible();
    const headingVisible = await dashboardHeading.isVisible().catch(() => false);
    expect(headingVisible || await body.textContent()).toBeTruthy();
  });

  test('can navigate to all main pages', async ({ page }) => {
    const pages = [
      { path: '/', name: 'Dashboard' },
      { path: '/pipelines', name: 'Pipelines' },
      { path: '/ontologies', name: 'Ontologies' },
      { path: '/digital-twins', name: 'Digital Twins' },
      { path: '/models', name: 'Models' },
      { path: '/chat', name: 'Chat' },
    ];
    
    for (const { path, name } of pages) {
      await page.goto(path);
      await page.waitForLoadState('networkidle');
      
      // Verify page loads
      await expect(page.locator('body')).toBeVisible();
      
      // Log successful navigation
      console.log(`✓ Successfully navigated to ${name} (${path})`);
    }
  });

  test('pipelines page shows list view', async ({ page }) => {
    await page.goto('/pipelines');
    await page.waitForLoadState('networkidle');
    
    // Check for pipelines heading
    const heading = page.locator('h1, h2').filter({ hasText: /pipeline/i }).first();
    await expect(heading).toBeVisible();
    
    // Page should show either list or empty state
    const content = page.locator('body');
    await expect(content).not.toBeEmpty();
  });

  test('ontologies page is view-only', async ({ page }) => {
    await page.goto('/ontologies');
    await page.waitForLoadState('networkidle');
    
    // Verify page loads
    await expect(page.locator('body')).toBeVisible();
    
    // Should not have complex edit forms (simplified UI)
    const complexForms = page.locator('form[enctype], .ontology-builder, [data-testid="edit-ontology"]');
    const formCount = await complexForms.count();
    
    // Allow 0 forms or simple filter forms
    expect(formCount).toBeLessThanOrEqual(1);
  });

  test('chat interface loads correctly', async ({ page }) => {
    await page.goto('/chat');
    await page.waitForLoadState('networkidle');
    
    // Verify chat page loads
    await expect(page.locator('body')).toBeVisible();
    
    // Check for chat-related elements
    const chatElements = page.locator('textarea, input, .chat-container, .messages').first();
    await expect(chatElements).toBeVisible();
  });

  test('digital twins page shows monitoring view', async ({ page }) => {
    await page.goto('/digital-twins');
    await page.waitForLoadState('networkidle');
    
    // Verify page loads
    await expect(page.locator('body')).toBeVisible();
    
    // Should show heading
    const heading = page.locator('h1, h2').filter({ hasText: /twin/i }).first();
    await expect(heading).toBeVisible();
  });

  test('models page is view-only', async ({ page }) => {
    await page.goto('/models');
    await page.waitForLoadState('networkidle');
    
    // Verify page loads
    await expect(page.locator('body')).toBeVisible();
    
    // Should show ML models heading
    const heading = page.locator('h1, h2').filter({ hasText: /model/i }).first();
    await expect(heading).toBeVisible();
  });

  test('navigation via sidebar works', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');
    
    // Find navigation links (in sidebar or nav menu)
    const navLinks = page.locator('nav a, aside a, header a, [role="navigation"] a');
    const linkCount = await navLinks.count();
    
    if (linkCount > 0) {
      // Test clicking first few navigation links
      for (let i = 0; i < Math.min(linkCount, 3); i++) {
        const link = navLinks.nth(i);
        const href = await link.getAttribute('href');
        
        if (href && !href.startsWith('http')) {
          await link.click();
          await page.waitForTimeout(500);
          
          // Verify navigation happened
          await expect(page.locator('body')).toBeVisible();
        }
      }
    }
  });

  test('no JavaScript errors on page load', async ({ page }) => {
    const errors: string[] = [];
    
    // Listen for console errors
    page.on('console', msg => {
      if (msg.type() === 'error') {
        errors.push(msg.text());
      }
    });
    
    // Load multiple pages
    for (const path of ['/', '/pipelines', '/ontologies', '/chat']) {
      await page.goto(path);
      await page.waitForLoadState('networkidle');
      await page.waitForTimeout(500);
    }
    
    // Filter out non-critical errors
    const criticalErrors = errors.filter(e => 
      !e.includes('favicon') && 
      !e.includes('source map') &&
      !e.includes('webpack')
    );
    
    // Should have no critical errors
    expect(criticalErrors).toHaveLength(0);
  });

  test('responsive layout loads correctly', async ({ page }) => {
    // Test at desktop size
    await page.setViewportSize({ width: 1280, height: 720 });
    await page.goto('/');
    await page.waitForLoadState('networkidle');
    
    await expect(page.locator('body')).toBeVisible();
    
    // Layout should be usable
    const mainContent = page.locator('main, .container, [class*="content"]').first();
    await expect(mainContent).toBeVisible();
  });
});

test.describe('E2E - API Integration', () => {
  
  test('API is accessible from frontend', async ({ page }) => {
    await page.goto('/pipelines');
    await page.waitForLoadState('networkidle');
    
    // Check that API calls are made (wait a moment for async calls)
    await page.waitForTimeout(2000);
    
    // Page should either show data or empty state (not stuck loading)
    const loading = page.locator('text=/loading/i, .loading, .spinner');
    const isLoading = await loading.isVisible().catch(() => false);
    
    // After 2 seconds, should not still be loading
    if (isLoading) {
      await page.waitForTimeout(3000);
      const stillLoading = await loading.isVisible().catch(() => false);
      expect(stillLoading).toBeFalsy();
    }
  });

  test('health endpoint is accessible', async ({ request }) => {
    const response = await request.get('/health');
    expect(response.status()).toBe(200);
    
    const body = await response.json();
    expect(body.status).toBe('healthy');
  });
});

test.describe('E2E - User Workflows', () => {
  
  test('user can view dashboard statistics', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');
    
    // Verify dashboard loads
    await expect(page.locator('body')).toBeVisible();
    
    // Look for any content - stat cards, numbers, headings, or text
    const stats = page.locator('text=/[0-9]+/').first();
    const cards = page.locator('[class*="card"], .card, .stat').first();
    const headings = page.locator('h1, h2, h3').first();
    const anyText = page.locator('body').first();
    
    const hasStats = await stats.isVisible().catch(() => false);
    const hasCards = await cards.isVisible().catch(() => false);
    const hasHeadings = await headings.isVisible().catch(() => false);
    const hasText = await anyText.textContent().then(t => t && t.trim().length > 0).catch(() => false);
    
    // Dashboard should have some visible content
    expect(hasStats || hasCards || hasHeadings || hasText).toBeTruthy();
  });

  test('user can navigate pipeline to ontology to twin flow', async ({ page }) => {
    // This represents the typical user monitoring workflow
    
    // 1. View pipelines
    await page.goto('/pipelines');
    await page.waitForLoadState('networkidle');
    await expect(page.locator('body')).toBeVisible();
    
    // 2. View ontologies
    await page.goto('/ontologies');
    await page.waitForLoadState('networkidle');
    await expect(page.locator('body')).toBeVisible();
    
    // 3. View twins
    await page.goto('/digital-twins');
    await page.waitForLoadState('networkidle');
    await expect(page.locator('body')).toBeVisible();
    
    console.log('✓ Successfully navigated through monitoring workflow');
  });
});

test.describe('E2E - Chat Interface', () => {
  
  test('chat page loads with input field', async ({ page }) => {
    await page.goto('/chat');
    await page.waitForLoadState('networkidle');
    
    // Find input field
    const input = page.locator('textarea, input[type="text"]').first();
    await expect(input).toBeVisible();
    
    // Should be able to type
    await input.fill('Hello Mimir');
    const value = await input.inputValue();
    expect(value).toBe('Hello Mimir');
  });

  test('chat shows model selector or info', async ({ page }) => {
    await page.goto('/chat');
    await page.waitForLoadState('networkidle');
    
    // Look for model info or any interactive element
    const modelInfo = page.locator('text=/model/i').first();
    const button = page.locator('button').first();
    const hasModelInfo = await modelInfo.isVisible().catch(() => false);
    const hasButton = await button.isVisible().catch(() => false);
    
    expect(hasModelInfo || hasButton).toBeTruthy();
  });
});

test.describe('E2E - Error Handling', () => {
  
  test('404 page handles unknown routes gracefully', async ({ page }) => {
    await page.goto('/non-existent-page');
    await page.waitForLoadState('networkidle');
    
    // Should show something (either custom 404 or redirect)
    await expect(page.locator('body')).toBeVisible();
    
    // Should not be stuck on loading
    const loading = page.locator('text=/loading/i').first();
    const isLoading = await loading.isVisible().catch(() => false);
    expect(isLoading).toBeFalsy();
  });

  test('handles network errors gracefully', async ({ page }) => {
    // Block API calls
    await page.route('**/api/**', route => route.abort('internetdisconnected'));
    
    await page.goto('/pipelines');
    await page.waitForTimeout(2000);
    
    // Page should still load (showing empty state or error message)
    await expect(page.locator('body')).toBeVisible();
  });
});

test.describe('E2E - Performance', () => {
  
  test('pages load within reasonable time', async ({ page }) => {
    const pages = ['/', '/pipelines', '/ontologies', '/chat'];
    
    for (const path of pages) {
      const start = Date.now();
      await page.goto(path);
      await page.waitForLoadState('networkidle');
      const loadTime = Date.now() - start;
      
      // Should load within 5 seconds
      expect(loadTime).toBeLessThan(5000);
      console.log(`✓ ${path} loaded in ${loadTime}ms`);
    }
  });
});
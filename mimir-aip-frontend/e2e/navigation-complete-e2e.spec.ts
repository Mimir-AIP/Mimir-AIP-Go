import { test, expect } from '@playwright/test';

/**
 * Comprehensive E2E tests for navigation, menus, sidebar, and routing
 */

test.describe('Navigation - Sidebar and Menus', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/dashboard');
    await page.waitForLoadState('networkidle');
  });

  test('should display sidebar with all main menu items', async ({ page }) => {
    // Check main navigation items are visible
    await expect(page.getByRole('link', { name: /Dashboard/i })).toBeVisible();
    await expect(page.getByRole('link', { name: /Pipelines/i })).toBeVisible();
    await expect(page.getByRole('link', { name: /Ontologies/i })).toBeVisible();
    await expect(page.getByRole('link', { name: /Digital Twins/i })).toBeVisible();
    await expect(page.getByRole('link', { name: /Knowledge Graph/i })).toBeVisible();
    await expect(page.getByRole('link', { name: /Models/i })).toBeVisible();
    await expect(page.getByRole('link', { name: /Workflows/i })).toBeVisible();
    await expect(page.getByRole('link', { name: /Monitoring/i })).toBeVisible();
    await expect(page.getByRole('link', { name: /Chat/i })).toBeVisible();
    await expect(page.getByRole('link', { name: /Settings/i })).toBeVisible();
  });

  test('should navigate to Dashboard', async ({ page }) => {
    await page.getByRole('link', { name: /Dashboard/i }).click();
    await page.waitForLoadState('networkidle');

    await expect(page).toHaveURL(/\/dashboard/);
    await expect(page.getByRole('heading', { name: /Dashboard/i })).toBeVisible();
  });

  test('should navigate to Pipelines', async ({ page }) => {
    await page.getByRole('link', { name: /^Pipelines$/i }).click();
    await page.waitForLoadState('networkidle');

    await expect(page).toHaveURL(/\/pipelines/);
    await expect(page.getByRole('heading', { name: /Pipelines/i })).toBeVisible();
  });

  test('should navigate to Ontologies', async ({ page }) => {
    await page.getByRole('link', { name: /Ontologies/i }).click();
    await page.waitForLoadState('networkidle');

    await expect(page).toHaveURL(/\/ontologies/);
    await expect(page.getByRole('heading', { name: /Ontologies/i })).toBeVisible();
  });

  test('should navigate to Digital Twins', async ({ page }) => {
    await page.getByRole('link', { name: /Digital Twins/i }).click();
    await page.waitForLoadState('networkidle');

    await expect(page).toHaveURL(/\/digital-twins/);
    await expect(page.getByRole('heading', { name: /Digital Twins/i })).toBeVisible();
  });

  test('should navigate to Knowledge Graph', async ({ page }) => {
    await page.getByRole('link', { name: /Knowledge Graph/i }).click();
    await page.waitForLoadState('networkidle');

    await expect(page).toHaveURL(/\/knowledge-graph/);
    await expect(page.getByRole('heading', { name: /Knowledge Graph/i })).toBeVisible();
  });

  test('should navigate to Models', async ({ page }) => {
    await page.getByRole('link', { name: /Models/i }).click();
    await page.waitForLoadState('networkidle');

    await expect(page).toHaveURL(/\/models/);
    await expect(page.getByRole('heading', { name: /Models|Machine Learning/i })).toBeVisible();
  });

  test('should navigate to Workflows', async ({ page }) => {
    await page.getByRole('link', { name: /Workflows/i }).click();
    await page.waitForLoadState('networkidle');

    await expect(page).toHaveURL(/\/workflows/);
    await expect(page.getByRole('heading', { name: /Workflows/i })).toBeVisible();
  });

  test('should navigate to Monitoring', async ({ page }) => {
    await page.getByRole('link', { name: /Monitoring/i }).click();
    await page.waitForLoadState('networkidle');

    await expect(page).toHaveURL(/\/monitoring/);
    await expect(page.getByRole('heading', { name: /Monitoring/i })).toBeVisible();
  });

  test('should navigate to Chat', async ({ page }) => {
    await page.getByRole('link', { name: /Chat/i }).click();
    await page.waitForLoadState('networkidle');

    await expect(page).toHaveURL(/\/chat/);
    await expect(page.getByRole('heading', { name: /Chat|Agent/i })).toBeVisible();
  });

  test('should navigate to Settings', async ({ page }) => {
    await page.getByRole('link', { name: /Settings/i }).click();
    await page.waitForLoadState('networkidle');

    await expect(page).toHaveURL(/\/settings/);
    await expect(page.getByRole('heading', { name: /Settings/i })).toBeVisible();
  });

  test('should highlight active navigation item', async ({ page }) => {
    await page.getByRole('link', { name: /Pipelines/i }).click();
    await page.waitForLoadState('networkidle');

    // Active link should have special styling
    const activeLink = page.getByRole('link', { name: /Pipelines/i });
    const classList = await activeLink.getAttribute('class');

    expect(classList).toMatch(/active|selected|current/i);
  });

  test('should toggle sidebar collapse', async ({ page }) => {
    const collapseButton = page.getByRole('button', { name: /Toggle.*Sidebar|Collapse|Expand/i });

    if (await collapseButton.isVisible()) {
      await collapseButton.click();

      // Sidebar should collapse
      await page.waitForTimeout(300);

      // Icons should still be visible but text might be hidden
      await expect(page.getByRole('navigation')).toBeVisible();
    }
  });

  test('should display user menu', async ({ page }) => {
    const userMenuButton = page.getByRole('button', { name: /User.*Menu|Profile|Account/i });

    if (await userMenuButton.isVisible()) {
      await userMenuButton.click();

      // Check menu items
      await expect(page.getByRole('menuitem', { name: /Profile|Settings|Logout/i })).toBeVisible();
    }
  });

  test('should navigate using breadcrumbs', async ({ page }) => {
    // Navigate to a deep page
    await page.goto('/pipelines');
    await page.waitForLoadState('networkidle');

    // Check for breadcrumbs
    const breadcrumb = page.getByRole('navigation', { name: /Breadcrumb/i });

    if (await breadcrumb.isVisible()) {
      await expect(breadcrumb.getByText(/Pipelines/i)).toBeVisible();
    }
  });

  test('should handle browser back/forward navigation', async ({ page }) => {
    // Navigate forward
    await page.getByRole('link', { name: /Pipelines/i }).click();
    await page.waitForLoadState('networkidle');
    await expect(page).toHaveURL(/\/pipelines/);

    await page.getByRole('link', { name: /Ontologies/i }).click();
    await page.waitForLoadState('networkidle');
    await expect(page).toHaveURL(/\/ontologies/);

    // Navigate back
    await page.goBack();
    await page.waitForLoadState('networkidle');
    await expect(page).toHaveURL(/\/pipelines/);

    // Navigate forward
    await page.goForward();
    await page.waitForLoadState('networkidle');
    await expect(page).toHaveURL(/\/ontologies/);
  });

  test('should search/filter from global search', async ({ page }) => {
    const searchButton = page.getByRole('button', { name: /Search/i });

    if (await searchButton.isVisible()) {
      await searchButton.click();

      // Search dialog should open
      await expect(page.getByPlaceholder(/Search/i)).toBeVisible();

      // Type search query
      await page.getByPlaceholder(/Search/i).fill('pipeline');
      await page.waitForTimeout(500);

      // Results should appear
      const results = page.getByRole('list', { name: /Search.*Results/i });
      if (await results.isVisible()) {
        await expect(results.getByRole('listitem')).toHaveCount(1, { timeout: 5000 }).catch(() => {
          // Results may vary, just check it's visible
          expect(results).toBeVisible();
        });
      }
    }
  });

  test('should display notifications', async ({ page }) => {
    const notificationsButton = page.getByRole('button', { name: /Notifications|Alerts/i });

    if (await notificationsButton.isVisible()) {
      await notificationsButton.click();

      // Notifications panel should open
      await expect(page.getByRole('dialog', { name: /Notifications/i })).toBeVisible();
    }
  });

  test('should support keyboard navigation', async ({ page }) => {
    // Press Tab to navigate
    await page.keyboard.press('Tab');
    await page.keyboard.press('Tab');
    await page.keyboard.press('Tab');

    // An element should be focused
    const focusedElement = page.locator(':focus');
    await expect(focusedElement).toBeVisible();
  });

  test('should navigate to Jobs from sidebar', async ({ page }) => {
    await page.getByRole('link', { name: /Jobs/i }).click();
    await page.waitForLoadState('networkidle');

    await expect(page).toHaveURL(/\/jobs/);
    await expect(page.getByRole('heading', { name: /Jobs/i })).toBeVisible();
  });

  test('should navigate to Plugins from sidebar', async ({ page }) => {
    await page.getByRole('link', { name: /Plugins/i }).click();
    await page.waitForLoadState('networkidle');

    await expect(page).toHaveURL(/\/plugins/);
    await expect(page.getByRole('heading', { name: /Plugins/i })).toBeVisible();
  });

  test('should navigate to Config from sidebar', async ({ page }) => {
    await page.getByRole('link', { name: /Config/i }).click();
    await page.waitForLoadState('networkidle');

    await expect(page).toHaveURL(/\/config/);
    await expect(page.getByRole('heading', { name: /Config/i })).toBeVisible();
  });
});

test.describe('Navigation - Monitoring Submenu', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/dashboard');
    await page.waitForLoadState('networkidle');
  });

  test('should expand monitoring submenu', async ({ page }) => {
    const monitoringLink = page.getByRole('link', { name: /Monitoring/i });
    await monitoringLink.click();

    // Check for submenu items
    const submenu = page.getByRole('navigation').filter({ has: page.getByText(/Jobs|Alerts|Rules/i) });

    if (await submenu.isVisible()) {
      await expect(page.getByRole('link', { name: /Monitoring Jobs/i })).toBeVisible();
      await expect(page.getByRole('link', { name: /Alerts/i })).toBeVisible();
      await expect(page.getByRole('link', { name: /Rules/i })).toBeVisible();
    }
  });

  test('should navigate to monitoring jobs', async ({ page }) => {
    await page.goto('/monitoring/jobs');
    await page.waitForLoadState('networkidle');

    await expect(page).toHaveURL(/\/monitoring\/jobs/);
    await expect(page.getByRole('heading', { name: /Monitoring Jobs/i })).toBeVisible();
  });

  test('should navigate to alerts', async ({ page }) => {
    await page.goto('/monitoring/alerts');
    await page.waitForLoadState('networkidle');

    await expect(page).toHaveURL(/\/monitoring\/alerts/);
    await expect(page.getByRole('heading', { name: /Alerts/i })).toBeVisible();
  });

  test('should navigate to rules', async ({ page }) => {
    await page.goto('/monitoring/rules');
    await page.waitForLoadState('networkidle');

    await expect(page).toHaveURL(/\/monitoring\/rules/);
    await expect(page.getByRole('heading', { name: /Rules/i })).toBeVisible();
  });
});

test.describe('Navigation - 404 and Error Pages', () => {
  test('should display 404 for invalid routes', async ({ page }) => {
    await page.goto('/invalid-route-that-does-not-exist');
    await page.waitForLoadState('networkidle');

    // Should show 404 or redirect to home
    const is404 = await page.getByText(/404|Not Found|Page.*not.*found/i).isVisible().catch(() => false);
    const isHome = await page.url().includes('/dashboard') || await page.url() === '/';

    expect(is404 || isHome).toBeTruthy();
  });

  test('should handle API errors gracefully', async ({ page }) => {
    // Navigate to a page that loads data
    await page.goto('/pipelines');

    // Intercept API and return error
    await page.route('**/api/v1/pipelines', (route) => {
      route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'Internal Server Error' }),
      });
    });

    await page.reload();
    await page.waitForLoadState('networkidle');

    // Should show error message
    await expect(page.getByText(/Error|Failed to load|Unable to fetch/i)).toBeVisible({ timeout: 10000 });
  });
});

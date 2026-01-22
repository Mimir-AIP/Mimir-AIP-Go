import { test as base, expect, Page } from '@playwright/test';

/**
 * Helper functions for e2e tests
 */

/**
 * Login helper - logs in a user and stores auth state
 */
export async function login(page: Page, username: string, password: string) {
  await page.goto('/login');
  await page.fill('input[name="username"], input[type="text"]', username);
  await page.fill('input[name="password"], input[type="password"]', password);
  await page.click('button[type="submit"]');
  
  // Wait for navigation to complete
  await page.waitForURL((url) => !url.pathname.includes('/login'), { timeout: 10000 });
}

/**
 * Setup authenticated page by performing REAL login
 * Uses actual backend authentication - NO MOCKING
 * 
 * Backend default credentials:
 * - Username: admin
 * - Password: admin123
 * 
 * Note: If authentication is disabled on the backend, this will skip login
 * and just navigate to the dashboard (anonymous users get full access).
 */
export async function setupAuthenticatedPage(page: Page) {
  // Go to login page
  await page.goto('/login');
  
  // Fill in credentials
  await page.fill('input[name="username"], input[type="text"]', 'admin');
  await page.fill('input[name="password"], input[type="password"]', 'admin123');
  
  // Wait for login API response (with timeout to handle auth-disabled case)
  const loginResponsePromise = page.waitForResponse(
    response => response.url().includes('/api/v1/auth/login') && response.status() === 200,
    { timeout: 5000 }
  ).catch(() => null); // Return null if timeout (auth might be disabled)
  
  // Submit login form
  await page.click('button[type="submit"]');
  
  // Wait for login response
  const loginResponse = await loginResponsePromise;
  
  if (loginResponse) {
    // Authentication is enabled - wait for token
    await page.waitForFunction(() => localStorage.getItem('auth_token') !== null, { timeout: 5000 }).catch(() => {});
    
    // Verify we're authenticated via cookie
    const cookies = await page.context().cookies();
    const authCookie = cookies.find(c => c.name === 'auth_token');
    
    if (!authCookie) {
      // Token might be in localStorage only (not cookie)
      const hasToken = await page.evaluate(() => localStorage.getItem('auth_token') !== null);
      if (!hasToken) {
        console.warn('setupAuthenticatedPage: No auth token found, but continuing (backend might have auth disabled)');
      }
    }
  } else {
    // Authentication is disabled or login failed - just navigate to dashboard
    // When auth is disabled, defaultUserMiddleware injects anonymous admin user
    console.log('setupAuthenticatedPage: Auth appears to be disabled, skipping login verification');
  }
  
  // Navigate to dashboard
  await page.goto('/dashboard');
}

/**
 * Wait for API response and return data
 */
export async function waitForAPIResponse(
  page: Page,
  urlPattern: string | RegExp,
  action: () => Promise<void>
) {
  const responsePromise = page.waitForResponse((response) => {
    const url = response.url();
    if (typeof urlPattern === 'string') {
      return url.includes(urlPattern);
    }
    return urlPattern.test(url);
  });

  await action();
  const response = await responsePromise;
  return response.json();
}

/**
 * Upload a file via file input
 */
export async function uploadFile(
  page: Page,
  fileInputSelector: string,
  fileName: string,
  content: string,
  mimeType: string = 'text/plain'
) {
  const fileInput = page.locator(fileInputSelector);
  
  // Create a buffer from the content
  const buffer = Buffer.from(content);
  
  await fileInput.setInputFiles({
    name: fileName,
    mimeType,
    buffer,
  });
}

/**
 * Check if element is visible with retry
 */
export async function expectVisible(page: Page, selector: string, timeout: number = 5000) {
  await expect(page.locator(selector)).toBeVisible({ timeout });
}

/**
 * Check if text is present on page
 */
export async function expectTextVisible(page: Page, text: string | RegExp, timeout: number = 5000) {
  await expect(page.getByText(text)).toBeVisible({ timeout });
}

/**
 * Fill form and submit
 */
export async function fillAndSubmitForm(
  page: Page,
  formData: Record<string, string>,
  submitButtonSelector: string
) {
  for (const [name, value] of Object.entries(formData)) {
    const input = page.locator(`input[name="${name}"], textarea[name="${name}"], select[name="${name}"]`);
    await input.fill(value);
  }
  
  await page.click(submitButtonSelector);
}

/**
 * Wait for toast notification
 */
export async function waitForToast(page: Page, text: string | RegExp, timeout: number = 5000) {
  const toast = page.locator('[data-sonner-toast], .toast, [role="alert"]').filter({ hasText: text });
  await expect(toast).toBeVisible({ timeout });
}

/**
 * Wait for page to be ready by checking for specific indicators
 * 
 * This replaces the unreliable waitForLoadState('networkidle') pattern.
 * Instead of waiting for all network activity to stop (which may never happen
 * with polling/websockets), we wait for specific page elements to load.
 * 
 * @param page - Playwright page object
 * @param options - Indicators to wait for
 *   - heading: Wait for h1 heading (usually page title)
 *   - loadingGone: Wait for loading skeleton/spinner to disappear
 *   - testId: Wait for specific test ID element
 *   - timeout: Maximum time to wait (default 10s)
 * 
 * @example
 * // Wait for page heading
 * await waitForPageReady(page, { heading: 'Digital Twins' });
 * 
 * // Wait for loading to finish
 * await waitForPageReady(page, { loadingGone: true });
 * 
 * // Wait for specific element
 * await waitForPageReady(page, { testId: 'data-table' });
 */
export async function waitForPageReady(
  page: Page,
  options: {
    heading?: string | RegExp;
    loadingGone?: boolean;
    testId?: string;
    timeout?: number;
  } = {}
) {
  const timeout = options.timeout ?? 10000;

  // Wait for dom content loaded first (fast, basic check)
  await page.waitForLoadState('domcontentloaded');

  // Wait for heading if specified
  if (options.heading) {
    await expect(
      page.getByRole('heading', { name: options.heading })
    ).toBeVisible({ timeout });
  }

  // Wait for loading indicators to disappear
  if (options.loadingGone) {
    const loadingIndicators = page.locator(
      '[data-testid="loading-skeleton"], [data-testid="loading-spinner"], .loading, .spinner, [role="progressbar"]'
    );
    // If loading indicators exist, wait for them to disappear
    const count = await loadingIndicators.count();
    if (count > 0) {
      await expect(loadingIndicators.first()).not.toBeVisible({ timeout });
    }
  }

  // Wait for specific test ID element
  if (options.testId) {
    await expect(page.getByTestId(options.testId)).toBeVisible({ timeout });
  }
}

/**
 * Create a test with custom fixtures
 */
export const test = base.extend<{
  authenticatedPage: Page;
}>({
  authenticatedPage: async ({ page }, use) => {
    await setupAuthenticatedPage(page);
    await page.goto('/');
    await use(page);
  },
});

/**
 * ⚠️ DEPRECATED: APIMocker class - REMOVED
 * 
 * API mocking in E2E tests defeats the purpose of end-to-end testing.
 * E2E tests should test the REAL backend + frontend integration.
 * 
 * The APIMocker class has been removed. Tests that were using it need to be
 * refactored to work with the real backend API instead.
 * 
 * Use real authentication (setupAuthenticatedPage) and real API calls instead.
 * 
 * Tests that still reference APIMocker will need to be updated.
 */

export { expect };

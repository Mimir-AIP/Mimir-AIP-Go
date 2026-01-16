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
 * Setup authenticated page with API mocking
 */
export async function setupAuthenticatedPage(page: Page) {
  // Set auth cookie (required by middleware)
  await page.context().addCookies([{
    name: 'auth_token',
    value: 'test-token-' + Date.now(),
    domain: 'localhost',
    path: '/',
    expires: Date.now() / 1000 + 3600, // 1 hour from now
    httpOnly: false,
    secure: false,
    sameSite: 'Lax'
  }]);
  
  // Mock authentication check
  await page.route('**/api/v1/auth/check', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        authenticated: true,
        user: { username: 'testuser', role: 'admin' },
      }),
    });
  });
  
  // Set local storage auth token
  await page.addInitScript(() => {
    localStorage.setItem('auth_token', 'test-token');
  });
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
 * Mock API endpoints for testing
 */
export class APIMocker {
  constructor(private page: Page) {}

  async mockOntologyList(ontologies: any[]) {
    await this.page.route('**/api/v1/ontology', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ data: ontologies }),
      });
    });
  }

  async mockOntologyGet(id: string, ontology: any) {
    await this.page.route(`**/api/v1/ontology/${id}`, async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: { ontology } }),
      });
    });
  }

  async mockSPARQLQuery(results: any) {
    await this.page.route('**/api/v1/kg/query', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: results }),
      });
    });
  }

  async mockExtractionJobs(jobs: any[]) {
    await this.page.route('**/api/v1/extraction/jobs*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: { jobs } }),
      });
    });
  }

  async mockPipelines(pipelines: any[]) {
    await this.page.route('**/api/v1/pipelines*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ pipelines }),
      });
    });
  }

  async mockDigitalTwins(twins: any[]) {
    await this.page.route('**/api/v1/digital-twins*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: { digital_twins: twins } }),
      });
    });
  }
}

export { expect };

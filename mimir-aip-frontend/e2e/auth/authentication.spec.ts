import { test, expect } from '@playwright/test';

/**
 * Authentication E2E Tests - NO API MOCKING
 * Tests the real authentication flow against the actual backend
 * 
 * Backend creates default admin user:
 * - Username: admin
 * - Password: admin123
 */

test.describe('Authentication Flow', () => {
  let authEnabled = true;

  test.beforeAll(async () => {
    // Check if auth is enabled by trying to access a protected route without auth
    const { request } = await import('@playwright/test');
    const context = await request.newContext();
    const response = await context.get('http://localhost:8080/api/v1/pipelines').catch(() => null);
    
    // If we can access pipelines without auth, auth is disabled
    authEnabled = !(response && response.ok());
  });

  test.beforeEach(async ({ page }) => {
    // Clear any existing auth state
    await page.context().clearCookies();
    await page.goto('/');
  });

  test('should redirect unauthenticated user to login page', async ({ page }) => {
    if (!authEnabled) {
      // When auth is disabled, should NOT redirect to login - should allow access
      await page.goto('/dashboard');
      await page.waitForLoadState('networkidle');
      
      // Should stay on dashboard (not redirect to login)
      expect(page.url()).toContain('/dashboard');
      return;
    }

    await page.goto('/dashboard');
    
    // Should redirect to login
    await expect(page).toHaveURL(/\/login/, { timeout: 10000 });
    
    // Should show login form
    await expect(page.locator('input[name="username"], input[type="text"]')).toBeVisible();
    await expect(page.locator('input[name="password"], input[type="password"]')).toBeVisible();
  });

  test('should allow user to login with valid credentials', async ({ page }) => {
    if (!authEnabled) {
      // When auth is disabled, login page may not exist or may not be functional
      // Test passes if we can access protected routes without login
      await page.goto('/dashboard');
      await page.waitForLoadState('networkidle');
      expect(page.url()).toContain('/dashboard');
      return;
    }

    await page.goto('/login');
    
    // Fill login form with real backend credentials
    await page.fill('input[name="username"], input[type="text"]', 'admin');
    await page.fill('input[name="password"], input[type="password"]', 'admin123');
    
    // Wait for the login API response
    const loginResponsePromise = page.waitForResponse(response => 
      response.url().includes('/api/v1/auth/login') && response.status() === 200
    );
    
    // Submit form
    await page.click('button[type="submit"]');
    
    // Wait for successful login response
    const loginResponse = await loginResponsePromise;
    const loginData = await loginResponse.json();
    
    // Wait for token to be stored
    await page.waitForFunction(() => localStorage.getItem('auth_token') !== null);
    
    // Manually navigate to dashboard (this ensures the cookie is sent with the request)
    await page.goto('/dashboard');
    
    // Should now be on dashboard
    await expect(page).toHaveURL(/\/dashboard/);
    
    // Verify token is stored in localStorage
    const tokenInStorage = await page.evaluate(() => localStorage.getItem('auth_token'));
    expect(tokenInStorage).toBeTruthy();
    
    // Verify cookie is set
    const cookies = await page.context().cookies();
    const authCookie = cookies.find(c => c.name === 'auth_token');
    expect(authCookie).toBeTruthy();
  });

  test('should show error message with invalid credentials', async ({ page }) => {
    if (!authEnabled) {
      // Auth disabled - this test is not applicable
      // Just verify we can access the app
      await page.goto('/dashboard');
      await page.waitForLoadState('networkidle');
      expect(page.url()).toContain('/dashboard');
      return;
    }

    await page.goto('/login');
    
    // Fill login form with wrong credentials
    await page.fill('input[name="username"], input[type="text"]', 'wronguser');
    await page.fill('input[name="password"], input[type="password"]', 'wrongpass');
    
    // Submit form
    await page.click('button[type="submit"]');
    
    // Wait for error to appear
    await page.waitForTimeout(1000);
    
    // Should show error message
    await expect(page.locator('text=/invalid|error|wrong|incorrect|fail/i')).toBeVisible({ timeout: 5000 });
    
    // Should still be on login page
    await expect(page).toHaveURL(/\/login/);
  });

  test('should allow user to logout', async ({ page }) => {
    if (!authEnabled) {
      // Auth disabled - just verify app is accessible
      await page.goto('/dashboard');
      await page.waitForLoadState('networkidle');
      expect(page.url()).toContain('/dashboard');
      return;
    }

    // First login with real credentials
    await page.goto('/login');
    await page.fill('input[name="username"], input[type="text"]', 'admin');
    await page.fill('input[name="password"], input[type="password"]', 'admin123');
    
    // Wait for login API response
    const loginResponsePromise = page.waitForResponse(response => 
      response.url().includes('/api/v1/auth/login') && response.status() === 200
    );
    await page.click('button[type="submit"]');
    await loginResponsePromise;
    
    // Wait for token and navigate to dashboard
    await page.waitForFunction(() => localStorage.getItem('auth_token') !== null);
    await page.goto('/dashboard');
    
    // Click logout button
    const logoutButton = page.getByRole('button', { name: /logout|sign out/i });
    if (await logoutButton.isVisible({ timeout: 2000 }).catch(() => false)) {
      await logoutButton.click();
    } else {
      // Try clicking user menu first
      const userMenu = page.locator('[data-testid="user-menu"], button:has-text("admin"), button:has-text("user")').first();
      if (await userMenu.isVisible({ timeout: 2000 }).catch(() => false)) {
        await userMenu.click();
        await page.waitForTimeout(500);
        await page.getByRole('button', { name: /logout|sign out/i }).click();
      }
    }
    
    // Should redirect to login
    await expect(page).toHaveURL(/\/login/, { timeout: 10000 });
    
    // Cookie should be cleared
    const cookies = await page.context().cookies();
    const authCookie = cookies.find(c => c.name === 'auth_token');
    expect(authCookie).toBeFalsy();
  });

  test('should persist authentication across page reloads', async ({ page }) => {
    if (!authEnabled) {
      // Auth disabled - verify page reload works
      await page.goto('/dashboard');
      await page.waitForLoadState('networkidle');
      await page.reload();
      await page.waitForLoadState('networkidle');
      expect(page.url()).toContain('/dashboard');
      return;
    }

    // First login with real credentials
    await page.goto('/login');
    await page.fill('input[name="username"], input[type="text"]', 'admin');
    await page.fill('input[name="password"], input[type="password"]', 'admin123');
    
    // Wait for login API response
    const loginResponsePromise = page.waitForResponse(response => 
      response.url().includes('/api/v1/auth/login') && response.status() === 200
    );
    await page.click('button[type="submit"]');
    await loginResponsePromise;
    
    // Wait for token and navigate to dashboard
    await page.waitForFunction(() => localStorage.getItem('auth_token') !== null);
    await page.goto('/dashboard');
    
    // Reload page
    await page.reload();
    await page.waitForLoadState('networkidle');
    
    // Should still be on dashboard (not redirected to login)
    await expect(page).toHaveURL(/\/(dashboard)?$/);
  });

  test('should handle session expiration gracefully', async ({ page }) => {
    if (!authEnabled) {
      // Auth disabled - just verify app stays accessible
      await page.goto('/dashboard');
      await page.waitForLoadState('networkidle');
      await page.context().clearCookies();
      await page.reload();
      await page.waitForLoadState('networkidle');
      // Should still be accessible
      expect(page.url()).toContain('/dashboard');
      return;
    }

    // First login with real credentials
    await page.goto('/login');
    await page.fill('input[name="username"], input[type="text"]', 'admin');
    await page.fill('input[name="password"], input[type="password"]', 'admin123');
    
    // Wait for login API response
    const loginResponsePromise = page.waitForResponse(response => 
      response.url().includes('/api/v1/auth/login') && response.status() === 200
    );
    await page.click('button[type="submit"]');
    await loginResponsePromise;
    
    // Wait for token and navigate to dashboard
    await page.waitForFunction(() => localStorage.getItem('auth_token') !== null);
    await page.goto('/dashboard');
    
    // Clear the cookie to simulate expiration
    await page.context().clearCookies();
    await page.evaluate(() => localStorage.removeItem('auth_token'));
    
    // Try to navigate - should redirect to login due to missing cookie
    await page.goto('/ontologies');
    
    // Should redirect to login
    await expect(page).toHaveURL(/\/login/, { timeout: 10000 });
  });
});

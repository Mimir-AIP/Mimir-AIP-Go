import { test, expect } from '@playwright/test';
import { testUsers } from '../fixtures/test-data';
import { login, expectVisible, expectTextVisible } from '../helpers';

test.describe('Authentication Flow', () => {
  test.beforeEach(async ({ page }) => {
    // Clear any existing auth state
    await page.context().clearCookies();
    await page.goto('/');
  });

  test('should redirect unauthenticated user to login page', async ({ page }) => {
    await page.goto('/dashboard');
    
    // Should redirect to login
    await expect(page).toHaveURL(/\/login/, { timeout: 10000 });
    await expectVisible(page, 'input[name="username"], input[type="text"]');
    await expectVisible(page, 'input[name="password"], input[type="password"]');
  });

  test('should allow user to login with valid credentials', async ({ page }) => {
    let loginAttempted = false;
    
    // Mock successful login
    await page.route('**/api/v1/auth/login', async (route) => {
      loginAttempted = true;
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          token: 'test-token-123',
          user: { username: testUsers.admin.username, role: 'admin' },
          expires_in: 86400,
        }),
      });
    });

    await page.goto('/login');
    
    // Fill login form
    await page.fill('input[name="username"], input[type="text"]', testUsers.admin.username);
    await page.fill('input[name="password"], input[type="password"]', testUsers.admin.password);
    
    // Submit form
    await page.click('button[type="submit"]');
    
    // Wait for login to be attempted
    await page.waitForTimeout(1000);
    
    // Verify login was attempted
    expect(loginAttempted).toBe(true);
    
    // Verify token is stored in localStorage
    const tokenInStorage = await page.evaluate(() => localStorage.getItem('auth_token'));
    expect(tokenInStorage).toBe('test-token-123');
    
    // Note: Full redirect test would require real backend or more complex cookie mocking
    // The middleware checks cookies which are set by JS, causing timing issues in tests
  });

  test('should show error message with invalid credentials', async ({ page }) => {
    // Mock failed login
    await page.route('**/api/v1/auth/login', async (route) => {
      await route.fulfill({
        status: 401,
        contentType: 'application/json',
        body: JSON.stringify({
          error: 'Invalid credentials',
        }),
      });
    });

    await page.goto('/login');
    
    // Fill login form with wrong credentials
    await page.fill('input[name="username"], input[type="text"]', 'wronguser');
    await page.fill('input[name="password"], input[type="password"]', 'wrongpass');
    
    // Submit form
    await page.click('button[type="submit"]');
    
    // Should show error message
    await expectTextVisible(page, /invalid|error|wrong|fail/i);
    
    // Should still be on login page
    await expect(page).toHaveURL(/\/login/);
  });

  test('should allow user to logout', async ({ page }) => {
    // Mock auth check
    await page.route('**/api/v1/auth/check', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          authenticated: true,
          user: { username: testUsers.admin.username },
        }),
      });
    });

    // Mock logout
    await page.route('**/api/v1/auth/logout', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true }),
      });
    });

    // Set auth token (both cookie and localStorage)
    await page.context().addCookies([{
      name: 'auth_token',
      value: 'test-token-' + Date.now(),
      domain: 'localhost',
      path: '/',
      expires: Date.now() / 1000 + 3600,
      httpOnly: false,
      secure: false,
      sameSite: 'Lax'
    }]);
    
    await page.addInitScript(() => {
      localStorage.setItem('auth_token', 'test-token');
    });

    await page.goto('/dashboard');
    
    // Click logout button (could be in nav or dropdown)
    const logoutButton = page.getByRole('button', { name: /logout|sign out/i });
    if (await logoutButton.isVisible({ timeout: 2000 }).catch(() => false)) {
      await logoutButton.click();
    } else {
      // Try clicking user menu first
      const userMenu = page.locator('[data-testid="user-menu"], button:has-text("admin"), button:has-text("user")').first();
      if (await userMenu.isVisible({ timeout: 2000 }).catch(() => false)) {
        await userMenu.click();
        await page.getByRole('button', { name: /logout|sign out/i }).click();
      }
    }
    
    // Should redirect to login
    await expect(page).toHaveURL(/\/login/, { timeout: 10000 });
  });

  test('should persist authentication across page reloads', async ({ page }) => {
    // Mock auth check
    await page.route('**/api/v1/auth/check', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          authenticated: true,
          user: { username: testUsers.admin.username },
        }),
      });
    });

    // Set auth token (both cookie and localStorage)
    await page.context().addCookies([{
      name: 'auth_token',
      value: 'test-token-' + Date.now(),
      domain: 'localhost',
      path: '/',
      expires: Date.now() / 1000 + 3600,
      httpOnly: false,
      secure: false,
      sameSite: 'Lax'
    }]);
    
    await page.addInitScript(() => {
      localStorage.setItem('auth_token', 'test-token');
    });

    await page.goto('/dashboard');
    
    // Should be on dashboard
    await expect(page).toHaveURL(/\/(dashboard)?$/);
    
    // Reload page
    await page.reload();
    
    // Should still be on dashboard (not redirected to login)
    await expect(page).toHaveURL(/\/(dashboard)?$/);
  });

  test('should handle session expiration gracefully', async ({ page }) => {
    // Set initial auth
    await page.context().addCookies([{
      name: 'auth_token',
      value: 'test-token-' + Date.now(),
      domain: 'localhost',
      path: '/',
      expires: Date.now() / 1000 + 3600,
      httpOnly: false,
      secure: false,
      sameSite: 'Lax'
    }]);
    
    await page.addInitScript(() => {
      localStorage.setItem('auth_token', 'test-token');
    });

    // Mock auth check - first succeeds
    await page.route('**/api/v1/auth/check', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          authenticated: true,
          user: { username: testUsers.admin.username },
        }),
      });
    });

    await page.goto('/dashboard');
    
    // Should be on dashboard
    await expect(page).toHaveURL(/\/(dashboard)?$/);
    
    // Now clear the cookie to simulate expiration
    await page.context().clearCookies();
    
    // Try to navigate - should redirect to login due to missing cookie
    await page.goto('/ontologies');
    
    // Should redirect to login
    await expect(page).toHaveURL(/\/login/, { timeout: 10000 });
  });
});

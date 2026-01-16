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
    // Mock successful login
    await page.route('**/api/v1/auth/login', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          token: 'test-token-123',
          user: { username: testUsers.admin.username, role: 'admin' },
        }),
      });
    });

    await page.goto('/login');
    
    // Fill login form
    await page.fill('input[name="username"], input[type="text"]', testUsers.admin.username);
    await page.fill('input[name="password"], input[type="password"]', testUsers.admin.password);
    
    // Submit form
    await page.click('button[type="submit"]');
    
    // Should redirect to dashboard
    await expect(page).toHaveURL(/\/(dashboard)?$/, { timeout: 10000 });
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
    let authCheckCount = 0;
    
    // Mock auth check - first call succeeds, second fails
    await page.route('**/api/v1/auth/check', async (route) => {
      authCheckCount++;
      if (authCheckCount === 1) {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            authenticated: true,
            user: { username: testUsers.admin.username },
          }),
        });
      } else {
        await route.fulfill({
          status: 401,
          contentType: 'application/json',
          body: JSON.stringify({ error: 'Session expired' }),
        });
      }
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
    
    // Trigger another auth check (e.g., by navigating)
    await page.goto('/ontologies');
    
    // Should redirect to login due to expired session
    await expect(page).toHaveURL(/\/login/, { timeout: 10000 });
  });
});

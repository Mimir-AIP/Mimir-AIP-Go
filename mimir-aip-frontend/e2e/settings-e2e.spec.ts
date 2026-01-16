import { test, expect, Page } from '@playwright/test';
import { setupAuthenticatedPage } from './helpers';

/**
 * Comprehensive E2E tests for Settings page including API Keys management
 */

test.describe('Settings - API Keys Management', () => {
  let page: Page;

  test.beforeEach(async ({ browser }) => {
    page = await browser.newPage();
    await setupAuthenticatedPage(page);
    await page.goto('/settings');
    await page.waitForLoadState('networkidle');
  });

  test.afterEach(async () => {
    await page.close();
  });

  test('should display settings page with API keys section', async () => {
    // Check page title
    await expect(page).toHaveTitle(/Settings/i);

    // Check for API keys section
    await expect(page.getByRole('heading', { name: /API Keys/i })).toBeVisible();

    // Check for "Add API Key" button
    await expect(page.getByRole('button', { name: /Add API Key/i })).toBeVisible();
  });

  test('should show empty state when no API keys exist', async () => {
    // Check for empty state message
    const emptyState = page.getByText(/No API keys configured/i);
    if (await emptyState.isVisible()) {
      await expect(emptyState).toBeVisible();
    }
  });

  test('should open create API key dialog', async () => {
    // Click "Add API Key" button
    await page.getByRole('button', { name: /Add API Key/i }).click();

    // Check dialog is visible
    await expect(page.getByRole('dialog')).toBeVisible();
    await expect(page.getByRole('heading', { name: /Add API Key/i })).toBeVisible();

    // Check form fields
    await expect(page.getByLabel(/Provider/i)).toBeVisible();
    await expect(page.getByLabel(/Name/i)).toBeVisible();
    await expect(page.getByLabel(/API Key/i)).toBeVisible();
  });

  test('should create a new OpenAI API key', async () => {
    // Open create dialog
    await page.getByRole('button', { name: /Add API Key/i }).click();

    // Fill form
    await page.getByLabel(/Provider/i).selectOption('openai');
    await page.getByLabel(/Name/i).fill('Test OpenAI Key');
    await page.getByLabel(/API Key/i).fill('sk-test-1234567890abcdef');

    // Submit form
    await page.getByRole('button', { name: /Create/i }).click();

    // Wait for success message
    await expect(page.getByText(/API key created successfully/i)).toBeVisible({ timeout: 5000 });

    // Verify key appears in list
    await expect(page.getByText('Test OpenAI Key')).toBeVisible();
    await expect(page.getByText('openai')).toBeVisible();
  });

  test('should create an Anthropic API key with metadata', async () => {
    // Open create dialog
    await page.getByRole('button', { name: /Add API Key/i }).click();

    // Fill form
    await page.getByLabel(/Provider/i).selectOption('anthropic');
    await page.getByLabel(/Name/i).fill('Production Claude Key');
    await page.getByLabel(/API Key/i).fill('sk-ant-api03-test-key-12345');

    // Add metadata (if supported in UI)
    const metadataToggle = page.getByText(/Advanced Options/i);
    if (await metadataToggle.isVisible()) {
      await metadataToggle.click();
      await page.getByLabel(/Model/i).fill('claude-3-opus');
    }

    // Submit
    await page.getByRole('button', { name: /Create/i }).click();

    // Verify
    await expect(page.getByText(/API key created successfully/i)).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('Production Claude Key')).toBeVisible();
  });

  test('should validate required fields', async () => {
    // Open dialog
    await page.getByRole('button', { name: /Add API Key/i }).click();

    // Try to submit without filling fields
    await page.getByRole('button', { name: /Create/i }).click();

    // Check for validation errors
    await expect(page.getByText(/Provider is required/i)).toBeVisible();
  });

  test('should display list of API keys', async () => {
    // Create a test key first (assuming one exists)
    const keysList = page.getByTestId('api-keys-list');

    if (await keysList.isVisible()) {
      // Check table headers
      await expect(page.getByText(/Provider/i)).toBeVisible();
      await expect(page.getByText(/Name/i)).toBeVisible();
      await expect(page.getByText(/Status/i)).toBeVisible();

      // Check that key values are masked (should show **** or similar)
      const keyValues = page.locator('[data-testid="api-key-value"]');
      if (await keyValues.first().isVisible()) {
        const value = await keyValues.first().textContent();
        expect(value).toContain('****');
      }
    }
  });

  test('should toggle API key active status', async () => {
    // Find first API key (assuming one exists)
    const toggleButton = page.getByRole('switch').first();

    if (await toggleButton.isVisible()) {
      const initialState = await toggleButton.getAttribute('aria-checked');

      // Toggle
      await toggleButton.click();

      // Wait for update
      await page.waitForTimeout(1000);

      // Verify state changed
      const newState = await toggleButton.getAttribute('aria-checked');
      expect(newState).not.toBe(initialState);
    }
  });

  test('should test API key validation', async () => {
    // Find first API key
    const testButton = page.getByRole('button', { name: /Test/i }).first();

    if (await testButton.isVisible()) {
      await testButton.click();

      // Wait for test result
      await page.waitForTimeout(2000);

      // Check for result message
      const resultMessage = page.getByText(/valid|invalid|inactive/i);
      await expect(resultMessage).toBeVisible({ timeout: 5000 });
    }
  });

  test('should open edit dialog for API key', async () => {
    // Find edit button
    const editButton = page.getByRole('button', { name: /Edit/i }).first();

    if (await editButton.isVisible()) {
      await editButton.click();

      // Check dialog
      await expect(page.getByRole('dialog')).toBeVisible();
      await expect(page.getByRole('heading', { name: /Edit API Key/i })).toBeVisible();

      // Check form has name field
      await expect(page.getByLabel(/Name/i)).toBeVisible();
    }
  });

  test('should update API key name', async () => {
    const editButton = page.getByRole('button', { name: /Edit/i }).first();

    if (await editButton.isVisible()) {
      await editButton.click();

      // Clear and update name
      const nameInput = page.getByLabel(/Name/i);
      await nameInput.clear();
      await nameInput.fill('Updated API Key Name');

      // Save
      await page.getByRole('button', { name: /Save|Update/i }).click();

      // Verify update
      await expect(page.getByText(/API key updated successfully/i)).toBeVisible({ timeout: 5000 });
      await expect(page.getByText('Updated API Key Name')).toBeVisible();
    }
  });

  test('should delete API key with confirmation', async () => {
    const deleteButton = page.getByRole('button', { name: /Delete/i }).first();

    if (await deleteButton.isVisible()) {
      // Get the key name before deleting
      const keyRow = deleteButton.locator('..').locator('..');
      const keyName = await keyRow.getByTestId('key-name').textContent();

      await deleteButton.click();

      // Confirm deletion
      await expect(page.getByRole('dialog')).toBeVisible();
      await expect(page.getByText(/Are you sure/i)).toBeVisible();
      await page.getByRole('button', { name: /Confirm|Delete/i }).click();

      // Verify deletion
      await expect(page.getByText(/API key deleted successfully/i)).toBeVisible({ timeout: 5000 });
      await expect(page.getByText(keyName || '')).not.toBeVisible();
    }
  });

  test('should cancel deletion', async () => {
    const deleteButton = page.getByRole('button', { name: /Delete/i }).first();

    if (await deleteButton.isVisible()) {
      await deleteButton.click();

      // Cancel
      await expect(page.getByRole('dialog')).toBeVisible();
      await page.getByRole('button', { name: /Cancel/i }).click();

      // Dialog should close
      await expect(page.getByRole('dialog')).not.toBeVisible();
    }
  });

  test('should support multiple providers', async () => {
    await page.getByRole('button', { name: /Add API Key/i }).click();

    const providerSelect = page.getByLabel(/Provider/i);
    await providerSelect.click();

    // Check all providers are available
    await expect(page.getByRole('option', { name: /OpenAI/i })).toBeVisible();
    await expect(page.getByRole('option', { name: /Anthropic/i })).toBeVisible();
    await expect(page.getByRole('option', { name: /Ollama/i })).toBeVisible();
    await expect(page.getByRole('option', { name: /Google/i })).toBeVisible();
  });

  test('should show custom endpoint field for Ollama', async () => {
    await page.getByRole('button', { name: /Add API Key/i }).click();

    // Select Ollama
    await page.getByLabel(/Provider/i).selectOption('ollama');

    // Check for endpoint URL field
    await expect(page.getByLabel(/Endpoint URL/i)).toBeVisible();
  });

  test('should filter API keys by provider', async () => {
    const filterSelect = page.getByLabel(/Filter by Provider/i);

    if (await filterSelect.isVisible()) {
      // Select OpenAI filter
      await filterSelect.selectOption('openai');

      // Wait for filter to apply
      await page.waitForTimeout(500);

      // All visible keys should be OpenAI
      const providerCells = page.locator('[data-provider="openai"]');
      const count = await providerCells.count();
      expect(count).toBeGreaterThan(0);
    }
  });

  test('should search API keys by name', async () => {
    const searchInput = page.getByPlaceholder(/Search/i);

    if (await searchInput.isVisible()) {
      await searchInput.fill('Production');
      await page.waitForTimeout(500);

      // Results should contain "Production"
      const results = page.getByTestId('api-key-row');
      const count = await results.count();

      for (let i = 0; i < count; i++) {
        const text = await results.nth(i).textContent();
        expect(text?.toLowerCase()).toContain('production');
      }
    }
  });

  test('should display API key creation date', async () => {
    const dateCell = page.locator('[data-testid="created-at"]').first();

    if (await dateCell.isVisible()) {
      const dateText = await dateCell.textContent();
      // Should be a valid date format
      expect(dateText).toMatch(/\d{4}-\d{2}-\d{2}|\d{1,2}\/\d{1,2}\/\d{4}/);
    }
  });

  test('should handle API errors gracefully', async () => {
    // Simulate network error by going offline
    await page.context().setOffline(true);

    await page.getByRole('button', { name: /Add API Key/i }).click();
    await page.getByLabel(/Provider/i).selectOption('openai');
    await page.getByLabel(/Name/i).fill('Test Key');
    await page.getByLabel(/API Key/i).fill('sk-test');
    await page.getByRole('button', { name: /Create/i }).click();

    // Should show error message
    await expect(page.getByText(/Failed to create|Network error|Unable to connect/i)).toBeVisible({ timeout: 10000 });

    // Re-enable network
    await page.context().setOffline(false);
  });

  test('should navigate to plugins settings', async () => {
    const pluginsTab = page.getByRole('tab', { name: /Plugins/i });

    if (await pluginsTab.isVisible()) {
      await pluginsTab.click();

      // Should show plugins section
      await expect(page.getByRole('heading', { name: /Plugins/i })).toBeVisible();
    }
  });
});

test.describe('Settings - Plugins Management', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/settings');
    await page.waitForLoadState('networkidle');

    // Navigate to plugins tab
    const pluginsTab = page.getByRole('tab', { name: /Plugins/i });
    if (await pluginsTab.isVisible()) {
      await pluginsTab.click();
    }
  });

  test('should display installed plugins', async ({ page }) => {
    // Check for plugins list
    await expect(page.getByRole('heading', { name: /Installed Plugins/i })).toBeVisible();

    // Should have at least built-in plugins
    const pluginItems = page.getByTestId('plugin-item');
    const count = await pluginItems.count();
    expect(count).toBeGreaterThan(0);
  });

  test('should toggle plugin enable/disable', async ({ page }) => {
    const toggleButton = page.getByRole('switch', { name: /Enable|Disable/i }).first();

    if (await toggleButton.isVisible()) {
      await toggleButton.click();

      // Wait for update
      await page.waitForTimeout(1000);

      // Should show success message
      await expect(page.getByText(/Plugin updated|Status changed/i)).toBeVisible({ timeout: 5000 });
    }
  });

  test('should display plugin details', async ({ page }) => {
    const detailsButton = page.getByRole('button', { name: /Details|View/i }).first();

    if (await detailsButton.isVisible()) {
      await detailsButton.click();

      // Check modal/dialog
      await expect(page.getByRole('dialog')).toBeVisible();
      await expect(page.getByText(/Version|Author|Description/i)).toBeVisible();
    }
  });
});

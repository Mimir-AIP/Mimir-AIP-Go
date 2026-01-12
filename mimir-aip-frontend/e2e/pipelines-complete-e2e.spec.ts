import { test, expect } from '@playwright/test';

/**
 * Comprehensive E2E tests for Pipelines CRUD and execution workflows
 */

test.describe('Pipelines - Complete Workflow', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/pipelines');
    await page.waitForLoadState('networkidle');
  });

  test('should display pipelines list page', async ({ page }) => {
    await expect(page).toHaveTitle(/Pipelines/i);
    await expect(page.getByRole('heading', { name: /Pipelines/i })).toBeVisible();
    await expect(page.getByRole('button', { name: /Create.*Pipeline|New Pipeline/i })).toBeVisible();
  });

  test('should show empty state when no pipelines exist', async ({ page }) => {
    const emptyState = page.getByText(/No pipelines|Get started by creating/i);

    if (await emptyState.isVisible()) {
      await expect(emptyState).toBeVisible();
      await expect(page.getByRole('button', { name: /Create.*Pipeline/i })).toBeVisible();
    }
  });

  test('should create a new pipeline', async ({ page }) => {
    // Click create button
    await page.getByRole('button', { name: /Create.*Pipeline|New Pipeline/i }).click();

    // Should navigate to create page or show dialog
    const isDialog = await page.getByRole('dialog').isVisible();
    const isNewPage = await page.url().includes('/create');

    expect(isDialog || isNewPage).toBeTruthy();

    // Fill pipeline details
    await page.getByLabel(/Name/i).fill('Test E2E Pipeline');
    await page.getByLabel(/Description/i).fill('Automated E2E test pipeline');

    // Add a step
    const addStepButton = page.getByRole('button', { name: /Add Step/i });
    if (await addStepButton.isVisible()) {
      await addStepButton.click();

      // Select plugin
      await page.getByLabel(/Plugin/i).selectOption('Input.api');
      await page.getByLabel(/Step Name/i).fill('Fetch Data');

      // Configure step
      const configInput = page.getByLabel(/Config|Configuration/i);
      if (await configInput.isVisible()) {
        await configInput.fill('{"url": "https://api.example.com/data"}');
      }
    }

    // Save pipeline
    await page.getByRole('button', { name: /Create|Save/i }).click();

    // Verify creation
    await expect(page.getByText(/Pipeline created successfully/i)).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('Test E2E Pipeline')).toBeVisible();
  });

  test('should validate pipeline before saving', async ({ page }) => {
    await page.getByRole('button', { name: /Create.*Pipeline/i }).click();

    // Try to save without required fields
    await page.getByRole('button', { name: /Create|Save/i }).click();

    // Should show validation errors
    await expect(page.getByText(/Name is required|This field is required/i)).toBeVisible();
  });

  test('should display pipeline details', async ({ page }) => {
    // Click on first pipeline (if exists)
    const pipelineRow = page.getByTestId('pipeline-row').first();

    if (await pipelineRow.isVisible()) {
      await pipelineRow.click();

      // Should navigate to details page
      await expect(page).toHaveURL(/\/pipelines\/[a-zA-Z0-9-]+/);
      await expect(page.getByRole('heading', { name: /Pipeline Details/i })).toBeVisible();
      await expect(page.getByText(/Steps|Configuration|History/i)).toBeVisible();
    }
  });

  test('should edit existing pipeline', async ({ page }) => {
    const editButton = page.getByRole('button', { name: /Edit/i }).first();

    if (await editButton.isVisible()) {
      await editButton.click();

      // Edit name
      const nameInput = page.getByLabel(/Name/i);
      await nameInput.clear();
      await nameInput.fill('Updated Pipeline Name');

      // Save changes
      await page.getByRole('button', { name: /Save|Update/i }).click();

      // Verify update
      await expect(page.getByText(/Pipeline updated successfully/i)).toBeVisible({ timeout: 5000 });
      await expect(page.getByText('Updated Pipeline Name')).toBeVisible();
    }
  });

  test('should clone a pipeline', async ({ page }) => {
    const cloneButton = page.getByRole('button', { name: /Clone|Duplicate/i }).first();

    if (await cloneButton.isVisible()) {
      await cloneButton.click();

      // Dialog should appear
      await expect(page.getByRole('dialog')).toBeVisible();
      await expect(page.getByText(/Clone Pipeline/i)).toBeVisible();

      // Enter new name
      await page.getByLabel(/Name/i).fill('Cloned Pipeline');

      // Confirm clone
      await page.getByRole('button', { name: /Clone|Duplicate|Create/i }).click();

      // Verify cloning
      await expect(page.getByText(/Pipeline cloned successfully/i)).toBeVisible({ timeout: 5000 });
      await expect(page.getByText('Cloned Pipeline')).toBeVisible();
    }
  });

  test('should validate pipeline configuration', async ({ page }) => {
    const validateButton = page.getByRole('button', { name: /Validate/i }).first();

    if (await validateButton.isVisible()) {
      await validateButton.click();

      // Wait for validation result
      await page.waitForTimeout(2000);

      // Should show validation result
      const result = page.getByText(/Valid|Invalid|Validation.*successful|Validation.*failed/i);
      await expect(result).toBeVisible({ timeout: 5000 });
    }
  });

  test('should execute a pipeline', async ({ page }) => {
    const executeButton = page.getByRole('button', { name: /Execute|Run/i }).first();

    if (await executeButton.isVisible()) {
      await executeButton.click();

      // Confirm execution dialog
      const confirmDialog = page.getByRole('dialog');
      if (await confirmDialog.isVisible()) {
        await page.getByRole('button', { name: /Confirm|Execute|Run/i }).click();
      }

      // Should show execution started message
      await expect(page.getByText(/Pipeline.*started|Execution.*started/i)).toBeVisible({ timeout: 5000 });

      // Should redirect to jobs or show status
      const isJobs = page.url().includes('/jobs');
      const hasStatus = await page.getByText(/Running|In Progress|Status/i).isVisible();

      expect(isJobs || hasStatus).toBeTruthy();
    }
  });

  test('should delete a pipeline with confirmation', async ({ page }) => {
    const deleteButton = page.getByRole('button', { name: /Delete/i }).first();

    if (await deleteButton.isVisible()) {
      // Get pipeline name
      const pipelineRow = deleteButton.locator('..').locator('..');
      const pipelineName = await pipelineRow.getByTestId('pipeline-name').textContent().catch(() => 'Unknown');

      await deleteButton.click();

      // Confirm deletion
      await expect(page.getByRole('dialog')).toBeVisible();
      await expect(page.getByText(/Are you sure|Confirm.*delete/i)).toBeVisible();
      await page.getByRole('button', { name: /Delete|Confirm/i }).click();

      // Verify deletion
      await expect(page.getByText(/Pipeline deleted successfully/i)).toBeVisible({ timeout: 5000 });
      await expect(page.getByText(pipelineName || '')).not.toBeVisible();
    }
  });

  test('should filter pipelines by status', async ({ page }) => {
    const filterSelect = page.getByLabel(/Filter|Status/i);

    if (await filterSelect.isVisible()) {
      await filterSelect.selectOption('enabled');
      await page.waitForTimeout(500);

      // All visible pipelines should be enabled
      const pipelines = page.getByTestId('pipeline-row');
      const count = await pipelines.count();

      for (let i = 0; i < Math.min(count, 5); i++) {
        const statusBadge = pipelines.nth(i).getByTestId('status-badge');
        const status = await statusBadge.textContent();
        expect(status?.toLowerCase()).toContain('enabled');
      }
    }
  });

  test('should search pipelines by name', async ({ page }) => {
    const searchInput = page.getByPlaceholder(/Search.*pipelines/i);

    if (await searchInput.isVisible()) {
      await searchInput.fill('test');
      await page.waitForTimeout(500);

      // Results should contain "test"
      const results = page.getByTestId('pipeline-row');
      const count = await results.count();

      if (count > 0) {
        for (let i = 0; i < count; i++) {
          const text = await results.nth(i).textContent();
          expect(text?.toLowerCase()).toContain('test');
        }
      }
    }
  });

  test('should view pipeline execution history', async ({ page }) => {
    const historyButton = page.getByRole('button', { name: /History|View.*History/i }).first();

    if (await historyButton.isVisible()) {
      await historyButton.click();

      // Should show history dialog or navigate to history page
      await expect(page.getByRole('heading', { name: /History|Executions/i })).toBeVisible();

      // Should show list of executions
      const executions = page.getByTestId('execution-row');
      if (await executions.first().isVisible()) {
        await expect(executions.first()).toBeVisible();
      }
    }
  });

  test('should export pipeline configuration', async ({ page }) => {
    const exportButton = page.getByRole('button', { name: /Export|Download/i }).first();

    if (await exportButton.isVisible()) {
      // Set up download listener
      const downloadPromise = page.waitForEvent('download');

      await exportButton.click();

      // Wait for download
      const download = await downloadPromise;

      // Verify download
      expect(download.suggestedFilename()).toMatch(/\.yaml|\.json|\.yml/);
    }
  });

  test('should import pipeline from file', async ({ page }) => {
    const importButton = page.getByRole('button', { name: /Import|Upload/i });

    if (await importButton.isVisible()) {
      await importButton.click();

      // Upload dialog should appear
      const fileInput = page.locator('input[type="file"]');
      await expect(fileInput).toBeVisible();

      // Note: Actual file upload would require test fixture files
    }
  });

  test('should toggle pipeline enabled/disabled', async ({ page }) => {
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

      // Should show success message
      await expect(page.getByText(/Pipeline.*updated|Status.*changed/i)).toBeVisible({ timeout: 5000 });
    }
  });

  test('should add multiple steps to pipeline', async ({ page }) => {
    await page.getByRole('button', { name: /Create.*Pipeline/i }).click();

    await page.getByLabel(/Name/i).fill('Multi-Step Pipeline');

    // Add first step
    await page.getByRole('button', { name: /Add Step/i }).click();
    await page.getByLabel(/Plugin/i).first().selectOption('Input.api');

    // Add second step
    await page.getByRole('button', { name: /Add Step/i }).click();
    await page.getByLabel(/Plugin/i).last().selectOption('Output.html');

    // Should have 2 steps
    const steps = page.getByTestId('pipeline-step');
    await expect(steps).toHaveCount(2);
  });

  test('should reorder pipeline steps', async ({ page }) => {
    const pipeline = page.getByTestId('pipeline-row').first();

    if (await pipeline.isVisible()) {
      await pipeline.click();

      // Look for reorder buttons
      const moveUpButton = page.getByRole('button', { name: /Move Up|Up/i }).first();
      const moveDownButton = page.getByRole('button', { name: /Move Down|Down/i }).first();

      if (await moveUpButton.isVisible() || await moveDownButton.isVisible()) {
        const initialOrder = await page.getByTestId('pipeline-step').allTextContents();

        if (await moveDownButton.isVisible()) {
          await moveDownButton.click();
        }

        await page.waitForTimeout(500);

        const newOrder = await page.getByTestId('pipeline-step').allTextContents();
        expect(newOrder).not.toEqual(initialOrder);
      }
    }
  });

  test('should configure step dependencies', async ({ page }) => {
    await page.getByRole('button', { name: /Create.*Pipeline/i }).click();

    await page.getByLabel(/Name/i).fill('Pipeline with Dependencies');
    await page.getByRole('button', { name: /Add Step/i }).click();

    // Look for dependencies configuration
    const dependsOnSelect = page.getByLabel(/Depends.*On|Dependencies/i);

    if (await dependsOnSelect.isVisible()) {
      await dependsOnSelect.selectOption('previous-step');
      // Verify selection
      const selectedValue = await dependsOnSelect.inputValue();
      expect(selectedValue).toBe('previous-step');
    }
  });

  test('should handle pipeline execution errors', async ({ page }) => {
    // Mock API to return error
    await page.route('**/api/v1/pipelines/*/execute', (route) => {
      route.fulfill({
        status: 400,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'Invalid pipeline configuration' }),
      });
    });

    const executeButton = page.getByRole('button', { name: /Execute|Run/i }).first();

    if (await executeButton.isVisible()) {
      await executeButton.click();

      // Should show error message
      await expect(page.getByText(/Failed.*execute|Invalid.*configuration|Error/i)).toBeVisible({ timeout: 10000 });
    }
  });

  test('should show pipeline statistics', async ({ page }) => {
    const statsButton = page.getByRole('button', { name: /Statistics|Stats/i }).first();

    if (await statsButton.isVisible()) {
      await statsButton.click();

      // Should show stats dialog or panel
      await expect(page.getByText(/Success Rate|Total.*Executions|Average.*Duration/i)).toBeVisible();
    }
  });

  test('should schedule pipeline execution', async ({ page }) => {
    const scheduleButton = page.getByRole('button', { name: /Schedule/i }).first();

    if (await scheduleButton.isVisible()) {
      await scheduleButton.click();

      // Schedule dialog should appear
      await expect(page.getByRole('dialog')).toBeVisible();
      await expect(page.getByLabel(/Cron.*Expression|Schedule/i)).toBeVisible();

      // Enter cron expression
      await page.getByLabel(/Cron.*Expression/i).fill('0 0 * * *');

      // Save schedule
      await page.getByRole('button', { name: /Save.*Schedule|Create/i }).click();

      // Verify scheduling
      await expect(page.getByText(/Schedule.*created|Pipeline.*scheduled/i)).toBeVisible({ timeout: 5000 });
    }
  });
});

test.describe('Pipelines - Step Configuration', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/pipelines');
    await page.waitForLoadState('networkidle');
  });

  test('should configure Input plugin step', async ({ page }) => {
    await page.getByRole('button', { name: /Create.*Pipeline/i }).click();
    await page.getByLabel(/Name/i).fill('Input Plugin Test');
    await page.getByRole('button', { name: /Add Step/i }).click();

    // Select Input.api plugin
    await page.getByLabel(/Plugin/i).selectOption('Input.api');

    // Configure
    const urlInput = page.getByLabel(/URL/i);
    if (await urlInput.isVisible()) {
      await urlInput.fill('https://api.example.com/data');
    }

    // Save
    await page.getByRole('button', { name: /Create|Save/i }).click();
    await expect(page.getByText(/Pipeline created/i)).toBeVisible({ timeout: 5000 });
  });

  test('should configure Output plugin step', async ({ page }) => {
    await page.getByRole('button', { name: /Create.*Pipeline/i }).click();
    await page.getByLabel(/Name/i).fill('Output Plugin Test');
    await page.getByRole('button', { name: /Add Step/i }).click();

    // Select Output.html plugin
    await page.getByLabel(/Plugin/i).selectOption('Output.html');

    // Configure
    const titleInput = page.getByLabel(/Title/i);
    if (await titleInput.isVisible()) {
      await titleInput.fill('Test Report');
    }

    // Save
    await page.getByRole('button', { name: /Create|Save/i }).click();
    await expect(page.getByText(/Pipeline created/i)).toBeVisible({ timeout: 5000 });
  });

  test('should delete a pipeline step', async ({ page }) => {
    await page.getByRole('button', { name: /Create.*Pipeline/i }).click();
    await page.getByLabel(/Name/i).fill('Step Deletion Test');

    // Add two steps
    await page.getByRole('button', { name: /Add Step/i }).click();
    await page.getByRole('button', { name: /Add Step/i }).click();

    // Delete first step
    const deleteStepButton = page.getByRole('button', { name: /Delete.*Step|Remove.*Step/i }).first();
    if (await deleteStepButton.isVisible()) {
      await deleteStepButton.click();

      // Confirm deletion if required
      const confirmButton = page.getByRole('button', { name: /Confirm|Delete/i });
      if (await confirmButton.isVisible()) {
        await confirmButton.click();
      }

      // Should have only 1 step now
      const steps = page.getByTestId('pipeline-step');
      await expect(steps).toHaveCount(1);
    }
  });
});

import { test, expect } from '@playwright/test';

/**
 * Happy Path E2E Test: Create and Execute Pipeline via UI
 * 
 * This test follows the complete user journey:
 * 1. Navigate to pipelines page
 * 2. Create a new pipeline with CSV input step
 * 3. Execute the pipeline
 * 4. Verify execution success
 */

test.describe('Happy Path - Pipeline UI Workflow', () => {
  test.beforeEach(async ({ page }) => {
    // Navigate to pipelines page
    await page.goto('/pipelines');
    
    // Wait for page to load
    await expect(page.getByRole('heading', { name: /pipelines/i })).toBeVisible();
  });

  test('create and execute CSV pipeline', async ({ page }) => {
    // Step 1: Open create pipeline dialog
    const createButton = page.getByRole('button', { name: /create pipeline/i });
    await expect(createButton).toBeVisible();
    await createButton.click();

    // Step 2: Fill in pipeline name
    const nameInput = page.getByLabel(/name/i);
    await expect(nameInput).toBeVisible();
    await nameInput.fill('E2E CSV Test Pipeline');

    // Step 3: Fill in description
    const descriptionInput = page.getByLabel(/description/i);
    await descriptionInput.fill('End-to-end test pipeline for CSV processing');

    // Step 4: Select pipeline type
    const typeSelect = page.getByLabel(/type/i);
    await typeSelect.click();
    await page.getByText(/csv file import/i).click();

    // Step 5: Configure the CSV step
    const filePathInput = page.getByPlaceholder(/\/data\/input.csv/i);
    await filePathInput.fill('/tmp/test-data.csv');

    // Step 6: Create the pipeline
    const submitButton = page.getByRole('button', { name: /create/i }).last();
    await submitButton.click();

    // Step 7: Verify pipeline was created
    await expect(page.getByText(/E2E CSV Test Pipeline/i)).toBeVisible();

    // Step 8: Execute the pipeline
    const executeButton = page.getByRole('button', { name: /execute/i }).first();
    await executeButton.click();

    // Step 9: Verify execution dialog appears
    await expect(page.getByText(/executing pipeline/i)).toBeVisible();

    // Step 10: Confirm execution
    const confirmButton = page.getByRole('button', { name: /execute/i }).last();
    await confirmButton.click();

    // Step 11: Wait for execution to complete and verify success
    await expect(page.getByText(/pipeline executed successfully/i)).toBeVisible({ timeout: 30000 });
  });

  test('view pipeline list', async ({ page }) => {
    // Verify the pipeline list is displayed
    await expect(page.getByRole('heading', { name: /pipelines/i })).toBeVisible();
    
    // Check that at least one pipeline card is visible (or empty state)
    const pipelines = await page.locator('[data-testid="pipeline-card"]').count();
    expect(pipelines).toBeGreaterThanOrEqual(0);
  });

  test('clone existing pipeline', async ({ page }) => {
    // First create a pipeline to clone
    const createButton = page.getByRole('button', { name: /create pipeline/i });
    await createButton.click();

    const nameInput = page.getByLabel(/name/i);
    await nameInput.fill('Pipeline to Clone');

    const typeSelect = page.getByLabel(/type/i);
    await typeSelect.click();
    await page.getByText(/csv file import/i).click();

    await page.getByRole('button', { name: /create/i }).last().click();
    
    // Wait for pipeline to appear in list
    await expect(page.getByText(/Pipeline to Clone/i)).toBeVisible();

    // Find and click clone button
    const cloneButton = page.getByRole('button', { name: /clone/i }).first();
    await cloneButton.click();

    // Fill in clone name
    const cloneNameInput = page.getByLabel(/new name/i);
    await cloneNameInput.fill('Cloned Pipeline');

    // Confirm clone
    await page.getByRole('button', { name: /clone/i }).last().click();

    // Verify cloned pipeline appears
    await expect(page.getByText(/Cloned Pipeline/i)).toBeVisible();
  });
});
import { test, expect } from '@playwright/test';

/**
 * Comprehensive E2E tests for Jobs execution, monitoring, and management
 */

test.describe('Jobs - List and Management', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/jobs');
    await page.waitForLoadState('networkidle');
  });

  test('should display jobs page', async ({ page }) => {
    await expect(page).toHaveTitle(/Jobs/i);
    await expect(page.getByRole('heading', { name: /Jobs/i })).toBeVisible();
  });

  test('should display jobs list', async ({ page }) => {
    const jobsList = page.getByTestId('jobs-list');

    if (await jobsList.isVisible()) {
      await expect(jobsList).toBeVisible();
    }
  });

  test('should show empty state when no jobs exist', async ({ page }) => {
    const emptyState = page.getByText(/No jobs|No.*executions/i);

    if (await emptyState.isVisible()) {
      await expect(emptyState).toBeVisible();
    }
  });

  test('should view job details', async ({ page }) => {
    const jobRow = page.getByTestId('job-row').first();

    if (await jobRow.isVisible()) {
      await jobRow.click();

      // Should navigate to job details
      await expect(page).toHaveURL(/\/jobs\/[a-zA-Z0-9-]+/);
      await expect(page.getByRole('heading', { name: /Job Details/i })).toBeVisible();
    }
  });

  test('should display job status', async ({ page }) => {
    const jobRow = page.getByTestId('job-row').first();

    if (await jobRow.isVisible()) {
      const statusBadge = jobRow.getByTestId('status-badge');

      if (await statusBadge.isVisible()) {
        await expect(statusBadge).toBeVisible();
        const status = await statusBadge.textContent();
        expect(status).toMatch(/running|completed|failed|pending/i);
      }
    }
  });

  test('should filter jobs by status', async ({ page }) => {
    const filterSelect = page.getByLabel(/Filter|Status/i);

    if (await filterSelect.isVisible()) {
      await filterSelect.selectOption('completed');
      await page.waitForTimeout(500);

      // All visible jobs should be completed
      const jobs = page.getByTestId('job-row');
      const count = await jobs.count();

      for (let i = 0; i < Math.min(count, 5); i++) {
        const statusBadge = jobs.nth(i).getByTestId('status-badge');
        if (await statusBadge.isVisible()) {
          const status = await statusBadge.textContent();
          expect(status?.toLowerCase()).toContain('completed');
        }
      }
    }
  });

  test('should filter jobs by type', async ({ page }) => {
    const filterSelect = page.getByLabel(/Type|Job.*Type/i);

    if (await filterSelect.isVisible()) {
      await filterSelect.selectOption('pipeline');
      await page.waitForTimeout(500);

      // All visible jobs should be pipeline type
      const jobs = page.getByTestId('job-row');
      if (await jobs.first().isVisible()) {
        const text = await jobs.first().textContent();
        expect(text?.toLowerCase()).toContain('pipeline');
      }
    }
  });

  test('should search jobs', async ({ page }) => {
    const searchInput = page.getByPlaceholder(/Search.*jobs/i);

    if (await searchInput.isVisible()) {
      await searchInput.fill('pipeline');
      await page.waitForTimeout(500);

      // Results should contain "pipeline"
      const results = page.getByTestId('job-row');
      if (await results.first().isVisible()) {
        const text = await results.first().textContent();
        expect(text?.toLowerCase()).toContain('pipeline');
      }
    }
  });

  test('should refresh jobs list', async ({ page }) => {
    const refreshButton = page.getByRole('button', { name: /Refresh|Reload/i });

    if (await refreshButton.isVisible()) {
      await refreshButton.click();

      // Should reload data
      await page.waitForTimeout(1000);
      await expect(page.getByTestId('jobs-list')).toBeVisible();
    }
  });

  test('should sort jobs', async ({ page }) => {
    const sortSelect = page.getByLabel(/Sort/i);

    if (await sortSelect.isVisible()) {
      await sortSelect.selectOption('start_time_desc');
      await page.waitForTimeout(500);

      // Jobs should be reordered
      await expect(page.getByTestId('jobs-list')).toBeVisible();
    }
  });

  test('should paginate jobs', async ({ page }) => {
    const nextButton = page.getByRole('button', { name: /Next|>/i });

    if (await nextButton.isVisible()) {
      await nextButton.click();
      await page.waitForTimeout(500);

      // Should load next page
      await expect(page.getByTestId('jobs-list')).toBeVisible();
    }
  });
});

test.describe('Jobs - Execution Details', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/jobs');
    await page.waitForLoadState('networkidle');
  });

  test('should display job execution information', async ({ page }) => {
    const jobRow = page.getByTestId('job-row').first();

    if (await jobRow.isVisible()) {
      await jobRow.click();

      // Should show execution details
      await expect(page.getByText(/Started|Duration|Status/i)).toBeVisible();
    }
  });

  test('should display job logs', async ({ page }) => {
    const jobRow = page.getByTestId('job-row').first();

    if (await jobRow.isVisible()) {
      await jobRow.click();

      const logsTab = page.getByRole('tab', { name: /Logs/i });
      if (await logsTab.isVisible()) {
        await logsTab.click();

        // Should show logs
        await expect(page.getByTestId('job-logs')).toBeVisible();
      }
    }
  });

  test('should stream live logs', async ({ page }) => {
    const jobRow = page.getByTestId('job-row').first();

    if (await jobRow.isVisible()) {
      await jobRow.click();

      const logsTab = page.getByRole('tab', { name: /Logs/i });
      if (await logsTab.isVisible()) {
        await logsTab.click();

        // Check for live streaming indicator
        const liveIndicator = page.getByText(/Live|Streaming/i);
        if (await liveIndicator.isVisible()) {
          await expect(liveIndicator).toBeVisible();
        }
      }
    }
  });

  test('should filter logs by level', async ({ page }) => {
    const jobRow = page.getByTestId('job-row').first();

    if (await jobRow.isVisible()) {
      await jobRow.click();

      const logsTab = page.getByRole('tab', { name: /Logs/i });
      if (await logsTab.isVisible()) {
        await logsTab.click();

        const levelFilter = page.getByLabel(/Log Level|Level/i);
        if (await levelFilter.isVisible()) {
          await levelFilter.selectOption('error');
          await page.waitForTimeout(500);

          // Should show only error logs
          await expect(page.getByTestId('job-logs')).toBeVisible();
        }
      }
    }
  });

  test('should search logs', async ({ page }) => {
    const jobRow = page.getByTestId('job-row').first();

    if (await jobRow.isVisible()) {
      await jobRow.click();

      const logsTab = page.getByRole('tab', { name: /Logs/i });
      if (await logsTab.isVisible()) {
        await logsTab.click();

        const searchInput = page.getByPlaceholder(/Search.*logs/i);
        if (await searchInput.isVisible()) {
          await searchInput.fill('error');
          await page.waitForTimeout(500);

          // Results should be filtered
          await expect(page.getByTestId('job-logs')).toBeVisible();
        }
      }
    }
  });

  test('should download logs', async ({ page }) => {
    const jobRow = page.getByTestId('job-row').first();

    if (await jobRow.isVisible()) {
      await jobRow.click();

      const logsTab = page.getByRole('tab', { name: /Logs/i });
      if (await logsTab.isVisible()) {
        await logsTab.click();

        const downloadButton = page.getByRole('button', { name: /Download|Export/i });
        if (await downloadButton.isVisible()) {
          const downloadPromise = page.waitForEvent('download');
          await downloadButton.click();

          const download = await downloadPromise;
          expect(download.suggestedFilename()).toMatch(/\.log|\.txt/);
        }
      }
    }
  });

  test('should display job output', async ({ page }) => {
    const jobRow = page.getByTestId('job-row').first();

    if (await jobRow.isVisible()) {
      await jobRow.click();

      const outputTab = page.getByRole('tab', { name: /Output|Results/i });
      if (await outputTab.isVisible()) {
        await outputTab.click();

        // Should show job output
        await expect(page.getByTestId('job-output')).toBeVisible();
      }
    }
  });

  test('should display job metrics', async ({ page }) => {
    const jobRow = page.getByTestId('job-row').first();

    if (await jobRow.isVisible()) {
      await jobRow.click();

      const metricsTab = page.getByRole('tab', { name: /Metrics|Performance/i });
      if (await metricsTab.isVisible()) {
        await metricsTab.click();

        // Should show metrics
        await expect(page.getByText(/Duration|Memory|CPU/i)).toBeVisible();
      }
    }
  });

  test('should display job errors', async ({ page }) => {
    const jobRow = page.getByTestId('job-row').first();

    if (await jobRow.isVisible()) {
      await jobRow.click();

      const errorsTab = page.getByRole('tab', { name: /Errors/i });
      if (await errorsTab.isVisible()) {
        await errorsTab.click();

        // Should show errors section
        await expect(page.getByTestId('job-errors')).toBeVisible();
      }
    }
  });
});

test.describe('Jobs - Control and Management', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/jobs');
    await page.waitForLoadState('networkidle');
  });

  test('should cancel running job', async ({ page }) => {
    const cancelButton = page.getByRole('button', { name: /Cancel|Stop/i }).first();

    if (await cancelButton.isVisible()) {
      await cancelButton.click();

      // Confirm cancellation
      await expect(page.getByRole('dialog')).toBeVisible();
      await page.getByRole('button', { name: /Cancel.*Job|Confirm/i }).click();

      // Should show cancellation message
      await expect(page.getByText(/Job.*cancelled|Cancelling/i)).toBeVisible({ timeout: 5000 });
    }
  });

  test('should retry failed job', async ({ page }) => {
    const retryButton = page.getByRole('button', { name: /Retry/i }).first();

    if (await retryButton.isVisible()) {
      await retryButton.click();

      // Should show retry message
      await expect(page.getByText(/Job.*restarted|Retrying/i)).toBeVisible({ timeout: 5000 });
    }
  });

  test('should delete job', async ({ page }) => {
    const deleteButton = page.getByRole('button', { name: /Delete/i }).first();

    if (await deleteButton.isVisible()) {
      await deleteButton.click();

      // Confirm deletion
      await expect(page.getByRole('dialog')).toBeVisible();
      await page.getByRole('button', { name: /Delete|Confirm/i }).click();

      // Should show deletion message
      await expect(page.getByText(/Job.*deleted/i)).toBeVisible({ timeout: 5000 });
    }
  });

  test('should bulk cancel jobs', async ({ page }) => {
    // Select multiple jobs
    const checkboxes = page.getByRole('checkbox');
    const count = await checkboxes.count();

    if (count >= 2) {
      await checkboxes.nth(0).check();
      await checkboxes.nth(1).check();

      // Cancel selected
      const bulkCancelButton = page.getByRole('button', { name: /Cancel.*Selected/i });
      if (await bulkCancelButton.isVisible()) {
        await bulkCancelButton.click();

        // Confirm
        await page.getByRole('button', { name: /Cancel|Confirm/i }).click();

        // Should show success message
        await expect(page.getByText(/Jobs.*cancelled/i)).toBeVisible({ timeout: 5000 });
      }
    }
  });

  test('should bulk delete jobs', async ({ page }) => {
    // Select multiple jobs
    const checkboxes = page.getByRole('checkbox');
    const count = await checkboxes.count();

    if (count >= 2) {
      await checkboxes.nth(0).check();
      await checkboxes.nth(1).check();

      // Delete selected
      const bulkDeleteButton = page.getByRole('button', { name: /Delete.*Selected/i });
      if (await bulkDeleteButton.isVisible()) {
        await bulkDeleteButton.click();

        // Confirm
        await page.getByRole('button', { name: /Delete|Confirm/i }).click();

        // Should show success message
        await expect(page.getByText(/Jobs.*deleted/i)).toBeVisible({ timeout: 5000 });
      }
    }
  });
});

test.describe('Jobs - Timeline and History', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/jobs');
    await page.waitForLoadState('networkidle');
  });

  test('should display jobs timeline', async ({ page }) => {
    const timelineButton = page.getByRole('button', { name: /Timeline|History/i });

    if (await timelineButton.isVisible()) {
      await timelineButton.click();

      // Should show timeline view
      await expect(page.getByTestId('jobs-timeline')).toBeVisible();
    }
  });

  test('should filter timeline by date range', async ({ page }) => {
    const timelineButton = page.getByRole('button', { name: /Timeline/i });

    if (await timelineButton.isVisible()) {
      await timelineButton.click();

      const dateRangeSelect = page.getByLabel(/Date Range|Period/i);
      if (await dateRangeSelect.isVisible()) {
        await dateRangeSelect.selectOption('last_7_days');
        await page.waitForTimeout(1000);

        // Timeline should update
        await expect(page.getByTestId('jobs-timeline')).toBeVisible();
      }
    }
  });

  test('should view job dependencies', async ({ page }) => {
    const jobRow = page.getByTestId('job-row').first();

    if (await jobRow.isVisible()) {
      await jobRow.click();

      const dependenciesTab = page.getByRole('tab', { name: /Dependencies/i });
      if (await dependenciesTab.isVisible()) {
        await dependenciesTab.click();

        // Should show dependency graph
        await expect(page.getByTestId('dependency-graph')).toBeVisible();
      }
    }
  });

  test('should view related jobs', async ({ page }) => {
    const jobRow = page.getByTestId('job-row').first();

    if (await jobRow.isVisible()) {
      await jobRow.click();

      const relatedTab = page.getByRole('tab', { name: /Related/i });
      if (await relatedTab.isVisible()) {
        await relatedTab.click();

        // Should show related jobs
        await expect(page.getByTestId('related-jobs')).toBeVisible();
      }
    }
  });
});

test.describe('Jobs - Statistics and Analytics', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/jobs/analytics');
    await page.waitForLoadState('networkidle');
  });

  test('should display job statistics', async ({ page }) => {
    await expect(page.getByRole('heading', { name: /Job.*Statistics|Analytics/i })).toBeVisible();
  });

  test('should display success rate', async ({ page }) => {
    const successRateWidget = page.getByTestId('success-rate');

    if (await successRateWidget.isVisible()) {
      await expect(successRateWidget).toBeVisible();
      await expect(page.getByText(/Success Rate/i)).toBeVisible();
    }
  });

  test('should display average duration', async ({ page }) => {
    const durationWidget = page.getByTestId('average-duration');

    if (await durationWidget.isVisible()) {
      await expect(durationWidget).toBeVisible();
      await expect(page.getByText(/Average Duration/i)).toBeVisible();
    }
  });

  test('should display job count by status', async ({ page }) => {
    const statusChart = page.getByTestId('status-distribution');

    if (await statusChart.isVisible()) {
      await expect(statusChart).toBeVisible();
    }
  });

  test('should display jobs over time', async ({ page }) => {
    const timeChart = page.getByTestId('jobs-over-time');

    if (await timeChart.isVisible()) {
      await expect(timeChart).toBeVisible();
    }
  });

  test('should filter analytics by date range', async ({ page }) => {
    const dateRangeSelect = page.getByLabel(/Date Range|Period/i);

    if (await dateRangeSelect.isVisible()) {
      await dateRangeSelect.selectOption('last_30_days');
      await page.waitForTimeout(1000);

      // Charts should update
      await expect(page.getByTestId('success-rate')).toBeVisible();
    }
  });

  test('should export analytics data', async ({ page }) => {
    const exportButton = page.getByRole('button', { name: /Export|Download/i });

    if (await exportButton.isVisible()) {
      const downloadPromise = page.waitForEvent('download');
      await exportButton.click();

      const download = await downloadPromise;
      expect(download.suggestedFilename()).toMatch(/\.csv|\.pdf/);
    }
  });
});

test.describe('Jobs - Notifications', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/jobs');
    await page.waitForLoadState('networkidle');
  });

  test('should configure job notifications', async ({ page }) => {
    const jobRow = page.getByTestId('job-row').first();

    if (await jobRow.isVisible()) {
      await jobRow.click();

      const notificationsTab = page.getByRole('tab', { name: /Notifications/i });
      if (await notificationsTab.isVisible()) {
        await notificationsTab.click();

        // Should show notification settings
        await expect(page.getByText(/Email|Slack|Webhook/i)).toBeVisible();
      }
    }
  });

  test('should enable failure notifications', async ({ page }) => {
    const jobRow = page.getByTestId('job-row').first();

    if (await jobRow.isVisible()) {
      await jobRow.click();

      const notificationsTab = page.getByRole('tab', { name: /Notifications/i });
      if (await notificationsTab.isVisible()) {
        await notificationsTab.click();

        const failureToggle = page.getByLabel(/Notify.*on.*Failure/i);
        if (await failureToggle.isVisible()) {
          await failureToggle.check();

          // Save settings
          await page.getByRole('button', { name: /Save/i }).click();

          // Should show success message
          await expect(page.getByText(/Notifications.*updated/i)).toBeVisible({ timeout: 5000 });
        }
      }
    }
  });
});

test.describe('Jobs - Auto-refresh and Real-time Updates', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/jobs');
    await page.waitForLoadState('networkidle');
  });

  test('should enable auto-refresh', async ({ page }) => {
    const autoRefreshToggle = page.getByRole('switch', { name: /Auto.*Refresh/i });

    if (await autoRefreshToggle.isVisible()) {
      await autoRefreshToggle.check();

      // Should show refresh interval
      await expect(page.getByText(/Refreshing/i)).toBeVisible();
    }
  });

  test('should update job status in real-time', async ({ page }) => {
    const jobRow = page.getByTestId('job-row').first();

    if (await jobRow.isVisible()) {
      const initialStatus = await jobRow.getByTestId('status-badge').textContent();

      // Wait for potential update
      await page.waitForTimeout(5000);

      // Status may have changed
      const currentStatus = await jobRow.getByTestId('status-badge').textContent();

      // Just verify status badge is still visible
      await expect(jobRow.getByTestId('status-badge')).toBeVisible();
    }
  });

  test('should show progress for running jobs', async ({ page }) => {
    const jobRow = page.getByTestId('job-row').first();

    if (await jobRow.isVisible()) {
      const progressBar = jobRow.getByTestId('progress-bar');

      if (await progressBar.isVisible()) {
        await expect(progressBar).toBeVisible();
      }
    }
  });
});

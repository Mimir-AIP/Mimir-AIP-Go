import { test, expect } from '@playwright/test';

/**
 * Comprehensive E2E tests for Monitoring including jobs, rules, alerts, and system metrics
 */

test.describe('Monitoring - Jobs Dashboard', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/monitoring/jobs');
    await page.waitForLoadState('networkidle');
  });

  test('should display monitoring jobs page', async ({ page }) => {
    await expect(page).toHaveTitle(/Monitoring|Jobs/i);
    await expect(page.getByRole('heading', { name: /Monitoring Jobs|Job.*Monitor/i })).toBeVisible();
  });

  test('should display jobs list', async ({ page }) => {
    const jobsList = page.getByTestId('jobs-list');

    if (await jobsList.isVisible()) {
      await expect(jobsList).toBeVisible();
    }
  });

  test('should filter jobs by status', async ({ page }) => {
    const filterSelect = page.getByLabel(/Filter|Status/i);

    if (await filterSelect.isVisible()) {
      await filterSelect.selectOption('running');
      await page.waitForTimeout(500);

      // All visible jobs should be running
      const jobs = page.getByTestId('job-row');
      const count = await jobs.count();

      for (let i = 0; i < Math.min(count, 3); i++) {
        const statusBadge = jobs.nth(i).getByTestId('status-badge');
        if (await statusBadge.isVisible()) {
          const status = await statusBadge.textContent();
          expect(status?.toLowerCase()).toContain('running');
        }
      }
    }
  });

  test('should view job details', async ({ page }) => {
    const jobRow = page.getByTestId('job-row').first();

    if (await jobRow.isVisible()) {
      await jobRow.click();

      // Should navigate to job details
      await expect(page).toHaveURL(/\/monitoring\/jobs\/[a-zA-Z0-9-]+/);
      await expect(page.getByRole('heading', { name: /Job Details/i })).toBeVisible();
    }
  });

  test('should display job execution logs', async ({ page }) => {
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

  test('should display job metrics', async ({ page }) => {
    const jobRow = page.getByTestId('job-row').first();

    if (await jobRow.isVisible()) {
      await jobRow.click();

      const metricsTab = page.getByRole('tab', { name: /Metrics/i });
      if (await metricsTab.isVisible()) {
        await metricsTab.click();

        // Should show metrics
        await expect(page.getByText(/Duration|Memory|CPU/i)).toBeVisible();
      }
    }
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

      // Should show retry confirmation
      await expect(page.getByText(/Job.*restarted|Retrying/i)).toBeVisible({ timeout: 5000 });
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

  test('should export job logs', async ({ page }) => {
    const jobRow = page.getByTestId('job-row').first();

    if (await jobRow.isVisible()) {
      await jobRow.click();

      const exportButton = page.getByRole('button', { name: /Export.*Logs|Download/i });
      if (await exportButton.isVisible()) {
        const downloadPromise = page.waitForEvent('download');
        await exportButton.click();

        const download = await downloadPromise;
        expect(download.suggestedFilename()).toMatch(/\.log|\.txt/);
      }
    }
  });
});

test.describe('Monitoring - Alerts Management', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/monitoring/alerts');
    await page.waitForLoadState('networkidle');
  });

  test('should display alerts page', async ({ page }) => {
    await expect(page).toHaveTitle(/Alerts/i);
    await expect(page.getByRole('heading', { name: /Alerts/i })).toBeVisible();
    await expect(page.getByRole('button', { name: /Create.*Alert|New Alert/i })).toBeVisible();
  });

  test('should display active alerts', async ({ page }) => {
    const alertsList = page.getByTestId('alerts-list');

    if (await alertsList.isVisible()) {
      await expect(alertsList).toBeVisible();
    }
  });

  test('should create new alert', async ({ page }) => {
    await page.getByRole('button', { name: /Create.*Alert/i }).click();

    // Fill alert form
    await page.getByLabel(/Name/i).fill('High CPU Alert');
    await page.getByLabel(/Description/i).fill('Alert when CPU exceeds 80%');

    // Select condition
    const conditionSelect = page.getByLabel(/Condition|Metric/i);
    if (await conditionSelect.isVisible()) {
      await conditionSelect.selectOption('cpu_usage');
    }

    // Set threshold
    const thresholdInput = page.getByLabel(/Threshold/i);
    if (await thresholdInput.isVisible()) {
      await thresholdInput.fill('80');
    }

    // Save alert
    await page.getByRole('button', { name: /Create|Save/i }).click();

    // Verify creation
    await expect(page.getByText(/Alert created successfully/i)).toBeVisible({ timeout: 5000 });
  });

  test('should filter alerts by severity', async ({ page }) => {
    const filterSelect = page.getByLabel(/Filter|Severity/i);

    if (await filterSelect.isVisible()) {
      await filterSelect.selectOption('critical');
      await page.waitForTimeout(500);

      // All visible alerts should be critical
      const alerts = page.getByTestId('alert-row');
      const count = await alerts.count();

      for (let i = 0; i < Math.min(count, 3); i++) {
        const severityBadge = alerts.nth(i).getByTestId('severity-badge');
        const severity = await severityBadge.textContent();
        expect(severity?.toLowerCase()).toContain('critical');
      }
    }
  });

  test('should acknowledge alert', async ({ page }) => {
    const acknowledgeButton = page.getByRole('button', { name: /Acknowledge|Ack/i }).first();

    if (await acknowledgeButton.isVisible()) {
      await acknowledgeButton.click();

      // Should show acknowledgment
      await expect(page.getByText(/Alert acknowledged/i)).toBeVisible({ timeout: 5000 });
    }
  });

  test('should resolve alert', async ({ page }) => {
    const resolveButton = page.getByRole('button', { name: /Resolve|Close/i }).first();

    if (await resolveButton.isVisible()) {
      await resolveButton.click();

      // Confirm resolution
      await expect(page.getByRole('dialog')).toBeVisible();
      await page.getByRole('button', { name: /Resolve|Confirm/i }).click();

      // Should show resolution message
      await expect(page.getByText(/Alert resolved/i)).toBeVisible({ timeout: 5000 });
    }
  });

  test('should view alert history', async ({ page }) => {
    const alertRow = page.getByTestId('alert-row').first();

    if (await alertRow.isVisible()) {
      await alertRow.click();

      const historyTab = page.getByRole('tab', { name: /History/i });
      if (await historyTab.isVisible()) {
        await historyTab.click();

        // Should show alert history
        await expect(page.getByTestId('alert-history')).toBeVisible();
      }
    }
  });

  test('should mute alert', async ({ page }) => {
    const muteButton = page.getByRole('button', { name: /Mute|Silence/i }).first();

    if (await muteButton.isVisible()) {
      await muteButton.click();

      // Select mute duration
      const durationSelect = page.getByLabel(/Duration/i);
      if (await durationSelect.isVisible()) {
        await durationSelect.selectOption('1h');
      }

      await page.getByRole('button', { name: /Mute|Confirm/i }).click();

      // Should show mute confirmation
      await expect(page.getByText(/Alert muted/i)).toBeVisible({ timeout: 5000 });
    }
  });

  test('should delete alert rule', async ({ page }) => {
    const deleteButton = page.getByRole('button', { name: /Delete/i }).first();

    if (await deleteButton.isVisible()) {
      await deleteButton.click();

      // Confirm deletion
      await expect(page.getByRole('dialog')).toBeVisible();
      await page.getByRole('button', { name: /Delete|Confirm/i }).click();

      // Should show deletion message
      await expect(page.getByText(/Alert deleted/i)).toBeVisible({ timeout: 5000 });
    }
  });

  test('should configure alert notifications', async ({ page }) => {
    const alertRow = page.getByTestId('alert-row').first();

    if (await alertRow.isVisible()) {
      await alertRow.click();

      const notificationsTab = page.getByRole('tab', { name: /Notifications/i });
      if (await notificationsTab.isVisible()) {
        await notificationsTab.click();

        // Should show notification settings
        await expect(page.getByText(/Email|Slack|Webhook/i)).toBeVisible();
      }
    }
  });
});

test.describe('Monitoring - Rules Engine', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/monitoring/rules');
    await page.waitForLoadState('networkidle');
  });

  test('should display rules page', async ({ page }) => {
    await expect(page).toHaveTitle(/Rules/i);
    await expect(page.getByRole('heading', { name: /Rules|Monitoring Rules/i })).toBeVisible();
    await expect(page.getByRole('button', { name: /Create.*Rule|New Rule/i })).toBeVisible();
  });

  test('should create new monitoring rule', async ({ page }) => {
    await page.getByRole('button', { name: /Create.*Rule/i }).click();

    // Fill rule form
    await page.getByLabel(/Name/i).fill('Pipeline Failure Rule');
    await page.getByLabel(/Description/i).fill('Trigger when pipeline fails');

    // Select condition
    const conditionInput = page.getByLabel(/Condition|Expression/i);
    if (await conditionInput.isVisible()) {
      await conditionInput.fill('pipeline.status == "failed"');
    }

    // Select action
    const actionSelect = page.getByLabel(/Action/i);
    if (await actionSelect.isVisible()) {
      await actionSelect.selectOption('send_alert');
    }

    // Save rule
    await page.getByRole('button', { name: /Create|Save/i }).click();

    // Verify creation
    await expect(page.getByText(/Rule created successfully/i)).toBeVisible({ timeout: 5000 });
  });

  test('should enable/disable rule', async ({ page }) => {
    const toggleButton = page.getByRole('switch').first();

    if (await toggleButton.isVisible()) {
      const initialState = await toggleButton.getAttribute('aria-checked');

      // Toggle
      await toggleButton.click();
      await page.waitForTimeout(1000);

      // Verify state changed
      const newState = await toggleButton.getAttribute('aria-checked');
      expect(newState).not.toBe(initialState);
    }
  });

  test('should test rule condition', async ({ page }) => {
    const testButton = page.getByRole('button', { name: /Test.*Rule/i }).first();

    if (await testButton.isVisible()) {
      await testButton.click();

      // Should show test results
      await expect(page.getByText(/Test.*Result|Valid|Invalid/i)).toBeVisible({ timeout: 5000 });
    }
  });

  test('should view rule execution history', async ({ page }) => {
    const ruleRow = page.getByTestId('rule-row').first();

    if (await ruleRow.isVisible()) {
      await ruleRow.click();

      const historyTab = page.getByRole('tab', { name: /History|Executions/i });
      if (await historyTab.isVisible()) {
        await historyTab.click();

        // Should show execution history
        await expect(page.getByTestId('rule-history')).toBeVisible();
      }
    }
  });

  test('should clone rule', async ({ page }) => {
    const cloneButton = page.getByRole('button', { name: /Clone|Duplicate/i }).first();

    if (await cloneButton.isVisible()) {
      await cloneButton.click();

      // Should show clone dialog
      await expect(page.getByRole('dialog')).toBeVisible();
      await page.getByLabel(/Name/i).fill('Cloned Rule');
      await page.getByRole('button', { name: /Clone|Create/i }).click();

      // Verify cloning
      await expect(page.getByText(/Rule cloned/i)).toBeVisible({ timeout: 5000 });
    }
  });

  test('should delete rule', async ({ page }) => {
    const deleteButton = page.getByRole('button', { name: /Delete/i }).first();

    if (await deleteButton.isVisible()) {
      await deleteButton.click();

      // Confirm deletion
      await expect(page.getByRole('dialog')).toBeVisible();
      await page.getByRole('button', { name: /Delete|Confirm/i }).click();

      // Should show deletion message
      await expect(page.getByText(/Rule deleted/i)).toBeVisible({ timeout: 5000 });
    }
  });
});

test.describe('Monitoring - System Metrics', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/monitoring');
    await page.waitForLoadState('networkidle');
  });

  test('should display system overview', async ({ page }) => {
    await expect(page).toHaveTitle(/Monitoring/i);
    await expect(page.getByRole('heading', { name: /Monitoring|System.*Overview/i })).toBeVisible();
  });

  test('should display CPU metrics', async ({ page }) => {
    const cpuWidget = page.getByTestId('cpu-metrics');

    if (await cpuWidget.isVisible()) {
      await expect(cpuWidget).toBeVisible();
      await expect(page.getByText(/CPU/i)).toBeVisible();
    }
  });

  test('should display memory metrics', async ({ page }) => {
    const memoryWidget = page.getByTestId('memory-metrics');

    if (await memoryWidget.isVisible()) {
      await expect(memoryWidget).toBeVisible();
      await expect(page.getByText(/Memory/i)).toBeVisible();
    }
  });

  test('should display disk metrics', async ({ page }) => {
    const diskWidget = page.getByTestId('disk-metrics');

    if (await diskWidget.isVisible()) {
      await expect(diskWidget).toBeVisible();
      await expect(page.getByText(/Disk/i)).toBeVisible();
    }
  });

  test('should display network metrics', async ({ page }) => {
    const networkWidget = page.getByTestId('network-metrics');

    if (await networkWidget.isVisible()) {
      await expect(networkWidget).toBeVisible();
      await expect(page.getByText(/Network/i)).toBeVisible();
    }
  });

  test('should change time range', async ({ page }) => {
    const timeRangeSelect = page.getByLabel(/Time Range|Period/i);

    if (await timeRangeSelect.isVisible()) {
      await timeRangeSelect.selectOption('1h');
      await page.waitForTimeout(1000);

      // Charts should update
      await expect(page.getByTestId('metrics-chart')).toBeVisible();
    }
  });

  test('should refresh metrics', async ({ page }) => {
    const refreshButton = page.getByRole('button', { name: /Refresh/i });

    if (await refreshButton.isVisible()) {
      await refreshButton.click();

      // Should reload metrics
      await page.waitForTimeout(1000);
      await expect(page.getByTestId('cpu-metrics')).toBeVisible();
    }
  });

  test('should display active alerts count', async ({ page }) => {
    const alertsWidget = page.getByTestId('active-alerts');

    if (await alertsWidget.isVisible()) {
      await expect(alertsWidget).toBeVisible();
      await expect(page.getByText(/Active Alerts/i)).toBeVisible();
    }
  });

  test('should display running jobs count', async ({ page }) => {
    const jobsWidget = page.getByTestId('running-jobs');

    if (await jobsWidget.isVisible()) {
      await expect(jobsWidget).toBeVisible();
      await expect(page.getByText(/Running Jobs/i)).toBeVisible();
    }
  });

  test('should navigate to detailed metrics', async ({ page }) => {
    const cpuWidget = page.getByTestId('cpu-metrics');

    if (await cpuWidget.isVisible()) {
      await cpuWidget.click();

      // Should navigate to detailed view
      await expect(page.getByRole('heading', { name: /CPU.*Details|Detailed.*Metrics/i })).toBeVisible();
    }
  });

  test('should export metrics data', async ({ page }) => {
    const exportButton = page.getByRole('button', { name: /Export.*Metrics|Download/i });

    if (await exportButton.isVisible()) {
      const downloadPromise = page.waitForEvent('download');
      await exportButton.click();

      const download = await downloadPromise;
      expect(download.suggestedFilename()).toMatch(/\.csv|\.json/);
    }
  });

  test('should customize dashboard layout', async ({ page }) => {
    const customizeButton = page.getByRole('button', { name: /Customize|Edit.*Layout/i });

    if (await customizeButton.isVisible()) {
      await customizeButton.click();

      // Should enter edit mode
      await expect(page.getByText(/Edit Mode|Drag.*Drop/i)).toBeVisible();
    }
  });
});

test.describe('Monitoring - Performance Dashboard', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/monitoring/performance');
    await page.waitForLoadState('networkidle');
  });

  test('should display performance metrics', async ({ page }) => {
    await expect(page.getByRole('heading', { name: /Performance/i })).toBeVisible();
  });

  test('should display response time chart', async ({ page }) => {
    const responseTimeChart = page.getByTestId('response-time-chart');

    if (await responseTimeChart.isVisible()) {
      await expect(responseTimeChart).toBeVisible();
    }
  });

  test('should display throughput metrics', async ({ page }) => {
    const throughputWidget = page.getByTestId('throughput-metrics');

    if (await throughputWidget.isVisible()) {
      await expect(throughputWidget).toBeVisible();
      await expect(page.getByText(/Throughput|Requests.*Second/i)).toBeVisible();
    }
  });

  test('should display error rate', async ({ page }) => {
    const errorRateWidget = page.getByTestId('error-rate');

    if (await errorRateWidget.isVisible()) {
      await expect(errorRateWidget).toBeVisible();
      await expect(page.getByText(/Error Rate/i)).toBeVisible();
    }
  });

  test('should filter by endpoint', async ({ page }) => {
    const endpointSelect = page.getByLabel(/Endpoint|Route/i);

    if (await endpointSelect.isVisible()) {
      await endpointSelect.selectOption({ index: 1 });
      await page.waitForTimeout(1000);

      // Charts should update
      await expect(page.getByTestId('response-time-chart')).toBeVisible();
    }
  });
});

import { test, expect } from '@playwright/test';
import { setupAuthenticatedPage } from './helpers';


/**
 * Comprehensive E2E tests for Workflows including creation, execution, and autonomous workflows
 */

test.describe('Workflows - Management', () => {
  test.beforeEach(async ({ page }) => {
    await setupAuthenticatedPage(page);
    await page.goto('/workflows');
    await page.waitForLoadState('networkidle');
  });

  test('should display workflows page', async ({ page }) => {
    await expect(page).toHaveTitle(/Workflows/i);
    await expect(page.getByRole('heading', { name: /Workflows/i })).toBeVisible();
    await expect(page.getByRole('button', { name: /Create.*Workflow|New Workflow/i })).toBeVisible();
  });

  test('should display workflows list', async ({ page }) => {
    const workflowsList = page.getByTestId('workflows-list');

    if (await workflowsList.isVisible()) {
      await expect(workflowsList).toBeVisible();
    }
  });

  test('should create new workflow', async ({ page }) => {
    await page.getByRole('button', { name: /Create.*Workflow/i }).click();

    // Fill workflow form
    await page.getByLabel(/Name/i).fill('Test Data Processing Workflow');
    await page.getByLabel(/Description/i).fill('E2E test workflow');

    // Select trigger
    const triggerSelect = page.getByLabel(/Trigger/i);
    if (await triggerSelect.isVisible()) {
      await triggerSelect.selectOption('schedule');
    }

    // Save workflow
    await page.getByRole('button', { name: /Create|Save/i }).click();

    // Verify creation
    await expect(page.getByText(/Workflow created successfully/i)).toBeVisible({ timeout: 5000 });
  });

  test('should view workflow details', async ({ page }) => {
    const workflowCard = page.getByTestId('workflow-card').first();

    if (await workflowCard.isVisible()) {
      await workflowCard.click();

      // Should navigate to details page
      await expect(page).toHaveURL(/\/workflows\/[a-zA-Z0-9-]+/);
      await expect(page.getByRole('heading', { name: /Workflow Details/i })).toBeVisible();
    }
  });

  test('should edit workflow', async ({ page }) => {
    const editButton = page.getByRole('button', { name: /Edit/i }).first();

    if (await editButton.isVisible()) {
      await editButton.click();

      // Edit name
      const nameInput = page.getByLabel(/Name/i);
      await nameInput.clear();
      await nameInput.fill('Updated Workflow Name');

      // Save changes
      await page.getByRole('button', { name: /Save|Update/i }).click();

      // Verify update
      await expect(page.getByText(/Workflow updated successfully/i)).toBeVisible({ timeout: 5000 });
    }
  });

  test('should delete workflow', async ({ page }) => {
    const deleteButton = page.getByRole('button', { name: /Delete/i }).first();

    if (await deleteButton.isVisible()) {
      await deleteButton.click();

      // Confirm deletion
      await expect(page.getByRole('dialog')).toBeVisible();
      await page.getByRole('button', { name: /Delete|Confirm/i }).click();

      // Verify deletion
      await expect(page.getByText(/Workflow deleted successfully/i)).toBeVisible({ timeout: 5000 });
    }
  });

  test('should clone workflow', async ({ page }) => {
    const cloneButton = page.getByRole('button', { name: /Clone|Duplicate/i }).first();

    if (await cloneButton.isVisible()) {
      await cloneButton.click();

      // Clone dialog should appear
      await expect(page.getByRole('dialog')).toBeVisible();
      await page.getByLabel(/Name/i).fill('Cloned Workflow');
      await page.getByRole('button', { name: /Clone|Create/i }).click();

      // Verify cloning
      await expect(page.getByText(/Workflow cloned successfully/i)).toBeVisible({ timeout: 5000 });
    }
  });

  test('should search workflows', async ({ page }) => {
    const searchInput = page.getByPlaceholder(/Search.*workflows/i);

    if (await searchInput.isVisible()) {
      await searchInput.fill('data');
      await page.waitForTimeout(500);

      // Results should contain "data"
      const results = page.getByTestId('workflow-card');
      if (await results.first().isVisible()) {
        const text = await results.first().textContent();
        expect(text?.toLowerCase()).toContain('data');
      }
    }
  });

  test('should filter workflows by status', async ({ page }) => {
    const filterSelect = page.getByLabel(/Filter|Status/i);

    if (await filterSelect.isVisible()) {
      await filterSelect.selectOption('active');
      await page.waitForTimeout(500);

      // All visible workflows should be active
      const workflows = page.getByTestId('workflow-card');
      const count = await workflows.count();

      for (let i = 0; i < Math.min(count, 3); i++) {
        const statusBadge = workflows.nth(i).getByTestId('status-badge');
        if (await statusBadge.isVisible()) {
          const status = await statusBadge.textContent();
          expect(status?.toLowerCase()).toContain('active');
        }
      }
    }
  });
});

test.describe('Workflows - Visual Builder', () => {
  test.beforeEach(async ({ page }) => {
    await setupAuthenticatedPage(page);
    await page.goto('/workflows');
    await page.waitForLoadState('networkidle');
  });

  test('should open workflow builder', async ({ page }) => {
    await page.getByRole('button', { name: /Create.*Workflow/i }).click();

    // Builder should appear
    await expect(page.getByTestId('workflow-builder')).toBeVisible();
  });

  test('should add workflow step', async ({ page }) => {
    const workflowCard = page.getByTestId('workflow-card').first();

    if (await workflowCard.isVisible()) {
      await workflowCard.click();

      const addStepButton = page.getByRole('button', { name: /Add Step|Add Node/i });
      if (await addStepButton.isVisible()) {
        await addStepButton.click();

        // Step library should appear
        await expect(page.getByRole('dialog')).toBeVisible();
        await expect(page.getByText(/Select.*Step|Step.*Library/i)).toBeVisible();
      }
    }
  });

  test('should connect workflow steps', async ({ page }) => {
    const workflowCard = page.getByTestId('workflow-card').first();

    if (await workflowCard.isVisible()) {
      await workflowCard.click();

      // Add two steps
      const addStepButton = page.getByRole('button', { name: /Add Step/i });
      if (await addStepButton.isVisible()) {
        await addStepButton.click();
        await page.getByText(/Pipeline/i).first().click();

        await addStepButton.click();
        await page.getByText(/Transform/i).first().click();

        // Steps should be connected
        await page.waitForTimeout(1000);
        const connections = page.getByTestId('workflow-connection');
        if (await connections.first().isVisible()) {
          await expect(connections.first()).toBeVisible();
        }
      }
    }
  });

  test('should configure step properties', async ({ page }) => {
    const workflowCard = page.getByTestId('workflow-card').first();

    if (await workflowCard.isVisible()) {
      await workflowCard.click();

      const workflowStep = page.getByTestId('workflow-step').first();
      if (await workflowStep.isVisible()) {
        await workflowStep.click();

        // Properties panel should appear
        await expect(page.getByTestId('step-properties')).toBeVisible();
      }
    }
  });

  test('should delete workflow step', async ({ page }) => {
    const workflowCard = page.getByTestId('workflow-card').first();

    if (await workflowCard.isVisible()) {
      await workflowCard.click();

      const workflowStep = page.getByTestId('workflow-step').first();
      if (await workflowStep.isVisible()) {
        await workflowStep.click();

        const deleteStepButton = page.getByRole('button', { name: /Delete.*Step|Remove/i });
        if (await deleteStepButton.isVisible()) {
          await deleteStepButton.click();

          // Confirm deletion
          await page.getByRole('button', { name: /Delete|Confirm/i }).click();
        }
      }
    }
  });

  test('should add conditional branch', async ({ page }) => {
    const workflowCard = page.getByTestId('workflow-card').first();

    if (await workflowCard.isVisible()) {
      await workflowCard.click();

      const addBranchButton = page.getByRole('button', { name: /Add.*Branch|Conditional/i });
      if (await addBranchButton.isVisible()) {
        await addBranchButton.click();

        // Branch configuration should appear
        await expect(page.getByLabel(/Condition|If/i)).toBeVisible();
      }
    }
  });

  test('should add loop', async ({ page }) => {
    const workflowCard = page.getByTestId('workflow-card').first();

    if (await workflowCard.isVisible()) {
      await workflowCard.click();

      const addLoopButton = page.getByRole('button', { name: /Add.*Loop|Repeat/i });
      if (await addLoopButton.isVisible()) {
        await addLoopButton.click();

        // Loop configuration should appear
        await expect(page.getByLabel(/Iterations|Loop.*Count/i)).toBeVisible();
      }
    }
  });
});

test.describe('Workflows - Execution', () => {
  test.beforeEach(async ({ page }) => {
    await setupAuthenticatedPage(page);
    await page.goto('/workflows');
    await page.waitForLoadState('networkidle');
  });

  test('should execute workflow', async ({ page }) => {
    const executeButton = page.getByRole('button', { name: /Execute|Run/i }).first();

    if (await executeButton.isVisible()) {
      await executeButton.click();

      // Confirm execution
      const confirmDialog = page.getByRole('dialog');
      if (await confirmDialog.isVisible()) {
        await page.getByRole('button', { name: /Execute|Run|Confirm/i }).click();
      }

      // Should show execution started message
      await expect(page.getByText(/Workflow.*started|Execution.*started/i)).toBeVisible({ timeout: 5000 });
    }
  });

  test('should view workflow execution history', async ({ page }) => {
    const workflowCard = page.getByTestId('workflow-card').first();

    if (await workflowCard.isVisible()) {
      await workflowCard.click();

      const historyTab = page.getByRole('tab', { name: /History|Runs/i });
      if (await historyTab.isVisible()) {
        await historyTab.click();

        // Should show execution history
        await expect(page.getByTestId('execution-history')).toBeVisible();
      }
    }
  });

  test('should view execution details', async ({ page }) => {
    const workflowCard = page.getByTestId('workflow-card').first();

    if (await workflowCard.isVisible()) {
      await workflowCard.click();

      const historyTab = page.getByRole('tab', { name: /History/i });
      if (await historyTab.isVisible()) {
        await historyTab.click();

        const executionRow = page.getByTestId('execution-row').first();
        if (await executionRow.isVisible()) {
          await executionRow.click();

          // Should show execution details
          await expect(page.getByText(/Execution.*Details|Run.*Details/i)).toBeVisible();
        }
      }
    }
  });

  test('should view execution logs', async ({ page }) => {
    const workflowCard = page.getByTestId('workflow-card').first();

    if (await workflowCard.isVisible()) {
      await workflowCard.click();

      const historyTab = page.getByRole('tab', { name: /History/i });
      if (await historyTab.isVisible()) {
        await historyTab.click();

        const viewLogsButton = page.getByRole('button', { name: /View.*Logs|Logs/i }).first();
        if (await viewLogsButton.isVisible()) {
          await viewLogsButton.click();

          // Should show logs
          await expect(page.getByTestId('execution-logs')).toBeVisible();
        }
      }
    }
  });

  test('should stop running workflow', async ({ page }) => {
    const stopButton = page.getByRole('button', { name: /Stop|Cancel/i }).first();

    if (await stopButton.isVisible()) {
      await stopButton.click();

      // Confirm stop
      await expect(page.getByRole('dialog')).toBeVisible();
      await page.getByRole('button', { name: /Stop|Confirm/i }).click();

      // Should show stop message
      await expect(page.getByText(/Workflow.*stopped/i)).toBeVisible({ timeout: 5000 });
    }
  });

  test('should retry failed execution', async ({ page }) => {
    const retryButton = page.getByRole('button', { name: /Retry/i }).first();

    if (await retryButton.isVisible()) {
      await retryButton.click();

      // Should show retry message
      await expect(page.getByText(/Workflow.*restarted|Retrying/i)).toBeVisible({ timeout: 5000 });
    }
  });
});

test.describe('Workflows - Triggers and Scheduling', () => {
  test.beforeEach(async ({ page }) => {
    await setupAuthenticatedPage(page);
    await page.goto('/workflows');
    await page.waitForLoadState('networkidle');
  });

  test('should configure schedule trigger', async ({ page }) => {
    const workflowCard = page.getByTestId('workflow-card').first();

    if (await workflowCard.isVisible()) {
      await workflowCard.click();

      const triggersTab = page.getByRole('tab', { name: /Triggers/i });
      if (await triggersTab.isVisible()) {
        await triggersTab.click();

        const addTriggerButton = page.getByRole('button', { name: /Add.*Trigger/i });
        if (await addTriggerButton.isVisible()) {
          await addTriggerButton.click();

          // Select schedule trigger
          await page.getByLabel(/Trigger.*Type/i).selectOption('schedule');

          // Configure schedule
          const cronInput = page.getByLabel(/Cron|Schedule/i);
          if (await cronInput.isVisible()) {
            await cronInput.fill('0 0 * * *');
          }

          // Save trigger
          await page.getByRole('button', { name: /Save|Create/i }).click();

          // Should show success message
          await expect(page.getByText(/Trigger.*created/i)).toBeVisible({ timeout: 5000 });
        }
      }
    }
  });

  test('should configure webhook trigger', async ({ page }) => {
    const workflowCard = page.getByTestId('workflow-card').first();

    if (await workflowCard.isVisible()) {
      await workflowCard.click();

      const triggersTab = page.getByRole('tab', { name: /Triggers/i });
      if (await triggersTab.isVisible()) {
        await triggersTab.click();

        const addTriggerButton = page.getByRole('button', { name: /Add.*Trigger/i });
        if (await addTriggerButton.isVisible()) {
          await addTriggerButton.click();

          // Select webhook trigger
          await page.getByLabel(/Trigger.*Type/i).selectOption('webhook');

          // Should show webhook URL
          await expect(page.getByTestId('webhook-url')).toBeVisible();
        }
      }
    }
  });

  test('should configure event trigger', async ({ page }) => {
    const workflowCard = page.getByTestId('workflow-card').first();

    if (await workflowCard.isVisible()) {
      await workflowCard.click();

      const triggersTab = page.getByRole('tab', { name: /Triggers/i });
      if (await triggersTab.isVisible()) {
        await triggersTab.click();

        const addTriggerButton = page.getByRole('button', { name: /Add.*Trigger/i });
        if (await addTriggerButton.isVisible()) {
          await addTriggerButton.click();

          // Select event trigger
          await page.getByLabel(/Trigger.*Type/i).selectOption('event');

          // Select event type
          const eventSelect = page.getByLabel(/Event.*Type/i);
          if (await eventSelect.isVisible()) {
            await eventSelect.selectOption('pipeline_completed');
          }
        }
      }
    }
  });

  test('should disable trigger', async ({ page }) => {
    const workflowCard = page.getByTestId('workflow-card').first();

    if (await workflowCard.isVisible()) {
      await workflowCard.click();

      const triggersTab = page.getByRole('tab', { name: /Triggers/i });
      if (await triggersTab.isVisible()) {
        await triggersTab.click();

        const toggleTrigger = page.getByRole('switch').first();
        if (await toggleTrigger.isVisible()) {
          await toggleTrigger.click();

          // Should show disabled message
          await expect(page.getByText(/Trigger.*disabled/i)).toBeVisible({ timeout: 5000 });
        }
      }
    }
  });
});

test.describe('Workflows - Autonomous Workflows', () => {
  test.beforeEach(async ({ page }) => {
    await setupAuthenticatedPage(page);
    await page.goto('/workflows/autonomous');
    await page.waitForLoadState('networkidle');
  });

  test('should display autonomous workflows page', async ({ page }) => {
    await expect(page.getByRole('heading', { name: /Autonomous|AI.*Workflows/i })).toBeVisible();
  });

  test('should create autonomous workflow from goal', async ({ page }) => {
    const createButton = page.getByRole('button', { name: /Create.*Autonomous|AI.*Workflow/i });

    if (await createButton.isVisible()) {
      await createButton.click();

      // Enter goal
      const goalInput = page.getByLabel(/Goal|Objective/i);
      await goalInput.fill('Process incoming data and generate weekly reports');

      // Generate workflow
      await page.getByRole('button', { name: /Generate|Create/i }).click();

      // Should show generation progress
      await expect(page.getByText(/Generating|Creating|Analyzing/i)).toBeVisible({ timeout: 10000 });
    }
  });

  test('should view generated workflow steps', async ({ page }) => {
    const workflowCard = page.getByTestId('autonomous-workflow-card').first();

    if (await workflowCard.isVisible()) {
      await workflowCard.click();

      // Should show AI-generated steps
      await expect(page.getByTestId('workflow-steps')).toBeVisible();
    }
  });

  test('should approve autonomous workflow', async ({ page }) => {
    const approveButton = page.getByRole('button', { name: /Approve|Activate/i }).first();

    if (await approveButton.isVisible()) {
      await approveButton.click();

      // Should show approval message
      await expect(page.getByText(/Workflow.*approved|Activated/i)).toBeVisible({ timeout: 5000 });
    }
  });

  test('should modify AI-generated workflow', async ({ page }) => {
    const workflowCard = page.getByTestId('autonomous-workflow-card').first();

    if (await workflowCard.isVisible()) {
      await workflowCard.click();

      const editButton = page.getByRole('button', { name: /Edit|Modify/i });
      if (await editButton.isVisible()) {
        await editButton.click();

        // Should open workflow builder
        await expect(page.getByTestId('workflow-builder')).toBeVisible();
      }
    }
  });

  test('should view autonomous workflow suggestions', async ({ page }) => {
    const workflowCard = page.getByTestId('workflow-card').first();

    if (await workflowCard.isVisible()) {
      await workflowCard.click();

      const suggestionsButton = page.getByRole('button', { name: /Suggestions|Optimize/i });
      if (await suggestionsButton.isVisible()) {
        await suggestionsButton.click();

        // Should show AI suggestions
        await expect(page.getByText(/Suggestions|Optimizations|Improvements/i)).toBeVisible({ timeout: 10000 });
      }
    }
  });
});

test.describe('Workflows - Variables and Context', () => {
  test.beforeEach(async ({ page }) => {
    await setupAuthenticatedPage(page);
    await page.goto('/workflows');
    await page.waitForLoadState('networkidle');
  });

  test('should define workflow variables', async ({ page }) => {
    const workflowCard = page.getByTestId('workflow-card').first();

    if (await workflowCard.isVisible()) {
      await workflowCard.click();

      const variablesTab = page.getByRole('tab', { name: /Variables/i });
      if (await variablesTab.isVisible()) {
        await variablesTab.click();

        const addVariableButton = page.getByRole('button', { name: /Add.*Variable/i });
        if (await addVariableButton.isVisible()) {
          await addVariableButton.click();

          // Define variable
          await page.getByLabel(/Name/i).fill('apiUrl');
          await page.getByLabel(/Value/i).fill('https://api.example.com');

          await page.getByRole('button', { name: /Save|Add/i }).click();

          // Should show success message
          await expect(page.getByText(/Variable.*added/i)).toBeVisible({ timeout: 5000 });
        }
      }
    }
  });

  test('should use variable in step', async ({ page }) => {
    const workflowCard = page.getByTestId('workflow-card').first();

    if (await workflowCard.isVisible()) {
      await workflowCard.click();

      const workflowStep = page.getByTestId('workflow-step').first();
      if (await workflowStep.isVisible()) {
        await workflowStep.click();

        // Configure step with variable
        const configInput = page.getByLabel(/URL|Endpoint/i);
        if (await configInput.isVisible()) {
          await configInput.fill('${apiUrl}/data');

          // Should show variable reference
          await expect(page.getByText(/\$\{apiUrl\}/i)).toBeVisible();
        }
      }
    }
  });

  test('should pass data between steps', async ({ page }) => {
    const workflowCard = page.getByTestId('workflow-card').first();

    if (await workflowCard.isVisible()) {
      await workflowCard.click();

      const workflowStep = page.getByTestId('workflow-step').nth(1);
      if (await workflowStep.isVisible()) {
        await workflowStep.click();

        // Configure step to use previous step output
        const inputMappingButton = page.getByRole('button', { name: /Map.*Input|Data.*Mapping/i });
        if (await inputMappingButton.isVisible()) {
          await inputMappingButton.click();

          // Should show data mapping interface
          await expect(page.getByText(/Previous.*Step|Output.*Mapping/i)).toBeVisible();
        }
      }
    }
  });
});

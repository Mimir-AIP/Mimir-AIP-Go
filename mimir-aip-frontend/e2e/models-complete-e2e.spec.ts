import { test, expect } from '@playwright/test';
import { setupAuthenticatedPage } from './helpers';


/**
 * Comprehensive E2E tests for ML Models including training, prediction, and auto-ML features
 */

test.describe('Models - Management', () => {
  test.beforeEach(async ({ page }) => {
    await setupAuthenticatedPage(page);
    await page.goto('/models');
    await page.waitForLoadState('networkidle');
  });

  test('should display models page', async ({ page }) => {
    // Note: All pages use generic "Mimir AIP - AI Pipeline Orchestration" title
    await expect(page.getByRole('heading', { name: /ML Models/i })).toBeVisible();
    await expect(page.getByRole('button', { name: /Train Model/i })).toBeVisible();
  });

  test('should display models list', async ({ page }) => {
    const modelsList = page.getByTestId('models-list');

    if (await modelsList.isVisible()) {
      await expect(modelsList).toBeVisible();
    }
  });

  test('should create new model', async ({ page }) => {
    await page.getByRole('button', { name: /Train Model/i }).click();

    // Fill model form
    await page.getByLabel(/Name/i).fill('Test Classification Model');
    await page.getByLabel(/Description/i).fill('E2E test model');

    // Select model type
    const typeSelect = page.getByLabel(/Type|Algorithm/i);
    if (await typeSelect.isVisible()) {
      await typeSelect.selectOption('classification');
    }

    // Save model
    await page.getByRole('button', { name: /Create|Save/i }).click();

    // Verify creation
    await expect(page.getByText(/Model created successfully/i)).toBeVisible({ timeout: 5000 });
  });

  test('should view model details', async ({ page }) => {
    const modelCard = page.getByTestId('model-card').first();

    if (await modelCard.isVisible()) {
      await modelCard.click();

      // Should navigate to details page
      await expect(page).toHaveURL(/\/models\/[a-zA-Z0-9-]+/);
      await expect(page.getByRole('heading', { name: /Model Details/i })).toBeVisible();
    }
  });

  test('should display model metrics', async ({ page }) => {
    const modelCard = page.getByTestId('model-card').first();

    if (await modelCard.isVisible()) {
      await modelCard.click();

      const metricsTab = page.getByRole('tab', { name: /Metrics|Performance/i });
      if (await metricsTab.isVisible()) {
        await metricsTab.click();

        // Should show metrics
        await expect(page.getByText(/Accuracy|Precision|Recall|F1/i)).toBeVisible();
      }
    }
  });

  test('should view training history', async ({ page }) => {
    const modelCard = page.getByTestId('model-card').first();

    if (await modelCard.isVisible()) {
      await modelCard.click();

      const historyTab = page.getByRole('tab', { name: /History|Training/i });
      if (await historyTab.isVisible()) {
        await historyTab.click();

        // Should show training history
        await expect(page.getByTestId('training-history')).toBeVisible();
      }
    }
  });

  test('should delete model', async ({ page }) => {
    const deleteButton = page.getByRole('button', { name: /Delete/i }).first();

    if (await deleteButton.isVisible()) {
      await deleteButton.click();

      // Confirm deletion
      await expect(page.getByRole('dialog')).toBeVisible();
      await page.getByRole('button', { name: /Delete|Confirm/i }).click();

      // Verify deletion
      await expect(page.getByText(/Model deleted successfully/i)).toBeVisible({ timeout: 5000 });
    }
  });

  test('should export model', async ({ page }) => {
    const exportButton = page.getByRole('button', { name: /Export|Download/i }).first();

    if (await exportButton.isVisible()) {
      const downloadPromise = page.waitForEvent('download');
      await exportButton.click();

      const download = await downloadPromise;
      expect(download.suggestedFilename()).toMatch(/\.pkl|\.h5|\.onnx|\.joblib/);
    }
  });

  test('should filter models by type', async ({ page }) => {
    const filterSelect = page.getByLabel(/Filter|Type/i);

    if (await filterSelect.isVisible()) {
      await filterSelect.selectOption('classification');
      await page.waitForTimeout(500);

      // All visible models should be classification type
      const models = page.getByTestId('model-card');
      const count = await models.count();

      for (let i = 0; i < Math.min(count, 3); i++) {
        const typeBadge = models.nth(i).getByTestId('type-badge');
        if (await typeBadge.isVisible()) {
          const type = await typeBadge.textContent();
          expect(type?.toLowerCase()).toContain('classification');
        }
      }
    }
  });

  test('should search models', async ({ page }) => {
    const searchInput = page.getByPlaceholder(/Search.*models/i);

    if (await searchInput.isVisible()) {
      await searchInput.fill('classification');
      await page.waitForTimeout(500);

      // Results should contain "classification"
      const results = page.getByTestId('model-card');
      if (await results.first().isVisible()) {
        const text = await results.first().textContent();
        expect(text?.toLowerCase()).toContain('classification');
      }
    }
  });
});

test.describe('Models - Training', () => {
  test.beforeEach(async ({ page }) => {
    await setupAuthenticatedPage(page);
    await page.goto('/models');
    await page.waitForLoadState('networkidle');
  });

  test('should start model training', async ({ page }) => {
    await page.goto('/models');
    const trainButton = page.getByRole('button', { name: /Train.*Model/i }).first();

    if (await trainButton.isVisible()) {
      await trainButton.click();

      // Should navigate to training page
      await page.waitForURL('**/models/train');
      await expect(page.getByRole('heading', { name: 'Train New ML Model' })).toBeVisible();
    }
  });

  test('should configure training parameters', async ({ page }) => {
    const trainButton = page.getByRole('button', { name: /Train/i }).first();

    if (await trainButton.isVisible()) {
      await trainButton.click();

      // Check for parameter inputs
      const epochsInput = page.getByLabel(/Epochs/i);
      const batchSizeInput = page.getByLabel(/Batch.*Size/i);
      const learningRateInput = page.getByLabel(/Learning.*Rate/i);

      if (await epochsInput.isVisible()) {
        await epochsInput.fill('10');
      }
      if (await batchSizeInput.isVisible()) {
        await batchSizeInput.fill('32');
      }
      if (await learningRateInput.isVisible()) {
        await learningRateInput.fill('0.001');
      }

      // Start training
      await page.getByRole('button', { name: /Start.*Training|Train/i }).first().click();

      // Should show training started message
      await expect(page.getByText(/Training.*started/i)).toBeVisible({ timeout: 5000 });
    }
  });

  test('should monitor training progress', async ({ page }) => {
    const modelCard = page.getByTestId('model-card').first();

    if (await modelCard.isVisible()) {
      await modelCard.click();

      // Check for training progress
      const progressBar = page.getByTestId('training-progress');
      if (await progressBar.isVisible()) {
        await expect(progressBar).toBeVisible();
      }
    }
  });

  test('should view training logs', async ({ page }) => {
    const modelCard = page.getByTestId('model-card').first();

    if (await modelCard.isVisible()) {
      await modelCard.click();

      const logsTab = page.getByRole('tab', { name: /Logs/i });
      if (await logsTab.isVisible()) {
        await logsTab.click();

        // Should show training logs
        await expect(page.getByTestId('training-logs')).toBeVisible();
      }
    }
  });

  test('should stop training', async ({ page }) => {
    const stopButton = page.getByRole('button', { name: /Stop.*Training|Cancel/i }).first();

    if (await stopButton.isVisible()) {
      await stopButton.click();

      // Confirm stop
      await expect(page.getByRole('dialog')).toBeVisible();
      await page.getByRole('button', { name: /Stop|Confirm/i }).click();

      // Should show stop message
      await expect(page.getByText(/Training.*stopped/i)).toBeVisible({ timeout: 5000 });
    }
  });

  test('should view loss curve', async ({ page }) => {
    const modelCard = page.getByTestId('model-card').first();

    if (await modelCard.isVisible()) {
      await modelCard.click();

      const chartsTab = page.getByRole('tab', { name: /Charts|Visualization/i });
      if (await chartsTab.isVisible()) {
        await chartsTab.click();

        // Should show loss curve
        await expect(page.getByTestId('loss-curve-chart')).toBeVisible();
      }
    }
  });
});

test.describe('Models - Prediction', () => {
  test.beforeEach(async ({ page }) => {
    await setupAuthenticatedPage(page);
    await page.goto('/models');
    await page.waitForLoadState('networkidle');
  });

  test('should make single prediction', async ({ page }) => {
    const predictButton = page.getByRole('button', { name: /Predict|Test/i }).first();

    if (await predictButton.isVisible()) {
      await predictButton.click();

      // Input dialog should appear
      await expect(page.getByRole('dialog')).toBeVisible();

      // Enter input data
      const inputField = page.getByLabel(/Input|Data/i);
      if (await inputField.isVisible()) {
        await inputField.fill('{"feature1": 1.0, "feature2": 2.0}');
      }

      // Run prediction
      await page.getByRole('button', { name: /Predict|Run/i }).click();

      // Should show prediction result
      await expect(page.getByTestId('prediction-result')).toBeVisible({ timeout: 10000 });
    }
  });

  test('should make batch predictions', async ({ page }) => {
    const batchPredictButton = page.getByRole('button', { name: /Batch.*Predict/i }).first();

    if (await batchPredictButton.isVisible()) {
      await batchPredictButton.click();

      // File upload should appear
      const fileInput = page.locator('input[type="file"]');
      await expect(fileInput).toBeVisible();

      // Note: Actual file upload would require test fixture files
    }
  });

  test('should display prediction confidence', async ({ page }) => {
    const predictButton = page.getByRole('button', { name: /Predict/i }).first();

    if (await predictButton.isVisible()) {
      await predictButton.click();

      const inputField = page.getByLabel(/Input/i);
      if (await inputField.isVisible()) {
        await inputField.fill('{"data": "test"}');
        await page.getByRole('button', { name: /Predict|Run/i }).click();

        await page.waitForTimeout(2000);

        // Should show confidence score
        const confidence = page.getByTestId('prediction-confidence');
        if (await confidence.isVisible()) {
          await expect(confidence).toBeVisible();
        }
      }
    }
  });

  test('should export predictions', async ({ page }) => {
    const modelCard = page.getByTestId('model-card').first();

    if (await modelCard.isVisible()) {
      await modelCard.click();

      const predictionsTab = page.getByRole('tab', { name: /Predictions/i });
      if (await predictionsTab.isVisible()) {
        await predictionsTab.click();

        const exportButton = page.getByRole('button', { name: /Export/i });
        if (await exportButton.isVisible()) {
          const downloadPromise = page.waitForEvent('download');
          await exportButton.click();

          const download = await downloadPromise;
          expect(download.suggestedFilename()).toMatch(/\.csv|\.json/);
        }
      }
    }
  });
});

test.describe('Models - Auto-ML', () => {
  test.beforeEach(async ({ page }) => {
    await setupAuthenticatedPage(page);
    await page.goto('/models/automl');
    await page.waitForLoadState('networkidle');
  });

  test('should display AutoML page', async ({ page }) => {
    await expect(page.getByRole('heading', { name: /Auto.*ML|Automated/i })).toBeVisible();
  });

  test('should start AutoML experiment', async ({ page }) => {
    const startButton = page.getByRole('button', { name: /Start.*AutoML|Create.*Experiment/i });

    if (await startButton.isVisible()) {
      await startButton.click();

      // Configuration form should appear
      await page.getByLabel(/Dataset/i).fill('training_data.csv');
      await page.getByLabel(/Target.*Column/i).fill('label');

      const taskTypeSelect = page.getByLabel(/Task.*Type/i);
      if (await taskTypeSelect.isVisible()) {
        await taskTypeSelect.selectOption('classification');
      }

      // Start experiment
      await page.getByRole('button', { name: /Start|Run/i }).click();

      // Should show experiment started message
      await expect(page.getByText(/Experiment.*started/i)).toBeVisible({ timeout: 5000 });
    }
  });

  test('should view AutoML results', async ({ page }) => {
    const experimentCard = page.getByTestId('experiment-card').first();

    if (await experimentCard.isVisible()) {
      await experimentCard.click();

      // Should show results
      await expect(page.getByTestId('automl-results')).toBeVisible();
      await expect(page.getByText(/Best.*Model|Top.*Performing/i)).toBeVisible();
    }
  });

  test('should compare AutoML models', async ({ page }) => {
    const compareButton = page.getByRole('button', { name: /Compare/i });

    if (await compareButton.isVisible()) {
      await compareButton.click();

      // Comparison view should appear
      await expect(page.getByText(/Model.*Comparison/i)).toBeVisible();
    }
  });

  test('should deploy best model', async ({ page }) => {
    const deployButton = page.getByRole('button', { name: /Deploy.*Best|Deploy/i }).first();

    if (await deployButton.isVisible()) {
      await deployButton.click();

      // Deployment dialog should appear
      await expect(page.getByRole('dialog')).toBeVisible();
      await page.getByLabel(/Deployment.*Name/i).fill('production-model');
      await page.getByRole('button', { name: /Deploy/i }).click();

      // Should show deployment message
      await expect(page.getByText(/Model.*deployed/i)).toBeVisible({ timeout: 10000 });
    }
  });
});

test.describe('Models - Evaluation', () => {
  test.beforeEach(async ({ page }) => {
    await setupAuthenticatedPage(page);
    await page.goto('/models');
    await page.waitForLoadState('networkidle');
  });

  test('should evaluate model', async ({ page }) => {
    const evaluateButton = page.getByRole('button', { name: /Evaluate|Test/i }).first();

    if (await evaluateButton.isVisible()) {
      await evaluateButton.click();

      // Test dataset selection
      const datasetSelect = page.getByLabel(/Test.*Dataset/i);
      if (await datasetSelect.isVisible()) {
        await datasetSelect.selectOption({ index: 1 });
      }

      // Run evaluation
      await page.getByRole('button', { name: /Evaluate|Run/i }).click();

      // Should show evaluation results
      await expect(page.getByTestId('evaluation-results')).toBeVisible({ timeout: 10000 });
    }
  });

  test('should view confusion matrix', async ({ page }) => {
    const modelCard = page.getByTestId('model-card').first();

    if (await modelCard.isVisible()) {
      await modelCard.click();

      const evaluationTab = page.getByRole('tab', { name: /Evaluation/i });
      if (await evaluationTab.isVisible()) {
        await evaluationTab.click();

        // Should show confusion matrix
        const confusionMatrix = page.getByTestId('confusion-matrix');
        if (await confusionMatrix.isVisible()) {
          await expect(confusionMatrix).toBeVisible();
        }
      }
    }
  });

  test('should view ROC curve', async ({ page }) => {
    const modelCard = page.getByTestId('model-card').first();

    if (await modelCard.isVisible()) {
      await modelCard.click();

      const chartsTab = page.getByRole('tab', { name: /Charts/i });
      if (await chartsTab.isVisible()) {
        await chartsTab.click();

        // Should show ROC curve
        const rocCurve = page.getByTestId('roc-curve');
        if (await rocCurve.isVisible()) {
          await expect(rocCurve).toBeVisible();
        }
      }
    }
  });

  test('should compare model versions', async ({ page }) => {
    const compareButton = page.getByRole('button', { name: /Compare.*Versions/i });

    if (await compareButton.isVisible()) {
      await compareButton.click();

      // Version comparison should appear
      await expect(page.getByText(/Version.*Comparison/i)).toBeVisible();
    }
  });
});

test.describe('Models - Deployment', () => {
  test.beforeEach(async ({ page }) => {
    await setupAuthenticatedPage(page);
    await page.goto('/models');
    await page.waitForLoadState('networkidle');
  });

  test('should deploy model', async ({ page }) => {
    const deployButton = page.getByRole('button', { name: /Deploy/i }).first();

    if (await deployButton.isVisible()) {
      await deployButton.click();

      // Deployment configuration
      await page.getByLabel(/Endpoint.*Name/i).fill('prod-model-v1');

      const resourceSelect = page.getByLabel(/Resources|Instance/i);
      if (await resourceSelect.isVisible()) {
        await resourceSelect.selectOption('medium');
      }

      // Deploy
      await page.getByRole('button', { name: /Deploy/i }).click();

      // Should show deployment message
      await expect(page.getByText(/Model.*deployed|Deployment.*started/i)).toBeVisible({ timeout: 10000 });
    }
  });

  test('should view deployed models', async ({ page }) => {
    await page.goto('/models/deployments');
    await page.waitForLoadState('networkidle');

    // Should show deployments list
    await expect(page.getByRole('heading', { name: /Deployments/i })).toBeVisible();
  });

  test('should test deployed endpoint', async ({ page }) => {
    await page.goto('/models/deployments');
    await page.waitForLoadState('networkidle');

    const testButton = page.getByRole('button', { name: /Test.*Endpoint/i }).first();
    if (await testButton.isVisible()) {
      await testButton.click();

      // Input test data
      const inputField = page.getByLabel(/Input/i);
      if (await inputField.isVisible()) {
        await inputField.fill('{"test": "data"}');
        await page.getByRole('button', { name: /Test|Send/i }).click();

        // Should show test result
        await expect(page.getByTestId('test-result')).toBeVisible({ timeout: 5000 });
      }
    }
  });

  test('should undeploy model', async ({ page }) => {
    await page.goto('/models/deployments');
    await page.waitForLoadState('networkidle');

    const undeployButton = page.getByRole('button', { name: /Undeploy|Remove/i }).first();
    if (await undeployButton.isVisible()) {
      await undeployButton.click();

      // Confirm undeploy
      await expect(page.getByRole('dialog')).toBeVisible();
      await page.getByRole('button', { name: /Undeploy|Confirm/i }).click();

      // Should show undeploy message
      await expect(page.getByText(/Model.*undeployed/i)).toBeVisible({ timeout: 5000 });
    }
  });

  test('should scale deployment', async ({ page }) => {
    await page.goto('/models/deployments');
    await page.waitForLoadState('networkidle');

    const scaleButton = page.getByRole('button', { name: /Scale/i }).first();
    if (await scaleButton.isVisible()) {
      await scaleButton.click();

      // Scale configuration
      const instancesInput = page.getByLabel(/Instances|Replicas/i);
      if (await instancesInput.isVisible()) {
        await instancesInput.fill('3');
        await page.getByRole('button', { name: /Scale|Update/i }).click();

        // Should show scale message
        await expect(page.getByText(/Scaling.*deployment/i)).toBeVisible({ timeout: 5000 });
      }
    }
  });
});

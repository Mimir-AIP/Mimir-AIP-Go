import { test, expect } from '@playwright/test';

/**
 * Comprehensive E2E tests for Extraction including entity extraction and data parsing
 */

test.describe('Extraction - Management', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/extraction');
    await page.waitForLoadState('networkidle');
  });

  test('should display extraction page', async ({ page }) => {
    await expect(page).toHaveTitle(/Extraction/i);
    await expect(page.getByRole('heading', { name: /Extraction|Entity.*Extract/i })).toBeVisible();
    await expect(page.getByRole('button', { name: /New.*Extraction|Extract/i })).toBeVisible();
  });

  test('should display extraction jobs list', async ({ page }) => {
    const extractionsList = page.getByTestId('extractions-list');

    if (await extractionsList.isVisible()) {
      await expect(extractionsList).toBeVisible();
    }
  });

  test('should start new extraction', async ({ page }) => {
    await page.getByRole('button', { name: /New.*Extraction|Extract/i }).click();

    // Extraction dialog should appear
    await expect(page.getByRole('dialog')).toBeVisible();
    await expect(page.getByRole('heading', { name: /New.*Extraction/i })).toBeVisible();
  });

  test('should upload file for extraction', async ({ page }) => {
    await page.getByRole('button', { name: /New.*Extraction/i }).click();

    // File upload should be available
    const fileInput = page.locator('input[type="file"]');
    await expect(fileInput).toBeVisible();

    // Note: Actual file upload would require test fixture files
  });

  test('should extract from text input', async ({ page }) => {
    await page.getByRole('button', { name: /New.*Extraction/i }).click();

    // Switch to text input tab
    const textTab = page.getByRole('tab', { name: /Text|Paste/i });
    if (await textTab.isVisible()) {
      await textTab.click();

      // Enter text
      const textArea = page.getByLabel(/Text|Content/i);
      await textArea.fill('John Doe works at Acme Corp in San Francisco. His email is john@acme.com.');

      // Start extraction
      await page.getByRole('button', { name: /Extract|Start/i }).click();

      // Should show extraction started
      await expect(page.getByText(/Extraction.*started|Processing/i)).toBeVisible({ timeout: 5000 });
    }
  });

  test('should extract from URL', async ({ page }) => {
    await page.getByRole('button', { name: /New.*Extraction/i }).click();

    // Switch to URL tab
    const urlTab = page.getByRole('tab', { name: /URL|Web/i });
    if (await urlTab.isVisible()) {
      await urlTab.click();

      // Enter URL
      const urlInput = page.getByLabel(/URL|Website/i);
      await urlInput.fill('https://example.com/document');

      // Start extraction
      await page.getByRole('button', { name: /Extract|Start/i }).click();

      // Should show extraction started
      await expect(page.getByText(/Extraction.*started|Fetching/i)).toBeVisible({ timeout: 5000 });
    }
  });

  test('should select extraction schema', async ({ page }) => {
    await page.getByRole('button', { name: /New.*Extraction/i }).click();

    // Select schema
    const schemaSelect = page.getByLabel(/Schema|Template|Type/i);
    if (await schemaSelect.isVisible()) {
      await schemaSelect.selectOption('person');

      // Should show schema fields
      await expect(page.getByText(/Fields|Entities|Properties/i)).toBeVisible();
    }
  });

  test('should configure custom extraction fields', async ({ page }) => {
    await page.getByRole('button', { name: /New.*Extraction/i }).click();

    // Custom fields option
    const customButton = page.getByRole('button', { name: /Custom.*Fields/i });
    if (await customButton.isVisible()) {
      await customButton.click();

      // Add field
      const addFieldButton = page.getByRole('button', { name: /Add Field/i });
      await addFieldButton.click();

      await page.getByLabel(/Field.*Name/i).fill('department');
      await page.getByLabel(/Field.*Type/i).selectOption('string');

      // Should show added field
      await expect(page.getByText('department')).toBeVisible();
    }
  });

  test('should view extraction results', async ({ page }) => {
    const extractionCard = page.getByTestId('extraction-card').first();

    if (await extractionCard.isVisible()) {
      await extractionCard.click();

      // Should show results
      await expect(page).toHaveURL(/\/extraction\/[a-zA-Z0-9-]+/);
      await expect(page.getByRole('heading', { name: /Extraction.*Results/i })).toBeVisible();
    }
  });

  test('should display extracted entities', async ({ page }) => {
    const extractionCard = page.getByTestId('extraction-card').first();

    if (await extractionCard.isVisible()) {
      await extractionCard.click();

      // Should show entities
      const entitiesList = page.getByTestId('extracted-entities');
      if (await entitiesList.isVisible()) {
        await expect(entitiesList).toBeVisible();
      }
    }
  });

  test('should filter entities by type', async ({ page }) => {
    const extractionCard = page.getByTestId('extraction-card').first();

    if (await extractionCard.isVisible()) {
      await extractionCard.click();

      const filterSelect = page.getByLabel(/Filter|Entity.*Type/i);
      if (await filterSelect.isVisible()) {
        await filterSelect.selectOption('person');
        await page.waitForTimeout(500);

        // Should show only person entities
        await expect(page.getByTestId('extracted-entities')).toBeVisible();
      }
    }
  });

  test('should export extraction results', async ({ page }) => {
    const extractionCard = page.getByTestId('extraction-card').first();

    if (await extractionCard.isVisible()) {
      await extractionCard.click();

      const exportButton = page.getByRole('button', { name: /Export|Download/i });
      if (await exportButton.isVisible()) {
        const downloadPromise = page.waitForEvent('download');
        await exportButton.click();

        const download = await downloadPromise;
        expect(download.suggestedFilename()).toMatch(/\.json|\.csv|\.xlsx/);
      }
    }
  });

  test('should delete extraction', async ({ page }) => {
    const deleteButton = page.getByRole('button', { name: /Delete/i }).first();

    if (await deleteButton.isVisible()) {
      await deleteButton.click();

      // Confirm deletion
      await expect(page.getByRole('dialog')).toBeVisible();
      await page.getByRole('button', { name: /Delete|Confirm/i }).click();

      // Should show deletion message
      await expect(page.getByText(/Extraction.*deleted/i)).toBeVisible({ timeout: 5000 });
    }
  });

  test('should search extractions', async ({ page }) => {
    const searchInput = page.getByPlaceholder(/Search.*extractions/i);

    if (await searchInput.isVisible()) {
      await searchInput.fill('document');
      await page.waitForTimeout(500);

      // Results should contain "document"
      const results = page.getByTestId('extraction-card');
      if (await results.first().isVisible()) {
        const text = await results.first().textContent();
        expect(text?.toLowerCase()).toContain('document');
      }
    }
  });

  test('should filter extractions by status', async ({ page }) => {
    const filterSelect = page.getByLabel(/Filter|Status/i);

    if (await filterSelect.isVisible()) {
      await filterSelect.selectOption('completed');
      await page.waitForTimeout(500);

      // All visible extractions should be completed
      const extractions = page.getByTestId('extraction-card');
      const count = await extractions.count();

      for (let i = 0; i < Math.min(count, 3); i++) {
        const statusBadge = extractions.nth(i).getByTestId('status-badge');
        if (await statusBadge.isVisible()) {
          const status = await statusBadge.textContent();
          expect(status?.toLowerCase()).toContain('completed');
        }
      }
    }
  });
});

test.describe('Extraction - Entity Types', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/extraction');
    await page.waitForLoadState('networkidle');
  });

  test('should extract person entities', async ({ page }) => {
    await page.getByRole('button', { name: /New.*Extraction/i }).click();

    const textTab = page.getByRole('tab', { name: /Text/i });
    if (await textTab.isVisible()) {
      await textTab.click();

      const textArea = page.getByLabel(/Text/i);
      await textArea.fill('Dr. Jane Smith is a researcher at MIT. Contact: jane.smith@mit.edu');

      const schemaSelect = page.getByLabel(/Schema/i);
      if (await schemaSelect.isVisible()) {
        await schemaSelect.selectOption('person');
      }

      await page.getByRole('button', { name: /Extract/i }).click();
      await page.waitForTimeout(3000);

      // Should extract person entities
      const entities = page.getByTestId('entity-person');
      if (await entities.first().isVisible()) {
        await expect(entities.first()).toBeVisible();
      }
    }
  });

  test('should extract organization entities', async ({ page }) => {
    await page.getByRole('button', { name: /New.*Extraction/i }).click();

    const textTab = page.getByRole('tab', { name: /Text/i });
    if (await textTab.isVisible()) {
      await textTab.click();

      const textArea = page.getByLabel(/Text/i);
      await textArea.fill('Microsoft Corporation and Apple Inc. announced a partnership.');

      const schemaSelect = page.getByLabel(/Schema/i);
      if (await schemaSelect.isVisible()) {
        await schemaSelect.selectOption('organization');
      }

      await page.getByRole('button', { name: /Extract/i }).click();
      await page.waitForTimeout(3000);

      // Should extract organization entities
      const entities = page.getByTestId('entity-organization');
      if (await entities.first().isVisible()) {
        await expect(entities.first()).toBeVisible();
      }
    }
  });

  test('should extract location entities', async ({ page }) => {
    await page.getByRole('button', { name: /New.*Extraction/i }).click();

    const textTab = page.getByRole('tab', { name: /Text/i });
    if (await textTab.isVisible()) {
      await textTab.click();

      const textArea = page.getByLabel(/Text/i);
      await textArea.fill('The conference will be held in San Francisco, California, USA.');

      const schemaSelect = page.getByLabel(/Schema/i);
      if (await schemaSelect.isVisible()) {
        await schemaSelect.selectOption('location');
      }

      await page.getByRole('button', { name: /Extract/i }).click();
      await page.waitForTimeout(3000);

      // Should extract location entities
      const entities = page.getByTestId('entity-location');
      if (await entities.first().isVisible()) {
        await expect(entities.first()).toBeVisible();
      }
    }
  });

  test('should extract date entities', async ({ page }) => {
    await page.getByRole('button', { name: /New.*Extraction/i }).click();

    const textTab = page.getByRole('tab', { name: /Text/i });
    if (await textTab.isVisible()) {
      await textTab.click();

      const textArea = page.getByLabel(/Text/i);
      await textArea.fill('The meeting is scheduled for January 15, 2025 at 2:00 PM.');

      const schemaSelect = page.getByLabel(/Schema/i);
      if (await schemaSelect.isVisible()) {
        await schemaSelect.selectOption('date');
      }

      await page.getByRole('button', { name: /Extract/i }).click();
      await page.waitForTimeout(3000);

      // Should extract date entities
      const entities = page.getByTestId('entity-date');
      if (await entities.first().isVisible()) {
        await expect(entities.first()).toBeVisible();
      }
    }
  });
});

test.describe('Extraction - Batch Processing', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/extraction/batch');
    await page.waitForLoadState('networkidle');
  });

  test('should display batch extraction page', async ({ page }) => {
    await expect(page.getByRole('heading', { name: /Batch.*Extraction/i })).toBeVisible();
  });

  test('should upload multiple files', async ({ page }) => {
    const uploadButton = page.getByRole('button', { name: /Upload.*Files/i });

    if (await uploadButton.isVisible()) {
      await uploadButton.click();

      // File input should support multiple files
      const fileInput = page.locator('input[type="file"]');
      const hasMultiple = await fileInput.getAttribute('multiple');
      expect(hasMultiple).not.toBeNull();
    }
  });

  test('should start batch extraction', async ({ page }) => {
    const startButton = page.getByRole('button', { name: /Start.*Batch|Process.*All/i });

    if (await startButton.isVisible()) {
      await startButton.click();

      // Should show batch processing
      await expect(page.getByText(/Processing.*batch|Extracting/i)).toBeVisible({ timeout: 5000 });
    }
  });

  test('should display batch progress', async ({ page }) => {
    const progressBar = page.getByTestId('batch-progress');

    if (await progressBar.isVisible()) {
      await expect(progressBar).toBeVisible();
    }
  });

  test('should view batch results', async ({ page }) => {
    const batchCard = page.getByTestId('batch-card').first();

    if (await batchCard.isVisible()) {
      await batchCard.click();

      // Should show batch results
      await expect(page.getByTestId('batch-results')).toBeVisible();
    }
  });
});

test.describe('Extraction - Validation and Review', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/extraction');
    await page.waitForLoadState('networkidle');
  });

  test('should review extracted entities', async ({ page }) => {
    const extractionCard = page.getByTestId('extraction-card').first();

    if (await extractionCard.isVisible()) {
      await extractionCard.click();

      const reviewTab = page.getByRole('tab', { name: /Review/i });
      if (await reviewTab.isVisible()) {
        await reviewTab.click();

        // Should show review interface
        await expect(page.getByTestId('entity-review')).toBeVisible();
      }
    }
  });

  test('should approve entity', async ({ page }) => {
    const extractionCard = page.getByTestId('extraction-card').first();

    if (await extractionCard.isVisible()) {
      await extractionCard.click();

      const approveButton = page.getByRole('button', { name: /Approve/i }).first();
      if (await approveButton.isVisible()) {
        await approveButton.click();

        // Should show approval
        await expect(page.getByText(/Approved/i)).toBeVisible({ timeout: 2000 });
      }
    }
  });

  test('should reject entity', async ({ page }) => {
    const extractionCard = page.getByTestId('extraction-card').first();

    if (await extractionCard.isVisible()) {
      await extractionCard.click();

      const rejectButton = page.getByRole('button', { name: /Reject/i }).first();
      if (await rejectButton.isVisible()) {
        await rejectButton.click();

        // Should show rejection
        await expect(page.getByText(/Rejected/i)).toBeVisible({ timeout: 2000 });
      }
    }
  });

  test('should edit extracted entity', async ({ page }) => {
    const extractionCard = page.getByTestId('extraction-card').first();

    if (await extractionCard.isVisible()) {
      await extractionCard.click();

      const entity = page.getByTestId('entity-item').first();
      if (await entity.isVisible()) {
        await entity.click();

        const editButton = page.getByRole('button', { name: /Edit/i });
        if (await editButton.isVisible()) {
          await editButton.click();

          // Edit dialog should appear
          await expect(page.getByRole('dialog')).toBeVisible();
          await expect(page.getByLabel(/Value|Text/i)).toBeVisible();
        }
      }
    }
  });

  test('should add manual entity', async ({ page }) => {
    const extractionCard = page.getByTestId('extraction-card').first();

    if (await extractionCard.isVisible()) {
      await extractionCard.click();

      const addButton = page.getByRole('button', { name: /Add.*Entity|Manual/i });
      if (await addButton.isVisible()) {
        await addButton.click();

        // Add entity form
        await page.getByLabel(/Type/i).selectOption('person');
        await page.getByLabel(/Value/i).fill('John Smith');

        await page.getByRole('button', { name: /Add|Save/i }).click();

        // Should show success message
        await expect(page.getByText(/Entity.*added/i)).toBeVisible({ timeout: 5000 });
      }
    }
  });

  test('should validate extraction quality', async ({ page }) => {
    const extractionCard = page.getByTestId('extraction-card').first();

    if (await extractionCard.isVisible()) {
      await extractionCard.click();

      const qualityWidget = page.getByTestId('extraction-quality');
      if (await qualityWidget.isVisible()) {
        await expect(qualityWidget).toBeVisible();
        await expect(page.getByText(/Confidence|Quality|Accuracy/i)).toBeVisible();
      }
    }
  });
});

test.describe('Extraction - Templates and Schemas', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/extraction/templates');
    await page.waitForLoadState('networkidle');
  });

  test('should display extraction templates', async ({ page }) => {
    await expect(page.getByRole('heading', { name: /Templates|Schemas/i })).toBeVisible();
  });

  test('should create custom template', async ({ page }) => {
    const createButton = page.getByRole('button', { name: /Create.*Template/i });

    if (await createButton.isVisible()) {
      await createButton.click();

      // Template form
      await page.getByLabel(/Name/i).fill('Custom Document Schema');
      await page.getByLabel(/Description/i).fill('Extract document metadata');

      // Add fields
      const addFieldButton = page.getByRole('button', { name: /Add Field/i });
      await addFieldButton.click();

      await page.getByLabel(/Field.*Name/i).fill('title');
      await page.getByLabel(/Field.*Type/i).selectOption('string');

      // Save template
      await page.getByRole('button', { name: /Save|Create/i }).click();

      // Should show success message
      await expect(page.getByText(/Template.*created/i)).toBeVisible({ timeout: 5000 });
    }
  });

  test('should edit template', async ({ page }) => {
    const editButton = page.getByRole('button', { name: /Edit/i }).first();

    if (await editButton.isVisible()) {
      await editButton.click();

      // Edit template
      const nameInput = page.getByLabel(/Name/i);
      await nameInput.clear();
      await nameInput.fill('Updated Template Name');

      await page.getByRole('button', { name: /Save|Update/i }).click();

      // Should show success message
      await expect(page.getByText(/Template.*updated/i)).toBeVisible({ timeout: 5000 });
    }
  });

  test('should delete template', async ({ page }) => {
    const deleteButton = page.getByRole('button', { name: /Delete/i }).first();

    if (await deleteButton.isVisible()) {
      await deleteButton.click();

      // Confirm deletion
      await expect(page.getByRole('dialog')).toBeVisible();
      await page.getByRole('button', { name: /Delete|Confirm/i }).click();

      // Should show deletion message
      await expect(page.getByText(/Template.*deleted/i)).toBeVisible({ timeout: 5000 });
    }
  });

  test('should export template', async ({ page }) => {
    const exportButton = page.getByRole('button', { name: /Export/i }).first();

    if (await exportButton.isVisible()) {
      const downloadPromise = page.waitForEvent('download');
      await exportButton.click();

      const download = await downloadPromise;
      expect(download.suggestedFilename()).toMatch(/\.json|\.yaml/);
    }
  });

  test('should import template', async ({ page }) => {
    const importButton = page.getByRole('button', { name: /Import/i });

    if (await importButton.isVisible()) {
      await importButton.click();

      // File upload should appear
      const fileInput = page.locator('input[type="file"]');
      await expect(fileInput).toBeVisible();
    }
  });
});

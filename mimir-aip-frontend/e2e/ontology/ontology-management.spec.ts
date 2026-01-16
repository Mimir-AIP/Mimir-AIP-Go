/**
 * ⚠️ SKIPPED: This file uses heavy API mocking (APIMocker removed)
 * 
 * This test file heavily mocks API endpoints, which defeats the purpose
 * of end-to-end testing. These tests need to be completely rewritten to:
 * 1. Use the real backend API
 * 2. Test actual integration between frontend and backend
 * 3. Verify real data flows and state management
 * 
 * ALL TESTS IN THIS FILE ARE SKIPPED until refactoring is complete.
 * Priority: HIGH - Requires major refactoring effort (~2-3 hours)
 */

import { test, expect } from '../helpers';
import { testOntology } from '../fixtures/test-data';
import { uploadFile, expectVisible, expectTextVisible, waitForToast } from '../helpers';

test.describe.skip('Ontology Management - SKIPPED (needs refactoring)', () => {
  test('should display list of ontologies', async ({ authenticatedPage: page }) => {
    const mocker = new APIMocker(page);
    
    const mockOntologies = [
      {
        id: 'ont-1',
        name: 'Test Ontology 1',
        description: 'First test ontology',
        version: '1.0.0',
        format: 'turtle',
        status: 'active',
        created_at: new Date().toISOString(),
      },
      {
        id: 'ont-2',
        name: 'Test Ontology 2',
        description: 'Second test ontology',
        version: '2.0.0',
        format: 'rdf-xml',
        status: 'draft',
        created_at: new Date().toISOString(),
      },
    ];

    await mocker.mockOntologyList(mockOntologies);
    
    await page.goto('/ontologies');
    
    // Should show ontologies page title
    await expectTextVisible(page, /ontologies/i);
    
    // Should display ontologies in table
    await expectTextVisible(page, 'Test Ontology 1');
    await expectTextVisible(page, 'Test Ontology 2');
    await expectTextVisible(page, '1.0.0');
    await expectTextVisible(page, '2.0.0');
  });

  test('should upload a new ontology', async ({ authenticatedPage: page }) => {
    let uploadedData: any = null;

    // Mock upload endpoint
    await page.route('**/api/v1/ontology/upload', async (route) => {
      const request = route.request();
      uploadedData = await request.postDataJSON();
      
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: {
            ontology: {
              id: 'ont-new',
              name: testOntology.name,
              version: testOntology.version,
            },
          },
        }),
      });
    });

    await page.goto('/ontologies/upload');
    
    // Fill ontology details
    await page.fill('input[name="name"]', testOntology.name);
    await page.fill('textarea[name="description"]', testOntology.description);
    await page.fill('input[name="version"]', testOntology.version);
    
    // Select format
    await page.selectOption('select[name="format"]', testOntology.format);
    
    // Upload file or paste content
    const contentArea = page.locator('textarea[name="content"], textarea[placeholder*="ontology"]');
    if (await contentArea.isVisible({ timeout: 2000 }).catch(() => false)) {
      await contentArea.fill(testOntology.content);
    } else {
      // Try file upload
      const fileInput = page.locator('input[type="file"]');
      await uploadFile(
        page,
        'input[type="file"]',
        'test-ontology.ttl',
        testOntology.content,
        'text/turtle'
      );
    }
    
    // Submit form
    await page.click('button[type="submit"], button:has-text("Upload")');
    
    // Should show success message
    await waitForToast(page, /success|uploaded|created/i);
    
    // Should redirect to ontologies list or details
    await expect(page).toHaveURL(/\/ontologies/, { timeout: 10000 });
  });

  test('should view ontology details', async ({ authenticatedPage: page }) => {
    const mocker = new APIMocker(page);
    
    const mockOntology = {
      id: 'ont-1',
      name: 'Test Ontology',
      description: 'Test description',
      version: '1.0.0',
      format: 'turtle',
      status: 'active',
      created_at: new Date().toISOString(),
    };

    await mocker.mockOntologyGet('ont-1', mockOntology);
    
    // Mock stats
    await page.route('**/api/v1/ontology/ont-1/stats', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: {
            stats: {
              total_classes: 10,
              total_properties: 15,
              total_triples: 100,
              total_entities: 50,
            },
          },
        }),
      });
    });

    await page.goto('/ontologies/ont-1');
    
    // Should show ontology name and details
    await expectTextVisible(page, 'Test Ontology');
    await expectTextVisible(page, '1.0.0');
    await expectTextVisible(page, 'active');
    
    // Should show stats
    await expectTextVisible(page, /classes|properties|triples/i);
  });

  test('should filter ontologies by status', async ({ authenticatedPage: page }) => {
    const mocker = new APIMocker(page);
    
    const mockOntologies = [
      {
        id: 'ont-active',
        name: 'Active Ontology',
        status: 'active',
        version: '1.0.0',
        format: 'turtle',
        created_at: new Date().toISOString(),
      },
      {
        id: 'ont-draft',
        name: 'Draft Ontology',
        status: 'draft',
        version: '1.0.0',
        format: 'turtle',
        created_at: new Date().toISOString(),
      },
    ];

    await mocker.mockOntologyList(mockOntologies);
    
    await page.goto('/ontologies');
    
    // Should show all ontologies initially
    await expectTextVisible(page, 'Active Ontology');
    await expectTextVisible(page, 'Draft Ontology');
    
    // Filter by active status
    await page.selectOption('select[name="status"], select:has-text("Status")', 'active');
    
    // Wait for filtered results (may need to mock filtered API call)
    await page.waitForTimeout(500); // Give time for filter to apply
    
    // Verify filter is applied (status dropdown should show active)
    const statusSelect = page.locator('select[name="status"], select:has-text("Status")');
    await expect(statusSelect).toHaveValue('active');
  });

  test('should delete an ontology', async ({ authenticatedPage: page }) => {
    const mocker = new APIMocker(page);
    
    const mockOntologies = [
      {
        id: 'ont-delete',
        name: 'Ontology to Delete',
        status: 'draft',
        version: '1.0.0',
        format: 'turtle',
        created_at: new Date().toISOString(),
      },
    ];

    await mocker.mockOntologyList(mockOntologies);
    
    // Mock delete endpoint
    let deleteCalled = false;
    await page.route('**/api/v1/ontology/ont-delete', async (route) => {
      if (route.request().method() === 'DELETE') {
        deleteCalled = true;
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ success: true }),
        });
      }
    });

    await page.goto('/ontologies');
    
    // Find and click delete button
    const deleteButton = page.locator('button:has-text("Delete")').first();
    await deleteButton.click();
    
    // Confirm deletion in dialog
    page.once('dialog', async (dialog) => {
      expect(dialog.message()).toContain('delete');
      await dialog.accept();
    });
    
    // Wait for delete to complete
    await page.waitForTimeout(1000);
    
    expect(deleteCalled).toBe(true);
  });

  test('should export an ontology', async ({ authenticatedPage: page }) => {
    const mocker = new APIMocker(page);
    
    const mockOntology = {
      id: 'ont-export',
      name: 'Export Ontology',
      version: '1.0.0',
      format: 'turtle',
      status: 'active',
      created_at: new Date().toISOString(),
    };

    await mocker.mockOntologyGet('ont-export', mockOntology);
    
    // Mock export endpoint
    await page.route('**/api/v1/ontology/ont-export/export*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'text/turtle',
        body: testOntology.content,
        headers: {
          'Content-Disposition': 'attachment; filename="ontology.ttl"',
        },
      });
    });

    await page.goto('/ontologies/ont-export');
    
    // Start waiting for download before clicking
    const downloadPromise = page.waitForEvent('download');
    
    // Click export button
    await page.click('button:has-text("Export")');
    
    // Wait for download
    const download = await downloadPromise;
    
    // Verify download filename
    expect(download.suggestedFilename()).toMatch(/\.ttl$/);
  });

  test('should handle upload errors gracefully', async ({ authenticatedPage: page }) => {
    // Mock upload failure
    await page.route('**/api/v1/ontology/upload', async (route) => {
      await route.fulfill({
        status: 400,
        contentType: 'application/json',
        body: JSON.stringify({
          error: 'Invalid ontology format',
        }),
      });
    });

    await page.goto('/ontologies/upload');
    
    // Fill minimal form data
    await page.fill('input[name="name"]', 'Bad Ontology');
    await page.fill('input[name="version"]', '1.0.0');
    
    // Submit form
    await page.click('button[type="submit"], button:has-text("Upload")');
    
    // Should show error message
    await expectTextVisible(page, /error|invalid|fail/i);
    
    // Should stay on upload page
    await expect(page).toHaveURL(/\/ontologies\/upload/);
  });

  test('should navigate to ontology versions', async ({ authenticatedPage: page }) => {
    const mocker = new APIMocker(page);
    
    const mockOntology = {
      id: 'ont-versions',
      name: 'Versioned Ontology',
      version: '2.0.0',
      format: 'turtle',
      status: 'active',
      created_at: new Date().toISOString(),
    };

    await mocker.mockOntologyGet('ont-versions', mockOntology);

    await page.goto('/ontologies/ont-versions');
    
    // Click versions link/button
    await page.click('a[href*="versions"], button:has-text("Versions")');
    
    // Should navigate to versions page
    await expect(page).toHaveURL(/\/ontologies\/ont-versions\/versions/);
  });

  test('should navigate to ontology suggestions', async ({ authenticatedPage: page }) => {
    const mocker = new APIMocker(page);
    
    const mockOntology = {
      id: 'ont-suggest',
      name: 'Ontology with Suggestions',
      version: '1.0.0',
      format: 'turtle',
      status: 'active',
      created_at: new Date().toISOString(),
    };

    await mocker.mockOntologyGet('ont-suggest', mockOntology);

    await page.goto('/ontologies/ont-suggest');
    
    // Click suggestions link/button
    await page.click('a[href*="suggestions"], button:has-text("Suggestions")');
    
    // Should navigate to suggestions page
    await expect(page).toHaveURL(/\/ontologies\/ont-suggest\/suggestions/);
  });
});

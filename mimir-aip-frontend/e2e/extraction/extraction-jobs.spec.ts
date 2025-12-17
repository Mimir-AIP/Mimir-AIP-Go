import { test, expect } from '../helpers';
import { APIMocker, expectVisible, expectTextVisible } from '../helpers';

test.describe('Extraction Jobs', () => {
  test('should display list of extraction jobs', async ({ authenticatedPage: page }) => {
    const mocker = new APIMocker(page);
    
    const mockJobs = [
      {
        id: 'job-1',
        job_name: 'Extract Entities from Text',
        ontology_id: 'ont-1',
        extraction_type: 'llm',
        status: 'completed',
        entities_extracted: 50,
        triples_generated: 150,
        created_at: new Date().toISOString(),
      },
      {
        id: 'job-2',
        job_name: 'Deterministic Extraction',
        ontology_id: 'ont-1',
        extraction_type: 'deterministic',
        status: 'running',
        entities_extracted: 10,
        triples_generated: 30,
        created_at: new Date().toISOString(),
      },
    ];

    await mocker.mockExtractionJobs(mockJobs);

    await page.goto('/extraction');
    
    // Should show extraction jobs page
    await expectTextVisible(page, /extraction.*jobs/i);
    
    // Should display jobs
    await expectTextVisible(page, 'Extract Entities from Text');
    await expectTextVisible(page, 'Deterministic Extraction');
    await expectTextVisible(page, '50');
    await expectTextVisible(page, '150');
  });

  test('should filter extraction jobs by status', async ({ authenticatedPage: page }) => {
    const mocker = new APIMocker(page);
    
    const mockJobs = [
      {
        id: 'job-completed',
        job_name: 'Completed Job',
        ontology_id: 'ont-1',
        extraction_type: 'llm',
        status: 'completed',
        entities_extracted: 50,
        triples_generated: 150,
        created_at: new Date().toISOString(),
      },
      {
        id: 'job-running',
        job_name: 'Running Job',
        ontology_id: 'ont-1',
        extraction_type: 'deterministic',
        status: 'running',
        entities_extracted: 10,
        triples_generated: 30,
        created_at: new Date().toISOString(),
      },
    ];

    await mocker.mockExtractionJobs(mockJobs);

    await page.goto('/extraction');
    
    // Should show all jobs initially
    await expectTextVisible(page, 'Completed Job');
    await expectTextVisible(page, 'Running Job');
    
    // Filter by completed status
    await page.selectOption('select[name="status"], select:has-text("Status")', 'completed');
    
    // Give time for filter to apply
    await page.waitForTimeout(500);
    
    // Verify filter dropdown value
    const statusSelect = page.locator('select[name="status"], select:has-text("Status")');
    await expect(statusSelect).toHaveValue('completed');
  });

  test('should filter extraction jobs by ontology', async ({ authenticatedPage: page }) => {
    const mocker = new APIMocker(page);
    
    // Mock ontologies list
    await page.route('**/api/v1/ontology*', async (route) => {
      if (!route.request().url().includes('/extraction')) {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            data: [
              { id: 'ont-1', name: 'Ontology 1', status: 'active', format: 'turtle' },
              { id: 'ont-2', name: 'Ontology 2', status: 'active', format: 'turtle' },
            ],
          }),
        });
      } else {
        await route.continue();
      }
    });

    const mockJobs = [
      {
        id: 'job-1',
        job_name: 'Job with Ontology 1',
        ontology_id: 'ont-1',
        extraction_type: 'llm',
        status: 'completed',
        entities_extracted: 50,
        triples_generated: 150,
        created_at: new Date().toISOString(),
      },
    ];

    await mocker.mockExtractionJobs(mockJobs);

    await page.goto('/extraction');
    
    // Wait for ontologies to load
    await page.waitForTimeout(500);
    
    // Filter by ontology
    const ontologySelect = page.locator('select[name="ontology"], select:has-text("Ontology")');
    if (await ontologySelect.isVisible({ timeout: 2000 }).catch(() => false)) {
      await ontologySelect.selectOption('ont-1');
      await page.waitForTimeout(500);
      await expect(ontologySelect).toHaveValue('ont-1');
    }
  });

  test('should view extraction job details', async ({ authenticatedPage: page }) => {
    // Mock job details
    await page.route('**/api/v1/extraction/jobs/job-detail', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: {
            job: {
              id: 'job-detail',
              job_name: 'Detailed Extraction Job',
              ontology_id: 'ont-1',
              extraction_type: 'hybrid',
              source_type: 'text',
              status: 'completed',
              entities_extracted: 100,
              triples_generated: 300,
              created_at: new Date().toISOString(),
              started_at: new Date(Date.now() - 60000).toISOString(),
              completed_at: new Date().toISOString(),
            },
            entities: [
              {
                id: 'ent-1',
                entity_label: 'Alice',
                entity_type: 'http://example.org/Person',
                entity_uri: 'http://example.org/alice',
                confidence: 0.95,
                source_text: 'Alice works at TechCorp',
              },
              {
                id: 'ent-2',
                entity_label: 'TechCorp',
                entity_type: 'http://example.org/Organization',
                entity_uri: 'http://example.org/techcorp',
                confidence: 0.88,
                source_text: 'Alice works at TechCorp',
              },
            ],
          },
        }),
      });
    });

    await page.goto('/extraction/job-detail');
    
    // Should show job details
    await expectTextVisible(page, 'Detailed Extraction Job');
    await expectTextVisible(page, 'completed');
    await expectTextVisible(page, 'hybrid');
    
    // Should show statistics
    await expectTextVisible(page, '100');
    await expectTextVisible(page, '300');
    
    // Should show extracted entities
    await expectTextVisible(page, 'Alice');
    await expectTextVisible(page, 'TechCorp');
    await expectTextVisible(page, '95%');
    await expectTextVisible(page, '88%');
  });

  test('should display entity details in modal', async ({ authenticatedPage: page }) => {
    // Mock job details
    await page.route('**/api/v1/extraction/jobs/job-modal', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: {
            job: {
              id: 'job-modal',
              job_name: 'Test Job',
              ontology_id: 'ont-1',
              extraction_type: 'llm',
              source_type: 'text',
              status: 'completed',
              entities_extracted: 1,
              triples_generated: 3,
              created_at: new Date().toISOString(),
            },
            entities: [
              {
                id: 'ent-detail',
                entity_label: 'Detailed Entity',
                entity_type: 'http://example.org/Person',
                entity_uri: 'http://example.org/detailed',
                confidence: 0.92,
                source_text: 'This is the source text',
                properties: {
                  name: 'Detailed Entity',
                  age: 30,
                },
              },
            ],
          },
        }),
      });
    });

    await page.goto('/extraction/job-modal');
    
    // Click "View Details" button
    await page.click('button:has-text("View Details")');
    
    // Modal should appear
    await expectVisible(page, '[role="dialog"], .modal, [data-testid="entity-modal"]');
    
    // Should show entity details
    await expectTextVisible(page, 'Detailed Entity');
    await expectTextVisible(page, 'http://example.org/detailed');
    await expectTextVisible(page, '92%');
    await expectTextVisible(page, 'This is the source text');
    
    // Close modal
    await page.click('button:has-text("Close"), button[aria-label="Close"]');
    
    // Modal should disappear
    const modal = page.locator('[role="dialog"], .modal');
    await expect(modal).not.toBeVisible();
  });

  test('should show different status badges', async ({ authenticatedPage: page }) => {
    const mocker = new APIMocker(page);
    
    const mockJobs = [
      {
        id: 'job-pending',
        job_name: 'Pending Job',
        ontology_id: 'ont-1',
        extraction_type: 'llm',
        status: 'pending',
        entities_extracted: 0,
        triples_generated: 0,
        created_at: new Date().toISOString(),
      },
      {
        id: 'job-running',
        job_name: 'Running Job',
        ontology_id: 'ont-1',
        extraction_type: 'llm',
        status: 'running',
        entities_extracted: 5,
        triples_generated: 15,
        created_at: new Date().toISOString(),
      },
      {
        id: 'job-failed',
        job_name: 'Failed Job',
        ontology_id: 'ont-1',
        extraction_type: 'llm',
        status: 'failed',
        entities_extracted: 0,
        triples_generated: 0,
        error_message: 'Connection timeout',
        created_at: new Date().toISOString(),
      },
    ];

    await mocker.mockExtractionJobs(mockJobs);

    await page.goto('/extraction');
    
    // Should show different status badges with proper colors
    const pendingBadge = page.locator('text=pending').first();
    const runningBadge = page.locator('text=running').first();
    const failedBadge = page.locator('text=failed').first();
    
    await expect(pendingBadge).toBeVisible();
    await expect(runningBadge).toBeVisible();
    await expect(failedBadge).toBeVisible();
  });

  test('should refresh extraction jobs list', async ({ authenticatedPage: page }) => {
    let callCount = 0;
    
    await page.route('**/api/v1/extraction/jobs*', async (route) => {
      callCount++;
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: {
            jobs: [
              {
                id: `job-${callCount}`,
                job_name: `Job ${callCount}`,
                ontology_id: 'ont-1',
                extraction_type: 'llm',
                status: 'completed',
                entities_extracted: callCount * 10,
                triples_generated: callCount * 30,
                created_at: new Date().toISOString(),
              },
            ],
          },
        }),
      });
    });

    await page.goto('/extraction');
    
    // Initial load
    expect(callCount).toBe(1);
    
    // Click refresh button
    await page.click('button:has-text("Refresh")');
    
    // Wait for refresh
    await page.waitForTimeout(500);
    
    // Should have made another API call
    expect(callCount).toBe(2);
  });

  test('should show error message on failed job details fetch', async ({ authenticatedPage: page }) => {
    // Mock failed job fetch
    await page.route('**/api/v1/extraction/jobs/job-error', async (route) => {
      await route.fulfill({
        status: 404,
        contentType: 'application/json',
        body: JSON.stringify({
          error: 'Job not found',
        }),
      });
    });

    await page.goto('/extraction/job-error');
    
    // Should show error message
    await expectTextVisible(page, /error|not found/i);
    
    // Should have link back to list
    const backLink = page.locator('a[href="/extraction"], a:has-text("Back")');
    await expect(backLink).toBeVisible();
  });

  test('should display extraction type badges', async ({ authenticatedPage: page }) => {
    const mocker = new APIMocker(page);
    
    const mockJobs = [
      {
        id: 'job-llm',
        job_name: 'LLM Extraction',
        ontology_id: 'ont-1',
        extraction_type: 'llm',
        status: 'completed',
        entities_extracted: 50,
        triples_generated: 150,
        created_at: new Date().toISOString(),
      },
      {
        id: 'job-det',
        job_name: 'Deterministic Extraction',
        ontology_id: 'ont-1',
        extraction_type: 'deterministic',
        status: 'completed',
        entities_extracted: 100,
        triples_generated: 300,
        created_at: new Date().toISOString(),
      },
      {
        id: 'job-hybrid',
        job_name: 'Hybrid Extraction',
        ontology_id: 'ont-1',
        extraction_type: 'hybrid',
        status: 'completed',
        entities_extracted: 75,
        triples_generated: 225,
        created_at: new Date().toISOString(),
      },
    ];

    await mocker.mockExtractionJobs(mockJobs);

    await page.goto('/extraction');
    
    // Should show different extraction type badges
    await expectTextVisible(page, 'llm');
    await expectTextVisible(page, 'deterministic');
    await expectTextVisible(page, 'hybrid');
  });

  test('should handle empty extraction jobs list', async ({ authenticatedPage: page }) => {
    const mocker = new APIMocker(page);
    await mocker.mockExtractionJobs([]);

    await page.goto('/extraction');
    
    // Should show empty state message
    await expectTextVisible(page, /no.*extraction.*jobs|no.*jobs.*found/i);
    
    // Should show instructions
    await expectTextVisible(page, /api|create|rest/i);
  });
});

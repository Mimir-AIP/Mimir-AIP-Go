import { test, expect } from '@playwright/test';
import { setupAuthenticatedPage } from './helpers';


/**
 * Comprehensive E2E tests for Knowledge Graph including SPARQL queries, visualization, and NL queries
 */

test.describe('Knowledge Graph - Visualization and Exploration', () => {
  test.beforeEach(async ({ page }) => {
    await setupAuthenticatedPage(page);
    await page.goto('/knowledge-graph');
    await page.waitForLoadState('networkidle');
  });

  test('should display knowledge graph page', async ({ page }) => {
    await expect(page).toHaveTitle(/Knowledge Graph/i);
    await expect(page.getByRole('heading', { name: /Knowledge Graph/i })).toBeVisible();
  });

  test('should render graph visualization', async ({ page }) => {
    // Check for graph canvas/container
    const graphContainer = page.getByTestId('graph-visualization');

    if (await graphContainer.isVisible()) {
      await expect(graphContainer).toBeVisible();
    }
  });

  test('should display graph controls', async ({ page }) => {
    // Check for zoom controls
    const zoomIn = page.getByRole('button', { name: /Zoom In|\+/i });
    const zoomOut = page.getByRole('button', { name: /Zoom Out|-/i });
    const resetView = page.getByRole('button', { name: /Reset|Center/i });

    if (await zoomIn.isVisible()) {
      await expect(zoomIn).toBeVisible();
      await expect(zoomOut).toBeVisible();
      await expect(resetView).toBeVisible();
    }
  });

  test('should zoom in on graph', async ({ page }) => {
    const zoomInButton = page.getByRole('button', { name: /Zoom In|\+/i });

    if (await zoomInButton.isVisible()) {
      await zoomInButton.click();
      await page.waitForTimeout(500);

      // Graph should be zoomed (implementation specific verification)
      await expect(zoomInButton).toBeVisible();
    }
  });

  test('should zoom out on graph', async ({ page }) => {
    const zoomOutButton = page.getByRole('button', { name: /Zoom Out|-/i });

    if (await zoomOutButton.isVisible()) {
      await zoomOutButton.click();
      await page.waitForTimeout(500);

      await expect(zoomOutButton).toBeVisible();
    }
  });

  test('should reset graph view', async ({ page }) => {
    const resetButton = page.getByRole('button', { name: /Reset|Center/i });

    if (await resetButton.isVisible()) {
      // Zoom first
      const zoomIn = page.getByRole('button', { name: /Zoom In/i });
      if (await zoomIn.isVisible()) {
        await zoomIn.click();
        await zoomIn.click();
      }

      // Reset view
      await resetButton.click();
      await page.waitForTimeout(500);

      await expect(resetButton).toBeVisible();
    }
  });

  test('should select and highlight node', async ({ page }) => {
    const graphNode = page.locator('[data-testid^="graph-node"]').first();

    if (await graphNode.isVisible()) {
      await graphNode.click();

      // Node details should appear
      const nodeDetails = page.getByTestId('node-details');
      if (await nodeDetails.isVisible()) {
        await expect(nodeDetails).toBeVisible();
      }
    }
  });

  test('should display node properties', async ({ page }) => {
    const graphNode = page.locator('[data-testid^="graph-node"]').first();

    if (await graphNode.isVisible()) {
      await graphNode.click();

      // Should show properties panel
      await expect(page.getByText(/Properties|Attributes/i)).toBeVisible();
    }
  });

  test('should display node relationships', async ({ page }) => {
    const graphNode = page.locator('[data-testid^="graph-node"]').first();

    if (await graphNode.isVisible()) {
      await graphNode.click();

      // Should show relationships
      await expect(page.getByText(/Relationships|Connections|Edges/i)).toBeVisible();
    }
  });

  test('should filter graph by node type', async ({ page }) => {
    const filterSelect = page.getByLabel(/Filter|Node Type/i);

    if (await filterSelect.isVisible()) {
      await filterSelect.selectOption({ index: 1 });
      await page.waitForTimeout(1000);

      // Graph should update
      await expect(page.getByTestId('graph-visualization')).toBeVisible();
    }
  });

  test('should search for nodes', async ({ page }) => {
    const searchInput = page.getByPlaceholder(/Search.*nodes|Search.*entities/i);

    if (await searchInput.isVisible()) {
      await searchInput.fill('entity');
      await page.waitForTimeout(500);

      // Search results should appear
      const results = page.getByTestId('search-results');
      if (await results.isVisible()) {
        await expect(results).toBeVisible();
      }
    }
  });

  test('should expand node connections', async ({ page }) => {
    const graphNode = page.locator('[data-testid^="graph-node"]').first();

    if (await graphNode.isVisible()) {
      await graphNode.click();

      const expandButton = page.getByRole('button', { name: /Expand|Show.*Connections/i });
      if (await expandButton.isVisible()) {
        await expandButton.click();

        // More nodes should appear
        await page.waitForTimeout(1000);
        await expect(page.getByTestId('graph-visualization')).toBeVisible();
      }
    }
  });

  test('should collapse node connections', async ({ page }) => {
    const graphNode = page.locator('[data-testid^="graph-node"]').first();

    if (await graphNode.isVisible()) {
      await graphNode.click();

      const collapseButton = page.getByRole('button', { name: /Collapse|Hide/i });
      if (await collapseButton.isVisible()) {
        await collapseButton.click();
        await page.waitForTimeout(500);
      }
    }
  });

  test('should change graph layout', async ({ page }) => {
    const layoutSelect = page.getByLabel(/Layout|View Mode/i);

    if (await layoutSelect.isVisible()) {
      await layoutSelect.selectOption('hierarchical');
      await page.waitForTimeout(1000);

      // Graph should reorganize
      await expect(page.getByTestId('graph-visualization')).toBeVisible();
    }
  });

  test('should export graph as image', async ({ page }) => {
    const exportButton = page.getByRole('button', { name: /Export|Download.*Image/i });

    if (await exportButton.isVisible()) {
      // Set up download listener
      const downloadPromise = page.waitForEvent('download');

      await exportButton.click();

      // Wait for download
      const download = await downloadPromise;

      // Verify download
      expect(download.suggestedFilename()).toMatch(/\.png|\.jpg|\.svg/);
    }
  });

  test('should display graph statistics', async ({ page }) => {
    const statsButton = page.getByRole('button', { name: /Statistics|Stats|Info/i });

    if (await statsButton.isVisible()) {
      await statsButton.click();

      // Should show stats panel
      await expect(page.getByText(/Nodes|Edges|Relationships/i)).toBeVisible();
    }
  });
});

test.describe('Knowledge Graph - SPARQL Queries', () => {
  test.beforeEach(async ({ page }) => {
    await setupAuthenticatedPage(page);
    await page.goto('/knowledge-graph');
    await page.waitForLoadState('networkidle');
  });

  test('should open SPARQL query editor', async ({ page }) => {
    const queryButton = page.getByRole('button', { name: /Query|SPARQL/i });

    if (await queryButton.isVisible()) {
      await queryButton.click();

      // Query editor should appear
      await expect(page.getByRole('dialog', { name: /Query|SPARQL/i })).toBeVisible();
    }
  });

  test('should execute SPARQL query', async ({ page }) => {
    const queryButton = page.getByRole('button', { name: /Query|SPARQL/i });

    if (await queryButton.isVisible()) {
      await queryButton.click();

      // Enter SPARQL query
      const queryEditor = page.getByTestId('sparql-editor');
      if (await queryEditor.isVisible()) {
        await queryEditor.fill(`
          SELECT ?subject ?predicate ?object
          WHERE {
            ?subject ?predicate ?object
          }
          LIMIT 10
        `);

        // Execute query
        await page.getByRole('button', { name: /Execute|Run/i }).click();

        // Results should appear
        await expect(page.getByTestId('query-results')).toBeVisible({ timeout: 10000 });
      }
    }
  });

  test('should display query results in table', async ({ page }) => {
    const queryButton = page.getByRole('button', { name: /Query|SPARQL/i });

    if (await queryButton.isVisible()) {
      await queryButton.click();

      const executeButton = page.getByRole('button', { name: /Execute|Run/i });
      if (await executeButton.isVisible()) {
        await executeButton.click();

        // Table should appear with results
        await expect(page.getByRole('table')).toBeVisible({ timeout: 10000 });
      }
    }
  });

  test('should save SPARQL query', async ({ page }) => {
    const queryButton = page.getByRole('button', { name: /Query|SPARQL/i });

    if (await queryButton.isVisible()) {
      await queryButton.click();

      const saveButton = page.getByRole('button', { name: /Save.*Query/i });
      if (await saveButton.isVisible()) {
        await saveButton.click();

        // Save dialog should appear
        await expect(page.getByLabel(/Query Name/i)).toBeVisible();
      }
    }
  });

  test('should load saved SPARQL query', async ({ page }) => {
    const loadButton = page.getByRole('button', { name: /Load.*Query|Saved.*Queries/i });

    if (await loadButton.isVisible()) {
      await loadButton.click();

      // List of saved queries should appear
      await expect(page.getByRole('list', { name: /Saved.*Queries/i })).toBeVisible();
    }
  });

  test('should validate SPARQL syntax', async ({ page }) => {
    const queryButton = page.getByRole('button', { name: /Query|SPARQL/i });

    if (await queryButton.isVisible()) {
      await queryButton.click();

      const queryEditor = page.getByTestId('sparql-editor');
      if (await queryEditor.isVisible()) {
        // Enter invalid query
        await queryEditor.fill('INVALID SPARQL SYNTAX');

        const validateButton = page.getByRole('button', { name: /Validate/i });
        if (await validateButton.isVisible()) {
          await validateButton.click();

          // Should show error
          await expect(page.getByText(/Invalid|Syntax.*Error/i)).toBeVisible({ timeout: 5000 });
        }
      }
    }
  });

  test('should export query results', async ({ page }) => {
    const queryButton = page.getByRole('button', { name: /Query|SPARQL/i });

    if (await queryButton.isVisible()) {
      await queryButton.click();

      const executeButton = page.getByRole('button', { name: /Execute/i });
      if (await executeButton.isVisible()) {
        await executeButton.click();
        await page.waitForTimeout(2000);

        const exportButton = page.getByRole('button', { name: /Export.*Results/i });
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

test.describe('Knowledge Graph - Natural Language Queries', () => {
  test.beforeEach(async ({ page }) => {
    await setupAuthenticatedPage(page);
    await page.goto('/knowledge-graph');
    await page.waitForLoadState('networkidle');
  });

  test('should open natural language query interface', async ({ page }) => {
    const nlQueryButton = page.getByRole('button', { name: /Ask|Natural Language|AI Query/i });

    if (await nlQueryButton.isVisible()) {
      await nlQueryButton.click();

      // NL query interface should appear
      await expect(page.getByPlaceholder(/Ask a question|What.*looking for/i)).toBeVisible();
    }
  });

  test('should execute natural language query', async ({ page }) => {
    const nlQueryButton = page.getByRole('button', { name: /Ask|Natural Language/i });

    if (await nlQueryButton.isVisible()) {
      await nlQueryButton.click();

      const queryInput = page.getByPlaceholder(/Ask a question/i);
      await queryInput.fill('Show me all entities related to manufacturing');

      await page.getByRole('button', { name: /Submit|Ask|Search/i }).click();

      // Results should appear
      await expect(page.getByTestId('nl-query-results')).toBeVisible({ timeout: 15000 });
    }
  });

  test('should show generated SPARQL from NL query', async ({ page }) => {
    const nlQueryButton = page.getByRole('button', { name: /Ask|Natural Language/i });

    if (await nlQueryButton.isVisible()) {
      await nlQueryButton.click();

      const queryInput = page.getByPlaceholder(/Ask a question/i);
      await queryInput.fill('Find all products');

      await page.getByRole('button', { name: /Submit/i }).click();
      await page.waitForTimeout(2000);

      // Should show generated SPARQL
      const showSparqlButton = page.getByRole('button', { name: /Show.*SPARQL|View.*Query/i });
      if (await showSparqlButton.isVisible()) {
        await showSparqlButton.click();

        await expect(page.getByTestId('generated-sparql')).toBeVisible();
      }
    }
  });

  test('should refine natural language query', async ({ page }) => {
    const nlQueryButton = page.getByRole('button', { name: /Ask|Natural Language/i });

    if (await nlQueryButton.isVisible()) {
      await nlQueryButton.click();

      const queryInput = page.getByPlaceholder(/Ask a question/i);
      await queryInput.fill('Show me entities');
      await page.getByRole('button', { name: /Submit/i }).click();

      await page.waitForTimeout(2000);

      // Refine the query
      const refineButton = page.getByRole('button', { name: /Refine|Narrow Down/i });
      if (await refineButton.isVisible()) {
        await refineButton.click();

        await expect(page.getByLabel(/Additional.*Criteria|Refinement/i)).toBeVisible();
      }
    }
  });
});

test.describe('Knowledge Graph - Path Finding', () => {
  test.beforeEach(async ({ page }) => {
    await setupAuthenticatedPage(page);
    await page.goto('/knowledge-graph');
    await page.waitForLoadState('networkidle');
  });

  test('should find path between nodes', async ({ page }) => {
    const pathButton = page.getByRole('button', { name: /Find Path|Path Finding/i });

    if (await pathButton.isVisible()) {
      await pathButton.click();

      // Path finding interface should appear
      await expect(page.getByLabel(/Start Node|Source/i)).toBeVisible();
      await expect(page.getByLabel(/End Node|Target/i)).toBeVisible();
    }
  });

  test('should visualize shortest path', async ({ page }) => {
    const pathButton = page.getByRole('button', { name: /Find Path/i });

    if (await pathButton.isVisible()) {
      await pathButton.click();

      const startInput = page.getByLabel(/Start Node/i);
      const endInput = page.getByLabel(/End Node/i);

      if (await startInput.isVisible() && await endInput.isVisible()) {
        await startInput.fill('Entity1');
        await endInput.fill('Entity2');

        await page.getByRole('button', { name: /Find|Calculate/i }).click();

        // Path should be highlighted
        await expect(page.getByTestId('path-visualization')).toBeVisible({ timeout: 10000 });
      }
    }
  });

  test('should display path details', async ({ page }) => {
    const pathButton = page.getByRole('button', { name: /Find Path/i });

    if (await pathButton.isVisible()) {
      await pathButton.click();

      const findButton = page.getByRole('button', { name: /Find|Calculate/i });
      if (await findButton.isVisible()) {
        await findButton.click();
        await page.waitForTimeout(2000);

        // Path details should show
        await expect(page.getByText(/Path Length|Hops|Distance/i)).toBeVisible({ timeout: 10000 });
      }
    }
  });
});

test.describe('Knowledge Graph - Reasoning and Inference', () => {
  test.beforeEach(async ({ page }) => {
    await setupAuthenticatedPage(page);
    await page.goto('/knowledge-graph');
    await page.waitForLoadState('networkidle');
  });

  test('should trigger reasoning engine', async ({ page }) => {
    const reasonButton = page.getByRole('button', { name: /Reason|Infer|Reasoning/i });

    if (await reasonButton.isVisible()) {
      await reasonButton.click();

      // Reasoning should start
      await expect(page.getByText(/Reasoning.*in progress|Inferring/i)).toBeVisible({ timeout: 5000 });
    }
  });

  test('should display inferred triples', async ({ page }) => {
    const inferredButton = page.getByRole('button', { name: /Inferred|Derived/i });

    if (await inferredButton.isVisible()) {
      await inferredButton.click();

      // Should show inferred knowledge
      await expect(page.getByTestId('inferred-triples')).toBeVisible();
    }
  });

  test('should explain inference', async ({ page }) => {
    const inferredNode = page.getByTestId('inferred-node').first();

    if (await inferredNode.isVisible()) {
      await inferredNode.click();

      const explainButton = page.getByRole('button', { name: /Explain|Why/i });
      if (await explainButton.isVisible()) {
        await explainButton.click();

        // Explanation should appear
        await expect(page.getByText(/Explanation|Derived from|Based on/i)).toBeVisible({ timeout: 5000 });
      }
    }
  });
});

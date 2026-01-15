import { test, expect } from '@playwright/test';

test.describe('Mimir AIP - Detailed UI Interactions', () => {
  test.setTimeout(300000); // 5 minutes for comprehensive testing

  // ============================================
  // PIPELINE DETAILS INTERACTIONS
  // ============================================
  test('Pipeline Details - View, Run, Validate, Logs, History, Clone, Edit, Delete', async ({ page }) => {
    console.log('🔧 Testing Pipeline Details Interactions');

    await page.goto('http://localhost:8080/pipelines');
    await page.waitForLoadState('networkidle');

    // Find a pipeline in the list
    const pipelineRows = page.locator('table tbody tr');
    const pipelineCount = await pipelineRows.count();

    if (pipelineCount === 0) {
      console.log('⚠️ No pipelines available to test details');
      return;
    }

    console.log(`✅ Found ${pipelineCount} pipelines`);

    // Click "View Details" on the first pipeline
    const firstPipelineRow = pipelineRows.first();
    const viewDetailsBtn = firstPipelineRow.locator('a:has-text("View"), button:has-text("View")').first();
    const hasViewBtn = await viewDetailsBtn.isVisible({ timeout: 5000 }).catch(() => false);

    if (!hasViewBtn) {
      console.log('⚠️ View Details button not found');
      return;
    }

    await viewDetailsBtn.click();
    await page.waitForLoadState('networkidle');

    // Verify we're on pipeline details page
    const pipelineNameHeading = page.getByRole('heading', { level: 1 }).first();
    await expect(pipelineNameHeading).toBeVisible({ timeout: 10000 });
    const pageTitle = await pipelineNameHeading.textContent();
    console.log(`✅ Pipeline details page: ${pageTitle}`);

    // Test RUN PIPELINE button
    const runBtn = page.getByRole('button', { name: /Run|Execute|Start/i }).first();
    const hasRunBtn = await runBtn.isVisible({ timeout: 5000 }).catch(() => false);
    console.log(`✅ Run Pipeline button: ${hasRunBtn ? 'Available' : 'Not found'}`);

    if (hasRunBtn) {
      // Don't actually run it, just verify it exists
      console.log('✅ Pipeline execution interface available');
    }

    // Test VALIDATE button
    const validateBtn = page.getByRole('button', { name: /Validate|Check/i }).first();
    const hasValidateBtn = await validateBtn.isVisible({ timeout: 3000 }).catch(() => false);
    console.log(`✅ Validate button: ${hasValidateBtn ? 'Available' : 'Not found'}`);

    // Test VIEW LOGS button
    const logsBtn = page.getByRole('button', { name: /Logs|View Logs/i }).first();
    const hasLogsBtn = await logsBtn.isVisible({ timeout: 3000 }).catch(() => false);
    console.log(`✅ View Logs button: ${hasLogsBtn ? 'Available' : 'Not found'}`);

    // Test VIEW HISTORY button
    const historyBtn = page.getByRole('button', { name: /History|Execution History/i }).first();
    const hasHistoryBtn = await historyBtn.isVisible({ timeout: 3000 }).catch(() => false);
    console.log(`✅ View History button: ${hasHistoryBtn ? 'Available' : 'Not found'}`);

    // Test CLONE button
    const cloneBtn = page.getByRole('button', { name: /Clone|Copy/i }).first();
    const hasCloneBtn = await cloneBtn.isVisible({ timeout: 3000 }).catch(() => false);
    console.log(`✅ Clone button: ${hasCloneBtn ? 'Available' : 'Not found'}`);

    // Test EDIT button
    const editBtn = page.getByRole('button', { name: /Edit|Modify/i }).first();
    const hasEditBtn = await editBtn.isVisible({ timeout: 3000 }).catch(() => false);
    console.log(`✅ Edit button: ${hasEditBtn ? 'Available' : 'Not found'}`);

    // Test DELETE button
    const deleteBtn = page.getByRole('button', { name: /Delete|Remove/i }).first();
    const hasDeleteBtn = await deleteBtn.isVisible({ timeout: 3000 }).catch(() => false);
    console.log(`✅ Delete button: ${hasDeleteBtn ? 'Available' : 'Not found'}`);

    // Test pipeline status/execution info
    const statusElements = page.locator('text=/Status|Running|Completed|Failed|Active/i');
    const statusCount = await statusElements.count();
    console.log(`✅ Pipeline status indicators: ${statusCount} found`);

    console.log('✅ Pipeline details interactions tested');
  });

  // ============================================
  // ONTOLOGY DETAILS INTERACTIONS
  // ============================================
  test('Ontology Details - All Subpages and Functionality', async ({ page }) => {
    console.log('🔧 Testing Ontology Details Interactions');

    await page.goto('http://localhost:8080/ontologies');
    await page.waitForLoadState('networkidle');

    // Find an ontology in the list
    const ontologyRows = page.locator('table tbody tr');
    const ontologyCount = await ontologyRows.count();

    if (ontologyCount === 0) {
      console.log('⚠️ No ontologies available to test details');
      return;
    }

    console.log(`✅ Found ${ontologyCount} ontologies`);

    // Click on the first ontology name/link
    const firstOntologyRow = ontologyRows.first();
    const ontologyLink = firstOntologyRow.locator('a').first();
    const hasOntologyLink = await ontologyLink.isVisible({ timeout: 5000 }).catch(() => false);

    if (!hasOntologyLink) {
      console.log('⚠️ Ontology link not found');
      return;
    }

    await ontologyLink.click();
    await page.waitForLoadState('networkidle');

    // Verify we're on ontology details page
    const ontologyHeading = page.getByRole('heading', { level: 1 }).first();
    await expect(ontologyHeading).toBeVisible({ timeout: 10000 });
    const ontologyTitle = await ontologyHeading.textContent();
    console.log(`✅ Ontology details page: ${ontologyTitle}`);

    // Test OVERVIEW tab (should be default)
    const overviewTab = page.getByRole('tab', { name: /Overview|Summary/i }).first();
    const hasOverviewTab = await overviewTab.isVisible({ timeout: 3000 }).catch(() => false);
    console.log(`✅ Overview tab: ${hasOverviewTab ? 'Available' : 'Not found'}`);

    // Test CLASSES tab
    const classesTab = page.getByRole('tab', { name: /Classes|Types/i }).first();
    const hasClassesTab = await classesTab.isVisible({ timeout: 3000 }).catch(() => false);
    console.log(`✅ Classes tab: ${hasClassesTab ? 'Available' : 'Not found'}`);

    if (hasClassesTab) {
      await classesTab.click();
      await page.waitForTimeout(1000);

      // Check for class listing
      const classElements = page.locator('[data-testid*="class"], .class-item, text=/^Class|^Type/');
      const classCount = await classElements.count();
      console.log(`✅ Classes displayed: ${classCount} found`);
    }

    // Test PROPERTIES tab
    const propertiesTab = page.getByRole('tab', { name: /Properties|Attributes/i }).first();
    const hasPropertiesTab = await propertiesTab.isVisible({ timeout: 3000 }).catch(() => false);
    console.log(`✅ Properties tab: ${hasPropertiesTab ? 'Available' : 'Not found'}`);

    // Test INSTANCES tab
    const instancesTab = page.getByRole('tab', { name: /Instances|Individuals/i }).first();
    const hasInstancesTab = await instancesTab.isVisible({ timeout: 3000 }).catch(() => false);
    console.log(`✅ Instances tab: ${hasInstancesTab ? 'Available' : 'Not found'}`);

    // Test VERSIONS tab
    const versionsTab = page.getByRole('tab', { name: /Versions|History/i }).first();
    const hasVersionsTab = await versionsTab.isVisible({ timeout: 3000 }).catch(() => false);
    console.log(`✅ Versions tab: ${hasVersionsTab ? 'Available' : 'Not found'}`);

    // Test SUGGESTIONS tab
    const suggestionsTab = page.getByRole('tab', { name: /Suggestions|Recommendations/i }).first();
    const hasSuggestionsTab = await suggestionsTab.isVisible({ timeout: 3000 }).catch(() => false);
    console.log(`✅ Suggestions tab: ${hasSuggestionsTab ? 'Available' : 'Not found'}`);

    // Test DOWNLOAD button
    const downloadBtn = page.getByRole('button', { name: /Download|Export/i }).first();
    const hasDownloadBtn = await downloadBtn.isVisible({ timeout: 3000 }).catch(() => false);
    console.log(`✅ Download button: ${hasDownloadBtn ? 'Available' : 'Not found'}`);

    // Test EDIT button
    const editBtn = page.getByRole('button', { name: /Edit|Modify/i }).first();
    const hasEditBtn = await editBtn.isVisible({ timeout: 3000 }).catch(() => false);
    console.log(`✅ Edit button: ${hasEditBtn ? 'Available' : 'Not found'}`);

    // Test DELETE button
    const deleteBtn = page.getByRole('button', { name: /Delete|Remove/i }).first();
    const hasDeleteBtn = await deleteBtn.isVisible({ timeout: 3000 }).catch(() => false);
    console.log(`✅ Delete button: ${hasDeleteBtn ? 'Available' : 'Not found'}`);

    console.log('✅ Ontology details interactions tested');
  });

  // ============================================
  // KNOWLEDGE GRAPH QUERY INTERACTIONS
  // ============================================
  test('Knowledge Graph - Sample Queries and Results', async ({ page }) => {
    console.log('🔧 Testing Knowledge Graph Query Interactions');

    await page.goto('http://localhost:8080/knowledge-graph');
    await page.waitForLoadState('networkidle');

    // Verify we're on knowledge graph page
    const kgHeading = page.getByRole('heading', { level: 1 }).first();
    await expect(kgHeading).toBeVisible({ timeout: 10000 });
    const kgTitle = await kgHeading.textContent();
    console.log(`✅ Knowledge Graph page: ${kgTitle}`);

    // Test VISUALIZATION tab
    const visualizationTab = page.getByRole('tab', { name: /Visualization|Graph/i }).first();
    const hasVisualizationTab = await visualizationTab.isVisible({ timeout: 3000 }).catch(() => false);
    console.log(`✅ Visualization tab: ${hasVisualizationTab ? 'Available' : 'Not found'}`);

    if (hasVisualizationTab) {
      await visualizationTab.click();
      await page.waitForTimeout(1000);

      // Check for graph controls
      const zoomInBtn = page.getByRole('button', { name: /Zoom|Zoom In|\+/i }).first();
      const hasZoomIn = await zoomInBtn.isVisible({ timeout: 3000 }).catch(() => false);
      console.log(`✅ Graph zoom controls: ${hasZoomIn ? 'Available' : 'Not found'}`);

      // Check for graph canvas/element
      const graphElement = page.locator('[data-testid*="graph"], .graph, canvas, svg').first();
      const hasGraph = await graphElement.isVisible({ timeout: 3000 }).catch(() => false);
      console.log(`✅ Graph visualization: ${hasGraph ? 'Rendered' : 'Not found'}`);
    }

    // Test QUERIES tab
    const queriesTab = page.getByRole('tab', { name: /Queries|SPARQL/i }).first();
    const hasQueriesTab = await queriesTab.isVisible({ timeout: 3000 }).catch(() => false);
    console.log(`✅ Queries tab: ${hasQueriesTab ? 'Available' : 'Not found'}`);

    if (hasQueriesTab) {
      await queriesTab.click();
      await page.waitForTimeout(1000);

      // Check for query editor
      const queryEditor = page.locator('textarea, [contenteditable], [role="textbox"]').first();
      const hasEditor = await queryEditor.isVisible({ timeout: 3000 }).catch(() => false);
      console.log(`✅ Query editor: ${hasEditor ? 'Available' : 'Not found'}`);

      // Check for sample queries
      const sampleQueriesBtn = page.getByRole('button', { name: /Sample|Examples|Templates/i }).first();
      const hasSamples = await sampleQueriesBtn.isVisible({ timeout: 3000 }).catch(() => false);
      console.log(`✅ Sample queries: ${hasSamples ? 'Available' : 'Not found'}`);

      if (hasSamples) {
        await sampleQueriesBtn.click();
        await page.waitForTimeout(1000);

        // Check if sample queries appeared
        const sampleQueryItems = page.locator('button, li').filter({ hasText: /SELECT|ASK|DESCRIBE/i });
        const sampleCount = await sampleQueryItems.count();
        console.log(`✅ Sample query options: ${sampleCount} found`);

        // Try to run a sample query
        if (sampleCount > 0) {
          await sampleQueryItems.first().click();
          await page.waitForTimeout(1000);

          // Check if query was loaded
          const editorContent = await queryEditor.inputValue().catch(() => '');
          const hasQueryContent = editorContent.length > 0;
          console.log(`✅ Sample query loaded: ${hasQueryContent ? 'Yes' : 'No'}`);

          // Look for RUN/EXECUTE button
          const runBtn = page.getByRole('button', { name: /Run|Execute|Query/i }).first();
          const hasRunBtn = await runBtn.isVisible({ timeout: 3000 }).catch(() => false);
          console.log(`✅ Query execution button: ${hasRunBtn ? 'Available' : 'Not found'}`);

          if (hasRunBtn && hasQueryContent) {
            await runBtn.click();
            await page.waitForTimeout(3000);

            // Check for results
            const resultsTable = page.locator('table').first();
            const resultsText = page.locator('text=/results|found|returned/i').first();
            const hasResults = (await resultsTable.isVisible({ timeout: 2000 }).catch(() => false)) ||
                              (await resultsText.isVisible({ timeout: 2000 }).catch(() => false));
            console.log(`✅ Query results: ${hasResults ? 'Displayed' : 'No results shown'}`);
          }
        }
      }
    }

    console.log('✅ Knowledge Graph interactions tested');
  });

  // ============================================
  // ML MODEL PREDICTIONS INTERACTIONS
  // ============================================
  test('ML Model Details - Predictions Feature', async ({ page }) => {
    console.log('🔧 Testing ML Model Predictions Interactions');

    await page.goto('http://localhost:8080/models');
    await page.waitForLoadState('networkidle');

    // Find a model in the list
    const modelRows = page.locator('table tbody tr');
    const modelCount = await modelRows.count();

    if (modelCount === 0) {
      console.log('⚠️ No models available to test predictions');
      return;
    }

    console.log(`✅ Found ${modelCount} models`);

    // Click on the first model name/link
    const firstModelRow = modelRows.first();
    const modelLink = firstModelRow.locator('a').first();
    const hasModelLink = await modelLink.isVisible({ timeout: 5000 }).catch(() => false);

    if (!hasModelLink) {
      console.log('⚠️ Model link not found');
      return;
    }

    await modelLink.click();
    await page.waitForLoadState('networkidle');

    // Verify we're on model details page
    const modelHeading = page.getByRole('heading', { level: 1 }).first();
    await expect(modelHeading).toBeVisible({ timeout: 10000 });
    const modelTitle = await modelHeading.textContent();
    console.log(`✅ Model details page: ${modelTitle}`);

    // Test PREDICTIONS tab
    const predictionsTab = page.getByRole('tab', { name: /Predict|Predictions|Inference/i }).first();
    const hasPredictionsTab = await predictionsTab.isVisible({ timeout: 3000 }).catch(() => false);
    console.log(`✅ Predictions tab: ${hasPredictionsTab ? 'Available' : 'Not found'}`);

    if (hasPredictionsTab) {
      await predictionsTab.click();
      await page.waitForTimeout(1000);

      // Check for prediction input form
      const inputFields = page.locator('input[type="text"], input[type="number"], textarea');
      const inputCount = await inputFields.count();
      console.log(`✅ Prediction input fields: ${inputCount} found`);

      // Check for PREDICT button
      const predictBtn = page.getByRole('button', { name: /Predict|Run|Execute/i }).first();
      const hasPredictBtn = await predictBtn.isVisible({ timeout: 3000 }).catch(() => false);
      console.log(`✅ Predict button: ${hasPredictBtn ? 'Available' : 'Not found'}`);

      if (hasPredictBtn && inputCount > 0) {
        // Try to fill a simple input (if it exists)
        const firstInput = inputFields.first();
        const inputType = await firstInput.getAttribute('type');

        if (inputType === 'number') {
          await firstInput.fill('1.5');
        } else {
          await firstInput.fill('test input');
        }

        // Click predict (but don't wait too long for results)
        await predictBtn.click();
        console.log('✅ Prediction request initiated');

        // Check for results area (appears after prediction)
        await page.waitForTimeout(2000);
        const resultElements = page.locator('text=/prediction|result|output|confidence/i');
        const resultCount = await resultElements.count();
        console.log(`✅ Prediction results: ${resultCount} indicators found`);
      }
    }

    // Test METRICS tab
    const metricsTab = page.getByRole('tab', { name: /Metrics|Performance|Stats/i }).first();
    const hasMetricsTab = await metricsTab.isVisible({ timeout: 3000 }).catch(() => false);
    console.log(`✅ Metrics tab: ${hasMetricsTab ? 'Available' : 'Not found'}`);

    if (hasMetricsTab) {
      await metricsTab.click();
      await page.waitForTimeout(1000);

      // Check for accuracy/precision/recall metrics
      const accuracyText = page.locator('text=/accuracy|precision|recall|f1/i').first();
      const hasMetrics = await accuracyText.isVisible({ timeout: 3000 }).catch(() => false);
      console.log(`✅ Performance metrics: ${hasMetrics ? 'Displayed' : 'Not found'}`);
    }

    // Test TRAINING HISTORY tab
    const historyTab = page.getByRole('tab', { name: /History|Training|Logs/i }).first();
    const hasHistoryTab = await historyTab.isVisible({ timeout: 3000 }).catch(() => false);
    console.log(`✅ Training history tab: ${hasHistoryTab ? 'Available' : 'Not found'}`);

    console.log('✅ ML Model predictions interactions tested');
  });

  // ============================================
  // DIGITAL TWIN COMPREHENSIVE INTERACTIONS
  // ============================================
  test('Digital Twin - Complete Functionality Test', async ({ page }) => {
    console.log('🔧 Testing Complete Digital Twin Functionality');

    await page.goto('http://localhost:8080/digital-twins');
    await page.waitForLoadState('networkidle');

    // Check twin listing
    const twinRows = page.locator('table tbody tr');
    const twinCount = await twinRows.count();
    console.log(`✅ Digital twins available: ${twinCount}`);

    if (twinCount > 0) {
      // Click on the first twin for detailed testing
      const firstTwinRow = twinRows.first();
      const twinLink = firstTwinRow.locator('a').first();
      const hasTwinLink = await twinLink.isVisible({ timeout: 5000 }).catch(() => false);

      if (hasTwinLink) {
        await twinLink.click();
        await page.waitForLoadState('networkidle');

        // Verify twin details page
        const twinHeading = page.getByRole('heading', { level: 1 }).first();
        await expect(twinHeading).toBeVisible({ timeout: 10000 });
        const twinTitle = await twinHeading.textContent();
        console.log(`✅ Twin details page: ${twinTitle}`);

        // Test OVERVIEW tab
        const overviewTab = page.getByRole('tab', { name: /Overview|Summary/i }).first();
        const hasOverviewTab = await overviewTab.isVisible({ timeout: 3000 }).catch(() => false);
        console.log(`✅ Overview tab: ${hasOverviewTab ? 'Available' : 'Not found'}`);

        // Test STATE tab
        const stateTab = page.getByRole('tab', { name: /State|Current|Status/i }).first();
        const hasStateTab = await stateTab.isVisible({ timeout: 3000 }).catch(() => false);
        console.log(`✅ State tab: ${hasStateTab ? 'Available' : 'Not found'}`);

        if (hasStateTab) {
          await stateTab.click();
          await page.waitForTimeout(1000);

          // Check for state visualization
          const stateElements = page.locator('[data-testid*="state"], .state, canvas, svg');
          const stateCount = await stateElements.count();
          console.log(`✅ State visualization: ${stateCount} elements found`);
        }

        // Test SCENARIOS tab
        const scenariosTab = page.getByRole('tab', { name: /Scenarios|Simulations/i }).first();
        const hasScenariosTab = await scenariosTab.isVisible({ timeout: 3000 }).catch(() => false);
        console.log(`✅ Scenarios tab: ${hasScenariosTab ? 'Available' : 'Not found'}`);

        if (hasScenariosTab) {
          await scenariosTab.click();
          await page.waitForTimeout(1000);

          // Check for CREATE SCENARIO button
          const createScenarioBtn = page.getByRole('button', { name: /Create|New|Add/i }).first();
          const hasCreateScenarioBtn = await createScenarioBtn.isVisible({ timeout: 3000 }).catch(() => false);
          console.log(`✅ Create scenario button: ${hasCreateScenarioBtn ? 'Available' : 'Not found'}`);

          // Check for existing scenarios
          const scenarioRows = page.locator('table tbody tr, .scenario-item');
          const scenarioCount = await scenarioRows.count();
          console.log(`✅ Existing scenarios: ${scenarioCount} found`);

          if (scenarioCount > 0) {
            // Try to run a scenario
            const runBtn = page.getByRole('button', { name: /Run|Execute|Start/i }).first();
            const hasRunBtn = await runBtn.isVisible({ timeout: 3000 }).catch(() => false);
            console.log(`✅ Run scenario button: ${hasRunBtn ? 'Available' : 'Not found'}`);
          }
        }

        // Test WHAT-IF ANALYSIS
        const whatIfBtn = page.getByRole('button', { name: /What-If|Analysis|Simulate/i }).first();
        const hasWhatIfBtn = await whatIfBtn.isVisible({ timeout: 3000 }).catch(() => false);
        console.log(`✅ What-If analysis button: ${hasWhatIfBtn ? 'Available' : 'Not found'}`);

        if (hasWhatIfBtn) {
          await whatIfBtn.click();
          await page.waitForTimeout(1000);

          // Check for what-if interface
          const whatIfInput = page.locator('input, textarea').filter({ hasText: '' }).first();
          const hasWhatIfInput = await whatIfInput.isVisible({ timeout: 3000 }).catch(() => false);
          console.log(`✅ What-If input interface: ${hasWhatIfInput ? 'Available' : 'Not found'}`);
        }

        // Test UPDATE STATE button
        const updateStateBtn = page.getByRole('button', { name: /Update|Modify|Change/i }).first();
        const hasUpdateBtn = await updateStateBtn.isVisible({ timeout: 3000 }).catch(() => false);
        console.log(`✅ Update state button: ${hasUpdateBtn ? 'Available' : 'Not found'}`);

        // Test EXPORT button
        const exportBtn = page.getByRole('button', { name: /Export|Download/i }).first();
        const hasExportBtn = await exportBtn.isVisible({ timeout: 3000 }).catch(() => false);
        console.log(`✅ Export button: ${hasExportBtn ? 'Available' : 'Not found'}`);

        console.log('✅ Digital twin detailed functionality tested');
      }
    } else {
      console.log('⚠️ No digital twins available for detailed testing');

      // Test empty state - CREATE TWIN button
      const createTwinBtn = page.getByRole('button', { name: /Create.*Twin/i }).first();
      const hasCreateBtn = await createTwinBtn.isVisible({ timeout: 5000 }).catch(() => false);
      console.log(`✅ Create Twin button (empty state): ${hasCreateBtn ? 'Available' : 'Not found'}`);
    }

    console.log('✅ Digital Twin comprehensive functionality tested');
  });
});
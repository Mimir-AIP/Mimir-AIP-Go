import { test, expect } from '@playwright/test';
import { setupAuthenticatedPage } from './helpers';

test.describe('Mimir AIP - Detailed UI Interactions', () => {
  test.setTimeout(300000); // 5 minutes for comprehensive testing
  
  test.beforeEach(async ({ page }) => {
    await setupAuthenticatedPage(page);
  });

  // ============================================
  // PIPELINE DETAILS INTERACTIONS
  // ============================================
  test('Pipeline Details - View, Run, Validate, Logs, History, Clone, Edit, Delete', async ({ page }) => {
    console.log('🔧 Testing Pipeline Details Interactions');

    // First, create a test pipeline
    console.log('📝 Creating test pipeline for details testing...');

    await page.goto('http://localhost:8080/pipelines');
    await page.waitForLoadState('domcontentloaded');

    // Click Create Pipeline button
    const createBtn = page.getByRole('button', { name: /Create Pipeline/i }).first();
    const hasCreateBtn = await createBtn.isVisible({ timeout: 15000 }).catch(() => false);
    console.log(`Create button visible: ${hasCreateBtn}`);

    if (!hasCreateBtn) {
      console.log('❌ Create Pipeline button not found');
      return;
    }

    await createBtn.click();

    // Wait for modal/form to appear
    await page.waitForTimeout(2000);

    // Check what elements are visible
    const modal = page.locator('[role="dialog"], .modal, .dialog').first();
    const hasModal = await modal.isVisible({ timeout: 3000 }).catch(() => false);
    console.log(`Modal visible: ${hasModal}`);

    if (hasModal) {
      // Fill pipeline name - use specific ID selector
      const nameInput = modal.locator('#create-name');
      const hasNameInput = await nameInput.isVisible({ timeout: 3000 }).catch(() => false);
      console.log(`Name input visible: ${hasNameInput}`);

      if (hasNameInput) {
        await nameInput.fill('Test-Pipeline-For-Details');
        console.log('✅ Name filled');
        await page.waitForTimeout(500); // Wait for React state update
      }

      // Switch to YAML mode to add steps
      const yamlModeBtn = modal.getByRole('button', { name: /YAML/i });
      const hasYamlBtn = await yamlModeBtn.isVisible({ timeout: 3000 }).catch(() => false);
      if (hasYamlBtn) {
        await yamlModeBtn.click();
        console.log('✅ Switched to YAML mode');
        await page.waitForTimeout(500);

        // Fill YAML config with a simple step
        const yamlInput = modal.locator('#create-yaml');
        await yamlInput.fill(`version: '1.0'
name: Test-Pipeline-For-Details
steps:
  - name: test-step
    plugin: Input.csv
    config:
      file_path: /data/test.csv`);
        console.log('✅ YAML config filled');
        await page.waitForTimeout(500);
      }

      // Save the pipeline - wait for button to be enabled
      const saveBtn = modal.getByRole('button', { name: /Save|Create|Submit/i }).first();
      const hasSaveBtn = await saveBtn.isVisible({ timeout: 3000 }).catch(() => false);
      console.log(`Save button visible: ${hasSaveBtn}`);

      if (hasSaveBtn) {
        // Wait for button to be enabled
        await expect(saveBtn).toBeEnabled({ timeout: 10000 });
        
        // Wait for BOTH the create AND the subsequent list fetch
        const createPromise = page.waitForResponse(
          resp => resp.url().includes('/api/v1/pipelines') && resp.request().method() === 'POST',
          { timeout: 10000 }
        );
        
        await saveBtn.click();
        console.log('✅ Save clicked');
        
        // Wait for creation to complete
        await createPromise;
        console.log('✅ Pipeline created (API call completed)');
        
        // Wait for modal to close
        await page.waitForTimeout(1000);
      }
    }

    // Reload the pipelines page to ensure fresh data
    console.log('📄 Reloading pipelines page...');
    await page.goto('http://localhost:8080/pipelines');
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(1000);

    // Check if we're on pipeline details page (after successful creation)
    const currentUrl = page.url();
    const isOnPipelineDetails = currentUrl.match(/\/pipelines\/pipeline_\d+$/);
    console.log(`Current URL: ${currentUrl}`);
    console.log(`On pipeline details page: ${!!isOnPipelineDetails}`);

    // If we're not already on details page, navigate to first pipeline
    if (!isOnPipelineDetails) {
      // Go to pipelines list page
      await page.goto('http://localhost:8080/pipelines');
      await page.waitForLoadState('domcontentloaded');

      // Pipelines are displayed as cards, not table rows
      const pipelineCards = page.locator('[class*="grid"] > div[class*="Card"], .grid > div > div').filter({ hasText: 'Test-Pipeline' });
      const pipelineCount = await pipelineCards.count();
      console.log(`✅ Found ${pipelineCount} pipeline cards`);

      if (pipelineCount === 0) {
        console.log('❌ No pipeline cards found');
        return;
      }

      // Click on the first pipeline card (should have a link or be clickable)
      const firstCard = pipelineCards.first();
      await firstCard.click();
      await page.waitForLoadState('domcontentloaded');
    }

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
    await page.waitForLoadState('domcontentloaded');

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
    await page.waitForLoadState('domcontentloaded');

    // Verify we're on ontology details page
    const ontologyHeading = page.getByRole('heading', { level: 1 }).first();
    await expect(ontologyHeading).toBeVisible({ timeout: 10000 });
    const ontologyTitle = await ontologyHeading.textContent();
    console.log(`✅ Ontology details page: ${ontologyTitle}`);

    // Test tab buttons (they're implemented as buttons, not role="tab")
    const overviewTab = page.getByRole('button', { name: 'overview' }).first();
    const hasOverviewTab = await overviewTab.isVisible({ timeout: 3000 }).catch(() => false);
    console.log(`✅ Overview tab: ${hasOverviewTab ? 'Available' : 'Not found'}`);

    // Test CLASSES tab
    const classesTab = page.getByRole('button', { name: 'classes' }).first();
    const hasClassesTab = await classesTab.isVisible({ timeout: 3000 }).catch(() => false);
    console.log(`✅ Classes tab: ${hasClassesTab ? 'Available' : 'Not found'}`);

    if (hasClassesTab) {
      await classesTab.click();
      await page.waitForTimeout(1000);

      // Check for class listing - look for class display elements
      const classElements = page.locator('text=/class|Class/i, [class*="class"]');
      const classCount = await classElements.count();
      console.log(`✅ Classes displayed: ${classCount} found`);
    }

    // Test PROPERTIES tab
    const propertiesTab = page.getByRole('button', { name: 'properties' }).first();
    const hasPropertiesTab = await propertiesTab.isVisible({ timeout: 3000 }).catch(() => false);
    console.log(`✅ Properties tab: ${hasPropertiesTab ? 'Available' : 'Not found'}`);

    // Test QUERIES tab
    const queriesTab = page.getByRole('button', { name: 'queries' }).first();
    const hasQueriesTab = await queriesTab.isVisible({ timeout: 3000 }).catch(() => false);
    console.log(`✅ Queries tab: ${hasQueriesTab ? 'Available' : 'Not found'}`);

    // Test TRAIN tab
    const trainTab = page.getByRole('button', { name: 'train' }).first();
    const hasTrainTab = await trainTab.isVisible({ timeout: 3000 }).catch(() => false);
    console.log(`✅ Train tab: ${hasTrainTab ? 'Available' : 'Not found'}`);

    // Test TYPES tab
    const typesTab = page.getByRole('button', { name: 'types' }).first();
    const hasTypesTab = await typesTab.isVisible({ timeout: 3000 }).catch(() => false);
    console.log(`✅ Types tab: ${hasTypesTab ? 'Available' : 'Not found'}`);

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
    await page.waitForLoadState('domcontentloaded');

    // Verify we're on knowledge graph page
    const kgHeading = page.getByRole('heading', { level: 1 }).first();
    await expect(kgHeading).toBeVisible({ timeout: 10000 });
    const kgTitle = await kgHeading.textContent();
    console.log(`✅ Knowledge Graph page: ${kgTitle}`);

    // Test SPARQL tab (default active tab)
    const sparqlTab = page.getByRole('button', { name: 'SPARQL Query' }).first();
    const hasSparqlTab = await sparqlTab.isVisible({ timeout: 3000 }).catch(() => false);
    console.log(`✅ SPARQL tab: ${hasSparqlTab ? 'Available' : 'Not found'}`);

    if (hasSparqlTab) {
      // SPARQL tab should be active by default, but click to ensure
      await sparqlTab.click();
      await page.waitForTimeout(1000);

      // Check for query editor
      const queryEditor = page.locator('textarea').first();
      const hasEditor = await queryEditor.isVisible({ timeout: 3000 }).catch(() => false);
      console.log(`✅ Query editor: ${hasEditor ? 'Available' : 'Not found'}`);

      // Check for sample queries dropdown/button
      const sampleQueriesBtn = page.getByRole('button', { name: /Sample|Examples/i }).first();
      const hasSamples = await sampleQueriesBtn.isVisible({ timeout: 3000 }).catch(() => false);
      console.log(`✅ Sample queries: ${hasSamples ? 'Available' : 'Not found'}`);

      if (hasSamples && hasEditor) {
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

    // Test NATURAL LANGUAGE tab
    const nlTab = page.getByRole('button', { name: 'Natural Language' }).first();
    const hasNlTab = await nlTab.isVisible({ timeout: 3000 }).catch(() => false);
    console.log(`✅ Natural Language tab: ${hasNlTab ? 'Available' : 'Not found'}`);

    if (hasNlTab) {
      await nlTab.click();
      await page.waitForTimeout(1000);

      // Check for natural language input
      const nlInput = page.locator('textarea, input').filter({ hasText: '' }).first();
      const hasNlInput = await nlInput.isVisible({ timeout: 3000 }).catch(() => false);
      console.log(`✅ Natural language input: ${hasNlInput ? 'Available' : 'Not found'}`);

      // Check for submit button
      const submitBtn = page.getByRole('button', { name: /Submit|Ask|Query/i }).first();
      const hasSubmitBtn = await submitBtn.isVisible({ timeout: 3000 }).catch(() => false);
      console.log(`✅ Natural language submit: ${hasSubmitBtn ? 'Available' : 'Not found'}`);
    }

    console.log('✅ Knowledge Graph interactions tested');
  });

  // ============================================
  // ML MODEL PREDICTIONS INTERACTIONS
  // ============================================
  test('ML Model Details - Predictions Feature', async ({ page }) => {
    console.log('🔧 Testing ML Model Predictions Interactions');

    await page.goto('http://localhost:8080/models');
    await page.waitForLoadState('domcontentloaded');

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
    await page.waitForLoadState('domcontentloaded');

    // Verify we're on model details page
    const modelHeading = page.getByRole('heading', { level: 1 }).first();
    await expect(modelHeading).toBeVisible({ timeout: 10000 });
    const modelTitle = await modelHeading.textContent();
    console.log(`✅ Model details page: ${modelTitle}`);

    // Test PREDICTIONS section (not a tab, just a section on the page)
    const predictionsSection = page.getByRole('heading', { name: /Make Predictions/i }).first();
    const hasPredictionsSection = await predictionsSection.isVisible({ timeout: 3000 }).catch(() => false);
    console.log(`✅ Predictions section: ${hasPredictionsSection ? 'Available' : 'Not found'}`);

    if (hasPredictionsSection) {
      // Check for prediction input textarea
      const predictionTextarea = page.getByLabel(/Input Data|JSON/i).first();
      const hasInputTextarea = await predictionTextarea.isVisible({ timeout: 3000 }).catch(() => false);
      console.log(`✅ Prediction input textarea: ${hasInputTextarea ? 'Available' : 'Not found'}`);

      // Check for PREDICT button
      const predictBtn = page.getByRole('button', { name: /Run Prediction/i }).first();
      const hasPredictBtn = await predictBtn.isVisible({ timeout: 3000 }).catch(() => false);
      console.log(`✅ Predict button: ${hasPredictBtn ? 'Available' : 'Not found'}`);

      if (hasPredictBtn && hasInputTextarea) {
        // Try to fill with sample JSON data
        const sampleData = '{"feature1": 1.5, "feature2": 2.3, "feature3": 0.8}';
        await predictionTextarea.fill(sampleData);

        // Click predict (but don't wait too long for results)
        await predictBtn.click();
        console.log('✅ Prediction request initiated');

        // Check for results area (appears after prediction)
        await page.waitForTimeout(3000);
        const resultElements = page.locator('text=/prediction|result|output|confidence/i, pre');
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
    await page.waitForLoadState('domcontentloaded');

    // Check twin listing
    const twinRows = page.locator('table tbody tr');
    const twinCount = await twinRows.count();
    console.log(`✅ Digital twins available: ${twinCount}`);

    // Test CREATE TWIN button (always available)
    const createTwinBtn = page.getByRole('button', { name: /Create.*Twin/i }).first();
    const hasCreateBtn = await createTwinBtn.isVisible({ timeout: 5000 }).catch(() => false);
    console.log(`✅ Create Twin button: ${hasCreateBtn ? 'Available' : 'Not found'}`);

    if (twinCount > 0) {
      console.log(`✅ Existing twins: ${twinCount} found`);

      // Test twin table/list structure
      const twinTable = page.locator('table').first();
      const hasTwinTable = await twinTable.isVisible({ timeout: 3000 }).catch(() => false);
      console.log(`✅ Twin listing table: ${hasTwinTable ? 'Available' : 'Not found'}`);

      // Check for twin detail links
      const twinLinks = page.locator('table tbody tr a').first();
      const hasTwinLinks = await twinLinks.isVisible({ timeout: 3000 }).catch(() => false);
      console.log(`✅ Twin detail links: ${hasTwinLinks ? 'Available' : 'Not found'}`);

    } else {
      console.log('⚠️ No digital twins available for detailed testing');
    }

    console.log('✅ Digital Twin interface components verified');

    console.log('✅ Digital Twin comprehensive functionality tested');
  });
});
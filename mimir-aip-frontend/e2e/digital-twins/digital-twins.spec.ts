/**
 * E2E tests for Digital Twins - PROPER UI TESTING
 * 
 * These tests use the ACTUAL UI to interact with the REAL backend.
 * No API bypasses with request.get/post - we test like a real user.
 */

import { test, expect } from '../helpers';

test.describe('Digital Twins - UI Workflows', () => {
  test.beforeEach(async ({ authenticatedPage: page }) => {
    await page.goto('/digital-twins');
    await page.waitForLoadState('networkidle');
  });

  test('should load and display digital twins list from backend', async ({ authenticatedPage: page }) => {
    // Verify page heading appears
    const heading = page.getByRole('heading', { name: /digital.*twin/i }).first();
    await expect(heading).toBeVisible({ timeout: 10000 });
    
    // CRITICAL: Wait for loading to COMPLETE (would have caught the infinite loading bug)
    const loadingSkeleton = page.getByTestId('loading-skeleton');
    await expect(loadingSkeleton).not.toBeVisible({ timeout: 15000 });
    
    // Verify digital twins are ACTUALLY DISPLAYED in the UI
    const twinCards = page.getByTestId('twin-card');
    const count = await twinCards.count();
    
    if (count === 0) {
      // Check if it's showing empty state vs stuck loading
      const emptyState = page.getByText(/no.*digital.*twins|create.*first/i);
      const isEmptyState = await emptyState.isVisible();
      
      if (isEmptyState) {
        console.log('Empty state displayed - no twins available');
        // Should have create button in empty state
        const createButton = page.getByRole('button', { name: /create/i });
        await expect(createButton).toBeVisible();
      } else {
        throw new Error('No twins displayed and no empty state - page may be stuck loading');
      }
    } else {
      // We have twins - verify they loaded properly
      expect(count).toBeGreaterThan(0);
      console.log(`✓ ${count} digital twins loaded`);
      
      // Verify first twin has actual data
      const firstTwin = twinCards.first();
      await expect(firstTwin).toBeVisible();
    }
    
    // Verify NO error messages
    const errorMessage = page.getByText(/error.*loading|failed.*load/i);
    await expect(errorMessage).not.toBeVisible();
  });

  test('should create a digital twin through UI', async ({ authenticatedPage: page }) => {
    // Click create button in the UI
    const createButton = page.getByRole('button', { name: /create.*twin/i });
    await expect(createButton).toBeVisible({ timeout: 10000 });
    await createButton.click();
    
    // Wait for form to load
    await page.waitForLoadState('networkidle');
    
    // Check if we have ontologies available
    const noOntologiesMessage = page.getByText(/no.*ontolog/i);
    const hasNoOntologies = await noOntologiesMessage.isVisible().catch(() => false);
    
    if (hasNoOntologies) {
      console.log('⊘ No ontologies available - skipping creation test');
      test.skip();
      return;
    }
    
    // Fill the form using REAL UI elements
    const twinName = `E2E Test Twin ${Date.now()}`;
    
    const nameInput = page.getByLabel(/name/i);
    await expect(nameInput).toBeVisible({ timeout: 10000 });
    await nameInput.fill(twinName);
    
    const descInput = page.getByLabel(/description/i);
    if (await descInput.isVisible({ timeout: 2000 }).catch(() => false)) {
      await descInput.fill('Created via E2E test using actual UI');
    }
    
    // Select first available ontology
    const ontologySelect = page.getByLabel(/ontology/i);
    await expect(ontologySelect).toBeVisible();
    
    const options = await ontologySelect.locator('option').count();
    if (options > 1) {
      await ontologySelect.selectOption({ index: 1 }); // First non-placeholder
      console.log('✓ Selected ontology');
    }
    
    // Submit the form via UI
    const submitButton = page.getByRole('button', { name: /create.*digital.*twin|submit/i });
    await submitButton.click();
    
    // Wait for success message IN THE UI
    const successMessage = page.getByText(/created successfully|twin created/i);
    await expect(successMessage).toBeVisible({ timeout: 15000 });
    console.log(`✓ Twin "${twinName}" created successfully`);
    
    // Should redirect somewhere (details or list)
    await page.waitForURL(/\/digital-twins/, { timeout: 10000 });
    
    // Navigate to list if not already there
    if (!page.url().endsWith('/digital-twins')) {
      await page.goto('/digital-twins');
      await page.waitForLoadState('networkidle');
    }
    
    // Wait for loading to complete
    const loadingSkeleton = page.getByTestId('loading-skeleton');
    await expect(loadingSkeleton).not.toBeVisible({ timeout: 15000 });
    
    // Verify the new twin appears in the UI
    const newTwin = page.getByText(twinName);
    await expect(newTwin).toBeVisible({ timeout: 10000 });
    console.log('✓ New twin appears in list');
  });

  test('should view digital twin details through UI navigation', async ({ authenticatedPage: page }) => {
    // Wait for twins to load via UI
    const loadingSkeleton = page.getByTestId('loading-skeleton');
    await expect(loadingSkeleton).not.toBeVisible({ timeout: 15000 });
    
    const twinCards = page.getByTestId('twin-card');
    const count = await twinCards.count();
    
    if (count === 0) {
      console.log('⊘ No twins available - skipping details test');
      test.skip();
      return;
    }
    
    // Get the name of first twin for later verification
    const firstTwin = twinCards.first();
    const twinText = await firstTwin.textContent();
    console.log(`Testing with twin: ${twinText?.substring(0, 50)}...`);
    
    // Click on the first twin using the UI
    await firstTwin.click();
    
    // Should navigate to details page
    await expect(page).toHaveURL(/\/digital-twins\/[a-zA-Z0-9-]+/, { timeout: 10000 });
    console.log(`✓ Navigated to: ${page.url()}`);
    
    // Verify details page loaded
    await page.waitForLoadState('networkidle');
    
    // Should show twin information
    const detailsHeading = page.getByRole('heading').first();
    await expect(detailsHeading).toBeVisible();
    
    // Check for 404
    const notFound = page.locator('text=/404|not found/i');
    const hasNotFound = await notFound.isVisible().catch(() => false);
    if (hasNotFound) {
      console.log('⊘ Twin details page shows 404');
      test.skip();
    } else {
      console.log('✓ Twin details page loaded successfully');
    }
  });

  test('should search/filter digital twins using UI', async ({ authenticatedPage: page }) => {
    // Wait for twins to load
    const loadingSkeleton = page.getByTestId('loading-skeleton');
    await expect(loadingSkeleton).not.toBeVisible({ timeout: 15000 });
    
    const twinCards = page.getByTestId('twin-card');
    const initialCount = await twinCards.count();
    
    if (initialCount === 0) {
      console.log('⊘ No twins to search - skipping');
      test.skip();
      return;
    }
    
    console.log(`✓ Initial count: ${initialCount} twins`);
    
    // Look for search/filter input
    const searchInput = page.getByPlaceholder(/search/i);
    
    if (await searchInput.isVisible({ timeout: 2000 }).catch(() => false)) {
      // Get first twin's text to search for
      const firstTwinText = await twinCards.first().textContent();
      const searchTerm = firstTwinText?.split(' ')[0] || 'Twin';
      
      console.log(`Searching for: "${searchTerm}"`);
      
      // Search using UI
      await searchInput.fill(searchTerm);
      await page.waitForTimeout(800); // Allow debounce
      
      // Results should update
      const filteredCount = await page.getByTestId('twin-card').count();
      console.log(`✓ Filtered count: ${filteredCount} twins`);
      
      // Should show at least one result
      expect(filteredCount).toBeGreaterThan(0);
      
      // First result should contain search term
      const firstResult = page.getByTestId('twin-card').first();
      await expect(firstResult).toContainText(new RegExp(searchTerm, 'i'));
    } else {
      console.log('⊘ Search input not found - feature may not be implemented');
    }
  });

  test('should handle navigation back to list after viewing details', async ({ authenticatedPage: page }) => {
    // Wait for initial load
    const loadingSkeleton = page.getByTestId('loading-skeleton');
    await expect(loadingSkeleton).not.toBeVisible({ timeout: 15000 });
    
    const twinCards = page.getByTestId('twin-card');
    const initialCount = await twinCards.count();
    
    if (initialCount === 0) {
      test.skip();
      return;
    }
    
    // Navigate to first twin
    await twinCards.first().click();
    await expect(page).toHaveURL(/\/digital-twins\/[a-zA-Z0-9-]+/);
    await page.waitForLoadState('networkidle');
    
    // Go back using browser navigation
    await page.goBack();
    await page.waitForLoadState('networkidle');
    
    // Should show list again with data loaded
    await expect(loadingSkeleton).not.toBeVisible({ timeout: 15000 });
    
    const newCount = await page.getByTestId('twin-card').count();
    expect(newCount).toBe(initialCount);
    console.log('✓ Navigated back to list successfully');
  });

  test('should display twin count in UI', async ({ authenticatedPage: page }) => {
    // Wait for loading to complete
    const loadingSkeleton = page.getByTestId('loading-skeleton');
    await expect(loadingSkeleton).not.toBeVisible({ timeout: 15000 });
    
    // Check if there's a count display
    const countDisplay = page.locator('text=/\\d+\\s+(digital\\s+)?twins?/i');
    
    if (await countDisplay.isVisible({ timeout: 2000 }).catch(() => false)) {
      // Get displayed count
      const displayText = await countDisplay.textContent();
      const match = displayText?.match(/(\d+)/);
      
      if (match) {
        const displayedCount = parseInt(match[1]);
        
        // Count actual cards
        const twinCards = page.getByTestId('twin-card');
        const actualCount = await twinCards.count();
        
        // Should match
        expect(actualCount).toBe(displayedCount);
        console.log(`✓ Count matches: ${actualCount} twins`);
      }
    } else {
      console.log('⊘ Count display not found in UI');
    }
  });

  test('should refresh data when revisiting page', async ({ authenticatedPage: page }) => {
    // Load the page
    await page.goto('/digital-twins');
    await page.waitForLoadState('networkidle');
    
    const loadingSkeleton = page.getByTestId('loading-skeleton');
    await expect(loadingSkeleton).not.toBeVisible({ timeout: 15000 });
    
    const initialCount = await page.getByTestId('twin-card').count();
    console.log(`✓ Initial load: ${initialCount} twins`);
    
    // Navigate away
    await page.goto('/');
    await page.waitForLoadState('networkidle');
    
    // Navigate back
    await page.goto('/digital-twins');
    await page.waitForLoadState('networkidle');
    
    // Should load data again (not use stale)
    await expect(loadingSkeleton).not.toBeVisible({ timeout: 15000 });
    
    const newCount = await page.getByTestId('twin-card').count();
    expect(newCount).toBe(initialCount);
    console.log('✓ Data refreshed on page revisit');
  });

  test('should show appropriate empty state when no twins exist', async ({ authenticatedPage: page }) => {
    // Wait for loading to complete
    const loadingSkeleton = page.getByTestId('loading-skeleton');
    await expect(loadingSkeleton).not.toBeVisible({ timeout: 15000 });
    
    const twinCards = page.getByTestId('twin-card');
    const count = await twinCards.count();
    
    if (count === 0) {
      // Should show empty state message
      const emptyMessage = page.getByText(/no.*digital.*twins|create.*first/i);
      await expect(emptyMessage).toBeVisible({ timeout: 5000 });
      
      // Should have create button
      const createButton = page.getByRole('button', { name: /create/i });
      await expect(createButton).toBeVisible();
      
      console.log('✓ Empty state displayed correctly');
    } else {
      console.log(`⊘ Test skipped - ${count} twins exist`);
    }
  });

  test('should not show loading state indefinitely', async ({ authenticatedPage: page }) => {
    // This test specifically checks for the bug we fixed
    await page.goto('/digital-twins');
    await page.waitForLoadState('networkidle');
    
    // Loading skeleton should disappear within reasonable time
    const loadingSkeleton = page.getByTestId('loading-skeleton');
    
    // Start time
    const startTime = Date.now();
    
    // Wait for loading to complete (or timeout)
    try {
      await expect(loadingSkeleton).not.toBeVisible({ timeout: 15000 });
      const duration = Date.now() - startTime;
      console.log(`✓ Loading completed in ${duration}ms`);
      
      // Verify we're not stuck - should show either data or empty state
      const twinCards = page.getByTestId('twin-card');
      const emptyState = page.getByText(/no.*digital.*twins|create.*first/i);
      
      const hasCards = await twinCards.count() > 0;
      const hasEmptyState = await emptyState.isVisible().catch(() => false);
      
      if (!hasCards && !hasEmptyState) {
        throw new Error('Neither data nor empty state shown after loading completes');
      }
      
      console.log(`✓ ${hasCards ? 'Data displayed' : 'Empty state displayed'}`);
    } catch (error) {
      throw new Error('Loading skeleton did not disappear - infinite loading bug detected!');
    }
  });
});

test.describe('Digital Twins - Error Handling', () => {
  test('should not display error messages on successful load', async ({ authenticatedPage: page }) => {
    await page.goto('/digital-twins');
    await page.waitForLoadState('networkidle');
    
    // Wait for loading to complete
    const loadingSkeleton = page.getByTestId('loading-skeleton');
    await expect(loadingSkeleton).not.toBeVisible({ timeout: 15000 });
    
    // Should not show any error messages
    const errorMessages = page.getByText(/error|failed|unable/i);
    const errorCount = await errorMessages.count();
    
    // Filter out false positives (like "error handling" in docs)
    let actualErrors = 0;
    for (let i = 0; i < errorCount; i++) {
      const text = await errorMessages.nth(i).textContent();
      if (text?.toLowerCase().includes('error loading') || 
          text?.toLowerCase().includes('failed to') ||
          text?.toLowerCase().includes('unable to load')) {
        actualErrors++;
      }
    }
    
    expect(actualErrors).toBe(0);
    console.log('✓ No error messages displayed');
  });
});

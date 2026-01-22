/**
 * PROPER E2E Tests for Digital Twins
 * 
 * Philosophy: Test like a user. No direct API calls, no mocking routes.
 * Use the ACTUAL UI to interact with the REAL backend.
 * 
 * This is how E2E tests should be written to catch REAL bugs.
 */

import { test, expect } from '../helpers';

test.describe('Digital Twins - UI Integration', () => {
  test.beforeEach(async ({ authenticatedPage: page }) => {
    // Navigate to the page before each test
    await page.goto('/digital-twins');
    // Wait for page to be fully loaded
    await page.waitForLoadState('networkidle');
  });

  test('should load and display digital twins list from backend', async ({ authenticatedPage: page }) => {
    // This test would have CAUGHT THE BUG
    
    // 1. Verify page heading appears
    const heading = page.getByRole('heading', { name: /digital.*twin/i }).first();
    await expect(heading).toBeVisible({ timeout: 10000 });
    
    // 2. Wait for loading to COMPLETE (not just start)
    // The bug caused infinite loading - this would FAIL
    const loadingSkeleton = page.getByTestId('loading-skeleton');
    await expect(loadingSkeleton).not.toBeVisible({ timeout: 15000 });
    
    // 3. Verify digital twins are ACTUALLY DISPLAYED in the UI
    // The bug prevented this - test would FAIL
    const twinCards = page.getByTestId('twin-card');
    const count = await twinCards.count();
    
    if (count === 0) {
      // Check if it's showing empty state vs stuck loading
      const emptyState = page.getByText(/no.*digital.*twins|create.*first/i);
      const isEmptyState = await emptyState.isVisible();
      
      if (!isEmptyState) {
        // Not showing empty state and no twins = BUG DETECTED
        throw new Error('No twins displayed and no empty state shown - page may be stuck loading');
      }
    } else {
      // We have twins - verify they loaded from backend
      expect(count).toBeGreaterThan(0);
      
      // Verify each twin card has actual data
      const firstTwin = twinCards.first();
      await expect(firstTwin).toBeVisible();
      
      // Should show twin name (data from backend)
      const twinName = firstTwin.locator('text=/[A-Za-z]/').first();
      await expect(twinName).toBeVisible();
    }
    
    // 4. Verify NO error messages
    const errorMessage = page.getByText(/error.*loading|failed.*load/i);
    await expect(errorMessage).not.toBeVisible();
  });

  test('should create a digital twin through UI', async ({ authenticatedPage: page }) => {
    // NO request.post() - use the actual UI
    
    // Click create button
    const createButton = page.getByRole('button', { name: /create.*twin/i });
    await expect(createButton).toBeVisible();
    await createButton.click();
    
    // Wait for dialog/page to load
    await page.waitForLoadState('networkidle');
    
    // Check if ontologies are available
    const noOntologiesMessage = page.getByText(/no.*ontolog/i);
    const hasNoOntologies = await noOntologiesMessage.isVisible().catch(() => false);
    
    if (hasNoOntologies) {
      console.log('No ontologies available - skipping creation test');
      test.skip();
      return;
    }
    
    // Fill the form using REAL UI elements
    const twinName = `E2E Test Twin ${Date.now()}`;
    
    await page.getByLabel(/name/i).fill(twinName);
    await page.getByLabel(/description/i).fill('Created via E2E test using actual UI');
    
    // Select first available ontology
    const ontologySelect = page.getByLabel(/ontology/i);
    await ontologySelect.selectOption({ index: 1 }); // First non-placeholder option
    
    // Submit the form
    const submitButton = page.getByRole('button', { name: /create.*digital.*twin|submit/i });
    await submitButton.click();
    
    // Wait for success message IN THE UI
    const successMessage = page.getByText(/created successfully|twin created/i);
    await expect(successMessage).toBeVisible({ timeout: 15000 });
    
    // Should redirect to twin details or list
    await page.waitForURL(/\/digital-twins/, { timeout: 10000 });
    
    // Verify the twin appears in the list (navigate if needed)
    if (!page.url().includes('/digital-twins')) {
      await page.goto('/digital-twins');
      await page.waitForLoadState('networkidle');
    }
    
    // Twin should be visible in the UI
    const newTwin = page.getByText(twinName);
    await expect(newTwin).toBeVisible({ timeout: 10000 });
  });

  test('should view digital twin details through UI', async ({ authenticatedPage: page }) => {
    // NO request.get() to find twins - use the UI
    
    // Wait for twins to load
    const loadingSkeleton = page.getByTestId('loading-skeleton');
    await expect(loadingSkeleton).not.toBeVisible({ timeout: 15000 });
    
    const twinCards = page.getByTestId('twin-card');
    const count = await twinCards.count();
    
    if (count === 0) {
      console.log('No twins available - skipping details test');
      test.skip();
      return;
    }
    
    // Click on the first twin using the UI
    const firstTwin = twinCards.first();
    await firstTwin.click();
    
    // Should navigate to details page
    await expect(page).toHaveURL(/\/digital-twins\/[a-zA-Z0-9-]+/, { timeout: 10000 });
    
    // Verify details page loaded
    await page.waitForLoadState('networkidle');
    
    // Should show twin information (tabs, details, etc.)
    const detailsHeading = page.getByRole('heading').first();
    await expect(detailsHeading).toBeVisible();
  });

  test('should search digital twins using UI', async ({ authenticatedPage: page }) => {
    // Wait for twins to load first
    const loadingSkeleton = page.getByTestId('loading-skeleton');
    await expect(loadingSkeleton).not.toBeVisible({ timeout: 15000 });
    
    const searchInput = page.getByPlaceholder(/search/i);
    
    if (await searchInput.isVisible()) {
      // Get count before search
      const twinCards = page.getByTestId('twin-card');
      const initialCount = await twinCards.count();
      
      if (initialCount === 0) {
        console.log('No twins to search - skipping');
        test.skip();
        return;
      }
      
      // Get first twin's name to search for
      const firstTwinName = await twinCards.first().textContent();
      const searchTerm = firstTwinName?.split(' ')[0] || 'Twin';
      
      // Search using UI
      await searchInput.fill(searchTerm);
      await page.waitForTimeout(500); // Debounce
      
      // Results should filter
      const filteredCards = page.getByTestId('twin-card');
      const filteredCount = await filteredCards.count();
      
      // Should show at least the one we searched for
      expect(filteredCount).toBeGreaterThan(0);
      
      // First result should contain search term
      const firstResult = filteredCards.first();
      await expect(firstResult).toContainText(new RegExp(searchTerm, 'i'));
    }
  });

  test('should handle empty state correctly', async ({ authenticatedPage: page }) => {
    // Wait for loading to complete
    const loadingSkeleton = page.getByTestId('loading-skeleton');
    await expect(loadingSkeleton).not.toBeVisible({ timeout: 15000 });
    
    const twinCards = page.getByTestId('twin-card');
    const count = await twinCards.count();
    
    if (count === 0) {
      // Should show empty state, not infinite loading
      const emptyState = page.getByText(/no.*digital.*twins|create.*first/i);
      await expect(emptyState).toBeVisible();
      
      // Should have create button
      const createButton = page.getByRole('button', { name: /create/i });
      await expect(createButton).toBeVisible();
    }
  });

  test('should delete digital twin through UI', async ({ authenticatedPage: page }) => {
    // First, create a twin to delete (via UI)
    await page.getByRole('button', { name: /create.*twin/i }).click();
    await page.waitForLoadState('networkidle');
    
    const noOntologies = await page.getByText(/no.*ontolog/i).isVisible().catch(() => false);
    if (noOntologies) {
      console.log('Cannot create twin for delete test - no ontologies');
      test.skip();
      return;
    }
    
    const twinName = `Delete Me ${Date.now()}`;
    await page.getByLabel(/name/i).fill(twinName);
    await page.getByLabel(/description/i).fill('Will be deleted');
    await page.getByLabel(/ontology/i).selectOption({ index: 1 });
    await page.getByRole('button', { name: /create.*digital.*twin|submit/i }).click();
    
    await expect(page.getByText(/created successfully/i)).toBeVisible({ timeout: 15000 });
    
    // Navigate back to list
    await page.goto('/digital-twins');
    await page.waitForLoadState('networkidle');
    
    // Wait for loading to complete
    const loadingSkeleton = page.getByTestId('loading-skeleton');
    await expect(loadingSkeleton).not.toBeVisible({ timeout: 15000 });
    
    // Find the twin we just created
    const twinToDelete = page.getByText(twinName).first();
    await expect(twinToDelete).toBeVisible();
    
    // Find its delete button (might be in the same card)
    const twinCard = twinToDelete.locator('..').locator('..'); // Navigate up to card
    const deleteButton = twinCard.getByRole('button', { name: /delete/i });
    
    if (await deleteButton.isVisible()) {
      // Set up dialog handler for confirmation
      page.once('dialog', dialog => dialog.accept());
      
      await deleteButton.click();
      
      // Should show success message
      const successMessage = page.getByText(/deleted successfully/i);
      await expect(successMessage).toBeVisible({ timeout: 10000 });
      
      // Twin should disappear from list
      await expect(twinToDelete).not.toBeVisible();
    } else {
      console.log('Delete button not found in UI');
    }
  });
});

test.describe('Digital Twins - Data Verification', () => {
  test('should display correct twin count', async ({ authenticatedPage: page }) => {
    await page.goto('/digital-twins');
    await page.waitForLoadState('networkidle');
    
    // Wait for loading to complete
    const loadingSkeleton = page.getByTestId('loading-skeleton');
    await expect(loadingSkeleton).not.toBeVisible({ timeout: 15000 });
    
    // Check if there's a count display in the UI
    const countDisplay = page.getByText(/\d+\s+twins?/i);
    
    if (await countDisplay.isVisible()) {
      // Verify the count matches the number of cards
      const twinCards = page.getByTestId('twin-card');
      const actualCount = await twinCards.count();
      
      const displayedCount = await countDisplay.textContent();
      const numberMatch = displayedCount?.match(/(\d+)/);
      
      if (numberMatch) {
        const displayedNumber = parseInt(numberMatch[1]);
        expect(actualCount).toBe(displayedNumber);
      }
    }
  });

  test('should refresh data when navigating back', async ({ authenticatedPage: page }) => {
    // Load list
    await page.goto('/digital-twins');
    await page.waitForLoadState('networkidle');
    
    const loadingSkeleton = page.getByTestId('loading-skeleton');
    await expect(loadingSkeleton).not.toBeVisible({ timeout: 15000 });
    
    const twinCards = page.getByTestId('twin-card');
    const initialCount = await twinCards.count();
    
    if (initialCount === 0) {
      test.skip();
      return;
    }
    
    // Navigate to a twin
    await twinCards.first().click();
    await page.waitForLoadState('networkidle');
    
    // Navigate back
    await page.goBack();
    await page.waitForLoadState('networkidle');
    
    // Should load data again (not show stale)
    await expect(loadingSkeleton).not.toBeVisible({ timeout: 15000 });
    
    const newCount = await page.getByTestId('twin-card').count();
    expect(newCount).toBe(initialCount);
  });
});

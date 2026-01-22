# Testing Guidelines for Mimir AIP

## Philosophy: Test Like a Human, Not a Robot

**E2E tests should use the ACTUAL UI with the REAL backend. No mocking, no API bypasses.**

This document exists because we had a critical bug that tests didn't catch:
- The Digital Twins page showed infinite loading instead of displaying 336 twins
- All tests passed ✅
- The bug made it to production 💥

Why? Tests bypassed the UI and called APIs directly with `request.get()`.

---

## The Golden Rules

### 1. **E2E Tests = User Simulation**

✅ **DO THIS:**
```typescript
test('should load digital twins', async ({ page }) => {
  // Navigate like a user
  await page.goto('/digital-twins');
  
  // Wait for loading to COMPLETE
  await expect(page.getByTestId('loading-skeleton')).not.toBeVisible({ timeout: 15000 });
  
  // Verify data appears in the UI
  const twins = page.getByTestId('twin-card');
  const count = await twins.count();
  expect(count).toBeGreaterThan(0);
  
  // Verify no errors
  await expect(page.getByText(/error/i)).not.toBeVisible();
});
```

❌ **DON'T DO THIS:**
```typescript
test('should load digital twins', async ({ page, request }) => {
  // BYPASSES THE UI - doesn't test if UI actually loads data
  const response = await request.get('/api/v1/twins');
  const twins = await response.json();
  
  // Test passes even if UI is broken!
  expect(twins.length).toBeGreaterThan(0);
});
```

---

### 2. **No Direct API Calls in E2E Tests**

Users don't have `request.get()`. They click buttons and see results.

✅ **DO THIS:**
```typescript
test('should create digital twin', async ({ page }) => {
  // Click the create button
  await page.getByRole('button', { name: 'Create Twin' }).click();
  
  // Fill the form
  await page.getByLabel('Name').fill('Test Twin');
  await page.getByLabel('Ontology').selectOption({ index: 1 });
  
  // Submit
  await page.getByRole('button', { name: /create/i }).click();
  
  // Verify success message appears
  await expect(page.getByText(/created successfully/i)).toBeVisible();
});
```

❌ **DON'T DO THIS:**
```typescript
test('should create digital twin', async ({ page, request }) => {
  // Creates via API, bypasses form validation, UI bugs, etc.
  const response = await request.post('/api/v1/twin/create', {
    data: { name: 'Test Twin', ontology_id: 'ont-1' }
  });
  
  expect(response.ok()).toBeTruthy();
});
```

---

### 3. **Don't Swallow Errors**

✅ **DO THIS:**
```typescript
test('should not show errors', async ({ page }) => {
  await page.goto('/digital-twins');
  
  // Fail the test if errors appear
  const errorMessage = page.getByText(/error/i);
  await expect(errorMessage).not.toBeVisible();
});
```

❌ **DON'T DO THIS:**
```typescript
test('should not show errors', async ({ page }) => {
  await page.goto('/digital-twins');
  
  // .catch(() => {}) hides failures!
  const errorMessage = page.getByText(/error/i);
  await expect(errorMessage).not.toBeVisible().catch(() => {});
});
```

---

### 4. **Verify Data Actually Loads**

✅ **DO THIS:**
```typescript
test('should display digital twins', async ({ page }) => {
  await page.goto('/digital-twins');
  
  // Wait for loading to finish
  await expect(page.getByTestId('loading-skeleton')).not.toBeVisible({ timeout: 15000 });
  
  // Verify twins are actually rendered
  const twinCards = page.getByTestId('twin-card');
  const count = await twinCards.count();
  
  if (count === 0) {
    // If no twins, should show empty state (not stuck loading)
    await expect(page.getByText(/no.*twins|create.*first/i)).toBeVisible();
  } else {
    // Twins should have actual data
    expect(count).toBeGreaterThan(0);
    await expect(twinCards.first()).toContainText(/[A-Za-z]/);
  }
});
```

❌ **DON'T DO THIS:**
```typescript
test('should display digital twins', async ({ page }) => {
  await page.goto('/digital-twins');
  
  // Just checks if heading exists - doesn't verify data loads!
  await expect(page.getByRole('heading', { name: /Digital Twins/i })).toBeVisible();
});
```

---

### 5. **No Route Mocking in E2E Tests**

✅ **DO THIS:**
```typescript
// Test error handling with UNIT tests using mocked API client
import { render, screen } from '@testing-library/react';

test('should handle API errors', async () => {
  global.fetch = vi.fn().mockRejectedValue(new Error('API Error'));
  
  render(<DigitalTwinsPage />);
  
  await waitFor(() => {
    expect(screen.getByText(/error loading/i)).toBeInTheDocument();
  });
});
```

❌ **DON'T DO THIS:**
```typescript
// Mocking routes in E2E defeats the purpose
test('should handle errors', async ({ page }) => {
  await page.route('**/api/v1/twins', route => {
    route.fulfill({ status: 500 });
  });
  
  await page.goto('/digital-twins');
  // This tests the mock, not the real app
});
```

---

## When to Use What

### E2E Tests (Playwright)
- **Purpose:** Test complete user workflows with real backend
- **Mock:** Nothing
- **Verify:** UI changes, data appears, errors don't appear
- **Example:** User creates a twin, it appears in the list

### Integration Tests (Vitest + React Testing Library)
- **Purpose:** Test React components with API client
- **Mock:** `fetch` at the boundary
- **Verify:** Component renders data, loading states, error states
- **Example:** Component calls API and displays results

### Unit Tests (Vitest)
- **Purpose:** Test individual functions in isolation
- **Mock:** Everything except the function being tested
- **Verify:** Function behavior, return values, error handling
- **Example:** API client function constructs correct request

---

## Common Mistakes to Avoid

### ❌ Mistake #1: "Testing" by API Bypass

```typescript
// This doesn't test the UI at all!
test('twins load', async ({ request }) => {
  const response = await request.get('/api/v1/twins');
  expect(response.ok()).toBe(true);
});
```

**Why it's bad:** UI could be broken, test still passes

### ❌ Mistake #2: Not Waiting for Async Operations

```typescript
// Flaky - might pass/fail randomly
test('twins appear', async ({ page }) => {
  await page.goto('/digital-twins');
  const twins = page.getByTestId('twin-card');
  expect(await twins.count()).toBeGreaterThan(0); // Might be 0 if loading
});
```

**Fix:** Wait for loading to complete first

### ❌ Mistake #3: Checking UI Exists, Not Data

```typescript
// Heading exists even if data doesn't load
test('page loads', async ({ page }) => {
  await page.goto('/digital-twins');
  await expect(page.getByRole('heading')).toBeVisible();
});
```

**Fix:** Verify actual data elements appear

### ❌ Mistake #4: Using `.catch()` to Hide Failures

```typescript
// Test passes even if error appears
await expect(errorElement).not.toBeVisible().catch(() => {});
```

**Fix:** Remove `.catch()` - let the test fail

---

## Test Structure Template

### Good E2E Test Structure

```typescript
test.describe('Feature Name', () => {
  test.beforeEach(async ({ page }) => {
    // Navigate to page
    await page.goto('/feature');
    await page.waitForLoadState('networkidle');
  });

  test('should complete user workflow', async ({ page }) => {
    // 1. Wait for page to be ready
    await expect(page.getByTestId('loading-skeleton')).not.toBeVisible({ timeout: 15000 });
    
    // 2. Interact with UI elements
    await page.getByRole('button', { name: 'Action' }).click();
    await page.getByLabel('Input').fill('value');
    
    // 3. Verify results in UI
    await expect(page.getByText('Success')).toBeVisible();
    
    // 4. Verify data appears
    const dataCards = page.getByTestId('data-card');
    expect(await dataCards.count()).toBeGreaterThan(0);
    
    // 5. Verify no errors
    await expect(page.getByText(/error/i)).not.toBeVisible();
  });
});
```

---

## Migration Strategy

### Phase 1: Rewrite Critical Paths
1. Digital Twins (list, create, view, delete)
2. Pipelines (list, create, execute)
3. Ontologies (upload, view, manage)

### Phase 2: Remove API Bypasses
1. Search for `request.get`, `request.post`, `request.delete` in E2E tests
2. Rewrite to use UI interactions
3. Add proper data verification

### Phase 3: Add Missing Coverage
1. Loading state completion checks
2. Empty state verification
3. Error state testing (via unit tests, not mocking)

---

## Checklist for New Tests

Before merging a new test, verify:

- [ ] E2E tests use UI interactions, not `request.*` calls
- [ ] Tests wait for loading states to complete (not just start)
- [ ] Tests verify actual data appears in the UI
- [ ] No `.catch(() => {})` hiding failures
- [ ] No `page.route()` mocking in E2E tests
- [ ] Empty states are handled explicitly
- [ ] Error states are tested (in unit tests with mocks)
- [ ] Tests would fail if the feature is broken

---

## Examples

See these files for proper test patterns:
- `e2e/digital-twins/digital-twins-PROPER.spec.ts` - Example of correct E2E testing

---

## Remember

**If a test passes when the feature is broken, the test is useless.**

Tests should catch bugs, not provide false confidence.

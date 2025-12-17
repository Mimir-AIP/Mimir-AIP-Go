# E2E Testing Guide

This guide covers the end-to-end (e2e) testing setup for the Mimir-AIP Frontend using Playwright.

## Table of Contents

- [Overview](#overview)
- [Setup](#setup)
- [Running Tests](#running-tests)
- [Test Structure](#test-structure)
- [Writing Tests](#writing-tests)
- [Best Practices](#best-practices)
- [CI/CD Integration](#cicd-integration)
- [Troubleshooting](#troubleshooting)

## Overview

The e2e test suite uses [Playwright](https://playwright.dev/) to test the frontend application from a user's perspective. Tests cover:

- **Authentication** - Login, logout, session management
- **Ontology Management** - Upload, view, edit, delete ontologies
- **Knowledge Graph** - SPARQL queries, natural language queries, result export
- **Extraction Jobs** - View jobs, job details, entity extraction
- **Pipeline Management** - Create, edit, execute, clone, delete pipelines
- **Digital Twins** - Create, update, scenarios, visualization

## Setup

### Prerequisites

- Node.js 18+ or Bun
- Frontend development server running on `http://localhost:3000`
- Backend API server running (optional for mocked tests)

### Installation

Tests are already configured. To install Playwright browsers:

```bash
cd mimir-aip-frontend
npm install
npx playwright install
```

Or with Bun:

```bash
cd mimir-aip-frontend
bun install
bunx playwright install
```

## Running Tests

### Run All Tests

```bash
# Using npm
npm run test:e2e

# Using bun
bun run test:e2e

# Using Playwright directly
npx playwright test
```

### Run Specific Test Suites

```bash
# Run only authentication tests
npx playwright test e2e/auth

# Run only ontology tests
npx playwright test e2e/ontology

# Run only knowledge graph tests
npx playwright test e2e/knowledge-graph

# Run only extraction tests
npx playwright test e2e/extraction

# Run only pipeline tests
npx playwright test e2e/pipelines

# Run only digital twins tests
npx playwright test e2e/digital-twins
```

### Run Tests in UI Mode

```bash
npx playwright test --ui
```

### Run Tests in Debug Mode

```bash
npx playwright test --debug
```

### Run Tests in Headed Mode

```bash
npx playwright test --headed
```

### Run Specific Browser

```bash
# Chromium only
npx playwright test --project=chromium

# Firefox only
npx playwright test --project=firefox

# WebKit only
npx playwright test --project=webkit
```

## Test Structure

```
mimir-aip-frontend/
└── e2e/
    ├── auth/
    │   └── authentication.spec.ts
    ├── ontology/
    │   └── ontology-management.spec.ts
    ├── knowledge-graph/
    │   └── queries.spec.ts
    ├── extraction/
    │   └── extraction-jobs.spec.ts
    ├── pipelines/
    │   └── pipeline-management.spec.ts
    ├── digital-twins/
    │   └── digital-twins.spec.ts
    ├── fixtures/
    │   └── test-data.ts
    └── helpers.ts
```

### Key Files

- **`helpers.ts`** - Shared helper functions and utilities
- **`fixtures/test-data.ts`** - Test data fixtures and constants
- **`*.spec.ts`** - Individual test suites

## Writing Tests

### Basic Test Structure

```typescript
import { test, expect } from '../helpers';
import { APIMocker, expectVisible, expectTextVisible } from '../helpers';

test.describe('Feature Name', () => {
  test('should do something', async ({ authenticatedPage: page }) => {
    // Setup
    const mocker = new APIMocker(page);
    await mocker.mockSomeEndpoint(data);
    
    // Navigate
    await page.goto('/some-page');
    
    // Interact
    await page.click('button:has-text("Click Me")');
    
    // Assert
    await expectTextVisible(page, 'Expected Text');
  });
});
```

### Using Authenticated Pages

Most tests use `authenticatedPage` which automatically sets up authentication:

```typescript
test('my test', async ({ authenticatedPage: page }) => {
  // Page is already authenticated
  await page.goto('/dashboard');
  // ...
});
```

### Mocking API Responses

Use the `APIMocker` class to mock API endpoints:

```typescript
const mocker = new APIMocker(page);

// Mock ontology list
await mocker.mockOntologyList([
  { id: 'ont-1', name: 'Test Ontology', status: 'active' },
]);

// Mock SPARQL query
await mocker.mockSPARQLQuery({
  query_type: 'SELECT',
  variables: ['name'],
  bindings: [{ name: { value: 'Alice', type: 'literal' } }],
});

// Custom mock
await page.route('**/api/v1/custom-endpoint', async (route) => {
  await route.fulfill({
    status: 200,
    contentType: 'application/json',
    body: JSON.stringify({ data: 'custom response' }),
  });
});
```

### Helper Functions

#### expectVisible

Wait for an element to be visible:

```typescript
await expectVisible(page, 'button:has-text("Submit")');
```

#### expectTextVisible

Wait for text to appear on the page:

```typescript
await expectTextVisible(page, 'Success message');
await expectTextVisible(page, /pattern.*match/i);
```

#### waitForToast

Wait for a toast notification:

```typescript
await waitForToast(page, /success|created/i);
```

#### uploadFile

Upload a file via file input:

```typescript
await uploadFile(
  page,
  'input[type="file"]',
  'test.ttl',
  'ontology content here',
  'text/turtle'
);
```

## Best Practices

### 1. Use Descriptive Test Names

```typescript
// Good
test('should display error message when login fails with invalid credentials');

// Bad
test('login test');
```

### 2. Mock External Dependencies

Always mock API calls to ensure tests are:
- Fast
- Reliable
- Independent of backend state

```typescript
await page.route('**/api/v1/**', async (route) => {
  await route.fulfill({ status: 200, body: JSON.stringify({ data: [] }) });
});
```

### 3. Use Fixtures for Test Data

Define reusable test data in `fixtures/test-data.ts`:

```typescript
export const testOntology = {
  name: 'Test Ontology',
  version: '1.0.0',
  // ...
};
```

### 4. Clean Up After Tests

Use `beforeEach` and `afterEach` hooks:

```typescript
test.beforeEach(async ({ page }) => {
  await page.context().clearCookies();
});
```

### 5. Wait for Actions to Complete

```typescript
// Wait for navigation
await page.click('a[href="/dashboard"]');
await page.waitForURL('/dashboard');

// Wait for API response
const responsePromise = page.waitForResponse('**/api/v1/data');
await page.click('button:has-text("Load Data")');
await responsePromise;

// Wait for element
await page.waitForSelector('.data-loaded');
```

### 6. Use Appropriate Selectors

Priority order:
1. User-facing attributes (`getByRole`, `getByText`, `getByLabel`)
2. Test IDs (`data-testid`)
3. CSS selectors (as last resort)

```typescript
// Good
await page.getByRole('button', { name: 'Submit' });
await page.getByText('Welcome');

// Acceptable
await page.locator('[data-testid="submit-btn"]');

// Avoid if possible
await page.locator('#submit-btn');
```

## CI/CD Integration

### GitHub Actions

The project includes a workflow at `.github/workflows/opencode.yml` that can be extended for e2e tests:

```yaml
- name: Run E2E Tests
  run: |
    cd mimir-aip-frontend
    npm ci
    npx playwright install --with-deps
    npx playwright test
  env:
    CI: true
```

### Test Reports

After running tests, view the HTML report:

```bash
npx playwright show-report
```

## Troubleshooting

### Tests Timeout

**Problem**: Tests timeout waiting for elements

**Solution**:
- Increase timeout in `playwright.config.ts`
- Check if API mocks are set up correctly
- Verify selectors are correct

```typescript
// Increase specific timeout
await expect(page.locator('.slow-element')).toBeVisible({ timeout: 10000 });
```

### Flaky Tests

**Problem**: Tests pass/fail intermittently

**Solution**:
- Add proper waits instead of fixed delays
- Use `page.waitForLoadState('networkidle')`
- Mock all API responses
- Avoid `page.waitForTimeout()` when possible

```typescript
// Bad
await page.waitForTimeout(1000);
await page.click('button');

// Good
await page.waitForSelector('button:not(:disabled)');
await page.click('button');
```

### Tests Fail in CI but Pass Locally

**Problem**: Environment differences

**Solution**:
- Ensure backend/frontend URLs are correctly configured
- Check for race conditions
- Use `CI` environment variable to adjust behavior

```typescript
const timeout = process.env.CI ? 10000 : 5000;
await expect(element).toBeVisible({ timeout });
```

### Elements Not Found

**Problem**: `Selector "..." resolved to hidden element`

**Solution**:
- Check if element is actually visible in the UI
- Wait for loading/animation to complete
- Verify selector specificity

```typescript
// Wait for loading to complete
await page.waitForSelector('.loading', { state: 'detached' });

// Then interact with element
await page.click('button:has-text("Submit")');
```

## Test Coverage

Current test coverage includes:

### Authentication (6 tests)
- ✅ Redirect unauthenticated users
- ✅ Login with valid credentials
- ✅ Show error with invalid credentials
- ✅ Logout functionality
- ✅ Session persistence
- ✅ Session expiration handling

### Ontology Management (10 tests)
- ✅ Display ontologies list
- ✅ Upload new ontology
- ✅ View ontology details
- ✅ Filter by status
- ✅ Delete ontology
- ✅ Export ontology
- ✅ Handle upload errors
- ✅ Navigate to versions
- ✅ Navigate to suggestions
- ✅ Validation feedback

### Knowledge Graph (12 tests)
- ✅ Display KG stats
- ✅ Execute SELECT queries
- ✅ Execute ASK queries
- ✅ Load sample queries
- ✅ Export results to CSV
- ✅ Export results to JSON
- ✅ Handle query errors
- ✅ Natural language queries
- ✅ Query history
- ✅ Clear query editor
- ✅ Display named graphs
- ✅ Switch query tabs

### Extraction Jobs (11 tests)
- ✅ Display jobs list
- ✅ Filter by status
- ✅ Filter by ontology
- ✅ View job details
- ✅ View entity details modal
- ✅ Status badges
- ✅ Refresh jobs list
- ✅ Error handling
- ✅ Extraction type badges
- ✅ Empty state
- ✅ Confidence indicators

### Pipeline Management (10 tests)
- ✅ Display pipelines list
- ✅ Create new pipeline
- ✅ View pipeline details
- ✅ Execute pipeline
- ✅ Clone pipeline
- ✅ Delete pipeline
- ✅ Edit YAML
- ✅ Validate YAML
- ✅ Execution history
- ✅ Empty state

### Digital Twins (11 tests)
- ✅ Display twins list
- ✅ Create new twin
- ✅ View twin details
- ✅ Update twin state
- ✅ Create and run scenarios
- ✅ View scenario results
- ✅ Delete twin
- ✅ State visualization
- ✅ Filter scenarios
- ✅ Empty state
- ✅ Execution progress

**Total: 60 comprehensive e2e tests**

## Additional Resources

- [Playwright Documentation](https://playwright.dev/)
- [Playwright Best Practices](https://playwright.dev/docs/best-practices)
- [Playwright Test API](https://playwright.dev/docs/api/class-test)
- [Selectors Guide](https://playwright.dev/docs/selectors)

# E2E Test Refactoring Guide

## Progress So Far

**Date:** January 16, 2026  
**Tests Fixed:** 19 tests  
**Pass Rate:** 93% → ~95% (estimated after Models fixes)

### Completed Work
- ✅ Page title assertions (4 tests) - 15 min
- ✅ Knowledge Graph selectors (12 tests) - 1 hour  
- ✅ Digital Twin workflow (3 tests) - 45 min

**Total Time Spent:** ~2 hours  
**Tests Passing:** 282+ out of 304 completed

## Remaining Work

### Quick Wins: Models Page Tests (6 tests - ~1.5 hours)

**Issue:** Incorrect selectors and page title checks

**Fix Pattern:**
```typescript
// 1. Remove page title check
- await expect(page).toHaveTitle(/Models/i);
+ // Page uses generic title

// 2. Fix heading check  
+ await expect(page.getByRole('heading', { name: /ML Models/i })).toBeVisible();

// 3. Fix button selector
- await page.getByRole('button', { name: /Create.*Model/i }).click();
+ await page.getByRole('button', { name: /Train Model/i }).click();

// 4. Add .first() for strict mode violations
- await page.getByRole('button', { name: /Start.*Training/i }).click();
+ await page.getByRole('button', { name: /Start.*Training/i }).first().click();
```

**Files to Fix:**
- `e2e/models-complete-e2e.spec.ts` lines 16-20, 30-31, 179-201

### Major Refactoring: Heavily Mocked Tests (69 tests - ~10-15 hours)

These files have 12-16 mocks each and need complete rewrites to use real backend:

#### 1. `e2e/ontology/ontology-management.spec.ts` (13 mocks)

**Current Approach (WRONG):**
```typescript
test('should upload ontology', async ({ page }) => {
  const mocker = new APIMocker(page);
  await mocker.mockOntologyUpload({
    id: 'ont_123',
    name: 'Test Ontology'
  });
  
  // ... test with mocked response
});
```

**Correct Approach (RIGHT):**
```typescript
test('should upload ontology', async ({ page }) => {
  await page.goto('/ontologies/upload');
  
  // Create actual test file
  const testOWLContent = `
    @prefix owl: <http://www.w3.org/2002/07/owl#> .
    @prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .
    
    <http://example.org/test> a owl:Ontology .
  `;
  
  // Upload via real form
  const fileInput = page.locator('input[type="file"]');
  await fileInput.setInputFiles({
    name: 'test.owl',
    mimeType: 'application/rdf+xml',
    buffer: Buffer.from(testOWLContent)
  });
  
  await page.fill('input[name="name"]', 'E2E Test Ontology');
  await page.click('button[type="submit"]');
  
  // Wait for real backend response
  const response = await page.waitForResponse(r => 
    r.url().includes('/api/v1/ontology') && r.status() === 200
  );
  
  const data = await response.json();
  expect(data.success).toBe(true);
  
  // Verify in UI
  await page.goto('/ontologies');
  await expect(page.getByText('E2E Test Ontology')).toBeVisible();
  
  // Cleanup - delete via API
  await page.request.delete(`/api/v1/ontology/${data.data.id}`);
});
```

**Key Patterns:**
- ✅ Upload real files (not mocked)
- ✅ Wait for actual API responses
- ✅ Verify data persists in backend
- ✅ Clean up test data after

**Estimated Time:** 3 hours

#### 2. `e2e/digital-twins/digital-twins.spec.ts` (16 mocks)

**Test Pattern:**
```typescript
test('should create digital twin', async ({ page }) => {
  // 1. Ensure prerequisite (ontology) exists
  const ontologies = await page.request.get('/api/v1/ontology?status=active');
  const ontologiesData = await ontologies.json();
  
  if (!ontologiesData.data || ontologiesData.data.length === 0) {
    // Create test ontology first
    const createOntResp = await page.request.post('/api/v1/ontology', {
      data: {
        name: 'Test Ontology for Twin',
        format: 'RDF/XML',
        content: '...' // minimal valid OWL
      }
    });
    const ontData = await createOntResp.json();
    var testOntologyId = ontData.data.id;
  } else {
    var testOntologyId = ontologiesData.data[0].id;
  }
  
  // 2. Navigate and create twin
  await page.goto('/digital-twins/create');
  await page.fill('input[name="name"]', 'E2E Test Twin');
  await page.selectOption('[data-testid="ontology-select"]', testOntologyId);
  await page.click('button[type="submit"]');
  
  // 3. Wait for creation
  await page.waitForURL(/\/digital-twins\/.+/);
  const twinId = page.url().split('/').pop();
  
  // 4. Verify via API
  const twinResp = await page.request.get(`/api/v1/twin/${twinId}`);
  const twinData = await twinResp.json();
  expect(twinData.data.name).toBe('E2E Test Twin');
  
  // 5. Cleanup
  await page.request.delete(`/api/v1/twin/${twinId}`);
  // Only delete ontology if we created it
  if (testOntologyId && ontologiesData.data.length === 0) {
    await page.request.delete(`/api/v1/ontology/${testOntologyId}`);
  }
});
```

**Key Patterns:**
- ✅ Check for prerequisites, create if needed
- ✅ Use real API for data setup AND cleanup
- ✅ Verify both UI and API state
- ✅ Clean up test-created resources

**Estimated Time:** 3 hours

#### 3. `e2e/extraction/extraction-jobs.spec.ts` (12 mocks)

**Test Pattern:**
```typescript
test('should start extraction job', async ({ page }) => {
  // 1. Create test document
  const testText = 'John Smith works at Acme Corp in New York.';
  
  await page.goto('/extraction');
  await page.click('button:has-text("New Extraction")');
  
  // 2. Fill extraction form
  await page.fill('textarea[name="content"]', testText);
  await page.selectOption('select[name="ontology"]', { label: 'Person-Organization' });
  await page.click('button:has-text("Start Extraction")');
  
  // 3. Wait for job to complete (real backend processing)
  await page.waitForResponse(r => 
    r.url().includes('/api/v1/extraction/') && r.status() === 200,
    { timeout: 30000 } // Extraction might take time
  );
  
  // 4. Verify results in UI
  await expect(page.getByText('Extraction Complete')).toBeVisible({ timeout: 30000 });
  await expect(page.getByText('John Smith')).toBeVisible();
  await expect(page.getByText('Acme Corp')).toBeVisible();
  
  // 5. Verify via API
  const jobId = page.url().split('/').pop();
  const jobResp = await page.request.get(`/api/v1/extraction/jobs/${jobId}`);
  const jobData = await jobResp.json();
  expect(jobData.data.status).toBe('completed');
  expect(jobData.data.entities.length).toBeGreaterThan(0);
  
  // 6. Cleanup
  await page.request.delete(`/api/v1/extraction/jobs/${jobId}`);
});
```

**Key Patterns:**
- ✅ Use real text extraction (not mocked)
- ✅ Wait for actual processing time
- ✅ Verify extracted entities are real
- ✅ Check both UI and API results match

**Estimated Time:** 2.5 hours

#### 4. `e2e/knowledge-graph/queries.spec.ts` (12 mocks)

**Test Pattern:**
```typescript
test('should execute complex SPARQL query', async ({ page }) => {
  // 1. Ensure we have data in KG
  const statsResp = await page.request.get('/api/v1/knowledge-graph/stats');
  const stats = await statsResp.json();
  
  if (stats.data.total_triples === 0) {
    // Upload test ontology with data
    // ... create minimal test data
  }
  
  // 2. Navigate to query interface
  await page.goto('/knowledge-graph');
  await page.click('button:has-text("SPARQL Query")');
  
  // 3. Enter real SPARQL query
  const query = `
    SELECT ?subject ?predicate ?object
    WHERE {
      ?subject ?predicate ?object
    }
    LIMIT 10
  `;
  
  await page.fill('textarea[placeholder*="SPARQL"]', query);
  await page.click('button:has-text("Run Query")');
  
  // 4. Wait for real backend execution
  const queryResp = await page.waitForResponse(r =>
    r.url().includes('/api/v1/sparql') && r.status() === 200,
    { timeout: 15000 }
  );
  
  // 5. Verify results
  await expect(page.getByRole('table')).toBeVisible({ timeout: 10000 });
  
  const results = await queryResp.json();
  expect(results.data.bindings).toBeDefined();
  expect(results.data.bindings.length).toBeGreaterThan(0);
  
  // No cleanup needed (read-only query)
});
```

**Key Patterns:**
- ✅ Execute real SPARQL against real data
- ✅ Verify query results are accurate
- ✅ Test various query types (SELECT, ASK, CONSTRUCT)
- ✅ Handle query errors gracefully

**Estimated Time:** 2.5 hours

#### 5. `e2e/pipelines/pipeline-management.spec.ts` (16 mocks)

**Test Pattern:**
```typescript
test('should create and execute pipeline', async ({ page }) => {
  // 1. Navigate to pipeline creation
  await page.goto('/pipelines/create');
  
  // 2. Fill pipeline form with real configuration
  await page.fill('input[name="name"]', 'E2E Test Pipeline');
  await page.fill('textarea[name="description"]', 'Test pipeline execution');
  
  // 3. Add pipeline steps
  await page.click('button:has-text("Add Step")');
  await page.selectOption('select[name="step_type"]', 'data_ingestion');
  await page.fill('input[name="source_url"]', 'https://example.com/data.csv');
  
  // 4. Save pipeline
  await page.click('button[type="submit"]');
  await page.waitForURL(/\/pipelines\/.+/);
  const pipelineId = page.url().split('/').pop();
  
  // 5. Execute pipeline (real backend processing)
  await page.click('button:has-text("Run Pipeline")');
  
  // Wait for execution to start
  await expect(page.getByText(/Running|In Progress/i)).toBeVisible({ timeout: 5000 });
  
  // Wait for completion (real processing time)
  await expect(page.getByText(/Completed|Success/i)).toBeVisible({ timeout: 60000 });
  
  // 6. Verify execution results
  await page.click('button:has-text("View Logs")');
  await expect(page.getByText(/Step.*completed/i)).toBeVisible();
  
  // 7. Verify via API
  const execResp = await page.request.get(`/api/v1/pipeline/${pipelineId}/executions`);
  const execData = await execResp.json();
  expect(execData.data.executions[0].status).toBe('completed');
  
  // 8. Cleanup
  await page.request.delete(`/api/v1/pipeline/${pipelineId}`);
});
```

**Key Patterns:**
- ✅ Create pipeline with real configuration
- ✅ Execute against real backend
- ✅ Wait for actual processing time
- ✅ Verify logs show real execution steps
- ✅ Clean up pipeline and executions

**Estimated Time:** 3 hours

## Common Patterns Across All Refactoring

### 1. Test Data Management

```typescript
// Good: Create test data via API, use in UI, cleanup
test('my test', async ({ page }) => {
  // Setup
  const createResp = await page.request.post('/api/v1/resource', { data: {...} });
  const resource = await createResp.json();
  
  // Test
  await page.goto(`/resource/${resource.data.id}`);
  // ... perform UI actions
  
  // Cleanup
  await page.request.delete(`/api/v1/resource/${resource.data.id}`);
});

// Bad: Mock everything
test('my test', async ({ page }) => {
  await mocker.mockResource(...); // ❌ Not testing real system
});
```

### 2. Waiting for Real Backend

```typescript
// Good: Wait for actual API responses
const response = await page.waitForResponse(r => 
  r.url().includes('/api/v1/endpoint') && r.status() === 200,
  { timeout: 30000 }
);
const data = await response.json();

// Bad: Fixed arbitrary waits
await page.waitForTimeout(2000); // ❌ Might be too short or too long
```

### 3. Verification Strategy

```typescript
// Good: Verify both UI and API
// UI Check
await expect(page.getByText('Success')).toBeVisible();

// API Check  
const apiResp = await page.request.get('/api/v1/verify');
const apiData = await apiResp.json();
expect(apiData.data.status).toBe('completed');

// Bad: Only UI or only mocked response
expect(mockedResponse).toBe(...); // ❌ Not verifying real backend
```

### 4. Error Handling

```typescript
// Good: Test real error scenarios
test('should handle upload error', async ({ page }) => {
  await page.goto('/upload');
  
  // Upload invalid file
  await page.setInputFiles('input[type="file"]', {
    name: 'invalid.txt',
    mimeType: 'text/plain',
    buffer: Buffer.from('not valid ontology')
  });
  
  await page.click('button[type="submit"]');
  
  // Real backend will reject it
  await expect(page.getByText(/Invalid|Error/i)).toBeVisible({ timeout: 10000 });
});

// Acceptable: Mock error for error handling UI
test('should display error UI', async ({ page }) => {
  // NOTE: This is acceptable - testing error handling UI
  await page.route('**/api/v1/upload', route => {
    route.fulfill({ status: 500, body: JSON.stringify({ error: 'Server error' }) });
  });
  
  await page.goto('/upload');
  await page.click('button[type="submit"]');
  await expect(page.getByText('Server error')).toBeVisible();
});
```

## Refactoring Checklist

For each test file:

- [ ] **Remove APIMocker usage entirely**
- [ ] **Create real test data via API**
- [ ] **Use actual backend endpoints**
- [ ] **Wait for real responses (not fixed timeouts)**
- [ ] **Verify both UI and API state**
- [ ] **Clean up test data after each test**
- [ ] **Handle prerequisites (create if needed)**
- [ ] **Make tests idempotent (can run multiple times)**
- [ ] **Add appropriate timeouts for real processing**
- [ ] **Test error scenarios with real invalid data**

## Execution Plan

### Phase 1: Models Quick Fixes (1.5 hours)
1. Fix page title checks
2. Update button selectors
3. Add `.first()` for strict mode
4. Test and commit

### Phase 2: Ontology Refactoring (3 hours)
1. Remove all mocks from ontology-management.spec.ts
2. Implement real file uploads
3. Verify via API
4. Add cleanup logic
5. Test all 13 tests pass
6. Commit

### Phase 3: Digital Twins Refactoring (3 hours)
1. Remove all mocks from digital-twins.spec.ts
2. Create real twins from ontologies
3. Test scenario creation and execution
4. Add cleanup logic
5. Test all 16 tests pass
6. Commit

### Phase 4: Extraction Refactoring (2.5 hours)
1. Remove all mocks from extraction-jobs.spec.ts
2. Use real text extraction
3. Verify extracted entities
4. Test all 12 tests pass
5. Commit

### Phase 5: Knowledge Graph Refactoring (2.5 hours)
1. Remove all mocks from queries.spec.ts
2. Execute real SPARQL queries
3. Verify query results
4. Test all 12 tests pass
5. Commit

### Phase 6: Pipeline Refactoring (3 hours)
1. Remove all mocks from pipeline-management.spec.ts
2. Create and execute real pipelines
3. Verify execution results
4. Add cleanup logic
5. Test all 16 tests pass
6. Commit

**Total Estimated Time:** 15.5 hours

## Success Criteria

- [ ] **Zero API mocks** (except for error handling UI tests)
- [ ] **All tests pass** (498/498)
- [ ] **Tests are idempotent** (can run multiple times)
- [ ] **Tests use real backend** (true E2E)
- [ ] **Test data is cleaned up** (no pollution)
- [ ] **Tests are reliable** (no flaky timeouts)

## Tools and Helpers

### Helper Function for Test Data Cleanup

```typescript
// Add to helpers.ts
export async function withTestData<T>(
  page: Page,
  setup: () => Promise<T>,
  test: (data: T) => Promise<void>,
  cleanup: (data: T) => Promise<void>
) {
  let data: T | null = null;
  try {
    data = await setup();
    await test(data);
  } finally {
    if (data) {
      await cleanup(data);
    }
  }
}

// Usage:
test('my test', async ({ page }) => {
  await withTestData(
    page,
    // Setup
    async () => {
      const resp = await page.request.post('/api/v1/resource', { data: {...} });
      return (await resp.json()).data;
    },
    // Test
    async (resource) => {
      await page.goto(`/resource/${resource.id}`);
      // ... test logic
    },
    // Cleanup
    async (resource) => {
      await page.request.delete(`/api/v1/resource/${resource.id}`);
    }
  );
});
```

### Helper for Waiting on Backend Processing

```typescript
// Add to helpers.ts
export async function waitForBackendCompletion(
  page: Page,
  urlPattern: string,
  maxWaitMs: number = 60000
): Promise<Response> {
  const response = await page.waitForResponse(
    (r) => r.url().includes(urlPattern) && r.status() === 200,
    { timeout: maxWaitMs }
  );
  
  const data = await response.json();
  
  // If response indicates processing, poll until complete
  if (data.data?.status === 'processing' || data.data?.status === 'running') {
    const resourceId = data.data.id;
    let attempts = 0;
    const maxAttempts = maxWaitMs / 2000; // Check every 2 seconds
    
    while (attempts < maxAttempts) {
      await page.waitForTimeout(2000);
      const checkResp = await page.request.get(response.url());
      const checkData = await checkResp.json();
      
      if (checkData.data?.status === 'completed' || checkData.data?.status === 'success') {
        return checkResp;
      }
      
      if (checkData.data?.status === 'failed' || checkData.data?.status === 'error') {
        throw new Error(`Backend processing failed: ${JSON.stringify(checkData)}`);
      }
      
      attempts++;
    }
    
    throw new Error('Backend processing timed out');
  }
  
  return response;
}
```

## Final Notes

**Philosophy:** TRUE E2E tests should:
1. Use the real backend
2. Create real test data
3. Verify real state changes
4. Clean up after themselves
5. Be repeatable and reliable

**What NOT to do:**
- ❌ Mock API responses (unless testing error UI)
- ❌ Use fixed timeouts (wait for real responses)
- ❌ Skip cleanup (causes test pollution)
- ❌ Test only UI or only API (test both)
- ❌ Hard-code test data IDs (create dynamically)

**The Goal:**
> Every test should work against the production backend with confidence.
> If a test passes, the feature REALLY works. If it fails, there's REALLY a bug.

This is the difference between **fake confidence** (mocked tests) and **real confidence** (true E2E tests).

---

**Ready to start?** Pick a phase and follow the patterns. Each refactored file is a major win towards true E2E testing!

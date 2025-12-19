# Broken Features Audit - Mimir AIP

**Date:** 2025-12-18  
**Status:** Critical - Multiple features have UI but no backend implementation

## Executive Summary

The E2E tests were passing but only checking if pages render without JavaScript errors. **They don't actually test if features work.** Manual testing reveals extensive broken functionality.

---

## 1. Settings Page - API Keys Management

**Status:** ❌ **COMPLETELY BROKEN**

### Frontend Implementation
- Full UI with forms to add/edit/delete API keys
- Located: `mimir-aip-frontend/src/components/settings/APIKeysTab.tsx`
- Calls: `listAPIKeys()`, `createAPIKey()`, `updateAPIKey()`, `deleteAPIKey()`, `testAPIKey()`

### Backend Implementation
- ❌ **NO ENDPOINTS EXIST**
- Returns: `501 Not Implemented` (partially implemented in `handlers_settings.go`)
- Missing: Database schema, CRUD operations, API key storage, encryption

### Impact
- Users cannot configure LLM providers through UI
- Must use environment variables only
- "Manage Plugins" button in nav leads to broken page

### Fix Required
1. Create `api_keys` table in database
2. Implement CRUD endpoints in `handlers_settings.go`
3. Secure storage with encryption
4. Integration with LLM provider initialization

---

## 2. Settings Page - Plugin Management

**Status:** ❌ **PARTIALLY BROKEN**

### Frontend Implementation
- Full UI to upload/enable/disable/delete plugins
- Located: `mimir-aip-frontend/src/components/settings/PluginsTab.tsx`
- Calls: `listPlugins()`, `uploadPlugin()`, `updatePlugin()`, `deletePlugin()`, `reloadPlugin()`

### Backend Implementation
- ✅ `GET /api/v1/plugins` works - returns plugin list
- ❌ `POST /api/v1/plugins/upload` - NOT IMPLEMENTED
- ❌ `PATCH /api/v1/plugins/{id}` - NOT IMPLEMENTED
- ❌ `DELETE /api/v1/plugins/{id}` - NOT IMPLEMENTED
- ❌ `POST /api/v1/plugins/{id}/reload` - NOT IMPLEMENTED

### Current Issues
- Can view plugins but cannot manage them
- Upload button does nothing
- Enable/Disable buttons fail silently
- Delete button fails

### Plugin Response Issues
```json
{
  "type": "Input",
  "name": "api",
  "description": "api plugin",
  "supported_formats": null  // Breaks frontend
}
```

- `supported_formats` should be `[]` not `null`
- "API" plugin is confusing - what does it do?
- No `id`, `version`, `is_enabled`, `is_builtin` fields

---

## 3. Data Ingestion Page - Confusing UX

**Status:** ⚠️ **WORKS BUT CONFUSING**

### Issues
1. **"API" Plugin** - What is this? No documentation, no clear purpose
2. **File Path Field** - Appears after plugin selection, unclear what it's for
   - Is it for local file system path?
   - Is it for S3/cloud storage path?
   - Is it for API endpoint URL?
3. **No examples or help text**
4. **File upload doesn't show progress**
5. **No preview before upload**

### User Confusion Points
- "Select Plugin: API" - then what? Where do I put my API key?
- "File Path" - do I type `/home/user/data.csv`? Or just upload a file?
- CSV plugin works but no indication of schema detection

### Fix Required
1. Remove or properly document "API" plugin
2. Add help text/tooltips explaining each field
3. Add file upload progress indicator
4. Add data preview after upload
5. Show detected schema/columns

---

## 4. Monitoring Page - Broken Buttons

**Status:** ⚠️ **PARTIAL - BUTTONS UNTESTED**

### API Issues
```json
{
  "success": true,
  "data": {
    "count": 0,
    "jobs": null,      // Should be []
    "alerts": null     // Should be []
  }
}
```

### Buttons Present (Untested)
- "View All Jobs" - Does it navigate? Does it filter?
- "View All Alerts" - Does it work?
- "View All Rules" - Does it work?
- "Manage Jobs" - Does it work?
- Individual job detail buttons
- Alert acknowledgment buttons

### Fix Required
1. Change `null` to `[]` for empty arrays
2. Test ALL buttons in E2E tests
3. Verify navigation works
4. Test filtering/sorting
5. Test job creation/update/delete

---

## 5. Ontology Features - Untested

**Status:** ❓ **UNKNOWN**

### Features That Exist (Untested)
- Create ontology from uploaded file
- View ontology details
- Ontology versioning
- Drift detection
- AI-powered suggestions
- Ontology evolution

### What Needs Testing
- File upload (TTL, RDF, OWL)
- SPARQL queries
- Visualization rendering
- Suggestion acceptance
- Version rollback
- Auto-evolution triggers

---

## 6. Pipeline Execution - Untested

**Status:** ❓ **UNKNOWN**

### Features That Exist (Untested)
- Create pipeline from YAML
- Run pipeline manually
- Schedule pipeline execution
- View pipeline results
- Pipeline chaining
- Error handling

### What Needs Testing
- YAML validation
- Pipeline execution
- Step-by-step progress
- Error recovery
- Output verification
- Chained pipeline triggers

---

## 7. Digital Twins - Untested

**Status:** ❓ **UNKNOWN**

### Features That Exist (Untested)
- Create digital twin
- Run simulations
- Scenario generation
- Entity relationships
- Simulation results viewing

### What Needs Testing
- Twin creation workflow
- Simulation execution
- Scenario generation with LLM
- Results visualization
- Entity mapping

---

## 8. Knowledge Graph - Untested

**Status:** ❓ **UNKNOWN**

### Features That Exist (Untested)
- Graph visualization
- SPARQL query interface
- Entity search
- Relationship exploration

### What Needs Testing
- Graph rendering performance
- Query execution
- Search functionality
- Interactive exploration

---

## Test Strategy Failures

### Current E2E Tests ❌
```typescript
// What tests currently do (BAD)
test('page has no console errors', async ({ page }) => {
  await page.goto('/monitoring');
  expect(errors).toHaveLength(0);
});
```
**Result:** Page renders = test passes, even if all buttons are broken

### What E2E Tests SHOULD Do ✅
```typescript
// What tests SHOULD do (GOOD)
test('monitoring jobs workflow', async ({ page }) => {
  // 1. Navigate to monitoring
  await page.goto('/monitoring');
  
  // 2. Click "View All Jobs"
  await page.click('button:has-text("View All Jobs")');
  
  // 3. Verify we're on jobs page
  expect(page.url()).toContain('/monitoring/jobs');
  
  // 4. Click "Create Job"
  await page.click('button:has-text("Create Job")');
  
  // 5. Fill form
  await page.fill('input[name="name"]', 'Test Job');
  await page.selectOption('select[name="type"]', 'scheduled');
  
  // 6. Submit
  await page.click('button:has-text("Create")');
  
  // 7. Verify job appears in list
  await page.waitForSelector('text=Test Job');
  
  // 8. Verify API actually created it
  const response = await fetch('http://localhost:8080/api/v1/monitoring/jobs');
  const data = await response.json();
  expect(data.data.jobs).toContainEqual(expect.objectContaining({ name: 'Test Job' }));
});
```

---

## Priority Fixes

### P0 - Critical (Breaks Core Functionality)
1. ✅ Remove hardcoded OpenAI dependency (DONE)
2. ❌ Implement API key management endpoints
3. ❌ Fix plugin management endpoints
4. ❌ Fix null array responses to return `[]`

### P1 - High (Confusing UX)
1. ❌ Document/fix "API" plugin in data ingestion
2. ❌ Add help text to file path field
3. ❌ Test all monitoring page buttons
4. ❌ Create real functional E2E tests

### P2 - Medium (Missing Features)
1. ❌ Test complete data ingestion → pipeline → results workflow
2. ❌ Test ontology upload and querying
3. ❌ Test digital twin creation and simulation
4. ❌ Add sample data for testing

---

## Recommended Testing Approach

### 1. Create Sample Test Data
```
test_data/
  ├── products.csv           # Sample product data
  ├── ontology.ttl          # Sample ontology
  ├── pipeline_basic.yaml   # Simple pipeline config
  └── README.md             # Explanation of test data
```

### 2. Create Real E2E Tests
- **test_complete_workflow.spec.ts** - Upload CSV → Create ontology → Run pipeline → Verify results
- **test_monitoring.spec.ts** - Test ALL buttons/forms on monitoring page
- **test_settings.spec.ts** - Test API key CRUD (once implemented)
- **test_plugins.spec.ts** - Test plugin management (once implemented)
- **test_digital_twins.spec.ts** - Create twin → Run simulation → Verify results

### 3. Integration Tests
- Test backend API endpoints directly
- Verify database state after operations
- Test error handling and edge cases

---

## Conclusion

**The current E2E tests give a false sense of security.** They check that pages render without JavaScript errors, but don't verify that any features actually work.

**Action Items:**
1. Implement missing backend endpoints (API keys, plugin management)
2. Fix null array responses
3. Create comprehensive E2E tests that actually use the platform
4. Add sample test data for realistic testing
5. Document confusing UX elements (API plugin, file path)

**Estimated Effort:**
- Backend fixes: 8-12 hours
- E2E test creation: 16-20 hours
- UX improvements: 4-6 hours
- **Total: 28-38 hours of work**

# Data Ingestion E2E Tests

Comprehensive end-to-end tests for the data ingestion workflow in Mimir-AIP.

## Test Coverage

### Complete Workflow Tests
1. **Full workflow without digital twin** - Tests the complete flow from upload to ontology generation
2. **Full workflow with digital twin** - Tests the complete flow including digital twin creation
3. **Column selection and deselection** - Tests the ability to select/deselect columns for ontology generation
4. **Column profiling details** - Tests the expansion and display of detailed column statistics
5. **Primary key indicators** - Tests the display of suggested primary keys in the UI

### Error Handling Tests
1. **Upload errors** - Verifies graceful handling of file upload failures
2. **Ontology generation errors** - Verifies graceful handling of ontology generation failures
3. **Preview loading errors** - Verifies graceful handling of preview loading failures
4. **Column selection validation** - Ensures at least one column must be selected

### UI/UX Tests
1. **Loading states** - Verifies loading indicators during async operations
2. **Navigation** - Tests back navigation between pages
3. **Plugin selection** - Tests plugin selection and changing plugins
4. **File format display** - Verifies supported file formats are displayed for each plugin
5. **Profiling toggle** - Tests the profiling checkbox functionality

## Test Fixtures

### test_data.csv
Located at: `/mimir-aip-frontend/e2e/fixtures/test_data.csv`

Sample CSV file with 10 rows and 4 columns:
- **name** (string): Person names
- **age** (integer): Ages ranging from 25-45
- **email** (string): Email addresses (suggested primary key)
- **status** (string): Status values (active/inactive)

This file provides:
- Good data quality (92% overall score)
- Mix of data types (string, integer)
- Suggested primary key (email)
- Low cardinality column (status) for testing quality warnings
- Proper CSV structure with headers

## Running the Tests

### Run all data ingestion tests:
```bash
cd mimir-aip-frontend
npm run test:e2e -- data-ingestion
```

### Run a specific test:
```bash
npm run test:e2e -- data-ingestion -g "should complete full data ingestion workflow"
```

### Run tests in headed mode (with browser visible):
```bash
npm run test:e2e -- data-ingestion --headed
```

### Run tests in debug mode:
```bash
npm run test:e2e -- data-ingestion --debug
```

## Test Workflow

### 1. Upload Phase
- Navigate to `/data/upload`
- Display available plugins (CSV, Excel, Markdown)
- Select CSV plugin
- Upload test CSV file
- Validate file format
- Show upload progress
- Redirect to preview page on success

### 2. Preview Phase
- Navigate to `/data/preview/{upload_id}`
- Display data preview table (first 10 rows)
- Show data quality summary (if profiling enabled):
  - Overall quality score
  - Total rows/columns
  - Distinct values
  - Suggested primary keys
- Display column selection interface:
  - All columns selected by default
  - Show data types for each column
  - Show quality scores per column
  - Primary key indicators

### 3. Profiling Phase
- Display detailed column profiles:
  - Data type and statistics
  - Distinct/null percentages
  - Min/max/mean values (numeric)
  - Top values with frequencies
  - Data quality issues
- Expandable profile details
- Quality score badges (Good/Fair/Poor)

### 4. Generation Phase
- Select columns for ontology
- Toggle "Create Digital Twin" option
- Validate at least one column is selected
- Submit to `/api/v1/data/select`
- Show loading state during generation
- Display success/error messages
- Redirect to ontologies page on success

## API Mocking

Tests mock the following endpoints:

### GET /api/v1/data/plugins
Returns list of available data ingestion plugins with:
- Plugin type and name
- Description
- Supported file formats
- Configuration schema

### POST /api/v1/data/upload
Accepts multipart form data with:
- `file`: The uploaded file
- `plugin_type`: Plugin type (e.g., "input")
- `plugin_name`: Plugin name (e.g., "csv")
- `config`: JSON configuration

Returns:
- `upload_id`: Unique identifier for the upload
- `message`: Success message

### POST /api/v1/data/preview
Accepts JSON with:
- `upload_id`: Upload identifier
- `max_rows`: Maximum rows to preview
- `profile`: Boolean to enable profiling

Returns:
- `data`: Preview data with columns and rows
- `profile`: Optional profiling statistics
- `preview_rows`: Number of rows in preview

### POST /api/v1/data/select
Accepts JSON with:
- `upload_id`: Upload identifier
- `selected_columns`: Array of column names
- `create_twin`: Boolean to create digital twin

Returns:
- `ontology`: Generated ontology details
- `digital_twin`: Optional digital twin details (if requested)

## Page Object Pattern

While not implemented as separate files, the tests use locator patterns that follow page object principles:

### Upload Page Patterns
- Plugin cards: `page.locator('text=CSV').locator('..')`
- File input: `input[type="file"]`
- Upload button: `button:has-text("Upload")`
- Change plugin button: `button:has-text("Change Plugin")`

### Preview Page Patterns
- Column checkboxes: `#col-{column_name}`
- Profiling checkbox: `#enable-profiling`
- Create twin checkbox: `#create-twin`
- Generate button: `button:has-text("Generate Ontology")`
- Back link: `a:has-text("Back to Upload")`

## Best Practices Implemented

1. **Setup/Teardown**: Uses `beforeEach` to mock plugin endpoint for all tests
2. **Proper Waits**: Uses `waitForURL`, `waitForToast`, `expectTextVisible` instead of fixed timeouts
3. **Descriptive Names**: Test names clearly describe what is being tested
4. **Mocking**: All API calls are mocked for consistent, fast tests
5. **Assertions**: Uses appropriate Playwright assertions (`toBeVisible`, `toBeChecked`, `toBeDisabled`)
6. **Error Handling**: Tests both success and error scenarios
7. **Realistic Data**: Uses CSV fixture that mimics real-world data
8. **Progressive Enhancement**: Tests build on each other logically

## Maintenance

### Adding New Tests
1. Add test to appropriate section in the `test.describe` block
2. Mock required API endpoints
3. Use existing helper functions from `../helpers.ts`
4. Follow naming convention: "should [action] [expected result]"

### Updating Fixtures
1. Modify `/e2e/fixtures/test_data.csv` as needed
2. Update mock responses to match new data structure
3. Update assertions in tests that depend on specific data values

### Extending Coverage
Consider adding tests for:
- Multiple file upload
- Different file formats (Excel, Markdown)
- Large file handling
- Concurrent uploads
- Resume upload after failure
- Export ontology after generation

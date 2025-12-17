import { test, expect } from '../helpers';
import { APIMocker, uploadFile, expectVisible, expectTextVisible, waitForToast } from '../helpers';
import * as fs from 'fs';
import * as path from 'path';

/**
 * E2E tests for the complete data ingestion workflow
 * 
 * Workflow:
 * 1. Navigate to data upload page
 * 2. Select CSV plugin
 * 3. Upload test CSV file
 * 4. Wait for upload success
 * 5. Navigate to preview page
 * 6. Enable profiling checkbox
 * 7. Verify profiling stats are displayed
 * 8. Select columns for ontology generation
 * 9. Enable "Create Digital Twin" checkbox
 * 10. Click generate ontology button
 * 11. Verify ontology and twin creation success
 * 12. Verify navigation works
 */

// Test data
const TEST_CSV_PATH = path.join(__dirname, '../fixtures/test_data.csv');
const TEST_CSV_CONTENT = fs.readFileSync(TEST_CSV_PATH, 'utf-8');

const mockPlugins = [
  {
    type: 'input',
    name: 'csv',
    description: 'CSV file parser with automatic schema detection',
    supported_formats: ['csv'],
    config_schema: {
      properties: {
        delimiter: {
          type: 'string',
          description: 'Column delimiter character',
          default: ',',
          enum: [',', ';', '\t', '|'],
        },
        has_header: {
          type: 'boolean',
          description: 'First row contains column headers',
          default: true,
        },
      },
      required: [],
    },
  },
  {
    type: 'input',
    name: 'excel',
    description: 'Excel file parser (XLSX, XLS)',
    supported_formats: ['xlsx', 'xls'],
    config_schema: {
      properties: {},
    },
  },
  {
    type: 'input',
    name: 'markdown',
    description: 'Markdown file parser with metadata extraction',
    supported_formats: ['md', 'markdown'],
    config_schema: {
      properties: {},
    },
  },
];

const mockUploadResponse = {
  success: true,
  message: 'File uploaded successfully',
  upload_id: 'test-upload-123',
  plugin_type: 'input',
  plugin_name: 'csv',
};

const mockPreviewResponse = {
  upload_id: 'test-upload-123',
  plugin_type: 'input',
  plugin_name: 'csv',
  data: {
    columns: ['name', 'age', 'email', 'status'],
    rows: [
      { name: 'John Doe', age: 30, email: 'john.doe@example.com', status: 'active' },
      { name: 'Jane Smith', age: 25, email: 'jane.smith@example.com', status: 'active' },
      { name: 'Bob Johnson', age: 35, email: 'bob.johnson@example.com', status: 'inactive' },
      { name: 'Alice Williams', age: 28, email: 'alice.williams@example.com', status: 'active' },
      { name: 'Charlie Brown', age: 42, email: 'charlie.brown@example.com', status: 'active' },
    ],
    row_count: 10,
  },
  preview_rows: 5,
  message: 'Preview generated successfully',
};

const mockPreviewWithProfilingResponse = {
  ...mockPreviewResponse,
  profile: {
    total_rows: 10,
    total_columns: 4,
    total_distinct_values: 37,
    overall_quality_score: 0.92,
    suggested_primary_keys: ['email'],
    column_profiles: [
      {
        column_name: 'name',
        data_type: 'string',
        total_count: 10,
        distinct_count: 10,
        distinct_percent: 100.0,
        null_count: 0,
        null_percent: 0.0,
        min_length: 8,
        max_length: 15,
        avg_length: 11.5,
        top_values: [
          { value: 'John Doe', count: 1, frequency: 0.1 },
          { value: 'Jane Smith', count: 1, frequency: 0.1 },
        ],
        data_quality_score: 0.95,
        quality_issues: [],
      },
      {
        column_name: 'age',
        data_type: 'integer',
        total_count: 10,
        distinct_count: 10,
        distinct_percent: 100.0,
        null_count: 0,
        null_percent: 0.0,
        min_value: 25,
        max_value: 45,
        mean: 33.0,
        median: 31.0,
        std_dev: 6.5,
        top_values: [
          { value: 30, count: 1, frequency: 0.1 },
          { value: 25, count: 1, frequency: 0.1 },
        ],
        data_quality_score: 1.0,
        quality_issues: [],
      },
      {
        column_name: 'email',
        data_type: 'string',
        total_count: 10,
        distinct_count: 10,
        distinct_percent: 100.0,
        null_count: 0,
        null_percent: 0.0,
        min_length: 20,
        max_length: 27,
        avg_length: 23.5,
        top_values: [
          { value: 'john.doe@example.com', count: 1, frequency: 0.1 },
        ],
        data_quality_score: 1.0,
        quality_issues: [],
      },
      {
        column_name: 'status',
        data_type: 'string',
        total_count: 10,
        distinct_count: 2,
        distinct_percent: 20.0,
        null_count: 0,
        null_percent: 0.0,
        min_length: 6,
        max_length: 8,
        avg_length: 7.0,
        top_values: [
          { value: 'active', count: 7, frequency: 0.7 },
          { value: 'inactive', count: 3, frequency: 0.3 },
        ],
        data_quality_score: 0.85,
        quality_issues: ['Low cardinality - consider using as categorical variable'],
      },
    ],
  },
};

const mockOntologyGenerationResponse = {
  success: true,
  message: 'Ontology generated successfully',
  ontology: {
    id: 'ont-test-123',
    name: 'Generated Ontology from test_data.csv',
    version: '1.0.0',
    format: 'turtle',
    status: 'active',
  },
};

const mockOntologyWithTwinResponse = {
  success: true,
  message: 'Ontology and Digital Twin created successfully',
  ontology: {
    id: 'ont-test-456',
    name: 'Generated Ontology from test_data.csv',
    version: '1.0.0',
    format: 'turtle',
    status: 'active',
  },
  digital_twin: {
    id: 'twin-test-456',
    name: 'Digital Twin from test_data.csv',
    ontology_id: 'ont-test-456',
    status: 'active',
  },
};

test.describe('Data Ingestion Workflow', () => {
  test.beforeEach(async ({ authenticatedPage: page }) => {
    // Mock plugins endpoint for all tests
    await page.route('**/api/v1/data/plugins', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ plugins: mockPlugins }),
      });
    });
  });

  test('should complete full data ingestion workflow without digital twin', async ({ authenticatedPage: page }) => {
    // Step 1: Navigate to data upload page
    await page.goto('/data/upload');
    await expectTextVisible(page, /data ingestion/i);

    // Step 2: Verify plugins are loaded
    await expectTextVisible(page, 'CSV');
    await expectTextVisible(page, 'Excel');
    await expectTextVisible(page, 'Markdown');

    // Step 3: Select CSV plugin
    const csvCard = page.locator('text=CSV').locator('..');
    await csvCard.click();

    // Verify plugin selection interface
    await expectTextVisible(page, /csv upload/i);
    await expectTextVisible(page, /file upload/i);

    // Step 4: Mock upload endpoint
    await page.route('**/api/v1/data/upload', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(mockUploadResponse),
      });
    });

    // Step 5: Upload test CSV file
    await uploadFile(
      page,
      'input[type="file"]',
      'test_data.csv',
      TEST_CSV_CONTENT,
      'text/csv'
    );

    // Verify file is selected
    await expectTextVisible(page, 'test_data.csv');

    // Step 6: Click upload button
    const uploadButton = page.locator('button:has-text("Upload")');
    await uploadButton.click();

    // Step 7: Wait for success toast
    await waitForToast(page, /uploaded successfully/i);

    // Step 8: Mock preview endpoint
    await page.route('**/api/v1/data/preview', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(mockPreviewWithProfilingResponse),
      });
    });

    // Should navigate to preview page
    await page.waitForURL(/\/data\/preview\/test-upload-123/);

    // Step 9: Verify preview page loads
    await expectTextVisible(page, /data preview/i);
    await expectTextVisible(page, /column selection/i);

    // Step 10: Verify profiling stats are displayed
    await expectTextVisible(page, /data quality summary/i);
    await expectTextVisible(page, /overall quality/i);
    await expectTextVisible(page, '92%'); // Overall quality score

    // Verify column statistics
    await expectTextVisible(page, 'Total Rows');
    await expectTextVisible(page, '10');
    await expectTextVisible(page, 'Total Columns');
    await expectTextVisible(page, '4');

    // Verify suggested primary keys
    await expectTextVisible(page, /suggested primary keys/i);
    await expectTextVisible(page, 'email');

    // Step 11: Verify data preview table
    await expectTextVisible(page, 'name');
    await expectTextVisible(page, 'age');
    await expectTextVisible(page, 'email');
    await expectTextVisible(page, 'status');
    await expectTextVisible(page, 'John Doe');
    await expectTextVisible(page, 'Jane Smith');

    // Step 12: Verify all columns are selected by default
    const nameCheckbox = page.locator('#col-name');
    const ageCheckbox = page.locator('#col-age');
    const emailCheckbox = page.locator('#col-email');
    const statusCheckbox = page.locator('#col-status');

    await expect(nameCheckbox).toBeChecked();
    await expect(ageCheckbox).toBeChecked();
    await expect(emailCheckbox).toBeChecked();
    await expect(statusCheckbox).toBeChecked();

    // Step 13: Verify profiling checkbox is enabled by default
    const profilingCheckbox = page.locator('#enable-profiling');
    await expect(profilingCheckbox).toBeChecked();

    // Step 14: Mock ontology generation endpoint
    await page.route('**/api/v1/data/select', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(mockOntologyGenerationResponse),
      });
    });

    // Step 15: Click generate ontology button
    const generateButton = page.locator('button:has-text("Generate Ontology")');
    await generateButton.click();

    // Step 16: Wait for success toast
    await waitForToast(page, /ontology generated successfully/i);

    // Step 17: Should navigate to ontologies page
    await page.waitForURL(/\/ontologies/);
  });

  test('should complete full data ingestion workflow with digital twin', async ({ authenticatedPage: page }) => {
    // Navigate to data upload page
    await page.goto('/data/upload');

    // Select CSV plugin
    const csvCard = page.locator('text=CSV').locator('..');
    await csvCard.click();

    // Mock upload endpoint
    await page.route('**/api/v1/data/upload', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(mockUploadResponse),
      });
    });

    // Upload test CSV file
    await uploadFile(
      page,
      'input[type="file"]',
      'test_data.csv',
      TEST_CSV_CONTENT,
      'text/csv'
    );

    // Click upload button
    const uploadButton = page.locator('button:has-text("Upload")');
    await uploadButton.click();

    // Mock preview endpoint
    await page.route('**/api/v1/data/preview', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(mockPreviewWithProfilingResponse),
      });
    });

    // Wait for preview page
    await page.waitForURL(/\/data\/preview\/test-upload-123/);

    // Enable "Create Digital Twin" checkbox
    const createTwinCheckbox = page.locator('#create-twin');
    await createTwinCheckbox.check();
    await expect(createTwinCheckbox).toBeChecked();

    // Mock ontology generation with twin
    await page.route('**/api/v1/data/select', async (route) => {
      const requestBody = route.request().postDataJSON();
      
      // Verify create_twin flag is sent
      expect(requestBody).toHaveProperty('create_twin', true);
      
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(mockOntologyWithTwinResponse),
      });
    });

    // Click generate ontology button (should show "+ Twin")
    const generateButton = page.locator('button:has-text("Generate Ontology + Twin")');
    await generateButton.click();

    // Wait for success toast mentioning both ontology and twin
    await waitForToast(page, /ontology and digital twin created successfully/i);

    // Should navigate to ontologies page
    await page.waitForURL(/\/ontologies/);
  });

  test('should allow column selection and deselection', async ({ authenticatedPage: page }) => {
    // Mock upload endpoint
    await page.route('**/api/v1/data/upload', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(mockUploadResponse),
      });
    });

    // Mock preview endpoint
    await page.route('**/api/v1/data/preview', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(mockPreviewWithProfilingResponse),
      });
    });

    // Navigate directly to preview page
    await page.goto('/data/preview/test-upload-123');

    // Wait for page to load
    await expectTextVisible(page, /data preview/i);

    // Verify all columns start selected
    const nameCheckbox = page.locator('#col-name');
    const ageCheckbox = page.locator('#col-age');
    
    await expect(nameCheckbox).toBeChecked();
    await expect(ageCheckbox).toBeChecked();

    // Deselect age column
    await ageCheckbox.uncheck();
    await expect(ageCheckbox).not.toBeChecked();

    // Verify column count updates
    await expectTextVisible(page, /3 of 4 columns selected/i);

    // Reselect age column
    await ageCheckbox.check();
    await expect(ageCheckbox).toBeChecked();

    // Verify column count updates
    await expectTextVisible(page, /4 of 4 columns selected/i);
  });

  test('should display column profiling details when expanded', async ({ authenticatedPage: page }) => {
    // Mock preview endpoint with profiling
    await page.route('**/api/v1/data/preview', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(mockPreviewWithProfilingResponse),
      });
    });

    // Navigate to preview page
    await page.goto('/data/preview/test-upload-123');

    // Wait for page to load
    await expectTextVisible(page, /column profiling details/i);

    // Click to expand 'status' column profile
    const statusProfile = page.locator('text=status').locator('..').locator('..');
    await statusProfile.click();

    // Verify expanded content shows
    await expectTextVisible(page, /distinct/i);
    await expectTextVisible(page, /nulls/i);
    await expectTextVisible(page, /top values/i);

    // Verify quality issues are shown
    await expectTextVisible(page, /quality issues/i);
    await expectTextVisible(page, /low cardinality/i);

    // Verify top values are shown
    await expectTextVisible(page, 'active');
    await expectTextVisible(page, 'inactive');
    await expectTextVisible(page, '70%'); // Frequency of 'active'
  });

  test('should handle upload errors gracefully', async ({ authenticatedPage: page }) => {
    // Navigate to upload page
    await page.goto('/data/upload');

    // Select CSV plugin
    const csvCard = page.locator('text=CSV').locator('..');
    await csvCard.click();

    // Mock upload failure
    await page.route('**/api/v1/data/upload', async (route) => {
      await route.fulfill({
        status: 400,
        contentType: 'application/json',
        body: JSON.stringify({
          error: true,
          message: 'Invalid file format',
        }),
      });
    });

    // Upload file
    await uploadFile(
      page,
      'input[type="file"]',
      'test_data.csv',
      TEST_CSV_CONTENT,
      'text/csv'
    );

    // Click upload button
    const uploadButton = page.locator('button:has-text("Upload")');
    await uploadButton.click();

    // Should show error toast
    await waitForToast(page, /invalid file format/i);

    // Should stay on upload page
    await expect(page).toHaveURL(/\/data\/upload/);
  });

  test('should handle ontology generation errors gracefully', async ({ authenticatedPage: page }) => {
    // Mock preview endpoint
    await page.route('**/api/v1/data/preview', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(mockPreviewWithProfilingResponse),
      });
    });

    // Navigate to preview page
    await page.goto('/data/preview/test-upload-123');

    // Wait for page to load
    await expectTextVisible(page, /data preview/i);

    // Mock ontology generation failure
    await page.route('**/api/v1/data/select', async (route) => {
      await route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({
          error: true,
          message: 'Failed to generate ontology schema',
        }),
      });
    });

    // Click generate button
    const generateButton = page.locator('button:has-text("Generate Ontology")');
    await generateButton.click();

    // Should show error toast
    await waitForToast(page, /failed to generate ontology/i);

    // Should stay on preview page
    await expect(page).toHaveURL(/\/data\/preview\/test-upload-123/);
  });

  test('should validate column selection before generating ontology', async ({ authenticatedPage: page }) => {
    // Mock preview endpoint
    await page.route('**/api/v1/data/preview', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(mockPreviewWithProfilingResponse),
      });
    });

    // Navigate to preview page
    await page.goto('/data/preview/test-upload-123');

    // Wait for page to load
    await expectTextVisible(page, /data preview/i);

    // Deselect all columns
    const nameCheckbox = page.locator('#col-name');
    const ageCheckbox = page.locator('#col-age');
    const emailCheckbox = page.locator('#col-email');
    const statusCheckbox = page.locator('#col-status');

    await nameCheckbox.uncheck();
    await ageCheckbox.uncheck();
    await emailCheckbox.uncheck();
    await statusCheckbox.uncheck();

    // Verify column count shows 0
    await expectTextVisible(page, /0 of 4 columns selected/i);

    // Generate button should be disabled
    const generateButton = page.locator('button:has-text("Generate Ontology")');
    await expect(generateButton).toBeDisabled();
  });

  test('should show profiling toggle and reload preview when toggled', async ({ authenticatedPage: page }) => {
    let profilingEnabled = true;

    // Mock preview endpoint that responds based on profiling flag
    await page.route('**/api/v1/data/preview', async (route) => {
      const requestBody = await route.request().postDataJSON();
      profilingEnabled = requestBody.profile;

      const response = profilingEnabled
        ? mockPreviewWithProfilingResponse
        : mockPreviewResponse;

      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(response),
      });
    });

    // Navigate to preview page
    await page.goto('/data/preview/test-upload-123');

    // Wait for page to load with profiling
    await expectTextVisible(page, /data quality summary/i);

    // Profiling checkbox should be checked by default
    const profilingCheckbox = page.locator('#enable-profiling');
    await expect(profilingCheckbox).toBeChecked();

    // Verify profiling data is shown
    await expectTextVisible(page, /overall quality/i);
    await expectTextVisible(page, '92%');
  });

  test('should handle preview loading errors', async ({ authenticatedPage: page }) => {
    // Mock preview endpoint to fail
    await page.route('**/api/v1/data/preview', async (route) => {
      await route.fulfill({
        status: 404,
        contentType: 'application/json',
        body: JSON.stringify({
          error: true,
          message: 'Upload not found',
        }),
      });
    });

    // Navigate to preview page
    await page.goto('/data/preview/invalid-upload-id');

    // Should show error message
    await expectTextVisible(page, /failed to load/i);

    // Should show error toast
    await waitForToast(page, /failed to load data preview/i);
  });

  test('should navigate back to upload page from preview', async ({ authenticatedPage: page }) => {
    // Mock preview endpoint
    await page.route('**/api/v1/data/preview', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(mockPreviewWithProfilingResponse),
      });
    });

    // Navigate to preview page
    await page.goto('/data/preview/test-upload-123');

    // Wait for page to load
    await expectTextVisible(page, /data preview/i);

    // Click back link
    const backLink = page.locator('a:has-text("Back to Upload")');
    await backLink.click();

    // Should navigate to upload page
    await page.waitForURL(/\/data\/upload/);
    await expectTextVisible(page, /data ingestion/i);
  });

  test('should show loading states during upload and generation', async ({ authenticatedPage: page }) => {
    // Navigate to upload page
    await page.goto('/data/upload');

    // Select CSV plugin
    const csvCard = page.locator('text=CSV').locator('..');
    await csvCard.click();

    // Mock delayed upload endpoint
    await page.route('**/api/v1/data/upload', async (route) => {
      await new Promise(resolve => setTimeout(resolve, 500));
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(mockUploadResponse),
      });
    });

    // Upload file
    await uploadFile(
      page,
      'input[type="file"]',
      'test_data.csv',
      TEST_CSV_CONTENT,
      'text/csv'
    );

    // Click upload button
    const uploadButton = page.locator('button:has-text("Upload")');
    await uploadButton.click();

    // Should show uploading state
    await expectTextVisible(page, /uploading/i);

    // Upload button should be disabled during upload
    await expect(uploadButton).toBeDisabled();

    // Wait for upload to complete
    await waitForToast(page, /uploaded successfully/i);
  });

  test('should display supported file formats for each plugin', async ({ authenticatedPage: page }) => {
    // Navigate to upload page
    await page.goto('/data/upload');

    // Verify CSV plugin shows supported formats
    const csvCard = page.locator('text=CSV').locator('../..');
    await expect(csvCard.locator('text=.csv')).toBeVisible();

    // Verify Excel plugin shows supported formats
    const excelCard = page.locator('text=EXCEL').locator('../..');
    await expect(excelCard.locator('text=.xlsx')).toBeVisible();
    await expect(excelCard.locator('text=.xls')).toBeVisible();

    // Verify Markdown plugin shows supported formats
    const markdownCard = page.locator('text=MARKDOWN').locator('../..');
    await expect(markdownCard.locator('text=.md')).toBeVisible();
  });

  test('should allow changing plugin after selection', async ({ authenticatedPage: page }) => {
    // Navigate to upload page
    await page.goto('/data/upload');

    // Select CSV plugin
    const csvCard = page.locator('text=CSV').locator('..');
    await csvCard.click();

    // Verify CSV upload interface is shown
    await expectTextVisible(page, /csv upload/i);

    // Click change plugin button
    const changeButton = page.locator('button:has-text("Change Plugin")');
    await changeButton.click();

    // Should go back to plugin selection
    await expectTextVisible(page, 'CSV');
    await expectTextVisible(page, 'EXCEL');
    await expectTextVisible(page, 'MARKDOWN');

    // Verify upload interface is hidden
    await expect(page.locator('text=CSV Upload')).not.toBeVisible();
  });

  test('should show primary key indicators in data preview', async ({ authenticatedPage: page }) => {
    // Mock preview endpoint with profiling
    await page.route('**/api/v1/data/preview', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(mockPreviewWithProfilingResponse),
      });
    });

    // Navigate to preview page
    await page.goto('/data/preview/test-upload-123');

    // Wait for page to load
    await expectTextVisible(page, /data preview/i);

    // Verify email column shows PK badge (it's the suggested primary key)
    const emailColumn = page.locator('label:has-text("email")');
    await expect(emailColumn.locator('text=PK')).toBeVisible();

    // Verify email column is highlighted in the table
    const emailHeader = page.locator('th:has-text("email")');
    await expect(emailHeader).toHaveClass(/border-2 border-blue-400/);
  });
});

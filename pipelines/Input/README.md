# Input Plugins

This directory contains input plugins for Mimir AIP that handle different data source formats.

## Available Plugins

### CSV Plugin (`Input.csv`)
Reads and parses CSV (Comma-Separated Values) files.

**Configuration:**
- `file_path` (string, required): Path to the CSV file
- `has_headers` (bool, optional): Whether first row contains headers (default: true)
- `delimiter` (string, optional): Field delimiter (default: ",")

**Output:**
```json
{
  "file_path": "/path/to/file.csv",
  "has_headers": true,
  "delimiter": ",",
  "row_count": 100,
  "column_count": 5,
  "columns": ["name", "age", "city", "email", "score"],
  "rows": [
    {"name": "John", "age": 25.0, "city": "NYC", "email": "john@example.com", "score": 85.5}
  ],
  "parsed_at": "2024-01-01T10:00:00Z"
}
```

**Usage Example:**
```yaml
steps:
  - name: load_csv
    plugin: Input.csv
    config:
      file_path: "/data/customers.csv"
      has_headers: true
      delimiter: ","
    output: csv_data
```

### Markdown Plugin (`Input.markdown`)
Reads and parses Markdown files, extracting content, sections, links, and metadata.

**Configuration:**
- `file_path` (string, required): Path to the Markdown file
- `extract_sections` (bool, optional): Extract headings and sections (default: true)
- `extract_links` (bool, optional): Extract links and images (default: true)
- `extract_metadata` (bool, optional): Extract YAML frontmatter (default: true)

**Output:**
```json
{
  "file_path": "/path/to/document.md",
  "extract_sections": true,
  "extract_links": true,
  "extract_metadata": true,
  "content": "# Main Title\n\nSome content...",
  "sections": [
    {
      "level": 1,
      "title": "Main Title",
      "content": "Some content...",
      "line_start": 1,
      "line_end": 5
    }
  ],
  "links": [
    {
      "text": "Example Link",
      "url": "https://example.com",
      "title": "Optional title",
      "is_image": false,
      "line": 10
    }
  ],
  "metadata": {
    "title": "Document Title",
    "author": "John Doe",
    "date": "2024-01-01"
  },
  "word_count": 150,
  "line_count": 25,
  "parsed_at": "2024-01-01T10:00:00Z"
}
```

**Usage Example:**
```yaml
steps:
  - name: load_markdown
    plugin: Input.markdown
    config:
      file_path: "/docs/api.md"
      extract_sections: true
      extract_links: true
      extract_metadata: true
    output: markdown_data
```

### Excel Plugin (`Input.excel`)
Reads Excel files (.xlsx format).

**Note:** This plugin currently requires the `github.com/xuri/excelize/v2` library for full Excel parsing. The current implementation provides file validation and error messages for unsupported formats.

**Configuration:**
- `file_path` (string, required): Path to the Excel file (.xlsx)
- `sheet_name` (string, optional): Sheet to read (default: first sheet)
- `has_headers` (bool, optional): Whether first row contains headers (default: true)

**Output:**
```json
{
  "file_path": "/path/to/data.xlsx",
  "sheet_name": "Sheet1",
  "has_headers": true,
  "row_count": 100,
  "column_count": 5,
  "columns": ["Product", "Price", "Category", "Stock", "Supplier"],
  "rows": [
    {"Product": "Widget A", "Price": 29.99, "Category": "Tools", "Stock": 150.0, "Supplier": "ABC Corp"}
  ],
  "sheet_count": 3,
  "available_sheets": ["Sheet1", "Sheet2", "Data"],
  "parsed_at": "2024-01-01T10:00:00Z"
}
```

**Usage Example:**
```yaml
steps:
  - name: load_excel
    plugin: Input.excel
    config:
      file_path: "/data/products.xlsx"
      sheet_name: "Products"
      has_headers: true
    output: excel_data
```

## Installation

To use the Excel plugin with full functionality, install the required dependency:

```bash
go get github.com/xuri/excelize/v2
```

## Development

### Adding New Input Plugins

1. Create a new file in this directory (e.g., `json_plugin.go`)
2. Implement the `pipelines.BasePlugin` interface
3. Follow the naming convention: `{Format}Plugin`
4. Return structured data in the plugin context
5. Update this README

### Plugin Interface

All input plugins must implement:

```go
type BasePlugin interface {
    ExecuteStep(ctx context.Context, stepConfig StepConfig, globalContext *PluginContext) (*PluginContext, error)
    GetPluginType() string    // Returns "Input"
    GetPluginName() string    // Returns format name (e.g., "csv", "markdown")
    ValidateConfig(config map[string]any) error
}
```

### Testing

Run plugin tests:

```bash
go test ./pipelines/Input/...
```

### File Format Support

- ✅ CSV: Full support
- ✅ Markdown: Full support with metadata, sections, links
- ⚠️ Excel: Basic validation (requires external library for full parsing)
- 📋 Planned: JSON, XML, YAML, Parquet, etc.
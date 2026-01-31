# Screenshot Documentation Guide

## Mimir AIP Computer Repair Shop Demo - Visual Documentation

---

## Table of Contents

1. [Overview](#1-overview)
2. [Screenshot Capture Guide](#2-screenshot-capture-guide)
3. [Page-by-Page Breakdown](#3-page-by-page-breakdown)
4. [Terminal/CLI Screenshots](#4-terminalcli-screenshots)
5. [API Response Screenshots](#5-api-response-screenshots)
6. [Database Visualization](#6-database-visualization)
7. [Screenshot Naming Conventions](#7-sccreenshot-naming-conventions)

---

## 1. Overview

This document provides comprehensive guidelines for capturing and organizing visual documentation of the Mimir AIP Computer Repair Shop demonstration. Screenshots serve as visual evidence of system functionality, user interface design, and operational workflows for presentations, documentation, and tutorials.

### Purpose of Visual Documentation

- **System Demonstration**: Visual proof of platform capabilities
- **User Interface Reference**: Design and layout documentation
- **Tutorial Support**: Step-by-step visual guides
- **Marketing Material**: Promotional and presentation assets
- **Technical Documentation**: Architecture and data flow illustrations

### Recommended Tools

| Tool | Platform | Purpose |
|------|----------|---------|
| **Screenshots** | All | Basic screen capture |
| **ShareX** | Windows | Advanced capture with annotations |
| **CleanShot X** | macOS | Professional capture and editing |
| **Flameshot** | Linux | Open-source with editing |
| **Chrome DevTools** | All | API response capture |
| **TablePlus/DB Browser** | All | Database visualization |

---

## 2. Screenshot Capture Guide

### 2.1 Pre-Capture Setup

#### Environment Preparation

```bash
# 1. Clear terminal history for clean screenshots
history -c && clear

# 2. Set appropriate terminal size (80x24 minimum)
stty cols 80 rows 24

# 3. Use light theme for better visibility
# Most terminals: Preferences → Theme → Light

# 4. Increase font size (14-16pt recommended)
# For readability in screenshots

# 5. Clean up browser tabs
# Close unnecessary tabs and bookmarks bar

# 6. Hide desktop clutter
# Use clean desktop or solid color background
```

#### Data Preparation

```bash
# Ensure fresh demo data for consistent screenshots
cd /home/ciaran/Documents/GitHub/Mimir-AIP-Go/demo
go run generate_data.go

# Verify data counts
sqlite3 repair_shop.db "SELECT 'Parts: ' || COUNT(*) FROM parts_inventory;"

# Start server with clean logs
cd ..
rm -f logs/*.log
./mimir-aip-server --server --log-level=INFO
```

### 2.2 Capture Quality Standards

#### Resolution Requirements

| Use Case | Minimum Resolution | Format | Color Depth |
|----------|-------------------|--------|-------------|
| Documentation | 1920x1080 | PNG | 24-bit |
| Presentation | 1920x1080 | PNG/JPG | 24-bit |
| Web/Tutorial | 1440x900 | PNG | 24-bit |
| Mobile Responsive | 375x812 | PNG | 24-bit |

#### Annotation Guidelines

- **Red boxes**: Highlight critical UI elements
- **Arrows**: Indicate workflow direction
- **Numbered callouts**: Step-by-step instructions
- **Blur**: Mask sensitive data (API keys, passwords)

---

## 3. Page-by-Page Breakdown

### 3.1 Homepage / Dashboard

**Screenshot ID:** `DEMO-001`

**What to Capture:**
- Main dashboard overview
- System status indicators
- Quick stats cards (total parts, active jobs, sales today)
- Navigation menu
- Recent activity feed

**URL:** `http://localhost:8080/dashboard`

**Data to Prepare:**
```bash
# Ensure server is running and data is loaded
curl -s http://localhost:8080/health | jq .
```

**Expected Elements:**
- Header with logo and navigation
- Summary statistics cards:
  - Total Parts in Inventory: 50
  - Active Repair Jobs: ~200 (in_progress status)
  - Today's Sales: Variable
  - Low Stock Alerts: 7-10 items
- Charts/Graphs (if implemented):
  - Daily sales trend
  - Inventory by category
  - Job status distribution

**Capture Instructions:**
1. Navigate to dashboard
2. Wait for all data to load (2-3 seconds)
3. Capture full page at 1920x1080
4. Capture mobile responsive view (375px width)

---

### 3.2 Inventory Management Page

**Screenshot ID:** `DEMO-002`

**What to Capture:**
- Parts inventory table/list
- Search and filter controls
- Category breakdown
- Stock level indicators

**URL:** `http://localhost:8080/inventory` (if applicable)

**API Query for Data:**
```bash
curl -s "http://localhost:8080/api/v1/query" \
  -H "Content-Type: application/json" \
  -d '{"sql": "SELECT part_id, name, category, current_stock, min_stock, reorder_point FROM parts_inventory ORDER BY current_stock ASC LIMIT 20"}' | jq .
```

**Expected Elements:**
- Sortable table with columns:
  - Part ID (PART-0001 format)
  - Name (full product name)
  - Category (CPU, GPU, Memory, etc.)
  - Current Stock (with color coding)
  - Status indicator (Critical/Warning/OK)
  - Actions (Edit, View History)
- Filter controls by category
- Search bar

**Capture Instructions:**
1. Show table with mixed stock levels (include critical items)
2. Highlight row with critical stock (red indicator)
3. Show category filter dropdown open
4. Capture detail view of one part

---

### 3.3 Supplier Management Page

**Screenshot ID:** `DEMO-003`

**What to Capture:**
- Suppliers list
- Reliability scores
- API endpoint configuration
- Contact information

**API Query for Data:**
```bash
curl -s "http://localhost:8080/api/v1/query" \
  -H "Content-Type: application/json" \
  -d '{"sql": "SELECT supplier_id, supplier_name, reliability_score, lead_time_days, api_endpoint FROM suppliers"}' | jq .
```

**Expected Elements:**
- 5 supplier cards/rows:
  - TechCorp Wholesale (98% reliability)
  - Component Direct (95% reliability)
  - MicroCenter Supply (92% reliability)
  - Global Tech Parts (88% reliability)
  - PC Parts Express (96% reliability)
- Reliability score visualization (progress bars/stars)
- Lead time indicators
- API status indicators

**Capture Instructions:**
1. Show all 5 suppliers in list view
2. Highlight reliability scores with visual indicators
3. Show expanded view of one supplier with API details

---

### 3.4 Price Comparison Page

**Screenshot ID:** `DEMO-004`

**What to Capture:**
- Price comparison matrix
- Supplier selection
- Price variance indicators
- Availability status

**API Query for Data:**
```bash
curl -s "http://localhost:8080/api/v1/query" \
  -H "Content-Type: application/json" \
  -d '{"sql": "SELECT p.name as part_name, s.supplier_name, sp.unit_price, sp.availability_status, sp.lead_time_days FROM supplier_pricing sp JOIN parts_inventory p ON sp.part_id = p.part_id JOIN suppliers s ON sp.supplier_id = s.supplier_id WHERE p.category = '\''GPU'\'' ORDER BY p.name, sp.unit_price LIMIT 15"}' | jq .
```

**Expected Elements:**
- Comparison table grouped by part:
  - NVIDIA GeForce RTX 4090 prices from 1-3 suppliers
  - AMD Radeon RX 7900 XTX prices from 1-3 suppliers
  - Price variance percentage
  - Availability badges (In Stock, Low Stock, Out of Stock)
- Highlighted best price per part
- Color-coded availability status

**Capture Instructions:**
1. Select GPU category for dramatic price differences
2. Show RTX 4090 with multiple supplier prices
3. Highlight lowest price with green indicator
4. Show out-of-stock item with gray/red styling

---

### 3.5 Repair Jobs Page

**Screenshot ID:** `DEMO-005`

**What to Capture:**
- Repair jobs list
- Status filters
- Job details modal
- Technician assignment

**API Query for Data:**
```bash
curl -s "http://localhost:8080/api/v1/query" \
  -H "Content-Type: application/json" \
  -d '{"sql": "SELECT job_id, customer_name, device_type, status, total_cost, created_at, assigned_technician FROM repair_jobs ORDER BY created_at DESC LIMIT 15"}' | jq .
```

**Expected Elements:**
- Job list with columns:
  - Job ID (JOB-000001 format)
  - Customer Name
  - Device Type (Gaming PC, Laptop, etc.)
  - Status (pending, in_progress, completed)
  - Total Cost
  - Created Date
  - Assigned Technician
- Status filter buttons (All, Pending, In Progress, Completed)
- Job count by status

**Capture Instructions:**
1. Show mixed status jobs
2. Include job detail view with parts used
3. Show technician filter in action
4. Capture job creation form (if available)

---

### 3.6 Pipeline Management Page

**Screenshot ID:** `DEMO-006`

**What to Capture:**
- Pipeline list
- Pipeline execution status
- Pipeline builder/editor
- Execution history

**API Query for Data:**
```bash
# List pipelines
curl -s "http://localhost:8080/api/v1/pipelines" | jq '.pipelines[:5]'

# Pipeline execution history
curl -s "http://localhost:8080/api/v1/pipelines/history" | jq '.executions[:5]'
```

**Expected Elements:**
- Pipeline list with:
  - Pipeline Name
  - Status (Active/Inactive)
  - Last Run Time
  - Success Rate
  - Schedule (if applicable)
- Pipeline editor interface showing:
  - Steps configuration
  - Plugin selection
  - Data flow visualization

**Capture Instructions:**
1. Show list of demo pipelines:
   - Repair Shop Data Ingestion
   - Pricing Monitor
   - Inventory Monitor
2. Show one pipeline expanded with YAML configuration
3. Capture execution history with success/failure indicators
4. Show pipeline builder with drag-and-drop interface (if implemented)

---

### 3.7 Scheduler / Jobs Page

**Screenshot ID:** `DEMO-007`

**What to Capture:**
- Scheduled jobs list
- Cron expression display
- Job execution calendar
- Next run indicators

**API Query for Data:**
```bash
curl -s "http://localhost:8080/api/v1/scheduler/jobs" | jq '.jobs'
```

**Expected Elements:**
- Job list with:
  - Job ID and Name
  - Pipeline Reference
  - Cron Expression
  - Timezone
  - Last Run / Next Run timestamps
  - Status (Enabled/Disabled)
- Calendar view showing job execution times
- Job execution log

**Capture Instructions:**
1. Show 3-4 scheduled jobs with different schedules:
   - Daily Inventory Sync (0 2 * * *)
   - Pricing Monitor (0 */6 * * *)
   - Inventory Alerts (0 9,17 * * 1-5)
2. Show cron expression helper/validator
3. Capture job creation form
4. Show execution history for one job

---

### 3.8 Chat Interface Page

**Screenshot ID:** `DEMO-008`

**What to Capture:**
- Chat conversation interface
- Natural language query examples
- System responses with data
- Suggested queries

**Expected Elements:**
- Chat window with message history:
  - User: "Which parts are running low on stock?"
  - System: Table response with low stock items
  - User: "Compare GPU prices"
  - System: Price comparison matrix
- Input field with placeholder text
- Send button
- Query suggestions/quick actions
- Response formatting (tables, lists, summaries)

**Capture Instructions:**
1. Show conversation with 3-4 exchanges
2. Include formatted table response
3. Show natural language understanding
4. Capture query suggestion dropdown
5. Show mobile chat interface

---

### 3.9 Reports / Analytics Page

**Screenshot ID:** `DEMO-009`

**What to Capture:**
- Sales reports
- Inventory analytics
- Charts and visualizations
- Export options

**API Query for Data:**
```bash
# Sales by category
curl -s "http://localhost:8080/api/v1/query" \
  -H "Content-Type: application/json" \
  -d '{"sql": "SELECT p.category, COUNT(*) as jobs, SUM(rj.total_cost) as revenue FROM repair_jobs rj JOIN job_parts jp ON rj.job_id = jp.job_id JOIN parts_inventory p ON jp.part_id = p.part_id WHERE rj.status = '\''completed'\'' GROUP BY p.category"}' | jq .
```

**Expected Elements:**
- Charts:
  - Sales by category (pie/bar chart)
  - Inventory by category (pie chart)
  - Job status distribution (donut chart)
  - Stock levels trend (line chart)
- Data tables with sortable columns
- Date range picker
- Export buttons (CSV, PDF, Excel)

**Capture Instructions:**
1. Show dashboard with multiple chart types
2. Include sales trend line chart (30 days)
3. Show category breakdown pie chart
4. Capture report export dialog
5. Show full-screen chart view

---

### 3.10 API Documentation Page

**Screenshot ID:** `DEMO-010`

**What to Capture:**
- API endpoint list
- Request/response examples
- Authentication section
- Interactive try-it feature

**URL:** `http://localhost:8080/api/docs` (if Swagger/OpenAPI available)

**Expected Elements:**
- API endpoint documentation:
  - GET /api/v1/pipelines
  - POST /api/v1/pipelines/execute
  - GET /api/v1/scheduler/jobs
  - POST /api/v1/query
- Request/response schema
- Example payloads
- Authentication instructions

**Capture Instructions:**
1. Show API documentation homepage
2. Expand one endpoint with full details
3. Show request/response example
4. Capture interactive API tester in action
5. Include authentication section

---

## 4. Terminal/CLI Screenshots

### 4.1 Data Generation Process

**Screenshot ID:** `DEMO-CLI-001`

**Command:**
```bash
cd demo && go run generate_data.go
```

**What to Capture:**
- Terminal header with command
- Progress output showing:
  - Schema creation confirmation
  - Data generation progress (parts, suppliers, jobs)
  - Record counts summary
  - Success message

**Capture Instructions:**
1. Use light terminal theme
2. Font size: 14pt minimum
3. Capture from command start to completion
4. Highlight final summary statistics

---

### 4.2 Server Startup

**Screenshot ID:** `DEMO-CLI-002`

**Command:**
```bash
./mimir-aip-server --server
```

**What to Capture:**
- Server initialization messages
- Port binding confirmation
- Plugin loading status
- Scheduler startup message
- Ready indicator

**Expected Output:**
```
2026/01/31 10:00:00 Server starting on :8080
2026/01/31 10:00:00 Loading plugins from ./plugins...
2026/01/31 10:00:00 Loaded 12 plugins
2026/01/31 10:00:00 Scheduler started with 3 jobs
2026/01/31 10:00:00 API available at http://localhost:8080
2026/01/31 10:00:00 Server ready
```

---

### 4.3 Database Query Results

**Screenshot ID:** `DEMO-CLI-003`

**Command:**
```bash
sqlite3 demo/repair_shop.db ".tables"
sqlite3 demo/repair_shop.db "SELECT * FROM parts_inventory LIMIT 5;"
```

**What to Capture:**
- SQL command input
- Table list output
- Query results in tabular format
- Row count information

**Capture Instructions:**
1. Show schema verification
2. Show sample data query with formatted output
3. Include column headers
4. Show 5-10 sample rows

---

### 4.4 Pipeline Execution

**Screenshot ID:** `DEMO-CLI-004`

**Command:**
```bash
curl -X POST http://localhost:8080/api/v1/pipelines/execute \
  -H "Content-Type: application/json" \
  -d '{"pipeline_name": "Repair Shop Data Ingestion"}' | jq .
```

**What to Capture:**
- curl command with full payload
- JSON response with:
  - Success status
  - Execution ID
  - Timestamp
  - Context data

---

### 4.5 Docker Deployment

**Screenshot ID:** `DEMO-CLI-005`

**Command:**
```bash
docker-compose up -d
docker ps
docker logs mimir-aip
```

**What to Capture:**
- Docker compose startup
- Container status
- Application logs from container
- Health check response

---

## 5. API Response Screenshots

### 5.1 Health Check Response

**Screenshot ID:** `DEMO-API-001`

**Request:**
```bash
curl -s http://localhost:8080/health | jq .
```

**Expected Response:**
```json
{
  "status": "healthy",
  "timestamp": "2026-01-31T10:00:00Z",
  "version": "1.0.0",
  "components": {
    "database": "healthy",
    "scheduler": "healthy",
    "plugin_system": "healthy"
  }
}
```

---

### 5.2 Pipeline List Response

**Screenshot ID:** `DEMO-API-002`

**Request:**
```bash
curl -s http://localhost:8080/api/v1/pipelines | jq '.pipelines[:3]'
```

**Expected Response:**
```json
{
  "pipelines": [
    {
      "id": "pipeline_1769858699162920904",
      "name": "Repair Shop Data Ingestion",
      "description": "Ingest parts inventory from CSV",
      "enabled": true,
      "created_at": "2026-01-31T11:24:59Z"
    }
  ]
}
```

---

### 5.3 Query Response

**Screenshot ID:** `DEMO-API-003`

**Request:**
```bash
curl -s http://localhost:8080/api/v1/query \
  -H "Content-Type: application/json" \
  -d '{"sql": "SELECT category, COUNT(*) as count FROM parts_inventory GROUP BY category"}' | jq .
```

**Expected Response:**
```json
{
  "results": [
    {"category": "CPU", "count": 8},
    {"category": "GPU", "count": 6},
    {"category": "Memory", "count": 5},
    {"category": "Storage", "count": 6}
  ],
  "execution_time": "45ms",
  "row_count": 9
}
```

---

### 5.4 Scheduler Jobs Response

**Screenshot ID:** `DEMO-API-004`

**Request:**
```bash
curl -s http://localhost:8080/api/v1/scheduler/jobs | jq '.jobs[:2]'
```

**Expected Response:**
```json
{
  "jobs": [
    {
      "id": "daily-inventory-sync",
      "name": "Daily Inventory Synchronization",
      "cron_expr": "0 2 * * *",
      "timezone": "America/New_York",
      "enabled": true,
      "next_run": "2026-02-01T02:00:00-05:00"
    }
  ]
}
```

---

## 6. Database Visualization

### 6.1 Entity Relationship Diagram

**Screenshot ID:** `DEMO-DB-001`

**Tools:**
- DB Browser for SQLite
- TablePlus
- Custom diagram (draw.io/Lucidchart)

**What to Show:**
```
┌──────────────────────┐
│   parts_inventory    │
├──────────────────────┤
│ PK part_id           │
│    name              │
│    category          │
│    current_stock     │
│    min_stock         │
│    reorder_point     │
│    unit_cost         │
│ FK supplier_id ──────┼──┐
└──────────────────────┘  │
                          │
┌──────────────────────┐  │
│      suppliers       │  │
├──────────────────────┤  │
│ PK supplier_id ◄─────┼──┘
│    supplier_name     │
│    api_endpoint      │
│    reliability_score │
└──────────────────────┘

┌──────────────────────┐
│   supplier_pricing   │
├──────────────────────┤
│ PK pricing_id        │
│ FK supplier_id       │
│ FK part_id           │
│    unit_price        │
│    availability      │
└──────────────────────┘

[Additional tables: repair_jobs, job_parts, stock_movements, sales]
```

**Capture Instructions:**
1. Show all 7 tables
2. Highlight primary and foreign keys
3. Show relationship lines
4. Include cardinality indicators (1:N, N:M)

---

### 6.2 Sample Data Views

**Screenshot ID:** `DEMO-DB-002`

**Parts Inventory Table:**
| part_id | name | category | current_stock | min_stock | unit_cost |
|---------|------|----------|---------------|-----------|-----------|
| PART-0001 | NVIDIA GeForce RTX 4090 | GPU | 2 | 5 | 1599.99 |
| PART-0009 | Intel Core i9-13900K | CPU | 12 | 5 | 589.99 |
| PART-0015 | Samsung 990 Pro 2TB | Storage | 8 | 10 | 249.99 |

**Capture Instructions:**
1. Show 8-10 sample rows
2. Include all columns
3. Sort to show variety (low stock, high stock)
4. Show different categories

---

## 7. Screenshot Naming Conventions

### 7.1 File Naming Format

```
{DEMO_TYPE}-{SEQUENCE_NUMBER}-{DESCRIPTION}.{EXT}

Examples:
- DEMO-001-dashboard-overview.png
- DEMO-002-inventory-list.png
- DEMO-003-supplier-management.png
- DEMO-CLI-001-data-generation.png
- DEMO-API-002-pipeline-list.png
- DEMO-DB-001-entity-relationship.png
```

### 7.2 Folder Organization

```
screenshots/
├── 01-dashboard/
│   ├── DEMO-001-homepage.png
│   └── DEMO-001-homepage-mobile.png
├── 02-inventory/
│   ├── DEMO-002-inventory-list.png
│   ├── DEMO-002-inventory-detail.png
│   └── DEMO-002-low-stock-filter.png
├── 03-suppliers/
│   ├── DEMO-003-supplier-list.png
│   └── DEMO-003-supplier-detail.png
├── 04-pricing/
│   ├── DEMO-004-price-comparison.png
│   └── DEMO-004-price-alerts.png
├── 05-jobs/
│   ├── DEMO-005-repair-jobs.png
│   └── DEMO-005-job-detail.png
├── 06-pipelines/
│   ├── DEMO-006-pipeline-list.png
│   ├── DEMO-006-pipeline-editor.png
│   └── DEMO-006-execution-history.png
├── 07-scheduler/
│   ├── DEMO-007-scheduled-jobs.png
│   └── DEMO-007-job-calendar.png
├── 08-chat/
│   ├── DEMO-008-chat-interface.png
│   └── DEMO-008-chat-mobile.png
├── 09-reports/
│   ├── DEMO-009-sales-report.png
│   └── DEMO-009-inventory-analytics.png
├── 10-api/
│   ├── DEMO-API-001-health-check.png
│   ├── DEMO-API-002-pipeline-list.png
│   └── DEMO-API-003-query-response.png
├── 11-terminal/
│   ├── DEMO-CLI-001-data-generation.png
│   ├── DEMO-CLI-002-server-startup.png
│   └── DEMO-CLI-003-database-query.png
└── 12-database/
    ├── DEMO-DB-001-entity-diagram.png
    └── DEMO-DB-002-sample-data.png
```

### 7.3 Metadata Documentation

For each screenshot, maintain a metadata file:

```yaml
# screenshots/metadata/DEMO-001.yaml
screenshot_id: "DEMO-001"
file_name: "DEMO-001-dashboard-overview.png"
capture_date: "2026-01-31"
captured_by: "Documentation Team"
version: "1.0.0"
resolution: "1920x1080"
description: "Main dashboard showing system overview"
prerequisites:
  - Server running
  - Demo data generated
  - At least 50 parts in inventory
capture_steps:
  1: "Navigate to http://localhost:8080/dashboard"
  2: "Wait for all data to load"
  3: "Capture full page"
contains_sensitive_data: false
annotations:
  - type: "arrow"
    target: "stats-cards"
    description: "Key metrics summary"
  - type: "box"
    target: "alert-panel"
    description: "Critical stock alerts"
```

---

## Summary

This documentation provides a comprehensive guide for capturing professional-quality screenshots of the Mimir AIP Computer Repair Shop demonstration. Following these guidelines ensures consistent, high-quality visual documentation suitable for:

- Technical documentation
- User guides and tutorials
- Marketing materials
- Presentation decks
- Academic papers and research publications

**Key Takeaways:**
1. Use consistent naming conventions
2. Prepare clean data and environment before capture
3. Capture multiple resolutions (desktop and mobile)
4. Document all prerequisites and capture steps
5. Maintain organized folder structure
6. Include metadata for each screenshot

---

**Last Updated:** 2026-01-31  
**Version:** 1.0.0  
**Total Screenshots Required:** 30-35 across all categories

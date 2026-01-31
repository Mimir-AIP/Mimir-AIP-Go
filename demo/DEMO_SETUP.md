# Technical Setup and Configuration Guide

## Mimir AIP Computer Repair Shop Demo - Deployment Manual

---

## Table of Contents

1. [Prerequisites](#1-prerequisites)
2. [Environment Setup](#2-environment-setup)
3. [Database Initialization](#3-database-initialization)
4. [Data Generation](#4-data-generation)
5. [Pipeline Configuration](#5-pipeline-configuration)
6. [Scheduled Jobs Setup](#6-scheduled-jobs-setup)
7. [Verification Procedures](#7-verification-procedures)
8. [Troubleshooting](#8-troubleshooting)

---

## 1. Prerequisites

### 1.1 System Requirements

| Resource | Minimum | Recommended |
|----------|---------|-------------|
| CPU | 2 cores | 4+ cores |
| RAM | 512MB | 2GB+ |
| Storage | 100MB | 500MB |
| Network | Localhost | Internet (for supplier APIs) |

### 1.2 Software Dependencies

#### Required Software

```bash
# Verify Go installation
go version
# Expected: go version go1.23.x or later

# Verify SQLite installation
sqlite3 --version
# Expected: 3.x.x

# Verify Git installation
git --version
```

#### Optional Dependencies

```bash
# Docker (for containerized deployment)
docker --version
docker-compose --version

# cURL (for API testing)
curl --version

# jq (for JSON processing)
jq --version
```

### 1.3 Go Package Dependencies

The following packages are automatically resolved via `go mod download`:

```go
// Core dependencies from go.mod
github.com/gin-gonic/gin           // Web framework
github.com/mattn/go-sqlite3        // SQLite driver
github.com/robfig/cron/v3          // Cron scheduler
github.com/google/uuid             // UUID generation
```

---

## 2. Environment Setup

### 2.1 Repository Cloning

```bash
# Clone the repository
git clone https://github.com/Mimir-AIP/Mimir-AIP-Go.git
cd Mimir-AIP-Go

# Verify directory structure
ls -la
# Expected output:
# README.md
# go.mod
# go.sum
# demo/
# pipelines/
# ...
```

### 2.2 Environment Configuration

Create environment configuration file:

```bash
# Create .env file for demo environment
cat > demo/.env << 'EOF'
# Database Configuration
DEMO_DB_PATH=./repair_shop.db
DEMO_CSV_EXPORT_PATH=./exports

# Server Configuration
MIMIR_SERVER_HOST=0.0.0.0
MIMIR_SERVER_PORT=8080
MIMIR_API_KEY=demo-api-key-change-in-production

# Pipeline Configuration
MIMIR_PIPELINES_DIRECTORY=./pipelines
MIMIR_PLUGINS_TIMEOUT=60s
MIMIR_PLUGINS_MAX_CONCURRENT=10

# Scheduler Configuration
MIMIR_SCHEDULER_ENABLED=true
MIMIR_SCHEDULER_TIMEZONE=America/New_York

# Logging
MIMIR_LOG_LEVEL=INFO
MIMIR_LOG_FORMAT=json
EOF
```

### 2.3 Directory Structure Preparation

```bash
# Create required directories
cd demo
mkdir -p exports
mkdir -p logs
mkdir -p backups

# Verify structure
tree -L 2 || find . -maxdepth 2 -type d

# Expected structure:
# ./
# ├── .env
# ├── exports/
# ├── logs/
# ├── backups/
# ├── generate_data.go
# ├── schema.sql
# └── README.md
```

---

## 3. Database Initialization

### 3.1 Schema Creation

The database schema is defined in `schema.sql` and creates seven interconnected tables:

```sql
-- Core schema entities
-- 1. parts_inventory: 50 parts catalog
-- 2. suppliers: 5 vendor records  
-- 3. supplier_pricing: Multi-vendor pricing (250+ records)
-- 4. repair_jobs: 500 work orders
-- 5. job_parts: Many-to-many junction table (1,200+ records)
-- 6. stock_movements: Historical tracking (1,000+ records)
-- 7. sales: Transaction records (300 records)
```

**Schema Creation Methods:**

```bash
# Method 1: Manual SQLite execution
sqlite3 repair_shop.db < schema.sql

# Method 2: Via data generator (recommended)
go run generate_data.go
# This executes schema.sql automatically

# Method 3: Via Mimir pipeline
# Configure Input.sql plugin with schema.sql file
```

### 3.2 Schema Verification

```bash
# Connect to database
sqlite3 repair_shop.db

# List all tables
.tables

# Expected output:
# job_parts           parts_inventory     sales
# repair_jobs         stock_movements     suppliers
# supplier_pricing

# Verify table structures
.schema parts_inventory
.schema suppliers
.schema repair_jobs

# Check foreign key constraints
PRAGMA foreign_key_list(job_parts);
PRAGMA foreign_key_list(supplier_pricing);

# Exit SQLite
.quit
```

### 3.3 Index Verification

The schema includes optimized indexes for common query patterns:

```sql
-- Performance indexes created by schema.sql
CREATE INDEX idx_parts_category ON parts_inventory(category);
CREATE INDEX idx_parts_supplier ON parts_inventory(supplier_id);
CREATE INDEX idx_parts_stock ON parts_inventory(current_stock);
CREATE INDEX idx_pricing_supplier ON supplier_pricing(supplier_id);
CREATE INDEX idx_pricing_part ON supplier_pricing(part_id);
CREATE INDEX idx_jobs_status ON repair_jobs(status);
CREATE INDEX idx_jobs_date ON repair_jobs(created_at);
CREATE INDEX idx_movements_part ON stock_movements(part_id);
CREATE INDEX idx_movements_date ON stock_movements(created_at);
```

Verify indexes:

```bash
sqlite3 repair_shop.db ".indexes"
```

---

## 4. Data Generation

### 4.1 Synthetic Data Generation

The `generate_data.go` module creates realistic data using weighted distributions:

```bash
# Execute data generator
cd demo
go run generate_data.go

# Detailed execution output:
# 🚀 Mimir Demo Data Generator
# ============================================================
# 📦 Creating database schema...
#   → Executing schema.sql
#   → Verifying table creation
# 
# 📝 Generating demo data...
#   → Creating 5 suppliers...
#     - SUP-001: TechCorp Wholesale (98% reliability)
#     - SUP-002: Component Direct (95% reliability)
#     - SUP-003: MicroCenter Supply (92% reliability)
#     - SUP-004: Global Tech Parts (88% reliability)
#     - SUP-005: PC Parts Express (96% reliability)
# 
#   → Creating 50 parts...
#     - 8 CPUs (Intel & AMD)
#     - 6 GPUs (NVIDIA & AMD)
#     - 5 Memory modules
#     - 6 Storage devices
#     - 4 Motherboards
#     - 4 Power Supplies
#     - 4 Cases
#     - 5 Cooling solutions
#     - 3 Displays
#     - 5 Peripherals
# 
#   → Creating 250+ price records...
#     - Each part priced by 1-3 suppliers
#     - Price variance: -10% to +10% from base
#     - Availability: in_stock, low_stock, out_of_stock, backorder
# 
#   → Creating 500 repair jobs...
#     - Status distribution: pending (20%), in_progress (35%), completed (40%), cancelled (5%)
#     - Time range: Last 90 days
#     - 8 technicians assigned
# 
#   → Creating 1000+ stock movements...
#     - Movement types: sale (60%), restock (30%), adjustment (5%), return (5%)
#     - 60-day historical range
#     - ML training ready
# 
#   → Creating sales transactions...
#     - 300 sales from completed jobs
#     - Payment methods: credit, cash, debit
# 
# ✅ Demo data generated successfully!
```

### 4.2 Data Distribution Configuration

Modify data generation parameters in `generate_data.go`:

```go
// Line 61-168: Data catalog definitions
var partsCatalog = []struct {
    Name     string
    Category string
    BaseCost float64
}{
    // Add/modify parts here
    {"Intel Core i9-13900K", "CPU", 589.99},
    // ...
}

var suppliers = []struct {
    Name        string
    APIEndpoint string
    LeadTime    int
    MinOrder    int
    Reliability float64
}{
    // Add/modify suppliers here
    {"TechCorp Wholesale", "https://api.techcorp.com/v2/pricing", 2, 5, 0.98},
    // ...
}
```

### 4.3 Record Count Customization

```go
// Line 195-220: Generation counts
generator.createSuppliers()                    // 5 suppliers (fixed)
generator.createParts()                        // 50 parts (fixed)
generator.createPricingData()                  // 250+ prices (variable)
generator.createRepairJobs(500)                // 500 jobs (configurable)
generator.createStockMovements(1000)           // 1000 movements (configurable)
generator.createSalesData()                    // Derived from completed jobs
```

### 4.4 CSV Export Configuration

The generator prepares CSV exports for pipeline ingestion:

```go
// Line 450-454: CSV export function
func (g *DemoDataGenerator) exportToCSV() {
    // Export parts to CSV for pipeline ingestion
    fmt.Println("  (CSV export functionality ready)")
}
```

Enable CSV export by implementing:

```go
func (g *DemoDataGenerator) exportToCSV() {
    // Export parts_inventory
    partsFile, _ := os.Create("exports/parts_inventory.csv")
    // Write CSV headers and data...
    
    // Export suppliers
    suppliersFile, _ := os.Create("exports/suppliers.csv")
    // Write CSV headers and data...
    
    // Export other tables as needed...
}
```

---

## 5. Pipeline Configuration

### 5.1 Pipeline YAML Structure

Pipelines are defined in YAML format with the following schema:

```yaml
id: pipeline_<unique_id>
name: <Pipeline Name>
metadata:
  id: pipeline_<unique_id>
  name: <Pipeline Name>
  description: <Description>
  tags:
    - <tag1>
    - <tag2>
  enabled: true|false
  auto_extract_ontology: true|false
  created_at: <ISO8601 timestamp>
  updated_at: <ISO8601 timestamp>
  created_by: <user>
  updated_by: <user>
  version: <integer>
config:
  name: <pipeline_name>
  enabled: true|false
  steps:
    - name: <step_name>
      plugin: <PluginType.plugin_name>
      config:
        <plugin_specific_config>
      input: <input_reference>  # Optional
      output: <output_reference>
```

### 5.2 Demo Pipeline Templates

#### Template 1: Data Ingestion Pipeline

```yaml
id: pipeline_demo_ingestion
name: Repair Shop Data Ingestion
metadata:
  id: pipeline_demo_ingestion
  name: Repair Shop Data Ingestion
  description: Ingest parts inventory and trigger auto-extraction
  tags:
    - ingestion
    - auto-extract
    - demo
  enabled: true
  auto_extract_ontology: true
  created_at: 2026-01-31T00:00:00Z
  updated_at: 2026-01-31T00:00:00Z
  created_by: demo
  updated_by: demo
  version: 1
config:
  name: repair-shop-ingestion
  enabled: true
  steps:
    - name: read-parts-csv
      plugin: Input.csv
      config:
        file_path: ./demo/exports/parts_inventory.csv
        has_headers: true
        delimiter: ","
        encoding: "UTF-8"
      output: parts_data
    
    - name: validate-parts
      plugin: Process.validate
      config:
        required_fields:
          - part_id
          - name
          - category
          - current_stock
        validation_rules:
          current_stock: "integer>=0"
          unit_cost: "decimal>0"
      input: parts_data
      output: validated_parts
    
    - name: load-to-database
      plugin: Output.sql
      config:
        connection: "sqlite://./demo/repair_shop.db"
        table: "parts_inventory"
        operation: "upsert"
        key_field: "part_id"
      input: validated_parts
```

#### Template 2: Pricing Monitor Pipeline

```yaml
id: pipeline_demo_pricing
name: Pricing Monitor
metadata:
  id: pipeline_demo_pricing
  name: Pricing Monitor
  description: Monitor vendor pricing and detect changes
  tags:
    - monitoring
    - pricing
    - suppliers
  enabled: true
  auto_extract_ontology: false
  created_at: 2026-01-31T00:00:00Z
  updated_at: 2026-01-31T00:00:00Z
  version: 1
config:
  name: pricing-monitor
  enabled: true
  steps:
    - name: query-current-pricing
      plugin: Input.sql
      config:
        connection: "sqlite://./demo/repair_shop.db"
        query: |
          SELECT 
            sp.pricing_id,
            sp.supplier_id,
            sp.part_id,
            sp.unit_price,
            sp.availability_status,
            s.supplier_name,
            p.name as part_name
          FROM supplier_pricing sp
          JOIN suppliers s ON sp.supplier_id = s.supplier_id
          JOIN parts_inventory p ON sp.part_id = p.part_id
          WHERE sp.last_updated < datetime('now', '-6 hours')
      output: stale_pricing
    
    - name: fetch-vendor-prices
      plugin: Input.http
      config:
        source: stale_pricing
        endpoint_field: "api_endpoint"  # From suppliers table
        method: "GET"
        headers:
          Authorization: "Bearer ${SUPPLIER_API_KEY}"
        timeout: 30s
      input: stale_pricing
      output: vendor_responses
    
    - name: detect-price-changes
      plugin: Process.transform
      config:
        transformation: |
          // Compare current vs new prices
          // Calculate price_change_pct
          // Flag significant changes (>5%)
      input: vendor_responses
      output: price_changes
    
    - name: update-pricing-database
      plugin: Output.sql
      config:
        connection: "sqlite://./demo/repair_shop.db"
        table: "supplier_pricing"
        operation: "update"
        key_field: "pricing_id"
      input: price_changes
```

#### Template 3: Inventory Monitor Pipeline

```yaml
id: pipeline_demo_inventory
name: Inventory Monitor
metadata:
  id: pipeline_demo_inventory
  name: Inventory Monitor
  description: Monitor stock levels and generate alerts
  tags:
    - inventory
    - monitoring
    - alerts
  enabled: true
  auto_extract_ontology: false
  version: 1
config:
  name: inventory-monitor
  enabled: true
  steps:
    - name: check-stock-levels
      plugin: Input.sql
      config:
        connection: "sqlite://./demo/repair_shop.db"
        query: |
          SELECT 
            part_id,
            name,
            category,
            current_stock,
            min_stock,
            reorder_point,
            (current_stock <= min_stock) as critical,
            (current_stock <= reorder_point) as warning
          FROM parts_inventory
          WHERE current_stock <= reorder_point
          ORDER BY (current_stock - min_stock) ASC
      output: low_stock_items
    
    - name: generate-alerts
      plugin: Process.transform
      config:
        template: |
          ALERT: {{if .critical}}CRITICAL{{else}}WARNING{{end}}
          Part: {{.name}} ({{.part_id}})
          Current Stock: {{.current_stock}}
          Minimum: {{.min_stock}}
          Reorder Point: {{.reorder_point}}
      input: low_stock_items
      output: stock_alerts
    
    - name: send-notifications
      plugin: Output.webhook
      config:
        url: "${ALERT_WEBHOOK_URL}"
        method: "POST"
        headers:
          Content-Type: "application/json"
        payload_template: |
          {
            "alerts": {{toJson .stock_alerts}},
            "timestamp": "{{now}}",
            "source": "inventory-monitor"
          }
      input: stock_alerts
```

### 5.3 Pipeline Deployment

```bash
# Create pipeline directory
mkdir -p ./pipelines

# Save pipeline configuration
cat > ./pipelines/demo_ingestion.yaml << 'EOF'
# Paste YAML content here
EOF

# Load pipeline via API
curl -X POST http://localhost:8080/api/v1/pipelines \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${MIMIR_API_KEY}" \
  -d @./pipelines/demo_ingestion.yaml

# Or use pipeline directory auto-discovery
# (Server scans ./pipelines/ on startup)
```

---

## 6. Scheduled Jobs Setup

### 6.1 Cron Expression Format

The scheduler uses standard cron expressions:

```
┌───────────── minute (0 - 59)
│ ┌───────────── hour (0 - 23)
│ │ ┌───────────── day of month (1 - 31)
│ │ │ ┌───────────── month (1 - 12)
│ │ │ │ ┌───────────── day of week (0 - 6) (Sunday to Saturday)
│ │ │ │ │
│ │ │ │ │
* * * * *
```

**Common Patterns:**

| Schedule | Cron Expression | Description |
|----------|-----------------|-------------|
| Every minute | `* * * * *` | Frequent testing |
| Hourly | `0 * * * *` | Top of each hour |
| Daily 2 AM | `0 2 * * *` | Daily maintenance window |
| Every 6 hours | `0 */6 * * *` | Regular monitoring |
| Weekly | `0 0 * * 0` | Sunday midnight |
| Business hours | `0 9-17 * * 1-5` | 9 AM - 5 PM weekdays |

### 6.2 Job Configuration via API

```bash
# Create scheduled job
curl -X POST http://localhost:8080/api/v1/scheduler/jobs \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${MIMIR_API_KEY}" \
  -d '{
    "id": "daily-inventory-sync",
    "name": "Daily Inventory Synchronization",
    "pipeline": "Repair Shop Data Ingestion",
    "cron_expr": "0 2 * * *",
    "timezone": "America/New_York",
    "enabled": true,
    "retry_policy": {
      "max_retries": 3,
      "backoff_seconds": 300
    }
  }'

# Create pricing monitor job
curl -X POST http://localhost:8080/api/v1/scheduler/jobs \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${MIMIR_API_KEY}" \
  -d '{
    "id": "pricing-monitor",
    "name": "Vendor Price Monitoring",
    "pipeline": "Pricing Monitor",
    "cron_expr": "0 */6 * * *",
    "timezone": "UTC",
    "enabled": true
  }'

# Create inventory alert job
curl -X POST http://localhost:8080/api/v1/scheduler/jobs \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${MIMIR_API_KEY}" \
  -d '{
    "id": "inventory-alerts",
    "name": "Low Stock Alert System",
    "pipeline": "Inventory Monitor",
    "cron_expr": "0 9,17 * * 1-5",
    "timezone": "America/New_York",
    "enabled": true
  }'
```

### 6.3 Job Management Commands

```bash
# List all scheduled jobs
curl http://localhost:8080/api/v1/scheduler/jobs \
  -H "Authorization: Bearer ${MIMIR_API_KEY}"

# Get job details
curl http://localhost:8080/api/v1/scheduler/jobs/daily-inventory-sync \
  -H "Authorization: Bearer ${MIMIR_API_KEY}"

# Update job
curl -X PUT http://localhost:8080/api/v1/scheduler/jobs/daily-inventory-sync \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${MIMIR_API_KEY}" \
  -d '{
    "cron_expr": "0 3 * * *",
    "enabled": true
  }'

# Delete job
curl -X DELETE http://localhost:8080/api/v1/scheduler/jobs/daily-inventory-sync \
  -H "Authorization: Bearer ${MIMIR_API_KEY}"

# Trigger job manually
curl -X POST http://localhost:8080/api/v1/scheduler/jobs/daily-inventory-sync/trigger \
  -H "Authorization: Bearer ${MIMIR_API_KEY}"
```

### 6.4 Configuration File Setup

```yaml
# config/scheduler.yaml
scheduler:
  enabled: true
  timezone: "America/New_York"
  max_jobs: 100
  job_timeout: 3600s
  
  jobs:
    - id: daily-inventory-sync
      name: "Daily Inventory Synchronization"
      pipeline: "Repair Shop Data Ingestion"
      cron_expr: "0 2 * * *"
      enabled: true
      
    - id: pricing-monitor
      name: "Vendor Price Monitoring"
      pipeline: "Pricing Monitor"
      cron_expr: "0 */6 * * *"
      enabled: true
      
    - id: inventory-alerts
      name: "Low Stock Alert System"
      pipeline: "Inventory Monitor"
      cron_expr: "0 9,17 * * 1-5"
      enabled: true
```

---

## 7. Verification Procedures

### 7.1 Database Verification

```bash
#!/bin/bash
# verify_database.sh

DB="./demo/repair_shop.db"

echo "=== Database Verification ==="
echo

# Check all tables exist
echo "1. Verifying table existence..."
tables=("parts_inventory" "suppliers" "supplier_pricing" "repair_jobs" "job_parts" "stock_movements" "sales")
for table in "${tables[@]}"; do
    count=$(sqlite3 "$DB" "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='$table';")
    if [ "$count" -eq 1 ]; then
        echo "   ✓ $table exists"
    else
        echo "   ✗ $table MISSING"
    fi
done

# Check record counts
echo
echo "2. Verifying record counts..."
sqlite3 "$DB" << 'EOF'
SELECT 'Parts: ' || COUNT(*) FROM parts_inventory;
SELECT 'Suppliers: ' || COUNT(*) FROM suppliers;
SELECT 'Pricing: ' || COUNT(*) FROM supplier_pricing;
SELECT 'Jobs: ' || COUNT(*) FROM repair_jobs;
SELECT 'Job Parts: ' || COUNT(*) FROM job_parts;
SELECT 'Movements: ' || COUNT(*) FROM stock_movements;
SELECT 'Sales: ' || COUNT(*) FROM sales;
EOF

# Verify referential integrity
echo
echo "3. Verifying foreign key integrity..."
sqlite3 "$DB" "PRAGMA foreign_key_check;"
if [ $? -eq 0 ]; then
    echo "   ✓ No foreign key violations"
else
    echo "   ✗ Foreign key violations detected"
fi

echo
echo "=== Verification Complete ==="
```

### 7.2 API Verification

```bash
#!/bin/bash
# verify_api.sh

API_URL="http://localhost:8080"
API_KEY="${MIMIR_API_KEY:-demo-api-key}"

echo "=== API Verification ==="
echo

# Health check
echo "1. Health endpoint..."
curl -s "${API_URL}/health" | jq .

# List plugins
echo
echo "2. Available plugins..."
curl -s "${API_URL}/api/v1/plugins" \
  -H "Authorization: Bearer ${API_KEY}" | jq '.plugins | length'

# List pipelines
echo
echo "3. Configured pipelines..."
curl -s "${API_URL}/api/v1/pipelines" \
  -H "Authorization: Bearer ${API_KEY}" | jq '.pipelines | length'

# List scheduled jobs
echo
echo "4. Scheduled jobs..."
curl -s "${API_URL}/api/v1/scheduler/jobs" \
  -H "Authorization: Bearer ${API_KEY}" | jq '.jobs | length'

# Execute test query
echo
echo "5. Database query test..."
curl -s "${API_URL}/api/v1/query" \
  -H "Authorization: Bearer ${API_KEY}" \
  -H "Content-Type: application/json" \
  -d '{"sql": "SELECT COUNT(*) as count FROM parts_inventory"}' | jq .

echo
echo "=== API Verification Complete ==="
```

### 7.3 Pipeline Verification

```bash
#!/bin/bash
# verify_pipelines.sh

API_URL="http://localhost:8080"
API_KEY="${MIMIR_API_KEY:-demo-api-key}"

echo "=== Pipeline Verification ==="
echo

# Test data ingestion pipeline
echo "1. Testing Data Ingestion Pipeline..."
curl -X POST "${API_URL}/api/v1/pipelines/execute" \
  -H "Authorization: Bearer ${API_KEY}" \
  -H "Content-Type: application/json" \
  -d '{
    "pipeline_name": "Repair Shop Data Ingestion",
    "context": {
      "test_mode": true
    }
  }' | jq .

# Check execution status
echo
echo "2. Pipeline execution history..."
curl -s "${API_URL}/api/v1/pipelines/history" \
  -H "Authorization: Bearer ${API_KEY}" | jq '.executions[:3]'

echo
echo "=== Pipeline Verification Complete ==="
```

---

## 8. Troubleshooting

### 8.1 Common Issues

#### Issue: Database locked or busy

**Symptoms:**
```
Error: database is locked
Error: database table is locked
```

**Resolution:**
```bash
# 1. Close all SQLite connections
# 2. Enable WAL mode for better concurrency
sqlite3 repair_shop.db "PRAGMA journal_mode=WAL;"

# 3. Check for open connections
lsof | grep repair_shop.db

# 4. Restart Mimir server
pkill mimir-aip-server
./mimir-aip-server --server
```

#### Issue: Pipeline execution fails

**Symptoms:**
```
Plugin not found: Input.csv
Failed to execute step: read-parts-csv
```

**Resolution:**
```bash
# 1. Verify plugin availability
curl http://localhost:8080/api/v1/plugins | jq '.plugins[] | select(.type == "Input")'

# 2. Check file path exists
ls -la ./demo/exports/parts_inventory.csv

# 3. Verify pipeline YAML syntax
cat ./pipelines/demo_ingestion.yaml | head -20

# 4. Check server logs
tail -f ./logs/mimir.log
```

#### Issue: Scheduled jobs not executing

**Symptoms:**
- Jobs remain in "pending" state
- No execution history recorded

**Resolution:**
```bash
# 1. Verify scheduler is enabled
curl http://localhost:8080/api/v1/config | jq '.scheduler.enabled'

# 2. Check job configuration
curl http://localhost:8080/api/v1/scheduler/jobs | jq '.jobs[] | {id, enabled, cron_expr}'

# 3. Verify timezone setting
curl http://localhost:8080/api/v1/config | jq '.scheduler.timezone'

# 4. Manually trigger job to test
curl -X POST http://localhost:8080/api/v1/scheduler/jobs/daily-inventory-sync/trigger
```

#### Issue: Data generator fails

**Symptoms:**
```
Failed to open database: unable to open database file
undefined: sql.Open
```

**Resolution:**
```bash
# 1. Check SQLite driver installation
go get github.com/mattn/go-sqlite3

# 2. Verify directory permissions
ls -la ./demo/
chmod 755 ./demo

# 3. Check CGO availability
go env CGO_ENABLED  # Should return 1

# 4. Reinstall dependencies
go mod tidy
go mod download
```

### 8.2 Log Analysis

```bash
# Enable debug logging
export MIMIR_LOG_LEVEL=DEBUG
./mimir-aip-server --server

# View structured logs
journalctl -u mimir-aip -f  # Systemd
# OR
tail -f ./logs/mimir.log | jq '.'  # JSON logs

# Filter for specific components
tail -f ./logs/mimir.log | jq 'select(.component == "scheduler")'
tail -f ./logs/mimir.log | jq 'select(.level == "ERROR")'
```

### 8.3 Performance Tuning

```bash
# Optimize SQLite for concurrent access
cat > optimize_db.sql << 'EOF'
PRAGMA journal_mode=WAL;
PRAGMA synchronous=NORMAL;
PRAGMA cache_size=-64000;  -- 64MB cache
PRAGMA temp_store=MEMORY;
PRAGMA mmap_size=268435456;  -- 256MB memory map
EOF

sqlite3 repair_shop.db < optimize_db.sql

# Configure Mimir for demo workload
cat > config/demo.yaml << 'EOF'
server:
  host: "0.0.0.0"
  port: 8080
  read_timeout: 60s
  write_timeout: 60s

plugins:
  max_concurrent: 20
  timeout: 120s

scheduler:
  enabled: true
  max_jobs: 50
  job_timeout: 1800s

monitoring:
  enabled: true
  metrics_interval: 60s
  health_check_interval: 30s
EOF
```

---

## Appendix A: Environment Variables Reference

| Variable | Default | Description |
|----------|---------|-------------|
| `MIMIR_SERVER_HOST` | `0.0.0.0` | Server bind address |
| `MIMIR_SERVER_PORT` | `8080` | Server port |
| `MIMIR_API_KEY` | `none` | API authentication key |
| `MIMIR_LOG_LEVEL` | `INFO` | Log verbosity (DEBUG, INFO, WARN, ERROR) |
| `MIMIR_LOG_FORMAT` | `json` | Log format (json, text) |
| `DEMO_DB_PATH` | `./repair_shop.db` | Demo database path |
| `DEMO_CSV_EXPORT_PATH` | `./exports` | CSV export directory |

---

## Appendix B: File Permissions

```bash
# Set appropriate permissions for demo environment
chmod 755 ./demo
chmod 644 ./demo/schema.sql
chmod 644 ./demo/generate_data.go
chmod 600 ./demo/.env  # Protect sensitive config
chmod 644 ./demo/repair_shop.db
chmod 755 ./demo/exports
chmod 755 ./demo/logs
chmod 755 ./demo/backups
```

---

**Last Updated:** 2026-01-31  
**Version:** 1.0.0  
**Maintainer:** Mimir AIP Development Team

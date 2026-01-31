# Mimir AIP: End-to-End Ontology-Driven Data Platform

## Computer Repair Shop Demonstration

---

## Abstract

This demonstration presents a comprehensive implementation of the Mimir AIP (Automated Information Processing) platform applied to a computer repair shop business scenario. The system showcases an end-to-end ontology-driven data architecture that integrates automated data ingestion, real-time inventory monitoring, predictive analytics, and conversational AI interfaces. By leveraging a relational database model with 1,000+ records across seven interconnected entities—including parts inventory, suppliers, repair jobs, and sales transactions—the platform demonstrates advanced capabilities in automated ontology extraction from CSV data sources, scheduled pipeline execution for continuous data synchronization, and multi-vendor price comparison algorithms. The demonstration validates the platform's ability to process heterogeneous data types, maintain referential integrity across complex relationships, and provide actionable business intelligence through both automated workflows and natural language query interfaces. Key contributions include a novel approach to automated schema inference from operational data, real-time pricing intelligence across distributed supplier networks, and an integrated chat-based analytics system that bridges the gap between technical data pipelines and business user accessibility.

---

## Table of Contents

1. [System Architecture](#system-architecture)
2. [Key Features Demonstrated](#key-features-demonstrated)
3. [Technology Stack](#technology-stack)
4. [Quick Start Guide](#quick-start-guide)
5. [Documentation References](#documentation-references)

---

## System Architecture

### High-Level System Overview

The Computer Repair Shop Demonstration implements a multi-tier architecture that integrates data generation, pipeline-based processing, and ontology-driven analytics:

```
┌─────────────────────────────────────────────────────────────────────┐
│                     PRESENTATION LAYER                              │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────────┐  │
│  │   REST API   │  │  MCP Server  │  │ Chat-based Analytics UI  │  │
│  │   Server     │  │ (LLM Tools)  │  │   (Natural Language)     │  │
│  └──────────────┘  └──────────────┘  └──────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────┐
│                     PIPELINE ORCHESTRATION LAYER                    │
│  ┌─────────────────┐  ┌──────────────┐  ┌──────────────────────┐   │
│  │  Data Ingestion │  │   Pricing    │  │ Inventory Monitoring │   │
│  │    Pipeline     │  │   Monitor    │  │      Pipeline        │   │
│  └─────────────────┘  └──────────────┘  └──────────────────────┘   │
│  ┌─────────────────┐  ┌──────────────┐  ┌──────────────────────┐   │
│  │  Auto-Extract   │  │   Daily Sync │  │  Predictive Analytics│   │
│  │    Pipeline     │  │   Pipeline   │  │      Pipeline        │   │
│  └─────────────────┘  └──────────────┘  └──────────────────────┘   │
└─────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────┐
│                     PLUGIN EXECUTION LAYER                          │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐              │
│  │ Input Plugins│  │   Process    │  │Output Plugins│              │
│  │  (CSV, SQL)  │  │  (Transform) │  │  (SQL, API)  │              │
│  └──────────────┘  └──────────────┘  └──────────────┘              │
└─────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────┐
│                     DATA PERSISTENCE LAYER                          │
│  ┌─────────────────┐  ┌──────────────┐  ┌──────────────────────┐   │
│  │   SQLite DB     │  │  Ontology    │  │   CSV Exports        │   │
│  │ (repair_shop.db)│  │  (JSON-LD)   │  │   (Pipeline Input)   │   │
│  └─────────────────┘  └──────────────┘  └──────────────────────┘   │
└─────────────────────────────────────────────────────────────────────┘
```

### Data Model Architecture

The demonstration utilizes a normalized relational schema consisting of seven primary entities with established referential integrity constraints:

| Entity | Records | Primary Function |
|--------|---------|------------------|
| `parts_inventory` | 50 | Core inventory catalog with stock levels |
| `suppliers` | 5 | Vendor management with API endpoints |
| `supplier_pricing` | 250+ | Real-time pricing from multiple vendors |
| `repair_jobs` | 500 | Work order tracking and customer management |
| `job_parts` | 1,200+ | Many-to-many relationship: jobs ↔ parts |
| `stock_movements` | 1,000+ | Historical inventory changes for ML training |
| `sales` | 300 | Transaction records for revenue analysis |

### Component Interaction Model

**Data Flow Architecture:**

1. **Data Generation Layer**: The `generate_data.go` module creates synthetic but realistic data using weighted random distributions and business-logic constraints
2. **Ingestion Pipeline**: CSV files are processed through the Mimir pipeline system with automatic ontology extraction
3. **Processing Layer**: Data transformations occur through the plugin system with context propagation
4. **Analytics Layer**: SQL queries and LLM-based natural language interfaces provide business intelligence

---

## Key Features Demonstrated

### 1. Automated Ontology Extraction

The platform automatically infers semantic structure from CSV data sources:

```yaml
# Pipeline Configuration with Auto-Extraction
config:
  name: repair-shop-ingestion
  enabled: true
  steps:
    - name: read-parts-csv
      plugin: Input.csv
      config:
        file_path: ./demo/parts_inventory.csv
        has_headers: true
        auto_extract_ontology: true  # ← Automatic schema inference
      output: parts_data
```

**Generated Ontology Example:**
```json
{
  "@context": {
    "part_id": "http://schema.org/productID",
    "name": "http://schema.org/name",
    "category": "http://schema.org/category",
    "current_stock": "http://schema.org/inventoryLevel"
  },
  "@type": "Product",
  "supplier": {
    "@type": "Organization",
    "@id": "supplier_reference"
  }
}
```

### 2. Multi-Vendor Price Comparison

Real-time pricing intelligence across distributed supplier networks:

**Query Pattern:**
```sql
SELECT 
    p.part_id,
    p.name,
    p.category,
    s.supplier_name,
    sp.unit_price,
    sp.lead_time_days,
    sp.availability_status,
    ((sp.unit_price - p.unit_cost) / p.unit_cost * 100) as price_variance_pct
FROM parts_inventory p
JOIN supplier_pricing sp ON p.part_id = sp.part_id
JOIN suppliers s ON sp.supplier_id = s.supplier_id
WHERE p.part_id = 'PART-0001'
ORDER BY sp.unit_price ASC;
```

### 3. Predictive Inventory Analytics

Machine learning-ready data structures for demand forecasting:

**Features Extracted:**
- Historical stock movement patterns (1,000+ records)
- Seasonal demand indicators from repair job frequency
- Supplier reliability scoring based on lead time variance
- Price volatility metrics for procurement optimization

### 4. Chat-Based Natural Language Analytics

Integration with the Model Context Protocol (MCP) server enables conversational data exploration:

**Example Interactions:**

| User Query | System Response |
|------------|-----------------|
| "Which parts are running low on stock?" | SQL query → Filtered inventory list |
| "Compare GPU prices across suppliers" | Aggregation query → Price comparison table |
| "Show me last week's sales" | Date-filtered sales report |
| "What parts are used most in repairs?" | Job parts aggregation → Top 10 list |

### 5. Scheduled Pipeline Execution

Automated data synchronization with cron-based scheduling:

```yaml
scheduler:
  jobs:
    - id: daily-inventory-sync
      name: Daily Inventory Synchronization
      pipeline: Repair_Shop_Data_Ingestion
      cron_expr: "0 2 * * *"  # 2:00 AM daily
      timezone: "America/New_York"
      enabled: true
    
    - id: pricing-monitor
      name: Vendor Price Monitoring
      pipeline: Pricing_Monitor
      cron_expr: "0 */6 * * *"  # Every 6 hours
      timezone: "UTC"
      enabled: true
```

---

## Technology Stack

### Core Platform

| Component | Technology | Version | Purpose |
|-----------|------------|---------|---------|
| Runtime | Go | 1.23+ | High-performance execution engine |
| Database | SQLite | 3.x | Embedded relational storage |
| API Server | Gin | Latest | RESTful endpoint exposure |
| Scheduler | robfig/cron | v3 | Cron-based job scheduling |
| MCP Server | Custom | v1 | LLM tool integration |

### Data Processing

| Component | Technology | Purpose |
|-----------|------------|---------|
| CSV Parsing | Standard Library | Data ingestion |
| SQL ORM | database/sql + mattn/go-sqlite3 | Database operations |
| Data Generation | math/rand + time | Synthetic data creation |
| Ontology | JSON-LD | Semantic data representation |

### Frontend & Interface

| Component | Technology | Purpose |
|-----------|------------|---------|
| Web UI | React + TypeScript | Management interface |
| Chat Interface | Custom MCP | Natural language queries |
| Visualization | ASCII + JSON | Terminal and API output |

### Deployment

| Component | Technology | Purpose |
|-----------|------------|---------|
| Containerization | Docker | Portable deployment |
| Orchestration | Docker Compose | Multi-service management |
| Build | Multi-stage | Optimized 9.34MB image |

---

## Quick Start Guide

### Prerequisites

- **Go Runtime**: Version 1.23 or later
- **Docker**: Docker Engine 20.10+ (optional, for containerized deployment)
- **Operating System**: Linux, macOS, or Windows
- **Memory**: Minimum 512MB RAM, recommended 2GB+
- **Storage**: Minimum 100MB available disk space

### Step 1: Repository Setup

```bash
# Clone the Mimir AIP repository
git clone https://github.com/Mimir-AIP/Mimir-AIP-Go.git
cd Mimir-AIP-Go

# Download dependencies
go mod download
```

### Step 2: Demo Data Generation

```bash
# Navigate to demo directory
cd demo

# Generate synthetic data (creates repair_shop.db + CSV exports)
go run generate_data.go

# Expected output:
# 🚀 Mimir Demo Data Generator
# ============================================================
# 📦 Creating database schema...
# 📝 Generating demo data...
#   → Creating 5 suppliers...
#   → Creating 50 parts...
#   → Creating 250+ price records...
#   → Creating 500 repair jobs...
#   → Creating 1000+ stock movements...
#   → Creating sales transactions...
# ✅ Demo data generated successfully!
```

### Step 3: Database Verification

```bash
# Verify database creation
sqlite3 repair_shop.db ".tables"

# Expected output:
# job_parts           parts_inventory     sales
# repair_jobs         stock_movements     suppliers
# supplier_pricing

# View record counts
sqlite3 repair_shop.db "SELECT 'Parts: ' || COUNT(*) FROM parts_inventory UNION ALL SELECT 'Jobs: ' || COUNT(*) FROM repair_jobs UNION ALL SELECT 'Movements: ' || COUNT(*) FROM stock_movements;"
```

### Step 4: Platform Startup

```bash
# Build the Mimir AIP server
cd ..
go build -o mimir-aip-server .

# Start the server with default configuration
./mimir-aip-server --server

# Or with custom configuration
./mimir-aip-server --server --config ./config/demo.yaml

# Expected output:
# 2026/01/31 10:00:00 Server starting on :8080
# 2026/01/31 10:00:00 API available at http://localhost:8080
# 2026/01/31 10:00:00 Scheduler started with 0 jobs
```

### Step 5: Pipeline Configuration

```bash
# Create data ingestion pipeline
curl -X POST http://localhost:8080/api/v1/pipelines \
  -H "Content-Type: application/json" \
  -d @- << 'EOF'
{
  "name": "Repair Shop Data Ingestion",
  "description": "Ingest parts inventory from CSV",
  "steps": [
    {
      "name": "read-parts-csv",
      "plugin": "Input.csv",
      "config": {
        "file_path": "./demo/parts_inventory.csv",
        "has_headers": true
      },
      "output": "parts_data"
    },
    {
      "name": "load-to-database",
      "plugin": "Output.sql",
      "config": {
        "connection": "sqlite://./demo/repair_shop.db",
        "table": "parts_inventory",
        "operation": "upsert"
      },
      "input": "parts_data"
    }
  ]
}
EOF
```

### Step 6: Verification

```bash
# Health check
curl http://localhost:8080/health

# List available plugins
curl http://localhost:8080/api/v1/plugins

# Execute pipeline manually
curl -X POST http://localhost:8080/api/v1/pipelines/execute \
  -H "Content-Type: application/json" \
  -d '{"pipeline_name": "Repair Shop Data Ingestion"}'

# Query inventory via API
curl "http://localhost:8080/api/v1/query?sql=SELECT%20*%20FROM%20parts_inventory%20LIMIT%205"
```

---

## Documentation References

### Internal Documentation

- [DEMO_SETUP.md](./DEMO_SETUP.md) - Detailed technical setup and configuration
- [SCENARIOS.md](./SCENARIOS.md) - Demo scenarios and workflow documentation
- [SCREENSHOTS.md](./SCREENSHOTS.md) - Visual documentation guidelines

### Platform Documentation

- [Main README](../README.md) - Core platform documentation
- [API Reference](../README.md#api-reference) - REST API specifications
- [Plugin Development](../README.md#plugin-development-framework) - Custom plugin creation
- [Docker Deployment](../docs/DOCKER_DEPLOYMENT_GUIDE.md) - Production deployment

---

## Citation

For academic and research purposes, please cite this demonstration as:

```
Mimir AIP: End-to-End Ontology-Driven Data Platform - Computer Repair Shop Demonstration.
Version 1.0.0. GitHub Repository: https://github.com/Mimir-AIP/Mimir-AIP-Go/tree/main/demo
```

---

**Project Repository**: https://github.com/Mimir-AIP/Mimir-AIP-Go  
**Documentation**: https://mimir-aip.github.io/wiki/  
**Issues & Support**: https://github.com/Mimir-AIP/Mimir-AIP-Go/issues

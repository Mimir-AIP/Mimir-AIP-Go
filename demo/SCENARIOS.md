# Demo Scenarios and Workflow Documentation

## Mimir AIP Computer Repair Shop - Use Case Scenarios

---

## Table of Contents

1. [Scenario 1: Daily Inventory Check](#scenario-1-daily-inventory-check)
2. [Scenario 2: Vendor Price Comparison](#scenario-2-vendor-price-comparison)
3. [Scenario 3: Predictive Maintenance](#scenario-3-predictive-maintenance)
4. [Scenario 4: Chat-based Analytics](#scenario-4-chat-based-analytics)
5. [Cross-Scenario Integration](#cross-scenario-integration)

---

## Scenario 1: Daily Inventory Check

### 1.1 Overview

The Daily Inventory Check scenario demonstrates automated stock level monitoring, reorder point analysis, and alert generation for the computer repair shop's parts inventory. This scenario showcases the Mimir AIP platform's ability to process relational data, apply business rules, and trigger automated workflows based on threshold conditions.

### 1.2 Business Context

**Actors:**
- Inventory Manager: Reviews daily stock reports
- System: Automated monitoring and alerting
- Suppliers: Receive reorder notifications

**Goals:**
- Maintain optimal stock levels (minimize stockouts and overstock)
- Automate reorder point detection
- Generate actionable alerts for procurement team
- Track inventory movement history for demand forecasting

### 1.3 Data Model Involved

| Entity | Role in Scenario | Key Fields |
|--------|-----------------|------------|
| `parts_inventory` | Primary source | `part_id`, `current_stock`, `min_stock`, `reorder_point` |
| `stock_movements` | Historical analysis | `movement_type`, `quantity_change`, `created_at` |
| `suppliers` | Reference data | `supplier_id`, `supplier_name`, `lead_time_days` |
| `repair_jobs` | Demand indicator | `status`, `created_at` (correlates with parts usage) |

### 1.4 Workflow Steps

#### Step 1: Scheduled Data Collection

```yaml
# Pipeline: Daily Inventory Sync
trigger:
  type: scheduled
  cron: "0 6 * * 1-5"  # 6 AM, weekdays only
  
steps:
  - name: fetch-inventory-data
    plugin: Input.sql
    config:
      connection: "sqlite://./demo/repair_shop.db"
      query: |
        SELECT 
          p.part_id,
          p.name,
          p.category,
          p.current_stock,
          p.min_stock,
          p.reorder_point,
          p.unit_cost,
          s.supplier_name,
          s.lead_time_days,
          (SELECT COUNT(*) FROM job_parts jp 
           JOIN repair_jobs rj ON jp.job_id = rj.job_id 
           WHERE jp.part_id = p.part_id 
           AND rj.created_at >= date('now', '-7 days')) as usage_7d
        FROM parts_inventory p
        JOIN suppliers s ON p.supplier_id = s.supplier_id
    output: inventory_data
```

#### Step 2: Stock Level Analysis

```yaml
  - name: analyze-stock-levels
    plugin: Process.transform
    config:
      transformation_rules:
        - condition: "current_stock <= min_stock"
          status: "CRITICAL"
          action: "urgent_reorder"
        - condition: "current_stock <= reorder_point"
          status: "WARNING"
          action: "planned_reorder"
        - condition: "usage_7d > (current_stock * 0.5)"
          status: "HIGH_TURNOVER"
          action: "increase_safety_stock"
    input: inventory_data
    output: stock_analysis
```

#### Step 3: Alert Generation

```yaml
  - name: generate-alerts
    plugin: Process.transform
    config:
      template: |
        {
          "alert_id": "INV-{{.part_id}}-{{now | format \"20060102\"}}",
          "part_id": "{{.part_id}}",
          "part_name": "{{.name}}",
          "category": "{{.category}}",
          "current_stock": {{.current_stock}},
          "min_stock": {{.min_stock}},
          "reorder_point": {{.reorder_point}},
          "status": "{{.status}}",
          "days_until_stockout": "{{if gt .usage_7d 0}}{{div .current_stock (div .usage_7d 7)}}{{else}}N/A{{end}}",
          "recommended_action": "{{.action}}",
          "supplier": "{{.supplier_name}}",
          "lead_time": {{.lead_time_days}},
          "timestamp": "{{now}}"
        }
      filter: "status IN ('CRITICAL', 'WARNING')"
    input: stock_analysis
    output: inventory_alerts
```

#### Step 4: Notification Distribution

```yaml
  - name: send-email-alerts
    plugin: Output.email
    config:
      smtp_server: "${SMTP_SERVER}"
      from: "inventory@mimir-demo.local"
      to: ["manager@repairshop.local", "procurement@repairshop.local"]
      subject: "Daily Inventory Alert - {{now | format \"2006-01-02\"}}"
      template: |
        <h2>Inventory Alert Report - {{now | format "January 2, 2006"}}</h2>
        
        <h3>Critical Items (Below Minimum Stock)</h3>
        <table>
          <tr><th>Part</th><th>Current</th><th>Minimum</th><th>Action</th></tr>
          {{range .inventory_alerts}}{{if eq .status "CRITICAL"}}
          <tr>
            <td>{{.part_name}}</td>
            <td>{{.current_stock}}</td>
            <td>{{.min_stock}}</td>
            <td>{{.recommended_action}}</td>
          </tr>
          {{end}}{{end}}
        </table>
        
        <h3>Warning Items (Below Reorder Point)</h3>
        <table>
          <tr><th>Part</th><th>Current</th><th>Reorder Pt</th><th>Supplier</th></tr>
          {{range .inventory_alerts}}{{if eq .status "WARNING"}}
          <tr>
            <td>{{.part_name}}</td>
            <td>{{.current_stock}}</td>
            <td>{{.reorder_point}}</td>
            <td>{{.supplier}}</td>
          </tr>
          {{end}}{{end}}
        </table>
    input: inventory_alerts
```

### 1.5 Expected Output Example

```json
{
  "report_date": "2026-01-31T06:00:00Z",
  "summary": {
    "total_parts": 50,
    "critical_items": 3,
    "warning_items": 7,
    "healthy_stock": 40
  },
  "alerts": [
    {
      "alert_id": "INV-PART-0001-20260131",
      "part_id": "PART-0001",
      "part_name": "NVIDIA GeForce RTX 4090",
      "category": "GPU",
      "current_stock": 2,
      "min_stock": 5,
      "reorder_point": 10,
      "status": "CRITICAL",
      "days_until_stockout": 1.4,
      "recommended_action": "urgent_reorder",
      "supplier": "TechCorp Wholesale",
      "lead_time": 2
    },
    {
      "alert_id": "INV-PART-0015-20260131",
      "part_id": "PART-0015",
      "part_name": "Samsung 990 Pro 2TB NVMe",
      "category": "Storage",
      "current_stock": 8,
      "min_stock": 5,
      "reorder_point": 10,
      "status": "WARNING",
      "days_until_stockout": 5.6,
      "recommended_action": "planned_reorder",
      "supplier": "Component Direct",
      "lead_time": 3
    }
  ]
}
```

### 1.6 Success Criteria

- [x] Pipeline executes automatically at scheduled time
- [x] All inventory items are evaluated against thresholds
- [x] Alerts are generated for items below reorder point
- [x] Notifications are sent to appropriate stakeholders
- [x] Historical movement data is considered in analysis
- [x] Lead time from suppliers is factored into urgency

---

## Scenario 2: Vendor Price Comparison

### 2.1 Overview

The Vendor Price Comparison scenario demonstrates real-time pricing intelligence across multiple suppliers. The system aggregates pricing data from distributed vendor APIs, performs comparative analysis, and identifies optimal procurement opportunities based on price, availability, and supplier reliability metrics.

### 2.2 Business Context

**Actors:**
- Procurement Manager: Makes purchasing decisions
- Vendor APIs: External pricing data sources
- System: Price aggregation and analysis engine

**Goals:**
- Monitor price fluctuations across all suppliers
- Identify best-price opportunities
- Track price volatility for contract negotiations
- Alert on significant price changes (>5%)

### 2.3 Data Model Involved

| Entity | Role in Scenario | Key Fields |
|--------|-----------------|------------|
| `supplier_pricing` | Primary source | `pricing_id`, `unit_price`, `availability_status` |
| `suppliers` | Supplier metadata | `supplier_id`, `reliability_score`, `lead_time_days` |
| `parts_inventory` | Product reference | `part_id`, `name`, `category`, `unit_cost` |

### 2.4 Workflow Steps

#### Step 1: Price Data Ingestion

```yaml
# Pipeline: Pricing Monitor
trigger:
  type: scheduled
  cron: "0 */6 * * *"  # Every 6 hours
  
steps:
  - name: query-stale-pricing
    plugin: Input.sql
    config:
      connection: "sqlite://./demo/repair_shop.db"
      query: |
        SELECT 
          sp.pricing_id,
          sp.supplier_id,
          sp.part_id,
          sp.unit_price as current_price,
          sp.availability_status,
          sp.last_updated,
          s.supplier_name,
          s.api_endpoint,
          s.api_key,
          s.reliability_score,
          s.lead_time_days,
          p.name as part_name,
          p.category,
          p.unit_cost as our_cost
        FROM supplier_pricing sp
        JOIN suppliers s ON sp.supplier_id = s.supplier_id
        JOIN parts_inventory p ON sp.part_id = p.part_id
        WHERE sp.last_updated < datetime('now', '-6 hours')
           OR sp.availability_status = 'unknown'
    output: pricing_checklist
```

#### Step 2: Parallel Vendor API Calls

```yaml
  - name: fetch-vendor-prices
    plugin: Input.http_parallel
    config:
      source_field: "api_endpoint"
      method: "POST"
      headers:
        Content-Type: "application/json"
        X-API-Key: "{{.api_key}}"
      body_template: |
        {
          "part_numbers": ["{{.part_id}}"],
          "currency": "USD",
          "include_availability": true
        }
      timeout: 30s
      max_concurrent: 5
      retry_policy:
        max_attempts: 3
        backoff: exponential
    input: pricing_checklist
    output: vendor_responses
```

#### Step 3: Price Change Detection

```yaml
  - name: detect-price-changes
    plugin: Process.transform
    config:
      transformation: |
        function process(record) {
          const newPrice = record.vendor_response.price;
          const oldPrice = record.current_price;
          const changePct = ((newPrice - oldPrice) / oldPrice) * 100;
          
          return {
            ...record,
            new_price: newPrice,
            price_change_pct: changePct.toFixed(2),
            price_change_abs: (newPrice - oldPrice).toFixed(2),
            availability_status: record.vendor_response.availability,
            stock_level: record.vendor_response.stock_quantity,
            significant_change: Math.abs(changePct) > 5.0,
            price_direction: changePct > 0 ? "increased" : "decreased",
            updated_at: new Date().toISOString()
          };
        }
    input: vendor_responses
    output: price_changes
```

#### Step 4: Competitive Analysis

```yaml
  - name: compare-across-vendors
    plugin: Process.aggregate
    config:
      group_by: "part_id"
      aggregations:
        - field: "new_price"
          operation: "min"
          alias: "best_price"
        - field: "new_price"
          operation: "max"
          alias: "worst_price"
        - field: "new_price"
          operation: "avg"
          alias: "avg_price"
        - field: "reliability_score"
          operation: "max"
          alias: "best_reliability"
      calculations:
        - name: "price_variance"
          expression: "((worst_price - best_price) / best_price) * 100"
        - name: "savings_opportunity"
          expression: "(our_cost - best_price) * 10"  # Assuming qty 10
    input: price_changes
    output: comparison_summary
```

#### Step 5: Alert and Report Generation

```yaml
  - name: generate-price-alerts
    plugin: Process.transform
    config:
      filter: "significant_change == true"
      template: |
        {
          "alert_type": "PRICE_{{.price_direction | upper}}",
          "priority": "{{if gt .price_change_pct 10}}HIGH{{else}}MEDIUM{{end}}",
          "part_id": "{{.part_id}}",
          "part_name": "{{.part_name}}",
          "supplier": "{{.supplier_name}}",
          "old_price": {{.current_price}},
          "new_price": {{.new_price}},
          "change_pct": {{.price_change_pct}},
          "change_abs": {{.price_change_abs}},
          "availability": "{{.availability_status}}",
          "recommendation": "{{if gt .price_change_pct 0}}Consider alternative supplier{{else}}Good buying opportunity{{end}}"
        }
    input: price_changes
    output: price_alerts
```

#### Step 6: Database Update

```yaml
  - name: update-pricing-database
    plugin: Output.sql
    config:
      connection: "sqlite://./demo/repair_shop.db"
      table: "supplier_pricing"
      operation: "update"
      key_field: "pricing_id"
      mappings:
        pricing_id: "{{.pricing_id}}"
        unit_price: "{{.new_price}}"
        availability_status: "{{.availability_status}}"
        last_updated: "{{.updated_at}}"
        price_change_pct: "{{.price_change_pct}}"
    input: price_changes
```

### 2.5 Expected Output Example

```json
{
  "monitoring_timestamp": "2026-01-31T12:00:00Z",
  "summary": {
    "parts_checked": 50,
    "vendors_queried": 5,
    "api_success_rate": 96.0,
    "significant_changes": 8,
    "price_increases": 5,
    "price_decreases": 3
  },
  "comparison_matrix": [
    {
      "part_id": "PART-0001",
      "part_name": "NVIDIA GeForce RTX 4090",
      "category": "GPU",
      "best_price": 1529.99,
      "best_supplier": "Global Tech Parts",
      "worst_price": 1689.99,
      "worst_supplier": "Component Direct",
      "avg_price": 1599.99,
      "price_variance": 10.46,
      "savings_opportunity": 700.00,
      "vendor_breakdown": [
        {
          "supplier": "TechCorp Wholesale",
          "price": 1599.99,
          "availability": "in_stock",
          "reliability": 0.98,
          "lead_time": 2
        },
        {
          "supplier": "Global Tech Parts",
          "price": 1529.99,
          "availability": "low_stock",
          "reliability": 0.88,
          "lead_time": 5
        }
      ]
    }
  ],
  "alerts": [
    {
      "alert_type": "PRICE_INCREASE",
      "priority": "HIGH",
      "part_id": "PART-0001",
      "part_name": "NVIDIA GeForce RTX 4090",
      "supplier": "TechCorp Wholesale",
      "old_price": 1599.99,
      "new_price": 1679.99,
      "change_pct": 5.0,
      "change_abs": 80.00,
      "recommendation": "Consider alternative supplier"
    }
  ]
}
```

### 2.6 Success Criteria

- [x] All supplier APIs are queried within timeout window
- [x] Price changes are detected with >95% accuracy
- [x] Alerts generated for changes exceeding threshold
- [x] Comparison matrix shows competitive landscape
- [x] Database updated with new pricing information
- [x] Historical price trends are maintained

---

## Scenario 3: Predictive Maintenance

### 3.1 Overview

The Predictive Maintenance scenario demonstrates machine learning-based demand forecasting using historical stock movement patterns, repair job frequency, and seasonal trends. The system analyzes time-series data to predict future inventory needs and optimize safety stock levels.

### 3.2 Business Context

**Actors:**
- Operations Manager: Reviews demand forecasts
- Data Science System: ML model training and prediction
- Procurement Team: Acts on reorder recommendations

**Goals:**
- Predict parts demand 7-30 days in advance
- Optimize safety stock levels by category
- Reduce stockouts while minimizing carrying costs
- Identify seasonal patterns in repair volume

### 3.3 Data Model Involved

| Entity | Role in Scenario | Key Fields |
|--------|-----------------|------------|
| `stock_movements` | Training data | `movement_type`, `quantity_change`, `created_at` |
| `repair_jobs` | Demand signals | `device_type`, `created_at`, `status` |
| `job_parts` | Parts usage correlation | `job_id`, `part_id`, `quantity_used` |
| `parts_inventory` | Current baseline | `part_id`, `category`, `current_stock` |

### 3.4 Workflow Steps

#### Step 1: Feature Extraction

```yaml
# Pipeline: Predictive Maintenance
trigger:
  type: scheduled
  cron: "0 1 * * 0"  # Weekly on Sunday at 1 AM
  
steps:
  - name: extract-movement-features
    plugin: Input.sql
    config:
      connection: "sqlite://./demo/repair_shop.db"
      query: |
        WITH daily_movements AS (
          SELECT 
            part_id,
            date(created_at) as date,
            SUM(CASE WHEN movement_type IN ('sale', 'adjustment') 
                     THEN ABS(quantity_change) ELSE 0 END) as daily_usage,
            SUM(CASE WHEN movement_type = 'restock' 
                     THEN quantity_change ELSE 0 END) as daily_restock,
            COUNT(*) as transaction_count
          FROM stock_movements
          WHERE created_at >= date('now', '-90 days')
          GROUP BY part_id, date(created_at)
        ),
        job_demand AS (
          SELECT 
            jp.part_id,
            date(rj.created_at) as date,
            SUM(jp.quantity_used) as job_usage,
            rj.device_type
          FROM job_parts jp
          JOIN repair_jobs rj ON jp.job_id = rj.job_id
          WHERE rj.created_at >= date('now', '-90 days')
            AND rj.status IN ('completed', 'in_progress')
          GROUP BY jp.part_id, date(rj.created_at), rj.device_type
        )
        SELECT 
          p.part_id,
          p.name,
          p.category,
          p.current_stock,
          dm.date,
          COALESCE(dm.daily_usage, 0) as usage_qty,
          COALESCE(dm.daily_restock, 0) as restock_qty,
          COALESCE(jd.job_usage, 0) as job_demand,
          jd.device_type,
          strftime('%w', dm.date) as day_of_week,
          strftime('%W', dm.date) as week_of_year
        FROM parts_inventory p
        LEFT JOIN daily_movements dm ON p.part_id = dm.part_id
        LEFT JOIN job_demand jd ON p.part_id = jd.part_id 
                              AND dm.date = jd.date
        WHERE dm.date IS NOT NULL
        ORDER BY p.part_id, dm.date
    output: feature_data
```

#### Step 2: Time Series Aggregation

```yaml
  - name: calculate-rolling-metrics
    plugin: Process.transform
    config:
      window_operations:
        - type: rolling_mean
          field: "usage_qty"
          window: 7
          alias: "avg_7d_usage"
        - type: rolling_mean
          field: "usage_qty"
          window: 30
          alias: "avg_30d_usage"
        - type: rolling_std
          field: "usage_qty"
          window: 14
          alias: "std_14d_usage"
        - type: lag
          field: "usage_qty"
          periods: [1, 7, 14]
      categorical_encoding:
        - field: "category"
          method: "one_hot"
        - field: "day_of_week"
          method: "ordinal"
        - field: "device_type"
          method: "frequency"
    input: feature_data
    output: engineered_features
```

#### Step 3: ML Model Prediction

```yaml
  - name: predict-demand
    plugin: AI_Model.predict
    config:
      model_type: "time_series_forecast"
      model_path: "./models/inventory_demand_v1.pkl"
      horizon: 30  # Days
      features:
        - avg_7d_usage
        - avg_30d_usage
        - std_14d_usage
        - day_of_week
        - week_of_year
        - category_encoded
      confidence_interval: 0.95
    input: engineered_features
    output: demand_forecasts
```

#### Step 4: Safety Stock Optimization

```yaml
  - name: optimize-safety-stock
    plugin: Process.transform
    config:
      calculation: |
        // Safety Stock = Z × σLT × √(LT)
        // Where:
        // Z = service level factor (1.65 for 95%)
        // σLT = standard deviation of demand during lead time
        // LT = lead time in days
        
        const Z = 1.65;  // 95% service level
        const leadTime = record.lead_time_days || 3;
        const demandStd = record.std_14d_usage || 0;
        const avgDemand = record.avg_30d_usage || 0;
        
        const safetyStock = Z * demandStd * Math.sqrt(leadTime);
        const reorderPoint = (avgDemand * leadTime) + safetyStock;
        const daysOfSupply = record.current_stock / (avgDemand || 1);
        
        return {
          ...record,
          safety_stock: Math.ceil(safetyStock),
          recommended_reorder_point: Math.ceil(reorderPoint),
          days_of_supply: daysOfSupply.toFixed(1),
          stock_status: daysOfSupply < leadTime ? "CRITICAL" : 
                       daysOfSupply < (leadTime * 2) ? "LOW" : "HEALTHY"
        };
    input: demand_forecasts
    output: optimization_results
```

#### Step 5: Recommendation Generation

```yaml
  - name: generate-recommendations
    plugin: Process.transform
    config:
      rules:
        - condition: "stock_status == 'CRITICAL' AND predicted_7d_demand > current_stock"
          action: "EMERGENCY_ORDER"
          priority: 1
          message: "Urgent: Order {{predicted_7d_demand - current_stock}} units immediately"
        - condition: "current_stock < recommended_reorder_point"
          action: "PLANNED_ORDER"
          priority: 2
          message: "Schedule order of {{recommended_reorder_point - current_stock}} units"
        - condition: "safety_stock > (current_stock * 0.5)"
          action: "INCREASE_SAFETY_STOCK"
          priority: 3
          message: "Consider increasing safety stock to {{safety_stock}}"
    input: optimization_results
    output: recommendations
```

#### Step 6: Report Generation

```yaml
  - name: create-forecast-report
    plugin: Output.report
    config:
      format: "pdf"
      template: "predictive_maintenance_report"
      sections:
        - title: "Executive Summary"
          content: |
            Forecast Period: {{.forecast_start}} to {{.forecast_end}}
            Parts Analyzed: {{.parts_count}}
            Critical Items: {{.critical_count}}
            Recommended Actions: {{.action_count}}
        - title: "Category Forecasts"
          chart_type: "line"
          data: "{{.category_forecasts}}"
        - title: "Reorder Recommendations"
          table_data: "{{.recommendations}}"
    input: recommendations
```

### 3.5 Expected Output Example

```json
{
  "forecast_period": {
    "start": "2026-01-31",
    "end": "2026-03-02",
    "horizon_days": 30
  },
  "model_performance": {
    "mae": 2.34,
    "rmse": 3.87,
    "mape": 12.5,
    "r2": 0.84
  },
  "forecasts_by_category": [
    {
      "category": "GPU",
      "current_demand_rate": 4.2,
      "predicted_30d_demand": 126,
      "confidence_interval": [108, 144],
      "seasonal_factor": 1.15,
      "trend": "increasing"
    },
    {
      "category": "Storage",
      "current_demand_rate": 8.7,
      "predicted_30d_demand": 261,
      "confidence_interval": [245, 277],
      "seasonal_factor": 0.98,
      "trend": "stable"
    }
  ],
  "critical_recommendations": [
    {
      "part_id": "PART-0001",
      "part_name": "NVIDIA GeForce RTX 4090",
      "category": "GPU",
      "current_stock": 2,
      "predicted_7d_demand": 8,
      "predicted_30d_demand": 34,
      "safety_stock": 12,
      "recommended_reorder_point": 28,
      "days_of_supply": "0.5",
      "stock_status": "CRITICAL",
      "action": "EMERGENCY_ORDER",
      "priority": 1,
      "message": "Urgent: Order 32 units immediately"
    }
  ]
}
```

### 3.6 Success Criteria

- [x] 90-day historical data successfully ingested
- [x] Feature engineering produces relevant ML inputs
- [x] Model generates predictions with <15% MAPE
- [x] Safety stock calculations follow industry formulas
- [x] Recommendations are prioritized and actionable
- [x] Category-level trends are identified
- [x] Confidence intervals provided for uncertainty

---

## Scenario 4: Chat-based Analytics

### 4.1 Overview

The Chat-based Analytics scenario demonstrates natural language interaction with the Mimir AIP platform through the Model Context Protocol (MCP) server. Users can query inventory, pricing, and operational data using conversational English, with the system translating queries into SQL and returning formatted responses.

### 4.2 Business Context

**Actors:**
- Shop Manager: Queries inventory status via chat
- Technicians: Check part availability
- Business Owner: Requests sales reports

**Goals:**
- Enable non-technical users to query data
- Provide instant answers to common questions
- Support complex multi-step analytical queries
- Maintain conversation context across queries

### 4.3 MCP Server Integration

#### Tool Registration

```yaml
# MCP Server Configuration
mcp_server:
  enabled: true
  port: 8081
  
  tools:
    - name: "query_inventory"
      description: "Query parts inventory levels and availability"
      parameters:
        - name: "part_name"
          type: "string"
          required: false
        - name: "category"
          type: "string"
          required: false
        - name: "stock_status"
          type: "string"
          enum: ["low", "critical", "all"]
          required: false
    
    - name: "compare_prices"
      description: "Compare prices across suppliers for a specific part"
      parameters:
        - name: "part_id"
          type: "string"
          required: true
    
    - name: "get_sales_report"
      description: "Generate sales report for a date range"
      parameters:
        - name: "start_date"
          type: "date"
          required: true
        - name: "end_date"
          type: "date"
          required: true
        - name: "group_by"
          type: "string"
          enum: ["day", "week", "month", "category"]
          required: false
    
    - name: "analyze_repair_trends"
      description: "Analyze repair job trends and patterns"
      parameters:
        - name: "device_type"
          type: "string"
          required: false
        - name: "time_period"
          type: "string"
          enum: ["week", "month", "quarter"]
          required: false
```

### 4.4 Conversation Examples

#### Example 1: Inventory Query

**User:** "Which parts are running low on stock?"

**System Processing:**
```
1. Intent Recognition: INVENTORY_QUERY
2. Entity Extraction: 
   - status: "low"
3. SQL Generation:
   SELECT part_id, name, category, current_stock, reorder_point
   FROM parts_inventory
   WHERE current_stock <= reorder_point
   ORDER BY (current_stock - min_stock) ASC
4. Result Formatting: Table with status indicators
```

**System Response:**
```
Here are the parts running low on stock:

| Part Name | Category | Current Stock | Reorder Point | Status |
|-----------|----------|---------------|---------------|--------|
| NVIDIA GeForce RTX 4090 | GPU | 2 | 10 | CRITICAL |
| Samsung 990 Pro 2TB | Storage | 8 | 10 | WARNING |
| Corsair Vengeance 32GB | Memory | 5 | 8 | WARNING |
| Intel Core i9-13900K | CPU | 3 | 6 | CRITICAL |

3 parts are below reorder point. I recommend placing orders for the 
CRITICAL items (RTX 4090 and i9-13900K) within 24 hours.
```

#### Example 2: Price Comparison

**User:** "Compare GPU prices across all suppliers"

**System Processing:**
```
1. Intent Recognition: PRICE_COMPARISON
2. Entity Extraction:
   - category: "GPU"
3. SQL Generation:
   SELECT p.name, s.supplier_name, sp.unit_price, 
          sp.availability_status, s.lead_time_days
   FROM parts_inventory p
   JOIN supplier_pricing sp ON p.part_id = sp.part_id
   JOIN suppliers s ON sp.supplier_id = s.supplier_id
   WHERE p.category = 'GPU'
   ORDER BY p.name, sp.unit_price ASC
4. Result Formatting: Grouped comparison table
```

**System Response:**
```
Here's the price comparison for all GPUs across suppliers:

**NVIDIA GeForce RTX 4090:**
- Global Tech Parts: $1,529.99 (low stock, 5-day lead)
- TechCorp Wholesale: $1,599.99 (in stock, 2-day lead) ⭐
- Component Direct: $1,689.99 (in stock, 3-day lead)

**AMD Radeon RX 7900 XTX:**
- PC Parts Express: $949.99 (in stock, 2-day lead) ⭐
- TechCorp Wholesale: $999.99 (in stock, 2-day lead)
- MicroCenter Supply: $1,049.99 (in stock, 1-day lead)

⭐ = Best balance of price and availability

The price variance across suppliers ranges from 5% to 15%, with 
storage items showing the most competitive pricing.
```

#### Example 3: Sales Analysis

**User:** "Show me last week's sales by category"

**System Processing:**
```
1. Intent Recognition: SALES_REPORT
2. Entity Extraction:
   - date_range: "last week"
   - group_by: "category"
3. SQL Generation:
   SELECT p.category, 
          COUNT(DISTINCT s.sale_id) as transaction_count,
          SUM(s.total_amount) as total_revenue,
          AVG(s.total_amount) as avg_transaction
   FROM sales s
   LEFT JOIN repair_jobs rj ON s.job_id = rj.job_id
   LEFT JOIN job_parts jp ON rj.job_id = jp.job_id
   LEFT JOIN parts_inventory p ON jp.part_id = p.part_id
   WHERE s.sale_date >= date('now', '-7 days')
   GROUP BY p.category
   ORDER BY total_revenue DESC
```

**System Response:**
```
Sales Report: January 24-31, 2026

| Category | Transactions | Revenue | Avg Transaction |
|----------|--------------|---------|-----------------|
| GPU | 12 | $18,450.00 | $1,537.50 |
| CPU | 8 | $5,920.00 | $740.00 |
| Storage | 15 | $3,875.00 | $258.33 |
| Memory | 10 | $2,450.00 | $245.00 |
| Motherboard | 6 | $2,100.00 | $350.00 |

**Summary:**
- Total Revenue: $32,795.00
- Total Transactions: 51
- Best Category: GPU (56% of revenue)
- Busiest Day: Wednesday (Jan 29) - $8,200

Compared to previous week: +12% revenue, +8% transactions
```

#### Example 4: Complex Multi-Step Query

**User:** "What's the most profitable part category this month, and which suppliers should I focus on for reordering?"

**System Processing:**
```
Step 1: Identify most profitable category
Step 2: Find low-stock items in that category
Step 3: Identify best suppliers for those items
Step 4: Compile recommendations
```

**System Response:**
```
Let me analyze that for you...

**Most Profitable Category: GPU**
- Revenue: $48,320 (42% of monthly total)
- Profit Margin: 28% (after parts costs)
- Top selling: RTX 4090 (18 units), RTX 4080 (12 units)

**Critical Reorder Needs for GPUs:**

1. **NVIDIA GeForce RTX 4090** (Current: 2, Reorder Pt: 10)
   - Recommended Supplier: TechCorp Wholesale
   - Price: $1,599.99 | Availability: In Stock | Lead: 2 days
   - Order Quantity: 15 units (covers 3 weeks demand)

2. **NVIDIA GeForce RTX 4080** (Current: 4, Reorder Pt: 8)
   - Recommended Supplier: TechCorp Wholesale
   - Price: $1,199.99 | Availability: In Stock | Lead: 2 days
   - Order Quantity: 12 units

**Supplier Priority Ranking:**
1. TechCorp Wholesale (98% reliability, best GPU pricing)
2. PC Parts Express (96% reliability, competitive prices)
3. MicroCenter Supply (fastest lead times for urgent items)

Would you like me to generate a purchase order for these items?
```

### 4.5 Context Management

#### Multi-turn Conversation Example

**Turn 1 - User:** "Show me parts low on stock"

**Turn 2 - User:** "What about GPUs specifically?"

**System Context:**
```json
{
  "conversation_id": "conv_20260131_001",
  "context": {
    "previous_query": "parts low on stock",
    "previous_filter": "stock_status: low",
    "current_focus": "inventory",
    "implicit_refinement": {
      "category": "GPU"
    }
  },
  "sql_modification": {
    "base_query": "low_stock_query",
    "add_filter": "category = 'GPU'"
  }
}
```

**Turn 3 - User:** "Order the RTX 4090s from TechCorp"

**System Context:**
```json
{
  "conversation_id": "conv_20260131_001",
  "context": {
    "action_intent": "CREATE_ORDER",
    "referenced_item": "RTX 4090",
    "referenced_supplier": "TechCorp Wholesale",
    "inferred_quantity": "15 (from previous recommendation)",
    "requires_confirmation": true
  }
}
```

### 4.6 Error Handling Examples

#### Ambiguous Query Resolution

**User:** "How's the 4090 doing?"

**System Response:**
```
I found multiple parts matching "4090":

1. NVIDIA GeForce RTX 4090 (GPU) - $1,599.99
2. Intel Core i9-13900K (CPU) - $589.99 (model number contains "13900")

Which one would you like information about? 
Or type "both" to see all matches.
```

#### Data Not Available

**User:** "Show me sales for next month"

**System Response:**
```
I'm unable to show sales for next month as that data doesn't exist yet. 

I can show you:
- Sales forecasts based on historical trends
- This month's sales so far
- Same month last year for comparison

Which would be helpful?
```

#### Complex Query Breakdown

**User:** "Analyze everything"

**System Response:**
```
That's a broad request! Let me help narrow it down. What would you 
like to analyze?

**Common Analysis Options:**
- Inventory health and reorder recommendations
- Sales trends and revenue analysis  
- Supplier performance comparison
- Repair job patterns by device type
- Price volatility across vendors

You can also ask something specific like:
"Which GPUs have increased in price this week?"
```

### 4.7 Success Criteria

- [x] Natural language queries are correctly interpreted (>90% accuracy)
- [x] SQL queries are generated without errors
- [x] Results are formatted in readable, contextual responses
- [x] Multi-turn conversations maintain context
- [x] Ambiguous queries trigger clarification prompts
- [x] Response time is <3 seconds for simple queries
- [x] Complex analytical queries complete within 10 seconds

---

## Cross-Scenario Integration

### Integrated Workflow: Daily Operations Dashboard

The four scenarios can be combined into a comprehensive daily operations workflow:

```yaml
# Master Pipeline: Daily Operations Dashboard
trigger:
  type: scheduled
  cron: "0 7 * * 1-5"  # 7 AM weekdays

workflows:
  - id: inventory_check
    scenario: "Scenario 1"
    pipeline: "Daily Inventory Sync"
    priority: 1
    
  - id: price_update
    scenario: "Scenario 2"  
    pipeline: "Pricing Monitor"
    priority: 2
    depends_on: inventory_check
    
  - id: demand_forecast
    scenario: "Scenario 3"
    pipeline: "Predictive Maintenance"
    priority: 3
    schedule: "0 1 * * 0"  # Weekly on Sunday
    
  - id: morning_report
    scenario: "All Scenarios"
    pipeline: "Generate Dashboard Report"
    priority: 4
    depends_on: [inventory_check, price_update]
    distribution:
      - email: "manager@repairshop.local"
      - chat: "morning_briefing"
```

### Data Flow Integration

```
┌─────────────────────────────────────────────────────────────────┐
│                     DAILY OPERATIONS FLOW                       │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  06:00 - Inventory Check (Scenario 1)                          │
│          ↓                                                      │
│          ├─→ Stock levels evaluated                             │
│          ├─→ Alerts generated                                   │
│          └─→ Reorder recommendations created                    │
│                                                                 │
│  06:15 - Price Update (Scenario 2)                             │
│          ↓                                                      │
│          ├─→ Vendor APIs queried                                │
│          ├─→ Price changes detected                             │
│          └─→ Comparison matrix updated                          │
│                                                                 │
│  07:00 - Morning Report Generation                              │
│          ↓                                                      │
│          ├─→ Aggregates inventory status                        │
│          ├─→ Includes price alerts                              │
│          ├─→ References forecast data                           │
│          └─→ Distributed via email + chat                       │
│                                                                 │
│  All Day - Chat Analytics (Scenario 4)                         │
│          ↓                                                      │
│          ├─→ Ad-hoc queries answered                            │
│          ├─→ Real-time availability checks                      │
│          └─→ Order confirmations                                │
│                                                                 │
│  Weekly - Predictive Analysis (Scenario 3)                     │
│          ↓                                                      │
│          ├─→ Demand forecasts generated                         │
│          ├─→ Safety stock optimized                             │
│          └─→ Procurement recommendations updated                │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## Summary

The four demonstration scenarios showcase the comprehensive capabilities of the Mimir AIP platform:

| Scenario | Primary Feature | Business Value | Technical Highlight |
|----------|----------------|----------------|---------------------|
| **1. Daily Inventory Check** | Automated monitoring | Prevents stockouts | Scheduled SQL pipelines |
| **2. Vendor Price Comparison** | Multi-source aggregation | Cost optimization | Parallel HTTP processing |
| **3. Predictive Maintenance** | ML-based forecasting | Inventory optimization | Time-series feature engineering |
| **4. Chat-based Analytics** | Natural language interface | Accessibility | MCP tool integration |

These scenarios can be executed independently or integrated into a cohesive operational intelligence platform for the computer repair shop business.

---

**Last Updated:** 2026-01-31  
**Version:** 1.0.0  
**Documentation Type:** Use Case Specification

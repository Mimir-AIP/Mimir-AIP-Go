# Automatic Scenario Generation Implementation

## Overview
This implementation adds automatic generation of realistic simulation scenarios for newly created Digital Twins from data ingestion. When a Digital Twin is created, the system automatically generates three default scenario types that can be immediately used for simulation and testing.

## Implementation Summary

### Files Modified
- **handlers.go** - Added helper functions and updated `createDigitalTwinFromOntology`
  - Added `database/sql` import (line 4)
  - Modified `createDigitalTwinFromOntology` (lines 1764-1781) to generate and save scenarios
  - Added response fields for scenario IDs (lines 1483-1498)

### New Functions Added (handlers.go:1784-2110)

#### 1. `generateDefaultScenarios(twin *DigitalTwin.DigitalTwin) []DigitalTwin.SimulationScenario`
**Location:** handlers.go:1784-1832  
**Purpose:** Main function that generates 3 default scenario types for a digital twin

**Returns:** Array of 3 SimulationScenario objects:
1. **Baseline Scenario** (30 steps)
   - Type: `"baseline"`
   - No events
   - Establishes normal operations baseline
   
2. **Data Quality Issues Scenario** (40 steps)
   - Type: `"data_quality_issue"`
   - Simulates missing/invalid data
   - Events: resource unavailability, process failures, restoration
   
3. **Capacity Stress Test Scenario** (50 steps)
   - Type: `"capacity_test"`
   - Simulates high volume/load conditions
   - Events: demand surges, capacity changes, market shifts

#### 2. `generateDataQualityEvents(twin *DigitalTwin.DigitalTwin) []DigitalTwin.SimulationEvent`
**Location:** handlers.go:1833-1925  
**Purpose:** Creates events simulating data quality problems

**Generated Events:**
- **Step 5:** Resource unavailable (data source failure)
  - Event type: `EventResourceUnavailable`
  - Severity: High
  - Includes propagation rules based on relationships
  
- **Step 15:** Process failure (data validation failure)
  - Event type: `EventProcessFailure`
  - Severity: Medium
  - Reason: "Data validation failure - invalid schema"
  
- **Step 25:** Resource restoration
  - Event type: `EventResourceAvailable`
  - Severity: Low
  - Restores failed data source
  
- **Step 30:** Data quality constraint added
  - Event type: `EventPolicyConstraintAdd`
  - Severity: Medium
  - 15% capacity reduction due to quality checks

**Features:**
- Targets up to 30% of entities
- Adds impact propagation through relationships (60% impact, 2 step delay)
- Realistic timing with restoration events

#### 3. `generateCapacityTestEvents(twin *DigitalTwin.DigitalTwin) []DigitalTwin.SimulationEvent`
**Location:** handlers.go:1927-2053  
**Purpose:** Creates events simulating high load and stress conditions

**Generated Events:**
- **Step 5:** Initial demand surge
  - 80% increase in utilization
  - 70% propagation impact
  
- **Step 15:** Secondary demand surge (cascading)
  - 120% increase
  - 80% propagation impact
  - Severity: High
  
- **Step 20:** Capacity reduction under load
  - 30% capacity reduction
  - Simulates resource throttling
  
- **Step 25:** External market shift
  - 50% demand impact
  - Severity: Critical
  - 50% propagation with 2 step delay
  
- **Step 35:** Process optimization (mitigation)
  - 25% efficiency improvement
  - Load balancing applied
  
- **Step 45:** Demand normalization
  - Return to 60% of peak load
  - System recovery

**Features:**
- Progressive load increase
- Realistic mitigation strategies
- Recovery phase
- Multiple propagation patterns

#### 4. `getEntityRelationshipTypes(twin *DigitalTwin.DigitalTwin, entityURI string) []string`
**Location:** handlers.go:2056-2071  
**Purpose:** Helper function to extract unique relationship types for an entity

**Usage:** Used by event generators to create appropriate propagation rules based on actual relationships in the twin

#### 5. `saveScenarioToDatabase(ctx context.Context, persistence interface{}, scenario *DigitalTwin.SimulationScenario) error`
**Location:** handlers.go:2073-2110  
**Purpose:** Persists simulation scenarios to the database

**Features:**
- Uses type assertion to get database connection from persistence backend
- Serializes events to JSON
- Inserts into `simulation_scenarios` table
- Returns descriptive errors

## API Response Changes

When creating a Digital Twin through data ingestion (`POST /api/v1/data/select`), the response now includes:

```json
{
  "digital_twin": {
    "id": "twin_1234567890",
    "name": "Dataset Name",
    "description": "Digital Twin created from ontology: ...",
    "model_type": "data_model",
    "entity_count": 5,
    "relationship_count": 3,
    "scenario_ids": [
      "scenario_twin_1234567890_baseline",
      "scenario_twin_1234567890_data_quality",
      "scenario_twin_1234567890_capacity"
    ],
    "scenario_count": 3
  }
}
```

## Scenario Characteristics

### Realistic Event Targeting
- Events target actual entity URIs from the twin
- Respects entity types and relationships
- Up to 30% of entities targeted in stress scenarios

### Smart Propagation Rules
- Automatically discovers relationship types
- Configures propagation based on scenario context:
  - Data quality: 60% impact, 2 step delay
  - Capacity test: 50-80% impact, 1-2 step delay
- Only adds propagation if relationships exist

### Domain-Appropriate Scenarios
The scenarios are designed to be realistic across different domains:
- **Person/Organization:** Simulates workforce issues, resource constraints
- **Process/Department:** Simulates workflow failures, capacity limits
- **Resource/System:** Simulates hardware failures, load issues

### Progressive Complexity
1. **Baseline:** Establishes normal behavior (0 events)
2. **Data Quality:** Moderate complexity (4 events, recovery included)
3. **Capacity Test:** High complexity (6 events, full stress cycle)

## Database Schema
Scenarios are stored in the existing `simulation_scenarios` table:

```sql
CREATE TABLE simulation_scenarios (
    id TEXT PRIMARY KEY,
    twin_id TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    scenario_type TEXT,
    events TEXT NOT NULL,  -- JSON array of events
    duration INTEGER,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (twin_id) REFERENCES digital_twins(id) ON DELETE CASCADE
);
```

## Usage Example

### 1. Upload and create Digital Twin with scenarios:
```bash
# Upload data file
curl -X POST http://localhost:8080/api/v1/data/upload \
  -F "file=@data.csv" \
  -F "plugin_type=Input" \
  -F "plugin_name=csv"

# Select data and create twin (scenarios auto-generated)
curl -X POST http://localhost:8080/api/v1/data/select \
  -H "Content-Type: application/json" \
  -d '{
    "upload_id": "upload_...",
    "selected_columns": ["name", "email", "department"],
    "create_twin": true
  }'
```

Response includes `scenario_ids` array.

### 2. List scenarios for a twin:
```bash
curl http://localhost:8080/api/v1/twins/{twin_id}/scenarios
```

### 3. Run a generated scenario:
```bash
curl -X POST http://localhost:8080/api/v1/twins/{twin_id}/scenarios/{scenario_id}/run \
  -H "Content-Type: application/json" \
  -d '{
    "snapshot_interval": 10,
    "max_steps": 100
  }'
```

## Testing

A comprehensive test suite is included in `scenario_generation_test.go`:

```bash
go test -v -run TestGenerateDefaultScenarios
```

**Test Coverage:**
- ✓ Generates exactly 3 scenarios
- ✓ Baseline scenario properties (type, name, duration, event count)
- ✓ Data quality scenario properties and event generation
- ✓ Capacity test scenario properties and event generation
- ✓ Scenario ID format validation
- ✓ Event targeting validation (only targets actual entities)
- ✓ Propagation rule verification

**Test Results:**
```
=== RUN   TestGenerateDefaultScenarios
Data Quality Scenario: 4 events
Capacity Test Scenario: 6 events
Event resource.unavailable has 1 propagation rules
Event demand.surge has 1 propagation rules
Event demand.surge has 2 propagation rules
Event external.market_shift has 1 propagation rules

Scenario Generation Test Summary:
✓ Generated 3 scenarios
✓ Baseline: Baseline Operations (duration: 30, events: 0)
✓ Data Quality: Data Quality Issues (duration: 40, events: 4)
✓ Capacity Test: Capacity Stress Test (duration: 50, events: 6)
--- PASS: TestGenerateDefaultScenarios (0.00s)
PASS
```

## Code References

### Main Functions
- `generateDefaultScenarios()` - handlers.go:1784
- `generateDataQualityEvents()` - handlers.go:1833
- `generateCapacityTestEvents()` - handlers.go:1927
- `saveScenarioToDatabase()` - handlers.go:2073
- `getEntityRelationshipTypes()` - handlers.go:2056

### Integration Points
- `createDigitalTwinFromOntology()` - handlers.go:1764-1781
- API response with scenario IDs - handlers.go:1483-1498

### Event Types Used
Defined in `pipelines/DigitalTwin/event_system.go`:
- `EventResourceUnavailable` (line 11)
- `EventResourceAvailable` (line 12)
- `EventResourceCapacityChange` (line 13)
- `EventDemandSurge` (line 18)
- `EventDemandDrop` (line 19)
- `EventProcessFailure` (line 24)
- `EventProcessOptimization` (line 25)
- `EventPolicyConstraintAdd` (line 31)
- `EventExternalMarketShift` (line 35)

## Benefits

1. **Immediate Testing:** Users can immediately test newly created Digital Twins
2. **Best Practices:** Scenarios follow simulation best practices
3. **Realistic:** Events and timing are domain-appropriate
4. **Comprehensive:** Covers baseline, failure, and stress scenarios
5. **Automated:** No manual scenario creation required
6. **Extensible:** Easy to add new scenario types
7. **Relationship-Aware:** Leverages actual twin structure for propagation

## Future Enhancements

Potential improvements:
- Machine learning to generate scenarios based on historical data patterns
- Domain-specific scenario templates (healthcare, finance, supply chain)
- Configurable scenario parameters (duration, severity, event density)
- Scenario recommendation based on twin characteristics
- Multi-scenario comparison and optimization
- Integration with external monitoring/alerting systems

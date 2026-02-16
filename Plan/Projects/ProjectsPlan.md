# Projects

Projects serve as workspaces for specific use cases, containing pipelines, ontologies, ML models, and digital twins. Each project maintains its own isolated data environment.

## Project Configuration Schema

Projects are defined using YAML configuration files with the following schema:

```yaml
name: string              # Required: Unique project name
description: string       # Optional: Human-readable description
version: string          # Optional: Semantic version (e.g., "1.0.0")
status: string           # Optional: "active" | "archived" | "draft" (default: "active")

metadata:
  created_at: string     # ISO 8601 timestamp (auto-generated)
  updated_at: string     # ISO 8601 timestamp (auto-updated)
  tags: string[]         # Optional: Array of tags for organization

components:
  pipelines: string[]    # Array of pipeline IDs associated with project
  ontologies: string[]   # Array of ontology IDs associated with project
  ml_models: string[]    # Array of ML model IDs associated with project
  digital_twins: string[] # Array of digital twin IDs associated with project

settings:
  timezone: string       # Optional: IANA timezone (default: "UTC")
  environment: string    # Optional: "development" | "staging" | "production"
```

### Validation Rules
- `name`: 3-50 characters, alphanumeric + hyphens/underscores, must be unique
- `version`: Must follow semantic versioning if provided
- `status`: Must be one of allowed values
- All component IDs must reference existing resources

## API Endpoints

### Project Management
- `POST /api/projects` - Create new project
  - Body: Project YAML configuration
  - Returns: Project ID and full configuration
- `GET /api/projects` - List all projects
  - Query params: status, tags, limit, offset
  - Returns: Array of project summaries
- `GET /api/projects/{id}` - Get project details
  - Returns: Full project configuration
- `PUT /api/projects/{id}` - Update project
  - Body: Partial project configuration
  - Returns: Updated project configuration
- `DELETE /api/projects/{id}` - Delete project
  - Cascades: Removes associations but preserves component resources
- `POST /api/projects/{id}/clone` - Clone project
  - Body: New project name
  - Returns: New project ID

### Component Association
- `POST /api/projects/{id}/pipelines/{pipeline_id}` - Associate pipeline
- `DELETE /api/projects/{id}/pipelines/{pipeline_id}` - Remove pipeline association
- `POST /api/projects/{id}/ontologies/{ontology_id}` - Associate ontology
- `DELETE /api/projects/{id}/ontologies/{ontology_id}` - Remove ontology association
- `POST /api/projects/{id}/ml-models/{model_id}` - Associate ML model
- `DELETE /api/projects/{id}/ml-models/{model_id}` - Remove ML model association
- `POST /api/projects/{id}/digital-twins/{dt_id}` - Associate digital twin
- `DELETE /api/projects/{id}/digital-twins/{dt_id}` - Remove digital twin association

## Project Lifecycle

### Creation
1. Validate YAML configuration against schema
2. Check name uniqueness
3. Generate project ID
4. Set timestamps and defaults
5. Store project configuration
6. Return project details

### Updates
1. Validate partial configuration
2. Preserve immutable fields (ID, created_at)
3. Update timestamps
4. Validate component references exist
5. Store updated configuration

### Deletion
1. Remove all component associations
2. Mark project as deleted (soft delete)
3. Preserve component resources for potential recovery
4. Update project status to "archived"

## Integration Points

### Pipeline Integration
- Projects provide execution context for pipelines
- Pipeline results are tagged with project ID
- Project settings influence pipeline behavior (timezone, environment)

### Ontology Integration
- Projects can have multiple ontologies
- First ontology creation triggers project initialization enhancements
- Ontologies define data structure within project scope

### ML Model Integration
- Models are trained on project-specific data
- Model artifacts stored with project association
- Inference requests scoped to project context

### Digital Twin Integration
- Digital twins operate within project boundaries
- Access project ontologies and data
- Output pipelines triggered by digital twin events

## Configuration Options

### Environment Settings
- `development`: Enables debug logging, relaxed validation
- `staging`: Pre-production testing environment
- `production`: Optimized for performance and stability

### Timezone Handling
- All timestamps stored in UTC
- Display times converted to project timezone
- Scheduled operations respect project timezone

## Examples

### Minimal Project
```yaml
name: customer-analytics
description: Customer behavior analysis project
```

### Full Project Configuration
```yaml
name: manufacturing-optimization
description: Digital twin for manufacturing line optimization
version: "2.1.0"
status: active

metadata:
  created_at: "2026-02-16T15:30:00Z"
  updated_at: "2026-02-16T16:45:00Z"
  tags: ["manufacturing", "optimization", "iot"]

components:
  pipelines: ["sensor-data-ingestion", "quality-control-pipeline"]
  ontologies: ["equipment-ontology", "process-ontology"]
  ml_models: ["predictive-maintenance-model"]
  digital_twins: ["production-line-twin"]

settings:
  timezone: "America/New_York"
  environment: "production"
```


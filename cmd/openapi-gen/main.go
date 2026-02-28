// openapi-gen generates the Mimir AIP OpenAPI 3.0 specification and writes it to stdout.
//
// Usage:
//
//	go run ./cmd/openapi-gen > docs/openapi.yaml
//
// The CI workflow runs this command and fails if the committed docs/openapi.yaml
// differs from the generated output.  Update this file whenever you add or
// change an API endpoint, then regenerate and commit docs/openapi.yaml.
package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

func main() {
	spec := buildSpec()
	data, err := yaml.Marshal(spec)
	if err != nil {
		fmt.Fprintf(os.Stderr, "openapi-gen: %v\n", err)
		os.Exit(1)
	}
	fmt.Print(string(data))
}

// ── helpers ──────────────────────────────────────────────────────────────────

type M = map[string]interface{}
type S = []interface{}

func ref(name string) M { return M{"$ref": "#/components/schemas/" + name} }

func strProp(desc string) M    { return M{"type": "string", "description": desc} }
func boolProp(desc string) M   { return M{"type": "boolean", "description": desc} }
func intProp(desc string) M    { return M{"type": "integer", "description": desc} }
func objProp(desc string) M    { return M{"type": "object", "description": desc} }
func arrProp(items M) M        { return M{"type": "array", "items": items} }
func arrOf(name string) M      { return arrProp(ref(name)) }

func pathParam(name, desc string) M {
	return M{
		"name":        name,
		"in":          "path",
		"required":    true,
		"description": desc,
		"schema":      M{"type": "string"},
	}
}

func queryParam(name, desc string, required bool) M {
	return M{
		"name":        name,
		"in":          "query",
		"required":    required,
		"description": desc,
		"schema":      M{"type": "string"},
	}
}

func jsonBody(schemaOrRef M) M {
	return M{"content": M{"application/json": M{"schema": schemaOrRef}}, "required": true}
}

func jsonResp(desc string, schemaOrRef M) M {
	return M{desc: M{"description": desc, "content": M{"application/json": M{"schema": schemaOrRef}}}}
}

func ok(schemaOrRef M) M       { return jsonResp("200", schemaOrRef) }
func created(schemaOrRef M) M  { return jsonResp("201", schemaOrRef) }
func noContent() M             { return M{"204": M{"description": "No content"}} }
func notFound() M              { return M{"404": M{"description": "Not found"}} }
func badRequest() M            { return M{"400": M{"description": "Bad request"}} }
func serverError() M           { return M{"500": M{"description": "Internal server error"}} }

func responses(maps ...M) M {
	out := M{}
	for _, m := range maps {
		for k, v := range m {
			out[k] = v
		}
	}
	return out
}

func op(summary, desc string, tags []string, params []interface{}, reqBody, resps M) M {
	o := M{
		"summary":     summary,
		"description": desc,
		"tags":        tags,
		"responses":   resps,
	}
	if len(params) > 0 {
		o["parameters"] = params
	}
	if reqBody != nil {
		o["requestBody"] = reqBody
	}
	return o
}

// ── spec ─────────────────────────────────────────────────────────────────────

func buildSpec() M {
	return M{
		"openapi": "3.0.3",
		"info": M{
			"title":       "Mimir AIP REST API",
			"description": "REST API for the Mimir AIP orchestrator. Covers projects, pipelines, schedules, plugins, storage, ontologies, ML models, digital twins, work tasks, and platform metrics.",
			"version":     "0.1.0",
			"contact": M{
				"name": "Mimir AIP",
				"url":  "https://github.com/Mimir-AIP/Mimir-AIP-Go",
			},
			"license": M{
				"name": "See repository",
				"url":  "https://github.com/Mimir-AIP/Mimir-AIP-Go/blob/main/LICENSE",
			},
		},
		"servers": S{
			M{"url": "http://localhost:8080", "description": "Local development"},
		},
		"tags": S{
			M{"name": "System", "description": "Health, readiness, and platform metrics"},
			M{"name": "Projects", "description": "Top-level organisational units"},
			M{"name": "Pipelines", "description": "Ordered processing step sequences"},
			M{"name": "Schedules", "description": "Cron-based pipeline triggers"},
			M{"name": "Plugins", "description": "Pipeline step executor plugins (Git-based)"},
			M{"name": "Storage", "description": "Storage backend configuration and CIR data operations"},
			M{"name": "Ontologies", "description": "OWL/Turtle vocabulary management"},
			M{"name": "Extraction", "description": "Entity extraction and ontology generation"},
			M{"name": "ML Models", "description": "ML model lifecycle (train, infer, monitor)"},
			M{"name": "Digital Twins", "description": "Live entity graphs with SPARQL querying"},
			M{"name": "Tasks", "description": "Work task queue (internal / worker-facing)"},
		},
		"paths":      buildPaths(),
		"components": M{"schemas": buildSchemas()},
	}
}

// ── paths ─────────────────────────────────────────────────────────────────────

func buildPaths() M {
	return M{
		// ── System ──
		"/health": M{
			"get": op("Health check", "Returns 200 when the orchestrator process is running.", []string{"System"}, nil, nil,
				responses(ok(M{"type": "object", "properties": M{"status": strProp("Always 'healthy'")}}))),
		},
		"/ready": M{
			"get": op("Readiness check", "Returns 200 when the queue is accessible and the orchestrator is ready to serve requests.", []string{"System"}, nil, nil,
				responses(ok(M{"type": "object", "properties": M{"status": strProp("'ready' or 'not ready'")}}), serverError())),
		},
		"/api/metrics": M{
			"get": op("Platform metrics", "Returns task counts by status and type, plus current queue depth.", []string{"System"}, nil, nil,
				responses(ok(ref("MetricsResponse")))),
		},

		// ── Projects ──
		"/api/projects": M{
			"get": op("List projects", "Returns all projects, optionally filtered by status.", []string{"Projects"},
				[]interface{}{queryParam("status", "Filter by project status", false)}, nil,
				responses(ok(arrOf("Project")))),
			"post": op("Create project", "Creates a new project.", []string{"Projects"}, nil,
				jsonBody(ref("ProjectCreateRequest")),
				responses(created(ref("Project")), badRequest())),
		},
		"/api/projects/clone": M{
			"post": op("Clone project", "Deep-clones an existing project including its pipelines, ontologies, ML models, digital twins, and storage configurations.", []string{"Projects"}, nil,
				jsonBody(ref("ProjectCloneRequest")),
				responses(created(ref("Project")), badRequest(), notFound())),
		},
		"/api/projects/{id}": M{
			"get": op("Get project", "Returns a single project by ID.", []string{"Projects"},
				[]interface{}{pathParam("id", "Project ID")}, nil,
				responses(ok(ref("Project")), notFound())),
			"put": op("Update project", "Updates a project's name, description, or status.", []string{"Projects"},
				[]interface{}{pathParam("id", "Project ID")},
				jsonBody(ref("ProjectUpdateRequest")),
				responses(ok(ref("Project")), badRequest(), notFound())),
			"delete": op("Delete project", "Deletes a project and all its components.", []string{"Projects"},
				[]interface{}{pathParam("id", "Project ID")}, nil,
				responses(noContent(), notFound())),
		},
		"/api/projects/{id}/{componentType}/{componentId}": M{
			"post": op("Add component to project", "Associates a pipeline, ontology, ML model, digital twin, or storage config with a project.", []string{"Projects"},
				[]interface{}{
					pathParam("id", "Project ID"),
					pathParam("componentType", "One of: pipelines, ontologies, mlmodels, digitaltwins, storage"),
					pathParam("componentId", "ID of the component to associate"),
				}, nil,
				responses(noContent(), badRequest(), notFound())),
			"delete": op("Remove component from project", "Removes the association between a component and a project.", []string{"Projects"},
				[]interface{}{
					pathParam("id", "Project ID"),
					pathParam("componentType", "One of: pipelines, ontologies, mlmodels, digitaltwins, storage"),
					pathParam("componentId", "ID of the component to disassociate"),
				}, nil,
				responses(noContent(), badRequest(), notFound())),
		},

		// ── Pipelines ──
		"/api/pipelines": M{
			"get": op("List pipelines", "Returns all pipelines, optionally filtered by project.", []string{"Pipelines"},
				[]interface{}{queryParam("project_id", "Filter by project ID", false)}, nil,
				responses(ok(arrOf("Pipeline")))),
			"post": op("Create pipeline", "Creates a new pipeline.", []string{"Pipelines"}, nil,
				jsonBody(ref("PipelineCreateRequest")),
				responses(created(ref("Pipeline")), badRequest())),
		},
		"/api/pipelines/{id}": M{
			"get": op("Get pipeline", "Returns a single pipeline by ID.", []string{"Pipelines"},
				[]interface{}{pathParam("id", "Pipeline ID")}, nil,
				responses(ok(ref("Pipeline")), notFound())),
			"put": op("Update pipeline", "Updates a pipeline's description, steps, or status.", []string{"Pipelines"},
				[]interface{}{pathParam("id", "Pipeline ID")},
				jsonBody(ref("PipelineUpdateRequest")),
				responses(ok(ref("Pipeline")), badRequest(), notFound())),
			"delete": op("Delete pipeline", "Deletes a pipeline.", []string{"Pipelines"},
				[]interface{}{pathParam("id", "Pipeline ID")}, nil,
				responses(noContent(), notFound())),
		},
		"/api/pipelines/execute": M{
			"post": op("Execute pipeline", "Enqueues a pipeline for asynchronous execution by a worker.", []string{"Pipelines"}, nil,
				jsonBody(ref("PipelineExecutionRequest")),
				responses(created(ref("WorkTask")), badRequest())),
		},

		// ── Schedules ──
		"/api/schedules": M{
			"get": op("List schedules", "Returns all cron schedules.", []string{"Schedules"}, nil, nil,
				responses(ok(arrOf("Schedule")))),
			"post": op("Create schedule", "Creates a new cron schedule.", []string{"Schedules"}, nil,
				jsonBody(ref("ScheduleCreateRequest")),
				responses(created(ref("Schedule")), badRequest())),
		},
		"/api/schedules/{id}": M{
			"get": op("Get schedule", "Returns a single schedule by ID.", []string{"Schedules"},
				[]interface{}{pathParam("id", "Schedule ID")}, nil,
				responses(ok(ref("Schedule")), notFound())),
			"put": op("Update schedule", "Updates a schedule.", []string{"Schedules"},
				[]interface{}{pathParam("id", "Schedule ID")},
				jsonBody(ref("ScheduleUpdateRequest")),
				responses(ok(ref("Schedule")), badRequest(), notFound())),
			"delete": op("Delete schedule", "Deletes a schedule.", []string{"Schedules"},
				[]interface{}{pathParam("id", "Schedule ID")}, nil,
				responses(noContent(), notFound())),
		},

		// ── Plugins ──
		"/api/plugins": M{
			"get": op("List plugins", "Returns all installed pipeline plugins.", []string{"Plugins"}, nil, nil,
				responses(ok(arrOf("Plugin")))),
			"post": op("Install plugin", "Installs a pipeline plugin from a Git repository. The worker clones and compiles the plugin at runtime.", []string{"Plugins"}, nil,
				jsonBody(ref("PluginInstallRequest")),
				responses(created(ref("Plugin")), badRequest())),
		},
		"/api/plugins/{name}": M{
			"get": op("Get plugin", "Returns metadata for a single installed plugin.", []string{"Plugins"},
				[]interface{}{pathParam("name", "Plugin name")}, nil,
				responses(ok(ref("Plugin")), notFound())),
			"put": op("Update plugin", "Pulls the latest version of a plugin from its repository.", []string{"Plugins"},
				[]interface{}{pathParam("name", "Plugin name")},
				jsonBody(ref("PluginUpdateRequest")),
				responses(ok(ref("Plugin")), notFound())),
			"delete": op("Uninstall plugin", "Removes a plugin.", []string{"Plugins"},
				[]interface{}{pathParam("name", "Plugin name")}, nil,
				responses(noContent(), notFound())),
		},

		// ── Storage ──
		"/api/storage/configs": M{
			"get": op("List storage configs", "Returns all storage configurations.", []string{"Storage"}, nil, nil,
				responses(ok(arrOf("StorageConfig")))),
			"post": op("Create storage config", "Registers a new storage backend configuration.", []string{"Storage"}, nil,
				jsonBody(ref("StorageConfigCreateRequest")),
				responses(created(ref("StorageConfig")), badRequest())),
		},
		"/api/storage/configs/{id}": M{
			"get": op("Get storage config", "Returns a single storage configuration.", []string{"Storage"},
				[]interface{}{pathParam("id", "Storage config ID")}, nil,
				responses(ok(ref("StorageConfig")), notFound())),
			"put": op("Update storage config", "Updates a storage configuration.", []string{"Storage"},
				[]interface{}{pathParam("id", "Storage config ID")},
				jsonBody(ref("StorageConfigUpdateRequest")),
				responses(ok(ref("StorageConfig")), notFound())),
			"delete": op("Delete storage config", "Removes a storage configuration.", []string{"Storage"},
				[]interface{}{pathParam("id", "Storage config ID")}, nil,
				responses(noContent(), notFound())),
		},
		"/api/storage/store": M{
			"post": op("Store CIR data", "Writes a CIR record to the specified storage backend.", []string{"Storage"}, nil,
				jsonBody(ref("StorageStoreRequest")),
				responses(ok(ref("StorageResult")), badRequest(), serverError())),
		},
		"/api/storage/retrieve": M{
			"post": op("Retrieve CIR data", "Queries CIR records from a storage backend.", []string{"Storage"}, nil,
				jsonBody(ref("StorageQueryRequest")),
				responses(ok(arrOf("CIR")), badRequest(), serverError())),
		},
		"/api/storage/update": M{
			"post": op("Update CIR data", "Updates CIR records matching a query.", []string{"Storage"}, nil,
				jsonBody(ref("StorageUpdateRequest")),
				responses(ok(ref("StorageResult")), badRequest(), serverError())),
		},
		"/api/storage/delete": M{
			"post": op("Delete CIR data", "Deletes CIR records matching a query.", []string{"Storage"}, nil,
				jsonBody(ref("StorageDeleteRequest")),
				responses(ok(ref("StorageResult")), badRequest(), serverError())),
		},
		"/api/storage/health": M{
			"post": op("Storage health check", "Tests the connection to a storage backend.", []string{"Storage"}, nil,
				jsonBody(M{"type": "object", "required": S{"storage_id"}, "properties": M{
					"storage_id": strProp("ID of the storage config to check"),
				}}),
				responses(ok(M{"type": "object", "properties": M{"healthy": boolProp("True if the backend is reachable"), "error": strProp("Error message if unhealthy")}}))),
		},

		// ── Ontologies ──
		"/api/ontologies": M{
			"get": op("List ontologies", "Returns all ontologies, optionally filtered by project.", []string{"Ontologies"},
				[]interface{}{queryParam("project_id", "Filter by project ID", false)}, nil,
				responses(ok(arrOf("Ontology")))),
			"post": op("Create ontology", "Creates a new ontology.", []string{"Ontologies"}, nil,
				jsonBody(ref("OntologyCreateRequest")),
				responses(created(ref("Ontology")), badRequest())),
		},
		"/api/ontologies/{id}": M{
			"get": op("Get ontology", "Returns a single ontology by ID.", []string{"Ontologies"},
				[]interface{}{pathParam("id", "Ontology ID")}, nil,
				responses(ok(ref("Ontology")), notFound())),
			"put": op("Update ontology", "Updates an ontology.", []string{"Ontologies"},
				[]interface{}{pathParam("id", "Ontology ID")},
				jsonBody(ref("OntologyUpdateRequest")),
				responses(ok(ref("Ontology")), notFound())),
			"delete": op("Delete ontology", "Deletes an ontology.", []string{"Ontologies"},
				[]interface{}{pathParam("id", "Ontology ID")}, nil,
				responses(noContent(), notFound())),
		},

		// ── Extraction ──
		"/api/extraction/generate-ontology": M{
			"post": op("Extract entities and generate ontology",
				"Reads data from a storage backend, runs the schema-inductive extraction algorithm, and returns a generated OWL/Turtle ontology along with any diff against the existing ontology.",
				[]string{"Extraction"}, nil,
				jsonBody(ref("ExtractionRequest")),
				responses(ok(ref("ExtractionResult")), badRequest(), serverError())),
		},

		// ── ML Models ──
		"/api/ml-models": M{
			"get": op("List ML models", "Returns all ML models, optionally filtered by project.", []string{"ML Models"},
				[]interface{}{queryParam("project_id", "Filter by project ID", false)}, nil,
				responses(ok(arrOf("MLModel")))),
			"post": op("Create ML model", "Registers a new ML model definition.", []string{"ML Models"}, nil,
				jsonBody(ref("MLModelCreateRequest")),
				responses(created(ref("MLModel")), badRequest())),
		},
		"/api/ml-models/recommend": M{
			"post": op("Recommend ML model type", "Analyses an ontology and storage data to recommend the most suitable model type.", []string{"ML Models"}, nil,
				jsonBody(ref("MLModelRecommendRequest")),
				responses(ok(ref("MLModelRecommendation")), badRequest())),
		},
		"/api/ml-models/train": M{
			"post": op("Trigger ML training", "Enqueues a training work task for the specified model.", []string{"ML Models"}, nil,
				jsonBody(ref("MLTrainingRequest")),
				responses(created(ref("WorkTask")), badRequest())),
		},
		"/api/ml-models/{id}": M{
			"get": op("Get ML model", "Returns a single ML model by ID.", []string{"ML Models"},
				[]interface{}{pathParam("id", "ML model ID")}, nil,
				responses(ok(ref("MLModel")), notFound())),
			"put": op("Update ML model", "Updates an ML model's metadata or configuration.", []string{"ML Models"},
				[]interface{}{pathParam("id", "ML model ID")},
				jsonBody(ref("MLModelUpdateRequest")),
				responses(ok(ref("MLModel")), notFound())),
			"delete": op("Delete ML model", "Deletes an ML model.", []string{"ML Models"},
				[]interface{}{pathParam("id", "ML model ID")}, nil,
				responses(noContent(), notFound())),
		},
		"/api/ml-models/{id}/training/complete": M{
			"post": op("Complete training (worker)", "Called by workers to report successful training completion and upload the model artifact path.", []string{"ML Models", "Tasks"},
				[]interface{}{pathParam("id", "ML model ID")},
				jsonBody(M{"type": "object", "required": S{"model_artifact_path", "performance_metrics"}, "properties": M{
					"model_artifact_path": strProp("Path to the trained model artifact"),
					"performance_metrics": objProp("Performance metrics from training"),
				}}),
				responses(ok(M{"type": "object"}), badRequest(), notFound())),
		},
		"/api/ml-models/{id}/training/fail": M{
			"post": op("Fail training (worker)", "Called by workers to report a training failure.", []string{"ML Models", "Tasks"},
				[]interface{}{pathParam("id", "ML model ID")},
				jsonBody(M{"type": "object", "properties": M{
					"error_message": strProp("Description of the failure"),
				}}),
				responses(ok(M{"type": "object"}), notFound())),
		},

		// ── Digital Twins ──
		"/api/digital-twins": M{
			"get": op("List digital twins", "Returns all digital twins, optionally filtered by project.", []string{"Digital Twins"},
				[]interface{}{queryParam("project_id", "Filter by project ID", false)}, nil,
				responses(ok(arrOf("DigitalTwin")))),
			"post": op("Create digital twin", "Creates a new digital twin.", []string{"Digital Twins"}, nil,
				jsonBody(ref("DigitalTwinCreateRequest")),
				responses(created(ref("DigitalTwin")), badRequest())),
		},
		"/api/digital-twins/{id}": M{
			"get": op("Get digital twin", "Returns a single digital twin by ID.", []string{"Digital Twins"},
				[]interface{}{pathParam("id", "Digital twin ID")}, nil,
				responses(ok(ref("DigitalTwin")), notFound())),
			"put": op("Update digital twin", "Updates a digital twin's metadata.", []string{"Digital Twins"},
				[]interface{}{pathParam("id", "Digital twin ID")},
				jsonBody(ref("DigitalTwinUpdateRequest")),
				responses(ok(ref("DigitalTwin")), notFound())),
			"delete": op("Delete digital twin", "Deletes a digital twin and all its cached entities.", []string{"Digital Twins"},
				[]interface{}{pathParam("id", "Digital twin ID")}, nil,
				responses(noContent(), notFound())),
		},
		"/api/digital-twins/{id}/sync": M{
			"post": op("Sync digital twin from storage", "Enqueues a work task to synchronise the digital twin's entity graph from its linked storage backends.", []string{"Digital Twins"},
				[]interface{}{pathParam("id", "Digital twin ID")}, nil,
				responses(created(ref("WorkTask")), notFound())),
		},
		"/api/digital-twins/{id}/sparql": M{
			"post": op("SPARQL query", "Executes a SPARQL SELECT query against the digital twin's in-memory entity graph.", []string{"Digital Twins"},
				[]interface{}{pathParam("id", "Digital twin ID")},
				jsonBody(M{"type": "object", "required": S{"query"}, "properties": M{
					"query": strProp("SPARQL SELECT query string"),
				}}),
				responses(ok(M{"type": "object", "properties": M{"results": arrProp(objProp("Row of bound variables"))}}), badRequest(), notFound())),
		},
		"/api/digital-twins/{id}/entities": M{
			"get": op("List entities", "Returns all entities in the digital twin's graph.", []string{"Digital Twins"},
				[]interface{}{pathParam("id", "Digital twin ID")}, nil,
				responses(ok(arrOf("Entity")), notFound())),
		},
		"/api/digital-twins/{id}/entities/{entityId}": M{
			"get": op("Get entity", "Returns a single entity by ID.", []string{"Digital Twins"},
				[]interface{}{pathParam("id", "Digital twin ID"), pathParam("entityId", "Entity ID")}, nil,
				responses(ok(ref("Entity")), notFound())),
		},
		"/api/digital-twins/{id}/entities/{entityId}/related": M{
			"get": op("Get related entities", "Returns entities related to the specified entity, optionally filtered by relationship type.", []string{"Digital Twins"},
				[]interface{}{
					pathParam("id", "Digital twin ID"),
					pathParam("entityId", "Entity ID"),
					queryParam("relationship", "Filter by relationship type", false),
				}, nil,
				responses(ok(arrOf("Entity")), notFound())),
		},
		"/api/digital-twins/{id}/scenarios": M{
			"post": op("Apply scenario", "Applies a set of what-if modifications in memory and returns predicted outcomes without mutating state.", []string{"Digital Twins"},
				[]interface{}{pathParam("id", "Digital twin ID")},
				jsonBody(ref("ScenarioRequest")),
				responses(ok(ref("ScenarioResult")), badRequest(), notFound())),
		},
		"/api/digital-twins/{id}/scenarios/{scenarioId}": M{
			"get": op("Get scenario result", "Returns the result of a previously applied scenario.", []string{"Digital Twins"},
				[]interface{}{pathParam("id", "Digital twin ID"), pathParam("scenarioId", "Scenario ID")}, nil,
				responses(ok(ref("ScenarioResult")), notFound())),
		},
		"/api/digital-twins/{id}/actions": M{
			"post": op("Apply action", "Triggers an action on a digital twin.", []string{"Digital Twins"},
				[]interface{}{pathParam("id", "Digital twin ID")},
				jsonBody(ref("ActionRequest")),
				responses(ok(ref("ActionResult")), badRequest(), notFound())),
		},
		"/api/digital-twins/{id}/actions/{actionId}": M{
			"get": op("Get action result", "Returns the result of a previously applied action.", []string{"Digital Twins"},
				[]interface{}{pathParam("id", "Digital twin ID"), pathParam("actionId", "Action ID")}, nil,
				responses(ok(ref("ActionResult")), notFound())),
		},

		// ── Tasks (worker-facing, requires Authorization: Bearer <token>) ──
		"/api/worktasks": M{
			"get": op("Get queue info", "Returns the current work task queue depth.", []string{"Tasks"}, nil, nil,
				responses(ok(M{"type": "object", "properties": M{"queue_length": intProp("Number of queued tasks")}}))),
			"post": op("Submit work task", "Submits a new work task to the queue. Requires the worker auth token.", []string{"Tasks"}, nil,
				jsonBody(ref("WorkTaskSubmissionRequest")),
				responses(created(ref("WorkTask")), badRequest())),
		},
		"/api/worktasks/{id}": M{
			"get": op("Get work task", "Returns a work task by ID. Requires the worker auth token.", []string{"Tasks"},
				[]interface{}{pathParam("id", "Work task ID")}, nil,
				responses(ok(ref("WorkTask")), notFound())),
			"post": op("Update work task status", "Called by workers to report task completion or failure. Requires the worker auth token.", []string{"Tasks"},
				[]interface{}{pathParam("id", "Work task ID")},
				jsonBody(ref("WorkTaskResult")),
				responses(ok(M{"type": "object"}), badRequest(), notFound())),
		},
	}
}

// ── schemas ───────────────────────────────────────────────────────────────────

func buildSchemas() M {
	return M{
		// ── Shared ──
		"Error": M{
			"type": "object",
			"properties": M{
				"error": strProp("Human-readable error message"),
			},
		},

		// ── Metrics ──
		"MetricsResponse": M{
			"type": "object",
			"properties": M{
				"queue":           M{"type": "object", "properties": M{"length": intProp("Current queue depth")}},
				"tasks_by_status": M{"type": "object", "additionalProperties": M{"type": "integer"}, "description": "Task count keyed by status"},
				"tasks_by_type":   M{"type": "object", "additionalProperties": M{"type": "integer"}, "description": "Task count keyed by type"},
				"timestamp":       strProp("ISO-8601 UTC timestamp of the snapshot"),
			},
		},

		// ── Projects ──
		"Project": M{
			"type": "object",
			"properties": M{
				"id":          strProp("Project ID (UUID)"),
				"name":        strProp("Project name"),
				"description": strProp("Project description"),
				"status":      strProp("Project status (active, archived)"),
				"components": M{"type": "object", "description": "Associated component IDs",
					"properties": M{
						"pipelines":     arrProp(M{"type": "string"}),
						"ontologies":    arrProp(M{"type": "string"}),
						"ml_models":     arrProp(M{"type": "string"}),
						"digital_twins": arrProp(M{"type": "string"}),
						"storage":       arrProp(M{"type": "string"}),
					},
				},
				"created_at": strProp("ISO-8601 creation timestamp"),
				"updated_at": strProp("ISO-8601 last-updated timestamp"),
			},
		},
		"ProjectCreateRequest": M{
			"type":     "object",
			"required": S{"name"},
			"properties": M{
				"name":        strProp("Project name"),
				"description": strProp("Project description"),
			},
		},
		"ProjectUpdateRequest": M{
			"type": "object",
			"properties": M{
				"name":        strProp("New name"),
				"description": strProp("New description"),
				"status":      strProp("New status"),
			},
		},
		"ProjectCloneRequest": M{
			"type":     "object",
			"required": S{"project_id", "name"},
			"properties": M{
				"project_id": strProp("ID of the project to clone"),
				"name":       strProp("Name for the cloned project"),
			},
		},

		// ── Pipelines ──
		"Pipeline": M{
			"type": "object",
			"properties": M{
				"id":          strProp("Pipeline ID (UUID)"),
				"project_id":  strProp("Owning project ID"),
				"name":        strProp("Pipeline name"),
				"type":        strProp("Pipeline type (ingestion, processing, output)"),
				"description": strProp("Pipeline description"),
				"steps":       arrOf("PipelineStep"),
				"status":      strProp("Pipeline status"),
				"created_at":  strProp("ISO-8601 creation timestamp"),
				"updated_at":  strProp("ISO-8601 last-updated timestamp"),
			},
		},
		"PipelineStep": M{
			"type":     "object",
			"required": S{"name", "plugin", "action"},
			"properties": M{
				"name":       strProp("Step name (unique within pipeline)"),
				"plugin":     strProp("Plugin name or 'default' for built-in actions"),
				"action":     strProp("Action name within the plugin"),
				"parameters": M{"type": "object", "additionalProperties": true, "description": "Action parameters"},
				"output":     M{"type": "object", "additionalProperties": M{"type": "string"}, "description": "Output mapping"},
			},
		},
		"PipelineCreateRequest": M{
			"type":     "object",
			"required": S{"project_id", "name", "type", "steps"},
			"properties": M{
				"project_id":  strProp("Owning project ID"),
				"name":        strProp("Pipeline name"),
				"type":        strProp("Pipeline type"),
				"description": strProp("Pipeline description"),
				"steps":       arrOf("PipelineStep"),
			},
		},
		"PipelineUpdateRequest": M{
			"type": "object",
			"properties": M{
				"description": strProp("New description"),
				"steps":       arrOf("PipelineStep"),
				"status":      strProp("New status"),
			},
		},
		"PipelineExecutionRequest": M{
			"type":     "object",
			"required": S{"pipeline_id"},
			"properties": M{
				"pipeline_id":  strProp("ID of the pipeline to execute"),
				"trigger_type": strProp("manual, scheduled, or automatic"),
				"triggered_by": strProp("Identifier of the trigger source"),
				"parameters":   M{"type": "object", "additionalProperties": true, "description": "Runtime parameters"},
			},
		},

		// ── Schedules ──
		"Schedule": M{
			"type": "object",
			"properties": M{
				"id":           strProp("Schedule ID (UUID)"),
				"name":         strProp("Schedule name"),
				"cron":         strProp("Cron expression (e.g. '0 * * * *')"),
				"pipeline_ids": arrProp(M{"type": "string"}),
				"enabled":      boolProp("Whether the schedule is active"),
				"created_at":   strProp("ISO-8601 creation timestamp"),
				"updated_at":   strProp("ISO-8601 last-updated timestamp"),
			},
		},
		"ScheduleCreateRequest": M{
			"type":     "object",
			"required": S{"name", "cron", "pipeline_ids"},
			"properties": M{
				"name":         strProp("Schedule name"),
				"cron":         strProp("Cron expression"),
				"pipeline_ids": arrProp(M{"type": "string"}),
				"enabled":      boolProp("Start enabled (default true)"),
			},
		},
		"ScheduleUpdateRequest": M{
			"type": "object",
			"properties": M{
				"name":         strProp("New name"),
				"cron":         strProp("New cron expression"),
				"pipeline_ids": arrProp(M{"type": "string"}),
				"enabled":      boolProp("Enable or disable the schedule"),
			},
		},

		// ── Plugins ──
		"Plugin": M{
			"type": "object",
			"properties": M{
				"name":           strProp("Plugin name"),
				"version":        strProp("Plugin version"),
				"description":    strProp("Plugin description"),
				"author":         strProp("Plugin author"),
				"repository_url": strProp("Git repository URL"),
				"actions":        arrProp(M{"type": "string"}),
				"installed_at":   strProp("ISO-8601 installation timestamp"),
			},
		},
		"PluginInstallRequest": M{
			"type":     "object",
			"required": S{"repository_url"},
			"properties": M{
				"repository_url": strProp("HTTPS Git URL of the plugin repository"),
				"git_ref":        strProp("Branch, tag, or commit SHA to install (default: main)"),
			},
		},
		"PluginUpdateRequest": M{
			"type": "object",
			"properties": M{
				"git_ref": strProp("Branch, tag, or commit SHA to update to"),
			},
		},

		// ── Storage ──
		"StorageConfig": M{
			"type": "object",
			"properties": M{
				"id":          strProp("Storage config ID (UUID)"),
				"project_id":  strProp("Owning project ID"),
				"plugin_type": strProp("Backend type: filesystem, postgresql, mysql, mongodb, s3, redis, elasticsearch, neo4j"),
				"config":      M{"type": "object", "additionalProperties": true, "description": "Backend-specific configuration"},
				"ontology_id": strProp("Optional linked ontology ID"),
				"active":      boolProp("Whether this config is active"),
				"created_at":  strProp("ISO-8601 creation timestamp"),
				"updated_at":  strProp("ISO-8601 last-updated timestamp"),
			},
		},
		"StorageConfigCreateRequest": M{
			"type":     "object",
			"required": S{"project_id", "plugin_type", "config"},
			"properties": M{
				"project_id":  strProp("Owning project ID"),
				"plugin_type": strProp("Backend type"),
				"config":      M{"type": "object", "additionalProperties": true},
				"ontology_id": strProp("Optional linked ontology ID"),
			},
		},
		"StorageConfigUpdateRequest": M{
			"type": "object",
			"properties": M{
				"config":      M{"type": "object", "additionalProperties": true},
				"ontology_id": strProp("Linked ontology ID"),
				"active":      boolProp("Active flag"),
			},
		},
		"CIR": M{
			"description": "Common Internal Representation — the normalised record format used across all storage backends.",
			"type":        "object",
			"properties": M{
				"id":       strProp("CIR record ID"),
				"source":   M{"type": "object", "description": "Provenance information", "additionalProperties": true},
				"data":     M{"type": "object", "description": "Record payload", "additionalProperties": true},
				"metadata": M{"type": "object", "description": "Record metadata", "additionalProperties": true},
			},
		},
		"CIRQuery": M{
			"type": "object",
			"properties": M{
				"entity_type": strProp("Filter by entity type"),
				"filters":     arrOf("CIRCondition"),
				"order_by":    arrOf("OrderByClause"),
				"limit":       intProp("Maximum number of results"),
				"offset":      intProp("Results offset for pagination"),
			},
		},
		"CIRCondition": M{
			"type":     "object",
			"required": S{"attribute", "operator", "value"},
			"properties": M{
				"attribute": strProp("Attribute name to filter on"),
				"operator":  strProp("Comparison operator: eq, neq, gt, gte, lt, lte, in, like"),
				"value":     M{"description": "Filter value"},
			},
		},
		"OrderByClause": M{
			"type":     "object",
			"required": S{"attribute"},
			"properties": M{
				"attribute": strProp("Attribute to sort by"),
				"direction": strProp("asc or desc (default asc)"),
			},
		},
		"StorageResult": M{
			"type": "object",
			"properties": M{
				"success":        boolProp("Whether the operation succeeded"),
				"affected_items": intProp("Number of records affected"),
				"error":          strProp("Error message if unsuccessful"),
			},
		},
		"StorageStoreRequest": M{
			"type":     "object",
			"required": S{"project_id", "storage_id", "cir_data"},
			"properties": M{
				"project_id": strProp("Project ID"),
				"storage_id": strProp("Storage config ID"),
				"cir_data":   ref("CIR"),
			},
		},
		"StorageQueryRequest": M{
			"type":     "object",
			"required": S{"project_id", "storage_id"},
			"properties": M{
				"project_id": strProp("Project ID"),
				"storage_id": strProp("Storage config ID"),
				"query":      ref("CIRQuery"),
			},
		},
		"StorageUpdateRequest": M{
			"type":     "object",
			"required": S{"project_id", "storage_id", "query", "updates"},
			"properties": M{
				"project_id": strProp("Project ID"),
				"storage_id": strProp("Storage config ID"),
				"query":      ref("CIRQuery"),
				"updates": M{"type": "object", "properties": M{
					"filters": arrOf("CIRCondition"),
					"updates": M{"type": "object", "additionalProperties": true},
				}},
			},
		},
		"StorageDeleteRequest": M{
			"type":     "object",
			"required": S{"project_id", "storage_id", "query"},
			"properties": M{
				"project_id": strProp("Project ID"),
				"storage_id": strProp("Storage config ID"),
				"query":      ref("CIRQuery"),
			},
		},

		// ── Ontologies ──
		"Ontology": M{
			"type": "object",
			"properties": M{
				"id":          strProp("Ontology ID (UUID)"),
				"project_id":  strProp("Owning project ID"),
				"name":        strProp("Ontology name"),
				"description": strProp("Ontology description"),
				"content":     strProp("OWL/Turtle ontology content"),
				"status":      strProp("Ontology status (draft, active, needs_review, deprecated)"),
				"created_at":  strProp("ISO-8601 creation timestamp"),
				"updated_at":  strProp("ISO-8601 last-updated timestamp"),
			},
		},
		"OntologyCreateRequest": M{
			"type":     "object",
			"required": S{"project_id", "name"},
			"properties": M{
				"project_id":  strProp("Owning project ID"),
				"name":        strProp("Ontology name"),
				"description": strProp("Ontology description"),
				"content":     strProp("Initial OWL/Turtle content"),
			},
		},
		"OntologyUpdateRequest": M{
			"type": "object",
			"properties": M{
				"name":        strProp("New name"),
				"description": strProp("New description"),
				"content":     strProp("Updated OWL/Turtle content"),
				"status":      strProp("New status"),
			},
		},

		// ── Extraction ──
		"ExtractionRequest": M{
			"type":     "object",
			"required": S{"project_id", "storage_id"},
			"properties": M{
				"project_id":  strProp("Project ID"),
				"storage_id":  strProp("Storage config ID to extract from"),
				"ontology_id": strProp("Optional existing ontology ID to diff against"),
			},
		},
		"ExtractionResult": M{
			"type": "object",
			"properties": M{
				"ontology":     ref("Ontology"),
				"diff":         objProp("OntologyDiff — added/removed/modified classes and properties"),
				"needs_review": boolProp("True if the diff was significant enough to flag for review"),
			},
		},

		// ── ML Models ──
		"MLModel": M{
			"type": "object",
			"properties": M{
				"id":                   strProp("ML model ID (UUID)"),
				"project_id":           strProp("Owning project ID"),
				"ontology_id":          strProp("Linked ontology ID"),
				"name":                 strProp("Model name"),
				"description":          strProp("Model description"),
				"type":                 strProp("Model type: decision_tree, random_forest, regression, neural_network"),
				"status":               strProp("Model status: draft, training, trained, failed, degraded, deprecated, archived"),
				"version":              strProp("Model version string"),
				"is_recommended":       boolProp("True if this model type was recommended by the recommendation engine"),
				"recommendation_score": intProp("Recommendation engine score"),
				"training_config":      objProp("Training configuration"),
				"training_metrics":     objProp("Metrics recorded during training"),
				"model_artifact_path":  strProp("Path to the serialised model artifact"),
				"performance_metrics":  objProp("Latest performance metrics"),
				"metadata":             M{"type": "object", "additionalProperties": true},
				"created_at":           strProp("ISO-8601 creation timestamp"),
				"updated_at":           strProp("ISO-8601 last-updated timestamp"),
			},
		},
		"MLModelCreateRequest": M{
			"type":     "object",
			"required": S{"project_id", "name", "type"},
			"properties": M{
				"project_id":      strProp("Owning project ID"),
				"ontology_id":     strProp("Linked ontology ID"),
				"name":            strProp("Model name"),
				"description":     strProp("Model description"),
				"type":            strProp("Model type"),
				"training_config": objProp("Training configuration"),
			},
		},
		"MLModelUpdateRequest": M{
			"type": "object",
			"properties": M{
				"name":            strProp("New name"),
				"description":     strProp("New description"),
				"training_config": objProp("Updated training configuration"),
				"status":          strProp("New status"),
			},
		},
		"MLModelRecommendRequest": M{
			"type":     "object",
			"required": S{"project_id"},
			"properties": M{
				"project_id":  strProp("Project ID"),
				"ontology_id": strProp("Ontology to base recommendation on"),
				"storage_id":  strProp("Storage config to sample data from"),
			},
		},
		"MLModelRecommendation": M{
			"type": "object",
			"properties": M{
				"recommended_type": strProp("Recommended model type"),
				"score":            intProp("Confidence score (0-100)"),
				"reason":           strProp("Explanation of the recommendation"),
				"alternatives":     arrProp(objProp("Alternative model type with score and reason")),
			},
		},
		"MLTrainingRequest": M{
			"type":     "object",
			"required": S{"model_id"},
			"properties": M{
				"model_id":   strProp("ID of the model to train"),
				"storage_id": strProp("Storage config to load training data from"),
			},
		},

		// ── Digital Twins ──
		"DigitalTwin": M{
			"type": "object",
			"properties": M{
				"id":          strProp("Digital twin ID (UUID)"),
				"project_id":  strProp("Owning project ID"),
				"name":        strProp("Digital twin name"),
				"description": strProp("Digital twin description"),
				"ontology_id": strProp("Linked ontology ID"),
				"status":      strProp("Twin status (initialising, active, syncing, error)"),
				"entity_count": intProp("Number of entities in the graph"),
				"metadata":    M{"type": "object", "additionalProperties": true},
				"created_at":  strProp("ISO-8601 creation timestamp"),
				"updated_at":  strProp("ISO-8601 last-updated timestamp"),
			},
		},
		"DigitalTwinCreateRequest": M{
			"type":     "object",
			"required": S{"project_id", "name"},
			"properties": M{
				"project_id":  strProp("Owning project ID"),
				"ontology_id": strProp("Linked ontology ID"),
				"name":        strProp("Digital twin name"),
				"description": strProp("Digital twin description"),
			},
		},
		"DigitalTwinUpdateRequest": M{
			"type": "object",
			"properties": M{
				"name":        strProp("New name"),
				"description": strProp("New description"),
				"ontology_id": strProp("New linked ontology ID"),
			},
		},
		"Entity": M{
			"type": "object",
			"properties": M{
				"id":               strProp("Entity ID (UUID)"),
				"type":             strProp("Entity type (from ontology)"),
				"attributes":       M{"type": "object", "additionalProperties": true, "description": "Entity attribute values"},
				"computed_values":  M{"type": "object", "additionalProperties": true, "description": "Values derived from ML inference or sync"},
				"relationships":    arrProp(objProp("Relationship to another entity")),
				"last_updated":     strProp("ISO-8601 last-updated timestamp"),
			},
		},
		"ScenarioRequest": M{
			"type":     "object",
			"required": S{"modifications"},
			"properties": M{
				"scenario_id":   strProp("Optional scenario ID (UUID generated if omitted)"),
				"modifications": arrProp(objProp("Entity attribute modification")),
			},
		},
		"ScenarioResult": M{
			"type": "object",
			"properties": M{
				"scenario_id": strProp("Scenario ID"),
				"entities":    arrOf("Entity"),
				"predictions": M{"type": "object", "additionalProperties": true, "description": "ML model predictions under the scenario"},
				"created_at":  strProp("ISO-8601 timestamp"),
			},
		},
		"ActionRequest": M{
			"type":     "object",
			"required": S{"action_type"},
			"properties": M{
				"action_type": strProp("Type of action to apply"),
				"parameters":  M{"type": "object", "additionalProperties": true},
			},
		},
		"ActionResult": M{
			"type": "object",
			"properties": M{
				"action_id":  strProp("Action ID"),
				"status":     strProp("Action status"),
				"result":     M{"type": "object", "additionalProperties": true},
				"applied_at": strProp("ISO-8601 timestamp"),
			},
		},

		// ── Work Tasks ──
		"WorkTask": M{
			"type": "object",
			"properties": M{
				"id":                    strProp("Work task ID (UUID)"),
				"type":                  strProp("Task type: pipeline_execution, ml_training, ml_inference, digital_twin_update"),
				"status":                strProp("Task status: queued, running, completed, failed"),
				"priority":              intProp("Task priority (higher = processed first)"),
				"project_id":            strProp("Owning project ID"),
				"submitted_at":          strProp("ISO-8601 submission timestamp"),
				"started_at":            strProp("ISO-8601 start timestamp"),
				"completed_at":          strProp("ISO-8601 completion timestamp"),
				"error_message":         strProp("Error message if task failed"),
				"retry_count":           intProp("Number of times this task has been retried"),
				"max_retries":           intProp("Maximum retry attempts"),
				"task_spec":             M{"type": "object", "additionalProperties": true, "description": "Task-type-specific parameters"},
				"resource_requirements": M{"type": "object", "description": "CPU/memory/GPU resource requests"},
			},
		},
		"WorkTaskSubmissionRequest": M{
			"type":     "object",
			"required": S{"type", "project_id"},
			"properties": M{
				"type":                  strProp("Task type"),
				"project_id":            strProp("Owning project ID"),
				"priority":              intProp("Task priority"),
				"task_spec":             M{"type": "object", "additionalProperties": true},
				"resource_requirements": M{"type": "object", "additionalProperties": true},
				"data_access":           M{"type": "object", "additionalProperties": true},
			},
		},
		"WorkTaskResult": M{
			"type":     "object",
			"required": S{"status"},
			"properties": M{
				"status":        strProp("New task status: completed or failed"),
				"error_message": strProp("Error message if status is failed"),
				"result":        M{"type": "object", "additionalProperties": true, "description": "Task output data"},
			},
		},
	}
}

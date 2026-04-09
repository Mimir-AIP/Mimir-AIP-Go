package api

import "github.com/mimir-aip/mimir-aip-go/pkg/api/doc"

func init() {
	// ── System ────────────────────────────────────────────────────────────────
	doc.Register("GET", "/health", doc.RouteDoc{
		Summary:     "Health check",
		Description: "Returns queue-backed orchestrator health. `status` is `healthy` when the queue is available with no failed tasks, and `degraded` when failed tasks are present.",
		Tags:        []string{"System"},
		Responses:   doc.R(doc.OK(doc.Ref("SystemHealthResponse")), doc.ServerError()),
	})
	doc.Register("GET", "/ready", doc.RouteDoc{
		Summary:     "Readiness check",
		Description: "Returns 200 only when the queue is configured and readable by the orchestrator process.",
		Tags:        []string{"System"},
		Responses:   doc.R(doc.OK(doc.Ref("ReadinessResponse")), doc.ServerError()),
	})
	doc.Register("GET", "/api/metrics", doc.RouteDoc{
		Summary:     "Platform metrics",
		Description: "Returns a queue snapshot with task counts by status and type.",
		Tags:        []string{"System"},
		Responses:   doc.R(doc.OK(doc.Ref("MetricsResponse"))),
	})

	doc.Register("GET", "/ws/tasks", doc.RouteDoc{
		Summary:     "Task update WebSocket",
		Description: "WebSocket stream that emits `task_update` events whenever a work task status changes. Frontend clients use this for queue and training progress updates.",
		Tags:        []string{"System"},
		Responses:   doc.R(doc.OK(doc.Str("WebSocket upgrade endpoint"))),
	})

	doc.Register("GET", "/openapi.yaml", doc.RouteDoc{
		Summary:     "OpenAPI specification",
		Description: "Returns the live OpenAPI 3.0 YAML specification, generated dynamically from all registered routes.",
		Tags:        []string{"System"},
		Responses:   doc.R(doc.OK(doc.Str("OpenAPI 3.0 YAML document"))),
	})
	doc.Register("POST", "/api/admin/settings/factory-reset", doc.RouteDoc{
		Summary:     "Factory reset metadata",
		Description: "Deletes all persisted Mimir metadata and clears the in-memory work queue after confirming there are no queued or active tasks. External data stored in connected storage backends is not deleted.",
		Tags:        []string{"System"},
		Responses: doc.R(
			doc.OK(doc.Ref("FactoryResetResponse")),
			map[string]doc.M{"409": {"description": "Conflict — reset blocked while queued or active tasks exist"}},
			doc.ServerError(),
		),
	})

	// ── Work Tasks (worker-facing, requires Authorization: Bearer) ────────────
	doc.Register("GET", "/api/worktasks", doc.RouteDoc{
		Summary:     "List work tasks",
		Description: "Returns all known work tasks plus the current queue depth. Requires the worker auth token.",
		Tags:        []string{"Tasks"},
		Responses:   doc.R(doc.OK(doc.Ref("WorkTaskListResponse"))),
	})
	doc.Register("POST", "/api/worktasks", doc.RouteDoc{
		Summary:     "Submit work task",
		Description: "Submits a new work task to the queue. Requires the worker auth token (Authorization: Bearer <token>).",
		Tags:        []string{"Tasks"},
		RequestBody: doc.JsonBody(doc.Ref("WorkTaskSubmissionRequest")),
		Responses:   doc.R(doc.Created(doc.Ref("WorkTask")), doc.BadRequest()),
	})
	doc.Register("GET", "/api/worktasks/{id}", doc.RouteDoc{
		Summary:     "Get work task",
		Description: "Returns a work task by ID. Requires the worker auth token.",
		Tags:        []string{"Tasks"},
		Params:      []doc.Param{doc.PParam("id", "Work task ID")},
		Responses:   doc.R(doc.OK(doc.Ref("WorkTask")), doc.NotFound()),
	})
	doc.Register("POST", "/api/worktasks/{id}", doc.RouteDoc{
		Summary:     "Update work task status",
		Description: "Called by workers to report task execution status, result metadata, output locations, or failure details. Requires the worker auth token.",
		Tags:        []string{"Tasks"},
		Params:      []doc.Param{doc.PParam("id", "Work task ID")},
		RequestBody: doc.JsonBody(doc.Ref("WorkTaskResult")),
		Responses:   doc.R(doc.OK(doc.M{"type": "object"}), doc.BadRequest(), doc.NotFound()),
	})
}

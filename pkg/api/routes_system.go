package api

import "github.com/mimir-aip/mimir-aip-go/pkg/api/doc"

func init() {
	// ── System ────────────────────────────────────────────────────────────────
	doc.Register("GET", "/health", doc.RouteDoc{
		Summary:     "Health check",
		Description: "Returns 200 when the orchestrator process is running.",
		Tags:        []string{"System"},
		Responses:   doc.R(doc.OK(doc.Props(nil, doc.M{"status": doc.Str("Always 'healthy'")}))),
	})
	doc.Register("GET", "/ready", doc.RouteDoc{
		Summary:     "Readiness check",
		Description: "Returns 200 when the queue is accessible and the orchestrator is ready to serve requests.",
		Tags:        []string{"System"},
		Responses:   doc.R(doc.OK(doc.Props(nil, doc.M{"status": doc.Str("'ready' or 'not ready'")})), doc.ServerError()),
	})
	doc.Register("GET", "/api/metrics", doc.RouteDoc{
		Summary:     "Platform metrics",
		Description: "Returns task counts by status and type, plus the current queue depth.",
		Tags:        []string{"System"},
		Responses:   doc.R(doc.OK(doc.Ref("MetricsResponse"))),
	})

	doc.Register("GET", "/openapi.yaml", doc.RouteDoc{
		Summary:     "OpenAPI specification",
		Description: "Returns the live OpenAPI 3.0 YAML specification, generated dynamically from all registered routes.",
		Tags:        []string{"System"},
		Responses:   doc.R(doc.OK(doc.Str("OpenAPI 3.0 YAML document"))),
	})

	// ── Work Tasks (worker-facing, requires Authorization: Bearer) ────────────
	doc.Register("GET", "/api/worktasks", doc.RouteDoc{
		Summary:     "Get queue info",
		Description: "Returns the current queue depth.",
		Tags:        []string{"Tasks"},
		Responses:   doc.R(doc.OK(doc.Props(nil, doc.M{"queue_length": doc.Int("Number of queued tasks")}))),
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
		Description: "Called by workers to report task completion or failure. Requires the worker auth token.",
		Tags:        []string{"Tasks"},
		Params:      []doc.Param{doc.PParam("id", "Work task ID")},
		RequestBody: doc.JsonBody(doc.Ref("WorkTaskResult")),
		Responses:   doc.R(doc.OK(doc.M{"type": "object"}), doc.BadRequest(), doc.NotFound()),
	})
}

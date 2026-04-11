package api

import "github.com/mimir-aip/mimir-aip-go/pkg/api/doc"

func init() {
	doc.Register("GET", "/api/pipelines", doc.RouteDoc{
		Summary:     "List pipelines",
		Description: "Returns all pipelines, optionally filtered by project.",
		Tags:        []string{"Pipelines"},
		Params:      []doc.Param{doc.QParam("project_id", "Filter by project ID", false)},
		Responses:   doc.R(doc.OK(doc.ArrOf("Pipeline"))),
	})
	doc.Register("POST", "/api/pipelines", doc.RouteDoc{
		Summary:     "Create pipeline",
		Description: "Creates a new pipeline.",
		Tags:        []string{"Pipelines"},
		RequestBody: doc.JsonBody(doc.Ref("PipelineCreateRequest")),
		Responses:   doc.R(doc.Created(doc.Ref("Pipeline")), doc.BadRequest()),
	})
	doc.Register("POST", "/api/pipelines/{id}/execute", doc.RouteDoc{
		Summary:     "Execute pipeline",
		Description: "Enqueues a pipeline for asynchronous execution by a worker.",
		Tags:        []string{"Pipelines"},
		Params:      []doc.Param{doc.PParam("id", "Pipeline ID")},
		RequestBody: doc.JsonBody(doc.Ref("PipelineExecutionRequest")),
		Responses:   doc.R(doc.Accepted(doc.Ref("WorkTask")), doc.BadRequest(), doc.NotFound()),
	})
	doc.Register("POST", "/api/pipelines/{id}/trigger", doc.RouteDoc{
		Summary:     "Trigger pipeline",
		Description: "Queues a pipeline through its manual/system trigger path. Uses the pipeline's trigger configuration and records trigger provenance in the resulting work task.",
		Tags:        []string{"Pipelines"},
		Params:      []doc.Param{doc.PParam("id", "Pipeline ID")},
		RequestBody: doc.JsonBody(doc.Ref("PipelineTriggerRequest")),
		Responses:   doc.R(doc.Accepted(doc.Ref("WorkTask")), doc.BadRequest(), doc.NotFound()),
	})
	doc.Register("POST", "/api/pipelines/{id}/webhook", doc.RouteDoc{
		Summary:     "Webhook-trigger pipeline",
		Description: "Authenticated webhook endpoint for remotely triggering a pipeline. Provide the configured webhook token via `X-Mimir-Webhook-Token`, query parameter `token`, or request body.",
		Tags:        []string{"Pipelines"},
		Params:      []doc.Param{doc.PParam("id", "Pipeline ID")},
		RequestBody: doc.JsonBody(doc.Ref("PipelineTriggerRequest")),
		Responses:   doc.R(doc.Accepted(doc.Ref("WorkTask")), doc.BadRequest(), doc.NotFound()),
	})
	doc.Register("GET", "/api/pipelines/{id}/checkpoints", doc.RouteDoc{
		Summary:     "Get pipeline checkpoint",
		Description: "Returns persisted pipeline checkpoint state for a pipeline step.",
		Tags:        []string{"Pipelines"},
		Params: []doc.Param{
			doc.PParam("id", "Pipeline ID"),
			doc.QParam("step_name", "Pipeline step name", true),
			doc.QParam("scope", "Optional checkpoint scope", false),
		},
		Responses: doc.R(doc.OK(doc.Ref("PipelineCheckpoint")), doc.NotFound(), doc.BadRequest()),
	})
	doc.Register("PUT", "/api/pipelines/{id}/checkpoints", doc.RouteDoc{
		Summary:     "Save pipeline checkpoint",
		Description: "Persists pipeline checkpoint state for a pipeline step with optimistic versioning.",
		Tags:        []string{"Pipelines"},
		Params: []doc.Param{
			doc.PParam("id", "Pipeline ID"),
			doc.QParam("step_name", "Pipeline step name", true),
			doc.QParam("scope", "Optional checkpoint scope", false),
		},
		RequestBody: doc.JsonBody(doc.Ref("PipelineCheckpoint")),
		Responses:   doc.R(doc.OK(doc.Ref("PipelineCheckpoint")), doc.BadRequest()),
	})

	doc.Register("GET", "/api/pipelines/{id}", doc.RouteDoc{
		Summary:     "Get pipeline",
		Description: "Returns a single pipeline by ID.",
		Tags:        []string{"Pipelines"},
		Params:      []doc.Param{doc.PParam("id", "Pipeline ID")},
		Responses:   doc.R(doc.OK(doc.Ref("Pipeline")), doc.NotFound()),
	})
	doc.Register("PUT", "/api/pipelines/{id}", doc.RouteDoc{
		Summary:     "Update pipeline",
		Description: "Updates a pipeline's description, steps, or status.",
		Tags:        []string{"Pipelines"},
		Params:      []doc.Param{doc.PParam("id", "Pipeline ID")},
		RequestBody: doc.JsonBody(doc.Ref("PipelineUpdateRequest")),
		Responses:   doc.R(doc.OK(doc.Ref("Pipeline")), doc.BadRequest(), doc.NotFound()),
	})
	doc.Register("DELETE", "/api/pipelines/{id}", doc.RouteDoc{
		Summary:     "Delete pipeline",
		Description: "Permanently deletes a pipeline when no persisted resources still reference it.",
		Tags:        []string{"Pipelines"},
		Params:      []doc.Param{doc.PParam("id", "Pipeline ID")},
		Responses:   doc.R(doc.NoContent(), doc.NotFound(), doc.BadRequest()),
	})

	// ── Schedules ─────────────────────────────────────────────────────────────
	doc.Register("GET", "/api/schedules", doc.RouteDoc{
		Summary:   "List schedules",
		Tags:      []string{"Schedules"},
		Responses: doc.R(doc.OK(doc.ArrOf("Schedule"))),
	})
	doc.Register("POST", "/api/schedules", doc.RouteDoc{
		Summary:     "Create schedule",
		Tags:        []string{"Schedules"},
		RequestBody: doc.JsonBody(doc.Ref("ScheduleCreateRequest")),
		Responses:   doc.R(doc.Created(doc.Ref("Schedule")), doc.BadRequest()),
	})
	doc.Register("GET", "/api/schedules/{id}", doc.RouteDoc{
		Summary:   "Get schedule",
		Tags:      []string{"Schedules"},
		Params:    []doc.Param{doc.PParam("id", "Schedule ID")},
		Responses: doc.R(doc.OK(doc.Ref("Schedule")), doc.NotFound()),
	})
	doc.Register("PUT", "/api/schedules/{id}", doc.RouteDoc{
		Summary:     "Update schedule",
		Tags:        []string{"Schedules"},
		Params:      []doc.Param{doc.PParam("id", "Schedule ID")},
		RequestBody: doc.JsonBody(doc.Ref("ScheduleUpdateRequest")),
		Responses:   doc.R(doc.OK(doc.Ref("Schedule")), doc.BadRequest(), doc.NotFound()),
	})
	doc.Register("DELETE", "/api/schedules/{id}", doc.RouteDoc{
		Summary:   "Delete schedule",
		Tags:      []string{"Schedules"},
		Params:    []doc.Param{doc.PParam("id", "Schedule ID")},
		Responses: doc.R(doc.NoContent(), doc.NotFound()),
	})
}

package api

import "github.com/mimir-aip/mimir-aip-go/pkg/api/doc"

func init() {
	// ── Digital Twins ──────────────────────────────────────────────────────────
	doc.Register("GET", "/api/digital-twins", doc.RouteDoc{
		Summary:     "List digital twins",
		Description: "Returns all digital twins for a project.",
		Tags:        []string{"Digital Twins"},
		Params:      []doc.Param{doc.QParam("project_id", "Filter by project ID", true)},
		Responses:   doc.R(doc.OK(doc.ArrOf("DigitalTwin"))),
	})
	doc.Register("POST", "/api/digital-twins", doc.RouteDoc{
		Summary:     "Create digital twin",
		Description: "Creates a persisted ontology-backed entity graph and initialises its starting entity set from the associated ontology.",
		Tags:        []string{"Digital Twins"},
		RequestBody: doc.JsonBody(doc.Ref("DigitalTwinCreateRequest")),
		Responses:   doc.R(doc.Created(doc.Ref("DigitalTwin")), doc.BadRequest()),
	})
	doc.Register("GET", "/api/digital-twins/{id}", doc.RouteDoc{
		Summary:   "Get digital twin",
		Tags:      []string{"Digital Twins"},
		Params:    []doc.Param{doc.PParam("id", "Digital twin ID")},
		Responses: doc.R(doc.OK(doc.Ref("DigitalTwin")), doc.NotFound()),
	})
	doc.Register("PUT", "/api/digital-twins/{id}", doc.RouteDoc{
		Summary:     "Update digital twin",
		Tags:        []string{"Digital Twins"},
		Params:      []doc.Param{doc.PParam("id", "Digital twin ID")},
		RequestBody: doc.JsonBody(doc.Ref("DigitalTwinUpdateRequest")),
		Responses:   doc.R(doc.OK(doc.Ref("DigitalTwin")), doc.BadRequest(), doc.NotFound()),
	})
	doc.Register("DELETE", "/api/digital-twins/{id}", doc.RouteDoc{
		Summary:   "Delete digital twin",
		Tags:      []string{"Digital Twins"},
		Params:    []doc.Param{doc.PParam("id", "Digital twin ID")},
		Responses: doc.R(doc.NoContent(), doc.NotFound()),
	})

	// ── Sync ───────────────────────────────────────────────────────────────────
	doc.Register("POST", "/api/digital-twins/{id}/sync", doc.RouteDoc{
		Summary:     "Queue digital twin sync",
		Description: "Enqueues background work to refresh the twin's entity graph from its configured storage backends.",
		Tags:        []string{"Digital Twins"},
		Params:      []doc.Param{doc.PParam("id", "Digital twin ID")},
		Responses: doc.Accepted(doc.Props(nil, doc.M{
			"work_task_id":    doc.Str("Queued work task ID"),
			"digital_twin_id": doc.Str("Digital twin ID"),
			"status":          doc.Str("'queued'"),
			"message":         doc.Str("Human-readable queue message"),
		})),
	})

	doc.Register("GET", "/api/digital-twins/{id}/history/runs", doc.RouteDoc{
		Summary:     "List twin sync runs",
		Description: "Returns digital twin synchronization/materialization runs, newest first. These are the version anchors for temporal twin history.",
		Tags:        []string{"Digital Twins"},
		Params: []doc.Param{
			doc.PParam("id", "Digital twin ID"),
			doc.QParam("limit", "Optional maximum number of sync runs to return", false),
		},
		Responses: doc.R(doc.OK(doc.ArrOf("TwinSyncRun")), doc.NotFound()),
	})
	doc.Register("GET", "/api/digital-twins/{id}/history/runs/{runId}", doc.RouteDoc{
		Summary:     "Get twin sync run",
		Description: "Returns one synchronization/materialization run by ID.",
		Tags:        []string{"Digital Twins"},
		Params: []doc.Param{
			doc.PParam("id", "Digital twin ID"),
			doc.PParam("runId", "Twin sync run ID"),
		},
		Responses: doc.R(doc.OK(doc.Ref("TwinSyncRun")), doc.NotFound()),
	})

	// ── Runs / Alerts / Automations ─────────────────────────────────────────────
	doc.Register("GET", "/api/digital-twins/{id}/runs", doc.RouteDoc{
		Summary:     "List twin processing runs",
		Description: "Returns the most recent explicit twin processing runs for this digital twin.",
		Tags:        []string{"Digital Twins"},
		Params: []doc.Param{
			doc.PParam("id", "Digital twin ID"),
			doc.QParam("limit", "Optional maximum number of runs to return", false),
		},
		Responses: doc.R(doc.OK(doc.ArrOf("TwinProcessingRun")), doc.NotFound()),
	})
	doc.Register("POST", "/api/digital-twins/{id}/runs", doc.RouteDoc{
		Summary:     "Queue twin processing run",
		Description: "Queues one explicit twin processing run using the current twin storage scope and automation stages.",
		Tags:        []string{"Digital Twins"},
		Params:      []doc.Param{doc.PParam("id", "Digital twin ID")},
		Responses:   doc.Accepted(doc.Ref("TwinProcessingRun")),
	})
	doc.Register("GET", "/api/digital-twins/{id}/runs/{runId}", doc.RouteDoc{
		Summary: "Get twin processing run",
		Tags:    []string{"Digital Twins"},
		Params: []doc.Param{
			doc.PParam("id", "Digital twin ID"),
			doc.PParam("runId", "Twin processing run ID"),
		},
		Responses: doc.R(doc.OK(doc.Ref("TwinProcessingRun")), doc.NotFound()),
	})
	doc.Register("GET", "/api/digital-twins/{id}/alerts", doc.RouteDoc{
		Summary:     "List alert events",
		Description: "Returns append-only alert events emitted during twin processing.",
		Tags:        []string{"Digital Twins"},
		Params: []doc.Param{
			doc.PParam("id", "Digital twin ID"),
			doc.QParam("limit", "Optional maximum number of alerts to return", false),
		},
		Responses: doc.R(doc.OK(doc.ArrOf("AlertEvent")), doc.NotFound()),
	})
	doc.Register("POST", "/api/digital-twins/{id}/alerts/{alertId}/approval", doc.RouteDoc{
		Summary:     "Review pending alert action",
		Description: "Applies an approve/reject decision to one alert event that is awaiting manual export approval.",
		Tags:        []string{"Digital Twins"},
		Params: []doc.Param{
			doc.PParam("id", "Digital twin ID"),
			doc.PParam("alertId", "Alert event ID"),
		},
		RequestBody: doc.JsonBody(doc.Ref("AlertApprovalRequest")),
		Responses:   doc.R(doc.OK(doc.Ref("AlertEvent")), doc.BadRequest(), doc.NotFound()),
	})

	doc.Register("GET", "/api/digital-twins/{id}/automations", doc.RouteDoc{
		Summary:     "List twin automations",
		Description: "Lists explicit automations scoped to this digital twin.",
		Tags:        []string{"Digital Twins"},
		Params:      []doc.Param{doc.PParam("id", "Digital twin ID")},
		Responses:   doc.R(doc.OK(doc.ArrOf("Automation")), doc.NotFound()),
	})
	doc.Register("POST", "/api/digital-twins/{id}/automations", doc.RouteDoc{
		Summary:     "Create twin automation",
		Description: "Creates an explicit automation scoped to this digital twin. Target metadata is derived from the route.",
		Tags:        []string{"Digital Twins"},
		Params:      []doc.Param{doc.PParam("id", "Digital twin ID")},
		RequestBody: doc.JsonBody(doc.Ref("AutomationCreateRequest")),
		Responses:   doc.R(doc.Created(doc.Ref("Automation")), doc.BadRequest()),
	})
	doc.Register("GET", "/api/digital-twins/{id}/automations/{automationId}", doc.RouteDoc{
		Summary: "Get twin automation",
		Tags:    []string{"Digital Twins"},
		Params: []doc.Param{
			doc.PParam("id", "Digital twin ID"),
			doc.PParam("automationId", "Automation ID"),
		},
		Responses: doc.R(doc.OK(doc.Ref("Automation")), doc.NotFound()),
	})
	doc.Register("PUT", "/api/digital-twins/{id}/automations/{automationId}", doc.RouteDoc{
		Summary: "Update twin automation",
		Tags:    []string{"Digital Twins"},
		Params: []doc.Param{
			doc.PParam("id", "Digital twin ID"),
			doc.PParam("automationId", "Automation ID"),
		},
		RequestBody: doc.JsonBody(doc.Ref("AutomationUpdateRequest")),
		Responses:   doc.R(doc.OK(doc.Ref("Automation")), doc.BadRequest(), doc.NotFound()),
	})
	doc.Register("DELETE", "/api/digital-twins/{id}/automations/{automationId}", doc.RouteDoc{
		Summary: "Delete twin automation",
		Tags:    []string{"Digital Twins"},
		Params: []doc.Param{
			doc.PParam("id", "Digital twin ID"),
			doc.PParam("automationId", "Automation ID"),
		},
		Responses: doc.R(doc.NoContent(), doc.NotFound()),
	})

	// ── Entities ───────────────────────────────────────────────────────────────
	doc.Register("GET", "/api/digital-twins/{id}/entities", doc.RouteDoc{
		Summary:   "List entities",
		Tags:      []string{"Digital Twins"},
		Params:    []doc.Param{doc.PParam("id", "Digital twin ID")},
		Responses: doc.R(doc.OK(doc.ArrOf("Entity"))),
	})
	doc.Register("GET", "/api/digital-twins/{id}/entities/{entityId}", doc.RouteDoc{
		Summary: "Get entity",
		Tags:    []string{"Digital Twins"},
		Params: []doc.Param{
			doc.PParam("id", "Digital twin ID"),
			doc.PParam("entityId", "Entity ID"),
		},
		Responses: doc.R(doc.OK(doc.Ref("Entity")), doc.NotFound()),
	})
	doc.Register("PUT", "/api/digital-twins/{id}/entities/{entityId}", doc.RouteDoc{
		Summary: "Update entity",
		Tags:    []string{"Digital Twins"},
		Params: []doc.Param{
			doc.PParam("id", "Digital twin ID"),
			doc.PParam("entityId", "Entity ID"),
		},
		RequestBody: doc.JsonBody(doc.Ref("EntityUpdateRequest")),
		Responses:   doc.R(doc.OK(doc.Ref("Entity")), doc.BadRequest(), doc.NotFound()),
	})
	doc.Register("GET", "/api/digital-twins/{id}/entities/{entityId}/history", doc.RouteDoc{
		Summary:     "Get entity history",
		Description: "Returns recorded entity revisions, newest first. Use the optional `limit` query parameter to bound the result size.",
		Tags:        []string{"Digital Twins"},
		Params: []doc.Param{
			doc.PParam("id", "Digital twin ID"),
			doc.PParam("entityId", "Entity ID"),
			doc.QParam("limit", "Maximum number of revisions to return", false),
		},
		Responses: doc.R(doc.OK(doc.ArrOf("EntityRevision"))),
	})
	doc.Register("GET", "/api/digital-twins/{id}/entities/{entityId}/related", doc.RouteDoc{
		Summary:     "Get related entities",
		Description: "Returns entities connected to the given entity by a typed relationship traversal.",
		Tags:        []string{"Digital Twins"},
		Params: []doc.Param{
			doc.PParam("id", "Digital twin ID"),
			doc.PParam("entityId", "Entity ID"),
			doc.QParam("relationship", "Relationship type to traverse", false),
		},
		Responses: doc.R(doc.OK(doc.ArrOf("Entity"))),
	})

	// ── SPARQL Query ───────────────────────────────────────────────────────────
	doc.Register("POST", "/api/digital-twins/{id}/query", doc.RouteDoc{
		Summary:     "Execute SPARQL query",
		Description: "Runs the digital twin's supported SPARQL-style SELECT query subset against the persisted entity graph.",
		Tags:        []string{"Digital Twins"},
		Params:      []doc.Param{doc.PParam("id", "Digital twin ID")},
		RequestBody: doc.JsonBody(doc.Ref("QueryRequest")),
		Responses:   doc.R(doc.OK(doc.Ref("QueryResult")), doc.BadRequest()),
	})

	// ── Prediction ─────────────────────────────────────────────────────────────
	doc.Register("POST", "/api/digital-twins/{id}/predict", doc.RouteDoc{
		Summary:     "Run prediction",
		Description: "Runs a single or batch inference using the twin's trained ML models. Provide a top-level 'inputs' array for batch mode; omit it for single-record mode.",
		Tags:        []string{"Digital Twins"},
		Params:      []doc.Param{doc.PParam("id", "Digital twin ID")},
		RequestBody: doc.JsonBody(doc.Ref("PredictionRequest")),
		Responses:   doc.R(doc.OK(doc.Ref("PredictionResult")), doc.BadRequest()),
	})

	// ── Scenarios ──────────────────────────────────────────────────────────────
	doc.Register("GET", "/api/digital-twins/{id}/scenarios", doc.RouteDoc{
		Summary:   "List scenarios",
		Tags:      []string{"Digital Twins"},
		Params:    []doc.Param{doc.PParam("id", "Digital twin ID")},
		Responses: doc.R(doc.OK(doc.ArrOf("Scenario"))),
	})
	doc.Register("POST", "/api/digital-twins/{id}/scenarios", doc.RouteDoc{
		Summary:     "Create scenario",
		Description: "Defines a what-if scenario by specifying entity attribute modifications. Results are computed in-memory; the live entity graph is never mutated.",
		Tags:        []string{"Digital Twins"},
		Params:      []doc.Param{doc.PParam("id", "Digital twin ID")},
		RequestBody: doc.JsonBody(doc.Ref("ScenarioCreateRequest")),
		Responses:   doc.R(doc.Created(doc.Ref("Scenario")), doc.BadRequest()),
	})
	doc.Register("GET", "/api/digital-twins/{id}/scenarios/{scenarioId}", doc.RouteDoc{
		Summary: "Get scenario",
		Tags:    []string{"Digital Twins"},
		Params: []doc.Param{
			doc.PParam("id", "Digital twin ID"),
			doc.PParam("scenarioId", "Scenario ID"),
		},
		Responses: doc.R(doc.OK(doc.Ref("Scenario")), doc.NotFound()),
	})
	doc.Register("DELETE", "/api/digital-twins/{id}/scenarios/{scenarioId}", doc.RouteDoc{
		Summary: "Delete scenario",
		Tags:    []string{"Digital Twins"},
		Params: []doc.Param{
			doc.PParam("id", "Digital twin ID"),
			doc.PParam("scenarioId", "Scenario ID"),
		},
		Responses: doc.R(doc.NoContent(), doc.NotFound()),
	})

	// ── Actions ────────────────────────────────────────────────────────────────
	doc.Register("GET", "/api/digital-twins/{id}/actions", doc.RouteDoc{
		Summary:   "List actions",
		Tags:      []string{"Digital Twins"},
		Params:    []doc.Param{doc.PParam("id", "Digital twin ID")},
		Responses: doc.R(doc.OK(doc.ArrOf("Action"))),
	})
	doc.Register("POST", "/api/digital-twins/{id}/actions", doc.RouteDoc{
		Summary:     "Create action",
		Description: "Registers a conditional pipeline trigger against digital twin attributes or prediction output.",
		Tags:        []string{"Digital Twins"},
		Params:      []doc.Param{doc.PParam("id", "Digital twin ID")},
		RequestBody: doc.JsonBody(doc.Ref("ActionCreateRequest")),
		Responses:   doc.R(doc.Created(doc.Ref("Action")), doc.BadRequest()),
	})
	doc.Register("GET", "/api/digital-twins/{id}/actions/{actionId}", doc.RouteDoc{
		Summary: "Get action",
		Tags:    []string{"Digital Twins"},
		Params: []doc.Param{
			doc.PParam("id", "Digital twin ID"),
			doc.PParam("actionId", "Action ID"),
		},
		Responses: doc.R(doc.OK(doc.Ref("Action")), doc.NotFound()),
	})
	doc.Register("DELETE", "/api/digital-twins/{id}/actions/{actionId}", doc.RouteDoc{
		Summary: "Delete action",
		Tags:    []string{"Digital Twins"},
		Params: []doc.Param{
			doc.PParam("id", "Digital twin ID"),
			doc.PParam("actionId", "Action ID"),
		},
		Responses: doc.R(doc.NoContent(), doc.NotFound()),
	})
}

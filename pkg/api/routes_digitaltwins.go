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
		Description: "Creates a new digital twin and initialises its entity graph from the associated ontology.",
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
		Summary:     "Sync digital twin",
		Description: "Pulls the latest records from the twin's linked storage backends and refreshes the entity graph.",
		Tags:        []string{"Digital Twins"},
		Params:      []doc.Param{doc.PParam("id", "Digital twin ID")},
		Responses:   doc.R(doc.OK(doc.Props(nil, doc.M{"status": doc.Str("'synced'")})), doc.NotFound()),
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
		Description: "Runs a SPARQL SELECT query against the twin's entity graph.",
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
		Description: "Registers an action that can be applied to entities within the digital twin.",
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

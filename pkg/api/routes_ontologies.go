package api

import "github.com/mimir-aip/mimir-aip-go/pkg/api/doc"

func init() {
	// ── Ontologies ─────────────────────────────────────────────────────────────
	doc.Register("GET", "/api/ontologies", doc.RouteDoc{
		Summary:     "List ontologies",
		Description: "Returns all ontologies for a project.",
		Tags:        []string{"Ontologies"},
		Params:      []doc.Param{doc.QParam("project_id", "Filter by project ID", true)},
		Responses:   doc.R(doc.OK(doc.ArrOf("Ontology"))),
	})
	doc.Register("POST", "/api/ontologies", doc.RouteDoc{
		Summary:     "Create ontology",
		Description: "Creates a new OWL/Turtle ontology for a project.",
		Tags:        []string{"Ontologies"},
		RequestBody: doc.JsonBody(doc.Ref("OntologyCreateRequest")),
		Responses:   doc.R(doc.Created(doc.Ref("Ontology")), doc.BadRequest()),
	})
	doc.Register("POST", "/api/ontologies/validate", doc.RouteDoc{
		Summary:     "Validate ontology content",
		Description: "Compiles OWL/Turtle content into Mimir's canonical ontology graph without persisting it. Error diagnostics make the response invalid; warnings are returned but do not block persistence.",
		Tags:        []string{"Ontologies"},
		RequestBody: doc.JsonBody(doc.Ref("OntologyValidationRequest")),
		Responses:   doc.R(doc.OK(doc.Ref("OntologyValidationResponse")), doc.BadRequest()),
	})
	doc.Register("GET", "/api/ontologies/{id}", doc.RouteDoc{
		Summary: "Get ontology",
		Tags:    []string{"Ontologies"},
		Params: []doc.Param{
			doc.PParam("id", "Ontology ID"),
			doc.QParam("project_id", "Owning project ID", true),
		},
		Responses: doc.R(doc.OK(doc.Ref("Ontology")), doc.NotFound(), map[string]doc.M{"403": {"description": "Forbidden — ontology belongs to another project"}}),
	})
	doc.Register("GET", "/api/ontologies/{id}/compiled", doc.RouteDoc{
		Summary:     "Get compiled ontology",
		Description: "Returns the persisted canonical ontology graph consumed by storage, retrieval, digital twins, aggregation, and predictive features.",
		Tags:        []string{"Ontologies"},
		Params: []doc.Param{
			doc.PParam("id", "Ontology ID"),
			doc.QParam("project_id", "Owning project ID", true),
		},
		Responses: doc.R(doc.OK(doc.Ref("CompiledOntology")), doc.NotFound(), map[string]doc.M{"403": {"description": "Forbidden — ontology belongs to another project"}}),
	})
	doc.Register("GET", "/api/ontologies/{id}/diagnostics", doc.RouteDoc{
		Summary:     "Get ontology diagnostics",
		Description: "Returns compiler diagnostics from the persisted ontology compilation.",
		Tags:        []string{"Ontologies"},
		Params: []doc.Param{
			doc.PParam("id", "Ontology ID"),
			doc.QParam("project_id", "Owning project ID", true),
		},
		Responses: doc.R(doc.OK(doc.ArrOf("OntologyDiagnostic")), doc.NotFound(), map[string]doc.M{"403": {"description": "Forbidden — ontology belongs to another project"}}),
	})
	doc.Register("GET", "/api/ontologies/{id}/search", doc.RouteDoc{
		Summary:     "Search ontology terms",
		Description: "Searches compiled class, property, and relation terms so downstream retrieval can resolve user language into ontology-backed entities and relationship paths.",
		Tags:        []string{"Ontologies"},
		Params: []doc.Param{
			doc.PParam("id", "Ontology ID"),
			doc.QParam("project_id", "Owning project ID", true),
			doc.QParam("q", "Search text; empty returns highest-weight terms", false),
			doc.QParam("limit", "Maximum results, default 20", false),
		},
		Responses: doc.R(doc.OK(doc.ArrOf("OntologySearchResult")), doc.BadRequest(), doc.NotFound(), map[string]doc.M{"403": {"description": "Forbidden — ontology belongs to another project"}}),
	})
	doc.Register("PUT", "/api/ontologies/{id}", doc.RouteDoc{
		Summary:     "Update ontology",
		Tags:        []string{"Ontologies"},
		Params:      []doc.Param{doc.PParam("id", "Ontology ID"), doc.QParam("project_id", "Owning project ID", true)},
		RequestBody: doc.JsonBody(doc.Ref("OntologyUpdateRequest")),
		Responses:   doc.R(doc.OK(doc.Ref("Ontology")), doc.BadRequest(), doc.NotFound(), map[string]doc.M{"403": {"description": "Forbidden — ontology belongs to another project"}}),
	})
	doc.Register("DELETE", "/api/ontologies/{id}", doc.RouteDoc{
		Summary:   "Delete ontology",
		Tags:      []string{"Ontologies"},
		Params:    []doc.Param{doc.PParam("id", "Ontology ID"), doc.QParam("project_id", "Owning project ID", true)},
		Responses: doc.R(doc.NoContent(), doc.NotFound(), map[string]doc.M{"403": {"description": "Forbidden — ontology belongs to another project"}}, map[string]doc.M{"409": {"description": "Conflict — ontology is still referenced by other project resources"}}),
	})

	// ── Extraction ─────────────────────────────────────────────────────────────
	doc.Register("POST", "/api/extraction/generate-ontology", doc.RouteDoc{
		Summary:     "Extract entities and generate ontology",
		Description: "Runs extraction over storage backends and creates or updates a generated ontology; persisted ontology content is compiled into the canonical graph before use by downstream systems.",
		Tags:        []string{"Extraction"},
		RequestBody: doc.JsonBody(doc.Ref("OntologyExtractionRequest")),
		Responses: doc.R(doc.Created(doc.Props(nil, doc.M{
			"ontology":           doc.Ref("Ontology"),
			"extraction_summary": doc.Obj("Entity and relationship counts"),
		})), doc.BadRequest()),
	})
}

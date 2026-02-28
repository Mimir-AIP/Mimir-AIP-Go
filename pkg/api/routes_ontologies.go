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
	doc.Register("GET", "/api/ontologies/{id}", doc.RouteDoc{
		Summary:   "Get ontology",
		Tags:      []string{"Ontologies"},
		Params:    []doc.Param{doc.PParam("id", "Ontology ID")},
		Responses: doc.R(doc.OK(doc.Ref("Ontology")), doc.NotFound()),
	})
	doc.Register("PUT", "/api/ontologies/{id}", doc.RouteDoc{
		Summary:     "Update ontology",
		Tags:        []string{"Ontologies"},
		Params:      []doc.Param{doc.PParam("id", "Ontology ID")},
		RequestBody: doc.JsonBody(doc.Ref("OntologyUpdateRequest")),
		Responses:   doc.R(doc.OK(doc.Ref("Ontology")), doc.BadRequest(), doc.NotFound()),
	})
	doc.Register("DELETE", "/api/ontologies/{id}", doc.RouteDoc{
		Summary:   "Delete ontology",
		Tags:      []string{"Ontologies"},
		Params:    []doc.Param{doc.PParam("id", "Ontology ID")},
		Responses: doc.R(doc.NoContent(), doc.NotFound()),
	})

	// ── Extraction ─────────────────────────────────────────────────────────────
	doc.Register("POST", "/api/extraction/generate-ontology", doc.RouteDoc{
		Summary:     "Extract entities and generate ontology",
		Description: "Runs the schema-inductive extraction algorithm over the specified storage backends, generates an OWL/Turtle ontology, and diffs it against existing active ontologies. If changes are detected the new ontology is flagged as 'needs_review' and the diff is returned.",
		Tags:        []string{"Extraction"},
		RequestBody: doc.JsonBody(doc.Ref("OntologyExtractionRequest")),
		Responses: doc.R(doc.Created(doc.Props(nil, doc.M{
			"ontology":           doc.Ref("Ontology"),
			"extraction_summary": doc.Obj("Entity and relationship counts"),
			"ontology_diff":      doc.Obj("Diff against the existing active ontology, if any"),
		})), doc.BadRequest()),
	})
}

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
		Summary: "Get ontology",
		Tags:    []string{"Ontologies"},
		Params: []doc.Param{
			doc.PParam("id", "Ontology ID"),
			doc.QParam("project_id", "Owning project ID", true),
		},
		Responses: doc.R(doc.OK(doc.Ref("Ontology")), doc.NotFound(), map[string]doc.M{"403": {"description": "Forbidden — ontology belongs to another project"}}),
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
		Description: "Runs the extraction algorithm over the specified storage backends and generates or updates a project ontology directly.",
		Tags:        []string{"Extraction"},
		RequestBody: doc.JsonBody(doc.Ref("OntologyExtractionRequest")),
		Responses: doc.R(doc.Created(doc.Props(nil, doc.M{
			"ontology":           doc.Ref("Ontology"),
			"extraction_summary": doc.Obj("Entity and relationship counts"),
		})), doc.BadRequest()),
	})
}

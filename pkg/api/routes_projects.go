package api

import "github.com/mimir-aip/mimir-aip-go/pkg/api/doc"

func init() {
	doc.Register("GET", "/api/projects", doc.RouteDoc{
		Summary:     "List projects",
		Description: "Returns all projects, optionally filtered by status.",
		Tags:        []string{"Projects"},
		Params:      []doc.Param{doc.QParam("status", "Filter by project status", false)},
		Responses:   doc.R(doc.OK(doc.ArrOf("Project"))),
	})
	doc.Register("POST", "/api/projects", doc.RouteDoc{
		Summary:     "Create project",
		Description: "Creates a new project.",
		Tags:        []string{"Projects"},
		RequestBody: doc.JsonBody(doc.Ref("ProjectCreateRequest")),
		Responses:   doc.R(doc.Created(doc.Ref("Project")), doc.BadRequest()),
	})
	doc.Register("POST", "/api/projects/{id}/clone", doc.RouteDoc{
		Summary:     "Clone project",
		Description: "Deep-clones an existing project into a new draft project, copying persisted project-owned configuration and remapping internal references to the cloned resources.",
		Tags:        []string{"Projects"},
		Params:      []doc.Param{doc.PParam("id", "Source project ID")},
		RequestBody: doc.JsonBody(doc.Ref("ProjectCloneRequest")),
		Responses:   doc.R(doc.Created(doc.Ref("Project")), doc.BadRequest(), doc.NotFound()),
	})
	doc.Register("GET", "/api/projects/{id}", doc.RouteDoc{
		Summary:     "Get project",
		Description: "Returns a single project by ID.",
		Tags:        []string{"Projects"},
		Params:      []doc.Param{doc.PParam("id", "Project ID")},
		Responses:   doc.R(doc.OK(doc.Ref("Project")), doc.NotFound()),
	})
	doc.Register("PUT", "/api/projects/{id}", doc.RouteDoc{
		Summary:     "Update project",
		Description: "Updates a project's name, description, or status.",
		Tags:        []string{"Projects"},
		Params:      []doc.Param{doc.PParam("id", "Project ID")},
		RequestBody: doc.JsonBody(doc.Ref("ProjectUpdateRequest")),
		Responses:   doc.R(doc.OK(doc.Ref("Project")), doc.BadRequest(), doc.NotFound()),
	})
	doc.Register("DELETE", "/api/projects/{id}", doc.RouteDoc{
		Summary:     "Archive project",
		Description: "Archives a project by marking its status archived. Project resources remain persisted and can still be inspected explicitly.",
		Tags:        []string{"Projects"},
		Params:      []doc.Param{doc.PParam("id", "Project ID")},
		Responses:   doc.R(doc.NoContent(), doc.NotFound()),
	})
	doc.Register("GET", "/api/projects/{id}/state-summary", doc.RouteDoc{
		Summary:     "Get project state summary",
		Description: "Returns a project-scoped backend activity summary for frontend section indicators.",
		Tags:        []string{"Projects"},
		Params:      []doc.Param{doc.PParam("id", "Project ID")},
		Responses:   doc.R(doc.OK(doc.Ref("ProjectStateSummary")), doc.NotFound()),
	})

	doc.Register("POST", "/api/projects/{id}/{componentType}/{componentId}", doc.RouteDoc{
		Summary:     "Add component to project",
		Description: "Associates a pipeline, ontology, ML model, digital twin, or storage config with a project.",
		Tags:        []string{"Projects"},
		Params: []doc.Param{
			doc.PParam("id", "Project ID"),
			doc.PParam("componentType", "pipelines | ontologies | mlmodels | digitaltwins | storage"),
			doc.PParam("componentId", "Component ID to associate"),
		},
		Responses: doc.R(doc.NoContent(), doc.BadRequest(), doc.NotFound()),
	})
	doc.Register("DELETE", "/api/projects/{id}/{componentType}/{componentId}", doc.RouteDoc{
		Summary:     "Remove component from project",
		Description: "Removes the association between a component and a project.",
		Tags:        []string{"Projects"},
		Params: []doc.Param{
			doc.PParam("id", "Project ID"),
			doc.PParam("componentType", "pipelines | ontologies | mlmodels | digitaltwins | storage"),
			doc.PParam("componentId", "Component ID to disassociate"),
		},
		Responses: doc.R(doc.NoContent(), doc.BadRequest(), doc.NotFound()),
	})
}

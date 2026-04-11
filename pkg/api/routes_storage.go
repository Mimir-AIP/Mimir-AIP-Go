package api

import "github.com/mimir-aip/mimir-aip-go/pkg/api/doc"

func init() {
	// ── Storage Configs ────────────────────────────────────────────────────────
	doc.Register("GET", "/api/storage/configs", doc.RouteDoc{
		Summary:     "List storage configs",
		Description: "Returns all storage configurations for a project.",
		Tags:        []string{"Storage"},
		Params:      []doc.Param{doc.QParam("project_id", "Filter by project ID", true)},
		Responses:   doc.R(doc.OK(doc.ArrOf("StorageConfig"))),
	})
	doc.Register("POST", "/api/storage/configs", doc.RouteDoc{
		Summary:     "Create storage config",
		Description: "Creates a new storage backend configuration for a project.",
		Tags:        []string{"Storage"},
		RequestBody: doc.JsonBody(doc.Ref("StorageConfigCreateRequest")),
		Responses:   doc.R(doc.Created(doc.Ref("StorageConfig")), doc.BadRequest()),
	})
	doc.Register("GET", "/api/storage/configs/{id}", doc.RouteDoc{
		Summary: "Get storage config",
		Tags:    []string{"Storage"},
		Params: []doc.Param{
			doc.PParam("id", "Storage config ID"),
			doc.QParam("project_id", "Owning project ID", true),
		},
		Responses: doc.R(doc.OK(doc.Ref("StorageConfig")), doc.NotFound(), map[string]doc.M{"403": {"description": "Forbidden — storage config belongs to another project"}}),
	})
	doc.Register("PUT", "/api/storage/configs/{id}", doc.RouteDoc{
		Summary:     "Update storage config",
		Tags:        []string{"Storage"},
		Params:      []doc.Param{doc.PParam("id", "Storage config ID"), doc.QParam("project_id", "Owning project ID", true)},
		RequestBody: doc.JsonBody(doc.Ref("StorageConfigUpdateRequest")),
		Responses:   doc.R(doc.OK(doc.Ref("StorageConfig")), doc.BadRequest(), doc.NotFound(), map[string]doc.M{"403": {"description": "Forbidden — storage config belongs to another project"}}),
	})
	doc.Register("DELETE", "/api/storage/configs/{id}", doc.RouteDoc{
		Summary:     "Delete storage config",
		Description: "Deletes a storage config only when no persisted project-owned resources still reference it.",
		Tags:        []string{"Storage"},
		Params:      []doc.Param{doc.PParam("id", "Storage config ID"), doc.QParam("project_id", "Owning project ID", true)},
		Responses: doc.R(
			doc.NoContent(),
			doc.NotFound(),
			map[string]doc.M{"403": {"description": "Forbidden — storage config belongs to another project"}},
			map[string]doc.M{"409": {"description": "Conflict — storage config is still referenced by project resources"}},
		),
	})

	// ── CIR Data Operations ────────────────────────────────────────────────────
	doc.Register("POST", "/api/storage/store", doc.RouteDoc{
		Summary:     "Store CIR data",
		Description: "Writes one or more CIR records to the specified project-owned storage backend.",
		Tags:        []string{"Storage"},
		RequestBody: doc.JsonBody(doc.Ref("StorageStoreRequest")),
		Responses:   doc.R(doc.OK(doc.Ref("StorageResult")), doc.BadRequest(), doc.NotFound(), map[string]doc.M{"403": {"description": "Forbidden — storage config belongs to another project"}}),
	})
	doc.Register("POST", "/api/storage/retrieve", doc.RouteDoc{
		Summary:     "Retrieve CIR data",
		Description: "Queries and returns CIR records from the specified project-owned storage backend.",
		Tags:        []string{"Storage"},
		RequestBody: doc.JsonBody(doc.Ref("StorageQueryRequest")),
		Responses:   doc.R(doc.OK(doc.ArrOf("CIR")), doc.BadRequest(), doc.NotFound(), map[string]doc.M{"403": {"description": "Forbidden — storage config belongs to another project"}}),
	})
	doc.Register("POST", "/api/storage/update", doc.RouteDoc{
		Summary:     "Update CIR data",
		Description: "Applies delta updates to matching CIR records in the specified project-owned storage backend.",
		Tags:        []string{"Storage"},
		RequestBody: doc.JsonBody(doc.Ref("StorageUpdateRequest")),
		Responses:   doc.R(doc.OK(doc.Ref("StorageResult")), doc.BadRequest(), doc.NotFound(), map[string]doc.M{"403": {"description": "Forbidden — storage config belongs to another project"}}),
	})
	doc.Register("POST", "/api/storage/delete", doc.RouteDoc{
		Summary:     "Delete CIR data",
		Description: "Deletes matching CIR records from the specified project-owned storage backend.",
		Tags:        []string{"Storage"},
		RequestBody: doc.JsonBody(doc.Ref("StorageDeleteRequest")),
		Responses:   doc.R(doc.OK(doc.Ref("StorageResult")), doc.BadRequest(), doc.NotFound(), map[string]doc.M{"403": {"description": "Forbidden — storage config belongs to another project"}}),
	})

	// ── Storage Health ─────────────────────────────────────────────────────────
	doc.Register("GET", "/api/storage/health", doc.RouteDoc{
		Summary:     "Storage health check",
		Description: "Checks connectivity to the underlying backend for the given project-owned storage config ID.",
		Tags:        []string{"Storage"},
		Params:      []doc.Param{doc.QParam("config_id", "Storage config ID", true), doc.QParam("project_id", "Owning project ID", true)},
		Responses: doc.R(doc.OK(doc.Props(nil, doc.M{
			"healthy": doc.Bool("Whether the backend is reachable"),
			"error":   doc.Str("Error message if unhealthy"),
		})), doc.NotFound(), map[string]doc.M{"403": {"description": "Forbidden — storage config belongs to another project"}}),
	})
	doc.Register("GET", "/api/storage/ingestion-health", doc.RouteDoc{
		Summary:     "Project ingestion health report",
		Description: "Computes freshness, completeness, and schema drift scores across all active storage sources in a project.",
		Tags:        []string{"Storage"},
		Params:      []doc.Param{doc.QParam("project_id", "Project ID", true)},
		Responses:   doc.R(doc.OK(doc.Ref("IngestionHealthReport")), doc.BadRequest(), doc.NotFound()),
	})
}

package api

import "github.com/mimir-aip/mimir-aip-go/pkg/api/doc"

func init() {
	// ── Pipeline Plugins ──────────────────────────────────────────────────────
	doc.Register("GET", "/api/plugins", doc.RouteDoc{
		Summary:     "List pipeline plugins",
		Description: "Returns all installed pipeline step plugins.",
		Tags:        []string{"Plugins"},
		Responses:   doc.R(doc.OK(doc.ArrOf("Plugin"))),
	})
	doc.Register("POST", "/api/plugins", doc.RouteDoc{
		Summary:     "Install pipeline plugin",
		Description: "Installs a pipeline step plugin from a Git repository. Workers clone and compile the plugin at runtime.",
		Tags:        []string{"Plugins"},
		RequestBody: doc.JsonBody(doc.Ref("PluginInstallRequest")),
		Responses:   doc.R(doc.Created(doc.Ref("Plugin")), doc.BadRequest()),
	})
	doc.Register("GET", "/api/plugins/{name}", doc.RouteDoc{
		Summary:   "Get pipeline plugin",
		Tags:      []string{"Plugins"},
		Params:    []doc.Param{doc.PParam("name", "Plugin name")},
		Responses: doc.R(doc.OK(doc.Ref("Plugin")), doc.NotFound()),
	})
	doc.Register("PUT", "/api/plugins/{name}", doc.RouteDoc{
		Summary:     "Update pipeline plugin",
		Description: "Pulls the latest version from the plugin's repository.",
		Tags:        []string{"Plugins"},
		Params:      []doc.Param{doc.PParam("name", "Plugin name")},
		RequestBody: doc.JsonBody(doc.Ref("PluginUpdateRequest")),
		Responses:   doc.R(doc.OK(doc.Ref("Plugin")), doc.NotFound()),
	})
	doc.Register("DELETE", "/api/plugins/{name}", doc.RouteDoc{
		Summary:   "Uninstall pipeline plugin",
		Tags:      []string{"Plugins"},
		Params:    []doc.Param{doc.PParam("name", "Plugin name")},
		Responses: doc.R(doc.NoContent(), doc.NotFound()),
	})

	// ── Dynamic Storage Plugins ───────────────────────────────────────────────
	doc.Register("GET", "/api/storage-plugins", doc.RouteDoc{
		Summary:     "List storage plugins",
		Description: "Returns all dynamically installed storage backend plugins.",
		Tags:        []string{"Storage Plugins"},
		Responses:   doc.R(doc.OK(doc.ArrOf("ExternalStoragePlugin"))),
	})
	doc.Register("POST", "/api/storage-plugins", doc.RouteDoc{
		Summary:     "Install storage plugin",
		Description: "Clones, compiles, and registers a custom storage backend plugin from a Git repository. The repository must export 'var Plugin' satisfying models.StoragePlugin.",
		Tags:        []string{"Storage Plugins"},
		RequestBody: doc.JsonBody(doc.Ref("ExternalStoragePluginInstallRequest")),
		Responses:   doc.R(doc.Created(doc.Ref("ExternalStoragePlugin")), doc.BadRequest(), doc.Unprocessable()),
	})
	doc.Register("GET", "/api/storage-plugins/{name}", doc.RouteDoc{
		Summary:   "Get storage plugin",
		Tags:      []string{"Storage Plugins"},
		Params:    []doc.Param{doc.PParam("name", "Plugin name")},
		Responses: doc.R(doc.OK(doc.Ref("ExternalStoragePlugin")), doc.NotFound()),
	})
	doc.Register("DELETE", "/api/storage-plugins/{name}", doc.RouteDoc{
		Summary:     "Uninstall storage plugin",
		Description: "Removes the plugin from the registry and deletes its cached .so. Note: Go plugins cannot be unloaded from memory; an orchestrator restart is required for full removal.",
		Tags:        []string{"Storage Plugins"},
		Params:      []doc.Param{doc.PParam("name", "Plugin name")},
		Responses:   doc.R(doc.NoContent(), doc.NotFound()),
	})
}

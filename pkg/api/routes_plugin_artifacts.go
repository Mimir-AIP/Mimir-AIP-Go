package api

import "github.com/mimir-aip/mimir-aip-go/pkg/api/doc"

func init() {
	doc.Register("GET", "/api/plugin-artifacts", doc.RouteDoc{
		Summary:     "Get active plugin artifact",
		Description: "Returns the active orchestrator-built artifact for a plugin kind/name pair. Workers use this metadata before downloading and verifying the artifact.",
		Tags:        []string{"Plugins"},
		Params: []doc.Param{
			doc.QParam("kind", "Plugin kind: pipeline | ml_provider | storage | llm_provider", true),
			doc.QParam("name", "Plugin or provider name", true),
		},
		Responses: doc.R(doc.OK(doc.Ref("PluginArtifact")), doc.BadRequest(), doc.NotFound()),
	})
	doc.Register("GET", "/api/plugin-artifacts/{id}", doc.RouteDoc{
		Summary:     "Get plugin artifact metadata",
		Description: "Returns metadata for one stored plugin artifact.",
		Tags:        []string{"Plugins"},
		Params:      []doc.Param{doc.PParam("id", "Artifact ID")},
		Responses:   doc.R(doc.OK(doc.Ref("PluginArtifact")), doc.NotFound()),
	})
	doc.Register("GET", "/api/plugin-artifacts/{id}/download", doc.RouteDoc{
		Summary:     "Download plugin artifact",
		Description: "Streams the compiled .so artifact. Workers verify the SHA-256 digest from artifact metadata before loading it.",
		Tags:        []string{"Plugins"},
		Params:      []doc.Param{doc.PParam("id", "Artifact ID")},
		Responses:   doc.R(doc.OK(doc.M{"type": "string", "format": "binary", "description": "Compiled Go plugin artifact"}), doc.NotFound()),
	})
}

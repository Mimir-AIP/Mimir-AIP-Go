package api

import "github.com/mimir-aip/mimir-aip-go/pkg/api/doc"

func init() {
	doc.Register("GET", "/api/connectors", doc.RouteDoc{
		Summary:     "List bundled connector templates",
		Description: "Returns the generic ingestion connector catalog used by guided onboarding and self-serve setup flows.",
		Tags:        []string{"Connectors"},
		Responses:   doc.R(doc.OK(doc.ArrOf("ConnectorTemplate"))),
	})

	doc.Register("POST", "/api/connectors", doc.RouteDoc{
		Summary:     "Materialize a bundled connector",
		Description: "Creates a standard pipeline and optional schedule from a bundled connector template.",
		Tags:        []string{"Connectors"},
		RequestBody: doc.JsonBody(doc.Ref("ConnectorSetupRequest")),
		Responses:   doc.R(doc.Created(doc.Ref("ConnectorSetupResponse")), doc.BadRequest()),
	})
}

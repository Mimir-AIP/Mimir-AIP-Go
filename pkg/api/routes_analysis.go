package api

import "github.com/mimir-aip/mimir-aip-go/pkg/api/doc"

func init() {
	doc.Register("POST", "/api/analysis/resolver", doc.RouteDoc{
		Summary:     "Run cross-source resolver analysis",
		Description: "Detects cross-source links, persists a review queue for uncertain links, and records resolver calibration metrics.",
		Tags:        []string{"Analysis"},
		Responses:   doc.R(doc.Created(doc.Obj("Resolver run result")), doc.BadRequest()),
	})

	doc.Register("GET", "/api/analysis/resolver/metrics", doc.RouteDoc{
		Summary:     "Get resolver precision metrics",
		Description: "Returns high-confidence precision and prior resolver run metrics over time for one project.",
		Tags:        []string{"Analysis"},
		Params:      []doc.Param{doc.QParam("project_id", "Project ID", true)},
		Responses:   doc.R(doc.OK(doc.Obj("Resolver metrics")), doc.BadRequest()),
	})

	doc.Register("GET", "/api/reviews", doc.RouteDoc{
		Summary:     "List review queue items",
		Description: "Lists persisted pending or decided review items for one project.",
		Tags:        []string{"Analysis"},
		Params: []doc.Param{
			doc.QParam("project_id", "Project ID", true),
			doc.QParam("status", "Optional review item status filter", false),
		},
		Responses: doc.R(doc.OK(doc.ArrOf("ReviewItem")), doc.BadRequest()),
	})

	doc.Register("POST", "/api/reviews/{id}/decision", doc.RouteDoc{
		Summary:     "Decide a review item",
		Description: "Accepts or rejects one review item and persists the feedback for future resolver calibration.",
		Tags:        []string{"Analysis"},
		Responses:   doc.R(doc.OK(doc.Ref("ReviewItem")), doc.BadRequest()),
	})

	doc.Register("GET", "/api/insights", doc.RouteDoc{
		Summary:     "List project insights",
		Description: "Lists persisted autonomous insights filtered by severity and confidence.",
		Tags:        []string{"Insights"},
		Params: []doc.Param{
			doc.QParam("project_id", "Project ID", true),
			doc.QParam("severity", "Optional severity filter", false),
			doc.QParam("min_confidence", "Optional minimum confidence filter", false),
		},
		Responses: doc.R(doc.OK(doc.ArrOf("Insight")), doc.BadRequest()),
	})

	doc.Register("POST", "/api/insights", doc.RouteDoc{
		Summary:     "Generate project insights",
		Description: "Runs generic anomaly, trend-break, and co-occurrence detectors for one project and persists the resulting insights.",
		Tags:        []string{"Insights"},
		Responses:   doc.R(doc.Created(doc.Obj("Insight generation result")), doc.BadRequest()),
	})
}

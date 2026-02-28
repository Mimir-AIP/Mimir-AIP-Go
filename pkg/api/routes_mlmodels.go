package api

import "github.com/mimir-aip/mimir-aip-go/pkg/api/doc"

func init() {
	// ── ML Models ──────────────────────────────────────────────────────────────
	doc.Register("GET", "/api/ml-models", doc.RouteDoc{
		Summary:     "List ML models",
		Description: "Returns all ML models for a project.",
		Tags:        []string{"ML Models"},
		Params:      []doc.Param{doc.QParam("project_id", "Filter by project ID", true)},
		Responses:   doc.R(doc.OK(doc.ArrOf("MLModel"))),
	})
	doc.Register("POST", "/api/ml-models", doc.RouteDoc{
		Summary:     "Create ML model",
		Description: "Creates a new ML model record (training must be triggered separately).",
		Tags:        []string{"ML Models"},
		RequestBody: doc.JsonBody(doc.Ref("ModelCreateRequest")),
		Responses:   doc.R(doc.Created(doc.Ref("MLModel")), doc.BadRequest()),
	})
	doc.Register("GET", "/api/ml-models/{id}", doc.RouteDoc{
		Summary:   "Get ML model",
		Tags:      []string{"ML Models"},
		Params:    []doc.Param{doc.PParam("id", "Model ID")},
		Responses: doc.R(doc.OK(doc.Ref("MLModel")), doc.NotFound()),
	})
	doc.Register("PUT", "/api/ml-models/{id}", doc.RouteDoc{
		Summary:     "Update ML model",
		Tags:        []string{"ML Models"},
		Params:      []doc.Param{doc.PParam("id", "Model ID")},
		RequestBody: doc.JsonBody(doc.Ref("ModelUpdateRequest")),
		Responses:   doc.R(doc.OK(doc.Ref("MLModel")), doc.BadRequest(), doc.NotFound()),
	})
	doc.Register("DELETE", "/api/ml-models/{id}", doc.RouteDoc{
		Summary:   "Delete ML model",
		Tags:      []string{"ML Models"},
		Params:    []doc.Param{doc.PParam("id", "Model ID")},
		Responses: doc.R(doc.NoContent(), doc.NotFound()),
	})

	// ── Training Actions ───────────────────────────────────────────────────────
	doc.Register("POST", "/api/ml-models/train", doc.RouteDoc{
		Summary:     "Trigger model training",
		Description: "Enqueues a Kubernetes training job for the specified model. Returns 202 Accepted immediately; poll the model's status field or listen on the WebSocket for completion.",
		Tags:        []string{"ML Models"},
		RequestBody: doc.JsonBody(doc.Ref("ModelTrainingRequest")),
		Responses:   doc.R(doc.Accepted(doc.Ref("MLModel")), doc.BadRequest()),
	})
	doc.Register("POST", "/api/ml-models/recommend", doc.RouteDoc{
		Summary:     "Recommend model type",
		Description: "Analyses the project's ontology and returns a suggested model type (e.g. decision_tree, neural_network) along with a confidence score.",
		Tags:        []string{"ML Models"},
		RequestBody: doc.JsonBody(doc.Ref("ModelRecommendationRequest")),
		Responses:   doc.R(doc.OK(doc.Ref("ModelRecommendation")), doc.BadRequest()),
	})

	// ── Worker Callbacks (called by Kubernetes training jobs) ──────────────────
	doc.Register("POST", "/api/ml-models/{id}/training/complete", doc.RouteDoc{
		Summary:     "Report training complete",
		Description: "Called by the worker job to record the trained artifact path and performance metrics.",
		Tags:        []string{"ML Models"},
		Params:      []doc.Param{doc.PParam("id", "Model ID")},
		RequestBody: doc.JsonBody(doc.Ref("TrainingCompleteRequest")),
		Responses:   doc.R(doc.OK(doc.Props(nil, doc.M{"status": doc.Str("'training completed'")})), doc.BadRequest()),
	})
	doc.Register("POST", "/api/ml-models/{id}/training/fail", doc.RouteDoc{
		Summary:     "Report training failure",
		Description: "Called by the worker job to record a training failure reason.",
		Tags:        []string{"ML Models"},
		Params:      []doc.Param{doc.PParam("id", "Model ID")},
		RequestBody: doc.JsonBody(doc.Props(nil, doc.M{"reason": doc.Str("Failure reason")})),
		Responses:   doc.R(doc.OK(doc.Props(nil, doc.M{"status": doc.Str("'training failed'")})), doc.BadRequest()),
	})
}

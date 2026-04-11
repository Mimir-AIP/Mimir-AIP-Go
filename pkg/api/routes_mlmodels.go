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
		Params:    []doc.Param{doc.PParam("id", "Model ID"), doc.QParam("project_id", "Owning project ID", true)},
		Responses: doc.R(doc.OK(doc.Ref("MLModel")), doc.NotFound(), map[string]doc.M{"403": {"description": "Forbidden — model belongs to another project"}}),
	})
	doc.Register("PUT", "/api/ml-models/{id}", doc.RouteDoc{
		Summary:     "Update ML model",
		Tags:        []string{"ML Models"},
		Params:      []doc.Param{doc.PParam("id", "Model ID"), doc.QParam("project_id", "Owning project ID", true)},
		RequestBody: doc.JsonBody(doc.Ref("ModelUpdateRequest")),
		Responses:   doc.R(doc.OK(doc.Ref("MLModel")), doc.BadRequest(), doc.NotFound(), map[string]doc.M{"403": {"description": "Forbidden — model belongs to another project"}}),
	})
	doc.Register("DELETE", "/api/ml-models/{id}", doc.RouteDoc{
		Summary:   "Delete ML model",
		Tags:      []string{"ML Models"},
		Params:    []doc.Param{doc.PParam("id", "Model ID"), doc.QParam("project_id", "Owning project ID", true)},
		Responses: doc.R(doc.NoContent(), doc.NotFound(), map[string]doc.M{"403": {"description": "Forbidden — model belongs to another project"}}, map[string]doc.M{"409": {"description": "Conflict — model is still referenced by twin actions, predictions, or active work tasks"}}),
	})

	// ── Training Actions ───────────────────────────────────────────────────────
	doc.Register("POST", "/api/ml-models/train", doc.RouteDoc{
		Summary:     "Trigger model training",
		Description: "Queues model training and returns the updated model immediately. The response now includes `training_task_id`, which is the canonical async handle for polling `/api/worktasks/{id}` or subscribing to `/ws/tasks`.",
		Tags:        []string{"ML Models"},
		RequestBody: doc.JsonBody(doc.Ref("ModelTrainingRequest")),
		Responses:   doc.R(doc.Accepted(doc.Ref("MLModel")), doc.BadRequest()),
	})
	doc.Register("POST", "/api/ml-models/recommend", doc.RouteDoc{
		Summary:     "Recommend model type",
		Description: "Analyses the project's ontology and storage and returns a suggested built-in model type plus score-based reasoning.",
		Tags:        []string{"ML Models"},
		RequestBody: doc.JsonBody(doc.Ref("ModelRecommendationRequest")),
		Responses:   doc.R(doc.OK(doc.Ref("ModelRecommendation")), doc.BadRequest()),
	})

	// ── Worker Callbacks (called by Kubernetes training jobs) ──────────────────
	doc.Register("POST", "/api/ml-models/{id}/training/complete", doc.RouteDoc{
		Summary:     "Report training complete",
		Description: "Called by the worker job to upload the trained model artifact bytes and final performance metrics. The orchestrator persists the artifact and clears the model's `training_task_id`.",
		Tags:        []string{"ML Models"},
		Params:      []doc.Param{doc.PParam("id", "Model ID")},
		RequestBody: doc.JsonBody(doc.Ref("TrainingCompleteRequest")),
		Responses:   doc.R(doc.OK(doc.Props(nil, doc.M{"status": doc.Str("'training completed'")})), doc.BadRequest()),
	})
	doc.Register("POST", "/api/ml-models/{id}/training/fail", doc.RouteDoc{
		Summary:     "Report training failure",
		Description: "Called by the worker job to record a training failure reason and clear the model's `training_task_id`.",
		Tags:        []string{"ML Models"},
		Params:      []doc.Param{doc.PParam("id", "Model ID")},
		RequestBody: doc.JsonBody(doc.Ref("TrainingFailRequest")),
		Responses:   doc.R(doc.OK(doc.Props(nil, doc.M{"status": doc.Str("'training failed'")})), doc.BadRequest()),
	})
}

package doc

func mlModelSchemas() M {
	return M{
		// ── ML Models ─────────────────────────────────────────────────────────
		// ── ML Models ─────────────────────────────────────────────────────────
		"MLSemanticFeature": Props([]string{"class_id", "property_id"}, M{
			"class_id":      Str("Ontology class ID"),
			"property_id":   Str("Ontology property ID"),
			"source_fields": Arr(M{"type": "string"}),
			"storage_ids":   Arr(M{"type": "string"}),
			"range":         Arr(M{"type": "string"}),
		}),
		"MLFeatureProvenance": Props([]string{"ontology_id", "storage_ids", "features", "generated_at"}, M{
			"ontology_id":  Str("Training ontology ID"),
			"content_hash": Str("Compiled ontology hash observed at training start"),
			"storage_ids":  Arr(M{"type": "string"}),
			"features":     ArrOf("MLSemanticFeature"),
			"generated_at": Str("ISO-8601 provenance timestamp"),
		}),
		"MLModel": Props(nil, M{
			"id":                   Str("ML model ID (UUID)"),
			"project_id":           Str("Owning project ID"),
			"ontology_id":          Str("Linked ontology ID"),
			"name":                 Str("Model name"),
			"description":          Str("Model description"),
			"type":                 Str("decision_tree | random_forest | regression | neural_network"),
			"status":               Str("draft | training | trained | failed | degraded | deprecated | archived"),
			"version":              Str("Model version string"),
			"is_recommended":       Bool("True if recommended by the recommendation engine"),
			"recommendation_score": Int("Recommendation engine score"),
			"training_config":      Obj("Training configuration"),
			"training_metrics":     Obj("Metrics recorded during training"),
			"training_task_id":     Str("Canonical async work-task handle while training is in progress"),
			"model_artifact_path":  Str("Orchestrator-local path to the persisted model artifact"),
			"performance_metrics":  Obj("Latest performance metrics"),
			"metadata":             M{"type": "object", "additionalProperties": true},
			"created_at":           Str("ISO-8601 creation timestamp"),
			"updated_at":           Str("ISO-8601 last-updated timestamp"),
		}),
		"ModelCreateRequest": Props([]string{"project_id", "ontology_id", "name", "type"}, M{
			"project_id":      Str("Owning project ID"),
			"ontology_id":     Str("Linked ontology ID"),
			"name":            Str("Model name"),
			"description":     Str("Description"),
			"type":            Str("decision_tree | random_forest | regression | neural_network"),
			"training_config": Obj("Training configuration"),
			"metadata":        M{"type": "object", "additionalProperties": true},
		}),
		"ModelUpdateRequest": Props(nil, M{
			"name":                Str("New name"),
			"description":         Str("New description"),
			"training_metrics":    Obj("Updated training metrics"),
			"performance_metrics": Obj("Updated performance metrics"),
			"status":              Str("New status"),
			"metadata":            M{"type": "object", "additionalProperties": true},
		}),
		"ModelRecommendationRequest": Props([]string{"project_id", "ontology_id"}, M{
			"project_id":  Str("Project ID"),
			"ontology_id": Str("Ontology to base recommendation on"),
		}),
		"ModelRecommendation": Props(nil, M{
			"recommended_type":  Str("Recommended model type"),
			"score":             Int("Score (0-100)"),
			"reasoning":         Str("Explanation"),
			"all_scores":        M{"type": "object", "additionalProperties": M{"type": "integer"}},
			"ontology_analysis": Obj("Ontology analysis used in the recommendation"),
			"data_analysis":     Obj("Data analysis used in the recommendation"),
		}),
		"ModelTrainingRequest": Props([]string{"model_id"}, M{
			"model_id":        Str("ID of the model to train"),
			"storage_ids":     Arr(M{"type": "string"}),
			"training_config": Obj("Optional training configuration override"),
		}),
		"TrainingCompleteRequest": Props([]string{"artifact_data_base64", "performance_metrics"}, M{
			"artifact_data_base64": Str("Base64-encoded serialized model artifact uploaded by the worker"),
			"performance_metrics":  Obj("Final performance metrics for the trained model"),
		}),
		"TrainingFailRequest": Props([]string{"reason"}, M{
			"reason": Str("Training failure reason recorded by the worker"),
		}),
	}
}

package doc

func init() {
	RegisterSchemas(M{
		// ── Metrics ──────────────────────────────────────────────────────────
		"MetricsResponse": Props(nil, M{
			"queue":           Props(nil, M{"length": Int("Current queue depth")}),
			"tasks_by_status": ObjMap("integer"),
			"tasks_by_type":   ObjMap("integer"),
			"timestamp":       Str("ISO-8601 UTC snapshot timestamp"),
		}),

		// ── Projects ─────────────────────────────────────────────────────────
		"Project": Props(nil, M{
			"id":          Str("Project ID (UUID)"),
			"name":        Str("Project name"),
			"description": Str("Project description"),
			"status":      Str("active | archived"),
			"components": Props(nil, M{
				"pipelines":     Arr(M{"type": "string"}),
				"ontologies":    Arr(M{"type": "string"}),
				"ml_models":     Arr(M{"type": "string"}),
				"digital_twins": Arr(M{"type": "string"}),
				"storage":       Arr(M{"type": "string"}),
			}),
			"created_at": Str("ISO-8601 creation timestamp"),
			"updated_at": Str("ISO-8601 last-updated timestamp"),
		}),
		"ProjectCreateRequest": Props([]string{"name"}, M{
			"name":        Str("Project name"),
			"description": Str("Project description"),
		}),
		"ProjectUpdateRequest": Props(nil, M{
			"name":        Str("New name"),
			"description": Str("New description"),
			"status":      Str("New status"),
		}),
		"ProjectCloneRequest": Props([]string{"project_id", "name"}, M{
			"project_id": Str("ID of the project to clone"),
			"name":       Str("Name for the cloned project"),
		}),

		// ── Pipelines ────────────────────────────────────────────────────────
		"Pipeline": Props(nil, M{
			"id":          Str("Pipeline ID (UUID)"),
			"project_id":  Str("Owning project ID"),
			"name":        Str("Pipeline name"),
			"type":        Str("ingestion | processing | output"),
			"description": Str("Pipeline description"),
			"steps":       ArrOf("PipelineStep"),
			"status":      Str("Pipeline status"),
			"created_at":  Str("ISO-8601 creation timestamp"),
			"updated_at":  Str("ISO-8601 last-updated timestamp"),
		}),
		"PipelineStep": Props([]string{"name", "plugin", "action"}, M{
			"name":       Str("Step name (unique within pipeline)"),
			"plugin":     Str("Plugin name or 'default' for built-in actions"),
			"action":     Str("Action name within the plugin"),
			"parameters": M{"type": "object", "additionalProperties": true},
			"output":     ObjMap("string"),
		}),
		"PipelineCreateRequest": Props([]string{"project_id", "name", "type", "steps"}, M{
			"project_id":  Str("Owning project ID"),
			"name":        Str("Pipeline name"),
			"type":        Str("Pipeline type"),
			"description": Str("Pipeline description"),
			"steps":       ArrOf("PipelineStep"),
		}),
		"PipelineUpdateRequest": Props(nil, M{
			"description": Str("New description"),
			"steps":       ArrOf("PipelineStep"),
			"status":      Str("New status"),
		}),
		"PipelineCheckpoint": Props([]string{"project_id", "pipeline_id", "step_name", "version", "checkpoint"}, M{
			"project_id":  Str("Owning project ID"),
			"pipeline_id": Str("Pipeline ID"),
			"step_name":   Str("Pipeline step name"),
			"scope":       Str("Optional checkpoint scope"),
			"version":     Int("Optimistic lock version"),
			"checkpoint":  M{"type": "object", "additionalProperties": true},
			"created_at":  Str("ISO-8601 creation timestamp"),
			"updated_at":  Str("ISO-8601 last-updated timestamp"),
		}),
		"PipelineExecutionRequest": Props([]string{"pipeline_id"}, M{
			"pipeline_id":  Str("Pipeline ID to execute"),
			"trigger_type": Str("manual | scheduled | automatic"),
			"triggered_by": Str("Trigger source identifier"),
			"parameters":   M{"type": "object", "additionalProperties": true},
		}),

		// ── Schedules ─────────────────────────────────────────────────────────
		"Schedule": Props(nil, M{
			"id":           Str("Schedule ID (UUID)"),
			"name":         Str("Schedule name"),
			"cron":         Str("Cron expression (e.g. '0 * * * *')"),
			"pipeline_ids": Arr(M{"type": "string"}),
			"enabled":      Bool("Whether the schedule is active"),
			"created_at":   Str("ISO-8601 creation timestamp"),
			"updated_at":   Str("ISO-8601 last-updated timestamp"),
		}),
		"ScheduleCreateRequest": Props([]string{"name", "cron", "pipeline_ids"}, M{
			"name":         Str("Schedule name"),
			"cron":         Str("Cron expression"),
			"pipeline_ids": Arr(M{"type": "string"}),
			"enabled":      Bool("Start enabled (default true)"),
		}),
		"ScheduleUpdateRequest": Props(nil, M{
			"name":         Str("New name"),
			"cron":         Str("New cron expression"),
			"pipeline_ids": Arr(M{"type": "string"}),
			"enabled":      Bool("Enable or disable"),
		}),

		// ── Pipeline Plugins ─────────────────────────────────────────────────
		"Plugin": Props(nil, M{
			"name":           Str("Plugin name"),
			"version":        Str("Plugin version"),
			"description":    Str("Plugin description"),
			"author":         Str("Plugin author"),
			"repository_url": Str("Git repository URL"),
			"actions":        Arr(M{"type": "string"}),
			"installed_at":   Str("ISO-8601 installation timestamp"),
		}),
		"PluginInstallRequest": Props([]string{"repository_url"}, M{
			"repository_url": Str("HTTPS Git URL of the plugin repository"),
			"git_ref":        Str("Branch, tag, or commit SHA (default: main)"),
		}),
		"PluginUpdateRequest": Props(nil, M{
			"git_ref": Str("Branch, tag, or commit SHA to update to"),
		}),

		// ── Storage ──────────────────────────────────────────────────────────
		"StorageConfig": Props(nil, M{
			"id":          Str("Storage config ID (UUID)"),
			"project_id":  Str("Owning project ID"),
			"plugin_type": Str("filesystem | postgresql | mysql | mongodb | s3 | redis | elasticsearch | neo4j | <custom>"),
			"config":      M{"type": "object", "additionalProperties": true},
			"ontology_id": Str("Optional linked ontology ID"),
			"active":      Bool("Whether this config is active"),
			"created_at":  Str("ISO-8601 creation timestamp"),
			"updated_at":  Str("ISO-8601 last-updated timestamp"),
		}),
		"StorageConfigCreateRequest": Props([]string{"project_id", "plugin_type", "config"}, M{
			"project_id":  Str("Owning project ID"),
			"plugin_type": Str("Backend type"),
			"config":      M{"type": "object", "additionalProperties": true},
			"ontology_id": Str("Optional linked ontology ID"),
		}),
		"StorageConfigUpdateRequest": Props(nil, M{
			"config":      M{"type": "object", "additionalProperties": true},
			"ontology_id": Str("Linked ontology ID"),
			"active":      Bool("Active flag"),
		}),
		"CIR": M{
			"description": "Common Internal Representation — the normalised record format used across all storage backends.",
			"type":        "object",
			"properties": M{
				"id":       Str("CIR record ID"),
				"source":   M{"type": "object", "additionalProperties": true, "description": "Provenance information"},
				"data":     M{"type": "object", "additionalProperties": true, "description": "Record payload"},
				"metadata": M{"type": "object", "additionalProperties": true, "description": "Record metadata"},
			},
		},
		"CIRQuery": Props(nil, M{
			"entity_type": Str("Filter by entity type"),
			"filters":     ArrOf("CIRCondition"),
			"order_by":    ArrOf("OrderByClause"),
			"limit":       Int("Maximum results"),
			"offset":      Int("Pagination offset"),
		}),
		"CIRCondition": Props([]string{"attribute", "operator", "value"}, M{
			"attribute": Str("Attribute name"),
			"operator":  Str("eq | neq | gt | gte | lt | lte | in | like"),
			"value":     M{"description": "Filter value"},
		}),
		"OrderByClause": Props([]string{"attribute"}, M{
			"attribute": Str("Attribute to sort by"),
			"direction": Str("asc | desc (default asc)"),
		}),
		"StorageResult": Props(nil, M{
			"success":        Bool("Whether the operation succeeded"),
			"affected_items": Int("Number of records affected"),
			"error":          Str("Error message if unsuccessful"),
		}),
		"IngestionHealthSource": Props([]string{"storage_id", "plugin_type", "sample_size", "freshness_score", "completeness_score", "schema_drift_score", "overall_score", "status"}, M{
			"storage_id":         Str("Storage config ID"),
			"plugin_type":        Str("Storage backend plugin type"),
			"sample_size":        Int("Number of sampled CIR records"),
			"last_ingested_at":   Str("Latest ingestion timestamp in sample"),
			"freshness_score":    M{"type": "number", "description": "Freshness score [0,1]"},
			"completeness_score": M{"type": "number", "description": "Completeness score [0,1]"},
			"schema_drift_score": M{"type": "number", "description": "Schema stability score [0,1], higher is better"},
			"overall_score":      M{"type": "number", "description": "Weighted ingestion health score [0,1]"},
			"status":             Str("healthy | warning | critical"),
			"findings":           Arr(M{"type": "string"}),
		}),
		"IngestionHealthReport": Props([]string{"project_id", "generated_at", "overall_score", "status", "sources"}, M{
			"project_id":      Str("Project ID"),
			"generated_at":    Str("Report generation timestamp"),
			"overall_score":   M{"type": "number", "description": "Project-level ingestion health score [0,1]"},
			"status":          Str("healthy | warning | critical"),
			"sources":         ArrOf("IngestionHealthSource"),
			"recommendations": Arr(M{"type": "string"}),
		}),
		"StorageStoreRequest": Props([]string{"project_id", "storage_id", "cir_data"}, M{
			"project_id": Str("Project ID"),
			"storage_id": Str("Storage config ID"),
			"cir_data":   Ref("CIR"),
		}),
		"StorageQueryRequest": Props([]string{"project_id", "storage_id"}, M{
			"project_id": Str("Project ID"),
			"storage_id": Str("Storage config ID"),
			"query":      Ref("CIRQuery"),
		}),
		"StorageUpdateRequest": Props([]string{"project_id", "storage_id", "query", "updates"}, M{
			"project_id": Str("Project ID"),
			"storage_id": Str("Storage config ID"),
			"query":      Ref("CIRQuery"),
			"updates": Props(nil, M{
				"filters": ArrOf("CIRCondition"),
				"updates": M{"type": "object", "additionalProperties": true},
			}),
		}),
		"StorageDeleteRequest": Props([]string{"project_id", "storage_id", "query"}, M{
			"project_id": Str("Project ID"),
			"storage_id": Str("Storage config ID"),
			"query":      Ref("CIRQuery"),
		}),

		// ── Storage Plugins (dynamic) ─────────────────────────────────────────
		"ExternalStoragePlugin": Props(nil, M{
			"name":            Str("Plugin name (derived from repository URL)"),
			"version":         Str("Version from plugin.yaml"),
			"description":     Str("Description from plugin.yaml"),
			"author":          Str("Author from plugin.yaml"),
			"repository_url":  Str("Git repository URL"),
			"git_commit_hash": Str("Commit SHA the current .so was compiled from"),
			"status":          Str("active | error"),
			"error_message":   Str("Compilation or load error, if status=error"),
			"installed_at":    Str("ISO-8601 installation timestamp"),
			"updated_at":      Str("ISO-8601 last-updated timestamp"),
		}),
		"ExternalStoragePluginInstallRequest": Props([]string{"repository_url"}, M{
			"repository_url": Str("HTTPS Git URL of the storage plugin repository"),
			"git_ref":        Str("Branch, tag, or commit SHA (default: main)"),
		}),

		// ── Analysis ──────────────────────────────────────────────────────────
		"AnalysisRun": Props(nil, M{
			"id":                Str("Analysis run ID (UUID)"),
			"project_id":        Str("Owning project ID"),
			"kind":              Str("resolver | insights"),
			"status":            Str("completed | failed"),
			"source_ids":        Arr(M{"type": "string"}),
			"algorithm_version": Str("Detector/resolver algorithm version"),
			"policy_version":    Str("Resolver policy version string"),
			"coverage":          M{"type": "object", "additionalProperties": true},
			"metrics":           M{"type": "object", "additionalProperties": true},
			"error":             Str("Failure message, when status=failed"),
			"created_at":        Str("Run creation timestamp"),
			"completed_at":      Str("Run completion timestamp"),
		}),
		"ReviewItem": Props(nil, M{
			"id":                 Str("Review item ID (UUID)"),
			"project_id":         Str("Owning project ID"),
			"run_id":             Str("Analysis run ID"),
			"finding_type":       Str("cross_source_link or another finding category"),
			"status":             Str("pending | accepted | rejected | auto_accepted"),
			"suggested_decision": Str("Suggested decision from the scoring policy"),
			"confidence":         M{"type": "number", "description": "Finding confidence [0,1]"},
			"payload":            M{"type": "object", "additionalProperties": true},
			"evidence":           M{"type": "object", "additionalProperties": true},
			"rationale":          Str("Human-readable explanation"),
			"reviewer":           Str("Reviewer identity, if decided"),
			"reviewed_at":        Str("Decision timestamp"),
			"created_at":         Str("Creation timestamp"),
			"updated_at":         Str("Update timestamp"),
		}),
		"ReviewDecisionRequest": Props([]string{"decision"}, M{
			"decision":  Str("accept | reject"),
			"rationale": Str("Optional review rationale"),
			"reviewer":  Str("Optional reviewer identifier"),
		}),
		"Insight": Props(nil, M{
			"id":               Str("Insight ID (UUID)"),
			"project_id":       Str("Owning project ID"),
			"run_id":           Str("Analysis run that produced this insight"),
			"type":             Str("anomaly_spike | trend_break | cooccurrence_surge | other generic detector type"),
			"severity":         Str("low | medium | high | critical"),
			"confidence":       M{"type": "number", "description": "Insight confidence [0,1]"},
			"explanation":      Str("Human-readable rationale"),
			"suggested_action": Str("Suggested follow-up action"),
			"evidence":         M{"type": "object", "additionalProperties": true},
			"status":           Str("Insight lifecycle state"),
			"created_at":       Str("Creation timestamp"),
			"updated_at":       Str("Update timestamp"),
		}),
		"InsightGenerateRequest": Props([]string{"project_id"}, M{
			"project_id": Str("Project ID"),
		}),
		"ResolverRunRequest": Props([]string{"project_id", "storage_ids"}, M{
			"project_id":  Str("Project ID"),
			"storage_ids": Arr(M{"type": "string"}),
		}),

		// ── Ontologies ────────────────────────────────────────────────────────
		"Ontology": Props(nil, M{
			"id":          Str("Ontology ID (UUID)"),
			"project_id":  Str("Owning project ID"),
			"name":        Str("Ontology name"),
			"description": Str("Ontology description"),
			"content":     Str("OWL/Turtle ontology content"),
			"status":      Str("draft | active | needs_review | deprecated"),
			"created_at":  Str("ISO-8601 creation timestamp"),
			"updated_at":  Str("ISO-8601 last-updated timestamp"),
		}),
		"OntologyCreateRequest": Props([]string{"project_id", "name"}, M{
			"project_id":  Str("Owning project ID"),
			"name":        Str("Ontology name"),
			"description": Str("Description"),
			"content":     Str("Initial OWL/Turtle content"),
		}),
		"OntologyUpdateRequest": Props(nil, M{
			"name":        Str("New name"),
			"description": Str("New description"),
			"content":     Str("Updated OWL/Turtle content"),
			"status":      Str("New status"),
		}),

		// ── Extraction ────────────────────────────────────────────────────────
		"ExtractionRequest": Props([]string{"project_id", "storage_id"}, M{
			"project_id":  Str("Project ID"),
			"storage_id":  Str("Storage config ID to extract from"),
			"ontology_id": Str("Existing ontology to diff against (optional)"),
		}),
		"ExtractionResult": Props(nil, M{
			"ontology":     Ref("Ontology"),
			"diff":         Obj("OntologyDiff — added/removed/modified classes and properties"),
			"needs_review": Bool("True if the diff is significant enough to flag for human review"),
		}),

		// ── ML Models ─────────────────────────────────────────────────────────
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
			"model_artifact_path":  Str("Path to the serialised model artifact"),
			"performance_metrics":  Obj("Latest performance metrics"),
			"metadata":             M{"type": "object", "additionalProperties": true},
			"created_at":           Str("ISO-8601 creation timestamp"),
			"updated_at":           Str("ISO-8601 last-updated timestamp"),
		}),
		"MLModelCreateRequest": Props([]string{"project_id", "name", "type"}, M{
			"project_id":      Str("Owning project ID"),
			"ontology_id":     Str("Linked ontology ID"),
			"name":            Str("Model name"),
			"description":     Str("Description"),
			"type":            Str("Model type"),
			"training_config": Obj("Training configuration"),
		}),
		"MLModelUpdateRequest": Props(nil, M{
			"name":            Str("New name"),
			"description":     Str("New description"),
			"training_config": Obj("Updated training configuration"),
			"status":          Str("New status"),
		}),
		"MLModelRecommendRequest": Props([]string{"project_id"}, M{
			"project_id":  Str("Project ID"),
			"ontology_id": Str("Ontology to base recommendation on"),
			"storage_id":  Str("Storage config to sample data from"),
		}),
		"MLModelRecommendation": Props(nil, M{
			"recommended_type": Str("Recommended model type"),
			"score":            Int("Confidence score (0-100)"),
			"reason":           Str("Explanation"),
			"alternatives":     Arr(Obj("Alternative model type with score and reason")),
		}),
		"MLTrainingRequest": Props([]string{"model_id"}, M{
			"model_id":   Str("ID of the model to train"),
			"storage_id": Str("Storage config to load training data from"),
		}),

		// ── Digital Twins ─────────────────────────────────────────────────────
		"DigitalTwin": Props(nil, M{
			"id":           Str("Digital twin ID (UUID)"),
			"project_id":   Str("Owning project ID"),
			"name":         Str("Digital twin name"),
			"description":  Str("Description"),
			"ontology_id":  Str("Linked ontology ID"),
			"status":       Str("initialising | active | syncing | error"),
			"entity_count": Int("Number of entities in the graph"),
			"metadata":     M{"type": "object", "additionalProperties": true},
			"created_at":   Str("ISO-8601 creation timestamp"),
			"updated_at":   Str("ISO-8601 last-updated timestamp"),
		}),
		"DigitalTwinCreateRequest": Props([]string{"project_id", "name"}, M{
			"project_id":  Str("Owning project ID"),
			"ontology_id": Str("Linked ontology ID"),
			"name":        Str("Digital twin name"),
			"description": Str("Description"),
		}),
		"DigitalTwinUpdateRequest": Props(nil, M{
			"name":        Str("New name"),
			"description": Str("New description"),
			"ontology_id": Str("New linked ontology ID"),
		}),
		"Entity": Props(nil, M{
			"id":              Str("Entity ID (UUID)"),
			"type":            Str("Entity type (from ontology)"),
			"attributes":      M{"type": "object", "additionalProperties": true},
			"computed_values": M{"type": "object", "additionalProperties": true},
			"relationships":   Arr(Obj("Relationship to another entity")),
			"last_updated":    Str("ISO-8601 last-updated timestamp"),
		}),
		"ScenarioRequest": Props([]string{"modifications"}, M{
			"scenario_id":   Str("Optional scenario ID (generated if omitted)"),
			"modifications": Arr(Obj("Entity attribute modification")),
		}),
		"ScenarioResult": Props(nil, M{
			"scenario_id": Str("Scenario ID"),
			"entities":    ArrOf("Entity"),
			"predictions": M{"type": "object", "additionalProperties": true},
			"created_at":  Str("ISO-8601 timestamp"),
		}),
		"ActionRequest": Props([]string{"action_type"}, M{
			"action_type": Str("Type of action to apply"),
			"parameters":  M{"type": "object", "additionalProperties": true},
		}),
		"ActionResult": Props(nil, M{
			"action_id":  Str("Action ID"),
			"status":     Str("Action status"),
			"result":     M{"type": "object", "additionalProperties": true},
			"applied_at": Str("ISO-8601 timestamp"),
		}),

		// ── Work Tasks ────────────────────────────────────────────────────────
		"WorkTask": Props(nil, M{
			"id":                    Str("Work task ID (UUID)"),
			"type":                  Str("pipeline_execution | ml_training | ml_inference | digital_twin_update"),
			"status":                Str("queued | running | completed | failed"),
			"priority":              Int("Task priority (higher = processed first)"),
			"project_id":            Str("Owning project ID"),
			"submitted_at":          Str("ISO-8601 submission timestamp"),
			"started_at":            Str("ISO-8601 start timestamp"),
			"completed_at":          Str("ISO-8601 completion timestamp"),
			"error_message":         Str("Error message if task failed"),
			"retry_count":           Int("Number of retries so far"),
			"max_retries":           Int("Maximum retry attempts"),
			"task_spec":             M{"type": "object", "additionalProperties": true},
			"resource_requirements": M{"type": "object", "additionalProperties": true},
		}),
		"WorkTaskSubmissionRequest": Props([]string{"type", "project_id"}, M{
			"type":                  Str("Task type"),
			"project_id":            Str("Owning project ID"),
			"priority":              Int("Task priority"),
			"task_spec":             M{"type": "object", "additionalProperties": true},
			"resource_requirements": M{"type": "object", "additionalProperties": true},
			"data_access":           M{"type": "object", "additionalProperties": true},
		}),
		"WorkTaskResult": Props([]string{"status"}, M{
			"status":        Str("completed | failed"),
			"error_message": Str("Error message if failed"),
			"result":        M{"type": "object", "additionalProperties": true},
		}),
	})
}

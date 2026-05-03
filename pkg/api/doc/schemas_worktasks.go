package doc

func workTaskSchemas() M {
	return M{
		// ── Work Tasks ────────────────────────────────────────────────────────
		"WorkTask": Props(nil, M{
			"id":                    Str("Work task ID (UUID)"),
			"type":                  Str("pipeline_execution | ml_training | ml_inference | digital_twin_processing"),
			"status":                Str("queued | scheduled | spawned | executing | completed | failed | timeout | cancelled"),
			"priority":              Int("Task priority (higher = processed first)"),
			"project_id":            Str("Owning project ID"),
			"submitted_at":          Str("ISO-8601 submission timestamp"),
			"started_at":            Str("ISO-8601 start timestamp"),
			"completed_at":          Str("ISO-8601 completion timestamp"),
			"output_location":       Str("Worker output location, if any"),
			"result_metadata":       M{"type": "object", "additionalProperties": true},
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
		"WorkTaskListResponse": Props([]string{"tasks", "queue_length"}, M{
			"tasks":        ArrOf("WorkTask"),
			"queue_length": Int("Number of currently queued tasks"),
		}),
		"WorkTaskResult": Props([]string{"status"}, M{
			"status":          Str("queued | scheduled | spawned | executing | completed | failed | timeout | cancelled"),
			"output_location": Str("Worker output location, if any"),
			"metadata":        M{"type": "object", "additionalProperties": true},
			"error_message":   Str("Error message if failed"),
		}),
	}
}

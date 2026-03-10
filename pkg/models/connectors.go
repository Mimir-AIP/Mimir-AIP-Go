package models

// ConnectorFieldOption describes one allowed value for a connector field.
type ConnectorFieldOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// ConnectorField describes one configurable input for a bundled connector.
type ConnectorField struct {
	Name        string                 `json:"name"`
	Label       string                 `json:"label"`
	Type        string                 `json:"type"`
	Description string                 `json:"description,omitempty"`
	Required    bool                   `json:"required"`
	Default     interface{}            `json:"default,omitempty"`
	Options     []ConnectorFieldOption `json:"options,omitempty"`
}

// ConnectorTemplate describes a broad, reusable bundled ingestion source.
type ConnectorTemplate struct {
	Kind             string           `json:"kind"`
	Label            string           `json:"label"`
	Description      string           `json:"description"`
	Category         string           `json:"category"`
	PipelineType     PipelineType     `json:"pipeline_type"`
	SupportsSchedule bool             `json:"supports_schedule"`
	Fields           []ConnectorField `json:"fields"`
}

// ConnectorScheduleRequest optionally creates a recurring schedule for the connector pipeline.
type ConnectorScheduleRequest struct {
	Name         string `json:"name,omitempty"`
	CronSchedule string `json:"cron_schedule"`
	Enabled      bool   `json:"enabled"`
}

// ConnectorSetupRequest materializes a bundled connector into standard Mimir resources.
type ConnectorSetupRequest struct {
	ProjectID    string                    `json:"project_id"`
	Kind         string                    `json:"kind"`
	Name         string                    `json:"name"`
	Description  string                    `json:"description,omitempty"`
	StorageID    string                    `json:"storage_id"`
	SourceConfig map[string]interface{}    `json:"source_config"`
	Schedule     *ConnectorScheduleRequest `json:"schedule,omitempty"`
}

// ConnectorSetupResponse returns the concrete resources created from a connector template.
type ConnectorSetupResponse struct {
	Template ConnectorTemplate `json:"template"`
	Pipeline *Pipeline         `json:"pipeline"`
	Schedule *Schedule         `json:"schedule,omitempty"`
}

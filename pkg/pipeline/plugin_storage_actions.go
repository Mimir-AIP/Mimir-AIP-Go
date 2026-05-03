package pipeline

import (
	"fmt"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

// ── Storage ───────────────────────────────────────────────────────────────────

// storeCIR stores a single CIR record into Mimir storage.
//
// Parameters:
//   - storage_id  (string, required): ID of the Mimir storage config to write to.
//   - data        (map|string, required): The record data. A JSON string is decoded automatically.
//   - source_uri  (string, optional): Source URI for provenance. Default: "pipeline://ingestion".
//   - source_type (string, optional): "api", "file", "database", or "stream". Default: "api".
//   - format      (string, optional): "json", "csv", etc. Default: "json".
func (p *DefaultPlugin) storeCIR(params map[string]interface{}, ctx *models.PipelineContext) (map[string]interface{}, error) {
	if p.storageSvc == nil {
		return nil, fmt.Errorf("store_cir: storage service is not available in this pipeline context; ensure the orchestrator has been configured to inject the storage service into pipelines")
	}

	storageID, ok := params["storage_id"].(string)
	if !ok || storageID == "" {
		return nil, fmt.Errorf("store_cir: storage_id parameter is required")
	}
	storageID = p.ResolveTemplates(storageID, ctx)

	sourceURI, _ := params["source_uri"].(string)
	sourceURI = p.ResolveTemplates(sourceURI, ctx)
	if sourceURI == "" {
		sourceURI = "pipeline://ingestion"
	}

	sourceTypeStr, _ := params["source_type"].(string)
	if sourceTypeStr == "" {
		sourceTypeStr = "api"
	}

	formatStr, _ := params["format"].(string)
	if formatStr == "" {
		formatStr = "json"
	}

	rawData, hasData := params["data"]
	if !hasData {
		return nil, fmt.Errorf("store_cir: data parameter is required")
	}

	data, err := p.resolveData(rawData, ctx)
	if err != nil {
		return nil, fmt.Errorf("store_cir: %w", err)
	}

	cir := models.NewCIR(
		models.SourceType(sourceTypeStr),
		sourceURI,
		models.DataFormat(formatStr),
		data,
	)

	result, err := p.storageSvc.Store(storageID, cir)
	if err != nil {
		return nil, fmt.Errorf("store_cir: failed to store CIR: %w", err)
	}

	return map[string]interface{}{
		"stored":         true,
		"affected_items": result.AffectedItems,
	}, nil
}

// storeCIRBatch stores an array of records as individual CIR entries.
//
// Parameters:
//   - storage_id  (string, required): ID of the Mimir storage config.
//   - items       ([]interface{}|string, required): Array of records, or a template resolving to one.
//   - source_uri  (string, optional): Source URI for provenance.
//   - source_type (string, optional): Default "api".
//   - format      (string, optional): Default "json".
func (p *DefaultPlugin) storeCIRBatch(params map[string]interface{}, ctx *models.PipelineContext) (map[string]interface{}, error) {
	if p.storageSvc == nil {
		return nil, fmt.Errorf("store_cir_batch: storage service is not available")
	}

	storageID, ok := params["storage_id"].(string)
	if !ok || storageID == "" {
		return nil, fmt.Errorf("store_cir_batch: storage_id parameter is required")
	}
	storageID = p.ResolveTemplates(storageID, ctx)

	sourceURI, _ := params["source_uri"].(string)
	sourceURI = p.ResolveTemplates(sourceURI, ctx)
	if sourceURI == "" {
		sourceURI = "pipeline://ingestion"
	}

	sourceTypeStr, _ := params["source_type"].(string)
	if sourceTypeStr == "" {
		sourceTypeStr = "api"
	}

	formatStr, _ := params["format"].(string)
	if formatStr == "" {
		formatStr = "json"
	}

	// Resolve items array
	rawItems, hasItems := params["items"]
	if !hasItems {
		return nil, fmt.Errorf("store_cir_batch: items parameter is required")
	}

	items, err := p.resolveArray(rawItems, ctx)
	if err != nil {
		return nil, fmt.Errorf("store_cir_batch: %w", err)
	}

	stored := 0
	for i, item := range items {
		cir := models.NewCIR(
			models.SourceType(sourceTypeStr),
			sourceURI,
			models.DataFormat(formatStr),
			item,
		)
		if _, err := p.storageSvc.Store(storageID, cir); err != nil {
			return nil, fmt.Errorf("store_cir_batch: failed to store item %d: %w", i, err)
		}
		stored++
	}

	return map[string]interface{}{
		"stored": stored,
		"total":  len(items),
	}, nil
}

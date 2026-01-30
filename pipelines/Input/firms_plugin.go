package Input

// NASA FIRMS plugin - lightweight implementation fetching CSV from FIRMS public endpoints

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
)

// FIRMSPlugin fetches wildfire detections from NASA FIRMS
type FIRMSPlugin struct{}

func NewFIRMSPlugin() *FIRMSPlugin { return &FIRMSPlugin{} }

func (p *FIRMSPlugin) ExecuteStep(ctx context.Context, stepConfig pipelines.StepConfig, globalContext *pipelines.PluginContext) (*pipelines.PluginContext, error) {
	cfg := stepConfig.Config
	// minimal config: source (VIIRS_SNPP | MODIS), days back or start/end
	source, _ := cfg["source"].(string)
	if source == "" {
		source = "VIIRS_SNPP"
	}

	days := 7
	if d, ok := cfg["days"].(int); ok {
		days = d
	}
	if d64, ok := cfg["days"].(float64); ok {
		days = int(d64)
	}

	// construct CSV URL (public archive)
	// Example: https://firms.modaps.eosdis.nasa.gov/api/area/csv?start=2023-01-01&end=2023-01-07&source=VIIRS_SNPP
	end := time.Now().UTC()
	start := end.AddDate(0, 0, -days)
	// allow bbox param or area param
	bbox, _ := cfg["bbox"].(string)
	area, _ := cfg["area"].(string)
	apiKey, _ := cfg["api_key"].(string)
	if apiKey == "" {
		apiKey = os.Getenv("FIRMS_API_KEY")
	}

	base := "https://firms.modaps.eosdis.nasa.gov/api/area/csv"
	if b, ok := cfg["base_url"].(string); ok && b != "" {
		base = b
	} else if envb := os.Getenv("FIRMS_API_BASE"); envb != "" {
		base = envb
	}
	qs := fmt.Sprintf("?start=%s&end=%s&source=%s", start.Format("2006-01-02"), end.Format("2006-01-02"), source)
	if bbox != "" {
		qs += fmt.Sprintf("&bbox=%s", bbox)
	}
	if area != "" {
		qs += fmt.Sprintf("&area=%s", area)
	}
	if apiKey != "" {
		qs += fmt.Sprintf("&api_key=%s", apiKey)
	}
	url := base + qs

	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	req.Header.Set("User-Agent", "Mimir-AIP/1.0")
	client := &http.Client{Timeout: 30 * time.Second}

	// simple retry/backoff
	var resp *http.Response
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		resp, err = client.Do(req)
		if err == nil && resp.StatusCode == 200 {
			break
		}
		// drain and close if non-nil
		if resp != nil {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
		time.Sleep(time.Duration(attempt+1) * 500 * time.Millisecond)
	}
	if err != nil {
		return nil, fmt.Errorf("firms request failed: %w", err)
	}
	if resp == nil {
		return nil, errors.New("firms: no response")
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("firms returned %d: %s", resp.StatusCode, string(body))
	}

	r := csv.NewReader(resp.Body)
	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV: %w", err)
	}

	var cols []string
	var rows []map[string]any
	if len(records) > 0 {
		cols = records[0]
		for _, rec := range records[1:] {
			row := make(map[string]any)
			for i, v := range rec {
				if i < len(cols) {
					// try numeric
					if f, err := strconv.ParseFloat(v, 64); err == nil {
						row[cols[i]] = f
					} else {
						row[cols[i]] = v
					}
				}
			}
			rows = append(rows, row)
		}
	}

	out := pipelines.NewPluginContext()
	outKey := stepConfig.Output
	if outKey == "" {
		outKey = "firms_data"
	}
	out.Set(outKey, map[string]any{"columns": cols, "rows": rows, "fetched_at": time.Now().Format(time.RFC3339)})
	return out, nil
}

func (p *FIRMSPlugin) GetPluginType() string { return "Input" }
func (p *FIRMSPlugin) GetPluginName() string { return "firms" }

func (p *FIRMSPlugin) ValidateConfig(config map[string]any) error { return nil }

func (p *FIRMSPlugin) GetInputSchema() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{"source": map[string]any{"type": "string"}, "days": map[string]any{"type": "integer"}}}
}

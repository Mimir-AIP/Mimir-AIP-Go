// Package doc provides the OpenAPI route registry for Mimir AIP.
//
// Each handler file in pkg/api registers its routes and documentation by
// calling Register() from an init() function.  The generator at
// cmd/openapi-gen/main.go imports _ "github.com/mimir-aip/mimir-aip-go/pkg/api"
// (which triggers every handler's init()) then calls GenerateSpec() to produce
// the full OpenAPI 3.0 YAML document.
//
// To document a new endpoint, add a Register() call inside the handler
// file's init() function.  No other files need to change — the spec is
// rebuilt automatically.
package doc

import (
	"fmt"
	"os"
	"sort"

	"gopkg.in/yaml.v3"
)

// M is a JSON/YAML object — shorthand used throughout schema definitions.
type M = map[string]interface{}

// S is a JSON/YAML array.
type S = []interface{}

// Param describes a single path or query parameter.
type Param struct {
	Name        string
	In          string // "path" | "query"
	Required    bool
	Description string
}

// RouteDoc contains the OpenAPI documentation for one HTTP operation.
type RouteDoc struct {
	Summary     string
	Description string
	Tags        []string
	Params      []Param
	RequestBody M              // nil → no request body
	Responses   map[string]M   // HTTP status string → response object
}

// Route is a fully described API endpoint stored in the registry.
type Route struct {
	Method string
	Path   string
	Doc    RouteDoc
}

var (
	routes  []Route
	schemas M // component schemas, set by RegisterSchemas
)

// Register adds a route to the global registry.
// It is safe to call from multiple init() functions.
func Register(method, path string, doc RouteDoc) {
	routes = append(routes, Route{Method: method, Path: path, Doc: doc})
}

// RegisterSchemas sets the component schemas map.
// Call this once from the schemas init() function.
func RegisterSchemas(s M) {
	schemas = s
}

// GenerateSpec produces the full OpenAPI 3.0 YAML specification.
func GenerateSpec() (string, error) {
	// Sort routes for stable output.
	sorted := make([]Route, len(routes))
	copy(sorted, routes)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Path != sorted[j].Path {
			return sorted[i].Path < sorted[j].Path
		}
		return sorted[i].Method < sorted[j].Method
	})

	paths := M{}
	for _, r := range sorted {
		if _, ok := paths[r.Path]; !ok {
			paths[r.Path] = M{}
		}
		paths[r.Path].(M)[methodKey(r.Method)] = buildOperation(r)
	}

	spec := M{
		"openapi": "3.0.3",
		"info": M{
			"title":       "Mimir AIP REST API",
			"description": "REST API for the Mimir AIP orchestrator. This specification is generated automatically from inline route registrations — it stays in sync with the codebase by construction.",
			"version":     "0.1.0",
			"contact": M{
				"name": "Mimir AIP",
				"url":  "https://github.com/Mimir-AIP/Mimir-AIP-Go",
			},
			"license": M{
				"name": "See repository",
				"url":  "https://github.com/Mimir-AIP/Mimir-AIP-Go/blob/main/LICENSE",
			},
		},
		"servers": S{
			M{"url": "http://localhost:8080", "description": "Local development"},
		},
		"tags": S{
			M{"name": "System", "description": "Health, readiness, and platform metrics"},
			M{"name": "Projects", "description": "Top-level organisational units"},
			M{"name": "Pipelines", "description": "Ordered processing step sequences"},
			M{"name": "Schedules", "description": "Cron-based pipeline triggers"},
			M{"name": "Connectors", "description": "Bundled ingestion templates that materialize into pipelines and schedules"},
			M{"name": "Analysis", "description": "Cross-source resolution, review queues, and calibration metrics"},
			M{"name": "Insights", "description": "Autonomous persisted findings generated from project storage data"},
			M{"name": "Plugins", "description": "Pipeline step executor plugins (Git-based, dynamic)"},
			M{"name": "Storage", "description": "Storage backend configuration and CIR data operations"},
			M{"name": "Storage Plugins", "description": "Dynamic storage backend plugins (Git-based, runtime-compiled)"},
			M{"name": "Ontologies", "description": "OWL/Turtle vocabulary management"},
			M{"name": "Extraction", "description": "Entity extraction and ontology generation from storage data"},
			M{"name": "ML Models", "description": "ML model lifecycle: create, train, infer, monitor"},
			M{"name": "Digital Twins", "description": "Live entity graphs with SPARQL querying and scenario analysis"},
			M{"name": "Tasks", "description": "Work task queue (internal / worker-facing endpoints)"},
		},
		"paths":      paths,
		"components": M{"schemas": schemas},
	}

	data, err := yaml.Marshal(spec)
	if err != nil {
		return "", fmt.Errorf("doc: failed to marshal OpenAPI spec: %w", err)
	}
	return string(data), nil
}

// WriteSpec writes the generated spec to path, creating parent dirs as needed.
func WriteSpec(path string) error {
	spec, err := GenerateSpec()
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(spec), 0644); err != nil {
		return fmt.Errorf("doc: failed to write spec to %s: %w", path, err)
	}
	return nil
}

// ── internal helpers ──────────────────────────────────────────────────────────

func methodKey(m string) string {
	switch m {
	case "GET":
		return "get"
	case "POST":
		return "post"
	case "PUT":
		return "put"
	case "DELETE":
		return "delete"
	case "PATCH":
		return "patch"
	default:
		return m
	}
}

func buildOperation(r Route) M {
	op := M{
		"summary":     r.Doc.Summary,
		"description": r.Doc.Description,
		"tags":        r.Doc.Tags,
		"responses":   r.Doc.Responses,
	}
	if len(r.Doc.Params) > 0 {
		params := make(S, 0, len(r.Doc.Params))
		for _, p := range r.Doc.Params {
			params = append(params, M{
				"name":        p.Name,
				"in":          p.In,
				"required":    p.Required,
				"description": p.Description,
				"schema":      M{"type": "string"},
			})
		}
		op["parameters"] = params
	}
	if r.Doc.RequestBody != nil {
		op["requestBody"] = r.Doc.RequestBody
	}
	return op
}

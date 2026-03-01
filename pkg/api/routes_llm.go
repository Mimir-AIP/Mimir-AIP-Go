package api

import "github.com/mimir-aip/mimir-aip-go/pkg/api/doc"

func init() {
	// ── LLM ────────────────────────────────────────────────────────────────────
	doc.Register("GET", "/api/llm/models", doc.RouteDoc{
		Summary:     "List available LLM models",
		Description: "Returns available models from the configured provider with TTL caching. Empty list when LLM not configured.",
		Tags:        []string{"LLM"},
		Responses: doc.R(doc.OK(doc.Props(nil, doc.M{
			"provider": doc.Str("Provider name or 'none'"),
			"enabled":  doc.Bool("Whether LLM is configured"),
			"models":   doc.ArrOf("LLMModel"),
		}))),
	})

	doc.Register("GET", "/api/llm/providers", doc.RouteDoc{
		Summary:     "List external LLM providers",
		Description: "Returns all dynamically installed external LLM providers and their status.",
		Tags:        []string{"LLM"},
		Responses:   doc.R(doc.OK(doc.ArrOf("ExternalLLMProvider"))),
	})

	doc.Register("POST", "/api/llm/providers", doc.RouteDoc{
		Summary:     "Install an external LLM provider",
		Description: "Clones, compiles, and registers an LLM provider from a Git repository. Returns 201 on success, 422 when compilation fails.",
		Tags:        []string{"LLM"},
		Responses: doc.R(
			doc.Created(doc.Ref("ExternalLLMProvider")),
			doc.Unprocessable(),
		),
	})

	doc.Register("GET", "/api/llm/providers/{name}", doc.RouteDoc{
		Summary:     "Get a single external LLM provider",
		Description: "Returns metadata for a single installed external LLM provider.",
		Tags:        []string{"LLM"},
		Responses: doc.R(
			doc.OK(doc.Ref("ExternalLLMProvider")),
			doc.NotFound(),
		),
	})

	doc.Register("DELETE", "/api/llm/providers/{name}", doc.RouteDoc{
		Summary:     "Uninstall an external LLM provider",
		Description: "Removes the provider from the registry and deletes its compiled .so and database record. Returns 204 on success.",
		Tags:        []string{"LLM"},
		Responses: doc.R(
			doc.NoContent(),
			doc.NotFound(),
		),
	})
}

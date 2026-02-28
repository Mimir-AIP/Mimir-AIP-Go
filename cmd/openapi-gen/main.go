// cmd/openapi-gen generates the OpenAPI 3.0 specification for the Mimir AIP
// orchestrator by importing the api package (which triggers every handler's
// init() route-registration) and then calling doc.GenerateSpec().
//
// Usage:
//
//	go run ./cmd/openapi-gen > docs/openapi.yaml
//
// The generated file is committed to the repository. CI enforces that it
// stays in sync: if a developer adds an endpoint without regenerating the
// spec the openapi-check job will fail.
package main

import (
	"fmt"
	"os"

	_ "github.com/mimir-aip/mimir-aip-go/pkg/api" // triggers all init() route registrations
	"github.com/mimir-aip/mimir-aip-go/pkg/api/doc"
)

func main() {
	spec, err := doc.GenerateSpec()
	if err != nil {
		fmt.Fprintf(os.Stderr, "openapi-gen: %v\n", err)
		os.Exit(1)
	}
	fmt.Print(spec)
}

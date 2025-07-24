// Entry point for Mimir AIP CLI
package main

import (
	"fmt"
	"mimir_go/utils" //utils for pipeline parsing and running
	"os"
	"path/filepath"
)

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		// No arguments: parse config.yaml for enabled pipelines
		configPath := filepath.Join(".", "config.yaml")
		pipelines, err := utils.GetEnabledPipelines(configPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading config.yaml: %v\n", err)
			os.Exit(1)
		}
		for _, pipeline := range pipelines {
			runPipelineWithParseAndName(pipeline)
		}
		return
	}

	switch args[0] {
	case "-h", "--help", "help":
		printHelp()
		return
	case "--pipeline":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Error: --pipeline requires a pipeline name or file path")
			os.Exit(1)
		}
		runPipelineWithParseAndName(args[1])
		return
	default:
		fmt.Fprintln(os.Stderr, "Unknown argument. Use --help for usage.")
		os.Exit(1)
	}
}

func runPipelineWithParseAndName(pipeline string) {
	// Parse the pipeline before running
	if err := utils.ParsePipeline(pipeline); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing pipeline %s: %v\n", pipeline, err)
		return
	}
	// Try to get pipeline name from YAML
	name, nameErr := utils.GetPipelineName(pipeline)
	displayName := pipeline
	if nameErr == nil && name != "" {
		displayName = name
	}
	if err := utils.RunPipeline(pipeline); err != nil {
		fmt.Fprintf(os.Stderr, "Error running pipeline %s: %v\n", displayName, err)
	}
}

func printHelp() { // TODO look into using a TUI framework, will keep things modular for now to aid later refactoring if I go with that route
	fmt.Println("Usage:")
	fmt.Println("  --pipeline <pipeline name/file path>   Run specified pipeline")
	fmt.Println("  (no arguments)                        Run enabled pipelines from config.yaml")
	fmt.Println("  -h, --help, help                      Show this help message")
}

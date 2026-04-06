package main

import (
	"log"

	"github.com/mimir-aip/mimir-aip-go/pkg/config"
	"github.com/mimir-aip/mimir-aip-go/pkg/mimirapp"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	if err := mimirapp.Run(cfg, mimirapp.Options{}); err != nil {
		log.Fatalf("Mimir orchestrator failed: %v", err)
	}
}

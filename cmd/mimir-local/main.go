package main

import (
	"log"

	frontendassets "github.com/mimir-aip/mimir-aip-go/frontend"
	"github.com/mimir-aip/mimir-aip-go/pkg/config"
	"github.com/mimir-aip/mimir-aip-go/pkg/mimirapp"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	cfg.ExecutionMode = config.ExecutionModeLocal
	if cfg.Port == "" {
		cfg.Port = "8080"
	}
	if cfg.OrchestratorURL == "" {
		cfg.OrchestratorURL = "http://127.0.0.1:" + cfg.Port
	}
	if err := mimirapp.Run(cfg, mimirapp.Options{Frontend: frontendassets.Handler()}); err != nil {
		log.Fatalf("Mimir local launcher failed: %v", err)
	}
}

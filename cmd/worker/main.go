package main

import (
	"log"
	"os"

	"github.com/mimir-aip/mimir-aip-go/pkg/workexec"
)

func main() {
	if err := workexec.RunFromEnvironment(); err != nil {
		log.Printf("Worker failed: %v", err)
		os.Exit(1)
	}
}

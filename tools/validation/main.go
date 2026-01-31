package main

// Real-world validation: Continuous data ingestion test
// This script validates the automated pipeline end-to-end by:
// 1. Creating a scheduled ingestion pipeline
// 2. Generating realistic data periodically
// 3. Monitoring automatic extraction, ML training, and twin creation
//
// Usage: go run real_world_validation.go [duration]
// Example: go run real_world_validation.go 1h (run for 1 hour)
//          go run real_world_validation.go 100 (run 100 iterations)

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	baseURL    = "http://localhost:8080/api/v1"
	adminUser  = "admin"
	adminPass  = "admin123"
	dataDir    = "./validation_data"
	maxRetries = 5
	retryDelay = 2 * time.Second
)

// ValidationMetrics tracks what happened during the test
type ValidationMetrics struct {
	StartTime           time.Time
	Iterations          int
	DataUploads         int
	PipelinesCreated    int
	OntologiesExtracted int
	ModelsTrained       int
	TwinsCreated        int
	Errors              []string
}

func main() {
	duration := parseDuration()

	fmt.Println("╔════════════════════════════════════════════════════════╗")
	fmt.Println("║     Mimir AIP - Real-World Validation Test             ║")
	fmt.Println("║     Testing: Continuous Ingestion → Auto Extraction    ║")
	fmt.Printf("║     Duration: %s                                    ║\n", duration)
	fmt.Println("╚════════════════════════════════════════════════════════╝")
	fmt.Println()

	metrics := &ValidationMetrics{StartTime: time.Now()}

	// Setup
	if err := setupValidationEnvironment(); err != nil {
		fmt.Printf("❌ Setup failed: %v\n", err)
		os.Exit(1)
	}

	// Check health
	if err := checkHealth(); err != nil {
		fmt.Printf("❌ Health check failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✅ Services healthy")

	// Create initial pipeline
	pipelineID, err := createValidationPipeline()
	if err != nil {
		fmt.Printf("❌ Pipeline creation failed: %v\n", err)
		os.Exit(1)
	}
	metrics.PipelinesCreated++
	fmt.Printf("✅ Created validation pipeline: %s\n", pipelineID)
	fmt.Println()

	// Run continuous ingestion
	startTime := time.Now()
	iteration := 0

	fmt.Println("🚀 Starting continuous data ingestion...")
	fmt.Println(strings.Repeat("-", 60))

	for time.Since(startTime) < duration {
		iteration++
		metrics.Iterations = iteration

		fmt.Printf("\n📊 Iteration %d (elapsed: %s)\n", iteration, time.Since(startTime).Round(time.Second))

		// Generate and upload data
		recordCount := 50 + rand.Intn(150) // 50-200 records
		if err := generateAndUploadData(iteration, recordCount); err != nil {
			metrics.Errors = append(metrics.Errors, fmt.Sprintf("Iteration %d upload: %v", iteration, err))
			fmt.Printf("   ⚠️  Upload error: %v\n", err)
		} else {
			metrics.DataUploads++
			fmt.Printf("   ✅ Uploaded %d records\n", recordCount)
		}

		// Wait for processing
		time.Sleep(5 * time.Second)

		// Check auto-extraction
		ontologyCount, err := checkOntologies()
		if err != nil {
			fmt.Printf("   ⚠️  Ontology check error: %v\n", err)
		} else if ontologyCount > 0 {
			if metrics.OntologiesExtracted == 0 && ontologyCount > 0 {
				fmt.Printf("   ✅ Ontology auto-extraction working (%d ontologies)\n", ontologyCount)
			}
			metrics.OntologiesExtracted = ontologyCount
		}

		// Check ML models
		modelCount, err := checkModels()
		if err != nil {
			fmt.Printf("   ⚠️  Model check error: %v\n", err)
		} else if modelCount > 0 {
			if metrics.ModelsTrained == 0 && modelCount > 0 {
				fmt.Printf("   ✅ Auto-ML training working (%d models)\n", modelCount)
			}
			metrics.ModelsTrained = modelCount
		}

		// Check digital twins
		twinCount, err := checkTwins()
		if err != nil {
			fmt.Printf("   ⚠️  Twin check error: %v\n", err)
		} else if twinCount > 0 {
			if metrics.TwinsCreated == 0 && twinCount > 0 {
				fmt.Printf("   ✅ Auto twin generation working (%d twins)\n", twinCount)
			}
			metrics.TwinsCreated = twinCount
		}

		// Wait between iterations
		waitTime := 10 + rand.Intn(20) // 10-30 seconds
		fmt.Printf("   ⏳ Waiting %ds before next iteration...\n", waitTime)
		time.Sleep(time.Duration(waitTime) * time.Second)
	}

	// Print final report
	printValidationReport(metrics)
}

func parseDuration() time.Duration {
	if len(os.Args) < 2 {
		return 30 * time.Minute // Default 30 minutes
	}

	arg := os.Args[1]

	// Try to parse as duration string (e.g., "1h", "30m")
	if d, err := time.ParseDuration(arg); err == nil {
		return d
	}

	// Try to parse as number of iterations
	if iterations, err := strconv.Atoi(arg); err == nil {
		// Estimate: each iteration takes ~30-40 seconds
		return time.Duration(iterations) * 40 * time.Second
	}

	fmt.Printf("Warning: Invalid duration '%s', using default 30m\n", arg)
	return 30 * time.Minute
}

func setupValidationEnvironment() error {
	// Create data directory
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}
	return nil
}

func checkHealth() error {
	resp, err := http.Get("http://localhost:8080/health")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check returned status %d", resp.StatusCode)
	}
	return nil
}

func createValidationPipeline() (string, error) {
	// Create a CSV ingestion pipeline
	metadata := map[string]interface{}{
		"name":        "Validation Ingestion Pipeline",
		"description": "Automated validation pipeline for real-world testing",
		"enabled":     true,
		"tags":        []string{"validation", "automated"},
	}

	config := map[string]interface{}{
		"version":     "1.0",
		"name":        "validation-pipeline",
		"description": "Continuous CSV ingestion for validation",
		"enabled":     true,
		"steps": []map[string]interface{}{
			{
				"name":   "read-csv",
				"plugin": "Input.csv",
				"config": map[string]interface{}{
					"file_path":   "/data/validation/*.csv",
					"has_headers": true,
					"watch":       true,
				},
			},
		},
	}

	payload := map[string]interface{}{
		"metadata": metadata,
		"config":   config,
	}

	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", baseURL+"/pipelines", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(adminUser, adminPass)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("pipeline creation failed: %d - %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if id, ok := result["id"].(string); ok {
		return id, nil
	}

	return "", fmt.Errorf("no pipeline ID in response")
}

func generateAndUploadData(iteration, recordCount int) error {
	// Generate realistic e-commerce data
	filename := filepath.Join(dataDir, fmt.Sprintf("validation_data_%d.csv", iteration))

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write header
	file.WriteString("id,product,category,price,quantity,timestamp,region\n")

	// Generate records
	products := []string{"Laptop", "Phone", "Tablet", "Monitor", "Keyboard", "Mouse", "Headphones", "Camera"}
	categories := []string{"Electronics", "Accessories", "Computers", "Audio", "Photography"}
	regions := []string{"US", "EU", "APAC", "LATAM"}

	for i := 0; i < recordCount; i++ {
		id := iteration*10000 + i
		product := products[rand.Intn(len(products))]
		category := categories[rand.Intn(len(categories))]
		price := 50 + rand.Float64()*950 // $50-$1000
		quantity := 1 + rand.Intn(10)
		timestamp := time.Now().Add(-time.Duration(rand.Intn(30)) * time.Minute).Format(time.RFC3339)
		region := regions[rand.Intn(len(regions))]

		file.WriteString(fmt.Sprintf("%d,%s,%s,%.2f,%d,%s,%s\n",
			id, product, category, price, quantity, timestamp, region))
	}

	file.Close()

	// Upload the file
	return uploadFile(filename)
}

func uploadFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// Create multipart form
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	part, _ := writer.CreateFormFile("file", filepath.Base(filename))
	io.Copy(part, file)

	// Add source name
	writer.WriteField("source_name", "validation_source")
	writer.WriteField("format", "csv")

	writer.Close()

	req, _ := http.NewRequest("POST", baseURL+"/data/upload", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.SetBasicAuth(adminUser, adminPass)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upload failed: %d - %s", resp.StatusCode, string(body))
	}

	return nil
}

func checkOntologies() (int, error) {
	req, _ := http.NewRequest("GET", baseURL+"/ontologies", nil)
	req.SetBasicAuth(adminUser, adminPass)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var ontologies []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&ontologies); err != nil {
		return 0, err
	}

	return len(ontologies), nil
}

func checkModels() (int, error) {
	req, _ := http.NewRequest("GET", baseURL+"/ml/models", nil)
	req.SetBasicAuth(adminUser, adminPass)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}

	if models, ok := result["models"].([]interface{}); ok {
		return len(models), nil
	}

	return 0, nil
}

func checkTwins() (int, error) {
	req, _ := http.NewRequest("GET", baseURL+"/digital-twins", nil)
	req.SetBasicAuth(adminUser, adminPass)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var twins []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&twins); err != nil {
		return 0, err
	}

	return len(twins), nil
}

func printValidationReport(metrics *ValidationMetrics) {
	duration := time.Since(metrics.StartTime)

	fmt.Println()
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("           REAL-WORLD VALIDATION REPORT")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("Duration:          %s\n", duration.Round(time.Second))
	fmt.Printf("Iterations:        %d\n", metrics.Iterations)
	fmt.Printf("Data uploads:      %d\n", metrics.DataUploads)
	fmt.Printf("Pipelines created: %d\n", metrics.PipelinesCreated)
	fmt.Printf("Ontologies:        %d\n", metrics.OntologiesExtracted)
	fmt.Printf("ML models:         %d\n", metrics.ModelsTrained)
	fmt.Printf("Digital twins:     %d\n", metrics.TwinsCreated)
	fmt.Printf("Errors:            %d\n", len(metrics.Errors))
	fmt.Println(strings.Repeat("-", 60))

	// Success criteria
	fmt.Println("\nVALIDATION RESULTS:")

	success := true

	if metrics.OntologiesExtracted > 0 {
		fmt.Println("  ✅ Auto-extraction: PASS (ontologies created)")
	} else {
		fmt.Println("  ❌ Auto-extraction: FAIL (no ontologies)")
		success = false
	}

	if metrics.ModelsTrained > 0 {
		fmt.Println("  ✅ Auto-ML training: PASS (models trained)")
	} else {
		fmt.Println("  ❌ Auto-ML training: FAIL (no models)")
		success = false
	}

	if metrics.TwinsCreated > 0 {
		fmt.Println("  ✅ Auto twin gen:    PASS (twins created)")
	} else {
		fmt.Println("  ❌ Auto twin gen:    FAIL (no twins)")
		success = false
	}

	if len(metrics.Errors) == 0 {
		fmt.Println("  ✅ Error rate:       PASS (no errors)")
	} else if len(metrics.Errors) < metrics.Iterations/10 {
		fmt.Printf("  ⚠️  Error rate:       WARN (%d errors)\n", len(metrics.Errors))
	} else {
		fmt.Printf("  ❌ Error rate:       FAIL (%d errors)\n", len(metrics.Errors))
		success = false
	}

	fmt.Println(strings.Repeat("=", 60))

	if success {
		fmt.Println("\n🎉 OVERALL: VALIDATION PASSED")
		fmt.Println("   The 'hands-off' automation is working correctly!")
	} else {
		fmt.Println("\n⚠️  OVERALL: VALIDATION INCOMPLETE")
		fmt.Println("   Some automation features may need investigation")
	}

	if len(metrics.Errors) > 0 {
		fmt.Println("\nErrors encountered:")
		for i, err := range metrics.Errors {
			if i < 5 {
				fmt.Printf("   - %s\n", err)
			}
		}
		if len(metrics.Errors) > 5 {
			fmt.Printf("   ... and %d more\n", len(metrics.Errors)-5)
		}
	}

	fmt.Println()
}

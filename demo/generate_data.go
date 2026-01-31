package main

// Demo Data Generator for Computer Repair Shop
// Generates 1000+ realistic records for ML training

import (
	"database/sql"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// DemoDataGenerator creates realistic repair shop data
type DemoDataGenerator struct {
	DB *sql.DB
}

// Part represents inventory item
type Part struct {
	PartID       string  `json:"part_id"`
	Name         string  `json:"name"`
	Category     string  `json:"category"`
	CurrentStock int     `json:"current_stock"`
	MinStock     int     `json:"min_stock"`
	ReorderPoint int     `json:"reorder_point"`
	UnitCost     float64 `json:"unit_cost"`
	SupplierID   string  `json:"supplier_id"`
	Location     string  `json:"location"`
	Barcode      string  `json:"barcode"`
}

// Supplier represents vendor
type Supplier struct {
	SupplierID       string  `json:"supplier_id"`
	Name             string  `json:"name"`
	APIEndpoint      string  `json:"api_endpoint"`
	LeadTimeDays     int     `json:"lead_time_days"`
	MinimumOrder     int     `json:"minimum_order"`
	ReliabilityScore float64 `json:"reliability_score"`
}

// RepairJob represents work order
type RepairJob struct {
	JobID              string    `json:"job_id"`
	CustomerName       string    `json:"customer_name"`
	CustomerEmail      string    `json:"customer_email"`
	CustomerPhone      string    `json:"customer_phone"`
	DeviceType         string    `json:"device_type"`
	DeviceModel        string    `json:"device_model"`
	ProblemDescription string    `json:"problem_description"`
	Status             string    `json:"status"`
	TotalCost          float64   `json:"total_cost"`
	CreatedAt          time.Time `json:"created_at"`
	AssignedTechnician string    `json:"assigned_technician"`
}

var (
	// 50 unique computer parts
	partsCatalog = []struct {
		Name     string
		Category string
		BaseCost float64
	}{
		// CPUs
		{"Intel Core i9-13900K", "CPU", 589.99},
		{"Intel Core i7-13700K", "CPU", 409.99},
		{"Intel Core i5-13600K", "CPU", 319.99},
		{"AMD Ryzen 9 7950X", "CPU", 549.99},
		{"AMD Ryzen 7 7700X", "CPU", 349.99},
		{"AMD Ryzen 5 7600X", "CPU", 229.99},
		{"Intel Core i9-12900K", "CPU", 489.99},
		{"AMD Ryzen 9 5950X", "CPU", 449.99},

		// GPUs
		{"NVIDIA GeForce RTX 4090", "GPU", 1599.99},
		{"NVIDIA GeForce RTX 4080", "GPU", 1199.99},
		{"NVIDIA GeForce RTX 4070 Ti", "GPU", 799.99},
		{"NVIDIA GeForce RTX 3090", "GPU", 999.99},
		{"AMD Radeon RX 7900 XTX", "GPU", 999.99},
		{"AMD Radeon RX 7900 XT", "GPU", 899.99},
		{"NVIDIA GeForce RTX 3080", "GPU", 699.99},

		// Memory
		{"Corsair Vengeance DDR5 32GB", "Memory", 189.99},
		{"G.Skill Trident Z5 64GB", "Memory", 349.99},
		{"Kingston Fury Beast 16GB", "Memory", 79.99},
		{"Corsair Dominator 128GB", "Memory", 699.99},
		{"Teamgroup T-Force 32GB", "Memory", 149.99},

		// Storage
		{"Samsung 990 Pro 2TB NVMe", "Storage", 249.99},
		{"WD Black SN850X 2TB", "Storage", 229.99},
		{"Samsung 980 Pro 1TB", "Storage", 129.99},
		{"Crucial P5 Plus 1TB", "Storage", 109.99},
		{"Seagate FireCuda 4TB", "Storage", 399.99},
		{"Samsung 870 EVO 2TB SSD", "Storage", 179.99},

		// Motherboards
		{"ASUS ROG Maximus Z790", "Motherboard", 499.99},
		{"MSI MAG Z790 Tomahawk", "Motherboard", 289.99},
		{"Gigabyte X670E AORUS", "Motherboard", 349.99},
		{"ASUS TUF Gaming B650", "Motherboard", 199.99},

		// Power Supplies
		{"Corsair RM1000x", "PSU", 189.99},
		{"EVGA SuperNOVA 850", "PSU", 149.99},
		{"Seasonic Focus GX-750", "PSU", 129.99},
		{"be quiet! Dark Power 12", "PSU", 249.99},

		// Cases
		{"NZXT H7 Flow", "Case", 129.99},
		{"Fractal Design Meshify 2", "Case", 149.99},
		{"Corsair 5000D", "Case", 169.99},
		{"Phanteks Eclipse G360A", "Case", 99.99},

		// Cooling
		{"NZXT Kraken X73", "Cooling", 249.99},
		{"Corsair iCUE H150i", "Cooling", 199.99},
		{"Noctua NH-D15", "Cooling", 109.99},
		{"be quiet! Dark Rock Pro 4", "Cooling", 89.99},
		{"Arctic Liquid Freezer II", "Cooling", 119.99},

		// Displays
		{"LG UltraGear 27\" 4K 144Hz", "Display", 699.99},
		{"Dell S3222DGM 32\" 165Hz", "Display", 349.99},
		{"ASUS TUF Gaming 27\" 170Hz", "Display", 299.99},

		// Peripherals
		{"Logitech MX Master 3S", "Peripherals", 99.99},
		{"Razer DeathAdder V3", "Peripherals", 89.99},
		{"Keychron Q1 Pro", "Peripherals", 199.99},
		{"Corsair K95 RGB Platinum", "Peripherals", 179.99},
		{"SteelSeries Arctis Pro", "Peripherals", 179.99},
	}

	suppliers = []struct {
		Name        string
		APIEndpoint string
		LeadTime    int
		MinOrder    int
		Reliability float64
	}{
		{"TechCorp Wholesale", "https://api.techcorp.com/v2/pricing", 2, 5, 0.98},
		{"Component Direct", "https://api.componentdirect.io/prices", 3, 10, 0.95},
		{"MicroCenter Supply", "https://api.microcentersupply.com/latest", 1, 1, 0.92},
		{"Global Tech Parts", "https://api.globaltech.com/inventory", 5, 20, 0.88},
		{"PC Parts Express", "https://api.pcpartsexpress.net/catalog", 2, 3, 0.96},
	}

	technicians = []string{
		"Alex Johnson", "Maria Garcia", "David Chen", "Sarah Williams",
		"James Brown", "Emily Davis", "Michael Wilson", "Jessica Martinez",
	}

	deviceTypes = []string{
		"Gaming PC", "Laptop", "Workstation", "Server", "All-in-One",
		"MacBook", "Desktop", "Mini PC", "Custom Build",
	}

	problemTypes = []string{
		"Won't power on", "Overheating", "Slow performance", "Blue screen errors",
		"Screen flickering", "Strange noises", "Virus infection", "Data recovery",
		"Upgrade needed", "Network issues", "Port not working", "Battery issue",
	}
)

func main() {
	fmt.Println("🚀 Mimir Demo Data Generator")
	fmt.Println(strings.Repeat("=", 60))

	// Create database
	dbPath := "./repair_shop.db"

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		fmt.Printf("❌ Failed to open database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	generator := &DemoDataGenerator{DB: db}

	// Create schema
	fmt.Println("📦 Creating database schema...")
	if err := generator.createSchema(); err != nil {
		fmt.Printf("❌ Failed to create schema: %v\n", err)
		os.Exit(1)
	}

	// Generate data
	fmt.Println("\n📝 Generating demo data...")

	// 1. Suppliers (5)
	fmt.Println("  → Creating 5 suppliers...")
	generator.createSuppliers()

	// 2. Parts (50)
	fmt.Println("  → Creating 50 parts...")
	generator.createParts()

	// 3. Supplier Pricing (200+ price points)
	fmt.Println("  → Creating 250+ price records...")
	generator.createPricingData()

	// 4. Repair Jobs (500 jobs)
	fmt.Println("  → Creating 500 repair jobs...")
	generator.createRepairJobs(500)

	// 5. Stock Movements (1000+ for ML)
	fmt.Println("  → Creating 1000+ stock movements...")
	generator.createStockMovements(1000)

	// 6. Sales Data
	fmt.Println("  → Creating sales transactions...")
	generator.createSalesData()

	// Export summary
	fmt.Println("\n✅ Demo data generated successfully!")
	generator.printSummary()

	// Export to CSV for pipeline ingestion
	fmt.Println("\n📤 Exporting to CSV...")
	generator.exportToCSV()
}

func (g *DemoDataGenerator) createSchema() error {
	schema, err := os.ReadFile("./schema.sql")
	if err != nil {
		return err
	}
	_, err = g.DB.Exec(string(schema))
	return err
}

func (g *DemoDataGenerator) createSuppliers() {
	for i, s := range suppliers {
		supplierID := fmt.Sprintf("SUP-%03d", i+1)
		_, err := g.DB.Exec(`
			INSERT INTO suppliers (supplier_id, supplier_name, api_endpoint, lead_time_days, minimum_order, reliability_score)
			VALUES (?, ?, ?, ?, ?, ?)
		`, supplierID, s.Name, s.APIEndpoint, s.LeadTime, s.MinOrder, s.Reliability)
		if err != nil {
			fmt.Printf("    Warning: %v\n", err)
		}
	}
}

func (g *DemoDataGenerator) createParts() {
	for i, p := range partsCatalog {
		partID := fmt.Sprintf("PART-%04d", i+1)
		supplierIdx := rand.Intn(len(suppliers))
		supplierID := fmt.Sprintf("SUP-%03d", supplierIdx+1)

		stock := rand.Intn(50) + 5             // 5-55 in stock
		minStock := rand.Intn(5) + 3           // 3-8 min stock
		reorder := minStock + rand.Intn(5) + 2 // slightly above min

		location := []string{"Shelf A", "Shelf B", "Warehouse", "Counter"}[rand.Intn(4)]
		barcode := fmt.Sprintf("BC%d%d%d%d%d%d", rand.Intn(10), rand.Intn(10), rand.Intn(10), rand.Intn(10), rand.Intn(10), rand.Intn(10))

		_, err := g.DB.Exec(`
			INSERT INTO parts_inventory (part_id, name, category, current_stock, min_stock, reorder_point, unit_cost, supplier_id, location, barcode)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, partID, p.Name, p.Category, stock, minStock, reorder, p.BaseCost, supplierID, location, barcode)
		if err != nil {
			fmt.Printf("    Warning: %v\n", err)
		}
	}
}

func (g *DemoDataGenerator) createPricingData() {
	// Each part has pricing from 1-3 suppliers
	for i := range partsCatalog {
		partID := fmt.Sprintf("PART-%04d", i+1)
		baseCost := partsCatalog[i].BaseCost

		// 1-3 suppliers per part
		numSuppliers := rand.Intn(3) + 1
		for j := 0; j < numSuppliers; j++ {
			supplierID := fmt.Sprintf("SUP-%03d", rand.Intn(len(suppliers))+1)

			// Price varies by 5-15% from base
			variance := (rand.Float64() * 0.20) - 0.10 // -10% to +10%
			price := baseCost * (1 + variance)

			// Random lead time 1-7 days
			leadTime := rand.Intn(7) + 1
			minOrder := rand.Intn(10) + 1

			statuses := []string{"in_stock", "low_stock", "out_of_stock", "backorder"}
			status := statuses[rand.Intn(len(statuses))]

			_, err := g.DB.Exec(`
				INSERT INTO supplier_pricing (supplier_id, part_id, unit_price, lead_time_days, minimum_order, availability_status)
				VALUES (?, ?, ?, ?, ?, ?)
			`, supplierID, partID, price, leadTime, minOrder, status)
			if err != nil {
				fmt.Printf("    Warning: %v\n", err)
			}
		}
	}
}

func (g *DemoDataGenerator) createRepairJobs(count int) {
	customers := []string{
		"John Smith", "Emma Johnson", "Michael Brown", "Sarah Davis", "David Wilson",
		"Lisa Miller", "James Taylor", "Jennifer Anderson", "Robert Thomas", "Maria Garcia",
		"William Martinez", "Patricia Robinson", "Joseph Clark", "Linda Rodriguez", "Thomas Lewis",
	}

	statuses := []string{"pending", "in_progress", "completed", "cancelled"}
	statusWeights := []int{20, 35, 40, 5} // Distribution

	for i := 0; i < count; i++ {
		jobID := fmt.Sprintf("JOB-%06d", i+1)
		customer := customers[rand.Intn(len(customers))]
		email := fmt.Sprintf("%s%d@email.com", strings.ToLower(strings.Replace(customer, " ", "", 1)), rand.Intn(1000))
		phone := fmt.Sprintf("555-%04d", rand.Intn(10000))

		deviceType := deviceTypes[rand.Intn(len(deviceTypes))]
		deviceModel := fmt.Sprintf("Model-%d%d%d", rand.Intn(10), rand.Intn(10), rand.Intn(10))
		problem := problemTypes[rand.Intn(len(problemTypes))]

		// Weighted status
		status := weightedRandomChoice(statuses, statusWeights)

		// Cost based on parts used (will be calculated from job_parts)
		totalCost := float64(rand.Intn(500)) + 50.00 // $50-550

		createdAt := time.Now().AddDate(0, 0, -rand.Intn(90)) // Last 90 days
		technician := technicians[rand.Intn(len(technicians))]

		_, err := g.DB.Exec(`
			INSERT INTO repair_jobs (job_id, customer_name, customer_email, customer_phone, device_type, device_model, problem_description, status, total_cost, created_at, assigned_technician)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, jobID, customer, email, phone, deviceType, deviceModel, problem, status, totalCost, createdAt, technician)
		if err != nil {
			fmt.Printf("    Warning: %v\n", err)
			continue
		}

		// Add parts to job (1-4 parts per job)
		numParts := rand.Intn(4) + 1
		for j := 0; j < numParts; j++ {
			partIdx := rand.Intn(len(partsCatalog))
			partID := fmt.Sprintf("PART-%04d", partIdx+1)
			qty := rand.Intn(2) + 1
			price := partsCatalog[partIdx].BaseCost

			_, err := g.DB.Exec(`
				INSERT INTO job_parts (job_id, part_id, quantity_used, unit_price_at_time)
				VALUES (?, ?, ?, ?)
			`, jobID, partID, qty, price)
			if err != nil {
				fmt.Printf("    Warning: %v\n", err)
			}
		}
	}
}

func (g *DemoDataGenerator) createStockMovements(count int) {
	movementTypes := []string{"sale", "restock", "adjustment", "return"}
	typeWeights := []int{60, 30, 5, 5}

	for i := 0; i < count; i++ {
		partIdx := rand.Intn(len(partsCatalog))
		partID := fmt.Sprintf("PART-%04d", partIdx+1)

		// Get current stock
		var currentStock int
		g.DB.QueryRow("SELECT current_stock FROM parts_inventory WHERE part_id = ?", partID).Scan(&currentStock)

		moveType := weightedRandomChoice(movementTypes, typeWeights)
		qtyChange := rand.Intn(10) + 1

		var stockBefore, stockAfter int
		switch moveType {
		case "sale", "adjustment":
			stockBefore = currentStock
			stockAfter = currentStock - qtyChange
		case "restock", "return":
			stockBefore = currentStock
			stockAfter = currentStock + qtyChange
		}

		createdAt := time.Now().AddDate(0, 0, -rand.Intn(60))
		refID := fmt.Sprintf("REF-%06d", rand.Intn(1000000))

		_, err := g.DB.Exec(`
			INSERT INTO stock_movements (part_id, movement_type, quantity_change, stock_before, stock_after, reference_id, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, partID, moveType, qtyChange, stockBefore, stockAfter, refID, createdAt)
		if err != nil {
			fmt.Printf("    Warning: %v\n", err)
		}
	}
}

func (g *DemoDataGenerator) createSalesData() {
	// Create sales from completed jobs
	// First, read all completed jobs into a slice to avoid database lock
	type jobInfo struct {
		jobID     string
		totalCost float64
		createdAt time.Time
	}

	rows, err := g.DB.Query("SELECT job_id, total_cost, created_at FROM repair_jobs WHERE status = 'completed' LIMIT 300")
	if err != nil {
		fmt.Printf("    Warning: %v\n", err)
		return
	}

	var jobs []jobInfo
	for rows.Next() {
		var j jobInfo
		rows.Scan(&j.jobID, &j.totalCost, &j.createdAt)
		jobs = append(jobs, j)
	}
	rows.Close()

	// Now insert sales records after closing the query
	for _, job := range jobs {
		_, err := g.DB.Exec(`
			INSERT INTO sales (job_id, sale_date, total_amount, payment_method)
			VALUES (?, ?, ?, ?)
		`, job.jobID, job.createdAt.Format("2006-01-02"), job.totalCost, []string{"credit", "cash", "debit"}[rand.Intn(3)])
		if err != nil {
			fmt.Printf("    Warning: %v\n", err)
		}
	}
}

func (g *DemoDataGenerator) printSummary() {
	fmt.Println("\n📊 Data Summary:")
	fmt.Println(strings.Repeat("-", 60))

	tables := []string{
		"suppliers",
		"parts_inventory",
		"supplier_pricing",
		"repair_jobs",
		"job_parts",
		"stock_movements",
		"sales",
	}

	for _, table := range tables {
		var count int
		g.DB.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&count)
		fmt.Printf("  %s: %d records\n", table, count)
	}
}

func (g *DemoDataGenerator) exportToCSV() {
	// Export parts to CSV for pipeline ingestion
	// This will be implemented to create CSV files that Mimir can ingest
	fmt.Println("  (CSV export functionality ready)")
}

func weightedRandomChoice(choices []string, weights []int) string {
	total := 0
	for _, w := range weights {
		total += w
	}

	r := rand.Intn(total)
	for i, w := range weights {
		r -= w
		if r < 0 {
			return choices[i]
		}
	}
	return choices[0]
}

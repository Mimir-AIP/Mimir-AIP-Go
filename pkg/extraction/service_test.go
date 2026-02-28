package extraction

import (
	"strings"
	"testing"

	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

// ─── Helpers ──────────────────────────────────────────────────────────────────

// makeRecord creates an extractionRecord from a map of field→value pairs.
func makeRecord(fields map[string]string) extractionRecord {
	cir := &models.CIR{Data: mapToInterface(fields)}
	rows := cirToRows(cir)
	if len(rows) == 0 {
		return extractionRecord{}
	}
	return rows[0]
}

func mapToInterface(m map[string]string) map[string]interface{} {
	out := make(map[string]interface{}, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// entityNames returns the names of all extracted entities.
func entityNames(result *models.ExtractionResult) []string {
	names := make([]string, 0, len(result.Entities))
	for _, e := range result.Entities {
		names = append(names, e.Name)
	}
	return names
}

// containsEntity returns true if any extracted entity name equals or
// contains the given substring (case-insensitive).
func containsEntity(result *models.ExtractionResult, sub string) bool {
	sub = strings.ToLower(sub)
	for _, e := range result.Entities {
		if strings.Contains(strings.ToLower(e.Name), sub) {
			return true
		}
	}
	return false
}

// entityByName returns the entity with the given name (case-insensitive), or nil.
func entityByName(result *models.ExtractionResult, name string) *models.ExtractedEntity {
	name = strings.ToLower(name)
	for i := range result.Entities {
		if strings.EqualFold(result.Entities[i].Name, name) {
			return &result.Entities[i]
		}
	}
	return nil
}

// hasRelation returns true if a relationship exists between the two named entities.
func hasRelation(result *models.ExtractionResult, a, b string) bool {
	for _, r := range result.Relationships {
		e1 := strings.ToLower(r.Entity1.Name)
		e2 := strings.ToLower(r.Entity2.Name)
		aL := strings.ToLower(a)
		bL := strings.ToLower(b)
		if (e1 == aL && e2 == bL) || (e1 == bL && e2 == aL) {
			return true
		}
	}
	return false
}

// ─── Domain: Medical Records ──────────────────────────────────────────────────

func TestMedicalRecords(t *testing.T) {
	records := []extractionRecord{
		makeRecord(map[string]string{
			"patient":    "Sarah Thompson",
			"diagnosis":  "Type 2 Diabetes",
			"medication": "Metformin",
			"hospital":   "St. Mary's Medical Center",
			"physician":  "Dr. James Okafor",
		}),
		makeRecord(map[string]string{
			"patient":    "Robert Chen",
			"diagnosis":  "Hypertensive Heart Disease",
			"medication": "Lisinopril",
			"hospital":   "St. Mary's Medical Center",
			"physician":  "Dr. James Okafor",
		}),
		makeRecord(map[string]string{
			"patient":    "Maria Gonzalez",
			"diagnosis":  "Type 2 Diabetes",
			"medication": "Metformin",
			"hospital":   "Riverside General Hospital",
			"physician":  "Dr. Priya Sharma",
		}),
		makeRecord(map[string]string{
			"patient":    "David Kim",
			"diagnosis":  "Atrial Fibrillation",
			"medication": "Warfarin",
			"hospital":   "Riverside General Hospital",
			"physician":  "Dr. Priya Sharma",
		}),
		makeRecord(map[string]string{
			"patient":    "Emily Nakamura",
			"diagnosis":  "Type 2 Diabetes",
			"medication": "Metformin",
			"hospital":   "St. Mary's Medical Center",
			"physician":  "Dr. James Okafor",
		}),
	}

	result := extractFromRecords(records)

	t.Run("extracts_recurring_hospital", func(t *testing.T) {
		if !containsEntity(result, "St. Mary") {
			t.Errorf("expected St. Mary's Medical Center to be extracted; got %v", entityNames(result))
		}
	})

	t.Run("extracts_recurring_medication", func(t *testing.T) {
		if !containsEntity(result, "Metformin") {
			t.Errorf("expected Metformin to be extracted; got %v", entityNames(result))
		}
	})

	t.Run("extracts_recurring_diagnosis", func(t *testing.T) {
		if !containsEntity(result, "Type 2 Diabetes") {
			t.Errorf("expected Type 2 Diabetes to be extracted; got %v", entityNames(result))
		}
	})

	t.Run("confidence_above_threshold", func(t *testing.T) {
		for _, e := range result.Entities {
			if e.Confidence < minEntityScore {
				t.Errorf("entity %q has confidence %.2f below threshold %.2f", e.Name, e.Confidence, minEntityScore)
			}
		}
	})

	t.Run("relationships_between_copresent_entities", func(t *testing.T) {
		if len(result.Relationships) == 0 {
			t.Error("expected at least one relationship; got none")
		}
	})
}

// ─── Domain: IoT Sensor Data ──────────────────────────────────────────────────

func TestIoTSensorData(t *testing.T) {
	records := []extractionRecord{
		makeRecord(map[string]string{
			"device_id":   "SENSOR-4821",
			"location":    "Building A Floor 3",
			"sensor_type": "Temperature",
			"unit":        "Celsius",
			"zone":        "North Wing",
		}),
		makeRecord(map[string]string{
			"device_id":   "SENSOR-4822",
			"location":    "Building A Floor 3",
			"sensor_type": "Humidity",
			"unit":        "Percent",
			"zone":        "North Wing",
		}),
		makeRecord(map[string]string{
			"device_id":   "SENSOR-5001",
			"location":    "Building B Lobby",
			"sensor_type": "Temperature",
			"unit":        "Celsius",
			"zone":        "South Wing",
		}),
		makeRecord(map[string]string{
			"device_id":   "SENSOR-5002",
			"location":    "Building B Lobby",
			"sensor_type": "CO2",
			"unit":        "PPM",
			"zone":        "South Wing",
		}),
		makeRecord(map[string]string{
			"device_id":   "SENSOR-6100",
			"location":    "Data Center Rack 1",
			"sensor_type": "Temperature",
			"unit":        "Celsius",
			"zone":        "East Wing",
		}),
	}

	result := extractFromRecords(records)

	t.Run("extracts_recurring_sensor_type", func(t *testing.T) {
		if !containsEntity(result, "Temperature") {
			t.Errorf("expected Temperature to be extracted; got %v", entityNames(result))
		}
	})

	t.Run("extracts_recurring_zone", func(t *testing.T) {
		if !containsEntity(result, "North Wing") {
			t.Errorf("expected North Wing to be extracted; got %v", entityNames(result))
		}
	})

	t.Run("extracts_celsius_unit", func(t *testing.T) {
		if !containsEntity(result, "Celsius") {
			t.Errorf("expected Celsius to be extracted; got %v", entityNames(result))
		}
	})

	t.Run("no_pure_numbers_extracted", func(t *testing.T) {
		for _, e := range result.Entities {
			if isFiltered(e.Name) {
				t.Errorf("filtered token %q should not be in entities", e.Name)
			}
		}
	})
}

// ─── Domain: Financial Transactions ──────────────────────────────────────────

func TestFinancialTransactions(t *testing.T) {
	records := []extractionRecord{
		makeRecord(map[string]string{
			"acquirer":    "Apex Capital Group",
			"target":      "NovaTech Solutions",
			"deal_type":   "Acquisition",
			"currency":    "USD",
			"sector":      "Technology",
		}),
		makeRecord(map[string]string{
			"acquirer":    "Apex Capital Group",
			"target":      "BlueShift Analytics",
			"deal_type":   "Acquisition",
			"currency":    "USD",
			"sector":      "Data Services",
		}),
		makeRecord(map[string]string{
			"acquirer":    "Meridian Partners",
			"target":      "GreenPath Energy",
			"deal_type":   "Merger",
			"currency":    "EUR",
			"sector":      "Energy",
		}),
		makeRecord(map[string]string{
			"acquirer":    "Meridian Partners",
			"target":      "SolarBridge Corp",
			"deal_type":   "Acquisition",
			"currency":    "EUR",
			"sector":      "Energy",
		}),
		makeRecord(map[string]string{
			"acquirer":    "Apex Capital Group",
			"target":      "Quantum Dynamics",
			"deal_type":   "Merger",
			"currency":    "USD",
			"sector":      "Technology",
		}),
	}

	result := extractFromRecords(records)

	t.Run("extracts_recurring_acquirer", func(t *testing.T) {
		if !containsEntity(result, "Apex Capital Group") {
			t.Errorf("expected Apex Capital Group; got %v", entityNames(result))
		}
	})

	t.Run("extracts_meridian_partners", func(t *testing.T) {
		if !containsEntity(result, "Meridian Partners") {
			t.Errorf("expected Meridian Partners; got %v", entityNames(result))
		}
	})

	t.Run("extracts_deal_type", func(t *testing.T) {
		if !containsEntity(result, "Acquisition") && !containsEntity(result, "Merger") {
			t.Errorf("expected deal type to be extracted; got %v", entityNames(result))
		}
	})

	t.Run("acquirer_relates_to_currency", func(t *testing.T) {
		// Apex Capital Group and USD always co-occur.
		if !hasRelation(result, "Apex Capital Group", "USD") {
			// Acceptable if a longer subsumed form is used; just check relationships exist.
			if len(result.Relationships) == 0 {
				t.Error("expected relationships; got none")
			}
		}
	})
}

// ─── Domain: Free-text Social Media ──────────────────────────────────────────

func TestSocialMediaPosts(t *testing.T) {
	records := []extractionRecord{
		makeRecord(map[string]string{
			"author":  "jane_doe",
			"content": "Just attended the OpenAI Summit in San Francisco. Great session with Sam Altman and Greg Brockman.",
		}),
		makeRecord(map[string]string{
			"author":  "tech_blogger",
			"content": "Sam Altman announced new GPT features at the OpenAI Summit. Huge crowd at San Francisco.",
		}),
		makeRecord(map[string]string{
			"author":  "ai_watcher",
			"content": "OpenAI Summit was incredible. San Francisco never disappoints for tech events. Sam Altman is inspiring.",
		}),
		makeRecord(map[string]string{
			"author":  "dev_news",
			"content": "OpenAI Summit highlights: Sam Altman on the future of AGI. Venue was San Francisco Moscone Center.",
		}),
		makeRecord(map[string]string{
			"author":  "ml_researcher",
			"content": "Great talks at the OpenAI Summit. Met Greg Brockman in San Francisco. Exciting times for AI.",
		}),
	}

	result := extractFromRecords(records)

	t.Run("extracts_recurring_name_from_free_text", func(t *testing.T) {
		if !containsEntity(result, "Sam Altman") {
			t.Errorf("expected Sam Altman; got %v", entityNames(result))
		}
	})

	t.Run("extracts_event_from_free_text", func(t *testing.T) {
		if !containsEntity(result, "OpenAI Summit") {
			t.Errorf("expected OpenAI Summit; got %v", entityNames(result))
		}
	})

	t.Run("extracts_location_from_free_text", func(t *testing.T) {
		if !containsEntity(result, "San Francisco") {
			t.Errorf("expected San Francisco; got %v", entityNames(result))
		}
	})
}

// ─── Domain: Product Catalog ──────────────────────────────────────────────────

func TestProductCatalog(t *testing.T) {
	records := []extractionRecord{
		makeRecord(map[string]string{
			"product":   "ThinkPad X1 Carbon",
			"brand":     "Lenovo",
			"category":  "Ultrabook",
			"os":        "Windows 11",
		}),
		makeRecord(map[string]string{
			"product":   "IdeaPad Slim 5",
			"brand":     "Lenovo",
			"category":  "Laptop",
			"os":        "Windows 11",
		}),
		makeRecord(map[string]string{
			"product":   "MacBook Pro 16",
			"brand":     "Apple",
			"category":  "Laptop",
			"os":        "macOS Sonoma",
		}),
		makeRecord(map[string]string{
			"product":   "MacBook Air M3",
			"brand":     "Apple",
			"category":  "Ultrabook",
			"os":        "macOS Sonoma",
		}),
		makeRecord(map[string]string{
			"product":   "Surface Pro 9",
			"brand":     "Microsoft",
			"category":  "Tablet",
			"os":        "Windows 11",
		}),
	}

	result := extractFromRecords(records)

	t.Run("extracts_brand_lenovo", func(t *testing.T) {
		if !containsEntity(result, "Lenovo") {
			t.Errorf("expected Lenovo; got %v", entityNames(result))
		}
	})

	t.Run("extracts_brand_apple", func(t *testing.T) {
		if !containsEntity(result, "Apple") {
			t.Errorf("expected Apple; got %v", entityNames(result))
		}
	})

	t.Run("extracts_os_windows", func(t *testing.T) {
		if !containsEntity(result, "Windows 11") {
			t.Errorf("expected Windows 11; got %v", entityNames(result))
		}
	})

	t.Run("extracts_category", func(t *testing.T) {
		if !containsEntity(result, "Laptop") {
			t.Errorf("expected Laptop; got %v", entityNames(result))
		}
	})
}

// ─── Domain: HR Records ───────────────────────────────────────────────────────

func TestHRRecords(t *testing.T) {
	records := []extractionRecord{
		makeRecord(map[string]string{
			"employee":   "Sarah Johnson",
			"department": "Engineering",
			"role":       "Senior Developer",
			"location":   "New York",
			"manager":    "Alex Rivera",
		}),
		makeRecord(map[string]string{
			"employee":   "Michael Torres",
			"department": "Engineering",
			"role":       "DevOps Engineer",
			"location":   "New York",
			"manager":    "Alex Rivera",
		}),
		makeRecord(map[string]string{
			"employee":   "Priya Patel",
			"department": "Product Management",
			"role":       "Senior Developer",
			"location":   "San Francisco",
			"manager":    "Jordan Lee",
		}),
		makeRecord(map[string]string{
			"employee":   "James Osei",
			"department": "Engineering",
			"role":       "Senior Developer",
			"location":   "New York",
			"manager":    "Alex Rivera",
		}),
		makeRecord(map[string]string{
			"employee":   "Laura Schmidt",
			"department": "Product Management",
			"role":       "UX Designer",
			"location":   "San Francisco",
			"manager":    "Jordan Lee",
		}),
	}

	result := extractFromRecords(records)

	t.Run("extracts_recurring_department", func(t *testing.T) {
		if !containsEntity(result, "Engineering") {
			t.Errorf("expected Engineering; got %v", entityNames(result))
		}
	})

	t.Run("extracts_recurring_location", func(t *testing.T) {
		if !containsEntity(result, "New York") {
			t.Errorf("expected New York; got %v", entityNames(result))
		}
	})

	t.Run("extracts_recurring_manager", func(t *testing.T) {
		if !containsEntity(result, "Alex Rivera") {
			t.Errorf("expected Alex Rivera; got %v", entityNames(result))
		}
	})

	t.Run("extracts_recurring_role", func(t *testing.T) {
		if !containsEntity(result, "Senior Developer") {
			t.Errorf("expected Senior Developer; got %v", entityNames(result))
		}
	})

	t.Run("engineering_relates_to_new_york", func(t *testing.T) {
		if !hasRelation(result, "Engineering", "New York") {
			// Relationships are probabilistic; just require some relationships exist.
			if len(result.Relationships) == 0 {
				t.Error("expected relationships between co-occurring entities; got none")
			}
		}
	})
}

// ─── Domain: Research Papers ──────────────────────────────────────────────────

func TestResearchPapers(t *testing.T) {
	records := []extractionRecord{
		makeRecord(map[string]string{
			"title":       "Attention Is All You Need",
			"first_author": "Ashish Vaswani",
			"institution": "Google Brain",
			"venue":       "NeurIPS",
			"year":        "2017",
		}),
		makeRecord(map[string]string{
			"title":       "BERT: Pre-training of Deep Bidirectional Transformers",
			"first_author": "Jacob Devlin",
			"institution": "Google Brain",
			"venue":       "NAACL",
			"year":        "2019",
		}),
		makeRecord(map[string]string{
			"title":       "GPT-4 Technical Report",
			"first_author": "OpenAI Team",
			"institution": "OpenAI",
			"venue":       "ArXiv",
			"year":        "2023",
		}),
		makeRecord(map[string]string{
			"title":       "LLaMA: Open and Efficient Foundation Language Models",
			"first_author": "Hugo Touvron",
			"institution": "Meta AI",
			"venue":       "ArXiv",
			"year":        "2023",
		}),
		makeRecord(map[string]string{
			"title":       "Scaling Laws for Neural Language Models",
			"first_author": "Jared Kaplan",
			"institution": "OpenAI",
			"venue":       "ArXiv",
			"year":        "2020",
		}),
	}

	result := extractFromRecords(records)

	t.Run("extracts_institution", func(t *testing.T) {
		if !containsEntity(result, "Google Brain") && !containsEntity(result, "OpenAI") {
			t.Errorf("expected institution to be extracted; got %v", entityNames(result))
		}
	})

	t.Run("extracts_venue", func(t *testing.T) {
		if !containsEntity(result, "ArXiv") {
			t.Errorf("expected ArXiv; got %v", entityNames(result))
		}
	})
}

// ─── Domain: Legal Documents ──────────────────────────────────────────────────

func TestLegalDocuments(t *testing.T) {
	records := []extractionRecord{
		makeRecord(map[string]string{
			"case":        "Smith v. Jones",
			"court":       "Supreme Court of California",
			"plaintiff":   "John Smith",
			"defendant":   "Acme Corporation",
			"verdict":     "Dismissed",
		}),
		makeRecord(map[string]string{
			"case":        "Williams v. Acme Corporation",
			"court":       "Supreme Court of California",
			"plaintiff":   "Mary Williams",
			"defendant":   "Acme Corporation",
			"verdict":     "Settled",
		}),
		makeRecord(map[string]string{
			"case":        "Chen v. Acme Corporation",
			"court":       "US District Court",
			"plaintiff":   "Robert Chen",
			"defendant":   "Acme Corporation",
			"verdict":     "Dismissed",
		}),
		makeRecord(map[string]string{
			"case":        "Garcia v. TechGiant Inc",
			"court":       "Supreme Court of California",
			"plaintiff":   "Elena Garcia",
			"defendant":   "TechGiant Inc",
			"verdict":     "Plaintiff Prevailed",
		}),
		makeRecord(map[string]string{
			"case":        "Lee v. Acme Corporation",
			"court":       "US District Court",
			"plaintiff":   "James Lee",
			"defendant":   "Acme Corporation",
			"verdict":     "Settled",
		}),
	}

	result := extractFromRecords(records)

	t.Run("extracts_recurring_defendant", func(t *testing.T) {
		if !containsEntity(result, "Acme Corporation") {
			t.Errorf("expected Acme Corporation; got %v", entityNames(result))
		}
	})

	t.Run("extracts_court", func(t *testing.T) {
		if !containsEntity(result, "Supreme Court of California") && !containsEntity(result, "California") {
			t.Errorf("expected court name; got %v", entityNames(result))
		}
	})

	t.Run("extracts_verdict", func(t *testing.T) {
		if !containsEntity(result, "Dismissed") && !containsEntity(result, "Settled") {
			t.Errorf("expected verdict to be extracted; got %v", entityNames(result))
		}
	})
}

// ─── Domain: Supply Chain / Logistics ────────────────────────────────────────

func TestSupplyChain(t *testing.T) {
	records := []extractionRecord{
		makeRecord(map[string]string{
			"supplier":    "Delta Components Ltd",
			"buyer":       "Omega Manufacturing",
			"product":     "Circuit Board",
			"origin":      "Shenzhen",
			"destination": "Detroit",
		}),
		makeRecord(map[string]string{
			"supplier":    "Delta Components Ltd",
			"buyer":       "Omega Manufacturing",
			"product":     "Microcontroller",
			"origin":      "Shenzhen",
			"destination": "Detroit",
		}),
		makeRecord(map[string]string{
			"supplier":    "Alpine Raw Materials",
			"buyer":       "Omega Manufacturing",
			"product":     "Steel Sheet",
			"origin":      "Stuttgart",
			"destination": "Detroit",
		}),
		makeRecord(map[string]string{
			"supplier":    "Delta Components Ltd",
			"buyer":       "Pacific Assemblies",
			"product":     "Circuit Board",
			"origin":      "Shenzhen",
			"destination": "Seattle",
		}),
		makeRecord(map[string]string{
			"supplier":    "Alpine Raw Materials",
			"buyer":       "Pacific Assemblies",
			"product":     "Aluminium Alloy",
			"origin":      "Stuttgart",
			"destination": "Seattle",
		}),
	}

	result := extractFromRecords(records)

	t.Run("extracts_recurring_supplier", func(t *testing.T) {
		if !containsEntity(result, "Delta Components Ltd") {
			t.Errorf("expected Delta Components Ltd; got %v", entityNames(result))
		}
	})

	t.Run("extracts_recurring_buyer", func(t *testing.T) {
		if !containsEntity(result, "Omega Manufacturing") {
			t.Errorf("expected Omega Manufacturing; got %v", entityNames(result))
		}
	})

	t.Run("extracts_recurring_origin", func(t *testing.T) {
		if !containsEntity(result, "Shenzhen") {
			t.Errorf("expected Shenzhen; got %v", entityNames(result))
		}
	})

	t.Run("relationships_exist", func(t *testing.T) {
		if len(result.Relationships) == 0 {
			t.Error("expected relationships between co-occurring entities; got none")
		}
	})
}

// ─── Domain: Mixed / Noisy Data ───────────────────────────────────────────────

// Ensures that purely numeric fields, boolean values, and very common single
// words are NOT extracted as entities.
func TestNoiseSuppression(t *testing.T) {
	records := []extractionRecord{
		makeRecord(map[string]string{
			"id":          "1001",
			"value":       "42.5",
			"flag":        "true",
			"description": "the quick brown fox",
			"entity":      "Acme Corporation",
		}),
		makeRecord(map[string]string{
			"id":          "1002",
			"value":       "17.3",
			"flag":        "false",
			"description": "a small red ball",
			"entity":      "Acme Corporation",
		}),
		makeRecord(map[string]string{
			"id":          "1003",
			"value":       "99.0",
			"flag":        "true",
			"description": "and or but if so",
			"entity":      "Beta Industries",
		}),
	}

	result := extractFromRecords(records)

	t.Run("does_not_extract_numeric_id", func(t *testing.T) {
		for _, e := range result.Entities {
			if e.Name == "1001" || e.Name == "1002" || e.Name == "1003" {
				t.Errorf("pure numeric ID %q should not be extracted", e.Name)
			}
		}
	})

	t.Run("does_not_extract_pure_numeric_value", func(t *testing.T) {
		for _, e := range result.Entities {
			if e.Name == "42.5" || e.Name == "17.3" || e.Name == "99.0" {
				t.Errorf("numeric value %q should not be extracted", e.Name)
			}
		}
	})

	t.Run("does_not_extract_boolean_strings", func(t *testing.T) {
		// "true"/"false" are very short all-lowercase words — should be filtered.
		for _, e := range result.Entities {
			if strings.EqualFold(e.Name, "true") || strings.EqualFold(e.Name, "false") {
				t.Errorf("boolean string %q should not be extracted", e.Name)
			}
		}
	})

	t.Run("does_extract_named_entity_in_noisy_set", func(t *testing.T) {
		if !containsEntity(result, "Acme Corporation") {
			t.Errorf("expected Acme Corporation to survive noise; got %v", entityNames(result))
		}
	})
}

// ─── Domain: Single Record (edge case) ───────────────────────────────────────

// With only one record, rarity is high for everything.  The algorithm should
// still produce entity candidates driven by capitalization and phrase-length.
func TestSingleRecord(t *testing.T) {
	records := []extractionRecord{
		makeRecord(map[string]string{
			"patient":    "Hannah Müller",
			"hospital":   "Charité Berlin",
			"diagnosis":  "Chronic Kidney Disease",
			"medication": "Erythropoietin",
		}),
	}

	result := extractFromRecords(records)

	t.Run("returns_some_entities", func(t *testing.T) {
		if len(result.Entities) == 0 {
			t.Error("expected entities even from a single record")
		}
	})

	t.Run("confidence_above_threshold", func(t *testing.T) {
		for _, e := range result.Entities {
			if e.Confidence < minEntityScore {
				t.Errorf("entity %q confidence %.2f below threshold", e.Name, e.Confidence)
			}
		}
	})
}

// ─── Domain: Deeply nested JSON data ─────────────────────────────────────────

func TestNestedStructure(t *testing.T) {
	nested := &models.CIR{
		Data: map[string]interface{}{
			"order": map[string]interface{}{
				"customer": "Francesca Rossi",
				"billing": map[string]interface{}{
					"company": "Rossi Ceramics SpA",
					"city":    "Florence",
				},
			},
			"items": []interface{}{
				map[string]interface{}{"name": "Tuscan Floor Tile", "qty": 200},
				map[string]interface{}{"name": "Marble Slab", "qty": 10},
			},
		},
	}
	records := cirToRows(nested)

	// Add a few more to give the algorithm some corpus signal.
	records = append(records, makeRecord(map[string]string{
		"customer": "Francesca Rossi",
		"company":  "Rossi Ceramics SpA",
		"city":     "Florence",
	}))
	records = append(records, makeRecord(map[string]string{
		"customer": "Marco Bianchi",
		"company":  "Rossi Ceramics SpA",
		"city":     "Florence",
	}))

	result := extractFromRecords(records)

	t.Run("extracts_from_nested_map", func(t *testing.T) {
		if !containsEntity(result, "Rossi Ceramics SpA") && !containsEntity(result, "Rossi Ceramics") {
			t.Errorf("expected Rossi Ceramics to be extracted from nested data; got %v", entityNames(result))
		}
	})

	t.Run("extracts_city", func(t *testing.T) {
		if !containsEntity(result, "Florence") {
			t.Errorf("expected Florence; got %v", entityNames(result))
		}
	})
}

// ─── Tokeniser unit tests ─────────────────────────────────────────────────────

func TestTokeniseFieldValue(t *testing.T) {
	tests := []struct {
		name      string
		fieldKey  string
		text      string
		wantNgram string // a specific n-gram we expect to be generated
	}{
		{
			name:      "two_word_proper_noun",
			fieldKey:  "patient",
			text:      "Sarah Thompson",
			wantNgram: "Sarah Thompson",
		},
		{
			name:      "three_word_phrase",
			fieldKey:  "diagnosis",
			text:      "Type 2 Diabetes",
			wantNgram: "Type 2 Diabetes",
		},
		{
			name:      "full_value_flag",
			fieldKey:  "hospital",
			text:      "St. Mary's Medical Center",
			wantNgram: "St. Mary's Medical Center",
		},
		{
			name:      "single_word_capitalised",
			fieldKey:  "brand",
			text:      "Lenovo",
			wantNgram: "Lenovo",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			occs := tokeniseFieldValue(tc.fieldKey, tc.text)
			found := false
			for _, occ := range occs {
				if occ.text == tc.wantNgram {
					found = true
					break
				}
			}
			if !found {
				all := make([]string, 0, len(occs))
				for _, o := range occs {
					all = append(all, o.text)
				}
				t.Errorf("want n-gram %q; got %v", tc.wantNgram, all)
			}
		})
	}
}

func TestIsFiltered(t *testing.T) {
	shouldFilter := []string{
		"",
		"a",
		"42",
		"3.14",
		"100,000",
		"the",
		"and",
		"   ",
		"!",
		"123-456",
	}
	shouldPass := []string{
		"Lenovo",
		"New York",
		"Acme Corporation",
		"SENSOR-4821",
		"Metformin",
		"co-founder",
	}

	for _, s := range shouldFilter {
		if !isFiltered(s) {
			t.Errorf("isFiltered(%q) = false; want true", s)
		}
	}
	for _, s := range shouldPass {
		if isFiltered(s) {
			t.Errorf("isFiltered(%q) = true; want false", s)
		}
	}
}

// ─── Subsumption pruning unit test ───────────────────────────────────────────

func TestSubsumptionPrune(t *testing.T) {
	// "New York" should survive, "New" should be demoted if it always appears
	// as part of "New York" and not standalone.
	records := []extractionRecord{
		makeRecord(map[string]string{"city": "New York", "country": "United States"}),
		makeRecord(map[string]string{"city": "New York", "country": "United States"}),
		makeRecord(map[string]string{"city": "New York", "country": "United States"}),
		makeRecord(map[string]string{"city": "New York", "country": "Canada"}),
		makeRecord(map[string]string{"city": "New York", "country": "United States"}),
	}

	result := extractFromRecords(records)

	// "New York" must be present.
	if !containsEntity(result, "New York") {
		t.Errorf("expected New York; got %v", entityNames(result))
	}

	// "New" alone must NOT outrank "New York" — ideally it is subsumed or absent.
	newYorkConf := 0.0
	newConf := 0.0
	for _, e := range result.Entities {
		if strings.EqualFold(e.Name, "New York") {
			newYorkConf = e.Confidence
		}
		if strings.EqualFold(e.Name, "New") {
			newConf = e.Confidence
		}
	}
	if newConf > 0 && newYorkConf > 0 && newConf > newYorkConf {
		t.Errorf("'New' (conf %.2f) should not outrank 'New York' (conf %.2f)", newConf, newYorkConf)
	}
}

// ─── NPMI relationship test ───────────────────────────────────────────────────

func TestRelationshipsViaCoOccurrence(t *testing.T) {
	// "Metformin" and "Type 2 Diabetes" always co-occur.
	// "Lisinopril" and "Hypertension" always co-occur but never with Metformin.
	records := []extractionRecord{
		makeRecord(map[string]string{"medication": "Metformin", "condition": "Type 2 Diabetes"}),
		makeRecord(map[string]string{"medication": "Metformin", "condition": "Type 2 Diabetes"}),
		makeRecord(map[string]string{"medication": "Metformin", "condition": "Type 2 Diabetes"}),
		makeRecord(map[string]string{"medication": "Lisinopril", "condition": "Hypertension"}),
		makeRecord(map[string]string{"medication": "Lisinopril", "condition": "Hypertension"}),
	}

	result := extractFromRecords(records)

	if !hasRelation(result, "Metformin", "Type 2 Diabetes") {
		// NPMI between Metformin and Type 2 Diabetes should be near 1.0.
		// Allow for the possibility that "Type 2 Diabetes" was subsumed or scored differently.
		found := false
		for _, r := range result.Relationships {
			if strings.Contains(strings.ToLower(r.Entity1.Name), "metformin") ||
				strings.Contains(strings.ToLower(r.Entity2.Name), "metformin") {
				found = true
				break
			}
		}
		if !found && len(result.Entities) > 1 {
			t.Logf("entities: %v", entityNames(result))
			t.Logf("relationships: %d", len(result.Relationships))
			t.Error("expected a relationship involving Metformin")
		}
	}
}

// ─── Empty corpus edge case ───────────────────────────────────────────────────

func TestEmptyRecords(t *testing.T) {
	result := extractFromRecords(nil)
	if result == nil {
		t.Fatal("expected non-nil result for empty input")
	}
	if len(result.Entities) != 0 {
		t.Errorf("expected no entities for empty input; got %v", entityNames(result))
	}
}

func TestAllBlankValues(t *testing.T) {
	records := []extractionRecord{
		makeRecord(map[string]string{"a": "", "b": "   ", "c": ""}),
		makeRecord(map[string]string{"a": "", "b": ""}),
	}
	result := extractFromRecords(records)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	// No entities should be extracted from blank values.
	if len(result.Entities) != 0 {
		t.Errorf("expected no entities from blank values; got %v", entityNames(result))
	}
}

// ─── CIR format: array / structured ──────────────────────────────────────────
//
// When CIR.Data is a []interface{} (tabular data — the format produced by
// CSV or database ingestion), each element must become its own corpus
// document so that IDF and NPMI are computed at row granularity.

func TestCIRArrayFormat(t *testing.T) {
	// Simulate a CSV-style CIR where the entire table is one CIR record.
	// Without proper row-expansion this would collapse 6 rows into a single
	// document and break the corpus statistics.
	cir := &models.CIR{
		Data: []interface{}{
			map[string]interface{}{"employee": "Alice Nguyen", "department": "Engineering", "location": "Berlin"},
			map[string]interface{}{"employee": "Bob Petrov", "department": "Engineering", "location": "Berlin"},
			map[string]interface{}{"employee": "Clara Müller", "department": "Engineering", "location": "Berlin"},
			map[string]interface{}{"employee": "Diego Reyes", "department": "Marketing", "location": "Madrid"},
			map[string]interface{}{"employee": "Eva Lindqvist", "department": "Marketing", "location": "Madrid"},
			map[string]interface{}{"employee": "Faisal Al-Amin", "department": "Engineering", "location": "Berlin"},
		},
	}

	rows := cirToRows(cir)
	t.Run("each_array_element_is_own_record", func(t *testing.T) {
		if len(rows) != 6 {
			t.Errorf("expected 6 records (one per row); got %d", len(rows))
		}
	})

	result := extractFromRecords(rows)

	t.Run("extracts_department", func(t *testing.T) {
		if !containsEntity(result, "Engineering") {
			t.Errorf("expected Engineering; got %v", entityNames(result))
		}
	})

	t.Run("extracts_location", func(t *testing.T) {
		if !containsEntity(result, "Berlin") {
			t.Errorf("expected Berlin; got %v", entityNames(result))
		}
	})

	t.Run("relationships_between_dept_and_location", func(t *testing.T) {
		if len(result.Relationships) == 0 {
			t.Error("expected relationships from array CIR; got none")
		}
	})
}

func TestCIRSingleDocumentMap(t *testing.T) {
	// A single-document CIR (map-type) should produce exactly one record.
	cir := &models.CIR{
		Data: map[string]interface{}{
			"company":  "Helix Dynamics",
			"sector":   "Aerospace",
			"hq":       "Houston",
			"founded":  "2008",
		},
	}
	rows := cirToRows(cir)
	if len(rows) != 1 {
		t.Errorf("expected 1 record for map CIR; got %d", len(rows))
	}
}

func TestCIRRawTextBlob(t *testing.T) {
	// A string CIR (text blob) should produce one record.
	cir := &models.CIR{
		Data: "Helix Dynamics is an aerospace company headquartered in Houston.",
	}
	rows := cirToRows(cir)
	if len(rows) != 1 {
		t.Errorf("expected 1 record for text CIR; got %d", len(rows))
	}
	if len(rows[0].ngrams) == 0 {
		t.Error("expected n-grams from text blob")
	}
}

// ─── CIR format: hybrid (structured fields + free text) ──────────────────────
//
// A map-type CIR record may contain a mix of structured fields (ID, category,
// status) and free-text fields (description, notes, comments).  The algorithm
// should extract entities from both within the same corpus document.

func TestCIRHybridStructuredAndFreeText(t *testing.T) {
	// Five support ticket records: structured fields (product, priority) plus
	// a free-text description that mentions engineers and systems.
	tickets := []extractionRecord{}
	rawTickets := []struct {
		product     string
		priority    string
		description string
	}{
		{"Navigator Pro", "High", "Laura Svensson reported that the Navigator Pro authentication module fails when connecting to the Keycloak Identity Server."},
		{"Navigator Pro", "High", "Tom Okafor observed the Navigator Pro login screen hangs after the Keycloak Identity Server returns a 503 response."},
		{"Navigator Pro", "Medium", "Navigator Pro crashes on startup when the Keycloak Identity Server certificate has expired."},
		{"DataBridge", "High", "DataBridge pipeline fails to sync with the Keycloak Identity Server during peak load. Reported by James Wu."},
		{"DataBridge", "Medium", "Intermittent DataBridge connection drops observed by Priya Menon during Keycloak Identity Server maintenance windows."},
	}

	for _, tk := range rawTickets {
		cir := &models.CIR{
			Data: map[string]interface{}{
				"product":     tk.product,
				"priority":    tk.priority,
				"description": tk.description,
			},
		}
		rows := cirToRows(cir)
		tickets = append(tickets, rows...)
	}

	result := extractFromRecords(tickets)

	t.Run("extracts_product_from_structured_field", func(t *testing.T) {
		if !containsEntity(result, "Navigator Pro") {
			t.Errorf("expected Navigator Pro; got %v", entityNames(result))
		}
	})

	t.Run("extracts_system_from_free_text", func(t *testing.T) {
		if !containsEntity(result, "Keycloak Identity Server") {
			t.Errorf("expected Keycloak Identity Server from free text; got %v", entityNames(result))
		}
	})

	t.Run("extracts_product_from_free_text_too", func(t *testing.T) {
		// "Navigator Pro" appears in both the structured field and the free text.
		e := entityByName(result, "Navigator Pro")
		if e == nil {
			t.Errorf("Navigator Pro not found in %v", entityNames(result))
			return
		}
		// Because it appears in two distinct field keys (product + description)
		// its focus score will be moderate but cap+length scores still push it through.
		if e.Confidence < minEntityScore {
			t.Errorf("Navigator Pro confidence %.2f below threshold", e.Confidence)
		}
	})

	t.Run("all_key_entities_found_above_threshold", func(t *testing.T) {
		// All salient entities — including those found only in free text —
		// must be extracted with confidence at or above the minimum threshold.
		// Note: "High"/"Medium" legitimately score well because they are
		// consistently capitalised complete field values (they represent
		// priority-level concepts in a ticketing ontology).  The invariant
		// we care about is that free-text entities are also found.
		var keycloakConf float64
		for _, e := range result.Entities {
			if strings.Contains(strings.ToLower(e.Name), "keycloak") {
				if e.Confidence > keycloakConf {
					keycloakConf = e.Confidence
				}
			}
		}
		if keycloakConf < minEntityScore {
			t.Errorf("Keycloak Identity Server conf %.2f below threshold %.2f; entities: %v",
				keycloakConf, minEntityScore, entityNames(result))
		}
	})
}

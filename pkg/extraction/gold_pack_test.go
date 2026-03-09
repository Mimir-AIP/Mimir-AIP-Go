package extraction

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

type goldPack struct {
	Domains []goldDomain `json:"domains"`
}

type goldDomain struct {
	Name                string              `json:"name"`
	Sources             []goldSource        `json:"sources"`
	ExpectedLinks       [][]string          `json:"expected_links"`
	UnstructuredRecords []map[string]string `json:"unstructured_records"`
	ExpectedEntities    []string            `json:"expected_entities"`
}

type goldSource struct {
	StorageID  string                   `json:"storage_id"`
	EntityType string                   `json:"entity_type"`
	Rows       []map[string]interface{} `json:"rows"`
}

func TestGoldPackCrossDomainAccuracy(t *testing.T) {
	pack := mustLoadGoldPack(t)
	for _, domain := range pack.Domains {
		domain := domain
		t.Run(domain.Name, func(t *testing.T) {
			profiles := buildProfilesFromGoldDomain(domain)
			expected := expectedKeysFromGoldDomain(domain)
			links := DetectCrossSourceLinks(profiles)
			if len(links) == 0 {
				t.Fatalf("expected links for domain %s", domain.Name)
			}

			pred := observedLinkKeys(links)
			precision, recall, f1, tp, fp, fn := scoreSet(pred, expected)
			if precision < 0.95 || recall < 0.95 || f1 < 0.95 {
				t.Fatalf("gold-pack %s below gate: precision=%.3f recall=%.3f f1=%.3f tp=%d fp=%d fn=%d\nwant=%v\ngot=%v",
					domain.Name, precision, recall, f1, tp, fp, fn, sortedKeys(expected), sortedKeys(pred))
			}
		})
	}
}

func TestGoldPackUnstructuredCoverage(t *testing.T) {
	pack := mustLoadGoldPack(t)
	for _, domain := range pack.Domains {
		domain := domain
		t.Run(domain.Name, func(t *testing.T) {
			records := make([]extractionRecord, 0, len(domain.UnstructuredRecords))
			for _, rec := range domain.UnstructuredRecords {
				records = append(records, makeRecord(rec))
			}
			result := extractFromRecords(records)
			if len(result.Entities) == 0 {
				t.Fatalf("expected entities for domain %s", domain.Name)
			}

			found := 0
			for _, want := range domain.ExpectedEntities {
				if containsEntity(result, want) {
					found++
				}
			}
			coverage := float64(found) / float64(len(domain.ExpectedEntities))
			if coverage < 0.80 {
				t.Fatalf("gold-pack %s unstructured coverage too low: %.3f (%d/%d), entities=%v",
					domain.Name, coverage, found, len(domain.ExpectedEntities), entityNames(result))
			}
		})
	}
}

func TestGoldPackPerformanceBudgets(t *testing.T) {
	pack := mustLoadGoldPack(t)
	profilesByDomain := make([][]models.ColumnProfile, 0, len(pack.Domains))
	recordsByDomain := make([][]extractionRecord, 0, len(pack.Domains))
	structuredByDomain := make([]*models.CIR, 0, len(pack.Domains))

	for _, domain := range pack.Domains {
		profilesByDomain = append(profilesByDomain, buildProfilesFromGoldDomain(domain))
		records := make([]extractionRecord, 0, len(domain.UnstructuredRecords))
		for _, rec := range domain.UnstructuredRecords {
			records = append(records, makeRecord(rec))
		}
		recordsByDomain = append(recordsByDomain, records)
		structuredByDomain = append(structuredByDomain, goldStructuredCIR(domain))
	}

	allocs := testing.AllocsPerRun(20, func() {
		for i := range pack.Domains {
			_ = DetectCrossSourceLinks(profilesByDomain[i])
			u := extractFromRecords(recordsByDomain[i])
			s, err := ExtractFromStructuredCIR(structuredByDomain[i])
			if err != nil {
				t.Fatalf("structured extraction failed: %v", err)
			}
			_ = ReconcileEntities(s, u)
		}
	})
	if allocs > 28000 {
		t.Fatalf("allocation budget exceeded: got %.0f allocs/run want <= 28000", allocs)
	}

	start := time.Now()
	const iterations = 40
	for i := 0; i < iterations; i++ {
		for di := range pack.Domains {
			_ = DetectCrossSourceLinks(profilesByDomain[di])
			u := extractFromRecords(recordsByDomain[di])
			s, err := ExtractFromStructuredCIR(structuredByDomain[di])
			if err != nil {
				t.Fatalf("structured extraction failed: %v", err)
			}
			_ = ReconcileEntities(s, u)
		}
	}
	avg := time.Since(start) / iterations
	if avg > 25*time.Millisecond {
		t.Fatalf("latency budget exceeded: avg=%s want <= 25ms", avg)
	}
}

func BenchmarkGoldPackScaleMatrix(b *testing.B) {
	tiers := []int{1000, 10000, 100000}
	for _, n := range tiers {
		n := n
		b.Run(fmt.Sprintf("rows_%d", n), func(b *testing.B) {
			ehrRows := make([]map[string]interface{}, 0, n)
			claimsRows := make([]map[string]interface{}, 0, n)
			labsRows := make([]map[string]interface{}, 0, n)
			for i := 1; i <= n; i++ {
				pid := fmt.Sprintf("P%07d", i)
				ehrRows = append(ehrRows, map[string]interface{}{"patient_id": pid, "facility_code": fmt.Sprintf("F%02d", i%12)})
				claimsRows = append(claimsRows, map[string]interface{}{"member_ref": pid, "claim_id": fmt.Sprintf("CLM%08d", i)})
				labsRows = append(labsRows, map[string]interface{}{"subject_key": pid, "lab_order": fmt.Sprintf("LAB%08d", i)})
			}
			profiles := append([]models.ColumnProfile{}, BuildColumnProfilesFromCIRs("ehr", []*models.CIR{makeCIR("ehr", "ClinicalRecord", ehrRows)})...)
			profiles = append(profiles, BuildColumnProfilesFromCIRs("claims", []*models.CIR{makeCIR("claims", "ClaimRecord", claimsRows)})...)
			profiles = append(profiles, BuildColumnProfilesFromCIRs("labs", []*models.CIR{makeCIR("labs", "LabRecord", labsRows)})...)

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				links := DetectCrossSourceLinks(profiles)
				if len(links) == 0 {
					b.Fatal("expected links")
				}
			}
		})
	}
}

func mustLoadGoldPack(t *testing.T) goldPack {
	t.Helper()
	raw, err := os.ReadFile("testdata/gold_pack.json")
	if err != nil {
		t.Fatalf("failed to read gold pack: %v", err)
	}
	var pack goldPack
	if err := json.Unmarshal(raw, &pack); err != nil {
		t.Fatalf("failed to parse gold pack: %v", err)
	}
	if len(pack.Domains) == 0 {
		t.Fatal("gold pack has no domains")
	}
	return pack
}

func buildProfilesFromGoldDomain(domain goldDomain) []models.ColumnProfile {
	profiles := make([]models.ColumnProfile, 0)
	for _, src := range domain.Sources {
		cir := makeCIR(src.StorageID, src.EntityType, src.Rows)
		profiles = append(profiles, BuildColumnProfilesFromCIRs(src.StorageID, []*models.CIR{cir})...)
	}
	return profiles
}

func expectedKeysFromGoldDomain(domain goldDomain) map[string]struct{} {
	expected := make(map[string]struct{}, len(domain.ExpectedLinks))
	for _, pair := range domain.ExpectedLinks {
		if len(pair) != 2 {
			continue
		}
		lhsStorage, lhsCol, okA := splitQualifiedKey(pair[0])
		rhsStorage, rhsCol, okB := splitQualifiedKey(pair[1])
		if !okA || !okB {
			continue
		}
		expected[canonicalLinkKey(lhsStorage, lhsCol, rhsStorage, rhsCol)] = struct{}{}
	}
	return expected
}

func splitQualifiedKey(qualified string) (storageID, col string, ok bool) {
	for i := 0; i < len(qualified); i++ {
		if qualified[i] == '.' {
			if i == 0 || i == len(qualified)-1 {
				return "", "", false
			}
			return qualified[:i], qualified[i+1:], true
		}
	}
	return "", "", false
}

func goldStructuredCIR(domain goldDomain) *models.CIR {
	if len(domain.Sources) == 0 {
		return makeCIR(domain.Name+"-structured", "StructuredRecord", nil)
	}
	// Use the first source as a representative structured dataset for mixed-mode budget checks.
	src := domain.Sources[0]
	rows := make([]map[string]interface{}, len(src.Rows))
	for i := range src.Rows {
		rows[i] = src.Rows[i]
	}
	return makeCIR(domain.Name+"-structured", src.EntityType, rows)
}

func sortedDomainNames(pack goldPack) []string {
	names := make([]string, 0, len(pack.Domains))
	for _, d := range pack.Domains {
		names = append(names, d.Name)
	}
	sort.Strings(names)
	return names
}

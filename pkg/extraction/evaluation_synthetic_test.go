package extraction

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

type syntheticDomainCase struct {
	name             string
	profiles         []models.ColumnProfile
	expectedLinkKeys map[string]struct{}
	records          []extractionRecord
	expectedEntities []string
}

func TestSyntheticCrossDomainLinkingAccuracy(t *testing.T) {
	domains := syntheticDomainCases()

	for _, domain := range domains {
		t.Run(domain.name, func(t *testing.T) {
			links := DetectCrossSourceLinks(domain.profiles)
			if len(links) == 0 {
				t.Fatalf("expected non-empty links for %s", domain.name)
			}

			pred := observedLinkKeys(links)
			precision, recall, f1, tp, fp, fn := scoreSet(pred, domain.expectedLinkKeys)

			if precision < 0.95 || recall < 0.95 || f1 < 0.95 {
				t.Fatalf("%s link quality below threshold: precision=%.3f recall=%.3f f1=%.3f tp=%d fp=%d fn=%d\nwant keys=%v\ngot keys=%v",
					domain.name, precision, recall, f1, tp, fp, fn, sortedKeys(domain.expectedLinkKeys), sortedKeys(pred))
			}
		})
	}
}

func TestSyntheticCrossDomainUnstructuredEntityCoverage(t *testing.T) {
	domains := syntheticDomainCases()

	for _, domain := range domains {
		t.Run(domain.name, func(t *testing.T) {
			result := extractFromRecords(domain.records)
			if len(result.Entities) == 0 {
				t.Fatalf("expected non-empty entities for %s", domain.name)
			}

			found := 0
			for _, want := range domain.expectedEntities {
				if containsEntity(result, want) {
					found++
				}
			}

			coverage := float64(found) / float64(len(domain.expectedEntities))
			if coverage < 0.80 {
				t.Fatalf("%s unstructured entity coverage below threshold: got %.3f (%d/%d), entities=%v",
					domain.name, coverage, found, len(domain.expectedEntities), entityNames(result))
			}
		})
	}
}

func TestSyntheticMixedModeReconciliation(t *testing.T) {
	domains := syntheticDomainCases()

	for _, domain := range domains {
		t.Run(domain.name, func(t *testing.T) {
			structuredCIR := syntheticStructuredCIRForDomain(domain.name)
			structuredResult, err := ExtractFromStructuredCIR(structuredCIR)
			if err != nil {
				t.Fatalf("ExtractFromStructuredCIR failed: %v", err)
			}
			unstructuredResult := extractFromRecords(domain.records)

			reconciled := ReconcileEntities(structuredResult, unstructuredResult)
			if len(reconciled.Entities) == 0 {
				t.Fatalf("expected reconciled entities for %s", domain.name)
			}

			// Verify a canonical recurring entity per domain is retained post-reconciliation.
			anchor := domain.expectedEntities[0]
			if !containsEntity(reconciled, anchor) {
				t.Fatalf("expected reconciled output to contain anchor entity %q for %s", anchor, domain.name)
			}
		})
	}
}

func BenchmarkSyntheticExtractionAccuracyPipeline(b *testing.B) {
	domains := syntheticDomainCases()
	structured := make([]*models.CIR, 0, len(domains))
	for _, d := range domains {
		structured = append(structured, syntheticStructuredCIRForDomain(d.name))
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for di, domain := range domains {
			_ = DetectCrossSourceLinks(domain.profiles)
			unstructuredResult := extractFromRecords(domain.records)
			structuredResult, err := ExtractFromStructuredCIR(structured[di])
			if err != nil {
				b.Fatalf("ExtractFromStructuredCIR failed: %v", err)
			}
			_ = ReconcileEntities(structuredResult, unstructuredResult)
		}
	}
}

func syntheticDomainCases() []syntheticDomainCase {
	healthcareProfiles, healthcareExpected := syntheticHealthcareProfiles()
	legalProfiles, legalExpected := syntheticLegalProfiles()
	mediaProfiles, mediaExpected := syntheticMediaProfiles()

	return []syntheticDomainCase{
		{
			name:             "healthcare",
			profiles:         healthcareProfiles,
			expectedLinkKeys: healthcareExpected,
			records:          syntheticHealthcareRecords(),
			expectedEntities: []string{"Metformin", "St. Mary's Medical Center", "Type 2 Diabetes"},
		},
		{
			name:             "legal",
			profiles:         legalProfiles,
			expectedLinkKeys: legalExpected,
			records:          syntheticLegalRecords(),
			expectedEntities: []string{"Acme Corporation", "Supreme Court of California", "Dismissed"},
		},
		{
			name:             "media",
			profiles:         mediaProfiles,
			expectedLinkKeys: mediaExpected,
			records:          syntheticMediaRecords(),
			expectedEntities: []string{"OpenAI Summit", "San Francisco", "Sam Altman"},
		},
	}
}

func syntheticHealthcareProfiles() ([]models.ColumnProfile, map[string]struct{}) {
	ehrRows := make([]map[string]interface{}, 0, 120)
	claimsRows := make([]map[string]interface{}, 0, 120)
	labsRows := make([]map[string]interface{}, 0, 120)
	for i := 1; i <= 120; i++ {
		id := fmt.Sprintf("P%04d", i)
		ehrRows = append(ehrRows, map[string]interface{}{
			"patient_id":     id,
			"diagnosis_code": fmt.Sprintf("DX%03d", i%18),
			"facility_code":  fmt.Sprintf("F%02d", i%6),
		})
		claimsRows = append(claimsRows, map[string]interface{}{
			"member_ref": id,
			"claim_id":   fmt.Sprintf("CLM%06d", i),
			"payer_code": fmt.Sprintf("PAY%02d", i%4),
		})
		labsRows = append(labsRows, map[string]interface{}{
			"subject_key": id,
			"lab_order":   fmt.Sprintf("LAB%06d", i),
			"panel_code":  fmt.Sprintf("LP%02d", i%8),
		})
	}

	ehr := makeCIR("ehr", "ClinicalRecord", ehrRows)
	claims := makeCIR("claims", "ClaimRecord", claimsRows)
	labs := makeCIR("labs", "LabRecord", labsRows)

	profiles := append([]models.ColumnProfile{}, BuildColumnProfilesFromCIRs("ehr", []*models.CIR{ehr})...)
	profiles = append(profiles, BuildColumnProfilesFromCIRs("claims", []*models.CIR{claims})...)
	profiles = append(profiles, BuildColumnProfilesFromCIRs("labs", []*models.CIR{labs})...)

	expected := map[string]struct{}{
		canonicalLinkKey("ehr", "patient_id", "claims", "member_ref"):   {},
		canonicalLinkKey("ehr", "patient_id", "labs", "subject_key"):    {},
		canonicalLinkKey("claims", "member_ref", "labs", "subject_key"): {},
	}
	return profiles, expected
}

func syntheticLegalProfiles() ([]models.ColumnProfile, map[string]struct{}) {
	casesRows := make([]map[string]interface{}, 0, 100)
	filingsRows := make([]map[string]interface{}, 0, 100)
	judgmentsRows := make([]map[string]interface{}, 0, 100)
	for i := 1; i <= 100; i++ {
		caseID := fmt.Sprintf("CASE-%05d", i)
		casesRows = append(casesRows, map[string]interface{}{
			"case_id":      caseID,
			"court_code":   fmt.Sprintf("CRT%02d", i%5),
			"verdict_code": fmt.Sprintf("V%02d", i%4),
		})
		filingsRows = append(filingsRows, map[string]interface{}{
			"docket_ref":  caseID,
			"filing_id":   fmt.Sprintf("FIL-%06d", i),
			"office_code": fmt.Sprintf("OF%02d", i%6),
		})
		judgmentsRows = append(judgmentsRows, map[string]interface{}{
			"matter_key": caseID,
			"order_id":   fmt.Sprintf("ORD-%06d", i),
			"panel_code": fmt.Sprintf("PNL%02d", i%7),
		})
	}

	cases := makeCIR("cases", "CaseRecord", casesRows)
	filings := makeCIR("filings", "FilingRecord", filingsRows)
	judgments := makeCIR("judgments", "JudgmentRecord", judgmentsRows)

	profiles := append([]models.ColumnProfile{}, BuildColumnProfilesFromCIRs("cases", []*models.CIR{cases})...)
	profiles = append(profiles, BuildColumnProfilesFromCIRs("filings", []*models.CIR{filings})...)
	profiles = append(profiles, BuildColumnProfilesFromCIRs("judgments", []*models.CIR{judgments})...)

	expected := map[string]struct{}{
		canonicalLinkKey("cases", "case_id", "filings", "docket_ref"):        {},
		canonicalLinkKey("cases", "case_id", "judgments", "matter_key"):      {},
		canonicalLinkKey("filings", "docket_ref", "judgments", "matter_key"): {},
	}
	return profiles, expected
}

func syntheticMediaProfiles() ([]models.ColumnProfile, map[string]struct{}) {
	cmsRows := make([]map[string]interface{}, 0, 150)
	analyticsRows := make([]map[string]interface{}, 0, 150)
	adsRows := make([]map[string]interface{}, 0, 150)
	for i := 1; i <= 150; i++ {
		articleID := fmt.Sprintf("ART-%05d", i)
		cmsRows = append(cmsRows, map[string]interface{}{
			"article_id": articleID,
			"section":    []string{"world", "business", "tech", "politics"}[i%4],
			"author_ref": fmt.Sprintf("AUTH%03d", i%17),
		})
		analyticsRows = append(analyticsRows, map[string]interface{}{
			"content_ref": articleID,
			"session_id":  fmt.Sprintf("SES-%07d", i),
			"geo_code":    fmt.Sprintf("G%02d", i%11),
		})
		adsRows = append(adsRows, map[string]interface{}{
			"story_key":  articleID,
			"ad_slot_id": fmt.Sprintf("AD-%06d", i),
			"campaign":   fmt.Sprintf("CMP%02d", i%9),
		})
	}

	cms := makeCIR("cms", "ArticleRecord", cmsRows)
	analytics := makeCIR("analytics", "EngagementRecord", analyticsRows)
	ads := makeCIR("ads", "AdInventoryRecord", adsRows)

	profiles := append([]models.ColumnProfile{}, BuildColumnProfilesFromCIRs("cms", []*models.CIR{cms})...)
	profiles = append(profiles, BuildColumnProfilesFromCIRs("analytics", []*models.CIR{analytics})...)
	profiles = append(profiles, BuildColumnProfilesFromCIRs("ads", []*models.CIR{ads})...)

	expected := map[string]struct{}{
		canonicalLinkKey("cms", "article_id", "analytics", "content_ref"): {},
		canonicalLinkKey("cms", "article_id", "ads", "story_key"):         {},
		canonicalLinkKey("analytics", "content_ref", "ads", "story_key"):  {},
	}
	return profiles, expected
}

func syntheticHealthcareRecords() []extractionRecord {
	return []extractionRecord{
		makeRecord(map[string]string{"patient": "Sarah Thompson", "diagnosis": "Type 2 Diabetes", "medication": "Metformin", "hospital": "St. Mary's Medical Center", "physician": "Dr. James Okafor"}),
		makeRecord(map[string]string{"patient": "Robert Chen", "diagnosis": "Type 2 Diabetes", "medication": "Metformin", "hospital": "St. Mary's Medical Center", "physician": "Dr. James Okafor"}),
		makeRecord(map[string]string{"patient": "Maria Gonzalez", "diagnosis": "Hypertensive Heart Disease", "medication": "Lisinopril", "hospital": "Riverside General Hospital", "physician": "Dr. Priya Sharma"}),
		makeRecord(map[string]string{"patient": "David Kim", "diagnosis": "Type 2 Diabetes", "medication": "Metformin", "hospital": "St. Mary's Medical Center", "physician": "Dr. James Okafor"}),
		makeRecord(map[string]string{"patient": "Emily Nakamura", "diagnosis": "Atrial Fibrillation", "medication": "Warfarin", "hospital": "Riverside General Hospital", "physician": "Dr. Priya Sharma"}),
	}
}

func syntheticLegalRecords() []extractionRecord {
	return []extractionRecord{
		makeRecord(map[string]string{"case": "Smith v. Jones", "court": "Supreme Court of California", "plaintiff": "John Smith", "defendant": "Acme Corporation", "verdict": "Dismissed"}),
		makeRecord(map[string]string{"case": "Williams v. Acme Corporation", "court": "Supreme Court of California", "plaintiff": "Mary Williams", "defendant": "Acme Corporation", "verdict": "Settled"}),
		makeRecord(map[string]string{"case": "Chen v. Acme Corporation", "court": "US District Court", "plaintiff": "Robert Chen", "defendant": "Acme Corporation", "verdict": "Dismissed"}),
		makeRecord(map[string]string{"case": "Garcia v. TechGiant Inc", "court": "Supreme Court of California", "plaintiff": "Elena Garcia", "defendant": "TechGiant Inc", "verdict": "Plaintiff Prevailed"}),
		makeRecord(map[string]string{"case": "Lee v. Acme Corporation", "court": "US District Court", "plaintiff": "James Lee", "defendant": "Acme Corporation", "verdict": "Settled"}),
	}
}

func syntheticMediaRecords() []extractionRecord {
	return []extractionRecord{
		makeRecord(map[string]string{"author": "jane_doe", "content": "Just attended the OpenAI Summit in San Francisco. Great session with Sam Altman and Greg Brockman."}),
		makeRecord(map[string]string{"author": "tech_blogger", "content": "Sam Altman announced new GPT features at the OpenAI Summit. Huge crowd at San Francisco."}),
		makeRecord(map[string]string{"author": "ai_watcher", "content": "OpenAI Summit was incredible. San Francisco never disappoints for tech events. Sam Altman is inspiring."}),
		makeRecord(map[string]string{"author": "dev_news", "content": "OpenAI Summit highlights: Sam Altman on the future of AGI. Venue was San Francisco Moscone Center."}),
		makeRecord(map[string]string{"author": "ml_researcher", "content": "Great talks at the OpenAI Summit. Met Greg Brockman in San Francisco. Exciting times for AI."}),
	}
}

func syntheticStructuredCIRForDomain(domain string) *models.CIR {
	switch domain {
	case "healthcare":
		rows := make([]map[string]interface{}, 0, 80)
		for i := 1; i <= 80; i++ {
			rows = append(rows, map[string]interface{}{
				"patient_id": fmt.Sprintf("P%04d", i),
				"hospital":   []string{"St. Mary's Medical Center", "Riverside General Hospital"}[i%2],
				"diagnosis":  []string{"Type 2 Diabetes", "Hypertensive Heart Disease", "Atrial Fibrillation"}[i%3],
			})
		}
		return makeCIR("healthcare-structured", "ClinicalRecord", rows)
	case "legal":
		rows := make([]map[string]interface{}, 0, 70)
		for i := 1; i <= 70; i++ {
			rows = append(rows, map[string]interface{}{
				"case_id":   fmt.Sprintf("CASE-%05d", i),
				"defendant": []string{"Acme Corporation", "TechGiant Inc"}[i%2],
				"court":     []string{"Supreme Court of California", "US District Court"}[i%2],
			})
		}
		return makeCIR("legal-structured", "CaseRecord", rows)
	default: // media
		rows := make([]map[string]interface{}, 0, 90)
		for i := 1; i <= 90; i++ {
			rows = append(rows, map[string]interface{}{
				"article_id": fmt.Sprintf("ART-%05d", i),
				"event":      "OpenAI Summit",
				"city":       "San Francisco",
			})
		}
		return makeCIR("media-structured", "ArticleRecord", rows)
	}
}

func observedLinkKeys(links []models.CrossSourceLink) map[string]struct{} {
	out := make(map[string]struct{}, len(links))
	for _, l := range links {
		out[canonicalLinkKey(l.StorageA, l.ColumnA, l.StorageB, l.ColumnB)] = struct{}{}
	}
	return out
}

func canonicalLinkKey(storageA, colA, storageB, colB string) string {
	a := storageA + "." + colA
	b := storageB + "." + colB
	if a > b {
		a, b = b, a
	}
	return a + "|" + b
}

func scoreSet(observed, expected map[string]struct{}) (precision, recall, f1 float64, tp, fp, fn int) {
	for k := range observed {
		if _, ok := expected[k]; ok {
			tp++
		} else {
			fp++
		}
	}
	for k := range expected {
		if _, ok := observed[k]; !ok {
			fn++
		}
	}
	if tp+fp > 0 {
		precision = float64(tp) / float64(tp+fp)
	}
	if tp+fn > 0 {
		recall = float64(tp) / float64(tp+fn)
	}
	if precision+recall > 0 {
		f1 = 2 * precision * recall / (precision + recall)
	}
	return precision, recall, f1, tp, fp, fn
}

func sortedKeys(m map[string]struct{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func TestSyntheticLinkingInvariantToColumnFormatAndNumericEncoding(t *testing.T) {
	leftRows := make([]map[string]interface{}, 0, 120)
	rightRows := make([]map[string]interface{}, 0, 120)
	for i := 1; i <= 120; i++ {
		leftRows = append(leftRows, map[string]interface{}{
			"customer_id": i,
			"region_code": fmt.Sprintf("R%02d", i%5),
		})
		rightRows = append(rightRows, map[string]interface{}{
			"CustomerID": fmt.Sprintf("%d.0", i),
			"segment":    []string{"enterprise", "smb", "public"}[i%3],
		})
	}

	left := makeCIR("crm", "CustomerRecord", leftRows)
	right := makeCIR("billing", "InvoiceRecord", rightRows)
	profiles := append([]models.ColumnProfile{}, BuildColumnProfilesFromCIRs("crm", []*models.CIR{left})...)
	profiles = append(profiles, BuildColumnProfilesFromCIRs("billing", []*models.CIR{right})...)

	links := DetectCrossSourceLinks(profiles)
	if len(links) == 0 {
		t.Fatal("expected links for numeric-format/casing invariant dataset")
	}

	pred := observedLinkKeys(links)
	want := map[string]struct{}{canonicalLinkKey("crm", "customer_id", "billing", "CustomerID"): {}}
	precision, recall, f1, tp, fp, fn := scoreSet(pred, want)
	if precision < 0.90 || recall < 1.0 || f1 < 0.90 {
		t.Fatalf("invariance link quality below threshold: precision=%.3f recall=%.3f f1=%.3f tp=%d fp=%d fn=%d\nwant=%v\ngot=%v",
			precision, recall, f1, tp, fp, fn, sortedKeys(want), sortedKeys(pred))
	}
}

func TestSyntheticMixedModeInvariantToTextNoise(t *testing.T) {
	records := []extractionRecord{
		makeRecord(map[string]string{"author": "jane_doe", "content": "FYI: OpenAI Summit!!! in San Francisco, with Sam Altman -- incredible."}),
		makeRecord(map[string]string{"author": "tech_blogger", "content": "openai summit @ san francisco: sam altman keynote recap..."}),
		makeRecord(map[string]string{"author": "ai_watcher", "content": "OpenAI Summit (San Francisco) had Sam Altman and Greg Brockman."}),
		makeRecord(map[string]string{"author": "dev_news", "content": "Breaking: OpenAI Summit, San Francisco Moscone Center, Sam Altman speaking."}),
		makeRecord(map[string]string{"author": "ml_researcher", "content": "Great talks at OpenAI Summit; met Greg Brockman in San Francisco."}),
	}

	result := extractFromRecords(records)
	if len(result.Entities) == 0 {
		t.Fatal("expected entities from noisy text records")
	}

	wants := []string{"OpenAI Summit", "San Francisco", "Sam Altman"}
	found := 0
	for _, want := range wants {
		if containsEntity(result, want) {
			found++
		}
	}
	coverage := float64(found) / float64(len(wants))
	if coverage < 0.66 {
		t.Fatalf("noisy text coverage below threshold: %.3f (%d/%d), entities=%v", coverage, found, len(wants), entityNames(result))
	}

	structured := syntheticStructuredCIRForDomain("media")
	structuredResult, err := ExtractFromStructuredCIR(structured)
	if err != nil {
		t.Fatalf("ExtractFromStructuredCIR failed: %v", err)
	}
	reconciled := ReconcileEntities(structuredResult, result)
	if len(reconciled.Entities) == 0 {
		t.Fatal("expected reconciled entities for noisy mixed-mode dataset")
	}
}

func TestSyntheticCrossSourceAdversarial_NoFalseJoinOnCategoricals(t *testing.T) {
	leftRows := make([]map[string]interface{}, 0, 400)
	rightRows := make([]map[string]interface{}, 0, 400)
	for i := 1; i <= 400; i++ {
		leftRows = append(leftRows, map[string]interface{}{
			"ticket_id": fmt.Sprintf("TCK-%06d", i),
			"status":    []string{"open", "closed"}[i%2],
			"priority":  []string{"low", "high"}[i%2],
		})
		rightRows = append(rightRows, map[string]interface{}{
			"incident_id": fmt.Sprintf("INC-%06d", i),
			"status":      []string{"open", "closed"}[i%2],
			"priority":    []string{"low", "high"}[i%2],
		})
	}

	left := makeCIR("support", "TicketRecord", leftRows)
	right := makeCIR("ops", "IncidentRecord", rightRows)
	profiles := append([]models.ColumnProfile{}, BuildColumnProfilesFromCIRs("support", []*models.CIR{left})...)
	profiles = append(profiles, BuildColumnProfilesFromCIRs("ops", []*models.CIR{right})...)

	links := DetectCrossSourceLinks(profiles)
	for _, l := range links {
		pair := strings.ToLower(l.ColumnA + "|" + l.ColumnB)
		if strings.Contains(pair, "status") || strings.Contains(pair, "priority") {
			t.Fatalf("categorical false positive detected: %+v", l)
		}
	}
}

func TestSyntheticCrossSourceAdversarial_RejectOneSidedKeyJoins(t *testing.T) {
	leftRows := make([]map[string]interface{}, 0, 180)
	rightRows := make([]map[string]interface{}, 0, 180)
	for i := 1; i <= 180; i++ {
		id := fmt.Sprintf("%d", i)
		leftRows = append(leftRows, map[string]interface{}{
			"customer_id": id,
			"channel":     []string{"retail", "online", "partner"}[i%3],
		})
		rightRows = append(rightRows, map[string]interface{}{
			"channel": []string{"retail", "online", "partner"}[i%3],
			"region":  []string{"na", "emea", "apac"}[i%3],
		})
	}

	left := makeCIR("sales", "CustomerRecord", leftRows)
	right := makeCIR("marketing", "CampaignRecord", rightRows)
	profiles := append([]models.ColumnProfile{}, BuildColumnProfilesFromCIRs("sales", []*models.CIR{left})...)
	profiles = append(profiles, BuildColumnProfilesFromCIRs("marketing", []*models.CIR{right})...)

	links := DetectCrossSourceLinks(profiles)
	for _, l := range links {
		if strings.EqualFold(l.ColumnA, "channel") || strings.EqualFold(l.ColumnB, "channel") {
			t.Fatalf("one-sided key gate failed; unexpected channel link: %+v", l)
		}
	}
}

func TestSyntheticCrossSourceConfidenceCalibration(t *testing.T) {
	profiles, expected := syntheticCalibrationProfiles()
	links := DetectCrossSourceLinks(profiles)
	if len(links) == 0 {
		t.Fatal("expected links for calibration dataset")
	}

	type check struct {
		threshold  float64
		minPrec    float64
		minSupport int
	}
	checks := []check{
		{threshold: 0.35, minPrec: 0.35, minSupport: 6},
		{threshold: 0.55, minPrec: 0.45, minSupport: 4},
		{threshold: 0.75, minPrec: 0.95, minSupport: 3},
	}

	observedPrec := make([]float64, 0, len(checks))
	for _, c := range checks {
		prec, support, pred := precisionAtThreshold(links, expected, c.threshold)
		if support < c.minSupport {
			t.Fatalf("threshold %.2f has insufficient support: got=%d want>=%d", c.threshold, support, c.minSupport)
		}
		if prec < c.minPrec {
			t.Fatalf("threshold %.2f precision below target: got=%.3f want>=%.3f; predicted=%v", c.threshold, prec, c.minPrec, sortedKeys(pred))
		}
		observedPrec = append(observedPrec, prec)
	}

	if observedPrec[len(observedPrec)-1] < observedPrec[0]+0.30 {
		t.Fatalf("calibration slope too weak: low-threshold precision=%.3f high-threshold precision=%.3f", observedPrec[0], observedPrec[len(observedPrec)-1])
	}
}

func syntheticCalibrationProfiles() ([]models.ColumnProfile, map[string]struct{}) {
	alphaRows := make([]map[string]interface{}, 0, 200)
	betaRows := make([]map[string]interface{}, 0, 200)
	gammaRows := make([]map[string]interface{}, 0, 200)

	for i := 1; i <= 200; i++ {
		recordID := fmt.Sprintf("R-%04d", i)
		customerRef := fmt.Sprintf("CUST-%04d", i)
		partnerRef := fmt.Sprintf("R-%04d", 1+(i%100))
		accountKey := fmt.Sprintf("R-%04d", 1+(i%140))

		alphaRows = append(alphaRows, map[string]interface{}{
			"record_id":    recordID,
			"customer_ref": customerRef,
		})
		betaRows = append(betaRows, map[string]interface{}{
			"record_id":   recordID,
			"partner_ref": partnerRef,
		})
		gammaRows = append(gammaRows, map[string]interface{}{
			"record_id":   recordID,
			"account_key": accountKey,
		})
	}

	alpha := makeCIR("alpha", "AlphaRecord", alphaRows)
	beta := makeCIR("beta", "BetaRecord", betaRows)
	gamma := makeCIR("gamma", "GammaRecord", gammaRows)

	profiles := append([]models.ColumnProfile{}, BuildColumnProfilesFromCIRs("alpha", []*models.CIR{alpha})...)
	profiles = append(profiles, BuildColumnProfilesFromCIRs("beta", []*models.CIR{beta})...)
	profiles = append(profiles, BuildColumnProfilesFromCIRs("gamma", []*models.CIR{gamma})...)

	expected := map[string]struct{}{
		canonicalLinkKey("alpha", "record_id", "beta", "record_id"):  {},
		canonicalLinkKey("alpha", "record_id", "gamma", "record_id"): {},
		canonicalLinkKey("beta", "record_id", "gamma", "record_id"):  {},
	}
	return profiles, expected
}

func precisionAtThreshold(links []models.CrossSourceLink, expected map[string]struct{}, threshold float64) (precision float64, support int, predicted map[string]struct{}) {
	predicted = make(map[string]struct{})
	for _, l := range links {
		if l.Confidence >= threshold {
			predicted[canonicalLinkKey(l.StorageA, l.ColumnA, l.StorageB, l.ColumnB)] = struct{}{}
		}
	}
	if len(predicted) == 0 {
		return 0, 0, predicted
	}
	tp := 0
	for k := range predicted {
		if _, ok := expected[k]; ok {
			tp++
		}
	}
	support = len(predicted)
	precision = float64(tp) / float64(support)
	return precision, support, predicted
}

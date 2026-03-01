package extraction

import (
	"fmt"
	"testing"

	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

// ─── Helpers ──────────────────────────────────────────────────────────────────

// makeCIR constructs a minimal CIR with an array-of-maps payload.
func makeCIR(storageID, entityType string, rows []map[string]interface{}) *models.CIR {
	data := make([]interface{}, len(rows))
	for i, r := range rows {
		data[i] = r
	}
	cir := models.NewCIR(
		models.SourceTypeDatabase,
		"storage://"+storageID+"/"+entityType,
		models.DataFormatJSON,
		data,
	)
	cir.SetParameter("entity_type", entityType)
	return cir
}

// studentRows generates N student rows for a grades-style dataset.
func studentGradeRows(n int) []map[string]interface{} {
	subjects := []string{"Mathematics", "English", "Science", "History", "Art"}
	grades := []string{"A", "B", "C", "D"}
	rows := make([]map[string]interface{}, 0, n*len(subjects))
	for i := 1; i <= n; i++ {
		for _, subj := range subjects {
			score := 50 + (i*7+len(subj))%50
			rows = append(rows, map[string]interface{}{
				"student_id": fmt.Sprintf("%d", i),
				"name":       fmt.Sprintf("Student_%d", i),
				"subject":    subj,
				"score":      score,
				"grade":      grades[score%4],
				"semester":   "Fall 2024",
			})
		}
	}
	return rows
}

// attendanceRows generates one attendance row per student.
func attendanceRows(n int) []map[string]interface{} {
	rows := make([]map[string]interface{}, n)
	for i := 1; i <= n; i++ {
		present := 40 + i%10
		absent := 50 - present
		rows[i-1] = map[string]interface{}{
			"student_id":      fmt.Sprintf("%d", i),
			"days_present":    present,
			"days_absent":     absent,
			"attendance_rate": float64(present) / 50.0,
			"academic_year":   "2024",
			"status":          "active",
		}
	}
	return rows
}

// advisorRows uses "sid" instead of "student_id" — a non-obvious name mismatch.
func advisorRows(n int) []map[string]interface{} {
	advisors := []string{"Dr. Smith", "Dr. Patel", "Dr. Kim"}
	rows := make([]map[string]interface{}, n)
	for i := 1; i <= n; i++ {
		rows[i-1] = map[string]interface{}{
			"sid":         fmt.Sprintf("%d", i), // intentionally different name
			"advisor":     advisors[i%3],
			"year_level":  []string{"Freshman", "Sophomore", "Junior", "Senior"}[i%4],
			"support_plan": i%5 == 0, // only 20% have support plans
		}
	}
	return rows
}

// ─── Test: obvious cross-source link (same column name, same values) ──────────

func TestDetectCrossSourceLinks_ObviousLink(t *testing.T) {
	const n = 30

	gradesCIR := makeCIR("grades_db", "GradeRecord", studentGradeRows(n))
	attendanceCIR := makeCIR("attendance_db", "AttendanceRecord", attendanceRows(n))

	profilesA := BuildColumnProfilesFromCIRs("grades_db", []*models.CIR{gradesCIR})
	profilesB := BuildColumnProfilesFromCIRs("attendance_db", []*models.CIR{attendanceCIR})

	allProfiles := append(profilesA, profilesB...)
	links := DetectCrossSourceLinks(allProfiles)

	// Must find at least one link, and the highest-confidence link should
	// involve student_id from both sources.
	if len(links) == 0 {
		t.Fatal("expected cross-source links, got none")
	}

	var studentIDLink *models.CrossSourceLink
	for i := range links {
		l := &links[i]
		aMatch := l.ColumnA == "student_id" || l.ColumnB == "student_id"
		bMatch := l.ColumnA == "student_id" || l.ColumnB == "student_id"
		if aMatch && bMatch &&
			((l.StorageA == "grades_db" && l.StorageB == "attendance_db") ||
				(l.StorageA == "attendance_db" && l.StorageB == "grades_db")) {
			studentIDLink = l
			break
		}
	}

	if studentIDLink == nil {
		t.Errorf("expected a link on student_id between grades_db and attendance_db; found links: %+v", links)
	} else {
		if studentIDLink.Confidence < 0.70 {
			t.Errorf("student_id link confidence too low: got %.3f, want >= 0.70", studentIDLink.Confidence)
		}
		if studentIDLink.ValueOverlap < 0.80 {
			t.Errorf("student_id value overlap too low: got %.3f, want >= 0.80", studentIDLink.ValueOverlap)
		}
		t.Logf("student_id link: confidence=%.3f, valueOverlap=%.3f, nameSim=%.3f, shared=%d",
			studentIDLink.Confidence, studentIDLink.ValueOverlap, studentIDLink.NameSimilarity, studentIDLink.SharedValueCount)
	}
}

// ─── Test: non-obvious link (different column name, same values) ──────────────

func TestDetectCrossSourceLinks_NonObviousNameMismatch(t *testing.T) {
	const n = 25

	gradesCIR := makeCIR("grades_db", "GradeRecord", studentGradeRows(n))
	advisorCIR := makeCIR("advisor_db", "AdvisorRecord", advisorRows(n))

	profilesA := BuildColumnProfilesFromCIRs("grades_db", []*models.CIR{gradesCIR})
	profilesB := BuildColumnProfilesFromCIRs("advisor_db", []*models.CIR{advisorCIR})

	links := DetectCrossSourceLinks(append(profilesA, profilesB...))

	// grades_db.student_id ↔ advisor_db.sid should be detected despite the
	// name mismatch because value overlap is the primary signal.
	var sidLink *models.CrossSourceLink
	for i := range links {
		l := &links[i]
		hasStudentID := l.ColumnA == "student_id" || l.ColumnB == "student_id"
		hasSid := l.ColumnA == "sid" || l.ColumnB == "sid"
		if hasStudentID && hasSid {
			sidLink = l
			break
		}
	}

	if sidLink == nil {
		t.Errorf("expected student_id ↔ sid link to be detected via value overlap; found links: %+v", links)
	} else {
		if sidLink.NameSimilarity > 0.5 {
			t.Errorf("name similarity should be low for student_id/sid mismatch, got %.3f", sidLink.NameSimilarity)
		}
		if sidLink.ValueOverlap < 0.60 {
			t.Errorf("value overlap too low for student_id/sid: got %.3f, want >= 0.60", sidLink.ValueOverlap)
		}
		t.Logf("sid link: confidence=%.3f, valueOverlap=%.3f, nameSim=%.3f",
			sidLink.Confidence, sidLink.ValueOverlap, sidLink.NameSimilarity)
	}
}

// ─── Test: false positive rejection for low-cardinality categorical columns ───

func TestDetectCrossSourceLinks_NoFalsePositiveForCategoricals(t *testing.T) {
	// Both sources have a "status" column with the same small vocabulary.
	// This should NOT be detected as a cross-source link.
	const n = 200

	gradeRows := make([]map[string]interface{}, n)
	attendRows := make([]map[string]interface{}, n)
	for i := 0; i < n; i++ {
		gradeRows[i] = map[string]interface{}{
			"student_id": fmt.Sprintf("%d", i+1),
			"grade":      []string{"A", "B", "C", "D"}[i%4],
			"status":     []string{"active", "inactive"}[i%2],
		}
		attendRows[i] = map[string]interface{}{
			"student_id":      fmt.Sprintf("%d", i+1),
			"attendance_rate": float64(i%100) / 100.0,
			"status":          []string{"active", "inactive"}[i%2],
		}
	}

	gradesCIR := makeCIR("grades_db", "GradeRecord", gradeRows)
	attendCIR := makeCIR("attendance_db", "AttendanceRecord", attendRows)

	profilesA := BuildColumnProfilesFromCIRs("grades_db", []*models.CIR{gradesCIR})
	profilesB := BuildColumnProfilesFromCIRs("attendance_db", []*models.CIR{attendCIR})

	links := DetectCrossSourceLinks(append(profilesA, profilesB...))

	// student_id link is expected (and correct).
	// "status" and "grade" links should NOT appear.
	for _, l := range links {
		colsInvolved := []string{l.ColumnA, l.ColumnB}
		for _, col := range colsInvolved {
			if col == "status" || col == "grade" {
				t.Errorf("false positive: column %q (low cardinality categorical) was linked: %+v", col, l)
			}
		}
	}

	// Verify student_id is still detected.
	found := false
	for _, l := range links {
		if (l.ColumnA == "student_id" || l.ColumnB == "student_id") &&
			(l.StorageA != l.StorageB) {
			found = true
		}
	}
	if !found {
		t.Error("student_id cross-source link should still be detected")
	}
}

// ─── Test: single storage produces no links ───────────────────────────────────

func TestDetectCrossSourceLinks_SingleStorage_NoLinks(t *testing.T) {
	gradesCIR := makeCIR("grades_db", "GradeRecord", studentGradeRows(20))
	profiles := BuildColumnProfilesFromCIRs("grades_db", []*models.CIR{gradesCIR})

	links := DetectCrossSourceLinks(profiles)
	if len(links) != 0 {
		t.Errorf("single storage should produce no cross-source links; got %d", len(links))
	}
}

// ─── Test: three sources, links detected across all pairs ─────────────────────

func TestDetectCrossSourceLinks_ThreeSources(t *testing.T) {
	const n = 20

	gradesCIR := makeCIR("grades_db", "GradeRecord", studentGradeRows(n))
	attendanceCIR := makeCIR("attendance_db", "AttendanceRecord", attendanceRows(n))
	advisorCIR := makeCIR("advisor_db", "AdvisorRecord", advisorRows(n))

	var profiles []models.ColumnProfile
	profiles = append(profiles, BuildColumnProfilesFromCIRs("grades_db", []*models.CIR{gradesCIR})...)
	profiles = append(profiles, BuildColumnProfilesFromCIRs("attendance_db", []*models.CIR{attendanceCIR})...)
	profiles = append(profiles, BuildColumnProfilesFromCIRs("advisor_db", []*models.CIR{advisorCIR})...)

	links := DetectCrossSourceLinks(profiles)

	if len(links) < 2 {
		t.Errorf("expected at least 2 cross-source links for 3 sources; got %d: %+v", len(links), links)
	}

	// Check that different storage pairs are represented.
	storagePairs := make(map[string]bool)
	for _, l := range links {
		pair := l.StorageA + "|" + l.StorageB
		storagePairs[pair] = true
		t.Logf("link: %s.%s ↔ %s.%s  conf=%.3f overlap=%.3f nameSim=%.3f",
			l.StorageA, l.ColumnA, l.StorageB, l.ColumnB,
			l.Confidence, l.ValueOverlap, l.NameSimilarity)
	}
	if len(storagePairs) < 2 {
		t.Errorf("expected links across at least 2 storage pairs; got pairs: %v", storagePairs)
	}
}

// ─── Test: column name tokenisation ──────────────────────────────────────────

func TestNormaliseColumnTokens(t *testing.T) {
	cases := []struct {
		input    string
		wantToks []string
	}{
		{"student_id", []string{"student", "id"}},
		{"studentId", []string{"student", "id"}},
		{"StudentID", []string{"student", "id"}},
		{"STUDENT_ID", []string{"student", "id"}},
		{"sid", []string{"sid"}},
		{"email", []string{"email"}},
		{"days_present", []string{"days", "present"}},
		{"attendanceRate", []string{"attendance", "rate"}},
	}

	for _, tc := range cases {
		got := normaliseColumnTokens(tc.input)
		if len(got) != len(tc.wantToks) {
			t.Errorf("normaliseColumnTokens(%q) = %v, want %v", tc.input, got, tc.wantToks)
			continue
		}
		for i, tok := range got {
			if tok != tc.wantToks[i] {
				t.Errorf("normaliseColumnTokens(%q)[%d] = %q, want %q", tc.input, i, tok, tc.wantToks[i])
			}
		}
	}
}

// ─── Test: column name similarity ────────────────────────────────────────────

func TestColumnNameSimilarity(t *testing.T) {
	cases := []struct {
		a, b        string
		wantAtLeast float64
		wantAtMost  float64
	}{
		{"student_id", "student_id", 1.0, 1.0},
		{"student_id", "studentId", 1.0, 1.0},
		{"student_id", "StudentID", 1.0, 1.0},
		{"student_id", "sid", 0.0, 0.3},   // low: no shared tokens
		{"email", "email_address", 0.3, 0.8}, // partial match
		{"grade", "grade_score", 0.3, 0.8},
	}

	for _, tc := range cases {
		got := columnNameSimilarity(tc.a, tc.b)
		if got < tc.wantAtLeast || got > tc.wantAtMost {
			t.Errorf("columnNameSimilarity(%q, %q) = %.3f, want [%.3f, %.3f]",
				tc.a, tc.b, got, tc.wantAtLeast, tc.wantAtMost)
		}
	}
}

// ─── Test: key column detection ───────────────────────────────────────────────

func TestIsKeyColumn(t *testing.T) {
	cases := []struct {
		name    string
		card    float64
		wantKey bool
	}{
		{"student_id", 1.0, true},    // high cardinality + name token
		{"student_id", 0.1, true},    // low cardinality but name token
		{"sid", 1.0, true},           // high cardinality (≥ 0.60)
		{"sid", 0.5, false},          // below threshold, no key token
		{"email", 0.1, true},         // key name token
		{"status", 0.01, false},      // low cardinality, no key token
		{"grade", 0.04, false},       // low cardinality categorical
		{"score", 0.8, true},         // high cardinality (≥ 0.60)
		{"subject", 0.01, false},     // low cardinality
		{"days_present", 0.5, false}, // below threshold, no key token
	}

	for _, tc := range cases {
		got := isKeyColumn(tc.name, tc.card)
		if got != tc.wantKey {
			t.Errorf("isKeyColumn(%q, %.2f) = %v, want %v", tc.name, tc.card, got, tc.wantKey)
		}
	}
}

// ─── Test: BuildColumnProfilesFromCIRs ───────────────────────────────────────

func TestBuildColumnProfilesFromCIRs(t *testing.T) {
	rows := studentGradeRows(50)
	cir := makeCIR("grades_db", "GradeRecord", rows)
	profiles := BuildColumnProfilesFromCIRs("grades_db", []*models.CIR{cir})

	if len(profiles) == 0 {
		t.Fatal("expected profiles, got none")
	}

	// Verify that student_id is detected as a likely key (high cardinality).
	var sidProfile *models.ColumnProfile
	for i := range profiles {
		if profiles[i].ColumnName == "student_id" {
			sidProfile = &profiles[i]
			break
		}
	}
	if sidProfile == nil {
		t.Fatal("student_id column profile not found")
	}
	if !sidProfile.IsLikelyKey {
		t.Errorf("student_id should be flagged as likely key (cardinality=%.2f)", sidProfile.CardinalityRatio)
	}
	if len(sidProfile.ValueSample) == 0 {
		t.Error("student_id value sample should not be empty")
	}
	t.Logf("student_id profile: cardinality=%.3f, unique=%d, isKey=%v",
		sidProfile.CardinalityRatio, sidProfile.UniqueCount, sidProfile.IsLikelyKey)

	// Verify that "subject" (low cardinality) is NOT flagged as a key.
	var subjProfile *models.ColumnProfile
	for i := range profiles {
		if profiles[i].ColumnName == "subject" {
			subjProfile = &profiles[i]
			break
		}
	}
	if subjProfile != nil && subjProfile.IsLikelyKey {
		t.Errorf("subject (low cardinality categorical) should NOT be flagged as a key")
	}
}

// ─── Test: Jaccard overlap ────────────────────────────────────────────────────

func TestJaccardOverlap(t *testing.T) {
	cases := []struct {
		a, b         map[string]bool
		wantOverlap  float64
		wantShared   int
	}{
		{
			map[string]bool{"1": true, "2": true, "3": true},
			map[string]bool{"1": true, "2": true, "3": true},
			1.0, 3,
		},
		{
			map[string]bool{"1": true, "2": true},
			map[string]bool{"3": true, "4": true},
			0.0, 0,
		},
		{
			map[string]bool{"1": true, "2": true, "3": true, "4": true},
			map[string]bool{"3": true, "4": true, "5": true, "6": true},
			// intersection={3,4}=2, union={1,2,3,4,5,6}=6
			2.0 / 6.0, 2,
		},
	}

	for i, tc := range cases {
		overlap, shared := jaccardOverlap(tc.a, tc.b)
		if abs(overlap-tc.wantOverlap) > 0.001 {
			t.Errorf("case %d: overlap = %.3f, want %.3f", i, overlap, tc.wantOverlap)
		}
		if shared != tc.wantShared {
			t.Errorf("case %d: shared = %d, want %d", i, shared, tc.wantShared)
		}
	}
}

func abs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}

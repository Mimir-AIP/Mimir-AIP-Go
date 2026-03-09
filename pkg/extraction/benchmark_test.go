package extraction

import (
	"testing"

	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

func BenchmarkExtractFromStructuredCIR_10kRows(b *testing.B) {
	rows := studentGradeRows(2000) // 10k rows (5 subjects each)
	cir := makeCIR("grades-bench", "GradeRecord", rows)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := ExtractFromStructuredCIR(cir)
		if err != nil {
			b.Fatalf("ExtractFromStructuredCIR failed: %v", err)
		}
		if len(result.Entities) == 0 {
			b.Fatal("expected extracted entities")
		}
	}
}

func BenchmarkBuildColumnProfilesFromCIRs_30kRows(b *testing.B) {
	grades := makeCIR("grades-bench", "GradeRecord", studentGradeRows(2000))
	attendance := makeCIR("attendance-bench", "AttendanceRecord", attendanceRows(10000))
	advisor := makeCIR("advisor-bench", "AdvisorRecord", advisorRows(10000))
	cirs := []*models.CIR{grades, attendance, advisor}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		profiles := BuildColumnProfilesFromCIRs("bench-source", cirs)
		if len(profiles) == 0 {
			b.Fatal("expected column profiles")
		}
	}
}

func BenchmarkDetectCrossSourceLinks_Profiles(b *testing.B) {
	grades := makeCIR("grades-bench", "GradeRecord", studentGradeRows(2000))
	attendance := makeCIR("attendance-bench", "AttendanceRecord", attendanceRows(10000))
	advisor := makeCIR("advisor-bench", "AdvisorRecord", advisorRows(10000))

	profiles := append([]models.ColumnProfile{}, BuildColumnProfilesFromCIRs("grades-bench", []*models.CIR{grades})...)
	profiles = append(profiles, BuildColumnProfilesFromCIRs("attendance-bench", []*models.CIR{attendance})...)
	profiles = append(profiles, BuildColumnProfilesFromCIRs("advisor-bench", []*models.CIR{advisor})...)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		links := DetectCrossSourceLinks(profiles)
		if len(links) == 0 {
			b.Fatal("expected cross-source links")
		}
	}
}

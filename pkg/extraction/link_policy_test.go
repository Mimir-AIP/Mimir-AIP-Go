package extraction

import (
	"testing"

	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

func TestLinkPolicyNormalize(t *testing.T) {
	policy := LinkPolicy{ReviewThreshold: -0.3, AutoAcceptThreshold: 1.9}.Normalize()
	if policy.ReviewThreshold != 0 {
		t.Fatalf("expected normalized review threshold 0, got %f", policy.ReviewThreshold)
	}
	if policy.AutoAcceptThreshold != 1 {
		t.Fatalf("expected normalized auto-accept threshold 1, got %f", policy.AutoAcceptThreshold)
	}

	policy = LinkPolicy{ReviewThreshold: 0.7, AutoAcceptThreshold: 0.6}.Normalize()
	if policy.AutoAcceptThreshold != policy.ReviewThreshold {
		t.Fatalf("expected auto-accept threshold to be raised to review threshold, got review=%f auto=%f", policy.ReviewThreshold, policy.AutoAcceptThreshold)
	}
}

func TestDecideCrossSourceLink(t *testing.T) {
	policy := DefaultLinkPolicy()
	tests := []struct {
		name string
		conf float64
		want LinkDecision
	}{
		{name: "reject low confidence", conf: 0.20, want: LinkReject},
		{name: "needs review mid confidence", conf: 0.55, want: LinkNeedsReview},
		{name: "auto accept high confidence", conf: 0.83, want: LinkAutoAccept},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			decision := DecideCrossSourceLink(models.CrossSourceLink{Confidence: tc.conf}, policy)
			if decision != tc.want {
				t.Fatalf("unexpected decision: got=%s want=%s", decision, tc.want)
			}
		})
	}
}

func TestFilterLinksByDecision(t *testing.T) {
	policy := DefaultLinkPolicy()
	links := []models.CrossSourceLink{
		{StorageA: "a", ColumnA: "id", StorageB: "b", ColumnB: "id", Confidence: 0.88},
		{StorageA: "a", ColumnA: "key", StorageB: "c", ColumnB: "ref", Confidence: 0.57},
		{StorageA: "a", ColumnA: "status", StorageB: "d", ColumnB: "status", Confidence: 0.22},
	}

	auto := FilterLinksByDecision(links, policy, LinkAutoAccept)
	if len(auto) != 1 || auto[0].Confidence != 0.88 {
		t.Fatalf("expected exactly one auto-accept link, got %+v", auto)
	}

	review := FilterLinksByDecision(links, policy, LinkNeedsReview)
	if len(review) != 1 || review[0].Confidence != 0.57 {
		t.Fatalf("expected exactly one needs-review link, got %+v", review)
	}

	reject := FilterLinksByDecision(links, policy, LinkReject)
	if len(reject) != 1 || reject[0].Confidence != 0.22 {
		t.Fatalf("expected exactly one reject link, got %+v", reject)
	}
}

package extraction

import "github.com/mimir-aip/mimir-aip-go/pkg/models"

// LinkDecision is the action recommended for a detected cross-source link.
type LinkDecision string

const (
	// LinkReject means confidence is too low and link should be ignored.
	LinkReject LinkDecision = "reject"
	// LinkNeedsReview means confidence is moderate and should be reviewed by a human.
	LinkNeedsReview LinkDecision = "needs_review"
	// LinkAutoAccept means confidence is high enough for automatic downstream use.
	LinkAutoAccept LinkDecision = "auto_accept"
)

// LinkPolicy defines confidence thresholds used to classify detected links.
type LinkPolicy struct {
	// ReviewThreshold is the minimum confidence for a link to be surfaced for review.
	ReviewThreshold float64
	// AutoAcceptThreshold is the minimum confidence for a link to be auto-accepted.
	AutoAcceptThreshold float64
}

// DefaultLinkPolicy returns the default confidence policy calibrated against synthetic evaluations.
func DefaultLinkPolicy() LinkPolicy {
	return LinkPolicy{
		ReviewThreshold:     0.35,
		AutoAcceptThreshold: 0.75,
	}
}

// Normalize validates and normalizes a policy to a safe deterministic range.
func (p LinkPolicy) Normalize() LinkPolicy {
	if p.ReviewThreshold < 0 {
		p.ReviewThreshold = 0
	}
	if p.ReviewThreshold > 1 {
		p.ReviewThreshold = 1
	}
	if p.AutoAcceptThreshold < 0 {
		p.AutoAcceptThreshold = 0
	}
	if p.AutoAcceptThreshold > 1 {
		p.AutoAcceptThreshold = 1
	}
	if p.AutoAcceptThreshold < p.ReviewThreshold {
		p.AutoAcceptThreshold = p.ReviewThreshold
	}
	return p
}

// DecideCrossSourceLink returns the recommended action for a link under the given policy.
func DecideCrossSourceLink(link models.CrossSourceLink, policy LinkPolicy) LinkDecision {
	normalized := policy.Normalize()
	if link.Confidence >= normalized.AutoAcceptThreshold {
		return LinkAutoAccept
	}
	if link.Confidence >= normalized.ReviewThreshold {
		return LinkNeedsReview
	}
	return LinkReject
}

// FilterLinksByDecision returns links that map to the requested decision under the policy.
func FilterLinksByDecision(links []models.CrossSourceLink, policy LinkPolicy, decision LinkDecision) []models.CrossSourceLink {
	if len(links) == 0 {
		return nil
	}
	result := make([]models.CrossSourceLink, 0, len(links))
	for _, link := range links {
		if DecideCrossSourceLink(link, policy) == decision {
			result = append(result, link)
		}
	}
	return result
}

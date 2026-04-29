package pluginruntime

import (
	"fmt"
	"net/url"
	"strings"
)

// ValidateRepositoryURL rejects local filesystem paths and unsupported URL schemes
// before the runtime invokes git. HTTPS and SSH-style Git URLs are allowed.
func ValidateRepositoryURL(raw string) error {
	value := strings.TrimSpace(raw)
	if value == "" {
		return fmt.Errorf("repository URL is required")
	}
	if strings.HasPrefix(value, "git@") && strings.Contains(value, ":") {
		return nil
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return fmt.Errorf("invalid repository URL: %w", err)
	}
	switch parsed.Scheme {
	case "https", "ssh", "git+ssh":
		if parsed.Host == "" {
			return fmt.Errorf("repository URL host is required")
		}
		return nil
	default:
		return fmt.Errorf("unsupported repository URL scheme %q; use https or ssh", parsed.Scheme)
	}
}

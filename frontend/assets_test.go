package frontendassets

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandlerServesEmbeddedIndexWithoutCDNs(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for index, got %d", rec.Code)
	}
	body := rec.Body.String()
	for _, disallowed := range []string{"https://unpkg.com", "https://cdn.jsdelivr.net", "https://fonts.googleapis.com"} {
		if strings.Contains(body, disallowed) {
			t.Fatalf("expected embedded index to avoid external dependency %q", disallowed)
		}
	}
	for _, expected := range []string{"vendor/react.production.min.js", "vendor/react-dom.production.min.js", "vendor/babel.min.js", "vendor/chart.umd.min.js"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected embedded index to reference %q", expected)
		}
	}
}

func TestHandlerServesVendoredRuntimeAsset(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/vendor/react.production.min.js", nil)
	rec := httptest.NewRecorder()

	Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for vendored asset, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "react.production.min.js") {
		t.Fatalf("expected vendored React asset content, got %q", rec.Body.String()[:min(80, len(rec.Body.String()))])
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

package llm

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
)

// ── Mock provider ──────────────────────────────────────────────────────────────

type mockProvider struct {
	name         string
	listModels   func(ctx context.Context) ([]Model, error)
	complete     func(ctx context.Context, req CompletionRequest) (CompletionResponse, error)
	listCalls    atomic.Int32
}

func (m *mockProvider) Name() string { return m.name }

func (m *mockProvider) ListModels(ctx context.Context) ([]Model, error) {
	m.listCalls.Add(1)
	if m.listModels != nil {
		return m.listModels(ctx)
	}
	return []Model{{ID: "test-model", Name: "Test Model", ProviderName: m.name}}, nil
}

func (m *mockProvider) Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error) {
	if m.complete != nil {
		return m.complete(ctx, req)
	}
	return CompletionResponse{Content: `{"Alice Johnson":"Person"}`}, nil
}

// ── Tests ─────────────────────────────────────────────────────────────────────

func TestIsEnabled_NilService(t *testing.T) {
	var s *Service
	if s.IsEnabled() {
		t.Error("nil Service.IsEnabled() should return false")
	}
}

func TestIsEnabled_NoProvider(t *testing.T) {
	s := NewService(nil, "", true)
	if s.IsEnabled() {
		t.Error("Service with nil provider should not be enabled")
	}
}

func TestIsEnabled_DisabledFlag(t *testing.T) {
	s := NewService(&mockProvider{name: "mock"}, "test", false)
	if s.IsEnabled() {
		t.Error("Service with enabled=false should report IsEnabled()=false")
	}
}

func TestProviderName_NilService(t *testing.T) {
	var s *Service
	if got := s.ProviderName(); got != "none" {
		t.Errorf("nil Service.ProviderName() = %q, want %q", got, "none")
	}
}

func TestListModels_Cache(t *testing.T) {
	mock := &mockProvider{name: "mock"}
	s := NewService(mock, "test-model", true)

	ctx := context.Background()

	// First call — must hit the provider.
	models1, err := s.ListModels(ctx)
	if err != nil {
		t.Fatalf("ListModels first call: %v", err)
	}
	if len(models1) == 0 {
		t.Fatal("expected at least one model")
	}

	// Second call — must use the cache; provider call count must stay at 1.
	_, err = s.ListModels(ctx)
	if err != nil {
		t.Fatalf("ListModels second call: %v", err)
	}

	if calls := mock.listCalls.Load(); calls != 1 {
		t.Errorf("ListModels called provider %d times, want 1", calls)
	}
}

func TestLabelEntityTypes_Disabled(t *testing.T) {
	s := NewService(nil, "", false)
	labels := s.LabelEntityTypes(context.Background(), []string{"Alice", "Bob"}, "source", nil)
	if len(labels) != 0 {
		t.Errorf("disabled service should return empty map, got %v", labels)
	}
}

func TestLabelEntityTypes_NilService(t *testing.T) {
	var s *Service
	labels := s.LabelEntityTypes(context.Background(), []string{"Alice"}, "src", nil)
	if len(labels) != 0 {
		t.Errorf("nil service should return empty map, got %v", labels)
	}
}

func TestLabelEntityTypes_ParsesMarkdownFencedJSON(t *testing.T) {
	mock := &mockProvider{
		name: "mock",
		complete: func(_ context.Context, _ CompletionRequest) (CompletionResponse, error) {
			// Simulate an LLM that wraps its response in markdown code fences.
			return CompletionResponse{
				Content: "```json\n{\"Alice Johnson\":\"Person\",\"CS101\":\"Course\"}\n```",
			}, nil
		},
	}
	s := NewService(mock, "test", true)

	labels := s.LabelEntityTypes(context.Background(), []string{"Alice Johnson", "CS101"}, "test", nil)

	if got := labels["Alice Johnson"]; got != "Person" {
		t.Errorf("Alice Johnson: got %q, want %q", got, "Person")
	}
	if got := labels["CS101"]; got != "Course" {
		t.Errorf("CS101: got %q, want %q", got, "Course")
	}
}

func TestLabelEntityTypes_DegradeOnError(t *testing.T) {
	mock := &mockProvider{
		name: "mock",
		complete: func(_ context.Context, _ CompletionRequest) (CompletionResponse, error) {
			return CompletionResponse{}, errors.New("provider unavailable")
		},
	}
	s := NewService(mock, "test", true)

	// Must not panic, must return empty map.
	labels := s.LabelEntityTypes(context.Background(), []string{"Alice", "Bob"}, "src", nil)
	if len(labels) != 0 {
		t.Errorf("expected empty map on error, got %v", labels)
	}
}

func TestLabelEntityTypes_InvalidJSON_Degrades(t *testing.T) {
	mock := &mockProvider{
		name: "mock",
		complete: func(_ context.Context, _ CompletionRequest) (CompletionResponse, error) {
			return CompletionResponse{Content: "not valid json at all"}, nil
		},
	}
	s := NewService(mock, "test", true)

	labels := s.LabelEntityTypes(context.Background(), []string{"Alice"}, "src", nil)
	if len(labels) != 0 {
		t.Errorf("expected empty map on parse failure, got %v", labels)
	}
}

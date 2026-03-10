package util

import (
	"context"
	"testing"
)

// mockAuthEntry implements ProviderCounter for testing
type mockAuthEntry struct {
	provider string
}

func (m *mockAuthEntry) GetProvider() string {
	return m.provider
}

// mockStore implements the List interface for testing
type mockStore[T any] struct {
	entries []T
	err     error
}

func (m *mockStore[T]) List(ctx context.Context) ([]T, error) {
	return m.entries, m.err
}

// nilStore is a wrapper that returns nil for the interface
type nilStore[T any] struct{}

func (n *nilStore[T]) List(ctx context.Context) ([]T, error) {
	return nil, nil
}

func TestCountAuthFiles_NilStore(t *testing.T) {
	// Use interface{} nil to test the nil check
	var store interface {
		List(context.Context) ([]string, error)
	} = nil
	count := CountAuthFiles[string](context.Background(), store)
	if count != 0 {
		t.Errorf("CountAuthFiles(nil) = %d, want 0", count)
	}
}

func TestCountAuthFiles_EmptyStore(t *testing.T) {
	store := &mockStore[string]{entries: []string{}}
	count := CountAuthFiles[string](context.Background(), store)
	if count != 0 {
		t.Errorf("CountAuthFiles(empty) = %d, want 0", count)
	}
}

func TestCountAuthFiles_WithEntries(t *testing.T) {
	store := &mockStore[string]{
		entries: []string{"a", "b", "c"},
	}
	count := CountAuthFiles[string](context.Background(), store)
	if count != 3 {
		t.Errorf("CountAuthFiles() = %d, want 3", count)
	}
}

func TestCountAuthFilesByProvider_NilStore(t *testing.T) {
	// Use interface{} nil to test the nil check
	var store interface {
		List(context.Context) ([]*mockAuthEntry, error)
	} = nil
	counts := CountAuthFilesByProvider[*mockAuthEntry](context.Background(), store)
	if len(counts) != 0 {
		t.Errorf("CountAuthFilesByProvider(nil) = %v, want empty map", counts)
	}
}

func TestCountAuthFilesByProvider_EmptyStore(t *testing.T) {
	store := &mockStore[*mockAuthEntry]{entries: []*mockAuthEntry{}}
	counts := CountAuthFilesByProvider[*mockAuthEntry](context.Background(), store)
	if len(counts) != 0 {
		t.Errorf("CountAuthFilesByProvider(empty) = %v, want empty map", counts)
	}
}

func TestCountAuthFilesByProvider_MixedProviders(t *testing.T) {
	store := &mockStore[*mockAuthEntry]{
		entries: []*mockAuthEntry{
			{provider: "codefree"},
			{provider: "gemini"},
			{provider: "codefree"},
			{provider: "claude"},
			{provider: "codefree"},
		},
	}
	counts := CountAuthFilesByProvider[*mockAuthEntry](context.Background(), store)

	if counts["codefree"] != 3 {
		t.Errorf("codefree count = %d, want 3", counts["codefree"])
	}
	if counts["gemini"] != 1 {
		t.Errorf("gemini count = %d, want 1", counts["gemini"])
	}
	if counts["claude"] != 1 {
		t.Errorf("claude count = %d, want 1", counts["claude"])
	}
}

func TestCountAuthFilesByProvider_EmptyProvider(t *testing.T) {
	store := &mockStore[*mockAuthEntry]{
		entries: []*mockAuthEntry{
			{provider: ""},
			{provider: "codefree"},
		},
	}
	counts := CountAuthFilesByProvider[*mockAuthEntry](context.Background(), store)

	// Empty provider should be counted as "other"
	if counts["other"] != 1 {
		t.Errorf("other count = %d, want 1", counts["other"])
	}
	if counts["codefree"] != 1 {
		t.Errorf("codefree count = %d, want 1", counts["codefree"])
	}
}

func TestCountAuthFilesByProvider_NonProviderCounterEntries(t *testing.T) {
	// Test with entries that don't implement ProviderCounter
	store := &mockStore[string]{
		entries: []string{"a", "b", "c"},
	}
	counts := CountAuthFilesByProvider[string](context.Background(), store)

	// Non-ProviderCounter entries should be counted as "other"
	if counts["other"] != 3 {
		t.Errorf("other count = %d, want 3", counts["other"])
	}
}

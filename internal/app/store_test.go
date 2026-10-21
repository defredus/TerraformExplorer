package app

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func TestStoreScansAndReadsKnownSource(t *testing.T) {
	store := NewStore()
	snapshot, err := store.Scan(filepath.Join("..", "..", "testdata", "simple-module"))
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.ID == "" || len(snapshot.Graph.Nodes) == 0 {
		t.Fatalf("incomplete snapshot: %#v", snapshot)
	}

	source, err := store.ReadSource(snapshot.ID, "network.tf")
	if err != nil {
		t.Fatal(err)
	}
	if source.Content == "" {
		t.Fatal("source content is empty")
	}

	if _, err := store.ReadSource(snapshot.ID, "../network.tf"); !errors.Is(err, ErrInvalidSource) {
		t.Fatalf("path traversal error = %v, want ErrInvalidSource", err)
	}
}

func TestStoreReadsSourceFromDiscoveredModule(t *testing.T) {
	store := NewStore()
	snapshot, err := store.Scan(filepath.Join("..", "..", "testdata", "nested-project"))
	if err != nil {
		t.Fatal(err)
	}

	source, err := store.ReadSource(snapshot.ID, "module.service.module.labels/main.tf")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(source.Content, `variable "prefix"`) {
		t.Fatalf("unexpected nested source: %q", source.Content)
	}
}

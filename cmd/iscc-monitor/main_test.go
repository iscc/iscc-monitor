// Test for the registerHubs wiring: parsing the realm fixture and registering it
// into a fresh store yields one follower target per hub with a positive hub_id and
// the origin-correct base URL, and a second registration is idempotent (identical
// hub_ids). The blocking Loop.Run is deliberately not exercised here.
package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/iscc/iscc-monitor/internal/registry"
	"github.com/iscc/iscc-monitor/internal/store"
)

func TestRegisterHubs(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "internal", "registry", "testdata", "realm.txt"))
	if err != nil {
		t.Fatalf("read realm fixture: %v", err)
	}
	entries, err := registry.Parse(data)
	if err != nil {
		t.Fatalf("registry.Parse: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("registry.Parse returned %d entries, want 2 (sb0.iscc.id, sb1.amlet.id)", len(entries))
	}

	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	defer func() { _ = st.Close() }()

	ctx := context.Background()
	targets, err := registerHubs(ctx, st, entries)
	if err != nil {
		t.Fatalf("registerHubs: %v", err)
	}

	if len(targets) != 2 {
		t.Fatalf("registerHubs returned %d targets, want 2", len(targets))
	}
	for i, target := range targets {
		want := entries[i].BaseURL
		if want != "https://"+entries[i].Domain {
			t.Fatalf("fixture entry %d BaseURL = %q, want %q", i, entries[i].BaseURL, "https://"+entries[i].Domain)
		}
		if target.BaseURL != want {
			t.Errorf("target %d BaseURL = %q, want %q", i, target.BaseURL, want)
		}
		if target.HubID <= 0 {
			t.Errorf("target %d HubID = %d, want > 0", i, target.HubID)
		}
	}

	// Idempotency: a second registration returns identical hub_ids (UpsertHub is
	// idempotent on the domain).
	again, err := registerHubs(ctx, st, entries)
	if err != nil {
		t.Fatalf("registerHubs (second call): %v", err)
	}
	if len(again) != len(targets) {
		t.Fatalf("second registerHubs returned %d targets, want %d", len(again), len(targets))
	}
	for i := range targets {
		if again[i].HubID != targets[i].HubID {
			t.Errorf("target %d HubID not idempotent: first %d, second %d", i, targets[i].HubID, again[i].HubID)
		}
		if again[i].BaseURL != targets[i].BaseURL {
			t.Errorf("target %d BaseURL changed: first %q, second %q", i, targets[i].BaseURL, again[i].BaseURL)
		}
	}
}

// Test for the registerHubs wiring: parsing the realm fixture and registering it
// into a fresh store yields one follower target per hub with a positive hub_id and
// the origin-correct base URL, and a second registration is idempotent (identical
// hub_ids). The blocking Loop.Run is deliberately not exercised here.
package main

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/iscc/iscc-monitor/internal/metrics"
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
	targets, routes, err := registerHubs(ctx, st, entries)
	if err != nil {
		t.Fatalf("registerHubs: %v", err)
	}

	if len(targets) != 2 {
		t.Fatalf("registerHubs returned %d targets, want 2", len(targets))
	}
	if len(routes) != 2 {
		t.Fatalf("registerHubs returned %d routes, want 2", len(routes))
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
		// The route is index-aligned with the target and carries the full
		// <domain>/log origin (never the bare domain) for the same hub_id.
		if routes[i].HubID != target.HubID {
			t.Errorf("route %d HubID = %d, want %d (aligned with target)", i, routes[i].HubID, target.HubID)
		}
		if want := entries[i].Domain + "/log"; routes[i].Origin != want {
			t.Errorf("route %d Origin = %q, want %q", i, routes[i].Origin, want)
		}
	}

	// Idempotency: a second registration returns identical hub_ids (UpsertHub is
	// idempotent on the domain).
	again, _, err := registerHubs(ctx, st, entries)
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

// TestMirrorRouter seeds one hub with a checkpoint BLOB and drives the combined
// mux (per-hub mirror + /metrics) directly via ServeHTTP, asserting the request
// routes to the hub's mirror at its <origin>/ prefix, that an unmirrored path
// under the prefix 404s, that a path missing the /log segment does not match the
// prefix (404), and that the existing /metrics route is untouched on the shared
// mux. It exercises the real router without binding a socket.
func TestMirrorRouter(t *testing.T) {
	ctx := context.Background()
	st, err := store.Open(filepath.Join(t.TempDir(), "mirror.db"))
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	defer func() { _ = st.Close() }()

	hub, err := st.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}
	checkpoint := []byte("sb0.iscc.id/log\n300\nrootbytes==\n\n— sb0 sig\n")
	if _, _, err := st.RecordCheckpoint(ctx, store.CheckpointRecord{
		HubID: hub, TreeSize: 300, Root: []byte("root"), Raw: checkpoint, ObservedAt: time.Unix(1700000000, 0),
	}); err != nil {
		t.Fatalf("RecordCheckpoint: %v", err)
	}

	routes := []hubRoute{{HubID: hub, Origin: "sb0.iscc.id/log"}}
	mux := buildMux(st, routes, metrics.New())

	// GET /sb0.iscc.id/log/checkpoint -> 200 byte-equal to the seeded BLOB.
	t.Run("checkpoint at origin prefix is 200 byte-equal", func(t *testing.T) {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/sb0.iscc.id/log/checkpoint", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		if !bytes.Equal(rec.Body.Bytes(), checkpoint) {
			t.Errorf("body = %q, want %q", rec.Body.Bytes(), checkpoint)
		}
	})

	// GET /sb0.iscc.id/log/tile/0/000 -> 404 (under the prefix but never mirrored).
	t.Run("unmirrored path under prefix is 404", func(t *testing.T) {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/sb0.iscc.id/log/tile/0/000", nil))
		if rec.Code != http.StatusNotFound {
			t.Errorf("status = %d, want 404", rec.Code)
		}
	})

	// GET /sb0.iscc.id/checkpoint -> 404: the origin must be the full <domain>/log,
	// so a path missing the /log segment does not match the prefix.
	t.Run("origin missing /log segment is 404", func(t *testing.T) {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/sb0.iscc.id/checkpoint", nil))
		if rec.Code != http.StatusNotFound {
			t.Errorf("status = %d, want 404", rec.Code)
		}
	})

	// GET /metrics -> 200: the existing route is untouched on the shared mux.
	t.Run("metrics still served on shared mux", func(t *testing.T) {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want 200", rec.Code)
		}
	})
}

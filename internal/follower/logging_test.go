// Tests for the loop's structured logging at the composition boundary. They
// inject a recording *slog.Logger over a bytes.Buffer-backed JSON handler into
// Loop.Logger, force a deterministic per-hub fault through the outbound-fetch seam
// (a fetcher that errors), and assert on the captured records (level + keys), not
// on formatted text, so the test is format-stable. Following the loop-test
// convention that Run is intentionally untested (it blocks on a ticker), the
// log-and-continue behavior is exercised through Tick with an injected now.
package follower

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"
	"time"
)

// errFetcher is a Fetcher that always fails, so PollHub's FetchCheckpoint returns
// a non-nil error and Tick folds it into firstErr — the deterministic per-hub
// fault path the logging test drives.
type errFetcher struct{}

func (errFetcher) Fetch(context.Context, string) ([]byte, error) {
	return nil, errFetch
}

// errFetch is the sentinel the failing fetcher returns, so the captured log
// record's "err" attribute is non-empty and the fault is unambiguous.
var errFetch = errFetchErr{}

type errFetchErr struct{}

func (errFetchErr) Error() string { return "synthetic fetch failure" }

// records decodes the JSON-handler buffer into one map per emitted slog record,
// so assertions read structured fields (level, err, hub_id) rather than parsing
// formatted text.
func records(t *testing.T, buf *bytes.Buffer) []map[string]any {
	t.Helper()
	var out []map[string]any
	dec := json.NewDecoder(bytes.NewReader(buf.Bytes()))
	for dec.More() {
		var rec map[string]any
		if err := dec.Decode(&rec); err != nil {
			t.Fatalf("decode slog record: %v", err)
		}
		out = append(out, rec)
	}
	return out
}

// TestLoopLogsTickError proves the previously-swallowed tick error is now
// observable: a single hub whose fetcher always fails makes Tick return a non-nil
// error, and the injected logger captures exactly one ERROR-level record carrying
// an "err" attribute and the hub_id. Critically the pass does NOT abort — Tick
// returns the fault to its caller (the loop would continue to the next tick), so
// the log-and-continue invariant holds.
func TestLoopLogsTickError(t *testing.T) {
	ctx := context.Background()
	s, _ := openTemp(t)
	hubID, err := s.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	loop := &Loop{
		Store:   s,
		Fetcher: errFetcher{},
		Targets: []HubTarget{{HubID: hubID, BaseURL: "https://sb0.iscc.id"}},
		Normal:  10 * time.Minute,
		Frozen:  time.Hour,
		Alert:   noopAlert,
		Logger:  logger,
	}

	// The tick must surface the fault (so the loop's log-and-continue path fires)
	// WITHOUT aborting — Tick returns the error to its caller, it never panics.
	if err := loop.Tick(ctx, sb0ObservedAt()); err == nil {
		t.Fatal("Tick error = nil, want the per-hub fetch fault surfaced")
	}

	recs := records(t, &buf)
	var errorRecs []map[string]any
	for _, r := range recs {
		if r["level"] == "ERROR" {
			errorRecs = append(errorRecs, r)
		}
	}
	if len(errorRecs) != 1 {
		t.Fatalf("ERROR-level records = %d, want exactly 1 (the swallowed tick error is now logged)", len(errorRecs))
	}
	rec := errorRecs[0]
	if _, ok := rec["err"]; !ok {
		t.Errorf("ERROR record missing %q attribute, got keys %v", "err", keysOf(rec))
	}
	if got, ok := rec["hub_id"]; !ok || int64(got.(float64)) != hubID {
		t.Errorf("ERROR record hub_id = %v, want %d", rec["hub_id"], hubID)
	}
}

// TestLoopLoggerNilSafe proves the logger() accessor falls back to slog.Default()
// so a bare &Loop{…} with no Logger runs a faulting tick without panicking — the
// existing &Loop{…} literals in loop_test.go keep working unchanged.
func TestLoopLoggerNilSafe(t *testing.T) {
	ctx := context.Background()
	s, _ := openTemp(t)
	hubID, err := s.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}

	// Redirect the default logger to a discarding buffer so the faulting tick does
	// not write to the test's stderr, then prove the nil-Logger loop still surfaces
	// the error without panicking.
	prev := slog.Default()
	t.Cleanup(func() { slog.SetDefault(prev) })
	var buf bytes.Buffer
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))

	loop := &Loop{
		Store:   s,
		Fetcher: errFetcher{},
		Targets: []HubTarget{{HubID: hubID, BaseURL: "https://sb0.iscc.id"}},
		Normal:  10 * time.Minute,
		Frozen:  time.Hour,
		Alert:   noopAlert,
	}
	if err := loop.Tick(ctx, sb0ObservedAt()); err == nil {
		t.Fatal("Tick error = nil, want the per-hub fetch fault surfaced (nil-Logger path)")
	}
	if got := records(t, &buf); len(got) == 0 {
		t.Error("nil-Logger fault produced no default-logger record, want at least one")
	}
}

// keysOf returns a record's keys for a readable assertion failure message.
func keysOf(rec map[string]any) []string {
	keys := make([]string, 0, len(rec))
	for k := range rec {
		keys = append(keys, k)
	}
	return keys
}

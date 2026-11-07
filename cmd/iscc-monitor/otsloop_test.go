// Tests for runOTSLoop's cadence: it runs one OpenTimestamps pass at startup without
// waiting an interval, a root stamped in one pass is checked for Bitcoin confirmation
// on the next interval tick, and every pass logs one INFO summary line. The Stamper and
// Upgrader are fakes over a real temp store, so no calendar is contacted.
package main

import (
	"bytes"
	"context"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/iscc/iscc-monitor/internal/follower"
	"github.com/iscc/iscc-monitor/internal/store"
)

// otsTestRoot is the 32-byte root of the seeded pending row.
var otsTestRoot = []byte("ots-loop-test-root-padding-32by!")

// seedUnstamped opens a temp store holding one hub with one pending, not-yet-stamped
// OTS row (the empty-OTSBytes row the poll path writes) and returns the store and hub.
func seedUnstamped(t *testing.T) (*store.Store, int64) {
	t.Helper()
	ctx := context.Background()
	st, err := store.Open(filepath.Join(t.TempDir(), "ots.db"))
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	hubID, err := st.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}
	if _, _, err := st.RecordOTS(ctx, store.OTSRecord{
		HubID: hubID, TreeSize: 100, Root: otsTestRoot, Status: store.OTSStatusPending, StampedAt: time.Now(),
	}); err != nil {
		t.Fatalf("RecordOTS: %v", err)
	}
	return st, hubID
}

// stampAll stamps every root with a fixed pending proof.
func stampAll(_ context.Context, _ [32]byte) ([]byte, string, error) {
	return []byte("calendar-pending-ots-proof-blob!"), "https://calendar.example", nil
}

// confirmAll reports every stamped root Bitcoin-confirmed at block 900000.
func confirmAll(_ context.Context, _ store.OTSRecord) (follower.UpgradeResult, error) {
	return follower.UpgradeResult{Confirmed: true, OTSBytes: []byte("confirmed-proof"), BTCHeight: 900_000}, nil
}

// isStamped reports whether the row holds a calendar proof.
func isStamped(r store.OTSRecord) bool { return len(r.OTSBytes) > 0 }

// isConfirmed reports whether the row is recorded as Bitcoin-confirmed.
func isConfirmed(r store.OTSRecord) bool { return r.Status == store.OTSStatusConfirmed }

// startLoop runs runOTSLoop in a goroutine and returns a stop function that cancels it
// and waits for it to return.
func startLoop(st *store.Store, stamper follower.Stamper, up follower.Upgrader, interval time.Duration, logger *slog.Logger) func() {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		runOTSLoop(ctx, st, stamper, up, interval, logger)
		close(done)
	}()
	return func() {
		cancel()
		<-done
	}
}

// waitForRow polls the seeded row until reached reports true, failing the test after
// ten seconds. Waiting on the stored row, not on the fake's call, means the pass has
// persisted its result before the test stops the loop.
func waitForRow(t *testing.T, st *store.Store, hubID int64, reached func(store.OTSRecord) bool, what string) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		r, found, err := st.OTSForRoot(context.Background(), hubID, 100, otsTestRoot)
		if err != nil {
			t.Fatalf("OTSForRoot: %v", err)
		}
		if found && reached(r) {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

// TestRunOTSLoopPassesAtStartup asserts the loop stamps a pending root right away with
// a one-hour interval, so a restart never postpones anchoring by an interval, and that
// the pass logs its INFO summary line. (Mutation: dropping the startup pass times out.)
func TestRunOTSLoopPassesAtStartup(t *testing.T) {
	st, hubID := seedUnstamped(t)
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))

	stop := startLoop(st, stampAll, confirmAll, time.Hour, logger)
	waitForRow(t, st, hubID, isStamped, "the startup stamp")
	stop()

	if out := logs.String(); !strings.Contains(out, `msg="ots pass" stamped=1 confirmed=0 pending=0 failed=0 next=`) {
		t.Errorf("log missing the ots pass summary line:\n%s", out)
	}
}

// TestRunOTSLoopUpgradesOnNextTick asserts a root stamped by the startup pass is checked
// for confirmation on the next interval tick and recorded as confirmed. (Mutation:
// ignoring the configured interval leaves the row pending until the timeout.)
func TestRunOTSLoopUpgradesOnNextTick(t *testing.T) {
	st, hubID := seedUnstamped(t)

	stop := startLoop(st, stampAll, confirmAll, 20*time.Millisecond, slog.New(slog.DiscardHandler))
	waitForRow(t, st, hubID, isConfirmed, "the confirmation")
	stop()
}

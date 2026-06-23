// Tests for the read-only hub-summary projection (ListHubs). Each drives the
// public store API against a t.TempDir() database (via openTemp) and asserts on the
// returned HubSummary slice — never on Store internals. The Anchor assertions are
// mutation-targeted: reverting the ots correlated subselect to a literal "" makes
// the confirmed/pending cases FAIL, and the never-stamped hub pins the honest empty
// state.
package store

import (
	"context"
	"testing"
	"time"
)

// TestListHubsAnchorStatus pins the Anchor projection: a hub with a confirmed OTS
// row reports Anchor == OTSStatusConfirmed, a hub with only a pending OTS row
// reports OTSStatusPending, the newest-stamped row wins when a hub has several, and
// a hub with no OTS row reports the empty never-stamped state ("").
func TestListHubsAnchorStatus(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)

	// Confirmed hub: one stamped root flipped to confirmed.
	confirmedID, err := s.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub confirmed: %v", err)
	}
	confirmedRoot := []byte("confirmed-anchor-root-padding-32!")
	if _, _, err := s.RecordOTS(ctx, OTSRecord{
		HubID:     confirmedID,
		TreeSize:  10,
		Root:      confirmedRoot,
		Status:    OTSStatusConfirmed,
		StampedAt: time.Unix(1_700_000_000, 0),
	}); err != nil {
		t.Fatalf("RecordOTS confirmed: %v", err)
	}

	// Pending hub: an older pending root plus a newer pending root — the newest
	// stamped row is the relevant one and both are pending.
	pendingID, err := s.UpsertHub(ctx, "sb1.amlet.id", "sb1.amlet.id/log", "https://sb1.amlet.id")
	if err != nil {
		t.Fatalf("UpsertHub pending: %v", err)
	}
	if _, _, err := s.RecordOTS(ctx, OTSRecord{
		HubID:     pendingID,
		TreeSize:  5,
		Root:      []byte("pending-anchor-root-older-pad-32!"),
		Status:    OTSStatusPending,
		StampedAt: time.Unix(1_700_000_000, 0),
	}); err != nil {
		t.Fatalf("RecordOTS pending older: %v", err)
	}
	if _, _, err := s.RecordOTS(ctx, OTSRecord{
		HubID:     pendingID,
		TreeSize:  8,
		Root:      []byte("pending-anchor-root-newer-pad-32!"),
		Status:    OTSStatusPending,
		StampedAt: time.Unix(1_700_001_000, 0),
	}); err != nil {
		t.Fatalf("RecordOTS pending newer: %v", err)
	}

	// Never-stamped hub: no OTS row at all → empty Anchor.
	if _, err := s.UpsertHub(ctx, "sb2.iscc.id", "sb2.iscc.id/log", "https://sb2.iscc.id"); err != nil {
		t.Fatalf("UpsertHub never-stamped: %v", err)
	}

	hubs, err := s.ListHubs(ctx)
	if err != nil {
		t.Fatalf("ListHubs: %v", err)
	}

	got := map[string]string{}
	for _, h := range hubs {
		got[h.Domain] = h.Anchor
	}
	cases := map[string]string{
		"sb0.iscc.id":  OTSStatusConfirmed,
		"sb1.amlet.id": OTSStatusPending,
		"sb2.iscc.id":  "",
	}
	for domain, want := range cases {
		if got[domain] != want {
			t.Errorf("Anchor for %s = %q, want %q", domain, got[domain], want)
		}
	}
}

// TestListHubsCheckpointAndAnchorHeight pins the two dossier subselects: a hub with
// an accepted checkpoint + a confirmed OTS row carrying a btc_height reports the
// newest checkpoint's observed_at as CheckpointObserved and the confirmed height as
// AnchorHeight; a hub with neither reports their zero values (NULL-safe). It is
// mutation-targeted: dropping either subselect makes the populated assertions FAIL.
func TestListHubsCheckpointAndAnchorHeight(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)

	// Anchored hub: an accepted checkpoint (sets observed_at + coverage) plus a
	// confirmed OTS row carrying a Bitcoin block height.
	observed := time.Unix(1_700_000_000, 0)
	anchoredRoot := []byte("anchored-checkpoint-root-pad-32!!")
	anchoredID, err := s.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub anchored: %v", err)
	}
	if err := s.AdvanceAccepted(ctx, CheckpointRecord{
		HubID:      anchoredID,
		TreeSize:   42,
		Root:       anchoredRoot,
		Raw:        []byte("raw-checkpoint-bytes"),
		ObservedAt: observed,
	}); err != nil {
		t.Fatalf("AdvanceAccepted anchored: %v", err)
	}
	if _, _, err := s.RecordOTS(ctx, OTSRecord{
		HubID:     anchoredID,
		TreeSize:  42,
		Root:      anchoredRoot,
		Status:    OTSStatusConfirmed,
		StampedAt: observed,
		BTCHeight: 869440,
	}); err != nil {
		t.Fatalf("RecordOTS anchored: %v", err)
	}

	// Bare hub: registered only, no checkpoint and no OTS row → zero values.
	if _, err := s.UpsertHub(ctx, "sb1.amlet.id", "sb1.amlet.id/log", "https://sb1.amlet.id"); err != nil {
		t.Fatalf("UpsertHub bare: %v", err)
	}

	hubs, err := s.ListHubs(ctx)
	if err != nil {
		t.Fatalf("ListHubs: %v", err)
	}
	byDomain := map[string]HubSummary{}
	for _, h := range hubs {
		byDomain[h.Domain] = h
	}

	anchored := byDomain["sb0.iscc.id"]
	if !anchored.CheckpointObserved.Equal(observed) {
		t.Errorf("CheckpointObserved = %v, want %v", anchored.CheckpointObserved, observed)
	}
	if anchored.AnchorHeight != 869440 {
		t.Errorf("AnchorHeight = %d, want 869440", anchored.AnchorHeight)
	}

	bare := byDomain["sb1.amlet.id"]
	if !bare.CheckpointObserved.IsZero() {
		t.Errorf("bare CheckpointObserved = %v, want zero", bare.CheckpointObserved)
	}
	if bare.AnchorHeight != 0 {
		t.Errorf("bare AnchorHeight = %d, want 0", bare.AnchorHeight)
	}
}

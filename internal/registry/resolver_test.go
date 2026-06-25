// Tests for the realm-resolution seam: Entries derivation, the nil-safe Resolve
// receiver, and the hot-swappable AtomicHubList. The regression test pins the live
// monitor.iscc.io bug — a mainnet Hub-List (hub_ids 1,2) must resolve the embedded
// hub_id, which a document-order index mapping (slot i = entry i) would miss for
// any hub whose real hub_id is not its line position.
package registry

import "testing"

// slotPtr makes a *uint16 for a registry.Hub slot literal in tests.
func slotPtr(v uint16) *uint16 { return &v }

// mainnetHubList is the realm-1 Hub-List shape from iscc-hub/hubs/mainnet.yaml:
// hub_id 1 -> iscc.id, hub_id 2 -> amlet.id. The embedded hub_ids are 1-based, so a
// document-order mapping would mis-slot both.
func mainnetHubList() *HubList {
	return &HubList{
		Version: 1,
		Network: "mainnet",
		Hubs: []Hub{
			{HubID: slotPtr(1), URL: "https://iscc.id", Active: true},
			{HubID: slotPtr(2), URL: "https://amlet.id", Active: true},
		},
	}
}

// TestResolveEmbeddedHubIDNotDocumentOrder is the regression for the live
// monitor.iscc.io "not found in this realm" bug: an ISCC-IDv1 minted by amlet.id
// carries hub_id 2, and the mainnet Hub-List must resolve slot 2 to amlet.id and
// slot 1 to iscc.id. A document-order mapping (slot i = entry i) would resolve slot
// 0/1 instead and miss slot 2 entirely — the exact failure this guards against.
func TestResolveEmbeddedHubIDNotDocumentOrder(t *testing.T) {
	hl := mainnetHubList()

	domain, ok := hl.Resolve(2)
	if !ok || domain != "amlet.id" {
		t.Fatalf("Resolve(2) = %q, %v; want amlet.id, true (the hub_id-2 issuer)", domain, ok)
	}
	if domain, ok := hl.Resolve(1); !ok || domain != "iscc.id" {
		t.Errorf("Resolve(1) = %q, %v; want iscc.id, true", domain, ok)
	}
	// Slot 0 is not in the mainnet realm — the document-order mapping would have
	// invented it for the first line. It must miss.
	if domain, ok := hl.Resolve(0); ok {
		t.Errorf("Resolve(0) = %q, true; want a miss (no hub_id 0 in mainnet realm)", domain)
	}
}

// TestResolveNilReceiver asserts a nil *HubList resolves nothing rather than
// panicking, so AtomicHubList can delegate to it before any snapshot is seeded.
func TestResolveNilReceiver(t *testing.T) {
	var hl *HubList
	if domain, ok := hl.Resolve(2); ok || domain != "" {
		t.Errorf("nil HubList Resolve(2) = %q, %v; want \"\", false", domain, ok)
	}
}

// TestEntries derives the follower's domains + base URLs from a Hub-List in
// document order, dropping the hub_id slot (the follower keys by domain).
func TestEntries(t *testing.T) {
	entries, err := mainnetHubList().Entries()
	if err != nil {
		t.Fatalf("Entries: %v", err)
	}
	want := []Entry{
		{Domain: "iscc.id", BaseURL: "https://iscc.id"},
		{Domain: "amlet.id", BaseURL: "https://amlet.id"},
	}
	if len(entries) != len(want) {
		t.Fatalf("Entries len = %d, want %d: %+v", len(entries), len(want), entries)
	}
	for i, e := range entries {
		if e != want[i] {
			t.Errorf("Entries[%d] = %+v, want %+v", i, e, want[i])
		}
	}
}

// TestEntriesNilReceiver asserts a nil *HubList yields no entries and no error.
func TestEntriesNilReceiver(t *testing.T) {
	var hl *HubList
	entries, err := hl.Entries()
	if err != nil || entries != nil {
		t.Errorf("nil HubList Entries = %+v, %v; want nil, nil", entries, err)
	}
}

// TestAtomicHubListSwap asserts the holder resolves through its current snapshot
// and that Store atomically swaps in a new one — the mechanism the hourly realm
// refresh uses to pick up a new hub without a restart.
func TestAtomicHubListSwap(t *testing.T) {
	a := NewAtomicHubList(mainnetHubList())
	if domain, ok := a.Resolve(2); !ok || domain != "amlet.id" {
		t.Fatalf("seeded Resolve(2) = %q, %v; want amlet.id, true", domain, ok)
	}

	// A refreshed realm adds hub_id 3.
	a.Store(&HubList{Version: 1, Hubs: []Hub{
		{HubID: slotPtr(3), URL: "https://newhub.example", Active: true},
	}})
	if domain, ok := a.Resolve(3); !ok || domain != "newhub.example" {
		t.Errorf("after swap Resolve(3) = %q, %v; want newhub.example, true", domain, ok)
	}
	if _, ok := a.Resolve(2); ok {
		t.Errorf("after swap Resolve(2) still resolves; want the swapped-in snapshot only")
	}
}

// TestAtomicHubListNilSnapshot asserts a holder seeded with nil resolves nothing
// (fail-closed empty realm) rather than panicking.
func TestAtomicHubListNilSnapshot(t *testing.T) {
	a := NewAtomicHubList(nil)
	if domain, ok := a.Resolve(2); ok || domain != "" {
		t.Errorf("nil-seeded Resolve(2) = %q, %v; want \"\", false", domain, ok)
	}
}

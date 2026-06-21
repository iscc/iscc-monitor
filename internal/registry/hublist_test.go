// Tests for the iscc-hub Hub-List parser and the (realm, hub_id) -> domain
// resolver: the golden testnet fixture parses to the two real testnet hubs and
// resolves hub_id 0/1 to their hard-coded domains; an inactive hub still resolves
// (ok == true); an unknown slot misses (ok == false); the full 12-bit range
// (4095) resolves; pubkey is parsed-and-ignored; and a malformed document
// (invalid YAML, missing hub_id, over-range hub_id, duplicate hub_id, empty url,
// or a scheme-less/scheme'd path-bearing url) fails closed with a nil list.
// Expected domains are hard-coded ground truth, not values re-derived from the
// parser.
package registry

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseHubListGolden(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "testnet.yaml"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	hl, err := ParseHubList(data)
	if err != nil {
		t.Fatalf("ParseHubList(testnet.yaml) error: %v", err)
	}
	if hl.Version != 1 {
		t.Errorf("Version = %d, want 1", hl.Version)
	}
	if hl.Network != "testnet" {
		t.Errorf("Network = %q, want %q", hl.Network, "testnet")
	}
	if len(hl.Hubs) != 2 {
		t.Fatalf("Hubs len = %d, want 2", len(hl.Hubs))
	}

	// Hard-coded ground truth (the real testnet hubs, matching realm.txt and
	// derive_vkey.py's HUBS) — not values re-derived from the parser.
	if got, ok := hl.Resolve(0); !ok || got != "sb0.iscc.id" {
		t.Errorf("Resolve(0) = (%q, %v), want (%q, true)", got, ok, "sb0.iscc.id")
	}
	if got, ok := hl.Resolve(1); !ok || got != "sb1.amlet.id" {
		t.Errorf("Resolve(1) = (%q, %v), want (%q, true)", got, ok, "sb1.amlet.id")
	}
}

func TestResolveUnknownSlotMisses(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "testnet.yaml"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	hl, err := ParseHubList(data)
	if err != nil {
		t.Fatalf("ParseHubList(testnet.yaml) error: %v", err)
	}
	if got, ok := hl.Resolve(9); ok || got != "" {
		t.Errorf("Resolve(9) = (%q, %v), want (\"\", false)", got, ok)
	}
}

func TestResolveInactiveStillResolves(t *testing.T) {
	const doc = `version: 1
network: testnet
hubs:
  - hub_id: 7
    url: https://paused.example
    active: false
`
	hl, err := ParseHubList([]byte(doc))
	if err != nil {
		t.Fatalf("ParseHubList error: %v", err)
	}
	// An inactive hub resolves (ok == true) — the badge conveys inactivity, not
	// resolution. ok == false is reserved for an unknown slot.
	if got, ok := hl.Resolve(7); !ok || got != "paused.example" {
		t.Errorf("Resolve(7) = (%q, %v), want (%q, true)", got, ok, "paused.example")
	}
}

func TestResolveFull12BitRange(t *testing.T) {
	const doc = `version: 1
network: testnet
hubs:
  - hub_id: 4095
    url: https://edge.example
    active: true
`
	hl, err := ParseHubList([]byte(doc))
	if err != nil {
		t.Fatalf("ParseHubList error: %v", err)
	}
	if got, ok := hl.Resolve(4095); !ok || got != "edge.example" {
		t.Errorf("Resolve(4095) = (%q, %v), want (%q, true)", got, ok, "edge.example")
	}
}

func TestParseHubListIgnoresPubkey(t *testing.T) {
	// The fixture carries a pubkey on hub 0; assert it is parsed-and-ignored — no
	// type holds it, so the Hub struct has no pubkey field to read it into.
	data, err := os.ReadFile(filepath.Join("testdata", "testnet.yaml"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	hl, err := ParseHubList(data)
	if err != nil {
		t.Fatalf("ParseHubList(testnet.yaml) error: %v", err)
	}
	// The Hub struct is exactly {HubID, URL, Active}; a pubkey-bearing fixture
	// parses without error and the kept hub holds no key. HubID is a *uint16
	// (presence-tracked), so compare the dereferenced slot plus the other fields.
	h := hl.Hubs[0]
	if h.HubID == nil || *h.HubID != 0 || h.URL != "https://sb0.iscc.id" || !h.Active {
		t.Errorf("Hubs[0] = %#v, want {hub_id 0, https://sb0.iscc.id, active}", h)
	}
}

func TestParseHubListEmptyHubs(t *testing.T) {
	// A document with zero hubs is valid (mirrors Parse's all-comment -> empty),
	// and every slot then misses.
	const doc = `version: 1
network: testnet
hubs: []
`
	hl, err := ParseHubList([]byte(doc))
	if err != nil {
		t.Fatalf("ParseHubList(empty hubs) error: %v", err)
	}
	if len(hl.Hubs) != 0 {
		t.Errorf("Hubs len = %d, want 0", len(hl.Hubs))
	}
	if got, ok := hl.Resolve(0); ok || got != "" {
		t.Errorf("Resolve(0) = (%q, %v), want (\"\", false)", got, ok)
	}
}

func TestParseHubListErrors(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		errFrag string
	}{
		{
			name:    "invalid yaml",
			in:      "version: 1\nhubs: [: not yaml\n",
			errFrag: "parse Hub-List",
		},
		{
			name: "hub_id over range",
			in: `version: 1
network: testnet
hubs:
  - hub_id: 4096
    url: https://over.example
    active: true
`,
			errFrag: "4096",
		},
		{
			name: "duplicate hub_id",
			in: `version: 1
network: testnet
hubs:
  - hub_id: 3
    url: https://a.example
    active: true
  - hub_id: 3
    url: https://b.example
    active: true
`,
			errFrag: "duplicate hub_id 3",
		},
		{
			name: "empty url",
			in: `version: 1
network: testnet
hubs:
  - hub_id: 0
    url: ""
    active: true
`,
			errFrag: "empty",
		},
		{
			// Scheme-less host/path: caught by the no-host branch (url.Parse puts
			// "sb0.iscc.id/log" in Path with an empty Host). Kept for coverage of
			// that branch; the scheme'd case below is the load-bearing path guard.
			name: "scheme-less path-bearing url",
			in: `version: 1
network: testnet
hubs:
  - hub_id: 0
    url: sb0.iscc.id/log
    active: true
`,
			errFrag: "no host",
		},
		{
			// Scheme'd path-bearing url: url.Parse accepts this and would otherwise
			// return host "sb0.iscc.id", silently dropping "/log". The non-vacuous
			// guard is the u.Path check; reverting it makes this case pass parsing
			// and so FAIL the test.
			name: "scheme'd path-bearing url",
			in: `version: 1
network: testnet
hubs:
  - hub_id: 0
    url: https://sb0.iscc.id/log
    active: true
`,
			errFrag: "not a bare host base url",
		},
		{
			// Missing hub_id: a plain uint16 would decode the absent key to slot 0;
			// the *uint16 presence check rejects it. Reverting that check makes this
			// entry silently become slot 0 and so FAIL the test.
			name: "missing hub_id",
			in: `version: 1
network: testnet
hubs:
  - url: https://sb0.iscc.id
    active: true
`,
			errFrag: "hub_id is required",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseHubList([]byte(tc.in))
			if err == nil {
				t.Fatalf("ParseHubList(%q) = %#v, want error", tc.in, got)
			}
			if got != nil {
				t.Errorf("ParseHubList(%q) returned %#v alongside error, want nil", tc.in, got)
			}
			if !strings.Contains(err.Error(), tc.errFrag) {
				t.Errorf("ParseHubList(%q) error %q does not contain %q", tc.in, err, tc.errFrag)
			}
		})
	}
}

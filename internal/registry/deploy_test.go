// Tests that the canonical deploy realm document (the mountable/bakeable
// deploy/realm-testnet.txt, baked into the image at /etc/iscc-monitor/realm.txt)
// parses to exactly the two ordered testnet hubs. Non-vacuous: it reads the real
// repo-root file, so deleting, malforming, URL-shaping, or drifting it fails here.
package registry

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseCanonicalDeployRealm(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "deploy", "realm-testnet.txt"))
	if err != nil {
		t.Fatalf("read deploy realm: %v", err)
	}
	got, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse(deploy/realm-testnet.txt) error: %v", err)
	}
	want := []Entry{
		{Domain: "sb0.iscc.id", BaseURL: "https://sb0.iscc.id"},
		{Domain: "sb1.amlet.id", BaseURL: "https://sb1.amlet.id"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Parse(deploy/realm-testnet.txt) = %#v, want %#v", got, want)
	}
}

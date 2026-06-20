// Tests for the pure checkpoint-acceptance decision. They drive AcceptCheckpoint
// through a fake Fetcher returning did.json bytes (the captured sb0 fixture plus
// inline-constructed documents for the rotated/malformed cases) against the live
// sb0 checkpoint fixture, asserting the four-way Status verdict on observable
// outputs only — never on follower internals. Together they cover verified,
// unverified, unresolvable (fetch failure AND fail-closed malformed timestamp),
// and rotated (good signature by an out-of-window key).
package logclient

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"
)

// sb0Multibase is the publicKeyMultibase from the sb0 did:web fixture, i.e. the
// key the captured sb0 checkpoint is signed with. Constructing did.json inline
// from it lets the rotated/malformed tests vary the validity window without
// editing the committed golden fixture.
const sb0Multibase = "z6MkqbHELZopsq6eKrn6qxiAgRoVvwp2Vp7mPfKsvrbYmGwJ"

// sb1Multibase is sb1's prior (z6MkiNW…, keyhash 22b08f3e) did:web key — a real
// Ed25519 key that does NOT sign the sb0 checkpoint, used to force StatusUnverified.
const sb1Multibase = "z6MkiNWE7CmYP2bSeYZ3KvHRVoKsAvfYa9aV1FBtaykbgWtb"

// didJSON builds a minimal one-method did:web document for the sb0 origin using
// the given publicKeyMultibase and optional validity fields. A field is omitted
// when empty so the document stays close to the live shape.
func didJSON(multibase, validFrom, validUntil, revoked string) []byte {
	vm := fmt.Sprintf(`{"id": "did:web:sb0.iscc.id#k", "type": "Multikey", "controller": "did:web:sb0.iscc.id", "publicKeyMultibase": %q`, multibase)
	if validFrom != "" {
		vm += fmt.Sprintf(`, "validFrom": %q`, validFrom)
	}
	if validUntil != "" {
		vm += fmt.Sprintf(`, "validUntil": %q`, validUntil)
	}
	if revoked != "" {
		vm += fmt.Sprintf(`, "revoked": %q`, revoked)
	}
	vm += "}"
	doc := fmt.Sprintf(`{"id": "did:web:sb0.iscc.id", "verificationMethod": [%s], "assertionMethod": ["did:web:sb0.iscc.id#k"]}`, vm)
	return []byte(doc)
}

func TestAcceptCheckpoint(t *testing.T) {
	sb0 := readCheckpoint(t, "sb0.iscc.id_checkpoint")
	now := time.Now()

	cases := []struct {
		name       string
		fetcher    Fetcher
		raw        []byte
		observedAt time.Time
		want       Status
	}{
		{
			// The captured sb0 did.json (no validity window) + the real sb0
			// checkpoint: signature matches and the unconstrained key is valid now.
			name:       "verified",
			fetcher:    &fakeFetcher{data: readFixture(t, "sb0.iscc.id_did.json")},
			raw:        sb0,
			observedAt: now,
			want:       StatusVerified,
		},
		{
			// A did.json advertising sb1's prior key: the signature on the sb0
			// checkpoint matches no listed key, so the hub is internally broken.
			name:       "unverified (mismatching key)",
			fetcher:    &fakeFetcher{data: didJSON(sb1Multibase, "", "", "")},
			raw:        sb0,
			observedAt: now,
			want:       StatusUnverified,
		},
		{
			// did.json cannot be fetched (404 -> os.ErrNotExist).
			name:       "unresolvable (not found)",
			fetcher:    &fakeFetcher{err: os.ErrNotExist},
			raw:        sb0,
			observedAt: now,
			want:       StatusUnresolvable,
		},
		{
			// A non-empty, unparseable revoked timestamp: the parser fails closed,
			// so this resolves to unresolvable, NEVER verified (the closed
			// parseTime fail-open issue).
			name:       "unresolvable (malformed revoked)",
			fetcher:    &fakeFetcher{data: didJSON(sb0Multibase, "", "", "not-a-date")},
			raw:        sb0,
			observedAt: now,
			want:       StatusUnresolvable,
		},
		{
			// The correct sb0 signer key, but its validity window closed before the
			// observation time: a good signature by an out-of-window key is rotated,
			// distinct from unverified and unresolvable.
			name:       "rotated (validUntil in the past)",
			fetcher:    &fakeFetcher{data: didJSON(sb0Multibase, "", "2020-01-01T00:00:00Z", "")},
			raw:        sb0,
			observedAt: now,
			want:       StatusRotated,
		},
		{
			// The correct sb0 signer key, but explicitly revoked before the
			// observation time: also rotated/revoked, not unverified.
			name:       "rotated (revoked in the past)",
			fetcher:    &fakeFetcher{data: didJSON(sb0Multibase, "", "", "2020-01-01T00:00:00Z")},
			raw:        sb0,
			observedAt: now,
			want:       StatusRotated,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, info, err := AcceptCheckpoint(context.Background(), tc.fetcher, "https://sb0.iscc.id", tc.raw, tc.observedAt)
			if err != nil {
				t.Fatalf("AcceptCheckpoint error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("Status = %s, want %s", got, tc.want)
			}
			if tc.want == StatusVerified {
				if info.Origin != "sb0.iscc.id/log" {
					t.Errorf("CheckpointInfo.Origin = %q, want sb0.iscc.id/log", info.Origin)
				}
				if info.TreeSize != 10183 {
					t.Errorf("CheckpointInfo.TreeSize = %d, want 10183", info.TreeSize)
				}
				if isZeroRoot(info.Root) {
					t.Errorf("CheckpointInfo.Root is all zero, want the verified root")
				}
			} else if info != (CheckpointInfo{}) {
				t.Errorf("CheckpointInfo = %+v, want zero value on non-verified verdict", info)
			}
		})
	}
}

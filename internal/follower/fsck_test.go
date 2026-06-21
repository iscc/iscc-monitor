// Tests for the M2 fsck root-rebuild wiring in PollHub (fsckMirror). They build ONE
// internally-consistent tlog-tiles log in-process with merkle's testonly.Tree (over
// rfc6962.DefaultHasher) as the single source of truth for both leaf and node
// hashes, serve it through a follower-package fetcher (did.json + signed checkpoint
// + the level-0 hash tile + the framed entry bundle), and drive a verified PollHub
// end-to-end. PollHub verifies the checkpoint, mirrors the tiles via ingestTiles, and
// then rebuilds the accepted root from the SQLiteFetcher and cross-checks it against
// the signed root (fsckMirror -> logclient.RunFsck) — the good-mirror case returns
// (StatusVerified, nil).
//
// The fixture is built so the rebuild legitimately matches: the entry bundle frames
// the SAME leaf preimages the tree is built from (their LeafHashes output equals the
// tree's leaf hashes) and the level-0 hash tile's bottom row equals the tree's leaf
// hashes — so RunFsck's re-derived root equals the signed root. The tree (prover) is
// independent of RunFsck/LeafHashes (verifier), and the bundle/multibase encoders are
// third code paths, so the cross-check is not a tautology.
//
// The corrupted-mirror subtest makes the test non-vacuous: after a clean poll it flips
// one byte of a mirrored tile BLOB in place and calls fsckMirror directly (PollHub's
// ingestTiles would overwrite the corruption first), asserting the rebuild now fails.
// A green-but-wrong RunFsck that ignored the root would pass that case.
package follower

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/transparency-dev/merkle/rfc6962"
	"github.com/transparency-dev/merkle/testonly"
	"github.com/transparency-dev/tessera/api"
	"golang.org/x/mod/sumdb/note"

	"github.com/iscc/iscc-monitor/internal/logclient"
	"github.com/iscc/iscc-monitor/internal/store"
	"github.com/iscc/iscc-monitor/internal/tiles"
)

// fsckOrigin is the synthetic signed-note name + checkpoint origin for the in-test
// log. The keypair is generated in-test, so no real hub key is involved; the name is
// the sb0 host the follower fixtures use so the host->did:web mapping is exercised.
const fsckOrigin = "sb0.iscc.id/log"

// fsckLeaves is the within-one-tile log size (< 256), so the log has exactly one
// partial level-0 hash tile at index 0 and one partial entry bundle at index 0, both
// of width fsckLeaves. This keeps the first green off the 256-leaf tile boundary.
const fsckLeaves = 5

// mirrorLeaves is the shared verified-path fixture size: > 256 so the mirror crosses
// the 256-leaf tile boundary (a full level-0 tile at index 0 + a 44-leaf partial at
// index 1 + the level-1 root tile), exercising ingestTiles and the fsck root-rebuild
// over the full/partial qualifiers. The verified-completion follower tests poll this
// in-process byte-accurate mirror so fsckMirror can rebuild the signed root.
const mirrorLeaves = 300

// b58Alphabet is the base58btc alphabet (Bitcoin ordering), matching the didweb
// package's b58decode so b58encode is its exact inverse.
const b58Alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

// b58encode base58btc-encodes bytes, preserving leading-zero bytes as '1's. It is the
// inverse of didweb.b58decode, used to build the did:key multibase (z6Mk…) the did.json
// advertises so didweb.VerifierKey reproduces exactly the vkey note.GenerateKey made.
func b58encode(b []byte) string {
	num := new(big.Int).SetBytes(b)
	radix := big.NewInt(58)
	zero := big.NewInt(0)
	mod := new(big.Int)
	var sb strings.Builder
	for num.Cmp(zero) > 0 {
		num.DivMod(num, radix, mod)
		sb.WriteByte(b58Alphabet[mod.Int64()])
	}
	for _, c := range b {
		if c != 0 {
			break
		}
		sb.WriteByte('1')
	}
	out := []byte(sb.String())
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return string(out)
}

// multibaseFromVKey extracts the 32-byte Ed25519 public key from a generated vkey
// string ("<name>+<keyid>+<base64(0x01||pub)>") and returns its did:key multibase
// (z6Mk…): the ed25519-pub multicodec header (0xED 0x01) prepended to the pubkey,
// base58btc-encoded with a leading 'z'. didweb.pubkeyFromDID is its exact inverse, so
// resolving the resulting did.json yields the same vkey that signed the checkpoint.
func multibaseFromVKey(t *testing.T, vkey string) string {
	t.Helper()
	parts := strings.SplitN(vkey, "+", 3)
	if len(parts) != 3 {
		t.Fatalf("vkey %q is not in name+keyid+base64 form", vkey)
	}
	enc, err := base64.StdEncoding.DecodeString(parts[2])
	if err != nil {
		t.Fatalf("decode vkey body: %v", err)
	}
	if len(enc) != 33 || enc[0] != 0x01 {
		t.Fatalf("vkey body is %d bytes (alg %#x), want 33 (0x01 || 32-byte ed25519 pub)", len(enc), enc[0])
	}
	pub := enc[1:]
	multicodec := append([]byte{0xED, 0x01}, pub...)
	return "z" + b58encode(multicodec)
}

// leafDeclSchema is the declaration inner note.$schema the leaf preimages carry
// (the same URI projection_test.go uses), so a poll over buildVerifiedMirror folds a
// known schema into the iscc_index projection. The fold is schema-agnostic (ADR-0008)
// — this is just a real-shaped value, never validated.
const leafDeclSchema = "http://purl.org/iscc/schema/iscc-note-0.8.0.json"

// leafISCCID returns the distinct iscc_id the leaf at index i carries — a synthetic
// ISCC:-prefixed string unique per leaf so the projection read-back is a clean
// one-seq-per-id lookup (SeqsForISCCID returns exactly [seq]).
func leafISCCID(i int) string {
	return fmt.Sprintf("ISCC:LEAF%08d", i)
}

// leafPreimages returns n distinct raw leaf preimages, each a valid canonical
// log-entry envelope {iscc_id, note:{$schema}} with a per-leaf iscc_id so
// logclient.BundleProjections decodes every entry bundle of buildVerifiedMirror
// (plaintext would make the projection fold error on every verified-path poll). fsck
// runs rfc6962.DefaultHasher.HashLeaf over each, so the tree is built from these SAME
// bytes (tree.AppendData) and the bundles frame the SAME bytes — the signed root
// stays self-consistent and fsck still rebuilds it.
func leafPreimages(n int) [][]byte {
	out := make([][]byte, n)
	for i := range out {
		out[i] = []byte(fmt.Sprintf(
			`{"$schema":"log-entry","iscc_id":%q,"note":{"$schema":%q}}`,
			leafISCCID(i), leafDeclSchema))
	}
	return out
}

// encodeBundle frames raw records into a tlog-tiles entry bundle (each record prefixed
// with its big-endian uint16 length, then concatenated) — the C2SP encoding
// api.EntryBundle.UnmarshalText decodes inside LeafHashes. This is the test's
// independent encode path.
func encodeBundle(records [][]byte) []byte {
	var out []byte
	for _, rec := range records {
		var prefix [2]byte
		binary.BigEndian.PutUint16(prefix[:], uint16(len(rec)))
		out = append(out, prefix[:]...)
		out = append(out, rec...)
	}
	return out
}

// mirrorBundleFetcher serves ONE internally-consistent tlog-tiles log: the did.json
// (advertising the generated key), the signed checkpoint, and the byte-accurate
// level-0 hash tile + entry bundle for EVERY enumerated coord of the tree. It routes
// by URL suffix on the canonical tlog-tiles paths so PollHub fetches consistent bytes
// for every coord ingestTiles walks; the tile/bundle branches precede the checkpoint
// fallback so those URLs never receive the checkpoint bytes. Unlike the recording
// fetcher in ingest_test.go (synthetic per-URL bytes), these bytes are byte-accurate
// to the tree's signed root, so the fsck rebuild legitimately matches. byPath maps
// each canonical tile/bundle path to its bytes; an unmapped non-did.json URL falls
// back to the checkpoint.
type mirrorBundleFetcher struct {
	didDoc     []byte
	checkpoint []byte
	byPath     map[string][]byte
}

func (f mirrorBundleFetcher) Fetch(_ context.Context, url string) ([]byte, error) {
	if strings.HasSuffix(url, "did.json") {
		return f.didDoc, nil
	}
	for path, body := range f.byPath {
		if strings.HasSuffix(url, path) {
			return body, nil
		}
	}
	return f.checkpoint, nil
}

// verifiedMirror is one byte-accurate in-process verified-log fixture: the routing
// fetcher, the underlying testonly.Tree (the single source of truth for roots), the
// signed checkpoint bytes, the tree size, and the signed-note key id. The follower's
// verified-path tests poll this fixture so the newly-wired fsckMirror can rebuild the
// signed root from the mirror — the real sb0 checkpoint fixture cannot supply a
// byte-accurate mirror (its leaf preimages are not captured), and real-sb0 signature
// parity stays covered by the internal/logclient tests + the derive_vkey.py oracle.
type verifiedMirror struct {
	fetcher    mirrorBundleFetcher
	tree       *testonly.Tree
	checkpoint []byte
	size       uint64
	keyID      uint32
}

// buildVerifiedMirror builds a consistent in-process log of `leaves` records over the
// RFC-6962 hasher and returns the fixture. The tree (built from the same preimages
// framed into the entry bundles and laid into the level-0 hash tiles) is the single
// source of truth: its leaf hashes fill the tiles and its root is signed into the
// checkpoint, so PollHub's mirror-then-rebuild matches the signed root. It serves the
// byte-accurate level-0 hash tile and entry bundle for every coord ingestTiles walks
// (across the 256-leaf tile boundary when leaves > 256).
func buildVerifiedMirror(t *testing.T, leaves int) verifiedMirror {
	t.Helper()

	preimages := leafPreimages(leaves)
	tree := testonly.New(rfc6962.DefaultHasher)
	tree.AppendData(preimages...)
	size := tree.Size()

	// Generate a synthetic keypair named for the origin and sign the checkpoint body
	// "<origin>\n<size>\n<base64(root)>\n" — the exact C2SP tlog-checkpoint framing.
	skey, vkey, err := note.GenerateKey(rand.Reader, fsckOrigin)
	if err != nil {
		t.Fatalf("note.GenerateKey: %v", err)
	}
	signer, err := note.NewSigner(skey)
	if err != nil {
		t.Fatalf("note.NewSigner: %v", err)
	}
	body := fmt.Sprintf("%s\n%d\n%s\n", fsckOrigin, size, base64.StdEncoding.EncodeToString(tree.Hash()))
	signed, err := note.Sign(&note.Note{Text: body}, signer)
	if err != nil {
		t.Fatalf("note.Sign: %v", err)
	}

	keyID, err := logclient.KeyIDFromVerifier(vkey)
	if err != nil {
		t.Fatalf("KeyIDFromVerifier: %v", err)
	}

	byPath := make(map[string][]byte)
	// One hash tile per coord. Each node at (tileLevel, tileIndex) is the RFC-6962
	// node hash at the matching tree position; the test recomputes it via a compact
	// range over the leaf hashes it covers (equivNodeHash, shared with
	// equivocation_test.go), matching the tree so the
	// served tiles are byte-accurate across the 256-leaf tile boundary and the upper
	// hash-tile levels (level >= 1). fsck rebuilds the root from the level-0 tiles +
	// bundles; the upper tiles are mirrored by ingestTiles but byte-accurate anyway.
	for _, c := range tiles.TileCoords(size) {
		treeLevel := c.Level * uint64(tiles.TileHeight)
		first := c.Index * tiles.TileWidth
		var nodes [][]byte
		for n := uint64(0); n < tiles.TileWidth; n++ {
			treeIndex := first + n
			if (treeIndex << treeLevel) >= size {
				break
			}
			nodes = append(nodes, equivNodeHash(t, tree, treeLevel, treeIndex, size))
		}
		raw, err := api.HashTile{Nodes: nodes}.MarshalText()
		if err != nil {
			t.Fatalf("HashTile.MarshalText (level %d index %d): %v", c.Level, c.Index, err)
		}
		byPath[tiles.TilePath(c.Level, c.Index, c.Partial)] = raw
	}
	// One entry bundle per coord, framing the same preimages the tile covers.
	for _, c := range tiles.BundleCoords(size) {
		first := c.Index * tiles.TileWidth
		last := first + tiles.TileWidth
		if last > size {
			last = size
		}
		byPath[tiles.EntriesPath(c.Index, c.Partial)] = encodeBundle(preimages[first:last])
	}

	return verifiedMirror{
		fetcher: mirrorBundleFetcher{
			didDoc:     didJSON(multibaseFromVKey(t, vkey)),
			checkpoint: signed,
			byPath:     byPath,
		},
		tree:       tree,
		checkpoint: signed,
		size:       size,
		keyID:      keyID,
	}
}

// TestPollHubFsck proves PollHub rebuilds the accepted root from the freshly-mirrored
// SQLiteFetcher and cross-checks it against the signed checkpoint root (fsckMirror).
// The two subtests make the wiring non-vacuous: the good mirror returns
// (StatusVerified, nil); a corrupted mirror BLOB makes the rebuild's fsckMirror return
// a non-nil fault — a green-but-wrong RunFsck that ignored the root would fail that.
func TestPollHubFsck(t *testing.T) {
	ctx := context.Background()

	t.Run("RebuildsSignedRoot", func(t *testing.T) {
		m := buildVerifiedMirror(t, fsckLeaves)
		s, _ := openTemp(t)
		hubID, err := s.UpsertHub(ctx, "sb0.iscc.id", fsckOrigin, "https://sb0.iscc.id")
		if err != nil {
			t.Fatalf("UpsertHub: %v", err)
		}

		// PollHub verifies, mirrors via ingestTiles, then fscks the mirror against the
		// signed root. A consistent mirror yields a clean verified verdict with no fault.
		status, err := PollHub(ctx, s, m.fetcher, hubID, "https://sb0.iscc.id", time.Unix(1, 0), noopAlert, nil)
		if err != nil {
			t.Fatalf("PollHub over a consistent mirror = %v, want nil (fsck must rebuild the signed root)", err)
		}
		if status != logclient.StatusVerified {
			t.Fatalf("status = %s, want verified", status)
		}
	})

	t.Run("RejectsCorruptedMirror", func(t *testing.T) {
		m := buildVerifiedMirror(t, fsckLeaves)
		s, _ := openTemp(t)
		hubID, err := s.UpsertHub(ctx, "sb0.iscc.id", fsckOrigin, "https://sb0.iscc.id")
		if err != nil {
			t.Fatalf("UpsertHub: %v", err)
		}

		// First, a clean poll populates the mirror and passes fsck.
		if _, err := PollHub(ctx, s, m.fetcher, hubID, "https://sb0.iscc.id", time.Unix(1, 0), noopAlert, nil); err != nil {
			t.Fatalf("initial PollHub: %v", err)
		}

		// Corrupt the mirrored level-0 hash tile in place: re-read it, flip one byte,
		// and re-record at the same (hub, level, index, width) key so the upsert
		// overwrites. The rebuilt root now diverges from the signed root.
		f := store.SQLiteFetcher{Store: s, HubID: hubID}
		raw, err := f.ReadTile(ctx, 0, 0, uint8(m.size))
		if err != nil {
			t.Fatalf("ReadTile seed: %v", err)
		}
		corrupt := make([]byte, len(raw))
		copy(corrupt, raw)
		corrupt[0] ^= 0xff
		if err := s.RecordTile(ctx, hubID, 0, 0, int(m.size), corrupt, time.Unix(2, 0)); err != nil {
			t.Fatalf("RecordTile corrupt: %v", err)
		}

		// fsckMirror is called directly (not through PollHub, whose ingestTiles would
		// re-fetch and overwrite the corruption first) to prove the rebuild genuinely
		// compares the re-derived root against the signed root.
		if err := fsckMirror(ctx, s, m.fetcher, hubID, "https://sb0.iscc.id"); err == nil {
			t.Fatal("fsckMirror over a corrupted mirror = nil, want non-nil (the rebuild must compare against the signed root)")
		}
	})
}

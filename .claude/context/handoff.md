## 2026-06-21 — Certificate downloadable proof-bundle assembler — `GET /inclusion/{iscc_id}.bundle`

**Done:** `GET /inclusion/{iscc_id}.bundle` now serves a self-contained, machine-readable proof
bundle — `{iscc_id, hub{domain,did}, checkpoint (verbatim signed-note text), inclusion
(IsccLogInclusionProof-shaped), record (base64-Std), key?}` — for a certifiable id, gated on the SAME
§3 fail-closed re-verification as the page ✓ (the bundle is offered only when the built inclusion
proof actually rebuilt the accepted root). The certificate's previously-disabled "Download proof
bundle (coming soon)" button is now an enabled `<a href="{{.IsccID}}.bundle" download>` when
`.HasBundle`, falling back to the disabled placeholder otherwise. `ots?` (§5) is omitted (`omitempty`)
— the OTS store seam does not exist yet.

**Files changed:**
- `internal/certificate/handler.go`: added `bundleSuffix` dispatch in `Handler` (strip `.bundle`
  BEFORE `index.Decode`, route to `serveBundle`); added the `proofBundle` / `bundleHub` / `bundleKey`
  JSON view-models and the `bundleArtifacts` struct; widened `buildData` to return
  `(certData, bundleArtifacts, int)` surfacing the §2 `raw` checkpoint, the verified §3 `builtProof` +
  `record`, and the §4 cached key/keyID it already computes; set `data.HasBundle = true` at the §3
  re-verification success point (the single gate); added `serveBundle` (JSON write, attachment header,
  drop-the-write-error-after-200 posture ported from `proofserve.writeEvidence`); added `encoding/json`
  import; updated file + `buildData` + `certData` docstrings.
- `internal/certificate/cert.html`: replaced the disabled button with the `{{if .HasBundle}}` enabled
  download link vs disabled placeholder; updated the honesty-panel copy (the download "lands in a later
  release" → "Download the bundle below and re-verify it offline" when a bundle exists); added the
  `.action-download-enabled` CSS variant (DS tokens `--text-link`/`--surface-card`, no CDN).
- `internal/certificate/bundle_test.go` (new): golden + mutation HTTP-seam tests.

`cmd/iscc-monitor/main.go` was NOT touched — the mount is already `certificate.PathPrefix` (subtree),
so `.bundle` routes to the same handler. Scope: 2 source files + 1 test file.

**Verification:** `mise run check` → green (all 21 packages `ok`; build + vet + test). Per-criterion:
- [x] `go test -count=1 -run TestCertificate ./internal/certificate` → PASS (all existing §1-§4/§6 +
  the 4 new bundle tests).
- [x] `go test -count=1 -v -run TestCertificateProofBundle ./internal/certificate` → PASS:
  `TestCertificateProofBundle` (JSON + attachment header + inclusion-proof hashes equal the tree
  prover + base64 record `leaf-0` + key id `40b74463`, AND `VerifyInclusionEvidence(...) == nil`),
  `TestCertificateProofBundleLinkRendered`, `TestCertificateProofBundleTileGap`,
  `TestCertificateProofBundleContradictory`.
- [x] Oracle gate (APPLIES — on the crypto path): `go test -count=1 ./internal/logclient
  ./internal/follower ./cmd/notecheck` all `ok`.
- [x] WASM/purity: `GOOS=js GOARCH=wasm go build ./internal/index ./internal/didweb` exit 0.
- [x] No new dependency: `git diff --stat HEAD -- go.mod go.sum` empty.
- [x] `gofmt -l .` empty.
- [x] Assertion (honest not-available): the tile-gap and contradictory-tree fixtures return 200
  `application/json` with `{iscc_id, error:"no proof bundle available for this id"}`, NO attachment
  header, NO `IsccLogInclusionProof` member — never a fabricated bundle, never a 5xx.

**Mutation (non-vacuity — reviewer can reproduce both):**
1. `proof.VerifyInclusion(...) == nil` → `... == nil || true` in the §3 branch makes BOTH
   `TestCertificateProofBundleContradictory` AND `TestCertificateInclusionProofContradictory` FAIL (the
   mutated handler offers a bundle / renders §3 whose proof does not rebuild tree B's accepted root).
2. An unconditional `data.HasBundle = true` (decoupling the bundle from the §3 gate, inserted before
   the final `return`) makes BOTH `TestCertificateProofBundleTileGap` AND
   `TestCertificateProofBundleContradictory` FAIL (a bundle is served with an attachment header where
   an honest "not available" was expected).
   Both mutations reverted; tree clean; tests green.

**Next:** The proof-bundle assembler completes the largest unblocked M-UI certificate criterion. The
remaining certificate work is §5 BITCOIN ANCHOR, still BLOCKED on the non-existent OTS/anchor store
seam (no `anchor`/`ots` store method) — landing it needs the store seam first, then the `ots` bundle
member + §5 clause. The other open M-UI items are the WASM in-browser re-verifier (tier-2 live verdict;
the static "Verify independently →" link is the placeholder) and a possible consistency-proof bundle
member if the M-UI exit review wants one (deferred per `next.md` — a single-id certificate has no
natural prior `from` size). Suggest: either the OTS store seam (unblocking §5 + the `ots` bundle
member) or the WASM verifier next.

**Notes:**
- `bundleArtifacts` widens `buildData`'s return rather than re-running the crypto chain (per `next.md`):
  the HTML path ignores the extra value (`data, _, status := buildData(...)`), the bundle path reads it.
  This keeps the §3 re-verification the SINGLE gate for both the page ✓ and the bundle — no second
  Merkle run, no second gate to drift.
- The bundle's `checkpoint` and `inclusion.checkpoint` both carry the verbatim signed-note TEXT (not
  base64), matching `InclusionEvidence.Checkpoint` — that is the body a client checks the hub signature
  on. Binary members (`record`, the proof siblings) are base64-Std, byte-identical to the §3 page chips
  and verify-for-me.
- The §4 `did:web:` + raw-domain `host:port` mis-render (`normal`, in `learnings/certificate.md`) is
  carried into the bundle's `hub.did` verbatim — I deliberately reused the SAME `"did:web:" + Domain`
  string §4 already builds (no regression, no expansion of scope per `next.md`). It is latent on the
  clean testnet realm (`sb0.iscc.id`/`sb1.amlet.id`, no port); fold the `%3A`-encode fix in when the
  DID-building string is next touched.
- `serveBundle`'s not-available branch returns a `map[string]string` `{iscc_id, error}` (the only
  non-fixed-shape write in the file); a `map[string]string` cannot fail to marshal for content reasons,
  so the drop-the-write-error-after-200 posture holds. The success path uses the fixed `proofBundle`
  struct.
- The link-rendered test asserts on the absence of a `.bundle"` link for a malformed (non-certifiable)
  id; the not-found template state carries no `actions` block at all, so there is no enabled-vs-disabled
  toggle to assert there — the disabled placeholder only appears on the certifiable-but-§3-declined
  page state, which the gap/contradictory bundle tests already exercise at the data layer.

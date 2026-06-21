# Next Work Package

## Step: Certificate downloadable proof-bundle assembler — `GET /inclusion/{iscc_id}.bundle`

## Advances
target.md **M-UI — Evidence Ledger frontend**, certificate Verify criterion:

> "the **realm-wide certificate** (`/inclusion/{iscc_id}` …) for a known id renders the numbered
> evidence clauses … and offers a **downloadable proof bundle** `{checkpoint, inclusion/consistency
> proof, record bytes, hub key, ots?}` …"

This is the **largest remaining unblocked** M-UI criterion (state.md "Next Milestone" §1; handoff
`**Next:**`). §5 BITCOIN ANCHOR is BLOCKED on the non-existent OTS store seam; the proof bundle is not
(its `ots?` member is optional and stays absent until OTS lands). It also re-engages the
oracle/conformance gate — it reuses the §3 build+verify crypto path verbatim — so the crypto gate stays
live on this clause. Milestone work, not a preempting issue: the certificate's "Download proof bundle"
action is currently a disabled `coming soon` placeholder (`cert.html:389`).

## Goal
Serve a self-contained, machine-readable proof bundle for a certifiable id —
`{checkpoint (raw signed-note text), inclusion proof, record bytes, hub key}` — that a client verifies
on its own (the Proof-bundle / Verifiable-cache contract), and wire the certificate's currently-disabled
"Download proof bundle" button to it. Same fail-closed re-VERIFICATION gate as §3: the bundle is only
offered when the built inclusion proof actually rebuilds the accepted root.

## Scope
- **Create**: `/workspace/iscc-monitor/internal/certificate/bundle_test.go` — the golden + mutation
  HTTP-seam test (test file, not counted toward the ≤3 budget).
- **Modify** (2 non-test source files, within the ≤3 budget):
  - `/workspace/iscc-monitor/internal/certificate/handler.go` — dispatch the `.bundle` suffix in
    `Handler`, add a `proofBundle` JSON view-model + a `serveBundle` writer, and surface the raw
    artifacts `buildData` already computes (checkpoint `raw`, leaf `record`, verified `builtProof`) so
    the bundle reuses the §3 crypto path without re-deriving Merkle. Set a `data.HasBundle` flag for the
    HTML page. Update the file/`buildData`/`certData` docstrings.
  - `/workspace/iscc-monitor/internal/certificate/cert.html` — replace the disabled
    `<button … disabled>Download proof bundle (coming soon)</button>` (line 389) with an enabled
    `<a class="action-download" href="{{.IsccID}}.bundle">Download proof bundle</a>` shown only when
    `.HasBundle`, falling back to the disabled placeholder otherwise; update the honesty-panel copy that
    says the download "lands in a later release" (lines 385). Add the enabled-link CSS variant (no CDN
    URL, reuse DS tokens).
  - `/workspace/iscc-monitor/cmd/iscc-monitor/main.go` — **only if** routing requires it; prefer
    dispatching inside `certificate.Handler` on the path suffix so this file stays untouched (the mount
    is already `certificate.PathPrefix`). If left untouched, drop it from this list in the advance.
- **Reference** (read before implementing):
  - `/workspace/iscc-monitor/.claude/context/learnings/certificate.md` — read FIRST: the §3 rebuild-gate
    mechanics, the base64-Std cross-surface convention, the `fixtureStoreTiled` must-seed-entry-bundles
    rule, the `html/template` base64-`+`→`&#43;` escape nuance, and `KeyIDFromCheckpoint` grounding.
  - `/workspace/iscc-monitor/.claude/context/learnings.md` Correctness rule: *"gate a rendered
    ✓/Merkle assertion on a re-VERIFICATION, not a status flag"* — the bundle MUST be gated on the §3
    re-verification, never offered on a flag.
  - `/workspace/iscc-monitor/internal/proofserve/handler.go` — `serveVerify` (476-590) is the verbatim
    proof build→verify path; `writeEvidence`/`writeVerdict`/`encodeJSON` (1043-1117) are the JSON-write
    posture (drop-the-write-error-after-200) to port.
  - `/workspace/iscc-monitor/internal/logclient/inclusioncheck.go` — `InclusionEvidence` (45-51) is the
    hub-compatible inclusion-proof JSON shape (`type`/`checkpoint`/`treeSize`/`leafIndex`/
    `inclusionProof`) to embed/mirror; `VerifyInclusionEvidence` (82+) is the external-oracle cross-check
    the test asserts on.
  - `/workspace/iscc-monitor/internal/certificate/handler_test.go` — `encodeBundle` (469),
    `fixtureStoreTiled` (506-596), `TestCertificateInclusionProof` (613),
    `TestCertificateInclusionProofContradictory` (706) and the §4 real-`raw` test (~801) — the
    real-mirror fixture + signed-checkpoint `raw` the bundle test reuses unchanged.
  - `/workspace/iscc-monitor/internal/store/checkpoints.go` — `CheckpointAt` (157) returns the `raw`
    checkpoint bytes; `LookupHubKey` (454) + `HubKey` (62-78) give the cached key (`PubkeyZ`, `KeyID`,
    `Revoked`).

## Not In Scope
- **OTS / Bitcoin anchor (`ots?` member, §5).** The OTS store seam does not exist; the bundle's `ots`
  field stays absent (`omitempty`), never a fabricated/empty stamp. That is the next step after this.
- **Consistency proof in the bundle.** The criterion says "inclusion/consistency proof", but a
  certificate's subject claim is an *inclusion* proof; a consistency proof needs a client-supplied prior
  `from` size with no natural value on a single-id certificate. Ship inclusion now; defer consistency to
  a follow-up if the M-UI exit review wants it.
- **WASM in-browser re-verification (tier-2 result).** The static "Verify independently →" link stays;
  the live verdict is the WASM milestone.
- **The §4 `did:web:` + raw-`host:port` DID-encoding fix** (filed `normal`) and the `hubDomain`
  ForceQuery fix (filed `normal`) — fold in ONLY if you actually edit the §4 DID-building string or the
  registry code; do not expand scope to chase them. If you DO emit the §4 DID in the bundle, reuse the
  same `"did:web:" + Domain` string §4 already builds (do not regress it) — or, if you encode the port,
  `%3A`-encode it (don't hand-roll).
- **A new top-level mux route in `main.go`** if it can be avoided — prefer dispatching inside
  `certificate.Handler` on the suffix.
- Interpreting the id/schema beyond what §3/§6 already do (ADR-0008: schema-agnostic).

## Implementation Notes
- **Routing — keep it inside `certificate.Handler`.** The handler already owns the whole `/inclusion/`
  subtree (`PathPrefix`). Dispatch on the suffix: `GET /inclusion/<id>.bundle` is the bundle request,
  everything else the HTML page. Prefer the `.bundle` suffix over a `?format=` query — it gives the
  download a clean filename and a distinct path. After `strings.TrimPrefix(r.URL.Path, PathPrefix)`,
  detect+strip `.bundle` (`strings.HasSuffix` / `TrimSuffix`) BEFORE `index.Decode`, then route to
  `serveBundle` instead of the template execute. `main.go`/`buildMux` stay untouched (the mount is
  already `certificate.PathPrefix`).
- **Reuse `buildData`'s crypto path, do not re-implement Merkle.** `buildData` already computes Position,
  the §2 root, the built+verified `builtProof`, the leaf `record` bytes, and the §4 key — and gates
  `HasClause3` on `proof.VerifyInclusion(...) == nil`. Surface the raw artifacts it has in hand rather
  than running the chain twice: widen the return to also yield the checkpoint `raw []byte`, the leaf
  `record []byte`, and `builtProof [][]byte` (e.g. a `bundleArtifacts` struct alongside `certData`); the
  HTML path ignores the extra value. This keeps the §3 re-verification the single gate for both the page
  ✓ and the bundle.
- **Bundle shape (JSON).** A small local struct, base64-Std for binary fields (learnings: byte-identical
  roots/siblings across surfaces):
  - `iscc_id` — the canonical subject id.
  - `hub` — resolved domain + the §4 `did:web:<domain>` DID.
  - `checkpoint` — the verbatim signed-note checkpoint **text** from `CheckpointAt`'s `raw` (this is the
    `golang.org/x/mod/sumdb/note` text body a client checks the signature on; keep it as text, matching
    `InclusionEvidence.Checkpoint`, NOT base64).
  - `inclusion` — an `InclusionEvidence`-shaped member (`type:"IsccLogInclusionProof"`, `checkpoint`,
    `treeSize`, `leafIndex`, `inclusionProof:[]base64`) so it feeds straight into
    `VerifyInclusionEvidence`. Prefer embedding `logclient.InclusionEvidence` directly.
  - `record` — the leaf's raw record bytes, base64-Std.
  - `key` — `{ id:"%08x", multibase:PubkeyZ, revoked?:RFC3339 }` from `LookupHubKey`.
  - `ots` — **omitted** (`omitempty`).
- **Fail closed, same as §3 (the load-bearing correctness rule).** Only assemble + offer the bundle when
  the §3 re-verification succeeded (`data.HasClause3`). If §3 declined (tile gap, `ErrLeafOutOfBundle`,
  or a proof that did not rebuild the root), the `.bundle` request is an honest **200** "no proof bundle
  available for this id" (a small JSON `{error:…}`), NEVER a fabricated bundle, never a 5xx for a
  coverage gap. A genuine DB/read fault stays a 500 (buffer-then-write, same posture as the page). Set
  `data.HasBundle = data.HasClause3` so the page only shows the live download link when a verified bundle
  actually exists.
- **Content-Type + filename.** `application/json` with
  `Content-Disposition: attachment; filename="<id>.bundle.json"`. Use the same
  drop-the-write-error-after-200 posture as `proofserve.writeEvidence` (a fixed-shape struct of
  strings/uints/[]string cannot fail to marshal for content reasons).
- **Test (reuse the §3 fixture verbatim).** `bundle_test.go`:
  `fixtureStoreTiled(t, "sb1.amlet.id", goldenID, seq, leaves, nil, false, raw)` with a REAL signed
  `raw` (copy the §4 test's signed-checkpoint setup so `key` populates and the bundle is complete).
  Assert: (1) `GET /inclusion/<golden>.bundle` → 200 `application/json` with the attachment header, body
  decodes to a bundle carrying the subject id, the `IsccLogInclusionProof` member whose `inclusionProof`
  hashes equal the §3 page's `ProofHashes`, the base64 record bytes, and key id `40b74463`; (2) the
  assembled `InclusionEvidence` **re-verifies** — `logclient.VerifyInclusionEvidence(ctx, fetcher, ev)
  == nil` (the external-oracle cross-check, the conformance gate); (3) the cert HTML page for the same id
  now renders an enabled download `<a href="…bundle">`, and a non-certifiable id keeps the disabled
  placeholder.
- **Mutation (prove non-vacuity — the gate review reproduces).** Gating the bundle on the §3
  re-verification must be load-bearing: a bundle-offered-unconditionally mutation, OR running the
  contradictory-tree fixture (`TestCertificateInclusionProofContradictory`'s `treeB.Hash()` accepted
  root), must make the bundle test FAIL — no bundle offered / `VerifyInclusionEvidence` returns non-nil.
  State this in the advance notes so review can reproduce it.

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...`, `gofmt -l .` empty).
- `go test -count=1 -run TestCertificate ./internal/certificate` passes (all existing §1-§4/§6 tests
  plus the new bundle tests).
- `go test -count=1 -v -run TestCertificateProofBundle ./internal/certificate` passes: the bundle is
  served as JSON with the inclusion-proof member, record bytes, and key, AND its `InclusionEvidence`
  re-verifies via `logclient.VerifyInclusionEvidence(...) == nil`.
- Oracle gate unbroken: `go test -count=1 ./internal/logclient ./internal/follower ./cmd/notecheck` all
  `ok` (this clause is ON the crypto path — the gate APPLIES, not N/A).
- WASM/purity unbroken: `GOOS=js GOARCH=wasm go build ./internal/index ./internal/didweb` exit 0.
- Assertion: requesting the bundle for an id whose §3 declined (the tile-gap fixture, or the
  contradictory-tree fixture) returns 200 with NO bundle (honest "not available"), never a fabricated
  bundle and never a 5xx — and `VerifyInclusionEvidence` on any bundle that IS served returns nil.
- No new dependency: `git diff --stat HEAD -- go.mod go.sum` is empty.
- Scope discipline: at most 3 non-test source files touched
  (`internal/certificate/handler.go`, `internal/certificate/cert.html`, and `cmd/iscc-monitor/main.go`
  only if routing forced it) plus the new test file; no §5 / OTS / consistency / §4-DID / registry work
  done.

## Done When
`GET /inclusion/{iscc_id}.bundle` serves a self-contained, externally-re-verifiable proof bundle for a
certifiable id (gated on the §3 re-verification), the certificate page links it via an enabled download
action, the bundle test is mutation-proven non-vacuous, and all Verification criteria pass including the
oracle cross-check.

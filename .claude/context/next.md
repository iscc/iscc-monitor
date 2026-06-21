# Next Work Package

## Step: Certificate §2 CHECKPOINT clause — render the accepted (size, root)

## Advances
target.md **M-UI — Evidence Ledger frontend** Verify criterion (the last open M-UI criterion):

> the **realm-wide certificate** (`/inclusion/{iscc_id}` …) for a known id renders the numbered
> evidence clauses (subject + position; **checkpoint `(size, root)`**; inclusion proof; signing key;
> anchor state; full per-id record history incl. any deletion) …

This is the §2 clause of the per-surface certificate landmark list ("§1 Subject · **§2 Checkpoint
(size, root)** · §3 Inclusion proof …", target.md Certificate mockup region). §1 SUBJECT is sound and
PASS-verified at HEAD (340303b); the `review` handoff `**Next:**` is explicit: "Proceed to the §2
Checkpoint clause (`HasClause2`): render the accepted `(size, root)` the cap already keys on, reusing
the `HubSummary.LastSize` carry." This is the next slice in the certificate clause-by-clause arc.

## Goal
Make the certificate's §2 CHECKPOINT clause real for a certifiable id: render the hub's accepted
checkpoint `(size, root)` — the same accepted tree the §1 cap already keys on (`hub.LastSize`). The
root is read back via `store.CheckpointAt(hubID, LastSize)` and rendered base64-Std, matching every
sibling SSR surface (log browser, verify-for-me). This converts the §2 placeholder from `HasClause2 ==
false` (renders nothing) into a populated clause without rework — the gated-clause template structure
already exists.

## Scope
- **Modify**:
  - `/workspace/iscc-monitor/internal/certificate/handler.go` — extend `certData` with
    `CheckpointSize uint64` + `CheckpointRoot string` (base64-Std); in `buildData`, in the
    `Certifiable = true` branch (after the accepted-tree cap passes), read
    `st.CheckpointAt(r.Context(), hub.HubID, hub.LastSize)` and populate `CheckpointSize = hub.LastSize`,
    `CheckpointRoot = base64.StdEncoding.EncodeToString(root)`, and set `HasClause2 = true`. A
    `CheckpointAt` DB error → 500 (buffer-then-200 already in place); a `found == false` leaves
    `HasClause2 = false` (no fabricated checkpoint — honest absence). (1 of ≤3 non-test source files.)
  - `/workspace/iscc-monitor/internal/certificate/cert.html` — fill the existing `{{if .HasClause2}}`
    §2 block (currently an empty `.clause-value`) with the accepted size + root + an honest note.
    (2 of ≤3.)
- **Modify (tests, not counted)**: `/workspace/iscc-monitor/internal/certificate/handler_test.go` —
  extend `TestCertificateKnownID` to assert the §2 clause renders the accepted size + base64-Std root;
  assert a non-certifiable case renders NO §2 clause.
- **Reference**:
  - `/workspace/iscc-monitor/.claude/context/learnings/certificate.md` — package mechanics
    (decode→resolve→ListHubs→cap chain, buffer-then-200, fail-closed 200 discipline). Read before editing.
  - `/workspace/iscc-monitor/.claude/context/learnings/store.md` — `CheckpointAt` / `HubSummary.LastSize`
    semantics (the accepted root is NOT persisted in follow_state; `CheckpointAt(hubID, treeSize)` reads
    it back; absent → `found=false, nil err`).
  - `/workspace/iscc-monitor/internal/store/checkpoints.go` (lines 143-169) — `CheckpointAt` signature
    `(root []byte, raw []byte, found bool, err error)`.
  - `/workspace/iscc-monitor/internal/proofserve/handler.go` (lines 511, 657) — the established SSR root
    encoding: `base64.StdEncoding` (the log browser `browserData.Root` and `VerifyVerdict.Root` both use
    it — match it for cross-surface consistency).
  - `/workspace/iscc-monitor/internal/certificate/cert.html` (lines 336-341) — the existing empty
    `{{if .HasClause2}}` §2 block to fill; (lines 188-217) — the `.clause` / `.clause-marker` /
    `.clause-value` / `.clause-mono` / `.clause-note` CSS classes to reuse (no new CSS).
  - `.claude/design/ISCC Monitor - Certificate.dc.html` — the §2 clause layout/copy (subordinate to the
    ADR/PRD hard constraints — flag any conflict).

## Not In Scope
- **§3 Inclusion proof and the downloadable proof-bundle assembler.** §3 re-engages the
  oracle/conformance crypto gate (the served inclusion proof must be mutation-proven non-vacuous against
  the hub's `IsccLogInclusionProof` / `notecheck`). §2 is a pure store read + render — keep this slice
  inside the N/A-oracle envelope so it lands clean before the crypto step. Leave `HasClause3..6 == false`.
- §4 signing key, §5 Bitcoin anchor, §6 record history.
- The separate Bitcoin-anchor vs comparison-anchor panels.
- The Hub-List `hubDomain` `ForceQuery` fail-open fix (the deferred `normal`): this step does not touch
  `internal/registry/registry.go`, so do NOT fold it in here — it waits for a step that edits `hubDomain`.
- The ADR-0011 Go 1.26 / iscc-lib bump (its own foundational increment; needs a Go 1.26 toolchain).
- Any new store method or second store round-trip beyond the single `CheckpointAt` call.

## Implementation Notes
Both populated fields live in `buildData`'s certifiable branch; the template's `{{if .HasClause2}}`
structure already exists, so only its inner `.clause-value` needs filling.

- **Reuse `hub.LastSize`, do NOT re-derive the accepted size.** `followedHub` already returns the
  `store.HubSummary` carrying `LastSize`; the cap branch (`hub.LastSize == 0` / `seqs[0] >= hub.LastSize`)
  has already proven `LastSize > 0` by the time you populate §2, so `CheckpointSize = hub.LastSize` needs
  no extra read. Only the *root* needs a store call.
- **Root read seam is `CheckpointAt(ctx, hub.HubID, hub.LastSize)`** (`checkpoints.go:157`), returning
  `(root []byte, raw []byte, found bool, err error)`. You only need `root`; ignore `raw` here (the raw
  signed-note bytes belong to the §3 proof-bundle step). A DB `err != nil` → `return certData{},
  http.StatusInternalServerError` (the existing 500 idiom; the handler buffers before 200). A
  `found == false` is the rare honest gap — leave `HasClause2 = false` and still render the certifiable
  §1 banner; never fabricate a root. Realistically `found` is always true on the certifiable path
  because `AdvanceAccepted` records the checkpoint at the same `tree_size` it advances `last_size` to,
  but the fail-closed branch keeps the render honest.
- **Encode the root base64-Std** (`base64.StdEncoding.EncodeToString(root)`), matching
  `proofserve/handler.go:511,657` (the log browser + verify-for-me) so the certificate's root string is
  byte-identical to what the rest of the federation surfaces show. Add `encoding/base64` to the imports.
- **`html/template` auto-escapes** `{{.CheckpointRoot}}` / `{{.CheckpointSize}}` (the template is already
  `html/template`, not `text/template`).
- **Template: fill the existing `{{if .HasClause2}}` block** (cert.html:336-341, currently
  `<div class="clause-value"></div>`). Render the accepted size + base64 root in a `.clause-mono` value
  and an honest `.clause-note` — e.g. value `size {{.CheckpointSize}} · root {{.CheckpointRoot}}` and a
  note like "The accepted checkpoint (size + RFC-6962 tree head) the monitor vouches for; this id's
  position ({{.Position}}) falls within it." Reuse the existing `.clause-*` CSS classes (no new CSS).
  Do NOT imply pre-coverage guarantees (ADR-0001 coverage honesty — Correctness rule).
- **Correctness rules in play:** *Coverage honesty (ADR-0001)* — §2 renders only the accepted-tree
  checkpoint, never a contradicted/unaccepted one; the cap already guarantees `Position < LastSize`. The
  *one origin/leaf* and *iscc_id→seq one-to-many* rules are unaffected (no new lookup).
- **store stays a leaf** — `CheckpointAt` is an existing store read; you add no store method and no
  net/http to store. Oracle/conformance gate is **N/A** for this slice (pure store read + HTML render;
  no signature/RFC-6962/Merkle/did:web/fsck/proof path) — say so in the advance notes; the gate APPLIES
  starting at §3.
- **Test (non-vacuous):** extend `internal/certificate/handler_test.go`. The existing
  `TestCertificateKnownID` fixture calls `AdvanceAccepted` with `Root: []byte("root")` and an accepted
  size — assert the rendered HTML for the certifiable id now contains the §2 marker (`§2 CHECKPOINT`),
  the accepted size, AND `base64.StdEncoding.EncodeToString([]byte("root"))`. Make it non-vacuous: a
  `TestCertificateUnacceptedLeaf` / `TestCertificateNotInLog` case must NOT render `§2 CHECKPOINT` (it is
  not Certifiable). Mutation check to record for review: neutering `HasClause2 = true` (or rendering a
  hardcoded wrong root) makes the §2 assertion FAIL.

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...`, `gofmt -l .` empty).
- `go test -count=1 ./internal/certificate` passes uncached.
- `go test -count=1 -run TestCertificateKnownID ./internal/certificate` passes and the response body
  contains `§2 CHECKPOINT`, the accepted tree size, and `base64.StdEncoding.EncodeToString([]byte("root"))`
  (the fixture's accepted root) — proving §2 renders the real accepted `(size, root)`.
- A non-certifiable id (`TestCertificateUnacceptedLeaf` / `TestCertificateNotInLog`) renders NO §2
  clause (`§2 CHECKPOINT` absent from the body).
- Mutation check (reviewer reproduces): neutering `HasClause2 = true` → the §2 assertion in
  `TestCertificateKnownID` FAILS; rendering a hardcoded wrong root → the base64-root assertion FAILS.
- `GOOS=js GOARCH=wasm go build ./internal/index` still succeeds (no-regression sanity check; this step
  does not touch `internal/index`).

## Done When
`mise run check` is green and the certificate's §2 CHECKPOINT clause renders the hub's accepted
`(size, root)` (size from `hub.LastSize`, root base64-Std from `CheckpointAt`) for a certifiable id and
nothing for a non-certifiable one, with the §2 assertion mutation-proven non-vacuous.

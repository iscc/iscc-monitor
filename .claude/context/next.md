# Next Work Package

## Step: Richer frozen Exhibit ("size before → presented" + evidence ref) + §5 fork/shrink pseudo-transition fix

## Advances
The lone open **`critical`** issue — *"Hub dossier richer frozen Exhibit (size before → presented,
evidence ref) + §3/§1 honesty — increment 2b of 2"* (issues.md) — which is the immediate, code-closable
DONE blocker. It advances the **M-UI** milestone Verify criterion / design-parity bar for the hub-dossier
surface: target.md M-UI names *"the frozen **Exhibit** (required above) above the sections"* and ADR-0006's
*"Irreplaceable evidence … the contradictory evidence below is preserved permanently"*. The matching
mockup region (`.claude/design/ISCC Monitor - Hub Dossier.dc.html`) shows the Exhibit's
"Tree size before → Then presented" + an "Evidence ref". This is the lone `critical` and preempts the
`normal` backlog; the handoff `**Next:**` (review `06c7749`) points directly here ("Increment 2b").

Increment 2b lists four sub-parts. This step closes the **two that share the same two files** (the
frozen-evidence honesty arc) — the richer Exhibit and the §5 fork/shrink fix — and defers the §3
size/time decouple (touches a 4th prod file `store/hubs.go`) and the §1 "resolved" wording (a design
call) to keep this step ≤3 files and coherent. See `## Not In Scope`.

## Goal
Make the frozen Exhibit render the two contradictory checkpoints' tree sizes ("size <before> →
<presented>") plus a stable, honest evidence ref derived from the raw bytes — and stop §5 from rendering
a nonsensical `size N → N` / `larger → smaller` pseudo-transition on a fork / equivocation / shrink frozen
hub. Both are frozen-evidence honesty fixes the loud Exhibit currently understates.

## Scope
- **Create**: `internal/logclient/checkpointsize.go` — one exported pure helper
  `CheckpointSizeFromRaw(raw []byte) (treeSize uint64, ok bool)` that reads the tree size from a raw
  hub-signed checkpoint note **without re-verifying the signature** (the stored evidence is already
  signature-verified). Plus its test `internal/logclient/checkpointsize_test.go`.
- **Modify** (≤3 non-test/doc prod files):
  1. `internal/logclient/checkpointsize.go` *(the new file above — counts as one of the 3)*.
  2. `internal/dossier/handler.go` — add the Exhibit "before → presented" + evidence-ref fields to
     `violationRow` + `violationRows`/`buildData`; fix the §5 transition loop to skip non-increasing pairs.
  3. `internal/dossier/dossier.html` — render the new Exhibit fields in the `class="exhibit"` section.
- **Test/doc (not counted)**: `internal/dossier/handler_test.go` (extend `frozenHub` +
  `TestDossierFrozenExhibit`, add a fork §5 case), `internal/logclient/checkpointsize_test.go`,
  `.claude/context/learnings/dossier.md`, `.claude/context/learnings/logclient.md`.
- **Reference** (read before implementing):
  - `.claude/context/learnings/dossier.md` — the numbered-layout rules, the §3 decouple + §1 wording
    notes (both deferred here), and the §5 monotonic-size **TRAP** (the exact bug + Codex's confirmed fix:
    skip non-increasing pairs, gate the singleton on a real transition).
  - `.claude/context/learnings/logclient.md` — the package-local verify/transport purity notes.
  - `internal/logclient/checkpointkey.go` — **port this pattern**: `note.Open(raw, note.VerifierList())`
    with an EMPTY verifier list returns `*note.UnverifiedNoteError` whose `.Note.Text` is the checkpoint
    body; read the tree size from body line 2 (mirror `verify.go`'s `parseCheckpointBody` leading-zeros +
    `strconv.ParseUint` handling). This is the unverified-but-honest read the Exhibit needs.
  - `internal/logclient/verify.go:72-102` — `parseCheckpointBody` (the verified-path size parser to
    mirror; it is unexported, so the new helper re-derives the same line-2 parse).
  - `internal/follower/follower.go:464-480` — `freeze()`: confirms `RawA = prevRaw` (prior accepted
    checkpoint), `RawB = raw` (the contradictory presented checkpoint) — both raw signed-note bytes.
  - `internal/store/checkpoints.go:262-270` — `store.Violation{RawA, RawB, ProofJSON}` shape.
  - `internal/dossier/handler.go:368-377` (`violationRows`) + `408-461` (`observationRows`/`freezeLine`).
  - `internal/dossier/dossier.html:466-491` — the current Exhibit `<section class="exhibit">` region.
  - `internal/dossier/handler_test.go:67-101` (`frozenHub`) + `103-149` (`TestDossierFrozenExhibit`).
  - `testdata/live/sb0.iscc.id_checkpoint` — a captured REAL hub-signed checkpoint (use as a real
    `RawA`/`RawB` fixture so the size parse is proven against ground truth, not a hand-built string).

## Not In Scope
- **The §3 frozen size/time decouple `normal`** (select `observed_at` for the `tree_size = f.last_size`
  row in `internal/store/hubs.go`) — it touches a 4th prod file and is its own §3 rework; leave for the
  next 2b sub-step.
- **The §1 "resolved"-vs-unresolvable wording `normal`** — a DESIGN call (the mockup specifies the static
  "Key resolved from" phrasing); do not silently rewrite mockup-specified copy here. Defer to a design pass.
- **Re-verifying the checkpoint signature in the Exhibit parse.** The stored `RawA`/`RawB` are already
  signature-verified evidence; the Exhibit only DISPLAYS the size each checkpoint claimed. Do NOT pull a
  vkey / did:web resolution into the dossier (it would break the dossier's minimal closure and is unneeded).
- **Rendering `ProofJSON` / raw bytes / a download in the Exhibit** — the proof-bundle surface owns that.
- Touching any M1/M2/M3/WASM/OTS/M-Deploy source, or the dashboard/proofserve overlay duplication.

## Implementation Notes
- **The new `CheckpointSizeFromRaw` is PURE (stdlib + `golang.org/x/mod/sumdb/note` only — no
  `net`/`os`/`sqlite`)**, like its sibling `KeyIDFromCheckpoint`. Port the empty-verifier-list pattern
  verbatim: `note.Open(raw, note.VerifierList())` → `errors.As(err, &ue)` where `ue
  *note.UnverifiedNoteError` → `ue.Note.Text` is the body. Then parse line 2 exactly as
  `parseCheckpointBody` does (reject leading zeros except "0", `strconv.ParseUint(_, 10, 64)`). Return
  `(0, false)` on any malformed / non-note input — **fail closed**: the Exhibit must degrade to an honest
  "size unavailable" rather than fabricate a number (the current `frozenHub` seeds `RawA:[]byte("a")`,
  which is NOT a note — that path must render gracefully). A bare checkpoint body without a signature line
  is still an `UnverifiedNoteError` only if `note.Open` accepts the framing; treat any non-`UnverifiedNoteError`
  (and a body with `<2` lines) as `(0, false)`.
- **Exhibit fields:** add to `violationRow` (the natural home, since the Exhibit ranges over it)
  `SizeBefore`/`SizePresented uint64`, a `HasSizes bool` gate (true only when BOTH `RawA` and `RawB` parse),
  and `EvidenceRef string`. `violationRows` (which already takes `[]store.Violation`) calls
  `CheckpointSizeFromRaw(v.RawA)` + `(v.RawB)` and derives the ref. Derive the evidence ref as a **stable
  short hash over the pair** — `sha256(RawA||RawB)` rendered as the first ~12 hex chars — never a fabricated
  id; it is non-empty whenever the violation carries any raw bytes (a stable handle even when the sizes are
  unparseable). When `HasSizes` is false the template shows the kind + detected-at + the existing "do not
  trust new state" copy with no fabricated size line.
- **§5 fork/shrink fix** (`observationRows`, `handler.go:430-443`): in the transition loop skip pairs where
  `newer.TreeSize <= older.TreeSize` (a same-size fork or a shrunk contradictory checkpoint is not a
  transition), and only emit the oldest-checkpoint singleton when **at least one real increasing transition
  was emitted** (track a `bool`). A frozen hub then renders the freeze pointer + any real growth history,
  never `size N → N` or `larger → smaller`. The freeze-pointer line + the anchor line are unchanged.
- **Correctness rules (learnings.md):** (1) the always-loaded SSR-honesty rule — *do not render an
  assertion a reader trusts that the data does not support*; here both fixes REMOVE a misleading line, so
  the failure mode is fabricating a size — hence fail-closed on an unparseable `RawA`/`RawB`. (2) the
  `proof/verify` purity rule generalizes: keep the new logclient helper WASM-shareable (no
  `net`/`os`/`sqlite`) — verify with `GOOS=js GOARCH=wasm go build ./internal/logclient`. (3) Origin/vkey
  rules are untouched (no signature path). The oracle/conformance gate is **N/A** — this reads a stored
  evidence body's plaintext size + renders HTML; it touches no signature / RFC-6962 / Merkle / did:web /
  fsck / proof **verification** path.
- `html/template` auto-escapes the new `.SizeBefore`/`.EvidenceRef` fields; keep the no-JS / no-CDN bans
  green (no `<script>` / `<button>` / ` hidden>` / CDN URL — the Exhibit non-dismissable bans in
  `TestDossierFrozenExhibit` still apply). `go.mod`/`go.sum`/`schema.sql` stay byte-identical.

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...`, `gofmt -l .` empty).
- `GOOS=js GOARCH=wasm go build ./internal/logclient` succeeds (the new helper stays WASM-shareable).
- `go test -count=1 -run TestCheckpointSizeFromRaw ./internal/logclient` passes:
  `CheckpointSizeFromRaw(<testdata/live/sb0.iscc.id_checkpoint bytes>)` returns `(size>=1, true)` and
  byte-equals `VerifyCheckpoint`'s `treeSize` for the same fixture; `CheckpointSizeFromRaw([]byte("not-a-note"))`
  returns `(0, false)`.
- `go test -count=1 -run TestDossierFrozenExhibit ./internal/dossier` passes: a frozen fixture whose
  `RawA`/`RawB` are REAL signed checkpoints of differing sizes renders "size <before> → <presented>" with
  the parsed numbers + a non-empty evidence ref, AND a frozen fixture with non-note `RawA`/`RawB` renders
  the Exhibit (kind + detected-at + "do not trust new state") without a fabricated size.
- `go test -count=1 -run TestDossierObservationLog ./internal/dossier` passes including a NEW fork case:
  an accepted size-N checkpoint + a same-size/different-root contradictory checkpoint renders **NO**
  `size N → N` line in §5 (and a shrink fixture renders no `larger → smaller` line), while the
  `froze hub (split view)` pointer still appears.
- Mutation check (reviewer-runnable, non-vacuous): reverting the `newer.TreeSize <= older.TreeSize` skip
  makes the fork §5 test FAIL; making `CheckpointSizeFromRaw` return a constant makes
  `TestDossierFrozenExhibit` FAIL.

## Done When
`mise run check` is green and the four `go test -run …` checks above pass — the frozen Exhibit renders the
two contradictory checkpoints' parsed tree sizes + a stable evidence ref (honestly absent when unparseable),
and §5 no longer renders a `size N → N` / `larger → smaller` pseudo-transition on a frozen hub.

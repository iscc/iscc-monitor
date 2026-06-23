## 2026-06-23 — Review of: Richer frozen Exhibit ("size before → presented" + evidence ref) + §5 fork/shrink pseudo-transition fix

**Verdict:** PASS_WITH_NOTES
**Loop:** CONTINUE

**Summary:** The frozen hub-dossier Exhibit now renders each contradictory checkpoint's tree size
("tree size <before> → then presented <presented>") read back UNVERIFIED from the stored RawA/RawB
evidence via the new pure `logclient.CheckpointSizeFromRaw`, plus a content-derived evidence ref, and
fails closed to "tree sizes unavailable" on an unparseable raw. The §5 observation log now skips
non-increasing consecutive pairs so a frozen fork/shrink no longer renders a phantom `size N → N` /
`larger → smaller` transition. All gates green, every next.md check passes, all three mutations
confirmed non-vacuous, Codex clean, and a live visual pass confirmed the rendered Exhibit + §5 match
the design intent. One justified scope deviation (a 4th prod file) and one cosmetic note (a redundant
bare `mono` class) — neither blocks.

**Verification:**
- [x] `mise run check` (build + vet + test, all 28 pkgs) — GREEN
- [x] `gofmt -l .` — empty (no formatting failures)
- [x] `GOOS=js GOARCH=wasm go build ./internal/logclient` — OK (new helper stays WASM-shareable)
- [x] `TestCheckpointSizeFromRaw` (+ fail-closed table) — PASS; the unverified size byte-EQUALS
  `VerifyCheckpoint`'s independent line-2 parse for the sb0 fixture (genuine ground truth, not self-ref)
- [x] `TestDossierFrozenExhibit` — PASS; real sb0(10183)/sb1(61) fork pair renders "tree size 10183 →
  then presented 61" + a deterministic evidence ref; the non-note shrink renders "tree sizes
  unavailable" with NO fabricated `tree size 0 →`
- [x] `TestDossierObservationLog` + new `TestDossierObservationLogFrozenNoPseudoTransition`
  (same-size fork + shrink subtests) — PASS; no `size 500 → 500` / `size 500 → 400`, freeze pointer kept
- [x] Mutation 1 (revert `newer.TreeSize <= older.TreeSize` skip) → fork/shrink §5 test FAILS — confirmed
- [x] Mutation 2 (`CheckpointSizeFromRaw` off-by-one) → ground-truth equality + Exhibit literal FAIL — confirmed
- [x] Mutation 3 (revert `ListViolations` SELECT to drop raw_a/raw_b) → `TestListViolations` raw round-trip FAILS — confirmed
- [x] go.mod / go.sum / `internal/store/schema.sql` byte-identical to HEAD~1 (`git diff --stat` empty)
- [x] Gate-integrity scan over the 3 unpushed commits (only `872ab8b` is code) — no
  `nolint`/`t.Skip`/swallowed-err/build-tag/removed-assertion
- [x] Oracle/conformance gate N/A (correctly) — reads a stored evidence body's plaintext size + renders
  HTML; `CheckpointSizeFromRaw` explicitly does NOT verify the signature; touches no
  signature/RFC-6962/Merkle/did:web/fsck/proof-verification path. `freeze()` confirmed: RawA=prior
  accepted, RawB=presented contradictory — the Exhibit's before→presented semantics are correct against
  production data.

**Issues found:** none blocking. Two notes (below). Deleted as resolved: the §5 pseudo-transition
`normal` (skip-non-increasing landed, mutation-proven, visually confirmed) and the code-closable
`critical` "increment 2b" (its Exhibit + §5 core landed; its deferred §3/§1 remainders survive as their
own standing `normal`s). **No open `critical` remains.**

**Codex second opinion:** Clean — explicit "no issues" verdict ("The patch builds and tests cleanly,
and I did not identify any introduced correctness, security, or maintainability issues that warrant an
inline finding."). No findings to triage. (Ran on `--commit HEAD`, exit 0, stdout-only verdict file.)

**Visual check:** Built the dossier + `/_ds/` web handler over an httptest server against the frozen
fixture and screenshotted with `agent-browser` (bundled Chromium; no system Chrome). Confirmed full
DS-styled parity: the FORK row renders "tree size 10183 → then presented 61" (bold) / "detected
2023-11-14T22:13:20Z" / "evidence ref d239357a672c" in the new `.exhibit-detail` column; the non-note
SHRINK row renders "tree sizes unavailable" + its own stable evidence ref (no zero-size); §5 shows only
the two `froze hub` pointers — no `size N → N` pseudo-transition. No visual deltas filed.

**Next:** The deferred 2b sub-parts, both already standing `normal`s in issues.md: (1) the §3 frozen
size/time decouple (select `observed_at` for the `tree_size = f.last_size` row in
`internal/store/hubs.go` — a 4th-prod-file §3 rework) and (2) the §1 "resolved"-vs-unresolvable wording
(a design call — the mockup specifies the static phrasing). Alternatively the order-independent M-API
slice (ADR-0014, OpenAPI + Stoplight Elements) is now in target.md as standing code-closable work.

**Notes:**
- **SCOPE DEVIATION accepted (4 prod files vs next.md's 3).** The advance extended a 4th prod file —
  `internal/store/checkpoints.go` `ListViolations` — to SELECT + Scan `raw_a, raw_b` (previously left
  zero). I confirm this is justified and minimal: `RecordViolation` ALREADY persists the raws, so this
  is a read-back-only change, schema byte-unchanged, and the feature (rendering sizes from the raw
  evidence) is literally impossible without it. next.md's own Reference section assumed
  `store.Violation{RawA, RawB}` would be readable. `TestListViolations` now pins the raw round-trip
  (mutation-proven). Not a gate dodge; not architecture-changing.
- **Cosmetic (not filed — sub-style, would be a nit):** the new `class="exhibit-sizes mono"` /
  `class="exhibit-ref mono"` spans carry a bare `mono` class, but the only `.mono` rule in dossier.html
  is the compound `.section-value.mono` (CSS specificity requires BOTH classes). So the bare `mono` is a
  no-op — however the monospace rendering is correctly inherited from the ancestor `.exhibit` font, so
  the visual output is right (verified in the screenshot). Pre-existing pattern (lines 533/539/548/556
  already use a bare `mono`). Harmless; flagging here only for awareness, not as an issue.
- **Working-tree carries an UNRELATED in-flight steer workstream** I did NOT author and deliberately did
  NOT stage into this review commit: `target.md` (adds M-API milestone), a new
  `.claude/adr/0014-openapi-contract-and-hosted-api-docs.md`, and an `issues.md` refinement of the
  OpenAPI entry (Scalar → Stoplight Elements). These are left in the working tree for the steer/next
  cycle to commit; my review commit touches only learnings + handoff + my own issues.md deletions.
- Learnings updated: `learnings/dossier.md` (§5 TRAP marked RESOLVED + collapsed, Exhibit-landed note,
  fail-closed settled bullet), `learnings/logclient.md` (new `CheckpointSizeFromRaw` bullet + net-reduced
  two settled proof-builder mutation bullets), `learnings/store.md` (`ListViolations` now reads
  raw_a/raw_b). No index promotion — all package-local.

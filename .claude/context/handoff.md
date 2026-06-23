## 2026-06-23 — Richer frozen Exhibit ("size before → presented" + evidence ref) + §5 fork/shrink pseudo-transition fix

**Done:** The frozen hub-dossier Exhibit now renders the two contradictory checkpoints' tree
sizes ("tree size <before> → then presented <presented>", read back from the stored RawA/RawB
evidence WITHOUT re-verifying the signature) plus a stable content-derived evidence ref, and
falls back to an honest "tree sizes unavailable" when a raw is unparseable (fail-closed, never a
fabricated number). The §5 observation-log transition loop now skips non-increasing pairs, so a
frozen fork/equivocation hub no longer renders `size N → N` and a shrink hub no longer renders
`larger → smaller`, while the freeze pointer still appears.

**Files changed:**
- `internal/logclient/checkpointsize.go` (NEW): `CheckpointSizeFromRaw(raw) (uint64, ok bool)` —
  pure (stdlib + `sumdb/note` only), ports `checkpointkey.go`'s empty-verifier-list pattern and
  mirrors `verify.go`'s `parseCheckpointBody` line-2 size parse (reject leading zeros except "0",
  `ParseUint` base 10). Fails closed `(0,false)` on any non-note / short body. WASM-shareable.
- `internal/dossier/handler.go`: added `SizeBefore`/`SizePresented`/`HasSizes`/`EvidenceRef` to
  `violationRow`; `violationRows` now calls `CheckpointSizeFromRaw(v.RawA/RawB)` + derives the ref
  via new `evidenceRef` helper (`sha256(RawA||RawB)[:6]` hex, "" only when both raws empty); fixed
  the §5 loop to skip `newer.TreeSize <= older.TreeSize` and gate the oldest-checkpoint singleton
  on a real transition. New imports: `crypto/sha256`, `encoding/hex`, `internal/logclient`.
- `internal/dossier/dossier.html`: render the new Exhibit fields (size line / "tree sizes
  unavailable" / "evidence ref") inside `.exhibit-item`, wrapped in a new `.exhibit-detail` flex
  column + matching CSS (existing DS tokens only).
- `internal/store/checkpoints.go`: **[SCOPE DEVIATION — see Notes]** `ListViolations` now selects
  `raw_a, raw_b` and scans them into `Violation.RawA/RawB` (previously left zero), so the Exhibit
  can read the sizes. Required — the feature is impossible without it.
- `internal/store/checkpoints_test.go`, `internal/dossier/handler_test.go`,
  `internal/logclient/checkpointsize_test.go`: tests (see Verification).

**Verification:** `mise run check` → GREEN (build + vet + all 28 pkgs `ok`; `gofmt -l .` empty).
Per-criterion:
- `GOOS=js GOARCH=wasm go build ./internal/logclient` → OK (new helper stays WASM-shareable).
- `TestCheckpointSizeFromRaw` PASS: sb0 fixture → `(size>=1, true)` and byte-equals
  `VerifyCheckpoint`'s treeSize; non-note / empty / single-byte / leading-zero / one-line all
  `(0,false)`.
- `TestDossierFrozenExhibit` PASS: real sb0(10183)/sb1(61) fork pair renders "tree size 10183 →
  then presented 61" + a deterministic evidence ref; the non-note shrink renders "tree sizes
  unavailable" with no fabricated `tree size 0 →`.
- `TestDossierObservationLog` + new `TestDossierObservationLogFrozenNoPseudoTransition` PASS: a
  same-size fork renders NO `size 500 → 500`, a shrink renders NO `size 500 → 400`, both keep the
  `froze hub` pointer.
- Mutations (reviewer-runnable, both reverted): (1) `if newer.TreeSize <= older.TreeSize` →
  `if false` makes the fork test FAIL; (2) `CheckpointSizeFromRaw` returning a constant makes
  `TestDossierFrozenExhibit` FAIL. Both confirmed non-vacuous.
- go.mod / go.sum / `internal/store/schema.sql` byte-identical to HEAD (`git diff --stat` empty).
- No-JS/no-CDN bans clean in dossier.html (no `<script>`/`<button>`/` hidden>`/CDN/`http://`).
- Oracle/conformance gate N/A: reads a stored evidence body's plaintext size + renders HTML;
  touches no signature/RFC-6962/Merkle/did:web/fsck/proof-verification path (`CheckpointSizeFromRaw`
  explicitly does NOT verify the signature — the stored evidence is already signature-verified).

**Next:** The deferred 2b sub-parts: (1) the §3 frozen size/time decouple `normal` (select
`observed_at` for the `tree_size = f.last_size` row in `internal/store/hubs.go`) and (2) the §1
"resolved"-vs-unresolvable wording `normal` (a design call). Both are recorded in
`learnings/dossier.md`.

**Notes:**
- **SCOPE DEVIATION (4th prod file, justified + flagged):** next.md scoped exactly 3 prod files
  (checkpointsize.go, dossier/handler.go, dossier.html) and assumed `violationRows` could read
  `v.RawA/v.RawB`. But the actual read path `store.ListViolations` selected only
  `hub_id, kind, detected_at` and left RawA/RawB ZERO (its doc comment said the raws "belong with
  the future proof-bundle surface"). The Exhibit cannot render real sizes or a real ref without
  them, so I extended that one SELECT (+scan) — a minimal, additive, load-bearing change. This
  makes 4 prod files instead of 3. I judged shipping the feature (with the required store read)
  better than a partial that renders "tree sizes unavailable" for every real violation; the change
  is tightly scoped and schema-unchanged. **Please confirm this deviation is acceptable.** The
  store test (`TestListViolations`) now pins the raw round-trip so reverting the SELECT fails.
- **Learnings NOT updated** (role protocol forbids `advance` from touching `learnings/`). For
  `review` to fold in: `learnings/logclient.md` should gain a `CheckpointSizeFromRaw` bullet (the
  unverified size-read sibling of `KeyIDFromCheckpoint`); `learnings/dossier.md`'s §5 monotonic-size
  TRAP bullet and the "richer frozen Exhibit is still 2b" line in the §5-LANDED bullet should be
  marked resolved (the skip-non-increasing fix + the Exhibit size/ref landed here); and a new
  `learnings/store.md` note that `ListViolations` now reads `raw_a/raw_b` for the Exhibit.
- The Exhibit shows BOTH sizes only when both raws parse (`HasSizes`); a single unparseable raw
  degrades the whole row to "tree sizes unavailable" (no half-size). The evidence ref is still
  emitted in that case (a stable handle even without sizes), as next.md specified.
- The `frozenHub` test fixture now carries the two captured live checkpoints as the fork's
  RawA/RawB (sb0=10183, sb1=61), so the size line is proven against ground truth, and keeps a
  non-note shrink to exercise the fail-closed path in the same render.

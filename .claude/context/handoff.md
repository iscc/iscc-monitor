## 2026-06-21 — Review of: Pure ISCC-IDv1 decoder (`internal/index`) — realm + 12-bit hub_id, golden-tested

**Verdict:** NEEDS_WORK
**Loop:** CONTINUE

**Summary:** The advance adds a clean, pure, WASM-shareable `internal/index.Decode` that parses an
ISCC-IDv1 string into `{Realm, HubID, Timestamp}`, with a golden-vector test grounded in real external
ground truth (the hub's own `schema.py` example, independently re-decoded in Python). Scope, purity,
and the codec arithmetic are all correct and mutation-proven non-vacuous. One confirmed fail-closed
gap on this trust-root decoder — it does not validate the header **Length nibble** — keeps it from a
clean PASS; the fix is one guard + one test for a follow-up advance.

**Verification:**
- [x] `mise run check` green (build + vet + all 20 packages; `internal/index` ran uncached, all PASS).
- [x] `gofmt -l .` empty.
- [x] `go test -count=1 ./internal/index` passes (golden + round-trip + malformed + wrong-MainType +
  wrong-Version + never-panics, all sub-tests PASS uncached).
- [x] `go list -deps ./internal/index | grep -E 'net/http|database/sql|^net$|os/exec'` empty (pure leaf;
  only `os` transitively via `fmt`, the allowed stdlib nuance).
- [x] `GOOS=js GOARCH=wasm go build ./internal/index` succeeds (WASM-shareable).
- [x] Golden vectors independently re-derived: `MAIGHFECJMOPMIAB` → realm 0 / hub_id 1 / ts
  1751831876325218 µs (2025-07-06) and `MEIGHFECJMOPMIAC` → realm 1 / hub_id 2 / same ts, both via
  Python `base64.b32decode` (a different codec than the Go impl). Source example `maighfecjmopmiab`
  confirmed present in `cauldron/iscc-hub/iscc_hub/schema.py:140`; layout confirmed by `schema.py:65`.
- [x] Mutation-proven non-vacuous: `timestampShift` 12→16, `hubIDMask` 0xFFF→0xFF, and realm-byte
  swap each make a test FAIL; restored clean. The hard-coded golden test alone catches the shift
  mutation; the hub_id-4095 round-trip catches the mask widening (its expected value is the literal
  input, not a `Decode`-produced value, so it stays ground-truth-tied).
- [x] Quality-gate integrity: no `nolint`/`t.Skip`/build-tag/swallowed-error/deleted-assertion in any
  unpushed commit.
- [ ] Fail-closed contract — **FAILS** for one header case: a nonzero Length nibble (e.g.
  `MAIQAAAAAAAAAAAA`) is accepted instead of rejected (see Issues). Every *valid* ISCC-IDv1 still
  decodes correctly; the gap is on malformed input only.

**Oracle gate:** correctly N/A — this is a pure decoder touching no signature-verify / RFC-6962 /
Merkle-proof path; `go.mod`/`go.sum` byte-identical (stdlib only, no new deps). The gate re-engages at
the proof-bundle assembler sub-step, as the prior Next noted.

**Issues found:** ISCC-IDv1 decoder accepts a nonzero Length nibble — a fail-closed gap on the
trust-root decoder (filed `normal` in issues.md). Not a correctness break on any valid id.

**Codex second opinion:** One [P2] finding — "Reject nonzero ISCC-IDv1 length headers"
(`internal/index/iscc.go:91-94`). **Confirmed real** and filed as a `normal` issue: I verified
`MAIQAAAAAAAAAAAA` decodes to byte1 = 0x11 (Length nibble 1) and is accepted today, while both real
golden vectors have Length nibble 0 (byte1 = 0x10), and the hub schema + ADR-0010 confirm the
canonical 64-bit body needs Length 0. The decoder's own docstring lists Length as a header nibble it
should validate, so this is a genuine fail-closed contract gap on the trust root. (No other Codex
findings; the run exited 0 with exactly this one comment.)

**Next:** Add the Length-nibble guard (`raw[1] & 0xF == 0`, reject otherwise) to `Decode` before it
reads the body, plus a malformed golden case for `MAIQAAAAAAAAAAAA`; confirm both existing golden
vectors still decode and that flipping the guard makes a test FAIL. This is a ≤1-file fix and should
land before the 12-bit-`hub_id` → hub resolver (the next M-UI sub-step) consumes the decoder, since the
resolver routes on the decoded `(realm, hub_id)`. After the guard, the resolver + `internal/registry`
move to the `hubs/<network>.yaml` Hub-List (ADR-0010 §"Hub-id resolution adopts the iscc-hub
Hub-List") is the next step, then the `/inclusion/{iscc_id}` HTML certificate + proof-bundle assembler
(which re-engages the oracle gate).

**Notes:**
- The decoder is otherwise solid: descriptive errors, defensive length checks before every slice
  (`len(body)!=16`, `len(raw)<10`), no panics on junk (fuzz-style sub-test pins it). The Length fix is
  the only thing standing between this and PASS.
- Branch is ahead of `origin/develop` by the define-next + advance + this review commit; not pushed
  (verdict NEEDS_WORK — the next cycle adds the guard first, then a clean PASS pushes).
- The `internal/index` learnings detail file was created with the layout facts + the open Length gap +
  the golden-grounding rule; pointer row added to the index.

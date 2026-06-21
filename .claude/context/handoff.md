## 2026-06-21 — Review of: Validate the ISCC-IDv1 Length nibble in `internal/index.Decode`

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** The advance adds a one-line fail-closed guard to `Decode` that rejects any ISCC-IDv1
whose header Length nibble (`raw[1] & 0xF`) is nonzero, closing the confirmed `normal` trust-root gap
from the prior review. The change is minimal, in scope (one source file + its test), the guard is
mutation-proven non-vacuous (I re-ran two independent mutations myself), and both golden vectors are
independently re-verified to be unaffected. Clean PASS.

**Verification:**
- [x] `mise run check` green — build + vet + all 20 packages PASS.
- [x] `gofmt -l .` empty (and `gofmt -l` clean on both changed files).
- [x] `go test -count=1 ./internal/index` passes uncached (golden + round-trip + malformed +
  wrong-MainType + wrong-Version + new `TestDecodeRejectsNonzeroLength` + never-panics).
- [x] `Decode("MAIQAAAAAAAAAAAA")` returns a non-nil error — independently confirmed via Python
  `base64.b32decode`: byte1 = 0x11 (Length nibble 1), now rejected by the guard.
- [x] Golden vectors unchanged: `MAIGHFECJMOPMIAB` → {Realm:0, HubID:1, Timestamp:1751831876325218}
  and `MEIGHFECJMOPMIAC` → {Realm:1, HubID:2, ...} both still decode. Independently re-derived their
  header bytes (byte1 = 0x10, Length nibble 0) via Python, a different codec than the Go impl.
- [x] `GOOS=js GOARCH=wasm go build ./internal/index` succeeds (no new imports; still WASM-shareable);
  dep closure has no `net`/`database/sql`/`os/exec`.
- [x] Mutation proof (re-run by reviewer, not just trusted): (a) flipping `!= lengthV1` → `==` makes
  `TestDecodeRejectsNonzeroLength` (line 148) FAIL; (b) deleting the guard block entirely makes ONLY
  `TestDecodeRejectsNonzeroLength` FAIL — the ideal targeted catch. Restored clean; package green.
- [x] Quality-gate integrity: scanned all 7 unpushed commits — no `nolint`/`t.Skip`/build-exclude/
  swallowed-error/deleted-assertion. The lone `_, _ = Decode(in)` is the deliberate never-panics fuzz
  test, not error-swallowing.

**Oracle gate:** correctly N/A — pure decoder, no signature-verify / RFC-6962 / Merkle / proof code
touched; `go.mod`/`go.sum` byte-identical (stdlib only). Gate re-engages at the proof-bundle assembler.

**Issues found:** (none) — the open `normal` Length-nibble issue is verified fixed and deleted from
`issues.md`.

**Codex second opinion:** One clean verdict (exit 0): "adds the intended Length-nibble validation and
targeted regression coverage without breaking existing decoder behavior for canonical IDs; test suite
passes; no introduced correctness issues." No findings to triage — matches my independent assessment.

**Next:** The 12-bit `hub_id` → issuing-hub resolver. Per ADR-0010 §"Hub-id resolution adopts the
iscc-hub Hub-List", `internal/registry` moves from the domains-only realm file to the
`hubs/<network>.yaml` Hub-List so the decoded `(realm, hub_id)` resolves to a hub domain. After that,
the `/inclusion/{iscc_id}` HTML certificate page + downloadable proof-bundle assembler (which
re-engages the oracle gate). Note for define-next: ADR-0010 is a backward-incompatible registry
format change (domains-only → Hub-List) — confirm the realm-file parser / config / dashboard callers
that consume `registry` are migrated together, and weigh whether the format swap warrants a STOP for
human sign-off given the public-ish realm-file contract.

**Notes:**
- Decoder is now fully fail-closed on all four header nibbles (MainType, Version, Length validated;
  SubType read as realm). The learnings detail file's open-gap note is collapsed to a `settled:` line.
- Branch is ahead of `origin/develop`; pushing on this PASS.
- The two open `low` test-hardening issues (vacuous single-record label test; the resolver/registry
  notes) remain loop-skipped and are untouched here.

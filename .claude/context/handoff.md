## 2026-06-21 — Validate the ISCC-IDv1 Length nibble in `internal/index.Decode`

**Done:** Added a fail-closed guard to `Decode` that rejects any ISCC-IDv1 whose header Length nibble
(`raw[1] & 0xF`) is nonzero, before the 64-bit body is read. This closes the `normal` issue
"ISCC-IDv1 decoder accepts a nonzero Length nibble" — the trust-root gap where a non-canonical header
like `MAIQAAAAAAAAAAAA` (byte1 = 0x11) was mis-read as a valid id and would route the wrong leaf.

**Files changed:**
- `internal/index/iscc.go`: added a `lengthV1 = 0` const next to `versionV1`, and a guard after the
  Version check that returns a descriptive `fmt.Errorf` ("ISCC-IDv1 Length nibble %d is unsupported
  (want %d): %q") when `raw[1] & 0xF != 0`. No change to the body-decode arithmetic. The file's
  docstring already listed Length as a header nibble, so this makes the doc true.
- `internal/index/iscc_test.go`: added `TestDecodeRejectsNonzeroLength`, mirroring
  `TestDecodeWrongVersion`'s shape — builds the malformed header from raw bytes
  (`{0x60,0x11,0,...}`), asserts non-nil error; also asserts the named literal `"MAIQAAAAAAAAAAAA"`
  is rejected; then flips byte1 to `0x10` (Length 0) and asserts it decodes, proving the rejection is
  specifically the Length check.

**Verification:** `mise run check` → green (build + vet + all 20 packages; `internal/index` ran
uncached PASS). Per-criterion:
- [x] `mise run check` green; `gofmt -l .` empty.
- [x] `go test -count=1 ./internal/index` passes (all existing sub-tests + new Length case).
- [x] `Decode("MAIQAAAAAAAAAAAA")` returns non-nil error (asserted in test, byte1 = 0x11).
- [x] Golden vectors unchanged: `MAIGHFECJMOPMIAB` → {Realm:0,HubID:1,Timestamp:1751831876325218},
  `MEIGHFECJMOPMIAC` → {Realm:1,HubID:2,...} (both Length-0, byte1 = 0x10) still decode (golden +
  round-trip tests PASS).
- [x] `GOOS=js GOARCH=wasm go build ./internal/index` succeeds (no new imports; still WASM-shareable).
- [x] Mutation proof: flipping `!= lengthV1` → `== lengthV1` makes `TestDecodeRejectsNonzeroLength`
  (line 148, the `MAIQAAAAAAAAAAAA` assertion) plus golden/round-trip/version tests FAIL; restored
  to `!=` and re-verified green before commit. The guard is non-vacuous.

**Next:** The 12-bit `hub_id` → issuing-hub resolver. Per the prior review's Next and ADR-0010
§"Hub-id resolution adopts the iscc-hub Hub-List", `internal/registry` moves from the domains-only
realm file to the `hubs/<network>.yaml` Hub-List so the decoded `(realm, hub_id)` resolves to a hub
domain. After that, the `/inclusion/{iscc_id}` HTML certificate page + downloadable proof-bundle
assembler (which re-engages the oracle gate).

**Notes:**
- Oracle gate correctly N/A: pure decoder, no signature-verify / RFC-6962 / Merkle path touched;
  `go.mod`/`go.sum` unchanged (stdlib only, no new deps). Gate re-engages at the proof-bundle
  assembler sub-step.
- Stayed in scope: one non-test source file (`iscc.go`) plus its test. The `lengthV1` const is the
  optional minimal polish `next.md` explicitly permitted; I did not rework the other header constants.
- Note for review: when the guard is flipped for the mutation check, several *other* tests (golden,
  round-trip, the version test's Length-0 sanity arm) also fail because they decode Length-0 vectors —
  expected; the load-bearing failure is `TestDecodeRejectsNonzeroLength` itself, which fails on the
  exact case it pins. All restored and green.

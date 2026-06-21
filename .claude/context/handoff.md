## 2026-06-21 — Pure ISCC-IDv1 decoder (`internal/index`) — realm + 12-bit hub_id, golden-tested

**Done:** Added the pure leaf package `internal/index` with `Decode(isccID string) (ISCCID, error)`
that parses an ISCC-IDv1 string into `{Realm, HubID, Timestamp}` (SubType nibble = realm, `body & 0xFFF`
= 12-bit hub_id, `body >> 12` = 52-bit µs timestamp), validating MainType = 6 (ID) and Version = 1 and
failing closed (descriptive error, never panic) on any malformed input. Golden vectors are grounded in
external ground truth, not the implementation's own arithmetic.

**Files changed:**
- `internal/index/iscc.go`: new package + the pure `Decode` + result type `ISCCID`; an unexported
  `encode` helper used only by the round-trip test. Stdlib only (`encoding/base32`, `encoding/binary`,
  `fmt`, `strings`).
- `internal/index/iscc_test.go`: golden-vector table (realm 0 + realm 1, non-zero hub slots, `ISCC:`
  prefix), encode→Decode round-trip (both realms, max slot 4095, near-max timestamp), malformed-input
  table, dedicated wrong-MainType and wrong-Version sub-tests, and a "never panics on junk" guard.

**Verification:** `mise run check` → green (build + vet + all 21 packages; `internal/index` ran uncached
0.004s). Per-criterion:
- [x] `gofmt -l .` empty.
- [x] `go test -count=1 ./internal/index` passes (golden + error-path, all sub-tests PASS).
- [x] `go list -deps ./internal/index | grep -E '^net/http$|^database/sql$|^net$|^os/exec$'` → empty
  (pure leaf; only `fmt`→`os` transitively, the allowed stdlib nuance).
- [x] `GOOS=js GOARCH=wasm go build ./internal/index` → succeeds (WASM-shareable).
- [x] Golden vector decodes to the documented `(realm, hub_id, timestamp)`; every error case (empty,
  prefix-only, too-short/long, non-base32, lowercase, wrong MainType, wrong Version) returns a non-nil
  error and does not panic.
- [x] `go.mod`/`go.sum` byte-identical (stdlib only; no new deps).

**Next:** The 12-bit-`hub_id` → hub resolver + the `internal/registry` move from domains-only `realm.txt`
to the iscc-hub `hubs/<network>.yaml` Hub-List (ADR-0010 §"Hub-id resolution adopts the iscc-hub
Hub-List"): map decoded `(realm, hub_id)` → hub url/domain (realm also keys the per-network DB, ADR-0007).
That step CONSUMES the `ISCCID` this step produces. After resolution lands, the `/inclusion/{iscc_id}`
HTML certificate handler and the proof-bundle assembler follow (the latter re-engages the oracle gate —
reviewer must mutation-prove the served inclusion proof non-vacuous and confirm notecheck/golden parity
on hub-signed material).

**Notes:**
- **Codec is grounded, not invented.** The reference `iscc/iscc-core` `iscc_id.py` (and the hub-side
  `iscc_hub/iscc_id.py`) is NOT vendored in `cauldron/`. I confirmed the exact layout against a real
  ISCC-ID published in `cauldron/iscc-hub/iscc_hub/schema.py` — the resolved-URL example
  `…/iscc_id/maighfecjmopmiab`. Decoding it (independently, via Python `base64.b32decode`) yields header
  bytes `0x60 0x10` → MainType 6 (ID), SubType 0 (realm 0), Version 1, and body `0x6394824b1cf62001` →
  hub_id 1, timestamp 1751831876325218 µs (2025-07-06, a plausible recent date). That ID is the realm-0
  golden vector; the realm-1 vector (`MEIGHFECJMOPMIAC`, hub_id 2) is constructed from first principles
  per ADR-0010's header layout. The expected fields are NOT computed by calling `Decode`/`encode`.
- **ADR caveat confirmed:** next.md warned the old test ids (`MAAGZTFQ…`, `MEAJU5BQ…`, `MAIG…`) might
  carry a different MainType — true for those, so I did NOT reuse them. I initially added a malformed
  row claiming `MEAAGZTFQLCYPKIZ` had a non-ID MainType; on re-checking it decodes to MainType 6 (a
  valid realm-1 ID), so I removed that incorrect row. The wrong-MainType path is covered correctly by
  `TestDecodeWrongMainType` (`AAAAAAAAAAAAAAAA` → MainType nibble 0).
- **ISCC base32 = RFC 4648 uppercase, NO padding** (`base32.StdEncoding.WithPadding(NoPadding)`),
  confirmed against the reference: 80 bits / 5 bits-per-char = exactly 16 chars, no `=`. Lowercase is
  rejected (the alphabet is uppercase-only), as is any char outside `A-Z2-7`.
- **Fail-closed guard kept** per the `derive_vkey.py` short-key precedent: length is checked before every
  slice (`len(body) != 16`, then `len(raw) < 10`), so short/garbage input errors instead of
  index-panicking. A "never panics on junk" test pins this.
- **Oracle/conformance gate is correctly N/A:** this is a pure decoder touching no
  signature-verify / RFC-6962-consistency / Merkle-proof path; `go.mod`/`go.sum` unchanged. The gate
  re-engages two sub-steps later at the proof-bundle assembler, as the prior review's Next noted.
- Scope held to exactly the 2 new files in next.md (`iscc.go` + `iscc_test.go`); no existing call site,
  registry, or `cmd/` touched.

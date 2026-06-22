# learnings: `internal/index` (pure ISCC-IDv1 decoder)

Pure leaf `Decode(isccID string) (ISCCID, error)` — the trust root of the realm-wide certificate
path: a wrong decode resolves the wrong hub and proves the wrong leaf. WASM-shareable (no
`net`/`net/http`/`database/sql`/`os` in the closure; only `os` transitively via `fmt`).

- **ISCC-IDv1 = 80-bit code = 16-bit header + 64-bit body, 16 base32 chars (RFC 4648 uppercase, NO
  `=` padding), optional `ISCC:` prefix.** Header = 4 nibbles: MainType, SubType, Version, **Length**.
  For a v1 ID: MainType nibble (`raw[0]>>4`) = 6, SubType nibble (`raw[0]&0xF`) = **realm** (0
  test/sandbox, 1 operational), Version nibble (`raw[1]>>4`) = 1, **Length nibble (`raw[1]&0xF`) = 0**
  for the canonical 64-bit body. Body = big-endian `uint64`: `timestamp = body>>12` (52 bits, µs since
  epoch), `hub_id = body & 0xFFF` (12 bits, slot 0-4095). Confirmed three ways: ADR-0010:83-88, the
  hub's own `cauldron/iscc-hub/iscc_hub/schema.py:65` ("52-bit timestamp + 12-bit hub ID"), and an
  independent Python `base64.b32decode` of the real published id.

- **The golden vector is real, not invented.** `MAIGHFECJMOPMIAB` is the uppercased form of
  `maighfecjmopmiab`, the metadata-URL example in `cauldron/iscc-hub/iscc_hub/schema.py:140`. It
  decodes (independently, via Python) to realm 0 / hub_id 1 / ts 1751831876325218 µs (2025-07-06).
  The realm-1 vector `MEIGHFECJMOPMIAC` is constructed per the ADR layout. Both have **Length nibble
  0** (byte1 = 0x10) — any new vector must too.

- **settled: all four header nibbles are now validated fail-closed.** `Decode` rejects a wrong
  MainType, wrong Version, AND a nonzero **Length nibble** (`raw[1]&0xF != 0`, the `MAIQAAAAAAAAAAAA`
  case) before reading the body. The Length guard sits between the Version check and the body read;
  both golden vectors (Length nibble 0) are unaffected. Any *new* valid vector must have byte1 low
  nibble 0. (Was the open Codex-found `[review]` gap; closed 2026-06-21, mutation-proven.)

- **Golden test must stay tied to ground truth, not the symbol.** `TestDecodeGoldenVectors` hard-codes
  the decoded fields (external truth) — it catches a `timestampShift` mutation but NOT a `hubIDMask`
  widening (its vectors have hub_id 1,2 < 256). `TestDecodeRoundTrip` covers hub_id 4095 via the
  unexported `encode`, and is still non-vacuous because the *expected* value is the literal input
  struct, not a value `Decode` produced. Any future vector should exercise a hub_id > 255 in a
  hard-coded (non-round-trip) assertion so the full 12-bit mask is pinned by ground truth alone.

- **`encode` is a test-only round-trip helper (unexported, YAGNI).** The monitor only ever decodes ids
  it is handed; do not export an encoder. Keep the public surface to `Decode` + `ISCCID`.

- **ADR-0011 tripwire lives here, NOT in `iscc.go` (`iscclib_tripwire_test.go`).** iscc-lib v0.5.0
  `IsccDecode` rejects every ISCC-IDv1 (Version=1) with the literal error `"iscc: invalid Version: 1"`
  (`codec.go:268`), so the in-repo `Decode` stays the interim ISCC-IDv1 codec until iscc/iscc-lib#43.
  The test imports iscc-lib; `iscc.go` must NOT (the WASM build excludes `_test.go`, so the import never
  leaks — guard with `GOOS=js GOARCH=wasm go build ./internal/index`, and `go list -deps ./internal/index`
  must show iscc-lib absent from the non-test build closure). When the tripwire flips RED, iscc-lib decodes
  ISCC-IDv1 → migrate `Decode` to it and delete the port. Assert the bare golden form; the prefixed form
  hits the same Version reject after `TrimPrefix("ISCC:")` so it adds nothing.

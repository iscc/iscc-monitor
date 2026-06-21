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

- **Length nibble is NOT validated (open `[review]` issue).** `Decode` checks MainType + Version but
  ignores `raw[1]&0xF`, so a header like `MAIQAAAAAAAAAAAA` (byte1 = 0x11, Length nibble 1) is accepted
  and its bytes 2:10 mis-read as the 64-bit body — a fail-closed gap on the trust root the file's own
  docstring claims to enforce ("MainType, SubType, Version, Length"). Codex-found, reviewer-confirmed.
  Fix = guard `raw[1]&0xF == 0` before reading the body; both golden vectors stay valid.

- **Golden test must stay tied to ground truth, not the symbol.** `TestDecodeGoldenVectors` hard-codes
  the decoded fields (external truth) — it catches a `timestampShift` mutation but NOT a `hubIDMask`
  widening (its vectors have hub_id 1,2 < 256). `TestDecodeRoundTrip` covers hub_id 4095 via the
  unexported `encode`, and is still non-vacuous because the *expected* value is the literal input
  struct, not a value `Decode` produced. Any future vector should exercise a hub_id > 255 in a
  hard-coded (non-round-trip) assertion so the full 12-bit mask is pinned by ground truth alone.

- **`encode` is a test-only round-trip helper (unexported, YAGNI).** The monitor only ever decodes ids
  it is handed; do not export an encoder. Keep the public surface to `Decode` + `ISCCID`.

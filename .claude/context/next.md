# Next Work Package

## Step: Pure ISCC-IDv1 decoder (`internal/index`) — realm + 12-bit hub_id, golden-tested

## Advances
M-UI (Evidence Ledger frontend), the last open M-UI Verify criterion:

> "the **realm-wide certificate** (`/inclusion/{iscc_id}`, keyed by the self-describing ISCC-IDv1 —
> decode realm + 12-bit `hub_id`, resolve the issuing hub via the registry) for a known id renders the
> numbered evidence clauses … and offers a **downloadable proof bundle** …"

The certificate is too large for one ≤3-file step (it needs: this decoder → a 12-bit-`hub_id`→hub
resolver / registry-format change → the HTML certificate handler with §1–§6 clauses → the proof-bundle
assembler that re-engages the oracle gate). This step lays the **verifiable skeleton**: the pure,
self-contained codec the whole slice is keyed on. ADR-0010 §"The inclusion certificate is a realm-wide
endpoint" + §"Implementation note" make this decoder the first concrete piece ("decode `(realm, hub_id)`
from the id — port the codec from `iscc/iscc-core` `iscc_id.py` … into the not-yet-built `internal/index`
`index/iscc.go`"). A wrong decode resolves the wrong hub and proves the wrong leaf, so this unit IS the
trust-root oracle for the certificate — it earns a golden-vector test (target Quality bar: "Pure units …
get table-driven golden-vector tests").

## Goal
Add `internal/index` with a pure `Decode(isccID string) (ISCCID, error)` that parses an ISCC-IDv1 string
into its realm (header SubType nibble), 12-bit `hub_id` (`body & 0xFFF`), and 52-bit microsecond
`timestamp` (`body >> 12`), rejecting any input that is not a well-formed ISCC-IDv1. This is the
foundation every later certificate sub-step builds on, and it is WASM-shareable (no I/O), so it must
stay import-clean.

## Scope
- **Create**: `internal/index/iscc.go` (the package + the pure decoder)
- **Create**: `internal/index/iscc_test.go` (golden-vector + error-path table test)
- **Modify**: (none — new leaf package; do NOT touch `cmd/iscc-monitor`, registry, or any handler this
  step)
- **Reference**:
  - `.claude/adr/0010-evidence-ledger-frontend.md:83-119` — the authoritative ISCC-IDv1 layout: 80-bit
    code = 16-bit header + 64-bit body; **SubType nibble = realm** (0 = test/sandbox, 1 = operational);
    body big-endian `uint64`; `timestamp = body >> 12` (52 bits, µs since epoch); `hub_id = body & 0xFFF`
    (12 bits, slot 0–4095). This file wins over any guess.
  - `iscc/iscc-core` `iscc_id.py` (public package, **not vendored in `cauldron/`** — port from the public
    source / ISO 24138 codec, do not invent): the `encode_base32`/`decode_base32` alphabet and the
    `write_header`/`read_header` nibble layout (MainType, SubType, Version, Length). The canonical ISCC
    base32 alphabet is RFC 4648 (`ABCDEFGHIJKLMNOPQRSTUVWXYZ234567`), **uppercase, no `=` padding** —
    confirm against the reference before relying on it.
  - `.claude/derive_vkey.py` — the precedent for a small, dependency-free Go port grounded in a Python
    reference, and the "keep a defensive length check before the assert so a short input errors instead of
    index-panicking" pattern (learnings index: "Keep this guard when porting crypto").
  - `.claude/context/learnings.md` (index) — the `proof/verify` purity rule applies by analogy: keep this
    decoder free of `net`/`net/http`/`database/sql`/`os` so it stays WASM-shareable.

## Not In Scope
- The HTTP handler `/inclusion/{iscc_id}` and the certificate HTML template (`.dc.html` §1–§6 clauses,
  two-tier honesty panel, Download-proof-bundle action) — a later sub-step, after resolution exists.
- The 12-bit-`hub_id` → hub resolver and the `internal/registry` move from domains-only `realm.txt` to
  the iscc-hub `hubs/<network>.yaml` Hub-List (ADR-0010 §"Hub-id resolution adopts the iscc-hub
  Hub-List") — its own M1/registry step; this decoder only PRODUCES the `(realm, hub_id)` it consumes.
- The downloadable proof-bundle assembler and re-plumbing `serveVerify` to retain the raw checkpoint
  bytes + resolved hub key — the oracle-gate sub-step; not touched here.
- ENCODING ISCC-IDs into strings as a public API (the monitor only ever decodes ids it is handed). Add a
  small *unexported* encode helper only if the golden test genuinely needs it to round-trip a vector —
  keep the package's public surface to `Decode` + the result type (YAGNI).
- Wiring the decoder into any existing call site (`SeqsForISCCID`, dashboard, dossier) — pure leaf only.

## Implementation Notes
- **Port, don't invent.** The base32 alphabet and the header-nibble (varnibble) encoding are external
  facts owned by `iscc/iscc-core` `iscc_id.py` / ISO 24138. Port them faithfully; cite the source in the
  file docstring. Do not approximate the alphabet from memory — confirm it is RFC 4648 uppercase-no-pad.
- **Decode pipeline (per ADR-0010):** strip an optional `ISCC:` prefix → base32-decode the body chars to
  bytes → read the 16-bit header (validate it is an ISCC-IDv1: MainType = ID, Version = 1; the **SubType
  nibble is the realm**) → the remaining 8 bytes are the big-endian `uint64` body → `timestamp = body >>
  12`, `hub_id = uint16(body & 0xFFF)`. Return realm + hub_id + timestamp.
- **Fail closed, never panic** (learnings index — the `derive_vkey.py` short-key guard). Guard length
  BEFORE every slice/index: empty string, missing/garbage base32 chars, a header that is not an
  ISCC-IDv1 (wrong MainType/Version), and a body shorter than 8 bytes must each return a descriptive
  `error`, not an index-out-of-range panic. A "decode random/short junk never panics" sub-test is cheap
  insurance.
- **Keep it pure / WASM-shareable** (learnings: `proof/verify` purity rule by analogy). No
  `net`/`net/http`/`database/sql`/`os` imports. `fmt` (→ transitively `os`) and `encoding/base32`,
  `encoding/binary` are fine; verify with `GOOS=js GOARCH=wasm go build ./internal/index`.
- **Golden-vector test grounding (the load-bearing part).** The existing test ISCC-IDs (`ISCC:MAAGZTFQ…`,
  `ISCC:MEAJU5BQ…`, `ISCC:MAIG…`) are NOT guaranteed valid ISCC-IDv1s — do NOT assume they decode; some
  carry a different MainType. Seed the golden test with a vector whose realm/hub_id/timestamp are
  documented from first principles: construct the header+body bytes per the ADR/iscc-core layout in the
  test (or hard-code the canonical string AND its decoded fields), then assert `Decode` returns exactly
  those fields. If you add the unexported encode helper, prove the round-trip
  (`Decode(encode(realm,hubID,ts)) == {realm,hubID,ts}`) for at least one **test-realm (0)** vector AND
  one **operational-realm (1)** vector, with a **non-zero `hub_id`** (e.g. slot 1) exercised, so realm 0
  vs realm 1 and a real slot are both covered. Cite where each expected value comes from in a comment —
  never write the test to mirror the implementation's own arithmetic (the "tie a test to ground truth,
  not the symbol under test" lesson, `learnings/http-surface.md`).
- **Result shape.** Return a small named struct (`type ISCCID struct { Realm uint8; HubID uint16;
  Timestamp uint64 }`) with an `error`, not a 4-value tuple — easier for the later resolver to consume.
  Write an evergreen docstring on the package and the function.

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...`, `gofmt -l .` empty).
- `go test -count=1 ./internal/index` passes (the golden-vector + error-path table test).
- `go list -deps ./internal/index | grep -E '^net/http$|^database/sql$'` is empty (pure leaf, no HTTP/SQL
  in the closure).
- `GOOS=js GOARCH=wasm go build ./internal/index` succeeds (WASM-shareable).
- For the golden vector, `Decode` returns the realm, `hub_id`, and timestamp documented in the test
  comment; for each error case (empty, non-base32, wrong MainType/Version, truncated body) `Decode`
  returns a non-nil error and does NOT panic.

## Done When
`internal/index` exists with a pure, golden-tested `Decode` that yields the correct `(Realm, HubID,
Timestamp)` for a documented ISCC-IDv1 vector and fails closed (error, no panic) on malformed input,
with `mise run check` green and the package WASM-buildable.

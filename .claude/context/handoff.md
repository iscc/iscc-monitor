## 2026-06-21 — Harden the two Hub-List resolver fail-opens (path-bearing url + missing hub_id)

**Done:** Closed the two `normal` fail-open issues filed by `review` against the Hub-List resolver in
`internal/registry`. `hubDomain` now rejects a scheme'd path/query/fragment-bearing url (previously
`https://host/log` returned `host`, silently dropping `/log`), and `Hub.HubID` became a `*uint16` so an
absent `hub_id` is rejected instead of decoding to the YAML zero-value slot 0. Both guards are
mutation-proven non-vacuous.

**Files changed:**
- `internal/registry/registry.go`: (1) `hubDomain` adds a `u.Path != "" || u.RawQuery != "" ||
  u.Fragment != ""` reject after the host check, with a "not a bare host base url" wrapped error;
  docstring updated. (2) `Hub.HubID` changed `uint16` → `*uint16` (`yaml:"hub_id"`) for presence
  tracking; `ParseHubList` rejects `h.HubID == nil` ("hub_id is required") and carries the dereferenced
  `hubID` into the over-range/duplicate/`hubDomain` checks; `Resolve` skips nil-HubID entries and
  dereferences. `Resolve`'s exported signature `(uint16) (string, bool)` is unchanged.
- `internal/registry/hublist_test.go`: `TestParseHubListIgnoresPubkey` now compares the dereferenced
  `*uint16` slot + fields (struct literal `Hub{HubID:0,...}` no longer compiles). `TestParseHubListErrors`
  gains a **scheme'd** path-bearing case (`https://sb0.iscc.id/log` → "not a bare host base url") and a
  missing-`hub_id` case (`hub_id is required`); the existing scheme-less case kept + renamed for
  branch coverage. File-level + case comments document why each new case is non-vacuous.

**Verification:** `mise run check` → green (build + vet + all 20 packages pass; `gofmt -l .` empty).
- `go test -count=1 ./internal/registry` passes uncached.
- Scheme'd path-bearing url fails closed: non-nil error + nil `*HubList` (new test case, asserts on
  hard-coded fragment "not a bare host base url").
- Missing-`hub_id` entry fails closed: non-nil error + nil `*HubList` (asserts "hub_id is required");
  no silent slot-0.
- Golden fixture unchanged: `Resolve(0) == ("sb0.iscc.id", true)`, `Resolve(1) == ("sb1.amlet.id",
  true)` still hold (`testnet.yaml` urls have empty paths). Inactive/4095/unknown-slot tests still pass.
- `GOOS=js GOARCH=wasm go build ./internal/registry` succeeds (resolver stays WASM-shareable; no new
  imports — `net/url`+`fmt` already in closure; `go.mod`/`go.sum` byte-unchanged).
- **Non-vacuity proven:** reverting the `u.Path/RawQuery/Fragment` check → scheme'd-path test FAILs
  (parser returns the hub, host stripped). Reverting the nil-HubID reject (nil→slot 0) →
  missing-`hub_id` test FAILs. Restored clean; re-run green.

**Next:** With this resolver leaf hardened, the two viable threads from the prior review still stand:
(b) the ADR-0011 Go 1.26 toolchain bump + iscc-lib v0.5.0 adoption — still blocked locally (toolchain
is go1.24.13, confirmed via `go version`), needs an iteration where mise can provision Go 1.26; and the
deferred `/inclusion/{iscc_id}` HTML certificate page + proof-bundle assembler, which now safely
consumes `Resolve` (`decode realm + 12-bit hub_id → resolve issuing hub`) and re-engages the oracle
conformance gate. The certificate page is the next feature slice that closes an M-UI Verify criterion;
(b) is foundational but toolchain-gated.

**Notes:**
- `Hub.HubID` is now `*uint16` — a public-shape change to the `Hub` struct, but `Resolve` is the only
  reader and it is in-package (no external caller; the registry→`HubTarget` wiring is still deferred),
  so the blast radius is the one test struct-literal I updated. Chose Option A from next.md (minimal,
  YAML-idiomatic, WASM-pure) over a custom `UnmarshalYAML`.
- `Resolve` defensively skips a nil `HubID` even though `ParseHubList` guarantees a parsed `*HubList`
  never carries one — keeps the method fail-closed for any hand-built list (documented in the Hub
  docstring).
- Trailing-slash edge handled as next.md recommended: `https://host/` has `u.Path == "/"` and is now
  rejected (a bare host base url carries no path); the fixture urls (`https://sb0.iscc.id`) have empty
  paths and stay green.
- Oracle/conformance gate correctly N/A: pure registry leaf, no signature-verify / RFC-6962 / Merkle /
  proof code touched; `go.mod`/`go.sum` byte-identical (no new deps). The gate re-engages at the
  proof-bundle assembler / certificate page.
- ADR-0011 issue left untouched in `issues.md` (toolchain-gated, out of scope). The two fail-open
  issues are now resolved and ready for `review` to delete after confirming.

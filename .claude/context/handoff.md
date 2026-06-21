## 2026-06-21 — Reject reserved/empty realm domains before mounting the bare-domain dossier

**Done:** Added a `reservedDomain` predicate (empty/whitespace or a reserved mount name —
`metrics`, `healthz`, and the `web.Prefix` segment `_ds` derived from the const) and wired it into
`registerHubs` (fails loudly at startup, naming the bad domain) and `mirrorHandler` (skips the exact
`/<domain>` dossier mount for such a domain as defense-in-depth). This clears the open `normal` issue:
a realm line `metrics` no longer panics `buildMux` at startup with `pattern "/metrics" … conflicts`.

**Files changed:**
- `cmd/iscc-monitor/main.go`: added `reservedMountNames` map (`_ds` derived via
  `strings.Trim(web.Prefix, "/")`, not hardcoded) + `reservedDomain(domain)` predicate; `registerHubs`
  returns a named error for a reserved/empty domain before `UpsertHub`; `mirrorHandler` guards the
  exact dossier mount with `if !reservedDomain(r.Domain)`; added `"strings"` import; doc comments
  updated on both functions.
- `cmd/iscc-monitor/main_test.go`: added `TestRegisterHubsRejectsReserved` (table-driven over
  `metrics`/`healthz`/`_ds`/empty/whitespace, asserts non-nil error) and
  `TestBuildMuxReservedDomainNoPanic` (`recover`-guarded; drives `buildMux` with
  `hubRoute{Domain:"metrics", Origin:"metrics/log"}`, asserts no panic); added `"strings"` + `web`
  imports.

**Verification:** `mise run check` → green (all 19 packages `ok`; `go build`/`go vet`/`go test` pass).
`gofmt -l .` empty. Per-criterion:
- [x] `go test -run TestRegisterHubs ./cmd/iscc-monitor` passes incl. new reserved/empty cases.
- [x] `TestBuildMuxReservedDomainNoPanic` passes (constructs the `metrics`/`metrics/log` route,
  asserts `buildMux` does not panic).
- [x] `TestMirrorRouter` still passes (legit `sb0.iscc.id`/`sb1.amlet.id` routing unchanged).
- [x] Reserved-set derives `_ds` from `web.Prefix` (no hardcoded `"_ds"` literal in the guard).
- [x] Mutation-proven non-vacuous: removing the `mirrorHandler` skip → `TestBuildMuxReservedDomainNoPanic`
  panics with the exact `pattern "/metrics" … conflicts`; removing the `registerHubs` guard →
  `TestRegisterHubsRejectsReserved/{metrics,healthz,web_prefix_segment}` FAIL. Both reverted byte-identical.
- [x] go.mod/go.sum/schema byte-identical (not in diff). Oracle/conformance gate N/A — pure HTTP-wiring
  config validation; touches no signature, RFC-6962, Merkle, did:web, fsck, or proof path.

**Next:** Resume the planned M-UI dossier surface — the frozen **Exhibit** sub-step. It needs a NEW
`store.ListViolations(hubID)` read over the `violations` table (only `RecordViolation` exists today)
plus the non-dismissable Exhibit markup (ADR-0006). After that: the remaining SSR screens (paginated
record list, single-record page, certificate — the certificate re-engages the oracle/inclusion-proof gate).

**Notes:**
- **Design note (deviation from next.md's strict letter, not a behavior surprise):** next.md Scope said
  "add a guard in `registerHubs`" and Not-In-Scope forbade *reordering mounts* as the fix. I also added
  a conditional skip in `mirrorHandler`. This was REQUIRED by the explicit verification criterion
  `TestBuildMuxReservedDomainNoPanic`, which drives `buildMux` directly (bypassing `registerHubs`) and
  asserts no panic — a guard solely in `registerHubs` cannot satisfy it. The `mirrorHandler` change is a
  conditional skip, NOT a mount reorder (the forbidden alternative), so it honors the Not-In-Scope
  constraint while making `buildMux` total over any route slice. `registerHubs` remains the loud
  fail-at-startup path; `mirrorHandler` is defense-in-depth (its branch is never taken in production
  because `registerHubs` already errored out).
- Used one combined error message ("domain is empty or a reserved mount name") via the shared
  `reservedDomain` predicate rather than two separate messages, to keep the check DRY. The offending
  domain is still named (`%q`), matching the existing `register hub %q: …` style; an empty domain
  renders as `register hub "": …` and whitespace as `register hub "   ": …`.
- The mirror subtree mount `"/"+Origin+"/"` (e.g. `/metrics/log/`) is left in place for a reserved
  domain — it is a subtree, disjoint from the built-in exact `/metrics`, so it never collides (per
  next.md's origin caveat). Only the exact dossier mount is conditional.
- Pre-existing `low` issues untouched (3x `overlayStatus` duplication, notecheck `out` param, Mirror
  seam, proofserve `writeReadError`) — out of scope per next.md.

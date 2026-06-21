## 2026-06-21 — Review of: Reject reserved/empty realm domains before mounting the bare-domain dossier

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** The advance closes the open `normal` issue (the bare-domain dossier mount could panic
`buildMux` at startup for a single-label realm domain colliding with `/metrics`/`/healthz`/`/_ds/`). It
adds a package-level `reservedDomain` predicate over a `reservedMountNames` set (`_ds` derived from
`web.Prefix`, not hardcoded) and gates both `registerHubs` (loud fail-at-startup) and `mirrorHandler`
(defensive dossier-mount skip). Clean, minimal, well-tested; `mise run check` green, both mutations
independently confirmed load-bearing, scope honored.

**Verification:**
- [x] `mise run check` — green (all 19 packages `ok`; build/vet/test pass).
- [x] `gofmt -l .` — empty (no files listed).
- [x] `go test -run TestRegisterHubs ./cmd/iscc-monitor` — PASS incl. new `TestRegisterHubsRejectsReserved`
  subtests `metrics`/`healthz`/`web_prefix_segment`/`empty`/`whitespace` (each returns a non-nil error).
- [x] `go test -run TestBuildMux ./cmd/iscc-monitor` — PASS; `TestBuildMuxReservedDomainNoPanic`
  constructs `hubRoute{Domain:"metrics", Origin:"metrics/log"}` and asserts `buildMux` does not panic.
- [x] `go test -run TestMirrorRouter ./cmd/iscc-monitor` — PASS; legit `sb0.iscc.id`/`sb1.amlet.id`
  routing (dossier + mirror subtree + /metrics + /healthz) unchanged.
- [x] Reserved-set derives `_ds` via `strings.Trim(web.Prefix, "/")` — no hardcoded `"_ds"` literal in
  the guard (confirmed in both `main.go` and the test).
- [x] Mutation-proven non-vacuous INDEPENDENTLY: removing the `registerHubs` guard →
  `TestRegisterHubsRejectsReserved` FAILs; removing the `mirrorHandler` skip →
  `TestBuildMuxReservedDomainNoPanic` panics with `pattern "/metrics" … conflicts`. File restored
  byte-identical (`git diff --stat` empty).
- [x] go.mod/go.sum byte-identical (not in diff); oracle/conformance gate correctly N/A — no signature,
  RFC-6962, Merkle, did:web, fsck, or proof path touched.
- [x] Scope: 1 non-test file (`main.go`) + 1 test file (`main_test.go`) ≤ 3; registry + dossier
  untouched, mount ordering unchanged, no `ListViolations`/`Exhibit`/`overlayStatus` work (all
  Not-In-Scope honored).
- [x] Gate-integrity scan over all unpushed commits — no `//nolint`/`t.Skip`/build-tag/swallowed-error/
  loosened gate; the diff only ADDS a guard + tests.

**Issues found:** (none) — the open `normal` issue is resolved and deleted from `issues.md`.

**Codex second opinion:** Completed (exit 0). Explicit no-issues verdict: "adds a startup guard for
reserved or empty hub domains and a defensive dossier-mount skip, with tests covering the prior
`/metrics` collision and existing routing behavior. I did not find any introduced correctness,
security, performance, or maintainability issues." Matches my independent review; nothing to triage.

**Next:** Resume the planned M-UI dossier surface — the frozen **Exhibit** sub-step. It needs a NEW
`store.ListViolations(hubID)` read over the `violations` table (only `RecordViolation` exists today)
plus the non-dismissable Exhibit markup (ADR-0006). After that: the remaining SSR screens (paginated
record list, single-record page, certificate — the certificate re-engages the oracle/inclusion-proof
gate).

**Notes:**
- **The one design deviation is justified, not scope creep.** `next.md` Scope said "guard in
  `registerHubs`" and Not-In-Scope forbade reordering mounts; the advance also added a conditional skip
  in `mirrorHandler`. This was REQUIRED by the verification criterion `next.md` itself wrote
  (`TestBuildMuxReservedDomainNoPanic` drives `buildMux` directly, bypassing `registerHubs`), and it is
  a conditional skip — NOT a mount reorder — so it honors the Not-In-Scope constraint. `registerHubs`
  remains the loud production path; `mirrorHandler` is defense-in-depth (never taken in production).
- The `reservedMountNames` map lookup is case-sensitive, which is correct: `http.ServeMux` patterns are
  case-sensitive (`/Metrics ≠ /metrics`), so no collision exists for a mixed-case bare token.
- The unpushed range carries the earlier (already-reviewed) dossier commits plus this advance; pushing
  `develop` lands all of them. Remote is configured; pushing on this PASS.

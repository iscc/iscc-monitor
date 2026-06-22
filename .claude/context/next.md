# Next Work Package

## Step: Render certificate timestamps in UTC (`.UTC().Format`) to green the gate on non-UTC hosts

## Advances
Preempts milestone work to restore the **quality gate** the rest of M-Deploy verifies against.
Closes the `normal` issue *"Certificate renders coverage + §5-confirmation timestamps in LOCAL time,
not UTC — TZ-dependent test failures (red on any non-UTC machine)"* (issues.md). Per target.md's
non-negotiable Quality bar — *"`mise run check` is green: … `go test ./...` … all pass"* and the
cross-platform requirement — the gate is currently **RED on any non-UTC host** (reproduced this
iteration: `TZ=America/New_York go test ./internal/certificate` FAILS; `TZ=UTC` passes). This preempts
the M-Deploy critical work (SIGTERM trap, Dockerfile/GHCR, version stamp) because **every one of those
steps' CI jobs and local verifications run `mise run check`** — a gate that is environment-dependently
red cannot cleanly verify the next increment. The state's "Next Milestone" item #1 and the `review`
handoff both name this as the slice to land first.

## Goal
Normalize the four `.Format(time.RFC3339)` call sites in `internal/certificate/handler.go` to
`.UTC().Format(time.RFC3339)` so every rendered timestamp is locale-independent `…Z`, matching the rest
of the federation (dashboard/dossier/log-browser all render UTC). This makes `mise run check` green
regardless of the host timezone — the precondition for cleanly verifying the queued M-Deploy work.

## Scope
- **Create**: (none)
- **Modify**: `internal/certificate/handler.go` (the one production file; 4 `.Format(time.RFC3339)`
  sites — lines ~653, ~962, ~1013, ~1040).
- **Modify (test)**: `internal/certificate/handler_test.go` — add ONE focused regression test that
  pins UTC rendering independent of host TZ (see Implementation Notes); does not count against the
  ≤3 non-test/doc budget.
- **Reference**:
  - `.claude/context/learnings/certificate.md` (cert handler mechanics: §5 `BTCConfirmedAt`,
    Comparison-Anchor `CoverageSince`, §4 `SigningKeyRevoked`, the `html/template` base64-escaping note
    — read before touching).
  - `.claude/context/issues.md` → "Certificate renders coverage + §5-confirmation timestamps in LOCAL
    time, not UTC" (the exact failing tests + expected `Z` strings + fix directive).
  - `internal/certificate/handler_test.go:1458` (`TestCertificateComparisonAnchor`) and `:1656`
    (`TestCertificateBitcoinAnchorConfirmed`) — the two existing tests this greens.

## Not In Scope
- **Do NOT trap SIGTERM, add a Dockerfile/GHCR workflow, version-stamp the binary, add a `deploy/`
  realm doc, write the operability doc, or the root README.** Those are the queued M-Deploy steps and
  each is its own ≤3-file increment; this step only restores the green gate they depend on. Touching
  `cmd/iscc-monitor/main.go` here is out of scope.
- Do NOT change the timestamp FORMAT (no humanizing `2026-02-14 18:40 UTC` — the cert §6 mockup-humanize
  note in certificate.md is a separate deliberate format-policy step). Keep RFC-3339; only fix the zone.
- Do NOT touch the other SSR surfaces' timestamp rendering (dashboard/dossier/log-browser already render
  UTC `Z` — verified). This is the cert handler diverging, not a codebase-wide change.
- Do NOT add a `t.Setenv("TZ", …)`-based test that mutates process-global state — prefer a test that
  asserts the rendered chip is `…Z` so it holds on any host without per-test env juggling (see
  Implementation Notes).

## Implementation Notes
- The four sites, all in `internal/certificate/handler.go`, store UTC instants and currently render them
  in the server's LOCAL zone:
  - `:653` — `key.Revoked.Format(time.RFC3339)` (proof-bundle key `Revoked`; currently untested).
  - `:962` — `key.Revoked.Format(time.RFC3339)` (§4 `SigningKeyRevoked`; currently untested).
  - `:1013` — `rec.UpgradedAt.Format(time.RFC3339)` (§5 `BTCConfirmedAt`; pinned by
    `TestCertificateBitcoinAnchorConfirmed`).
  - `:1040` — `hub.Coverage.Since.Format(time.RFC3339)` (Comparison-Anchor `CoverageSince`; pinned by
    `TestCertificateComparisonAnchor`).
  Change each to `.UTC().Format(time.RFC3339)` — e.g. `rec.UpgradedAt.UTC().Format(time.RFC3339)`,
  `hub.Coverage.Since.UTC().Format(time.RFC3339)`, `key.Revoked.UTC().Format(time.RFC3339)` (twice).
  Fix ALL FOUR even though only two are test-covered, to satisfy the issue's grep-the-handler directive
  and keep the surface uniformly locale-independent (the two `Revoked` sites are the same hazard, just
  un-asserted today). Line numbers are approximate — grep `Format(time.RFC3339)` to locate exactly.
- **Correctness rule (always-loaded learnings, Coverage honesty / ADR-0001):** the certificate's
  RFC-3339 times are honesty-bearing UI; rendering them in UTC `Z` keeps every instance's output
  byte-identical regardless of host, which is also the cross-platform requirement (target.md Quality bar).
- **`html/template` escaping nuance (certificate.md):** these timestamp chips are plain text nodes, not
  base64 — no `+`/`/` entity-escaping applies, so a test may assert the raw `…Z` substring directly
  (unlike the §3/§5 base64 chips, which the existing tests `html.UnescapeString` first).
- **Test design — TZ-independent assertion:** add a `TestCertificate…UTC…` test that renders the §5
  confirmed / coverage chip on a fixture seeded with a NON-zero-offset wall instant and asserts the body
  contains the `Z`-suffixed UTC string (never a `+HH:MM` offset). The strongest non-vacuous form: seed a
  timestamp like `time.Date(2026, 2, 14, 18, 40, 0, 0, time.FixedZone("CET", 3600))` and assert the body
  renders `2026-02-14T17:40:00Z` (the UTC equivalent) AND does NOT contain `+01:00` — this fails before
  the `.UTC()` fix on EVERY host (the stored instant carries a non-UTC location), so the test is
  host-independent and proves the fix rather than merely tracking `mise run check`'s TZ-sensitivity.
  (The two existing tests already cover the UTC-stored path; the new test covers the non-UTC-stored path
  so reverting any `.UTC()` makes it FAIL regardless of `TZ`.)
- Oracle gate is **N/A**: pure timestamp-rendering change, no signature / RFC-6962 / Merkle / did:web /
  proof / `go.mod` / `go.sum` / `schema.sql` path touched (state this in the verdict).

## Verification
- `mise run check` is green **on a non-UTC host** — e.g. `TZ=America/New_York mise run check` passes
  (it FAILS on parent `HEAD` — reproduced this iteration).
- `TZ=America/New_York go test -count=1 ./internal/certificate` passes (parent: 2 failures).
- `TZ=UTC go test -count=1 ./internal/certificate` still passes (no regression on UTC hosts).
- The new test FAILS when any one `.UTC()` is reverted: with the fix in place,
  `TZ=UTC go test -count=1 -run TestCertificate ./internal/certificate` is green; manually reverting a
  single `.UTC()` insertion turns the new non-UTC-stored test red (mutation check).
- `gofmt -l internal/certificate/handler.go internal/certificate/handler_test.go` is empty and
  `go vet ./internal/certificate` is clean.

## Done When
`mise run check` is green on any host timezone (verified via `TZ=America/New_York`), the new TZ-independent
test passes and fails on a reverted `.UTC()`, and all four cert handler `.Format(time.RFC3339)` sites are
`.UTC()`-normalized — clearing the gate so the queued M-Deploy critical steps can be verified cleanly.

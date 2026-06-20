# Next Work Package

## Step: Pure realm-registry parser (domains-only membership document)

## Goal
Add `internal/registry` — a pure, dependency-free parser that turns a domains-only realm
membership document into a list of hub entries with their derived base URLs. This is the missing
prerequisite tissue between the per-network poll loop and a real `cmd/` binary: the `Loop` today
takes a hand-written `[]HubTarget`, and the registry is the spec'd source of those targets
(domains only, no keys — ADR-0009). Doing the pure parse first (before any fetch or `main` wiring)
keeps a testable unit ahead of infrastructure.

## Goal-fit (state → target gap)
M1's first Verify half (`origin`/`verifierKey`/single-poll) and two of three triggers (shrink+fork
freeze) are met end-to-end, and the poll loop now drives `PollHub` on a cadence. The named M1 list
still opens with "config + realm registry (domains only)" — and the registry is currently absent
(`internal/registry/` does not exist). Both review and state name the `cmd/iscc-monitor` binary as
the lowest-risk unblocked slice, but that binary needs *somewhere to get its hubs from*: the `Loop`
takes a fixed `[]HubTarget`, and the realm registry is the spec'd source. The registry's **parse**
half is a pure, golden-testable leaf with no I/O and no new dependency — exactly the "pure functions
before infrastructure, runnable+testable before wiring" ordering the loop prefers. It unblocks the
`cmd/` binary without coupling to it, and defers the riskier merkle-equivocation + `hub_keys`/
`derive_vkey.py` refresh (both trip the oracle gate) to their own later steps.

## Scope
- **Create**: `/workspace/iscc-monitor/internal/registry/registry.go` — the pure parser + entry type.
- **Create**: `/workspace/iscc-monitor/internal/registry/registry_test.go` — table-driven golden tests.
- **Create**: `/workspace/iscc-monitor/internal/registry/testdata/realm.txt` — a small golden
  membership fixture (the two live golden hubs + a comment + a blank line).
- **Modify**: (none — this is a new leaf package; touch nothing else)
- **Reference**:
  - `/workspace/iscc-monitor/.claude/plans/cosmic-baking-octopus.md` lines 96–118 (package layout:
    `internal/registry/` = "fetch+parse realm membership (domains only); reconcile add/remove/inactive")
    and the `hubs(... domain, origin, base_url, active, status ...)` schema block (lines 135–147).
  - `/workspace/iscc-monitor/.claude/adr/0009-didweb-trust-root.md` (realm registry advertises
    **domains/membership only — no keys**; domain ownership *is* identity).
  - `/workspace/iscc-monitor/.claude/prd/0001-iscc-monitor-v1.md` line 138 ("For pilot deployments
    the Hub-List is a static document").
  - `/workspace/iscc-monitor/internal/follower/loop.go` lines 26–34 (`HubTarget{HubID, BaseURL}` —
    the shape the registry feeds, indirectly, once wiring lands; the registry produces `BaseURL`).
  - `/workspace/iscc-monitor/internal/logclient/origin.go` (origin derivation — note it is
    package-private; the registry must NOT reach for it, see Implementation Notes).

## Not In Scope
- **No network fetch.** Do not add an HTTP fetch of a remote realm document, a `Fetcher` call, or
  any `net/http` import. This step parses bytes already in hand; the fetch seam is a later step.
- **No `cmd/iscc-monitor` binary and no `internal/config`.** Wiring DI / `Loop.Run` / flags / env /
  reading the file from disk is the *next* step and depends on this one — do not start it here.
- **No change to `internal/follower` or `HubTarget`.** Do not rewire `Loop` to consume registry
  output yet (that needs `HubID`s, which come from `store.UpsertHub` at wiring time).
- **No `origin()` derivation in this package.** The registry produces `BaseURL` only; the follower
  already derives origin + verifier key from `BaseURL` inside `PollHub`. Do not duplicate or export
  `logclient.origin`.
- **No YAML/JSON dependency.** Use the line-based format below; do not add `gopkg.in/yaml.v3` or any
  parser dep for a pilot static document (YAGNI; keep `go.mod`/`go.sum` byte-identical).
- **No `active`/`inactive`/`status` reconciliation, sorting, or dedupe.** The plan mentions reconcile
  add/remove/inactive, but that is store-coupled and belongs with the wiring step. Parse membership
  only, preserving input order.

## Implementation Notes
- **Format (KISS, pilot static document):** one hub domain per line; ignore blank lines and lines
  whose first non-whitespace character is `#` (comments); trim surrounding whitespace on each kept
  line. This is the simplest thing that satisfies "static document, domains only" without a new dep.
  Document the format in the package + function docstrings (evergreen wording, no "new"/"improved").
- **Entry type:** export a small struct, e.g. `type Entry struct { Domain string; BaseURL string }`.
  Derive `BaseURL` as `"https://" + Domain` (hubs are HTTPS; the spec origin example `sb0.iscc.id/log`
  is served over TLS). Keep `Domain` as the bare host (e.g. `sb0.iscc.id`) so a later step can pass it
  to `store.UpsertHub(domain, …)`.
- **Parse signature:** a pure func over bytes, e.g. `func Parse(data []byte) ([]Entry, error)`.
  Prefer `bufio.NewScanner` over a `bytes.NewReader` (stdlib only). Reject a domain containing a
  scheme (`://`), whitespace, or a path/slash (`/`) with a wrapped error naming the offending line —
  fail closed, do not silently coerce a URL into a domain. A `host:port` form is acceptable to allow
  (live hubs have none, but the colon path is already handled downstream by `didweb`); do not
  over-validate beyond "no scheme, no slash, non-empty after trim".
- **Determinism:** preserve input order in the returned slice (the poll loop iterates targets in
  order; stable order keeps tests + logs deterministic). Do not sort or dedupe in this step — dedupe
  is reconciliation, which is Not In Scope.
- **Purity:** this is a leaf, so keep it import-clean: `bufio`, `bytes`, `fmt`, `strings` only. No
  `net`, no `net/http`, no `os` (the *caller* reads the file and passes bytes — embedding/reading is
  the wiring step's job). Verify the package's own `.Imports` are exactly those stdlib packages.
- **Golden fixture (`testdata/realm.txt`):** the two live golden hubs plus at least one comment line
  and one blank line, e.g.:
  ```
  # iscc testnet realm — pilot membership (domains only, ADR-0009)
  sb0.iscc.id

  sb1.amlet.id
  ```
  In the test, read the fixture with `os.ReadFile` (the *test* may use `os`; the package must not) and
  assert `Parse(...)` yields exactly `[{sb0.iscc.id, https://sb0.iscc.id}, {sb1.amlet.id,
  https://sb1.amlet.id}]` (comment + blank dropped, order preserved). Add table-driven error cases (a
  line with `https://...`, a line with `sb0.iscc.id/log`) and an all-comment/all-blank document case
  (→ empty slice, nil err).
- **Relevant Correctness rule (learnings):** "did:web is the only key source (ADR-0009) — the realm
  registry advertises domains only." This parser must therefore carry **no key field** and reject
  anything URL-shaped. It is purely a domain list.

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...` all pass).
- `gofmt -l internal/registry` prints nothing.
- `go test -count=1 ./internal/registry` passes.
- `go test -count=1 -run TestParse ./internal/registry` passes; the golden case asserts
  `Parse(testdata/realm.txt)` returns exactly two entries in input order:
  `{Domain:"sb0.iscc.id", BaseURL:"https://sb0.iscc.id"}` then
  `{Domain:"sb1.amlet.id", BaseURL:"https://sb1.amlet.id"}`.
- A line containing `https://sb0.iscc.id` or `sb0.iscc.id/log` returns a non-nil error naming the
  bad line; an all-comment / all-blank document returns `(len 0, nil)`.
- `git status --short go.mod go.sum` is empty (no dependency added).
- `go list -deps ./internal/registry | grep -E '^(net|net/http)$'` prints nothing (leaf, no net stack).

## Done When
`internal/registry` exists as a pure leaf package whose `Parse` turns the golden domains-only
`realm.txt` into the two ordered `Entry{Domain, BaseURL}` values, rejects URL-shaped lines, and all
Verification criteria pass with `mise run check` green and no new dependency.

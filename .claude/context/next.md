# Next Work Package

## Step: Reject reserved/empty realm domains before mounting the bare-domain dossier

## Advances
Preempts the open **`normal` issue** "Bare-domain dossier mount collides with reserved exact routes
(`/metrics`, `/healthz`) → startup panic" (review-confirmed by Codex P2 reproduction). It is the
review handoff's explicit **Next:** ("Fix the reserved/empty-domain mount collision at the root
before adding more dossier surface") and step 1 of state's "Next Milestone". This `normal` defect is
review-blocking: it must clear before the **M-UI hub dossier** screen counts as met, so it preempts
the remaining M-UI screen work (frozen Exhibit, record list, certificate). target.md §Done When
requires "no open `critical` or `normal` issue in `issues.md`", so this issue also blocks DONE.

## Goal
Stop the monitor from panicking at startup when a realm-document line is a single-label token that
collides with a built-in exact mount (`metrics`, `healthz`, the `web.Prefix` segment `_ds`) or is
empty. Fail loudly at registration with a named error so the operator learns the config is invalid,
rather than crashing in `http.ServeMux.Handle` inside `buildMux`.

## Scope
- **Modify**: `/workspace/iscc-monitor/cmd/iscc-monitor/main.go` (add a reserved/empty-domain guard in
  `registerHubs`)
- **Modify (test — uncounted)**: `/workspace/iscc-monitor/cmd/iscc-monitor/main_test.go` (add a
  reserved-name + empty-domain case driving `registerHubs`, plus a `buildMux`-no-longer-panics
  assertion)
- **Reference**:
  - `/workspace/iscc-monitor/.claude/context/learnings/cmd-monitor.md` — the panic mechanism, mount
    ordering, and the bullet that already flagged this exact reserved-name class; the
    `registerHubs` returns index-aligned `([]HubTarget, []hubRoute, error)` contract.
  - `/workspace/iscc-monitor/cmd/iscc-monitor/main.go:169-208` — `buildMux` + `mirrorHandler`:
    `mirrorHandler` registers `"/"+r.Domain` (205) BEFORE `buildMux` registers `/metrics` (172),
    `/healthz` (173), and `web.Prefix` (174); the later built-in `mux.Handle("/metrics", …)` is what
    panics on the duplicate pattern.
  - `/workspace/iscc-monitor/cmd/iscc-monitor/main.go:266-282` — `registerHubs`, the testable choke
    point that owns the `hubRoute` carrying `Domain`.
  - `/workspace/iscc-monitor/internal/web/web.go:46-49` — `Prefix = "/_ds/"`, the source of the third
    reserved name (segment `_ds`).
  - `/workspace/iscc-monitor/internal/registry/registry.go:49-69` — `Parse` only rejects `://` and
    `/`, so reserved bare tokens pass through as valid `Domain`s.
  - `/workspace/iscc-monitor/cmd/iscc-monitor/main_test.go:32-105` — `TestRegisterHubs` (the table to
    extend) and `:113-133` — `TestMirrorRouter` (the `buildMux` invocation shape to copy).

## Not In Scope
- Do NOT add the reserved-name guard to `internal/registry.Parse`. The reserved set
  (`metrics`, `healthz`, `_ds`) is a property of the binary's HTTP mount layer, not of the realm
  document format; the registry is a pure domain-list leaf (ADR-0009) and must not learn HTTP mount
  names. Keep the guard in `cmd/iscc-monitor` where the mounts live.
- Do NOT change the mount ordering in `buildMux`/`mirrorHandler` as the fix. Reordering to register
  built-ins first would only turn the panic into a different collision (the dossier would then panic
  on `/metrics`); rejecting the bad domain at the root is the correct fix.
- Do NOT build the frozen **Exhibit**, `store.ListViolations`, the paginated record list, the
  single-record page, or the certificate — those are later M-UI steps that resume after the dossier
  is met.
- Do NOT touch the dossier handler (`internal/dossier`) — the page passed its own Verify criteria;
  only the wiring is broken.
- Do NOT consolidate the 3x `overlayStatus` duplication or any other `low` issue.

## Implementation Notes
- **Where:** add the guard inside the `registerHubs` loop in `main.go` (266-282), before the
  `UpsertHub` call. `registerHubs` already returns an `error` and is the choke point that builds the
  `hubRoute{Domain}` the dossier mounts on — it is the right root because both the offending mount
  (`"/"+r.Domain`) and the collision derive from it. Failing here short-circuits with the offending
  domain named, matching the existing `fmt.Errorf("register hub %q: %w", e.Domain, err)` style.
- **Reserved set:** define a small package-level slice/set of reserved mount segments —
  `"metrics"`, `"healthz"`, and `strings.Trim(web.Prefix, "/")` (which evaluates to `_ds`; derive it
  from the const, do NOT hardcode `"_ds"`, so it tracks `web.Prefix` if that ever changes). Add a
  `"strings"` import if not already present.
- **Empty case:** reject `strings.TrimSpace(e.Domain) == ""` too — an empty Domain mounts the exact
  `/`, colliding with the dashboard's `/` mount; the handoff and learnings flag empty and reserved as
  the same class.
- **Error, not skip:** prefer failing loudly —
  `return nil, nil, fmt.Errorf("register hub %q: domain is a reserved mount name", e.Domain)` — over a
  silent skip, so a misconfigured realm surfaces at startup rather than silently dropping a hub.
  State's Next-Milestone step 1: "prefer failing `registerHubs`/`registry.Parse` loudly over a silent
  skip".
- **Origin caveat:** the origin is `<domain>/log`, so a reserved bare domain `metrics` mounts the
  mirror subtree at `/metrics/log/` (a *subtree*, no collision) but the dossier at `/metrics` (exact,
  collides). Rejecting the domain covers both mounts uniformly — no need to special-case the mirror.
- **Learnings rule (cmd-monitor.md):** the existing bullet documents this exact failure
  (`pattern "/metrics" … conflicts`) and prescribes "reject/skip reserved + empty domains before
  mounting … and a test must drive a reserved name through `buildMux`". The load-bearing invariant is
  the single-listener / single-mux model — keep one listener, one mux; the fix lives purely at
  registration.
- **Oracle/conformance gate is N/A:** this is pure HTTP-wiring config validation; it touches no
  signature, RFC-6962, Merkle, did:web, fsck, or proof path. go.mod/go.sum/schema must stay
  byte-identical. State this in the advance.
- **Test seam:** extend `TestRegisterHubs` (or add a sibling `TestRegisterHubsRejectsReserved`) to
  drive `registerHubs(ctx, st, []registry.Entry{{Domain:"metrics", BaseURL:"https://metrics"}})` and
  assert a non-nil error — table-driven over `metrics`, `healthz`, `_ds`, and an empty/whitespace
  domain. For the panic-no-longer regression, copy `TestMirrorRouter`'s `buildMux(st, routes,
  metrics.New())` shape with `routes := []hubRoute{{HubID:1, Domain:"metrics", Origin:"metrics/log"}}`
  and assert it does NOT panic — a bare call fails the test on panic, or use a `recover`-guarded
  helper. The `issues.md` repro uses exactly that `hubRoute` shape.

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...` all pass, `gofmt -l .`
  empty).
- `go test -run TestRegisterHubs ./cmd/iscc-monitor` passes, including the new reserved/empty cases:
  `registerHubs` returns a non-nil error for a `Domain` of `metrics`, `healthz`, `_ds`, and `""`/`" "`.
- A new test (e.g. `TestBuildMuxReservedDomainNoPanic`) constructs
  `hubRoute{Domain:"metrics", Origin:"metrics/log"}` and asserts `buildMux` does NOT panic —
  `go test -run TestBuildMux ./cmd/iscc-monitor` passes (this would have FAILED with a
  `pattern "/metrics" … conflicts` panic before the fix; mutation-prove by temporarily removing the
  guard → the test panics/fails, then restore).
- `go test -run TestMirrorRouter ./cmd/iscc-monitor` still passes (the legitimate
  `sb0.iscc.id`/`sb1.amlet.id` routing is unchanged).
- The reserved-set check derives `_ds` from `web.Prefix` (no hardcoded `"_ds"` literal in the guard).

## Done When
`registerHubs` rejects a reserved (`metrics`/`healthz`/`_ds`) or empty realm domain with a named
error, `buildMux` no longer panics for such a config, and all Verification checks pass — clearing the
open `normal` issue and unblocking the hub dossier as a met M-UI screen.

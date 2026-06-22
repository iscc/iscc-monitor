# Next Work Package

## Step: Reject a trailing `?` (ForceQuery) in the Hub-List `hubDomain` bare-host guard

## Advances
Closes the open `normal` issue **"Hub-List `hubDomain` accepts a trailing `?` (ForceQuery fail-open
against the bare-host contract)"** (issues.md). The handoff's `**Next:**` from the latest `review`
names this as item (1) in handoff-named order: *"the `hubDomain` ForceQuery fail-open in
`internal/registry/registry.go` (add `|| u.ForceQuery` to the line-188 reject)"*. It preempts further
milestone work because every milestone Verify criterion still open is non-code-closable right now: the
WASM "published" half is **human-blocked** (Pages repo-Settings step a workflow file cannot
self-enable), the WASM signature half is a flagged **design-first STOP-candidate**, and the OTS
Bitcoin-confirmed half is **offline-unprovable**. `internal/registry` is the resolver the certificate
page consumes (decode → `Resolve(hub_id)` → domain), so a trust-root-adjacent resolver should fully
enforce its stated "bare host base url" contract before that path widens. DONE requires 0 `normal`
issues, so closing this moves the gate.

## Goal
Make `hubDomain` reject a url whose only non-host component is a bare trailing `?` (e.g.
`https://sb0.iscc.id?`), which `net/url` represents as `ForceQuery == true` with `RawQuery == ""` — a
fail-open the current `Path/RawQuery/Fragment` guard misses. This fully enforces the resolver's
documented "no path/query/fragment" contract so a query delimiter can never survive into a resolved
hub domain.

## Scope
- **Create**: (none)
- **Modify**: `internal/registry/registry.go` — the single line-188 reject inside `hubDomain` (add
  `|| u.ForceQuery`), plus the adjacent docstring/comment to name `ForceQuery` so the evergreen comment
  matches the code. (1 non-test/doc file.)
- **Modify (test)**: `internal/registry/hublist_test.go` — add one case to the existing
  `TestParseHubListErrors` table (around `hublist_test.go:184`, beside the path-bearing-url cases).
- **Reference**:
  - `.claude/context/learnings/registry.md` — the `url.Parse` fail-open trap bullet ("`ForceQuery`:
    `u.RawQuery != ""` does NOT catch `https://host?` …") and the general rule: *enumerate `url.URL`'s
    shape-carrying fields (`Path RawQuery ForceQuery Fragment Opaque User`), not just the obvious three.*
  - issues.md entry "Hub-List `hubDomain` accepts a trailing `?` …" — exact location + verify recipe.
  - `internal/registry/registry.go:177-192` (the `hubDomain` body) and `:120-145` (`ParseHubList`, the
    caller the test drives).

## Not In Scope
- Do **not** also reject `u.Opaque` / `u.User` / the `#`-only `Fragment` form in this step — the issue
  and learnings establish that `https://host#` is dropped by Go on round-trip (harmless) and `Opaque`
  is unreachable for a `https://`-scheme'd host; only `ForceQuery` is the live load-bearing fail-open.
  (A broader field-by-field audit, if ever wanted, is a separate step — KISS: fix the one proven hole
  the handoff named.)
- Do **not** touch the §5 OTS-digest-binding `normal` or the certificate tier-2 honesty-copy `normal`
  (the other handoff-named items) — each is its own later step.
- Do **not** wire `ParseHubList`/`Resolve` into the live realm-loading path (still deferred) or change
  the `Resolve` / `Hub` / `HubList` public shape.
- Do **not** add any `low`-priority hardening (the loop skips `low`).

## Implementation Notes
- The current guard (`registry.go:188`) is
  `if u.Path != "" || u.RawQuery != "" || u.Fragment != "" {` — add `|| u.ForceQuery` to the condition.
  Confirmed against `net/url`: `url.Parse("https://sb0.iscc.id?")` yields `Host="sb0.iscc.id"`,
  `Path=""`, `RawQuery=""`, `ForceQuery=true`, and `u.String()` round-trips the `?` (so without the fix
  the delimiter survives into the resolved domain via `Hub.URL`). `https://sb0.iscc.id#` and
  `https://sb0.iscc.id` both have `ForceQuery=false`, so adding the clause does **not** regress the
  clean-domain or trailing-`#` paths.
- Fail closed, mirroring `Parse` and the two fail-opens already hardened (the `hubDomain`
  path-bearing-url reject and the required `hub_id`): **reuse the existing** "is not a bare host base
  url (path/query/fragment not allowed)" wrapped error so `ParseHubList` reports
  `registry: hub_id N: url … is not a bare host base url …`. Do not invent a new message.
- Make the comments evergreen: the `hubDomain` docstring (`registry.go:171-176`) and the in-line guard
  comment should mention that a trailing `?` (ForceQuery) is rejected too, not just path/query/fragment
  — so the comment describes the current behavior (CLAUDE.md "evergreen comments").
- **Test (non-vacuous, mutation-provable):** add a case to `TestParseHubListErrors` with
  `name: "trailing question mark (ForceQuery)"`, a one-hub document whose `url: https://sb0.iscc.id?`,
  and `errFrag: "not a bare host base url"` (match what the existing path-bearing cases assert). The
  table drives `ParseHubList`, which calls `hubDomain`, so the case exercises the real reject. Confirm
  non-vacuous: with `|| u.ForceQuery` removed, `ParseHubList` returns `(non-nil *HubList, nil err)` and
  the new case FAILS.
- Relevant learnings rule: `internal/registry` is a **pure leaf** — imports stay exactly
  `{bufio bytes fmt net/url strings yaml.v3}`; do not add a new import (`ForceQuery` is a field on the
  already-parsed `*url.URL`). The oracle/conformance gate is **N/A** (no proof/verify/didweb/merkle
  signature path touched); `go.mod`/`go.sum` stay byte-identical.

## Verification
- `mise run check` is green (build + vet + `go test ./...` all pass; `gofmt -l .` empty).
- `mise exec -- go test -count=1 -run TestParseHubListErrors ./internal/registry` passes, and its
  verbose output (`-v`) lists the new `trailing_question_mark_ForceQuery` subtest.
- Mutation check: removing `|| u.ForceQuery` from the `hubDomain` reject makes
  `TestParseHubListErrors/trailing_question_mark_ForceQuery` FAIL; restoring it returns green.
- Assertion: `ParseHubList` over a one-hub document with `url: https://sb0.iscc.id?` returns a non-nil
  error (fragment `not a bare host base url`) and a nil `*HubList`; the clean `https://sb0.iscc.id`
  golden (`TestParseHubListGolden`) still parses unchanged.

## Done When
`mise run check` is green, `TestParseHubListErrors` covers the trailing-`?` ForceQuery case and is
mutation-proven (removing `|| u.ForceQuery` fails it), and `hubDomain` rejects `https://host?` while
clean `https://host` domains still resolve byte-identically.

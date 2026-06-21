# Next Work Package

## Step: Harden the two Hub-List resolver fail-opens (path-bearing url + missing hub_id)

## Advances
This step closes two `normal` issues filed by `review` against the just-landed Hub-List resolver — it
does not directly close a milestone Verify criterion but **preempts certificate-page work** by hardening
a trust-root-adjacent leaf before it is wired in. The criterion it protects is the M-UI certificate
clause:

> the **realm-wide certificate** (`/inclusion/{iscc_id}`, keyed by the self-describing ISCC-IDv1 —
> decode realm + 12-bit `hub_id`, resolve the issuing hub via the registry) for a known id renders the
> numbered evidence clauses…

A wrong `(realm, hub_id) → domain` resolution would prove the wrong leaf (per `learnings/index.md`:
"a wrong domain proves the wrong leaf"). Per `state.md` Next-Milestone ordering and the review handoff
`**Next:** (a)`, the two fail-opens are the cheapest, most self-contained slice and should land
**before** the resolver is consumed. The third `normal` issue (ADR-0011 Go 1.26 bump) is deferred this
iteration because the local toolchain is go1.24.13 — confirmed via `go version` — so flipping `go.mod`
would red the whole gate; it must run where mise can provision Go 1.26.

## Goal
Make `internal/registry`'s Hub-List resolver fail **closed** on a scheme'd path-bearing url and on an
entry that omits `hub_id`, so the resolver's claimed contract (reject anything that could silently
coerce the wrong domain or the wrong slot) actually holds before the certificate page wires it in.

## Scope
- **Create**: (none)
- **Modify**:
  - `internal/registry/registry.go` (the only non-test source file — 1 of ≤3)
  - `internal/registry/hublist_test.go` (test, not counted)
- **Reference**:
  - `/workspace/iscc-monitor/.claude/context/learnings/registry.md` — the `hubDomain` fail-open + the
    YAML-zero-value (`missing hub_id → slot 0`) notes, and the `KnownFields(false)` parse-and-ignore
    contract to preserve.
  - `/workspace/iscc-monitor/.claude/context/learnings.md` index — the WASM-purity nuance ("`os`
    appears only transitively via `fmt`; prove WASM with `GOOS=js GOARCH=wasm go build`, not by
    grepping `os`").
  - `/workspace/iscc-monitor/.claude/context/issues.md` — the two `normal` issues being closed (exact
    repro lines: `hubDomain` registry.go:161-173; `Hub.HubID` registry.go:86).
  - `/workspace/iscc-monitor/internal/registry/registry.go` — `hubDomain` (lines 161-173), `Hub` struct
    (lines 85-89), `ParseHubList` (lines 114-135): the exact sites to edit.
  - `/workspace/iscc-monitor/internal/registry/testdata/testnet.yaml` — the golden fixture that must
    still parse + resolve unchanged (its urls have empty paths).

## Not In Scope
- The ADR-0011 Go 1.26 toolchain bump + `iscc-lib` v0.5.0 adoption (third `normal` issue) — local
  toolchain is go1.24.13; defer to an iteration that can provision Go 1.26.
- Wiring `Resolve`/`ParseHubList` into config, dashboard, or the certificate page — live wiring is the
  certificate-page slice, not this one.
- The certificate-of-inclusion HTML page or the proof-bundle assembler.
- Flipping `KnownFields(false)` → `true` (the deliberate parse-and-ignore `pubkey`/future-field
  contract — see `learnings/registry.md`); keep tolerating unknown keys.
- Any change to the domains-only `Parse`/`Entry`/`realm.txt` path (untouched, as in the prior slice).

## Implementation Notes
Two fail-opens, both in `internal/registry/registry.go`:

1. **Path-bearing url (issues.md "Hub-List `hubDomain` accepts a scheme'd path-bearing URL").** In
   `hubDomain`, after the `u.Host == ""` check at line 169, reject a url that carries a non-empty
   `u.Path`, `u.RawQuery`, or `u.Fragment` with a wrapped error (e.g. `"url %q is not a bare host base
   url (path/query/fragment not allowed)"`). Today `hubDomain("https://sb0.iscc.id/log")` returns
   `("sb0.iscc.id", nil)` — confirmed by review and Codex. The existing `"path-bearing url"` test case
   uses the **scheme-less** `sb0.iscc.id/log`, caught by the *no-host* branch — false confidence — so the
   new test must use a **scheme'd** path url (`https://sb0.iscc.id/log`) to make the guard non-vacuous.
   - Root-path edge: `url.Parse("https://host")` → `u.Path == ""`, but `url.Parse("https://host/")` →
     `u.Path == "/"`. The fixture urls (`https://sb0.iscc.id`, `https://sb1.amlet.id`) have empty paths,
     so a strict `u.Path != ""` check keeps the golden green. Recommend rejecting a lone trailing `/`
     too (a bare host base url carries no path); whatever you choose, `testnet.yaml` must still parse +
     resolve.

2. **Missing `hub_id` → slot 0 (issues.md "Hub-List entry missing `hub_id` silently becomes slot 0").**
   `Hub.HubID uint16` cannot distinguish absent from `0`, so an entry with only `url`/`active` silently
   becomes slot 0 and `Resolve(0)` returns it. Add presence-tracking and reject an absent `hub_id` with
   a wrapped error. Prefer the smallest YAML-idiomatic, WASM-pure change:
   - **Option A (recommended, minimal):** change the field to `HubID *uint16 \`yaml:"hub_id"\`` and, in
     the `ParseHubList` loop, reject `h.HubID == nil` ("hub_id is required") before dereferencing; carry
     the dereferenced value into the over-range/duplicate checks and `Resolve`. This changes the public
     `Hub` struct shape — update `TestParseHubListIgnoresPubkey`'s struct-literal comparison
     (`Hub{HubID: 0, URL: ..., Active: true}` at hublist_test.go:110) accordingly.
   - **Option B (no public-shape change):** keep `HubID uint16` and add a custom `UnmarshalYAML` (or
     decode each entry via `yaml.Node`) that errors when the `hub_id` key is absent. Heavier; choose A
     unless keeping the value-type field matters (no external caller exists — `Resolve` is the only
     reader, in-package).
   - Either way, the over-range/duplicate guards and `Resolve`'s loop comparison must operate on the
     resolved `uint16` value, and `Resolve`'s signature (`Resolve(uint16) (string, bool)`) must NOT
     change — only the internal storage of the slot.

**Fail-closed contract** mirrors `Parse` and the existing `ParseHubList`: return a `nil` `*HubList`
alongside a wrapped, fault-naming error. Keep `learnings.md`'s WASM-purity rule: do **not** add
`net`/`net/http`/`database/sql`; `net/url` and `fmt` are already in the closure. Prove with
`GOOS=js GOARCH=wasm go build ./internal/registry`, not by grepping `os` out of the dep list.

Add two test cases to `TestParseHubListErrors` (hublist_test.go): a **scheme'd** path-bearing url
(`https://sb0.iscc.id/log` → error) and a `hubs:` entry omitting `hub_id` (→ error). Keep (or rename)
the existing scheme-less `sb0.iscc.id/log` case — still valid via the no-host branch; the scheme'd case
is the load-bearing addition. Use hard-coded expected error fragments so the gate ties to ground truth,
not to the symbol under test.

## Verification
- `mise run check` is green (build + vet + all packages + `gofmt -l .` empty).
- `go test -count=1 ./internal/registry` passes uncached.
- A scheme'd path-bearing url fails closed: `ParseHubList` over a doc whose hub `url` is
  `https://sb0.iscc.id/log` returns a non-nil error **and** a nil `*HubList`.
- A missing-`hub_id` entry fails closed: `ParseHubList` over a `hubs:` entry with only `url`/`active`
  returns a non-nil error **and** a nil `*HubList` (no silent slot-0).
- The golden fixture is unchanged behavior: `Resolve(0) == ("sb0.iscc.id", true)` and
  `Resolve(1) == ("sb1.amlet.id", true)` still hold (the `testnet.yaml` urls have empty paths).
- `GOOS=js GOARCH=wasm go build ./internal/registry` still succeeds (resolver stays WASM-shareable).
- **Non-vacuity (for `review` to confirm):** reverting the new `u.Path` check makes the scheme'd-path
  test FAIL; reverting the `hub_id`-presence check makes the missing-`hub_id` test FAIL.

## Done When
`mise run check` and `go test -count=1 ./internal/registry` are green, both the scheme'd path-bearing
url and the missing-`hub_id` entry fail closed (non-nil error + nil list), the golden fixture still
resolves hub 0/1 to their hard-coded domains, and `GOOS=js GOARCH=wasm go build ./internal/registry`
still succeeds.

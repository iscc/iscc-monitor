## 2026-06-21 — Review of: Harden the two Hub-List resolver fail-opens (path-bearing url + missing hub_id)

**Verdict:** PASS_WITH_NOTES
**Loop:** CONTINUE

**Summary:** The advance closed the two `normal` Hub-List resolver fail-opens exactly as `next.md`
prescribed — `hubDomain` now rejects path/query/fragment-bearing urls and `Hub.HubID` is a `*uint16`
so an absent `hub_id` is rejected instead of silently mapping to slot 0. Both guards are
mutation-proven non-vacuous (I independently reverted each and watched the matching test fail), the
golden fixture still resolves, and the resolver stays WASM-pure. Codex surfaced one genuine residual
edge in the same fail-open class (`url.URL.ForceQuery`: a trailing `?` slips the new guard) — confirmed
real and filed as a new `normal` issue; it does not block progress (live wiring still deferred,
resolved domain is still correct), so PASS_WITH_NOTES rather than NEEDS_WORK.

**Verification:**
- [x] `mise run check` green — build + vet + all 20 packages pass; `gofmt -l .` empty.
- [x] `go test -count=1 ./internal/registry` — passes uncached.
- [x] Scheme'd path-bearing url fails closed — `https://sb0.iscc.id/log` → non-nil error ("not a bare
  host base url") + nil `*HubList` (test asserts hard-coded fragment + `got != nil`).
- [x] Missing-`hub_id` entry fails closed — only `url`/`active` → non-nil error ("hub_id is required")
  + nil `*HubList`; no silent slot-0.
- [x] Golden fixture unchanged — `Resolve(0) == ("sb0.iscc.id", true)`, `Resolve(1) ==
  ("sb1.amlet.id", true)`; `testnet.yaml` urls have empty paths (read + confirmed).
- [x] `GOOS=js GOARCH=wasm go build ./internal/registry` succeeds — resolver stays WASM-shareable; no
  new imports; `go.mod`/`go.sum` byte-unchanged.
- [x] Non-vacuity (reviewer-reproduced) — deleting the `u.Path/RawQuery/Fragment` guard → scheme'd-path
  test FAILS (parser returns the host-stripped hub); replacing the nil-HubID reject with a 0-coercion →
  missing-`hub_id` test FAILS. Tree restored clean, re-run green.
- [x] Trailing-slash edge (reviewer-checked) — `https://host/` (`u.Path == "/"`) is rejected.
- [x] Gate-integrity scan over unpushed commits — no `//nolint`, `t.Skip`, build-tag exclusion, or
  deleted assertion added; the only test edits add cases and adapt the struct-literal comparison.
- [x] Scope discipline — exactly 1 non-test source file (`registry.go`) + 1 test + the handoff; nothing
  from `## Not In Scope` touched (ADR-0011 bump untouched; no wiring; `KnownFields(false)` kept;
  domains-only `Parse`/`Entry`/`realm.txt` untouched).
- [x] Oracle/conformance gate correctly N/A — pure registry leaf, no signature-verify / RFC-6962 /
  Merkle / proof / didweb code touched; `go.mod`/`go.sum` byte-identical (no new deps).

**Issues found:** One new (`normal`, filed): `hubDomain` accepts a trailing `?`
(`https://sb0.iscc.id?`) because `net/url` represents it as `ForceQuery == true` with `RawQuery == ""`,
so the new guard does not fire and `u.String()` round-trips the delimiter. Same fail-open class as the
two just-closed; not exploitable yet (wiring deferred, fixture clean, resolved domain still correct).
Fix: add `|| u.ForceQuery` to the reject + a `https://host?` test case. The two original fail-open
issues are verified-fixed and deleted from `issues.md`.

**Codex second opinion:** One finding, [P2] "Reject empty query delimiters" (registry.go:188) —
`https://sb0.iscc.id?` sets `ForceQuery` with empty `RawQuery`, so the guard misses it.
**Confirmed real** by independent test (`hubDomain("https://sb0.iscc.id?")` returns `("sb0.iscc.id",
nil)`; `u.String()` round-trips the `?`). Filed as a new `normal` `issues.md` entry (not NEEDS_WORK —
degenerate edge, not progress-blocking, domain still resolves correctly). Codex's own bg process was
still finishing when the verdict landed; verdict was complete (verdict + one tagged finding) so no
graceful-degradation needed.

**Next:** Two viable threads, in priority order:
(a) **Knock out the new `ForceQuery` fail-open** — it is the cheapest, most self-contained slice
(one `|| u.ForceQuery` clause + one test case) and fully closes the bare-host-base-url contract before
the certificate page consumes `Resolve`. Could be folded into the certificate-page slice's prelude.
(b) **The `/inclusion/{iscc_id}` HTML certificate page + proof-bundle assembler** — the next M-UI
Verify-closing feature; now safely consumes `Resolve` (`decode realm + 12-bit hub_id → issuing hub`)
and re-engages the oracle conformance gate.
The ADR-0011 Go 1.26 / iscc-lib bump stays toolchain-gated (local is go1.24.13) — needs an iteration
where mise can provision Go 1.26; do not flip `go.mod`'s `go` directive before then.

**Notes:**
- `Hub.HubID` is now `*uint16` — a public-shape change to the `Hub` struct, but `Resolve` is the only
  reader and it is in-package (registry→`HubTarget` wiring still deferred), so blast radius is the one
  test struct-literal comparison the advance updated. Option A from `next.md` (minimal, YAML-idiomatic,
  WASM-pure) over a custom `UnmarshalYAML` — correct call.
- `Resolve` defensively skips a nil `HubID` even though a parsed `*HubList` never carries one — keeps
  the method fail-closed for any hand-built list; documented in the `Hub` docstring. Good.
- General lesson recorded in `learnings/registry.md`: a `url.URL` "host only" guard must enumerate ALL
  shape-carrying fields (`Path RawQuery ForceQuery Fragment Opaque User`), not just the obvious three —
  this is exactly the field the advance (and `next.md`) missed.
- No remote push issues anticipated; remote `origin` configured, tracking `origin/develop`. Pushing on
  this PASS_WITH_NOTES.

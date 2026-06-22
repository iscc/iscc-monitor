## 2026-06-22 — Normalize Surface-C `readTarget` to return the parsed `u.href`, not the raw monitor string

**Done:** Changed `readTarget` (`internal/verifier/verifier.html`) to return the WHATWG-normalized
monitor URL (`{ monitor: u.href, id: id }`) instead of the raw `?monitor=` query string, so an
opaque-scheme form (`https:example.com`) now flows downstream as `https://example.com/` — the guard's
own parsed-and-validated URL is the single source of truth. This closes the only pure-code-closable
`normal` ("Surface-C `readTarget` accepts opaque-scheme monitor forms the Go `parseTarget` rejected").

**Files changed:**
- `internal/verifier/verifier.html`: `readTarget` returns `u.href` (was the raw `monitor` string);
  leading comment updated to state it returns the normalized URL (evergreen docstring). The protocol/
  host/hash reject branches are untouched — only WHAT a passing target returns changed, not WHETHER it
  passes. The downstream `bundleURL` (`:588`) already strips the trailing `/` (`u.href`'s `/`) before
  appending `/inclusion/<id>.bundle`, so no other edit was needed; `target.id` is unchanged.
- `internal/verifier/handler_test.go`: added `TestVerifierReadTargetReturnsNormalizedURL` — a
  mutation-provable markup assertion that the rendered body contains `return { monitor: u.href, id: id };`
  and does NOT contain `return { monitor: monitor, id: id };`.

**Verification:** `mise run check` → green (build + vet + `go test ./...`, all 28 packages ok);
`gofmt -l .` empty. Per-criterion:
- [x] `go test -count=1 -run TestVerifier ./internal/verifier` → PASS (whole suite, incl. the unchanged
  `TestVerifierNoServerSideTarget`, `TestVerifierNoExternalCDN`, `TestVerifierStaticBodyAlwaysCarriesLoader`).
- [x] No-CDN ban green: `u.href` is a runtime expression computed from `location.search`, never a static
  literal; no `"https://"`/`"http://"` literal was introduced (the existing `"http:"`/`"https:"`
  protocol-comparison literals stay as-is). `TestVerifierNoExternalCDN` passes.
- [x] Mutation (independent): reverting the production return to `{ monitor: monitor, id: id }` makes
  `TestVerifierReadTargetReturnsNormalizedURL` FAIL; restored byte-identical, the return at `:556` is
  `{ monitor: u.href, id: id }`.
- [x] Oracle/conformance gate — N/A. No signature / RFC-6962 / Merkle / `proof` / `didweb` / `logclient`
  / fork-shrink-equivocation path touched; this is a JS-source normalization fix inside a `<script>`
  block on a pure static HTML render (no go-test JS-execution gate exists — markup-golden by design).

**Next:** With this drained, the remaining open `normal`s are no longer pure-code-closable in one
package: (1) the WASM-verifier signature half (no browser did:web checkpoint-signature check) is the
front-of-queue design-first / STOP-candidate — do a design pass before touching `verifier.html` or the
WASM core; (2) the certificate §6 per-record timestamp needs a store schema column + follower-ingest
write; (3) the `/` Checkpoint/Anchor data columns + config-driven instance identity need a store/
projection + config change. The WASM-verifier "published" half stays human-blocked (Pages custom-domain
repo-settings step). DONE requires 0 normal — the count is now 4 normal (was 5) after this close.

**Notes:**
- Strictly in scope: 2 files (1 production + 1 test), no `## Not In Scope` item touched — no JS
  execution harness added, signature half untouched, three-state verdict / no-CDN ban / loader fetch
  flow / `?` ForceQuery handling all unchanged, no server-side target parse/island reintroduced
  (`TestVerifierNoServerSideTarget` still renders byte-identical for a query-bearing request).
- The fix is a correctness/normalization-hygiene change, NOT a trust-boundary change: `readTarget` is a
  usability guard; the browser re-fetches AND WASM-re-verifies the bundle regardless (per
  `learnings/verifier.md`). Did not over-engineer an origin-prefix match against the raw input — the
  issue's primary remedy (return parsed `u.href`) is sufficient and minimal.
- The `?`/`ForceQuery` trailing-query case (the same harmless permissiveness class) is left out of scope
  as `next.md` directs; it stays a documented harmless divergence, not regressed.
- Visual check not applicable here (no layout/region change — a JS-source return value computed at
  runtime, never displayed in the no-JS body). The named-region/affordance markup is byte-unchanged.

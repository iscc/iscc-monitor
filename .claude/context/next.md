# Next Work Package

## Step: Normalize Surface-C `readTarget` to return the parsed `u.href`, not the raw monitor string

## Advances
Closes the open `normal` issue **"Surface-C `readTarget` accepts opaque-scheme monitor forms
(`https:example.com`) the Go `parseTarget` rejected — JS port is more permissive"** (`issues.md`).
This is the only **pure code-closable `normal`** left (per `state.md` "Next Milestone" §2 and the latest
`review` handoff "Next" — the other open normals each need a store-schema / config / design-first change).
It serves the **WASM verifier upgrade** milestone's Surface-C verifier (`monitor.iscc.codes`): the verifier
loader should fetch the bundle from a normalized monitor origin so the guard's own normalization is the
single source of truth, not the raw user string. DONE requires `0 normal`, so draining this advances the
gate while the front-of-queue WASM "published" half stays human-blocked.

## Goal
Make `readTarget` (`internal/verifier/verifier.html`) return the WHATWG-normalized `u.href` (the parsed
URL) instead of the raw `monitor` query string, so the downstream bundle fetch uses the normalized origin
and an opaque-scheme form (`https:example.com`) flows as `https://example.com/…` — the guard's parsed URL
is what propagates, eliminating the raw-string passthrough the issue flags.

## Scope
- **Create**: (none)
- **Modify**: `internal/verifier/verifier.html` (one production file — change `readTarget`'s return to use
  `u.href`; update the function's leading comment to describe returning the normalized URL)
- **Modify (test)**: `internal/verifier/handler_test.go` (add a markup assertion that `readTarget` returns
  the parsed URL, not the raw `monitor` — mutation-provable)
- **Reference**:
  - `.claude/context/learnings/verifier.md` — Surface-C mechanics: `readTarget` is a usability guard NOT a
    trust boundary; fail-closed to baseline (return `null`), never an `error` render; the JS-vs-Go
    opaque-URL permissiveness note (the exact gap this step closes); the no-CDN body ban runs the no-target
    baseline so the runtime monitor URL never enters the static body.
  - `internal/verifier/verifier.html:537-554` — current `readTarget` (`return { monitor: monitor, id: id }`)
  - `internal/verifier/verifier.html:588` — `bundleURL` (the lone consumer of `target.monitor`)
  - `internal/verifier/handler_test.go` — existing markup golden tests (the test seam is markup-only; there
    is no JS execution harness, so the close is asserted on the rendered `readTarget` source)

## Not In Scope
- Do NOT add a JS execution harness / headless-browser test — the verifier suite is markup-golden by design
  (`learnings/verifier.md`); assert on the rendered `readTarget` source, the same way every other verifier
  region is tested.
- Do NOT touch the WASM verifier's **signature half** (`internal/proof/verify`, `cmd/wasm`, the
  `verifier.html` "did:web key" success copy) — that is the separate design-first `normal` / STOP-candidate.
- Do NOT change the three-state verdict logic, the no-CDN ban, the loader fetch flow, or the `?` /
  `ForceQuery` trailing-query handling (that case is the same harmless class and stays out of scope).
- Do NOT reintroduce any server-side target parse/island — the target stays client-side from
  `location.search` (`TestVerifierNoServerSideTarget` must keep passing byte-identical renders).
- Do NOT also close the certificate §6 timestamp, the `/` sub-region deltas, or the Pages docs note — each
  is a separate issue requiring a store-schema / config / human-coordination change.

## Implementation Notes
- The single behavioral change: in `readTarget`, `return { monitor: u.href, id: id }` (was
  `{ monitor: monitor, ... }`). `u` is the already-constructed `new URL(monitor)`; `u.href` is the WHATWG
  serialization, so `new URL("https:example.com").href === "https://example.com/"` — the opaque-scheme form
  is normalized to a real origin+path. The downstream `bundleURL` (`:588`) already does
  `target.monitor.replace(/\/+$/, "")`, which trims the trailing `/` from `u.href` before appending
  `/inclusion/<id>.bundle`, so the fetch URL is correct with no other edit. `target.id` is unchanged.
- Keep the guard fail-closed: the protocol/host/hash checks (`:550`) still run on `u` before the return; an
  invalid/absent target still returns `null` (baseline untouched, never an `error` render). Do not loosen
  any existing reject branch — the fix changes only WHAT a *passing* target returns, not WHETHER it passes.
- Update the `readTarget` leading comment to state it returns the **normalized** monitor URL (`u.href`), so
  the docstring matches the code (CLAUDE.md "evergreen comments describe the current state").
- No-CDN ban safety: `u.href` for an `https:` target contains `https://`, but `readTarget` is JS *source*
  inside a `<script>` block; `TestVerifierNoExternalCDN` runs the no-target baseline and the `u.href` value
  is computed only at runtime from `location.search` — it never appears in the static body. Do NOT introduce
  a static literal `"https://"`/`"http://"` anywhere (the existing `"http:"`/`"https:"` protocol-comparison
  literals stay as-is); `u.href` is a runtime expression, not a literal, so the ban stays green (confirm
  `TestVerifierNoExternalCDN` still passes under `mise run check`).
- Relevant learnings rule: `readTarget` is a **usability guard, NOT a trust boundary** — the browser
  re-fetches AND WASM-re-verifies the bundle, so this is a correctness/normalization hygiene fix, not a
  security boundary change. Do not over-engineer it into an origin-prefix match against the raw input; the
  issue's primary remedy ("return the parsed `u.href`") is sufficient and minimal.

## Verification
- `mise run check` is green (build + vet + `go test ./...` all 28 packages, `gofmt -l .` empty).
- `go test -count=1 -run TestVerifier ./internal/verifier` passes (the whole verifier suite, including the
  unchanged `TestVerifierNoServerSideTarget`, `TestVerifierNoExternalCDN`,
  `TestVerifierStaticBodyAlwaysCarriesLoader`).
- A new/extended markup assertion proves the normalization: the rendered body contains the normalized
  return (e.g. `monitor: u.href`) and does **not** contain the raw-string return (e.g. `monitor: monitor`).
  The assertion must be **mutation-provable** — reverting the return to `{ monitor: monitor, id: id }` makes
  that verifier test FAIL.

## Done When
`mise run check` is green and `go test -run TestVerifier ./internal/verifier` passes with the new
mutation-provable assertion confirming `readTarget` returns the parsed `u.href` (not the raw monitor
string), closing the Surface-C `readTarget` `normal`.

## 2026-06-22 — Reject a trailing `?` (ForceQuery) in the Hub-List `hubDomain` bare-host guard

**Done:** Added `|| u.ForceQuery` to the `hubDomain` bare-host guard in
`internal/registry/registry.go` so a url whose only non-host component is a bare trailing `?` (e.g.
`https://sb0.iscc.id?`) — which `net/url` records as `ForceQuery == true` with `RawQuery == ""` and
round-trips through `u.String()` — fails closed with the existing "not a bare host base url" error,
instead of slipping the query delimiter into the resolved hub domain. Reused the existing wrapped
error message (no new message), updated the docstring + in-line guard comment to name `ForceQuery`,
and added a mutation-proven table case.

**Files changed:**
- `internal/registry/registry.go`: `hubDomain` guard now also rejects `u.ForceQuery`; docstring (the
  `hubDomain` comment block) and a one-line in-line comment updated to name the trailing-`?`
  (ForceQuery) reject so the evergreen comment matches the code. No new import (`ForceQuery` is a
  field on the already-parsed `*url.URL`); imports stay `{bufio bytes fmt net/url strings yaml.v3}`.
- `internal/registry/hublist_test.go`: added one `TestParseHubListErrors` case
  `"trailing question mark (ForceQuery)"` (`url: https://sb0.iscc.id?`, `errFrag: "not a bare host
  base url"`), placed beside the path-bearing-url cases.

**Verification:** `mise run check` → green (build + vet + `go test ./...`, all 27 packages ok);
`gofmt -l .` empty. Per-criterion:
- [x] `go test -count=1 -v -run TestParseHubListErrors ./internal/registry` passes; `-v` lists the
      new `trailing_question_mark_(ForceQuery)` subtest.
- [x] Mutation: removing `|| u.ForceQuery` makes `.../trailing_question_mark_(ForceQuery)` FAIL — the
      failure dump confirms `ParseHubList` returns a non-nil `*HubList` (`URL:"https://sb0.iscc.id?"`,
      i.e. the `?` survived) alongside a nil error; restoring the clause returns green. `registry.go`
      restored byte-identical (only the additive diff remains).
- [x] Clean-domain regression: `TestParseHubListGolden` (the `https://sb0.iscc.id` / `sb1.amlet.id`
      golden) still parses unchanged.
- [x] `go.mod`/`go.sum` byte-identical (`git diff` empty). `GOOS=js GOARCH=wasm go build
      ./internal/registry` still builds — leaf stays WASM-shareable / import-clean.
- [x] Oracle/conformance gate N/A: no proof/verify/didweb/merkle/signature path touched.

**Next:** Continue closing code-closable `normal`s in handoff-named order. With the `hubDomain`
ForceQuery fail-open now closed, the next items are (2) the §5 OTS-digest-binding `bytes.Equal` gap
and (3) the certificate tier-2 honesty-copy overstatement (`cert.html:465`) — each its own step. The
§6 `· at` timestamp needs a store schema column (larger). The front-of-queue WASM-verifier signature
half remains design-first / a STOP-candidate (browser did:web resolution) — do a design pass before
touching `verifier.html` copy.

**Notes:**
- Scope held exactly to `next.md`: one production file + one test file, no new import, nothing from
  `## Not In Scope` touched (did not also reject `u.Opaque`/`u.User`/`#`-only `Fragment`, did not wire
  `ParseHubList`/`Resolve` into the realm-loading path, did not change the `Resolve`/`Hub`/`HubList`
  public shape). The `learnings/registry.md` "enumerate `url.URL`'s shape-carrying fields" rule
  established this is the one live load-bearing fail-open (`https://host#` is dropped by Go on
  round-trip; `Opaque` is unreachable for an `https://`-scheme'd host).
- The mutation dump is the clearest evidence the case is non-vacuous: it shows the exact fail-open it
  guards — a parsed hub whose `URL` retains the trailing `?`.

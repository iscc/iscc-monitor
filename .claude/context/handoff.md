## 2026-06-22 — Review of: Reject a trailing `?` (ForceQuery) in the Hub-List `hubDomain` bare-host guard

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** The advance added `|| u.ForceQuery` to the single `hubDomain` bare-host reject in
`internal/registry/registry.go` so a url whose only non-host component is a bare trailing `?` (e.g.
`https://sb0.iscc.id?`) — which `net/url` records as `ForceQuery == true` with `RawQuery == ""` —
fails closed with the existing "not a bare host base url" wrapped error instead of round-tripping
the delimiter into the resolved domain. Scope is exemplary: exactly one production file + one test
file, no new import, nothing from `## Not In Scope` touched; the docstring + in-line comment were
updated so the evergreen comment matches the code. The `normal` ForceQuery fail-open is fully closed
and mutation-proven.

**Verification:**
- [x] `mise run check` — green (build + vet + `go test ./...`, all 27 packages ok; exit 0 reconfirmed
      independently).
- [x] `gofmt -l .` — empty (no formatting failures).
- [x] `go test -count=1 -v -run TestParseHubListErrors ./internal/registry` — PASS; `-v` lists the new
      `trailing_question_mark_(ForceQuery)` subtest among the 8 cases.
- [x] Mutation (independent) — reverting `|| u.ForceQuery` makes
      `TestParseHubListErrors/trailing_question_mark_(ForceQuery)` FAIL; the failure dump shows the exact
      fail-open it guards (a parsed `HubList` retaining `URL:"https://sb0.iscc.id?"`). `registry.go`
      restored byte-identical (`git diff --stat` empty).
- [x] Clean-domain regression — `TestParseHubListGolden` (`https://sb0.iscc.id` / `sb1.amlet.id`) still
      parses unchanged.
- [x] `go.mod`/`go.sum` byte-identical vs HEAD~1 (no dependency drift). `GOOS=js GOARCH=wasm go build
      ./internal/registry` builds — leaf stays WASM-shareable / import-clean.
- [x] Oracle/conformance gate — N/A: no proof/verify/didweb/merkle/signature/fork-shrink-equivocation
      path touched; `internal/registry` is a pure leaf.
- [x] Gate-circumvention scan over all unpushed commits (`@{upstream}..HEAD`) — no `//nolint`, `t.Skip`,
      build-tag exclusion, swallowed error, or deleted assertion in added Go lines. The only Go diff in
      the unpushed range is the two scoped registry files (27+/7-); the other two unpushed commits are
      cid context-doc commits.

**Issues found:** (none new). Resolved + deleted the `normal` "Hub-List `hubDomain` accepts a trailing
`?` (ForceQuery fail-open)" issue after independently mutation-verifying the fix.

**Codex second opinion:** Clean — "The change narrowly rejects bare trailing query delimiters in
Hub-List URLs and adds a regression test for that case. I found no introduced correctness, security,
or maintainability issues." Independently corroborates the reviewer's verification; no findings to
triage.

**Visual check:** n/a — no SSR surface changed. The diff touches only the `internal/registry` pure
leaf (URL-shape guard + its test); no `dashboard`/`dossier`/`web`/`certificate`/template surface and no
rendered output is affected.

**Next:** Continue closing code-closable `normal`s in handoff-named order. Remaining: (1) the §5
OTS-digest-binding `bytes.Equal` gap (`internal/certificate/handler.go:843-845` + `internal/ots`); (2)
the certificate tier-2 honesty-copy overstatement (`cert.html:465`). The §6 `· at` timestamp needs a
store schema column (larger). The front-of-queue WASM-verifier signature half remains design-first / a
STOP-candidate (browser did:web resolution) — do a design pass before touching `verifier.html` copy.

**Notes:**
- Scope held exactly to `next.md`: the change is purely additive (one operand on one condition + an
  evergreen-comment update + one table case). Nothing from `## Not In Scope` (no `u.Opaque`/`u.User`/
  `#`-only fragment reject, no live-wiring of `ParseHubList`/`Resolve`, no public-shape change).
- The registry detail learnings now records all three landed fail-open guards as `settled (landed)`;
  the `ForceQuery` trap moved from "open issue" to settled. No new durable cross-cutting rule to promote
  to the index (the `url.URL` shape-field enumeration rule already lives in the detail file).
- 1 advance commit ahead of `origin/develop` (plus the 2 cid context commits). Pushing on PASS. The
  known `Pages` workflow failure on develop is the human-blocked custom-domain repo-settings step, not
  a code regression.

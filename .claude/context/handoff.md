## 2026-06-21 — Review of: Hub dossier skeleton — `GET /<domain>` Evidence-Ledger page (status + coverage honesty), no-JS

**Verdict:** NEEDS_WORK
**Loop:** CONTINUE

**Summary:** The dossier page itself is well-built: a faithful third copy of the dashboard
status/coverage shape, honest coverage rendering, the five-status badge overlay, no-JS/no-CDN DS
shell, and a clean mux mount proven to coexist with the `/<domain>/log/` subtree. All Verification
criteria pass and `mise run check` is green. BUT Codex surfaced a real, reviewer-reproduced defect the
advance introduced: the new bare-domain mount panics `buildMux` at startup if a realm entry uses a
single-label host that collides with a built-in exact route (`metrics`/`healthz`). That blocks PASS;
the root-cause fix (reject reserved mount names before mounting) is the next advance.

**Verification:**
- [x] `mise run check` — green (all 19 packages `ok`; build/vet/test pass).
- [x] `gofmt -l .` — empty (also after my comment re-wrap fix).
- [x] `go test -count=1 ./internal/dossier` — PASS (new package, 7 tests).
- [x] `go test -count=1 ./cmd/iscc-monitor` — PASS; `TestMirrorRouter` proves `/sb0.iscc.id` (200
  text/html) AND `/sb0.iscc.id/log/` subtree both resolve on the shared mux (re-ran `-v`, all subtests pass).
- [x] DS shell wired — `href="/_ds/tokens.css"`, `href="/_ds/fonts.css"`, `var(--font-sans/mono)` — asserted at the seam.
- [x] Domain through `hubStatusBadge` partial (`class="hub-status-badge"` + label + silhouette marker) — PASS.
- [x] Coverage honesty (ADR-0001): covered → `size 42 at 2023-11-14T22:13:20Z`; uncovered → "no coverage
  yet" with Observed-size `0` clearly labeled and distinct (independently re-rendered the body to confirm).
- [x] In-memory overlay reaches the dossier (verified+live-unresolvable → renders unresolvable, NO
  `data-status="verified"`); CSS-literal trap handled (unquoted `[data-status=verified]` selectors,
  0 quoted literals in the template — negative assert is honest).
- [x] No `<table>`, no `http://`/`https://`/`cdn.`/`jsdelivr` in body — asserted.
- [x] Leaf check: `go list -deps ./internal/store ./internal/badge | grep -E 'net/http|internal/dossier'` empty.
- [x] go.mod/go.sum byte-identical (not in the diff); oracle gate correctly N/A (pure HTML render).
- [x] Gate-integrity scan over the 3 unpushed commits — no `//nolint`/`t.Skip`/build-tag/swallowed-error/loosened gate; the diff only ADDS tests.
- [x] Mutation-proven non-vacuous independently: broke `/_ds/fonts.css` href → FAIL; `{{- if false}}`
  coverage branch → FAIL; bare `{{.Status}}` instead of badge → FAIL; neutered `overlayStatus` (always
  return store status) → `TestOverlayStatusPrecedence` FAIL. All reverted byte-identical, tree clean.
- [ ] **Reserved-domain startup safety** — FAIL. A realm entry `metrics`/`healthz` panics `buildMux`
  (reproduced: `pattern "/metrics" … conflicts`). See Issues.

**Issues found:**
- **[BLOCKS PASS, normal] Bare-domain dossier mount collides with reserved exact routes** — `main.go:205`
  registers `/`+Domain inside `mirrorHandler`, which `buildMux` runs BEFORE mounting the exact
  `/metrics`/`/healthz`/`/_ds/`. `registry.Parse` accepts any non-URL token as a Domain, so a realm line
  `metrics` makes the dossier register `/metrics` first → the later built-in `mux.Handle("/metrics", …)`
  panics → monitor fails to start. Reviewer reproduced the exact panic through `buildMux`. This is a
  misconfiguration crash (real hub domains are multi-label), but it is a NEW crash surface this advance
  introduced (the prior `/<domain>/log/` subtree never collided with the exact `/metrics`). Root-cause
  fix: reject/skip a reserved-or-empty Domain before mounting (prefer failing `registerHubs`/`Parse`
  loudly). Filed in `issues.md` with a repro + verify recipe.
- Pre-existing `low` issues untouched (notecheck `out` param, Mirror seam, proofserve `writeReadError`).
  The overlay-duplication issue is now 3x (dossier added a third verbatim copy) — re-titled + noted, still `low`.

**Codex second opinion:** Completed (exit 0, ~5 min). One finding — **[P2] reserved dossier mount
names collide with built-in `/metrics`/`/healthz` → startup panic** (`main.go:205`). **CONFIRMED REAL**
by reproduction (filed as the blocking issue above). No other findings. Codex's other note ("other
tested behavior appears intact") matches my independent review.

**Next:** Fix the reserved/empty-domain mount collision at the root before adding more dossier surface:
reject a `Domain` equal to a reserved mount name (`metrics`, `healthz`, the `web.Prefix` segment) and
the empty case in `registerHubs` (or `registry.Parse`), with a `buildMux` test driving a reserved name.
Then resume the planned frozen **Exhibit** sub-step (needs a NEW `store.ListViolations(hubID)` read over
the `violations` table — only `RecordViolation` exists today — plus the non-dismissable Exhibit markup,
ADR-0006). After that, the remaining M-UI SSR screens (record list / single record / certificate — the
certificate re-engages the oracle/inclusion-proof gate).

**Notes:**
- Minor fix applied by review: re-wrapped a 132-char doc-comment line + a dangling "The same metrics"
  fragment in `buildMux`'s comment (`main.go:154-157`) that the advance left unwrapped. Comment-only, no
  behavior change; gofmt clean, build green.
- The dossier page quality is genuinely good — only the wiring edge case fails. The fix is small and
  well-scoped; this is NEEDS_WORK on a single root cause, not a rewrite.
- `hubRoute.Domain` is load-bearing (an empty Domain mounts `/` and collides with the dashboard) — the
  reserved-name fix should cover the empty case too (the handoff flagged it; both are the same class).
- No push (verdict is NEEDS_WORK). Remote is configured; the next PASS cycle pushes `develop`.

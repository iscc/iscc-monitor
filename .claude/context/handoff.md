## 2026-06-21 — Review of: Embed the DS v2 token CSS as a shared `internal/web` leaf and link it from `GET /`

**Verdict:** PASS_WITH_NOTES
**Loop:** CONTINUE

**Summary:** `advance` (`ffa6049`) stood up a pure stdlib `internal/web` leaf that `go:embed`s a
concatenated, CDN-free DS v2 token stylesheet and serves it at the exact path `/_ds/tokens.css`
(GET→200 text/css, non-GET→405), mounted it in `buildMux`, and linked it from the dashboard `<head>`.
The diff is scope-clean (3 non-test source files at the limit), the port is verbatim-faithful (grain
`background-image` neutralized, `fonts.css` excluded), and every gate is green. One confirmed Codex
[P2] (immutable cache on a stable overwrite-in-place URL) is filed as a `normal` follow-up — it is
cosmetic and spec-sanctioned, so it does not block this increment.

**Verification:**
- [x] `mise run check` green — all 18 packages `ok` (build + vet + test).
- [x] `gofmt -l .` empty (whole tree).
- [x] `go test -count=1 -run 'TestTokens|TestDashboard' ./internal/web ./internal/dashboard` — PASS (uncached).
- [x] `internal/web` test: GET→200 `text/css` + `--iscc-blue` present; POST→405; body has no `jsdelivr`/`http`/`url(` — PASS.
- [x] CDN-free non-vacuous (my mutation, reverted): appended an `https://cdn.jsdelivr.net` line → `TestTokensCDNFree` FAILS all three asserts; restored byte-identical → green.
- [x] Dashboard-link non-vacuous (my mutation, reverted): dropped the `<link>` → `TestDashboardLinksTokensNoCDN` FAILS; restored byte-identical → green.
- [x] End-to-end through full `buildMux` (throwaway test, removed): `GET /_ds/tokens.css`→200 text/css + `immutable` Cache-Control + CORS `*`; POST→405; `GET /`→200 html linking the css; `GET /_ds/fonts/x.woff2`→404 (exact-path mount, no subtree leak).
- [x] Leaf purity: `go list -deps ./internal/web` internal closure is only `internal/web`; no store/metrics/logclient. `GOOS=js GOARCH=wasm go build ./internal/web` → OK.
- [x] `git diff --stat HEAD~1..HEAD -- go.mod go.sum internal/store/schema.sql` empty (no dep/schema change).
- [x] Port fidelity: only `url(`/CDN reference across colors/typography/spacing/base is the grain `background-image` (base.css:46), which is dropped; `fonts.css` (all jsDelivr URLs) excluded; size/blend no-ops kept.
- [x] Gate-integrity scan over unpushed range (`@{upstream}..HEAD`) — no `//nolint`/`t.Skip`/build-tag/swallowed-error/deleted-assertion. The `_, _ = ...WriteTo(w)` site is the documented post-200 write-drop convention.
- [x] Oracle/conformance gate correctly N/A — pure static-asset transport + a static `<link>`; no signature/RFC-6962/Merkle/did:web/fsck/proof path.

**Issues found:** One — Codex [P2], reviewer-confirmed (see below). Filed in `issues.md` as `normal`
(`/_ds/tokens.css` immutable cache on a stable URL). No defect that blocks this increment.

**Codex second opinion:** First run reviewed the WRONG commit. HEAD had advanced past the advance
(`ffa6049`) to a human loop-doc commit (`4a8a313 fix(review-agent): capture only Codex's verdict`), so
`--commit HEAD` reviewed the agent-doc change, not the tokens work ("only updates the review-agent
instructions … no issue"). I re-ran `codex review --commit ffa6049` (the real increment). Its one
finding:
- **[P2] immutable Cache-Control on the stable `/_ds/tokens.css` URL (`web.go:42`) — CONFIRMED.** The
  dashboard links the stable, non-content-addressed path, so a redeploy overwrites the bytes in place
  while clients can pin the year-long `immutable` response → new HTML, stale CSS. I verified this
  contradicts the project's OWN convention: `internal/tilesserve` uses `immutable` only for
  content-addressed FULL tiles and `no-cache` (+ strong ETag) for overwrite-in-place resources, whose
  comment explicitly warns against `immutable`. `next.md` sanctioned `immutable` ("build-pinned"), but
  build-pinned ≠ content-addressed. Cosmetic (stale tokens; no correctness/security/trust-root impact)
  → `normal` issue for a later M-UI slice, not a NEEDS_WORK blocker. Fix: `no-cache`+ETag+304 (the
  `tilesserve.writeBlob` shape) or a fingerprinted path.

**Next:** The fonts sub-step — fetch + commit the Readex Pro / JetBrains Mono woff2 binaries,
`go:embed` them, serve under `/_ds/fonts/...`, add a self-hosted `@font-face` stylesheet (its own ≤3-file
change). **When that lands, apply the `no-cache`+ETag cache fix to BOTH `/_ds/...` assets** (it touches
`internal/web` anyway — fold the issue fix in there) and **narrow the CDN-free `url(` ban** to
"no external `url(`" (self-hosted `src: url("/_ds/fonts/...")` is a legitimate same-origin `url(`).
After fonts: the realm-index redress (table → Evidence-Ledger grid + token classes), then thread tokens
into the log browser / dossier / record / certificate surfaces.

**Notes:**
- HEAD layout this iteration: the advance is `ffa6049` (HEAD~1); a human commit `4a8a313` (loop-doc
  only, `.claude/agents/review.md`: split Codex stdout/stderr) sits on top. My code review was of
  `ffa6049` (the context diff snapshot predated the human commit and showed exactly its content). The
  `4a8a313` change touches no source code — no gate impact.
- Scope clean: 2 new source files (`internal/web/web.go`, `tokens.css`) + 1 modified (`cmd/iscc-monitor/main.go`)
  = 3 non-test, at the limit; nothing from `## Not In Scope` touched (no `fonts.css`/`@font-face`/woff2/
  grain asset/markup restyle/log-browser wiring/store-metrics import/dep/schema change).
- Learnings: created `learnings/web.md` (+ index pointer row) recording the immutable-on-stable-path
  trap, the exact-path-mount guard, the `url(`-ban-vs-coming-fonts narrowing, the dashboard `https://`
  ban relying on scheme-less `h.origin`, and the WASM-green leaf shape. No cross-cutting index promotion
  (all package-local). Budgets OK.
- Milestones: M1/M2/M3 met; M-UI in progress (~8 open), WASM verifier + OTS not started — so Loop =
  CONTINUE, not DONE. No human-only decision open (the cache finding is a sanctioned `normal` follow-up),
  so not STOP.
- Working tree clean after review (all mutations + the throwaway e2e test reverted; no codex `.scratch`).

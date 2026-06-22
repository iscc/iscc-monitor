## 2026-06-22 — Review of: GitHub-Pages publish workflow for the Surface-C verifier (`monitor.iscc.codes`)

**Verdict:** PASS_WITH_NOTES
**Loop:** CONTINUE

**Summary:** The advance adds `.github/workflows/pages.yml` (the modern Actions Pages build→deploy of
the Surface-C verifier) plus a tracked `.github/pages/CNAME` and one `CLAUDE.md` doc line — closing the
"published" half of the WASM milestone's Verify bar by deploying the byte-pinned verifier tree from
commit (copy-not-rebuild). The diff is tight (0 Go production files; 1 workflow + 1 domain file + 1 doc
line + context), all gates are green, the workflow shape matches the canonical Pages contract exactly,
and the deployed `verify.wasm` hash equals `web.WasmVerifyHash`. One confirmed Codex finding (P2): the
custom-domain binding is a one-time repo-Settings step (the artifact CNAME is a no-op under Actions),
already flagged for the human and now filed `normal` — does not block the increment's stated goal.

**Verification:**
- [x] `mise run check` (build + vet + test) — GREEN, all 27 packages ok (incl. `cmd/verifier-site`).
- [x] `gofmt -l .` — empty (no formatting failures).
- [x] `pages.yml` is valid YAML — parsed via cached `gopkg.in/yaml.v3` into `map[string]any`, exit 0, 5
  top-level keys (name/on/permissions/concurrency/jobs); deep-parsed: `build` job = 6 steps
  (checkout@v4 → setup-go@v5 go-1.26 → `go run ./cmd/verifier-site -out dist` → `cp …/CNAME dist/CNAME`
  → configure-pages@v5 → upload-pages-artifact@v3 `path: dist`), `deploy` job = `needs: build`,
  `environment: github-pages`, deploy-pages@v4. Permissions {contents:read, pages:write, id-token:write},
  concurrency {group:pages, cancel-in-progress:false}, triggers push[develop]+workflow_dispatch (no PR).
- [x] `go run ./cmd/verifier-site -out /tmp/pages-verify` exits 0 and writes the full **14-file** tree
  (index.html + 5 named `/_ds/` assets + 8 woff2) — reviewer re-ran; `find … | wc -l` = 14.
- [x] Deployed-tree `verify.wasm` SHA-256 = `7d57ab1bbc0b…f22d2c` == `web.WasmVerifyHash` (web.go:93) —
  reviewer `sha256sum`-verified; the workflow has NO `mise run build:wasm`/`GOOS=js` step (copy-not-
  rebuild, grep-confirmed). `TestGenerate` uncached re-pins this: PASS.
- [x] CNAME content exactly `monitor.iscc.codes` (`grep -qx` exit 0; 19 bytes, one trailing newline);
  the workflow `cp`s it to `dist/CNAME` after the render so it lands at the artifact root.
- [x] `grep -q "pages: write"`, `grep -q "deploy-pages"`, and `grep -q "go run ./cmd/verifier-site"` all
  exit 0 on `pages.yml`.
- [x] Scope: only `.github/workflows/pages.yml` (new), `.github/pages/CNAME` (new), `CLAUDE.md` (the one
  allowed doc line) + handoff — 0 non-test/doc production files (≤3). No Not-In-Scope path touched
  (`cmd/verifier-site/main.go`, `cmd/wasm`, `internal/proof/verify`, `.github/workflows/ci.yml`,
  `internal/dossier`, `internal/certificate` all untouched — verified).
- [x] Gate-integrity scan over the 3 unpushed commits (`origin/develop..HEAD`) — no `//nolint`/`t.Skip`/
  build-tag/swallowed-error/deleted-assertion in the code diff; working tree clean (no stray binary, no
  tracked `dist/`).
- [x] Oracle/conformance gate — N/A: no signature/RFC-6962/Merkle/did:web/fsck/proof code touched (CI
  plumbing + a domain file + a doc line); the published `verify.wasm` is the already-golden byte-pinned
  blob, hash re-verified equal here.

**Issues found:**
- (Codex P2, confirmed → filed `normal`) The artifact `CNAME` does NOT bind the custom domain under
  Actions-based Pages deploy — `actions/deploy-pages` ignores a `CNAME` in the uploaded artifact; the
  custom domain `monitor.iscc.codes` (and the "GitHub Actions" Pages source) must be set once in repo
  Settings/API. Until configured, the deploy lands at the default project URL `iscc.github.io/iscc-monitor/`
  where the page's root-absolute `/_ds/...` asset paths 404 (reviewer-confirmed the refs are root-
  absolute). Real but contingent on the domain not being configured; the handoff already flagged the
  human step; the `cp CNAME` is a harmless intent-documenting no-op under Actions (correct mechanism for
  a branch source). Fix = a deploy-setup doc note naming the one-time Settings/API step. Does NOT block.

**Codex second opinion:** Codex ran (slow — ~14 min, completed exit 0) and produced ONE finding, [P2]
"Configure the custom domain in Pages settings" at `pages.yml:50`. Triage: CONFIRMED real on the
mechanism — under `actions/deploy-pages` GitHub takes the custom domain from repo Settings, not from an
artifact `CNAME`, so the `cp CNAME` step alone does not make the site serve `monitor.iscc.codes`. But
Codex's severity is OVERSTATED: (1) the workflow correctly builds + uploads the byte-pinned tree (its
stated goal); (2) the handoff already flags the human Settings step; (3) the "broken absolute asset
paths" half is CONTINGENT — root-absolute `/_ds/...` resolve correctly on the apex custom domain and
only break on the default project URL (i.e. only if the domain is NOT configured). Filed `normal` (a doc
follow-up), recorded the mechanism in `learnings/ci.md`. Not a code defect, not a gate weakening, does
not block.

**Visual check:** n/a — no SSR surface changed. This increment is CI plumbing (`pages.yml`), a domain
file (`CNAME`), and one doc line; no template or server-rendered handler was touched. The published
`index.html` is byte-identical to `verifier.Handler`'s already-reviewed output.

**Next:** The "published" half of the Surface-C WASM milestone is in place (deploy workflow landed).
Front-of-queue options for define-next: (1) the **dossier tier-2 WASM caller** (the OTHER WASM sub-step —
the dossier's `verify ↗` is still a static link, not a WASM island), which would advance the WASM
milestone; or (2) the **cross-origin verifier-scope `normal`** (the WASM core proves inclusion math only
— no checkpoint-signature/did:web-key/id-binding — so a malicious monitor can render a green `verified`,
and the Surface-C copy overstates this) — the most trust-root-meaningful WASM gap. Lower-value follow-ups:
the new Pages custom-domain doc note, the `verifier.readTarget` `u.href` normalization, the
`cmd/verifier-site` non-atomic-output `low`. Pick one WASM front.

**Notes:**
- The DRIFT WATCH (amber) from state.md is now SATISFIED for this front: this increment CLOSES the
  "published" half of the WASM Verify bar (the deploy workflow exists and is reproducible-from-commit),
  not more build plumbing around an unpublished artifact. The artifact is now deploy-on-push to `develop`.
- **Human action on first deploy (operational, not code):** set the repo Pages source to "GitHub Actions"
  AND the custom domain to `monitor.iscc.codes` in Settings → Pages (or via API), plus the
  `monitor.iscc.codes` DNS CNAME at the registrar. The workflow cannot assert these; until done the site
  serves at the default project URL with broken `/_ds/` paths (the new `normal` issue tracks the doc fix).
- `dist/` remains OUT of `.gitignore` (matches the established note: the test uses `t.TempDir()`; CI
  builds the tree fresh). Working tree clean across the reviewer's `go run`. Minor watch: a developer
  running `go run ./cmd/verifier-site` at the repo root on `develop` could accidentally `git add` a
  generated `dist/`; not currently an issue (none tracked), noted in case a future increment adds it.
- Issue count after this iteration: 0 critical / 11 normal / 8 low (one `normal` added — the Pages
  custom-domain doc gap; none resolved — no open issue touched the changed paths). CI green expected at
  this commit (the workflow file is outside the `mise run check` Go gate; `ci.yml` unchanged).

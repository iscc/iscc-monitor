## 2026-06-22 — Review of: Thread config-driven instance identity into the certificate masthead

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** The advance threads `dashboard.Identity` into the certificate handler
(`Handler(hubList, st, statuses, id)`), renders the configured instance + operator strings in the
`/inclusion/{iscc_id}` masthead, and falls back to today's static copy on a zero-value Identity —
completing the third (and final SSR-masthead) leg of the identity arc, byte-identical to the already-landed
`/` and dossier mastheads. The diff is exactly the spec's 3 production files + 2 test files (additive only:
a new mutation-proven identity test, 32 mechanical call-site updates, no gate weakening). I independently
re-ran both mutations, the full check suite, and a live visual pass; all green.

**Verification:**
- [x] `mise run check` green — build + vet + `go test ./...`, all 27 packages ok.
- [x] `go test -count=1 -run TestCertificate ./internal/certificate` — pass (existing tests under the new
  4-arg signature + the new identity test).
- [x] `go test -count=1 -v -run TestCertificateRendersInstanceIdentity ./internal/certificate` — PASS
  (populated path: both operator strings present, masthead placeholder absent; zero-value path: fallback +
  generic operator line render).
- [x] `go test -count=1 ./cmd/iscc-monitor` — pass (new `certificate.Handler(...,id)` signature compiles).
- [x] `gofmt -l .` (outside `cauldron/`) — empty.
- [x] Mutation A (reviewer-run + reverted byte-clean): `{{.Instance}}` → literal `monitor instance` in
  cert.html → `TestCertificateRendersInstanceIdentity` FAILS. Restored, tree clean.
- [x] Mutation B (reviewer-run + reverted byte-clean): `resolveIdentity` forced to `instance, operator = "", ""`
  → test FAILS. Restored, tree clean.
- [x] No-CDN ban intact — `monitor.iscc.codes` still present (3x); the diff introduces no
  `cdn.`/`jsdelivr`/`unpkg`/`googleapis`/`http://` token.
- [x] Byte-identical-chrome rule verified — the `.chrome-identity`/`.chrome-operator` CSS rule bodies AND
  the `<div class="chrome-actions">` masthead block are now byte-identical across dashboard.html,
  dossier.html, and cert.html.
- [x] No import cycle (dashboard does not import certificate/dossier); no purity regression (certificate is
  an HTTP handler, not a WASM-shared pure package — it already imports `net/http`).
- [x] Quality-gate integrity scan over all unpushed commits — no `//nolint`/`t.Skip`/build-tag/swallowed
  error/deleted assertion. All changes additive.
- [x] **Oracle gate: N/A** — pure HTML render of masthead strings; no signature / RFC-6962 / Merkle /
  did:web / fsck / proof / store path touched. `go.mod`/`go.sum`/`schema.sql` byte-identical (not in the diff).

**Issues found:** (none new) The two pre-existing related issues stay open and unblocking, as next.md
directed: the config-leaf env move (`normal`, the explicit NEXT sub-step) and the masthead-fallback-const
duplication (`low`) — I updated the latter's title/body to record the cert copy as the 3rd duplicate (was
"and cert next").

**Codex second opinion:** Clean. Verdict: "The identity is consistently threaded into the certificate
handler and rendered with existing fallback behavior, with call sites and tests updated. I did not identify
any introduced correctness issue." No findings to triage; matches my independent review.

**Visual check:** Built the binary and launched a fixture instance with the three identity env vars set,
then screenshotted `/inclusion/NOTANISCCID` (the honest-200 masthead path) via `agent-browser` and read the
PNG. The cert masthead renders the configured `monitor.iscc.id` / `instance operated by ISCC Foundation ·
ISCC mainnet` right-aligned beside the ISCC logo + `verify ↗ monitor.iscc.codes` tier-2 link — byte-identical
chrome to the already-verified dashboard/dossier mastheads, "CANNOT CERTIFY INCLUSION" honest state below.
No visual deltas to file.

**Next:** The config-leaf env move (the explicit NEXT sub-step, closes the open `normal`): move the three
identity env keys (`ISCC_MONITOR_INSTANCE` / `ISCC_MONITOR_OPERATOR` / `ISCC_MONITOR_REALM_NAME`) from
`main.go`'s inline `identity()` into `internal/config`'s `optional(get, key, fallback)` leaf (ratifying the
realm-name key name) and add them to CLAUDE.md's env table — a focused `config.go` + `main.go` + CLAUDE.md
change. After that: the three proofserve mastheads (`browser.html`, `records.html`, `record.html`) — the
natural trigger to ALSO fold the now-3x duplicated `instanceFallback`/`operatorFallback` consts + a single
exported `Resolve` into one shared leaf (closes the `low`). `internal/verifier` stays EXCLUDED (its `.codes`
chrome is the verifier-app identity).

**Notes:**
- The fallback-const + `resolveIdentity` copy now lives in THREE packages (dashboard owns
  `Identity.resolve`; dossier + cert carry byte-identical private copies with the "MUST stay byte-identical"
  comment). A masthead-copy change is now a three-site edit (four once proofserve lands) — best consolidated
  WITH the proofserve slice. Tracked `low`, updated this iteration.
- Test-collision trap recorded in `learnings/certificate.md`: a bare `Contains(body, "monitor instance")`
  placeholder-absence check is vacuous here (the cert footer at cert.html:419 carries "issued by this
  monitor instance"). The advance correctly pinned the masthead element `chrome-instance">monitor instance`
  instead — non-vacuous against the masthead, not the footer.
- The operator fallback const's literal `&` renders as `&amp;` (html/template text-node escape); the test
  correctly asserts the escaped form.
- A transient `.gitignore` working-tree modification (additive secrets/DB ignores, NOT in the advance
  commit, NOT in my review commit) appeared mid-review and reset itself — environmental, benign, not part of
  this increment.
- `learnings/certificate.md` (176 lines) and `dashboard.md` (154 lines) remain slightly over the ~150-line
  soft cap; I net-collapsed the §4/§5/§6 settled blocks this iteration to absorb the new masthead bullet.
  Both should be rotated harder when the masthead arc completes (the settled clause-by-clause detail is
  git-history material).

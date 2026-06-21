## 2026-06-21 — Hub dossier skeleton — `GET /<domain>` Evidence-Ledger page (status + coverage honesty), no-JS

**Done:** Stood up a per-hub dossier page at the bare `GET /<domain>` (e.g. `/sb0.iscc.id`): a new
`internal/dossier` leaf that renders one hub's five-status badge (store subset overlaid with the
in-memory live verdict) and honest coverage window in the Evidence-Ledger card pattern, wired into the
mux via a new exact-path mount alongside each hub's `/<domain>/log/` mirror subtree.

**Files changed:**
- `internal/dossier/handler.go` (new): `Handler(st, hubID, statuses)` + `dossierData` view-model +
  `findHub`/`buildData`/`overlayStatus`/`hubStatus`/`coverageTime`, ported from
  `internal/dashboard/handler.go` (deliberate third copy — consolidation is its own tracked step).
- `internal/dossier/dossier.html` (new asset): the Evidence-Ledger card — DS `<head>` shell (the two
  `/_ds/tokens.css` + `/_ds/fonts.css` `<link>`s + page-scoped `<style>` over `var(--*)`), `.chrome`
  masthead, a `.ledger` card with definition rows (Domain, Origin, Status badge, honest Coverage,
  Observed size) + a relative link to `/{{.Origin}}/`. Unquoted `data-status=` selectors (CSS-literal
  trap), `--status-error-bg` used with the literal fallback.
- `internal/dossier/handler_test.go` (new test): HTTP-seam asserts — 200 text/html, DS-shell wiring,
  domain through the `hubStatusBadge` partial, coverage honesty (covered → `size 42 at 2023-11-14…Z`;
  uncovered → "no coverage yet"), no `<table>`/CDN, non-GET → 405, missing summary → 500, plus the
  `hubStatus`/`overlayStatus` table tests mirrored from dashboard.
- `cmd/iscc-monitor/main.go`: added `Domain string` to `hubRoute`, set it from `e.Domain` in
  `registerHubs`, mounted `dossier.Handler(st, r.HubID, m)` at `"/"+r.Domain` in `mirrorHandler`,
  updated the `buildMux`/`hubRoute`/`mirrorHandler` doc comments, added the `internal/dossier` import.
- `cmd/iscc-monitor/main_test.go`: set the now-required `Domain` on the three `hubRoute` fixtures (an
  empty Domain registered `mux.Handle("/")` and collided with the dashboard mount → panic), asserted
  `routes[i].Domain` in the registerHubs test, and added a `TestMirrorRouter` subtest proving the bare
  `/sb0.iscc.id` dossier resolves 200 text/html alongside the `/sb0.iscc.id/log/` subtree.
- `CLAUDE.md`: added the `GET /<domain>` route bullet after `GET /`.

**Verification:** `mise run check` → green (`go build`/`go vet`/`go test` all 19 packages `ok`);
`gofmt -l .` empty.
- 200 text/html + DS shell (`href="/_ds/tokens.css"`, `href="/_ds/fonts.css"`, `var(--font-sans)`,
  `var(--font-mono)`) — PASS.
- Domain rendered through `hubStatusBadge` (`class="hub-status-badge"` + label + silhouette) — PASS.
- Coverage honesty: covered hub `size 42 at 2023-11-14T22:13:20Z`; uncovered hub "no coverage yet" with
  no fabricated `row-mono">size` — PASS.
- In-memory overlay reaches the dossier (store-verified + live `unresolvable` → renders unresolvable,
  body carries NO `data-status="verified"`) — PASS.
- No `<table>`, no `http://`/`https://`/`cdn.`/`jsdelivr` in body — PASS.
- `POST /<domain>` → 405; missing store summary → 500 (real inconsistency, not 404) — PASS.
- Mutation-proven non-vacuous (reverted byte-identical, tree clean): broke `/_ds/fonts.css` href → FAIL;
  forced the coverage branch to `{{- if false}}` → FAIL.
- `go list -deps ./internal/store ./internal/badge | grep -E 'net/http|internal/dossier'` empty (leaves
  intact, no cycle). `cmd/iscc-monitor` test green (`/<domain>` AND `/<domain>/log/` both resolve).
- Oracle/conformance gate N/A (pure HTML render of persisted rows + in-memory overlay; no
  signature/RFC-6962/Merkle/did:web/fsck/proof path). go.mod/go.sum byte-identical; no schema change.

**Next:** The frozen **Exhibit** sub-step (the categorically-distinct, non-dismissable
violation-detail block, ADR-0006) is now the natural follow-on — it needs a NEW store read
(`store.ListViolations(hubID)` over the `violations` table; only `RecordViolation` exists today, no
read) plus the Exhibit markup in `dossier.html`. The dossier skeleton renders the `frozen` *status
badge* honestly via the overlay but deliberately fabricates no violation detail. After that, the
remaining M-UI SSR screens (paginated record list / single record / certificate-of-inclusion) — the
certificate re-engages the oracle gate (inclusion-proof path).

**Notes:**
- The `hubRoute.Domain` field is now load-bearing: an empty `Domain` makes `mirrorHandler` register
  `mux.Handle("/")`, which collides with the dashboard's `/` mount and panics `ServeMux.register`. The
  three `cmd/iscc-monitor` test fixtures had to set it (production `registerHubs` always does). Flagging
  so review knows the test-fixture change is required, not gratuitous.
- Deliberate duplication: `overlayStatus`/`hubStatus`/`coverageTime` are now a THIRD copy (dashboard,
  proofserve, dossier). Per `next.md` Not-In-Scope, consolidating into `internal/badge` is its own
  tracked `low` issue ("Hub-status overlay precedence is duplicated") — NOT done here to avoid touching
  unrelated packages. Review may want to bump that issue's weight now that it is 3x.
- Followed the CSS-literal trap (learnings/http-surface.md): used unquoted `[data-status=verified]`
  selectors so the negative `data-status="verified"` overlay assert stays honest, and used
  `var(--status-error-bg, rgba(245, 97, 105, 0.06))` with the literal fallback (token not in tokens.css).

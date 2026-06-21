## 2026-06-21 — Review of: Server-rendered dashboard at `GET /` listing every realm hub (status + coverage)

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** `advance` added a server-rendered HTML dashboard at the exact path `/` that lists every
followed hub with its store-provable glossary status (inactive / frozen / verified) and coverage window
(ADR-0001). The diff is scope-clean (`internal/store/hubs.go` `ListHubs` + a new `internal/dashboard`
leaf + the `/` mount in `main.go` + template/test/doc), keeps `internal/store` a leaf, and is golden +
mutation-proven at the HTTP seam. This closes the third of M3's four Verify criteria (M3 → 3/4); the
HTML log browser is the last.

**Verification:**
- [x] `mise run check` green — all 16 packages `ok` (build + vet + test).
- [x] `gofmt -l .` empty (clean).
- [x] `go test -count=1 ./internal/dashboard` PASS uncached (golden HTTP-seam + status-mapping table).
- [x] `go test -count=1 ./internal/store ./cmd/iscc-monitor` PASS uncached (ListHubs + mux wiring intact).
- [x] `go list -deps ./internal/store | grep -E 'net/http|internal/dashboard|internal/logclient'` empty
  — store stays a leaf. Dashboard own imports: `bytes embed html/template net/http internal/store`.
- [x] `git diff --stat HEAD~1..HEAD -- go.mod go.sum internal/store/schema.sql` empty — no schema/dep change.
- [x] HTTP seam on the fixture store: `GET /` → 200, `text/html; charset=utf-8`; body names both hubs'
  domain+origin, renders `verified` + `frozen` in the correct cells, shows `size 42 at <RFC3339>` for
  the verified hub and `no coverage yet` for the frozen (advance-follow-state-only) hub — ADR-0001
  coverage honesty observable. `POST /` → 405; `GET /unknown` → 404. (Confirmed by dumping the rendered
  HTML via a throwaway test, then removing it.)
- [x] `GOOS=js GOARCH=wasm go build ./internal/didweb` → OK (WASM verifier seam untouched).
- [x] Gate-integrity scan over all unpushed commits (`@{upstream}..HEAD`) — no `//nolint` / `t.Skip` /
  build-tag exclusion / swallowed-error / deleted-assertion. The lone `t.Skip`-pattern hit is the prior
  review's checklist text inside `handoff.md`, not code.
- [x] Mutation-proven non-vacuous (3 mutations, all reverted): (1) `hubStatus` frozen→verified →
  `TestHubStatusMapping/frozen_when_active` FAILS; (2) drop the `{{.Status}}` template cell → golden
  body assert FAILS; (3) drop `size {{.SinceSize}}` from the template → coverage-start assert FAILS.
- [x] `/`-mount collision verified with a standalone mux: a `/a/` subtree + `/` root resolves
  `/a`,`/a/`,`/a/x` to the subtree and only `/b` to root — so `/metrics`, `/healthz`, and each
  `/<domain>/log/` subtree all still win over the dashboard's `/` (most-specific match).
- [x] Oracle/conformance gate correctly N/A — pure HTML rendering of persisted store rows; no
  signature / RFC-6962 / Merkle / did:web / fsck / proof path.

**Issues found:** (none) — no defect; no issues.md change. The one open `low` notecheck item is unrelated.

**Codex second opinion:** Codex (gpt-5.5, xhigh) finished after ~3.5 min with one finding, triaged:
- **[P3] handler.go:69-74 — method-gate (405) runs before the exact-path guard (404), so `POST
  /unknown` returns 405 instead of 404 → DISMISSED (style nit, not a defect).** The handler follows the
  established codebase convention: `proofserve.Handler` (handler.go:76-77) and `healthz.Handler`
  (handler.go:45-46) BOTH method-gate first. Re-ordering only the dashboard would make it inconsistent
  with its sibling leaves. `POST /unknown` is an unspecified cross-product edge (both 404 and 405 are
  4xx client errors on a non-existent path); no `target.md` Verify criterion or ADR pins the ordering,
  and the realistic cases the spec named (`GET /unknown`→404, `POST /`→405) are both correct. Per the
  no-style-nits rule, logged here and not filed.

**Next:** The HTML **log browser** `GET /<domain>/log/` — the fourth and final M3 Verify criterion (M3 →
4/4). It mounts under the per-hub subtree (reached through `hubHandler` in `main.go`, a different mount
than `/`), so it belongs in `proofserve` or a new per-hub HTML leaf — NOT in `internal/dashboard`. After
M3 closes, the proof-bundle assembler (the authoritative client-verifies path) is the natural next arc.

**Notes:**
- **`inactive` status is currently unreachable through the public store API** (no `SetActive` writer;
  `UpsertHub` inserts the schema default `active=1`), so the golden HTTP-seam test cannot drive a hub to
  `inactive`. The advance covered it via a white-box table test on the package-private `hubStatus`
  (hence `package dashboard`, not `dashboard_test`). When a registry-deactivation writer lands, add an
  end-to-end inactive-render assertion through the public surface. Documented limitation, not a defect.
- **Status is the store-provable subset only** (inactive > frozen > verified), mirroring
  `proofserve.hubStatus` and extending it with `inactive`. The richer in-memory metrics statuses
  (`unverified`/`unresolvable`/`rotated`) are deliberately NOT threaded in — out of scope, keeps the
  page golden-testable on a fixture store. Threading `metrics.Registry` into the dashboard is later work.
- Learnings: new detail file `learnings/dashboard.md` (+ index pointer row) captures the
  `/`-mount-vs-exact-path-guard rule, the store-provable status subset, the inactive-unreachable caveat,
  and the coverage-honesty render. No promotion to the cross-cutting index (all package-local).

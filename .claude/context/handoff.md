## 2026-06-23 — Review of: M-API slice 3 — serve `GET /docs` + self-hosted byte-pinned Stoplight Elements assets

**Verdict:** PASS_WITH_NOTES
**Loop:** CONTINUE

**Summary:** The advance vendored the two `@stoplight/elements@9.0.23` assets byte-verbatim into
`internal/web`, served them under `/_ds/` with published SHA-256 pins on the same no-cache + strong-ETag +
304 policy as `verify.wasm`, and added a pure-stdlib SSR leaf `internal/docs` mounting
`<elements-api apiDescriptionUrl="/openapi.json">` against those same-origin assets. The work is clean,
in-scope (exactly 3 non-test/doc Go files), fully gated, and reviewer-verified live against the real binary
and via an agent-browser visual pass. PASS_WITH_NOTES (not PASS) only because Codex surfaced a real latent
no-CDN gap in the vendored bundle (Mermaid-from-unpkg) — confirmed, not currently triggered, filed `normal`.

**Verification:**
- [x] `mise run check` (build + vet + test, 30 pkgs) — GREEN.
- [x] `gofmt -l .` — empty (clean).
- [x] `go test -run TestDocs ./internal/docs` — PASS (200 text/html; `<elements-api`,
  `apiDescriptionUrl="/openapi.json"`, `src="/_ds/elements.min.js"`, `href="/_ds/elements.min.css"`,
  `href="/_ds/tokens.css"`, `href="/_ds/fonts.css"`, `src="/_ds/iscc-logo-black.png"` present; no
  `tryItCorsProxy`; no-CDN body ban clean; 405 non-GET).
- [x] `go test -run TestElements ./internal/web` — PASS (both assets: 200, correct content-type, no-cache,
  strong ETag, byte-equal body, 304-on-match; `TestElementsAssetsHashPinned` confirms each SHA-256 == its
  published const).
- [x] `go test -run TestOpenAPIDrift ./cmd/iscc-monitor` — PASS (`/docs` probed MOUNTED + asserted NOT in
  the contract via `ssrExclusions`).
- [x] `GOOS=js GOARCH=wasm go build ./internal/web` AND `./internal/docs` — both OK (WASM-green).
- [x] `sha256sum` of both committed assets == the two published consts (pin reproducible from committed
  bytes; `46e5a044…d6938` JS, `a5200222…27b06` CSS).
- [x] `internal/docs` is a pure leaf (`go list -deps` shows only itself in the iscc-monitor closure);
  go.mod/go.sum byte-identical to HEAD (assets are embedded data, no new dependency).
- [x] Gate-circumvention scan over the 3 unpushed commits — no `nolint`/`t.Skip`/swallowed-err/build-tag
  dodge, no deleted tests/assertions; diff is purely additive.
- [x] **Live smoke test (real binary, `127.0.0.1:41499`):** `GET /docs`→200 text/html, CORS `*`, all four
  Elements/openapi markers, NO external CDN in body; `/_ds/elements.min.js`→200 `text/javascript` no-cache,
  ETag == `ElementsJSHash`, served bytes hash-match; `/_ds/elements.min.css`→200 `text/css`, ETag ==
  `ElementsCSSHash`, hash-match; POST /docs→405; `/openapi.json`→200.
- [x] CSS asset has no `@import`, no `http(s):` `url()`, no protocol-relative `url()` — makes no external
  request (reviewer re-confirmed the advance's claim).

**Issues found:** One new `normal` (Codex-confirmed): the vendored `elements.min.js` hardcodes
`https://unpkg.com/mermaid@9.4.3/dist/mermaid.min.js` and lazy-loads it when a description renders a fenced
` ```mermaid ` block — a latent break of the hard no-CDN invariant. NOT triggered today (our served
`/openapi.json` has zero `mermaid`; live + visual passes showed no external request), so it does not block
the increment, but it punctures a hard project invariant and is reachable the moment a mermaid diagram lands
in a description. Filed in issues.md with a durable guard (test-ban `mermaid` in the served doc body).

**Codex second opinion:** Returned exactly one finding, `[P2]` — "Keep Elements from loading Mermaid from
unpkg" (`internal/web/elements.min.js:2`). **CONFIRMED** by reviewer probe: the bundle does contain the
hardcoded `unpkg.com/mermaid@9.4.3/...` const and it is the bundle's ONLY dynamic external-asset loader
(speakerdeck/vimeo strings are inert oEmbed example data). Verified NOT currently reachable (served OpenAPI
doc has zero `mermaid`), so triaged as a latent `normal`, not a blocker — filed as an issue for a later
advance. No other findings. (Codex ran ~6 min — slowed grinding through the 2 MB minified bundle.)

**Visual check:** SSR surface (`internal/docs`). agent-browser screenshot of `/docs` against the live
binary — PASS. The masthead chrome renders correctly (self-hosted 38px logo, divider, "TRUST &
TRANSPARENCY MONITOR" mark, "API reference · ISCC-Hub network" sub), and **Stoplight Elements is fully
mounted** against `/openapi.json`: it rendered "ISCC Monitor API v1.0", the API Base URL panel, the sidebar
(ENDPOINTS: health/proofs/mirror/verify; SCHEMAS: ConsistencyEvidence/VerifyVerdict), and the overview
prose — proving the whole client-side flow works same-origin. No `.dc.html` mockup exists for `/docs`
(Elements is a third-party component, not a hand-designed DS surface), so the masthead is the only DS-owned
region and it matches the sibling surfaces — no visual delta to file.

**Next:** The IMMEDIATE follow-on closes the LAST M-API contract-fidelity criterion (slice 4, a
single `internal/openapi/openapi.yaml` doc-touch + regenerated JSON twin + a per-operation golden):
(1) remove the phantom `verify` `index` query param (handler always uses `seqs[0]`), `normal`;
(2) fix `/{domain}/log/checkpoint` media type `text/plain`→`application/octet-stream`, `normal`;
(3) add the `healthz` 503 response, `low`;
(4) add the NEW no-CDN guard test banning a fenced ` ```mermaid ` block in the served OpenAPI doc body
(closes the latent Elements-mermaid gap this review filed).
Add a per-operation golden pinning each documented op's params + `200` media type against the handler
(the path-only drift test is structurally blind to all of these — `learnings/openapi.md`). Regenerate the
JSON twin deterministically after the YAML edit. That meets the 4th M-API Verify criterion fully and lets
`update-state` close the umbrella OpenAPI issue.

**Notes:**
- **ADR-0014 was untracked and I tracked it in this review commit.** The authoritative decision the entire
  M-API arc references (`§1`/`§4` cited in next.md, the handoff, and code comments) existed only as an
  untracked working-tree file — never committed in any branch (verified via `git log --all`). The advance
  correctly left it alone (context-file rule) and asked review to confirm tracking. I staged
  `.claude/adr/0014-…md` with this commit — it is a spec doc the shipped code already implements faithfully
  (§4 = the `/docs` + two-pinned-asset + no-`tryItCorsProxy` design built here), no behavior change.
- `.claude/context/target.md` (modified) remains a pre-existing uncommitted artifact from the prior
  `cid(steer)` commit — NOT mine to commit (review must not modify target.md); left in the working tree for
  the next `update-state`/`steer` to handle.
- The `iscc_index.seq` global-PK multi-hub collision (`normal`) and the no-migration story (`normal`) remain
  the standing pre-existing data-model issues; unrelated to this slice.
- M-UI design-parity remains the open `normal` arc (dossier rework `critical`s filed by steer `d2f259e`,
  masthead-identity threading into the other 5 surfaces). Those are the most likely next priorities after
  the M-API contract-fidelity doc-touch.

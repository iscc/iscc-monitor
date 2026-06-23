## 2026-06-23 — M-API slice 1: embed + serve the OpenAPI 3.1 contract at `/openapi.json` + `/openapi.yaml`, with a route↔spec drift test

**Done:** Added the `internal/openapi` leaf — a hand-authored OpenAPI 3.1 document (YAML
source-of-truth + a byte-identical JSON twin, both `go:embed`-ed) served byte-verbatim at
`GET /openapi.json` (`application/json`) and `GET /openapi.yaml` (`application/yaml`) with the
project's `no-cache` + strong-ETag + `If-None-Match`→304 cache shape, plus `Handler()`,
`JSON`/`YAML` vars, `JSONPath`/`YAMLPath` consts, and `DeclaredPaths()`. Wired both exact mounts into
`buildMux` (riding the existing `corsmw` `*` wrap) and the two names into `reservedMountNames`. The
document covers only the machine surface (the proof/artifact/Prometheus routes + the `.bundle`); HTML
SSR surfaces are excluded; `verify-for-me` is flagged in-band as the weaker tier-1 path. A non-vacuous
drift test reconciles the document against the real mux.

**Files changed:**
- `internal/openapi/openapi.yaml` (new): hand-authored OpenAPI 3.1 source-of-truth, 13 machine paths.
- `internal/openapi/openapi.json` (new): the byte-identical JSON twin (regenerated from the YAML via
  `yaml.v3` → deterministic JSON; verified byte-equal to a fresh regeneration).
- `internal/openapi/openapi.go` (new): `package openapi` — embeds + `Handler()` + `DeclaredPaths()` +
  `writeDoc` (the `internal/web.writeAsset` cache shape). Pure stdlib + `gopkg.in/yaml.v3` leaf.
- `internal/openapi/openapi_test.go` (new): HTTP-seam tests — JSON/YAML verbatim+content-type, 405,
  404, 304, JSON/YAML path-set agreement, valid-3.1 skeleton, verify-for-me-weaker golden.
- `cmd/iscc-monitor/main.go`: import `internal/openapi`; mount `/openapi.json` + `/openapi.yaml` in
  `buildMux`; add both to `reservedMountNames` (derived from the consts); docstring updates.
- `cmd/iscc-monitor/openapi_drift_test.go` (new): the load-bearing drift test (byte-verbatim+CORS,
  declared⇔mounted set-equality, concrete mount probes, SSR exclusion check).
- `CLAUDE.md`: documented `GET /openapi.json` + `GET /openapi.yaml` in the dev-instance endpoint list.

**Verification:** `mise run check` → GREEN (build + vet + test, all 28 pkgs). `gofmt -l .` empty.
Per-criterion:
- [x] `go test -run TestOpenAPI ./internal/openapi` — PASS (200/content-type/byte-equal for JSON+YAML,
  405, 404, 304-echo-ETag, JSON⇔YAML path-set identical, valid 3.1 skeleton, verify-for-me weaker).
- [x] `go test -run TestOpenAPIDrift ./cmd/iscc-monitor` — PASS; `TestOpenAPIServedVerbatimWithCORS`
  asserts the served `/openapi.json` is byte-equal to `openapi.JSON` and carries
  `Access-Control-Allow-Origin: *` through the full corsmw-wrapped mux.
- [x] `/openapi.json` parses as `openapi: 3.1.0` with a non-empty 13-path `paths` object.
- [x] `grep -c "openapi" CLAUDE.md` = 1 (> 0).
- [x] `go.mod`/`go.sum` byte-identical to HEAD (`yaml.v3` already a dep); `internal/openapi` dep
  closure has no `internal/store|metrics|logclient` (pure leaf).

**Non-vacuity (4 mutations, all confirmed FAIL then reverted):**
1. Drop `/version` from both embedded docs → `TestOpenAPIDrift` set-equality FAILS.
2. Drop `/version` from the JSON only → `TestOpenAPIDocsAgree` FAILS (the two artifacts can't drift).
3. Add an undocumented machine route to the ground-truth set (mux mounts it, doc lacks it) →
   set-equality FAILS.
4. Declare + ground-truth a path the mux does NOT mount → the (b) concrete probe FAILS (404).

**Next:** M-API slice 2 should be the human-facing **`GET /docs` + Stoplight Elements** assets
(ADR-0014 §4): self-host + byte-pin the Elements web-component JS *and* its stylesheet under `/_ds/`
with published `internal/web` hash constants (the fragile-pin discipline, like `verify.wasm`), render a
no-CDN DS shell mounting `<elements-api apiDescriptionUrl="/openapi.json">`, leave `tryItCorsProxy`
unset. That closes the last two M-API Verify criteria. It is its own ≤3-file slice and intentionally
out of scope here. The remaining standing `normal`s (dossier §3 size/time decouple, §1 wording) are
unchanged.

**Notes:**
- **Oracle/conformance gate N/A (correctly).** This serves committed bytes and reconciles route
  strings — it touches no signature / RFC-6962 / Merkle / did:web / fsck / proof path. The package
  docstring states this explicitly, matching `internal/web`'s "oracle gate N/A" note.
- **JSON twin provenance:** I generated `openapi.json` from `openapi.yaml` with a throwaway `yaml.v3`
  converter run *inside the module* (created under `internal/openapi/tmpconv/`, run, then deleted — it
  is NOT committed and `git status` confirms no leak). Re-confirmed at the end that the committed JSON
  is byte-identical to a fresh regeneration from the current YAML, so the two artifacts agree and the
  agreement test is honest. If the YAML is hand-edited later, regenerate the JSON the same way (or by
  any equivalent deterministic YAML→JSON) — `TestOpenAPIDocsAgree` only gates the path SET, while the
  drift test gates paths-vs-mux; neither gates the full body, so a careless one-sided body edit to
  descriptions would not be caught (acceptable — descriptions are prose, not contract shape).
- **Drift-test mount probe is data-dependent by design.** It seeds a fully-verified one-hub mirror
  (tile + entry bundle + iscc_index + checkpoint + OTS + a one-slot Hub-List for the `.bundle`) so
  EVERY documented machine route returns a non-404 success; an unmounted path falls through to
  tilesserve/the dashboard and 404s, making non-404 the unambiguous mount signal. The canonical
  partial-tile probe path is `/<domain>/log/tile/0/000.p/5` (a 5-leaf partial), NOT `tile/0/000`
  (which 404s — there is no full tile). Reuses the existing `frameEntryBundle` test helper.
- **`reservedMountNames` defense-in-depth:** added `strings.TrimPrefix(openapi.JSONPath/YAMLPath, "/")`
  (i.e. `openapi.json` / `openapi.yaml`) so a realm domain literally named `openapi.json` can't mount a
  dossier colliding with the exact route — same pattern as the existing `metrics`/`healthz`/`version`
  entries.
- **Working-tree steer files left unstaged (as the prior review flagged):** `target.md` (M-API
  milestone) and `.claude/adr/0014-...md` are an in-flight steer workstream I did not author and did
  NOT stage into this commit — only my implementation/test files + `CLAUDE.md` + `handoff.md` are
  committed.

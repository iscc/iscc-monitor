# Next Work Package

## Step: Frozen Exhibit on the hub dossier (`store.ListViolations` + non-dismissable markup)

## Advances
Milestone **M-UI — Evidence Ledger frontend** (ADR-0010). Closes the next open M-UI Verify clause —
quoting `target.md`:

> `HubStatusBadge` renders … with `frozen` rendered as the categorically-distinct **Exhibit**
> (violation kind + detected-at + "do not trust new state", non-dismissable), visibly different markup
> from the `unresolvable`/`unverified` caution

The frozen-status five-status badge already renders, but the dossier does **not** yet surface the
**Exhibit** (the violation `kind` + `detected_at` + the non-dismissable "do not trust new state"
panel). This step adds the store read + the dossier markup that closes that clause. The handoff
`**Next:**` from the last `review` PASS names exactly this sub-step.

## Goal
Render a categorically-distinct, non-dismissable **Exhibit** panel on the per-hub dossier
(`GET /<domain>`) for a frozen hub, listing each self-consistency violation's `kind` + `detected_at`
(ADR-0006 irreplaceable evidence), backed by a new leaf read `store.ListViolations(hubID)` over the
`violations` table (today only `RecordViolation` writes it).

## Scope
- **Create**: (none)
- **Modify** (≤3 non-test/doc files):
  - `/workspace/iscc-monitor/internal/store/checkpoints.go` — add
    `ListViolations(ctx context.Context, hubID int64) ([]Violation, error)`, a pure read mirroring the
    existing `ListHubs`/`SeqsForISCCID` query shape. Reuse the existing `Violation` struct
    (`HubID, Kind, RawA, RawB, ProofJSON, DetectedAt`); read back at minimum `hub_id, kind,
    detected_at` (the markup needs `kind` + `detected_at`; `raw_a/raw_b/proof_json` are not rendered
    here — read them or leave them zero, your call, but do not require them). Order newest-first
    (`ORDER BY detected_at DESC, id DESC`). Read `detected_at` through `sql.NullInt64` → zero
    `time.Time` when NULL (the `unixOrNil` write inverse).
  - `/workspace/iscc-monitor/internal/dossier/handler.go` — when the resolved status is `frozen`, call
    `st.ListViolations(r.Context(), summary.HubID)` and thread the rows into the view-model
    (`dossierData`) as a slice of a small render struct (e.g. `Kind string`, `DetectedAt string`
    RFC-3339-or-empty, reusing the `coverageTime` formatting idiom). Add a `Frozen bool` +
    `Violations []…` field to `dossierData`. Do NOT call `ListViolations` on non-frozen hubs (keep the
    read off the hot path).
  - `/workspace/iscc-monitor/internal/dossier/dossier.html` — add the Exhibit block, rendered only
    `{{if .Frozen}}`, as markup **visibly distinct** from the badge caution: a panel with a clear "do
    not trust new state" notice and a list of `{{.Kind}}` + `{{.DetectedAt}}` rows. No dismiss/close
    control (non-dismissable: no JS, no `hidden`, no collapse). Style it with a page-scoped rule over
    DS tokens (you may reuse the `.ledger[data-status=frozen]` tint conventions); no external/CDN URL.
- **Modify (tests — uncounted)**:
  - `/workspace/iscc-monitor/internal/store/checkpoints_test.go` — `TestListViolations`.
  - `/workspace/iscc-monitor/internal/dossier/handler_test.go` — a frozen-dossier Exhibit assertion.
- **Reference**:
  - `/workspace/iscc-monitor/.claude/context/learnings/store.md` — the `RecordViolation`/`Freeze` seam
    + leaf-purity rules (no `net/http`/`internal/logclient` in the store closure; `unixOrNil`/NULL-time
    convention).
  - `/workspace/iscc-monitor/.claude/context/learnings/dashboard.md` +
    `/workspace/iscc-monitor/.claude/context/learnings/badge.md` — the coverage-honesty render pattern
    and the badge-partial composition the dossier already uses.
  - `/workspace/iscc-monitor/internal/store/checkpoints.go:259-292` (the `Violation` struct +
    `RecordViolation`) and `/workspace/iscc-monitor/internal/store/hubs.go` (the `ListHubs` query/scan
    shape to mirror).
  - `/workspace/iscc-monitor/internal/store/schema.sql:58-69` (the `violations` table columns).
  - `/workspace/iscc-monitor/.claude/adr/0006-*.md` (kinds `fork|shrink|equivocation`; permanent
    evidence, no auto-unfreeze).

## Not In Scope
- The paginated **record list**, the **single-record page**, the **certificate of inclusion**, and the
  **proof-bundle assembler** — later M-UI sub-steps (the certificate is the one that re-engages the
  oracle/conformance gate; keep that pressure off this pure-render step).
- Separate **Bitcoin-anchor vs comparison-anchor** panels — a distinct M-UI clause.
- Consolidating the triplicated `overlayStatus`/`hubStatus` (the `low` issue) — leave the third copy
  as-is; do not refactor `internal/badge`.
- Rendering `raw_a`/`raw_b`/`proof_json` bytes or a download — the Exhibit shows `kind` + `detected_at`
  only; the raw evidence bytes belong with the future proof-bundle surface.
- Any **unfreeze** affordance — ADR-0006 forbids auto-unfreeze; the panel is informational and
  non-dismissable.
- Adding `ListViolations` callers in `internal/dashboard` / `internal/proofserve` / the realm index —
  this step wires it into the dossier only.

## Implementation Notes
- **`store` stays a leaf.** `ListViolations` is a plain read on `s.db` using only `context`,
  `database/sql`, `fmt`, `time` (all already imported in `checkpoints.go`). Do not introduce
  `net/http` or any `internal/*` import; verify with
  `go list -deps ./internal/store | grep -E 'net/http|internal/'` staying empty (store.md rule).
- **NULL-time symmetry.** `detected_at` is written via `unixOrNil` (zero → NULL). Read it back through
  `sql.NullInt64`; `!Valid` → zero `time.Time`. In the dossier, format with the same
  `Since.UTC().Format("2006-01-02T15:04:05Z")` idiom `coverageTime` uses, and render an explicit
  empty-string fallback (the template shows "time unknown" or simply omits the time) rather than a
  fabricated epoch — coverage-honesty discipline applies to evidence timestamps too.
- **Frozen-only read.** Gate the `ListViolations` call on the resolved status being `frozen`. `frozen`
  is store-provable and the overlay never downgrades it, so `hubStatus(summary) == "frozen"` is a safe
  gate (the overlay only ever promotes `verified`). Non-frozen dossiers must issue no extra query.
- **Exhibit is markup-distinct, non-dismissable.** ADR-0010 requires the frozen Exhibit be *visibly
  different markup* from the `unresolvable`/`unverified` caution — not just a recolored badge. Render a
  dedicated panel (its own class, a heading like "Exhibit — self-consistency violation", the literal
  copy "do not trust new state", and the per-violation `kind` + `detected_at` list). No `<button>`,
  no JS toggle, no `hidden` — content present in the served HTML (no-JS baseline).
- **No-JS / no-CDN baseline** (target.md M-UI Verify): the Exhibit content must be in the served HTML,
  style only through DS `var(--*)` tokens, and the body must carry no `http://`/`https://`/`cdn.`/
  `jsdelivr` (the existing dossier test already bans these; keep it true).
- **Empty-violations edge case:** a hub can be `frozen` with violation rows present (the freeze path
  always writes a `RecordViolation` before `Freeze`), but defend against zero rows — render the panel
  header + "do not trust new state" even if the list is empty, never a broken/empty `{{range}}`.
- **Relevant learnings (Correctness):** "A self-consistency violation freezes, never crashes
  (ADR-0006). … Persist both raw checkpoints + proof permanently … no auto-unfreeze." The Exhibit is
  the read-side surfacing of that permanent evidence; it must never imply the freeze can be cleared.
- Render into the existing `bytes.Buffer` before `WriteHeader(200)` (the dossier's broken-client
  convention) so a `ListViolations` error is a 500 *before* any 200 is committed.

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...` all pass, `gofmt -l .`
  empty).
- `go test -run TestListViolations ./internal/store` passes — a new test seeds two violations
  (`fork`, then `shrink` with a later `DetectedAt`) via `RecordViolation`, then asserts
  `ListViolations(hubID)` returns both newest-first with the correct `Kind` + non-zero `DetectedAt`,
  and returns an empty result (nil/empty slice, no error) for a hub with none.
- `go test -run TestDossier ./internal/dossier` passes — a new test freezes a hub
  (`RecordViolation` + `Freeze`), renders `GET /<domain>`, and asserts the body contains the Exhibit
  panel markup (the distinct class/heading + "do not trust new state" + the violation `kind` +
  `detected_at`) AND the `frozen` badge silhouette marker (`M8.2 3.3h7.6`); the existing
  verified-dossier test still passes and the verified dossier renders **no** Exhibit panel (assert the
  Exhibit class/"do not trust new state" copy is absent for a non-frozen hub).
- `go list -deps ./internal/store | grep -E 'net/http|internal/'` stays empty (store leaf purity).
- `GOOS=js GOARCH=wasm go build ./internal/badge` still green (badge untouched; sanity).

## Done When
`store.ListViolations` reads the `violations` table as a leaf, the frozen dossier renders the
categorically-distinct non-dismissable Exhibit (violation kind + detected-at + "do not trust new
state") while non-frozen dossiers do not, and all Verification checks pass with `mise run check` green.

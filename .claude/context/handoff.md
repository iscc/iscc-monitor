## 2026-06-23 — Hub dossier §5 honest observation log (increment 2a — checkpoint-event log)

**Done:** Filled §5 of the served hub dossier (`GET /<domain>`) with an honest observation log
derived from recorded `checkpoints` rows: a new `store.ListCheckpoints` leaf read (newest-first by
`observed_at`), a view-layer derivation that emits one size-transition line per consecutive recorded
pair (newest-first) plus the oldest checkpoint as a singleton, an "anchored · block N" line only for a
confirmed anchor with a real height, and a "froze hub (split view)" pointer per recorded violation on
the frozen path. It never emits a synthesized per-poll "consistent" verdict (the recurring SSR-honesty
trap). The richer frozen Exhibit and the §3/§1 honesty `normal`s remain the sibling follow-on (not in
scope).

**Files changed:**
- `internal/store/checkpoints.go`: added `CheckpointSummary` + `ListCheckpoints(ctx, hubID, n)` — a
  pure leaf read of `tree_size, observed_at` newest-first (`ORDER BY observed_at DESC, id DESC LIMIT ?`,
  NULL-safe `sql.NullInt64` → zero `time.Time`, absent hub → empty slice + nil err), mirroring
  `ListViolations`'s shape. Deliberately `observed_at DESC` (chronological log), distinct from `ListHubs`'s
  §3 `tree_size DESC` subselect.
- `internal/dossier/handler.go`: added `observationRow{Line, Tone}` + `Observations` field on
  `dossierData`; the handler reads `ListCheckpoints(…, observationCap=8)` before `tmpl.Execute` (read
  error → 500 like the existing reads); `observationRows`/`freezeLine`/`withObservedTime` derive §5 in
  the view layer (size transitions + confirmed-anchor line + frozen pointer, honesty-gated, no per-poll
  verdict). `buildData` threads the checkpoints through.
- `internal/dossier/dossier.html`: replaced the §5 `obs-empty` placeholder with
  `{{if .Observations}}{{range}}<div class="obs-line" data-tone>…{{else}}<p class="obs-empty">honest empty</p>{{end}}`;
  added `.obs-line` (+ decorative `[data-tone=freeze]`) styles. No `<script>`, no external/CDN URL.
- Tests (not counted): `internal/store/checkpoints_test.go` (`TestListCheckpoints`,
  `TestListCheckpointsNullObservedAt`); `internal/dossier/handler_test.go` (`TestDossierObservationLog`,
  `TestDossierObservationLogEmpty`, `TestDossierObservationLogAnchorAndFreeze` + a `multiCheckpointHub`
  helper).

**Verification:** `mise run check` → green (build + vet + all 28 pkgs `ok`; `gofmt -l .` empty).
- `go test -run TestListCheckpoints ./internal/store` — PASS (newest-first `observed_at DESC` order, `n`
  cap, NULL `observed_at` → zero time, absent hub → empty + nil err; insertion order ≠ result order to
  prove ORDER BY drives it).
- `go test -run TestDossier ./internal/dossier` — PASS, incl. the new §5 cases: a ≥2-checkpoint hub
  renders a transition line per consecutive pair in newest-first order with each value traceable to a
  fixture row; a ≤1-checkpoint hub renders the honest empty state; §5 contains NO
  "consistent"/"consistency PASSED"/"verified-poll" substring; confirmed-anchor renders "block 869440"
  (never "block 0") and the frozen pointer maps fork → "split view".
- Mutation checks (non-vacuous, reviewer-runnable): dropping the size-transition append →
  `TestDossierObservationLog` FAILS; reverting `ORDER BY observed_at DESC` → `ASC` → `TestListCheckpoints`
  order assertions AND the dossier order assertion FAIL. Both restored.
- Store stays a leaf: `go list -deps ./internal/store | grep '^net/http$'` empty.

**Next:** The sibling follow-on of the same `critical` (increment 2b): the richer frozen Exhibit
("Tree size before → Then presented" + a stable evidence ref from `Violation.RawA`/`RawB`) — which needs
an *unverified* tree-size parse of the two stored checkpoint blobs (no exported pure parser exists today;
`parseCheckpointBody` is unexported and verifies the signature first), plus folding in the §3 frozen
size/time-decouple `normal` (select `observed_at` for the row whose `tree_size = f.last_size`) and the §1
"resolved"-vs-unresolvable `normal`.

**Notes:**
- Honesty/empty-state decision: a LONE checkpoint surfaces NO §5 line on its own (a single checkpoint is
  not a transition); the singleton "size N observed" tail only renders when there were ≥2 checkpoints.
  This is the literal reading of next.md's "≤1 checkpoint and no anchor → honest empty state" — it
  corrected an initial draft where `coveredHub`'s single seeded checkpoint produced a spurious
  "size 42 observed" line and failed `TestDossierObservationLogEmpty`. (A lone checkpoint with a confirmed
  anchor or a freeze still renders those real event lines — only the bare singleton is suppressed.)
- The freeze pointer reads ONLY the `violations` slice the handler already fetches on the frozen path
  (gated on `s.Frozen`); no second `ListViolations` read on the non-frozen path, per next.md.
- The mock's illustrative `observations` lines (`.dc.html` 118-122) carry "consistency FAILED" /
  "consistent" — those are ILLUSTRATIVE, NOT a parity requirement, and are deliberately NOT emitted (they
  would assert un-run per-poll checks). Flagged here as the design deviation next.md asked for.
- `data-tone` is decorative (grayscale-safe, ADR-0010 inv.4); the `Line` carries the meaning. `Tone` is
  only "normal" or "freeze" today.
- Oracle/conformance gate N/A — pure HTML render of persisted `checkpoints` rows + an in-memory overlay;
  touches no signature/RFC-6962/Merkle/did:web/fsck/proof path. `schema.sql`, `go.mod`, `go.sum`
  byte-identical. No visual pass run by the implementer; the SSR §5 region changed, so `review` may want a
  live look (the rendered §5 markup is in the test bodies).

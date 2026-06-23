# Next Work Package

## Step: Hub dossier §5 honest observation log (increment 2a — checkpoint-event log)

## Advances
The lone open `critical` — **"Hub dossier §5 honest observation log + richer frozen Exhibit (size
before → presented, evidence ref) — increment 2 of 2"** (`issues.md`) — and the target.md M-UI
dossier named-region bar:

> **§5 observation log** — rendering **only events the monitor actually recorded** (checkpoint size
> transitions, freezes, anchor confirmations), **never a synthesized per-poll "consistent" verdict**
> (the follower logs no such per-poll check, so emitting one would assert an un-run verification)

Increment 2 (the critical) is too large for one ≤3-file step — it bundles (a) a brand-new
`ListCheckpoints` store read + a §5 view derivation, (b) the richer frozen Exhibit ("size before →
presented" + evidence ref) which needs an *unverified* tree-size parse of `Violation.RawA`/`RawB`, and
(c) folding in the §3/§1 honesty `normal`s. This step lands **half (a)** — the §5 observation log,
skeleton-first: the new leaf read + the honest size-transition / anchor-confirmation log lines. The
Exhibit detail and the §3/§1 fixes are the sibling follow-on (`## Not In Scope`), continuing the same arc.

## Goal
Fill §5 of the served hub dossier (`GET /<domain>`) with an honest observation log derived from
recorded checkpoint rows — newest-first size transitions plus any Bitcoin-anchor confirmation — so the
mockup's §5 region renders real evidence instead of today's "not yet surfaced" placeholder, never
emitting a per-poll "consistent" line the follower never recorded.

## Scope
- **Create**: (none)
- **Modify** (3 non-test/doc files):
  - `internal/store/checkpoints.go` — add a `ListCheckpoints(ctx, hubID, n)` leaf read: newest-first
    over `checkpoints` carrying `tree_size` + `observed_at`, capped at `n`. Pure leaf (database/sql +
    stdlib only), mirroring the existing `ListViolations` read shape (lines 294-332).
  - `internal/dossier/handler.go` — read `ListCheckpoints` (small cap, e.g. 8) for the dossier hub and
    derive the §5 observation rows in the view layer (size transitions + anchor confirmation only); add
    an `Observations []observationRow` field to `dossierData`.
  - `internal/dossier/dossier.html` — replace the §5 `obs-empty` placeholder (lines 517-520) with a
    `{{range .Observations}}` list, keeping an honest empty state when there are ≤1 checkpoints / no
    events.
- **Tests (not counted)**: `internal/store/checkpoints_test.go`, `internal/dossier/handler_test.go`.
  No CLAUDE.md/README change — the `GET /<domain>` URL contract is unchanged (it already documents the
  dossier + its log link).
- **Reference** (read before implementing):
  - `.claude/context/learnings/dossier.md` — the numbered-layout rules + the §5/Exhibit-is-increment-2
    note + the binding "no synthesized per-poll consistent line" trap.
  - `.claude/context/learnings/store.md` — single-writer SQLite leaf; the read-only-projection discipline
    (store keeps no net/http/web dep).
  - `.claude/design/ISCC Monitor - Hub Dossier.dc.html` lines 94-99 (the §5 region) + 118-122 (the
    mock's illustrative `observations` lines — the mock's "consistent" lines are ILLUSTRATIVE, NOT a
    parity requirement; flag the deviation).
  - `internal/store/checkpoints.go` `ListViolations` (lines 294-332) — the exact leaf-read shape to
    mirror (NULL-safe `sql.NullInt64` for the unix-seconds time, newest-first ORDER BY,
    absent-row-is-not-error).
  - `internal/dossier/handler.go` `violationRows` (337-350) + `observedTime` (300-305) + `coverageDays`
    (311-320) — the view-layer derivation + honest-"" idioms.

## Not In Scope
- **The richer frozen Exhibit** ("Tree size before → Then presented" + a stable evidence ref from
  `Violation.RawA`/`RawB`). That needs an *unverified* tree-size parse of the two stored checkpoint
  blobs — there is NO exported pure parser for that today (`parseCheckpointBody` is unexported and
  verifies the signature first; `KeyIDFromCheckpoint` recovers only name+keyID, not tree size). It is
  the sibling follow-on sub-step of the same `critical`; leave the Exhibit exactly as increment 1 shipped.
- **The §3 frozen size/time-decouple `normal`** and **the §1 "resolved"-vs-unresolvable `normal`** —
  fold those into the Exhibit/§3-rework sub-step, not here (this step touches neither §3's size/time
  binding nor §1's wording).
- **Synthesizing per-poll "consistent" / "consistency PASSED" lines** — the recurring SSR-honesty trap:
  the follower records no per-poll verdict, so emitting one asserts an un-run check. Render ONLY size
  transitions between consecutive recorded checkpoints and anchor confirmations.
- No new metrics, no `schema.sql` change, no migration, no consolidation of the 3× overlay/identity dups.

## Implementation Notes
- **`ListCheckpoints` leaf (store):** mirror `ListViolations` verbatim in shape —
  `func (s *Store) ListCheckpoints(ctx context.Context, hubID int64, n int) ([]CheckpointSummary, error)`
  selecting `tree_size, observed_at FROM checkpoints WHERE hub_id = ? ORDER BY observed_at DESC, id DESC
  LIMIT ?`. Read `observed_at` through `sql.NullInt64` (the `unixOrNil` write inverse) → zero `time.Time`
  on NULL, exactly as `ListViolations` does for `detected_at`. Return a small new struct (e.g.
  `CheckpointSummary{TreeSize uint64; ObservedAt time.Time}`) — plain Go types, store stays a leaf. An
  absent hub returns an empty slice + nil error (the absent-row convention). Note the §3 subselect in
  `hubs.go` orders by `tree_size DESC`; here order by **`observed_at DESC`** (chronological log) with an
  `id DESC` tie-break so a NULL/equal `observed_at` stays deterministic.
- **§5 derivation (dossier view layer):** an `observationRow{Line string; Tone string}` — `Tone` is the
  decorative mock keyword (keep it optional / grayscale-safe; the `Line` carries the meaning). Walk the
  newest-first checkpoints and emit one line per **size transition** between consecutive recorded sizes
  (the newest pair → "size <prev> → <new>", older singletons → "size <n> observed"), phrased honestly,
  never "consistent". Append an anchor-confirmation line ONLY when the hub is confirmed
  (`s.Anchor == store.OTSStatusConfirmed`, already on `HubSummary`) gated on a non-zero height
  (`HasAnchorHeight`), e.g. "anchored · block <height>" — never "block 0". With ≤1 checkpoint and no
  anchor, render the honest empty state (keep an `obs-empty`-style line, no fabricated history).
- **Honesty rule (load-bearing — learnings always-loaded SSR-honesty rule + the dossier "no synthesized
  per-poll consistent line" note):** every §5 line must be traceable to a recorded `checkpoints` row or a
  confirmed `ots` row. No "consistency PASSED", no per-poll "consistent", no fabricated timestamp — when a
  checkpoint's `observed_at` is NULL, render the size without a time (mirror `observedTime`'s "" idiom),
  never a fake instant.
- **Optional freeze line:** a frozen hub already gets the loud Exhibit above the sections, so a §5 freeze
  line is optional polish. Include it ONLY if it stays a pure view-layer derivation off data the handler
  already fetches — the frozen path already calls `ListViolations`, so a "froze hub (<kind→split view>)"
  line off that slice is acceptable; do NOT add a second `ListViolations` read on the non-frozen path.
- **Template:** replace the single `<p class="obs-empty">` with
  `{{if .Observations}}{{range .Observations}}<div class="obs-line">{{.Line}}</div>{{end}}{{else}}<p
  class="obs-empty">… honest empty …</p>{{end}}`. Keep no-JS / no-CDN (no `<script>`, no external URL) —
  the existing golden bans assert this; the new markup must not trip them. `html/template` auto-escapes
  the rendered lines — keep it (NOT `text/template`).
- **Buffer-then-200 stays intact** — the handler already renders into a `bytes.Buffer` before writing
  200; the extra `ListCheckpoints` read happens before `tmpl.Execute`, and a read error maps to 500 like
  the existing `ListHubs`/`ListViolations` reads.
- **Oracle/conformance gate N/A:** pure HTML render of persisted `checkpoints` rows + an in-memory
  overlay; touches no signature / RFC-6962 / Merkle / did:web / fsck / proof path. `schema.sql`,
  `go.mod`, `go.sum` stay byte-identical.

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...`, `gofmt -l .` empty).
- `go test -count=1 -run TestListCheckpoints ./internal/store` passes — asserts newest-first
  (`observed_at DESC`) order, the `n` cap, a NULL `observed_at` → zero time, and an absent hub → empty
  slice + nil error.
- `go test -count=1 -run TestDossier ./internal/dossier` passes, including a new case that seeds a hub
  with ≥2 recorded checkpoints (distinct sizes) and asserts the served §5 HTML contains a size-transition
  line for each consecutive pair, in newest-first order, every value traceable to a fixture row.
- A new dossier test asserts the §5 HTML contains **no** "consistent" / "consistency PASSED" /
  "verified-poll" substring (the honest-derivation invariant), and that a hub with ≤1 checkpoint renders
  the honest empty §5 (no fabricated history line).
- Mutation check (reviewer-runnable, non-vacuous): dropping the size-transition append in the §5
  derivation makes the new dossier test FAIL; reverting `ORDER BY observed_at DESC` to `ASC` makes the
  order assertion FAIL.
- Store stays a leaf: `go list -deps ./internal/store` shows no `net/http` / logclient in the closure
  (unchanged from today).

## Done When
`mise run check` is green and the served hub dossier renders a §5 observation log of real recorded size
transitions (+ any confirmed anchor) newest-first with no synthesized per-poll "consistent" line, proven
by the new `TestListCheckpoints` store test and the new `TestDossier…` §5 seam tests.

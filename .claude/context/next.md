# Next Work Package

## Step: Light up certificate §5 BITCOIN ANCHOR (confirmed / pending) from the mirrored OTS row

## Advances
The **M-UI — Evidence Ledger frontend** milestone Verify criterion (target.md M-UI):

> "the **Bitcoin-anchor** panel and the **comparison-anchor** panel are separate, distinctly-labelled
> elements ("anchoring" copy is Bitcoin-only; a not-yet-anchored root renders the normal "pending"
> state, not an error)"

and the per-surface certificate named region (target.md "certificate of inclusion" mockup region):

> "the numbered clauses **§1 Subject · §2 Checkpoint · §3 Inclusion proof · §4 Signing key · §5 Bitcoin
> anchor · §6 Record history**"

This is the **last open certificate clause** (`HasClause5` is declared at
`internal/certificate/handler.go:286` but **never assigned `true`** — re-verified this iteration). It is
the nearest unmet **observable** Verify-closer; the state, handoff, and review `**Next:**` all name it
as the single next step. It also touches the **OTS / Bitcoin anchoring** milestone (the §5 surface is
the certificate-side observable for the anchor), whose remaining bar after this is the
offline-unprovable live-chain confirmation.

## Goal
Render the §5 BITCOIN ANCHOR clause on the certificate page for a certifiable id whose accepted root has
an OTS row: a **confirmed** anchor shows the Bitcoin block height (+ confirmation time), a **pending**
(stamped-but-not-yet-confirmed) anchor shows the honest "pending" state — never an error — and an
un-anchored root simply omits §5. This closes the last numbered certificate clause and is the
certificate-side observable surface for the anchoring milestone.

## Scope
- **Create**: (none — §5 reuses the existing `cert.html` `{{if .HasClause5}}` block, filled in)
- **Modify**:
  - `internal/certificate/handler.go` (≤3 budget, 1 of 1) — populate the §5 fields in `buildData`
  - `internal/certificate/cert.html` (template, not counted against the 3-file budget) — fill the empty
    §5 `clause-value` with the confirmed/pending markup
  - `internal/certificate/handler_test.go` (test, not counted) — add §5 HTTP-seam tests
- **Reference**:
  - `.claude/context/learnings/certificate.md` (the §1–§6 buildData mechanics + buffer-then-200 +
    `html/template` base64-escape trap)
  - `.claude/context/learnings/ots.md` (`ots.Confirmed` is fail-closed, NOT WASM-pure — keep it out of
    WASM-shared closures; certificate is server-side-only so importing it is fine)
  - `.claude/context/learnings/http-surface.md` (the `.ots` route's empty-`OTSBytes` sentinel edge case)
  - `internal/store/ots.go` (`OTSForRoot` signature + the empty-`OTSBytes` sentinel + miss-is-not-error)
  - `internal/ots/ots.go` (`Confirmed(otsBytes) (confirmed, height int64, err)`)
  - `.claude/design/ISCC Monitor - Certificate.dc.html:66` (the §5 named-region: status dot +
    `block N · <time> UTC` + "OpenTimestamps … run `ots verify`" note)
  - `internal/certificate/cert.html:375-380` (the empty `HasClause5` block to fill)

## Not In Scope
- The **separate comparison-anchor panel** (target.md names Bitcoin-anchor AND comparison-anchor as
  distinct elements). §5 is the Bitcoin-anchor side; the comparison-anchor element is a later sub-step —
  do not build it here.
- The **dossier §4 Bitcoin-anchor** region (`ISCC Monitor - Hub Dossier.dc.html` "§4 Bitcoin anchor")
  — a different surface; a separate step.
- The **`safeStamp` guard** (open `normal` OTS issue) — that lives on the *stamp* path
  (`internal/otsclient`/`internal/follower`), NOT touched by this read-only §5 clause; fold it in when
  the stamp path is next edited.
- The deferred **`host:port` DID `%3A`-encode** fix — §5 does not build a DID; do not touch §4's
  `"did:web:" + data.Domain` here (handler.go is touched, but §5 adds no DID; leave §4/bundle DID alone
  so the fix stays a single coherent later step).
- A **Bitcoin-confirmed live-chain transit** — offline-unprovable (needs a live calendar + real BTC
  confirmation); §5 renders whatever the mirrored row already holds.
- The **proof-bundle `ots?` member** — the bundle currently omits OTS; adding it is a later sub-step,
  not part of the page §5 clause.
- Touching `cmd/iscc-monitor/main.go` — the §5 read uses the `st *store.Store` the handler already
  holds; no new wiring is needed, so main.go must stay byte-unchanged.

## Implementation Notes
- **Where:** add the §5 read in `buildData` AFTER §2 sets `HasClause2`/`data.CheckpointSize`/`root` (it
  needs the accepted `root []byte` and `hub.LastSize`), guarded by `if data.HasClause2`. The §5 read is
  `st.OTSForRoot(r.Context(), hub.HubID, hub.LastSize, root)` — same `(hubID, treeSize, root)` key the
  `.ots` route and store CRUD use. `root` here is `CheckpointAt`'s **raw `[]byte`** (already in hand in
  the §2 branch), NOT the base64 `CheckpointRoot` string — `OTSForRoot` keys on raw bytes.
- **Three honest states (fail-closed, ADR-0001):**
  1. `OTSForRoot` miss (`found == false`) OR the empty-`OTSBytes` sentinel (`len(rec.OTSBytes) == 0`,
     stamped-at-observation-but-not-yet-calendar-submitted — the load-bearing edge case the `.ots`
     route guards, see learnings/http-surface.md) → **no §5** (`HasClause5` stays false). An
     un-anchored root simply omits the clause; this is NOT an error.
  2. A row with non-empty `OTSBytes`: classify via `ots.Confirmed(rec.OTSBytes)`. A `Confirmed` **error**
     (malformed/garbage proof) is a SILENT decline of §5 (`HasClause5` stays false), NEVER a 500 — same
     discipline as §3's `proof.VerifyInclusion` non-nil silent decline. OTS must never crash/fault the
     surface (learnings/ots.md: `Confirmed` is fail-closed and panic-recovered).
  3. A parseable row: set `HasClause5 = true`. If `confirmed`, populate the block height (`int64`) and,
     if present, the confirmation time from `rec.UpgradedAt` (RFC-3339, omit if zero — mirror §4's
     `SigningKeyRevoked` zero-time guard). If NOT confirmed, render the honest **"pending"** state
     ("calendar-asserted, awaiting Bitcoin confirmation"), per target.md "a not-yet-anchored root
     renders the normal 'pending' state, not an error".
- **A `store.OTSForRoot` DB error is a 500** (buffer-then-200, like every other clause's DB-fault split
  — `CheckpointAt`/`LookupHubKey`/`RecordAt`). Only the genuine DB fault 500s; a miss / sentinel /
  `Confirmed`-parse-error are honest declines.
- **certData fields:** `HasClause5 bool` is already declared — add `BTCConfirmed bool`, `BTCHeight int64`
  (or a pre-formatted string), `BTCConfirmedAt string` (RFC-3339, rendered conditionally), with
  evergreen docstrings matching the §4 field style. Keep the view-model flat (no nested struct) to match
  the existing clauses.
- **Template (`cert.html:375-380`):** fill the empty `<div class="clause-value"></div>` per the mockup
  region (`.dc.html:66`): a status dot + `{{if .BTCConfirmed}}block {{.BTCHeight}}{{if .BTCConfirmedAt}}
  · {{.BTCConfirmedAt}}{{end}}{{else}}pending — calendar-asserted, awaiting Bitcoin confirmation{{end}}`
  + the note "OpenTimestamps. Run `ots verify` for the authoritative check." Use existing `clause-mono`
  / `clause-note` classes for consistency with §3/§4/§6. **"anchoring" copy is Bitcoin-only** (target.md
  invariant) — do not use the word for the comparison-anchor role.
- **Correctness rule (learnings index):** "OTS never blocks / never crashes the follower" — here it
  generalizes to: an OTS parse fault on a *read* surface fails closed (decline §5), never 5xx.
- **Fixture:** seed a confirmed OTS row by writing a real bundled vector. The `internal/ots/testdata`
  vectors (`hello-world.txt.ots` → height 358391, `empty.ots` → 129405) are the external `ots verify`
  oracle. Copy ONE verbatim into `internal/certificate/testdata/` (hermetic — do NOT read another
  package's testdata at runtime) and `RecordOTS` it for the accepted `(hubID, LastSize, root)` of the
  `fixtureStoreTiled` certifiable hub, then assert the rendered page carries `block 358391`. For the
  un-anchored assertion seed either no `ots` row or a row with the empty-`OTSBytes` sentinel and assert
  the body lacks the `§5 BITCOIN ANCHOR` marker. For the pending state, if no calendar-only bundled
  vector is available, cover it via a row whose `Confirmed` returns `(false, 0, nil)` (a non-empty proof
  with only a calendar attestation) — otherwise document that confirmed + un-anchored fully exercise the
  `HasClause5` gate and the pending branch is asserted at the template-string level.

## Verification
- `mise run check` is green (build + vet + test, all packages; `gofmt -l .` clean; `go mod tidy -diff`
  clean — no new prod dep beyond `internal/ots`/`internal/store`, both already in the module).
- `go test -count=1 -run TestCertificate ./internal/certificate` passes (existing + new §5 tests).
- A new `TestCertificateBitcoinAnchorConfirmed` (or similarly named): a certifiable id whose accepted
  root has a confirmed OTS row renders a §5 clause body containing `§5 BITCOIN ANCHOR` AND the block
  height literal (e.g. `block 358391`).
- A new `TestCertificateBitcoinAnchorUnanchored` (or similar): a certifiable id whose accepted root has
  NO OTS row (or only the empty-`OTSBytes` sentinel) renders the page **without** the `§5 BITCOIN
  ANCHOR` marker (`HasClause5 == false`), and the rest of the certificate (§1–§4,§6) is unaffected.
- **Mutation (non-vacuous):** forcing `HasClause5 = true` unconditionally renders §5 for the un-anchored
  fixture → `TestCertificateBitcoinAnchorUnanchored` FAILS; reverting restores green. (Record this in
  the advance so review can re-run it.)
- `GOOS=js GOARCH=wasm go build ./internal/didweb ./internal/index ./internal/badge` still builds (proves
  `internal/ots`'s non-WASM-pure closure did NOT leak into a WASM-shared package — certificate is
  server-side only).

## Done When
§5 BITCOIN ANCHOR renders the confirmed (block height) and honest pending states from the mirrored OTS
row and omits cleanly for an un-anchored root, with all Verification criteria — including the
unanchored-no-§5 mutation and the WASM-purity guard — passing under `mise run check`.

# Next Work Package

## Step: Make the certificate §1 inclusion claim sound — accepted-tree cap + ISCC:-prefixed lookup

## Advances
Closes the two open `critical` issues that block the M-UI certificate Verify criterion. Per the
`review` handoff (**Next:** "Fix the two `critical` defects — they belong together in ONE slice") and
`state.md` "Next Milestone" step 1, these preempt all other work: the headline §1 SUBJECT clause
currently makes an affirmative inclusion claim it cannot back. The criterion advanced toward is
target.md M-UI:

> "the **realm-wide certificate** (`/inclusion/{iscc_id}` …) for a known id renders the numbered
> evidence clauses (subject + position; …) … while an unknown id renders the documented 'not found in
> log' state (200, never 5xx)"

The skeleton landed NEEDS_WORK because §1 certifies leaves *outside the accepted tree* and looks up a
*bare* id that never matches the stored `ISCC:`-prefixed key. This step makes the §1 claim honest so
the criterion can progress (its §2–§6 clauses follow in later steps, see `## Not In Scope`).

## Goal
Gate the certificate's affirmative inclusion claim on the accepted-tree cap (`seqs[0] < LastSize`,
like every sibling record route) and canonicalize the lookup id to the stored `ISCC:`-prefixed form,
so a real declaration in an accepted checkpoint certifies while an unaccepted / unindexed / pre-coverage
id renders the honest cannot-certify state — with fixtures re-grounded to production's storage format.

## Scope
- **Modify**: `internal/certificate/handler.go` (`buildData` + `followedHub`) — the only non-test source
  file. (1 of ≤3.)
- **Modify (tests, not counted)**: `internal/certificate/handler_test.go` — re-ground `fixtureStore`
  to index the leaf under the `ISCC:`-prefixed id and seed an accepted checkpoint covering it; add the
  two cannot-certify seam tests below.
- **Reference**:
  - `/workspace/iscc-monitor/.claude/context/learnings/certificate.md` — both OPEN traps documented
    verbatim (the accepted-tree cap + the prefixed-key mismatch); read before editing.
  - `/workspace/iscc-monitor/.claude/context/learnings/store.md` — `AdvanceAccepted` tx,
    `ListHubs`/`FollowState` leaf reads, the iscc_index contract.
  - `/workspace/iscc-monitor/.claude/context/learnings/http-surface.md` — the documented
    "iscc_index can hold projections ABOVE LastSize" trap + every sibling route's `>= size` cap.
  - `/workspace/iscc-monitor/.claude/context/issues.md` — the two `critical` entries carry the exact fix
    + verify recipe.
  - `/workspace/iscc-monitor/internal/store/hubs.go` (line 23: `HubSummary` carries `.LastSize` +
    `.Frozen`) and `/workspace/iscc-monitor/internal/store/iscc_index.go` (line 201: `SeqsForISCCID`,
    ORDER BY seq ascending).
  - `/workspace/iscc-monitor/internal/store/checkpoints.go` — the accept path (`AdvanceAccepted` /
    `FollowState`) the fixture uses to set `LastSize`; confirm the exact signature there.
  - `/workspace/iscc-monitor/internal/logclient/projection.go` (lines 31-32) — ground truth: `iscc_id`
    is stored **`ISCC:`-prefixed, verbatim**.
  - `/workspace/iscc-monitor/internal/certificate/cert.html` (lines 324, 378-391) — the
    `{{if .Certifiable}} … {{else}}` not-found branch already renders any `.Reason`; **no template
    change is needed** for the new cannot-certify states.

## Not In Scope
- The §2–§6 clauses (checkpoint, inclusion proof, signing key, Bitcoin anchor, record history) and the
  downloadable **proof-bundle assembler** — those re-engage the oracle/conformance gate and are their
  own later slices. Leave `HasClause2..6` false and the download button disabled.
- The separate Bitcoin-anchor vs comparison-anchor panels.
- The deferred `internal/registry` `hubDomain` `ForceQuery` fail-open (`normal`) — do NOT touch
  `registry.go` this step; it rides the slice that next edits `hubDomain`.
- The ADR-0011 Go 1.26 + iscc-lib bump (`normal`, human-sequenced, needs a Go 1.26 toolchain).
- Any new store query: the cap reads `LastSize` from the `HubSummary` that `followedHub` already fetches
  via `ListHubs` — do **not** add a second `FollowState` call.
- Editing `cert.html` — the existing `{{else}}` not-found branch covers both new states via `.Reason`.

## Implementation Notes
Both fixes live in `buildData` (+ a small `followedHub` return-type change). The template needs no
edit — the cannot-certify states route through the existing `{{else}}` branch by setting `data.Reason`.

1. **Carry `LastSize` out of the hub lookup (no extra query).** `followedHub` already calls
   `st.ListHubs` and returns the matching `HubSummary.HubID`. `HubSummary` also carries `.LastSize`
   and `.Frozen` (`internal/store/hubs.go:23`). Change `followedHub` to return the matched
   `store.HubSummary` (or at minimum `(hubID int64, lastSize uint64, ok bool, err error)`) so
   `buildData` has `LastSize` in hand without a second store round-trip. Keep it one linear scan.

2. **Accepted-tree cap (Correctness rule: coverage honesty, ADR-0001).** After
   `seqs, err := st.SeqsForISCCID(...)`, keep the current `len(seqs) == 0 → "not found in log"`. Then
   add the cap **before** setting `Certifiable`:
   - if `LastSize == 0` → `data.Reason = "no accepted checkpoint yet"`, return 200 (not certifiable);
   - else if `seqs[0] >= LastSize` → `data.Reason = "not in accepted tree"`, return 200 (the leaf is
     indexed but above the accepted checkpoint — a frozen/failed poll left an unaccepted projection;
     see learnings/http-surface.md "iscc_index can hold projections ABOVE LastSize").
   Only `len(seqs) > 0 && seqs[0] < LastSize` sets `data.Certifiable = true`. This mirrors
   `serveInclusion` / `serveEntries` / `serveRecord` (`leafIndex/seq >= size → not served`).
   `seqs` is ascending (`SeqsForISCCID` ORDER BY seq), so `seqs[0]` is the earliest indexed candidate —
   the right one to gate on. Set `data.Domain = domain` on these branches too, so the page names the hub.

3. **Canonicalize the lookup id to the stored prefixed form (ADR-0008 schema-agnostic index).**
   Production stores `iscc_id` `ISCC:`-prefixed and verbatim (`projection.go:31-32`). The handler
   currently passes the bare path suffix `rawID`. After `index.Decode(rawID)` succeeds (so we know it
   is a valid ISCC-IDv1), build the canonical key once:
   `lookupID := "ISCC:" + strings.TrimPrefix(rawID, "ISCC:")` — this accepts either `/inclusion/MAIG…`
   or `/inclusion/ISCC:MAIG…` and always queries the single stored prefixed form (do NOT double-prefix).
   Pass `lookupID` (not `rawID`) to `SeqsForISCCID`. Keep echoing `rawID` as `data.IsccID` for display.
   `index.iscPrefix` is unexported, so use the literal `"ISCC:"` here (matching how `cert.html` carries
   literal `/_ds/` paths).

4. **Re-ground the fixtures to ground truth, not to the code.** In `handler_test.go` `fixtureStore`:
   - index the leaf under the **prefixed** id: `IsccID: "ISCC:" + indexedID` in the `ProjectionRecord`
     (currently it seeds the bare form), matching `projection_test.go`'s `"ISCC:MAIG…"`. Keep the
     `indexedID` argument the bare golden id and prefix it inside `fixtureStore`.
   - seed an **accepted checkpoint** covering the leaf so `LastSize > seq`. Use the store's accept path
     (`AdvanceAccepted` — the same call `checkpoints_test.go` uses to set `LastSize`); confirm the API
     in `internal/store/checkpoints.go`. For the golden seq `24815`, set `LastSize` to e.g. `24816`+.
   `TestCertificateKnownID` then proves the real production path (prefixed key + accepted tree), and the
   bare-suffix request `/inclusion/MAIGHFECJMOPMIAB` still certifies (proving the canonicalization).
   The existing not-in-log / malformed / unresolvable / not-followed / empty / nil-HubList tests stay
   green (none of them assert `Certifiable`); fix any that now need an accepted checkpoint to certify.

5. **Add two cannot-certify seam tests** (mutation-provable, ground-truthed):
   - `TestCertificateUnacceptedLeaf`: index the prefixed leaf at a seq `>= LastSize` (or with
     `LastSize == 0` — no accepted checkpoint) and assert the body renders the cannot-certify state
     ("not in accepted tree" / "no accepted checkpoint yet") and **not** the "is included in the
     transparency log" banner. Reverting the cap makes this FAIL.
   - `TestCertificatePrefixedLookup` (or fold into the known-id test): with the leaf indexed under the
     prefixed id, the bare-suffix request still certifies; reverting the canonicalization makes it FAIL.

Edge cases: a frozen hub's `LastSize` is its last *accepted* size (freeze stops advance, ADR-0006), so
the same `seqs[0] < LastSize` cap correctly caps a frozen hub at its accepted window — no separate
frozen branch needed. Keep all branches 200 (fail-closed); a store error stays the only 500
(buffer-then-200, unchanged).

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...`, `gofmt -l .` empty).
- `go test -count=1 ./internal/certificate` passes uncached.
- `go test -count=1 -run TestCertificateKnownID ./internal/certificate` passes with the leaf indexed
  under `"ISCC:"+goldenID` and an accepted checkpoint with `LastSize > 24815`, asserting the certifiable
  subject banner + position `24815` for the bare-suffix request `/inclusion/MAIGHFECJMOPMIAB`.
- `go test -count=1 -run TestCertificateUnacceptedLeaf ./internal/certificate` passes: a prefixed leaf
  at seq `>= LastSize` (or `LastSize == 0`) renders the cannot-certify state, not the subject banner.
- Mutation check (reviewer reproduces): (a) removing the `seqs[0] < LastSize` cap →
  `TestCertificateUnacceptedLeaf` FAILS; (b) reverting the lookup to bare `rawID` →
  `TestCertificateKnownID` FAILS (declaration reports "not found in log").
- `GET /inclusion/MAIGHFECJMOPMIAB` and `GET /inclusion/ISCC:MAIGHFECJMOPMIAB` both certify the same
  leaf (canonicalization accepts either input form).

## Done When
`buildData` gates `Certifiable` on `len(seqs) > 0 && seqs[0] < LastSize` and looks up the
`ISCC:`-prefixed id, the fixtures are re-grounded to the prefixed form + an accepted checkpoint, both
new tests pass and are mutation-proven non-vacuous, and `mise run check` is green — closing the two
open `critical` issues so the certificate §1 claim is sound.

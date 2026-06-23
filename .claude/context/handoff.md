## 2026-06-23 — Review of: Hub dossier increment 1 — numbered trust-document layout + §1–§4

**Verdict:** PASS_WITH_NOTES
**Loop:** CONTINUE

**Summary:** The advance rebuilt the served hub dossier (`GET /<domain>`) from a flat 5-row ledger card
into the mockup's numbered trust document — trust-document head, the 2×2 §1–§4 grid (honesty-gated),
§5 placeholder, the two action links, the `fork → "split view"` map, with the frozen Exhibit + masthead
intact. The work is clean, well-documented, scope-disciplined (3 prod + 2 test files), gate-green, and
mutation-proven; a live visual pass confirms full design-parity with the mockup. Two confirmed Codex P2
honesty nits in edge states (frozen §3 size/time decouple; §1 "resolved" vs unresolvable) are real but
narrow, do not block, and are filed `normal` for increment 2 / a design pass.

**Verification:**
- [x] `mise run check` (build + vet + test) — green, all 28 pkgs `ok`; `gofmt -l .` empty.
- [x] `go test -count=1 -run TestDossier ./internal/dossier` — 12 tests pass (fresh, uncached), incl. the
  4 new/updated cases (`RendersCoveredHub` §1–§4/head/actions/§5, `ConfirmedAnchorRendersHeight`,
  `PendingAnchorHonest`, `CautionForUnverified`).
- [x] `go test -count=1 -run TestListHubs ./internal/store` — `TestListHubsCheckpointAndAnchorHeight` +
  existing `TestListHubsAnchorStatus` pass (populated + NULL-safe zero-value).
- [x] §1–§4 each render label + value; confirmed-anchor fixture renders `block 869440`; pending/absent
  renders honest "pending"/"not anchored", no 5xx, no error styling, no fabricated `block 0`.
- [x] Action links resolve: `href="/">Prove an ISCC-ID…`, `href="/sb0.iscc.id/log/">Browse the log…`.
- [x] Frozen Exhibit + soft caution distinct; §5 honest minimal placeholder (heading, no fabricated lines).
- [x] No-CDN body ban (`jsdelivr`/`cdn.`/`unpkg`/`googleapis`/`http://`) + no-JS (`<script` banned) asserted.
- [x] Store stays a leaf (additive `HubSummary` fields, NULL-safe subselects mirroring the `Anchor` pattern;
  `time.Unix(observed,0)` / `uint64(height)` consistent with schema's INTEGER cols + existing read idioms).
- [x] Mutation checks non-vacuous (reviewer-run): break §3 `ObservedTime` binding → `TestDossierRendersCoveredHub`
  FAILS; drop the `AnchorHeight` assignment → `TestListHubsCheckpointAndAnchorHeight` FAILS.
- [x] Gate-integrity scan over 3 unpushed commits — no `nolint`/`t.Skip`/swallowed-err/build-tag/removed
  assertion; `TestDossierRendersCoveredHub` only ADDED assertions (replaced 2 coverage markers with the
  §2-format equivalents + many section markers).
- [x] Oracle/conformance gate N/A — pure HTML render of persisted store rows + in-memory overlay; touches no
  signature/RFC-6962/Merkle/did:web/fsck/proof path; `go.mod`/`go.sum`/`schema.sql` byte-identical.

**Issues found:** Two confirmed Codex findings filed `normal` (below); the resolved increment-1 critical
deleted from issues.md; the increment-2 critical un-gated (now pickable) with the §3 fix folded in.

**Codex second opinion:** Finished (exit 0). Two `[P2]` findings, BOTH reviewer-confirmed real but narrow:
- **§3 frozen size/time decouple (`hubs.go:69`)** — CONFIRMED against `follower.freeze` (`follower.go:475`
  does `RecordCheckpoint` of the contradictory, often-higher-tree-size checkpoint without advancing
  `last_size`; `RecordCheckpoint` never writes `last_size`). So a frozen hub can pair the accepted §3 size
  with the rejected checkpoint's `observed_at`. Confined to the frozen edge state (the loud Exhibit already
  says "do not trust new state"); `shrink` can't trigger it. → filed `normal`, folded into increment 2 (it
  reworks §3). Does not block.
- **§1 "Key resolved from" vs `unresolvable` (`dossier.html:490`)** — CONFIRMED: §1 renders "Key resolved
  from did:web:…" unconditionally while the `unresolvable` caution says the key is unresolved — a same-page
  contradiction. BUT the advance followed next.md's Implementation Note + the mockup literally (static §1
  phrasing), so this is design-rooted; fix needs neutral wording / a design call, not a silent copy rewrite.
  → filed `normal`. Does not block.
- Neither finding touches the trust root; no oracle conflict. Both are honesty-wording nits in edge states,
  not correctness/verification defects.

**Visual check:** Done (SSR surface changed — `internal/dossier`). `agent-browser` launched headless against
a fixture-rich confirmed-anchor hub (seeded via a throwaway harness serving the dossier + `/_ds/` assets,
since the live cold-start index is empty); harness removed before commit. The rendered surface is a full
design-parity match to `.claude/design/ISCC Monitor - Hub Dossier.dc.html`: logo + instance-identity +
`verify ↗` masthead, `← Realm index` back-link, trust-document head (eyebrow "HUB DOSSIER", `<h1>`
`sb0.iscc.id`, `sb0.iscc.id/log`, `md` green Verified badge, heavy rule, "Compiled by monitor.iscc.id · …"),
the §1–§4 grid with the green confirmed dot + "confirmed · block 869440", the §5 honest placeholder, and the
two action buttons (filled "Prove an ISCC-ID…", bordered "Browse the log →"). No visual delta filed.

**Next:** Increment 2 (the now-pickable sibling `critical`) — the §5 observation log (a `ListCheckpoints`-style
leaf read: size transitions + freeze + anchor confirmations, NO synthesized per-poll "consistent" lines) and
the richer frozen Exhibit ("size before → presented" + a stable evidence ref from `RawA`/`RawB`). Fold in the
§3 frozen size/time-decouple `normal` while reworking §3 (select `observed_at` for the `f.last_size` row).

**Notes:**
- `dashboard.md` learnings was over the ~150-line rotation budget (183); created `learnings/dossier.md`
  (52 lines, with index pointer) and net-reduced dashboard.md to 170 (collapsed the settled dossier-masthead
  bullet to a one-line `settled:` + trimmed the `/`-identity bullet). Still slightly over 150 but materially
  reduced this iteration; remaining content is the active dashboard surface.
- The §2 "N days observed" is a live `time.Since(c.Since)` derivation (grows with wall-clock); tests assert
  the literal "days observed" suffix, not a fixed number — deterministic. Future-dated `Since` clamps to 0.
- The §4 height subselect is correctly scoped to `status = OTSStatusConfirmed` (distinct from the pre-existing
  per-hub `Anchor` status subselect, which is newest-stamped-regardless-of-status) — two subselects by design.
- 3 unpushed commits in `@{upstream}..HEAD` (update-state, define-next, advance); the two human UI tweaks
  (`0bb1963`, `f28f57e`) are already on the remote. Pushing `develop` on this PASS_WITH_NOTES.

# Next Work Package

## Step: Prune the 4 resolved-but-unpruned issues and surface the human/design gate

## Advances
No `target.md` milestone Verify criterion is open: **M1, M2, M3, M-UI code-halves, the WASM target
criteria, OTS observable halves, M-Deploy, and M-API (4/4) are all MET.** This step therefore advances
no Verify criterion — and that is correct here, because **the loop is out of autonomous code-closable
work** (`state.md` "Next Milestone" + auto-memory `loop-stalls-on-human-blocked-done`):

- the lone **`critical`** (dossier→log-browser navigation + record-list parity) is **fully code-closed
  and human-blocked** on the M-UI exit sign-off (ADR-0012) — the protocol forbids re-attempting it;
- the two genuinely-open `normal`s (WASM cross-origin signature half, realm-index per-hub-vs-per-checkpoint
  Anchor honesty) are both **design-blocked** — each issue body itself says "needs a design pass," so a
  blind code attempt is forbidden;
- the remaining four open issues are **resolved-but-unpruned bookkeeping** — I re-verified at the
  served-spec seam (not on faith) that their fixes already landed (the OpenAPI verify/checkpoint fixes in
  `53ee328`; the `/` realm-index sub-items in `b74931f`/`b30b84e`/`6a442b4` + the human design tweak).
  Pruning them is the one concrete, mechanically-verifiable action available that does **not** manufacture
  a cosmetic refactor. Preempting milestone work is justified because there is no open milestone Verify
  criterion and no code-closable issue.

## Goal
Delete the four issues whose fixes have already shipped (so `issues.md` reflects reality and the DONE
gate's "0 open `normal`" condition is honestly counted), and leave the residual open set — the
human-blocked `critical` + the two design-blocked `normal`s — as the clear marker that the only remaining
blockers are the human M-UI exit sign-off and two design passes. Prevents the loop from spinning on
cosmetic chrome.

## Scope
- **Create**: none
- **Modify**: `.claude/context/issues.md` (a doc/backlog file — touches **zero** source files; the ≤3
  non-test/doc code-file budget is unused)
- **Reference**:
  - `internal/openapi/openapi.yaml` (verify op lines 220-251 declare only `Domain`+`iscc_id`; checkpoint
    200 lines 264-270 = `application/octet-stream`) and `internal/openapi/openapi.json` — the proof the
    two M-API contract-accuracy issues are already fixed.
  - `.claude/context/state.md` "Convergence" + "Next Milestone" (the prune list + the blocked-gate
    framing this step executes).
  - `.claude/context/handoff.md` ("prefer pruning + flagging the human/design gate over manufacturing a
    `low` refactor").

## Not In Scope
- **Do NOT touch any source, test, or template file.** No `.go`, no `.html`, no `openapi.{yaml,json}`
  edit — the specs are already correct; this is backlog hygiene only.
- **Do NOT re-attempt the human-blocked `critical`** (M-UI exit sign-off) or the two design-blocked
  `normal`s (WASM signature half, realm-index Anchor honesty) — they stay OPEN; a code attempt is
  forbidden until the human/design input lands.
- **Do NOT delete any `low` issue** — lows are the human-directed backlog; leave all ~22 intact.
- Do NOT add a new "loop is blocked" prose entry to `issues.md` — that would duplicate `state.md`'s "Next
  Milestone" and risk drift; the residual open set is itself the marker.
- Do NOT manufacture a cosmetic/locality `low` refactor to keep the loop busy (auto-memory
  `loop-stalls-on-human-blocked-done`).

## Implementation Notes
Delete exactly these four `## …` issue sections from `.claude/context/issues.md` (each header is unique —
grep-verified, current line numbers shown but re-grep before editing). For each, remove the header line
through the last line before the next `## ` header (or the next `---` / HTML-comment separator), leaving
no orphan body lines:

1. **`## `/` realm-index: only the "recent declarers checked" hero footer remains …`** (line ~268,
   `normal`) — the issue's own body ends "**All four sub-items of this issue are now CLOSED** — the next
   `update-state` may prune this entry."
2. **`## OpenAPI contract advertises a phantom `index` query param on `/{domain}/log/verify` …`**
   (line ~638, `normal`) — VERIFIED fixed: the served `verify` operation declares only `Domain` +
   `iscc_id`, no `index` (openapi.yaml:235-242; json twin confirmed).
3. **`## OpenAPI contract advertises `text/plain` for `/{domain}/log/checkpoint` …`** (line ~662,
   `normal`) — VERIFIED fixed: the `checkpoint` 200 content type is `application/octet-stream`
   (openapi.yaml:264-270; json twin confirmed).
4. **`## No machine-readable API contract (OpenAPI) and no interactive API docs hosted by the app`**
   (line ~699, `normal` umbrella) — slices 1-4 all landed; its last code-closable child (#2/#3 above) is
   resolved and the remaining child is the `low` healthz-503, so the umbrella prunes.

Keep the surrounding structure intact: do **not** remove the `---` separators or the
`<!-- pre-deployment asks … M-Deploy -->` HTML-comment block; only remove whole issue sections. After the
deletions the still-open `normal`s in the file are exactly the WASM-signature-half issue and the
realm-index-Anchor-honesty issue (both design-blocked), plus the human-blocked `critical` at the top and
the `low`s.

Relevant learnings: the always-loaded `learnings.md` has no rule bearing on backlog editing, and this
step touches no package detail file. The correctness-relevant fact this step RELIES ON — that the OpenAPI
contract is the accurate machine surface — was re-verified at the served-spec seam above, not taken on
faith from `state.md`.

## Verification
- The four pruned headers are **absent** from the tree (each grep exits non-zero, so the negation exits 0):
  `! grep -qF 'recent declarers checked' .claude/context/issues.md`,
  `! grep -qF 'advertises a phantom' .claude/context/issues.md`,
  `! grep -qF 'serves `application/octet-stream`' .claude/context/issues.md`,
  `! grep -qF 'No machine-readable API contract' .claude/context/issues.md`.
- The two design-blocked `normal`s **survive** (not over-pruned):
  `grep -qF 'never checks the checkpoint signature against the hub' .claude/context/issues.md` passes AND
  `grep -qF 'Anchor column is per-hub' .claude/context/issues.md` passes.
- The human-blocked `critical` **survives**:
  `grep -qF 'Browse the log →' .claude/context/issues.md` passes.
- **No source/test/template changed** — only the two context files are dirty:
  `git status --porcelain | grep -vE '\.claude/context/(issues|next)\.md$' | grep -q .` exits **non-zero**.
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...`, `gofmt -l .` empty) —
  confirms the doc-only edit broke nothing.
- The served OpenAPI is still accurate (the prune did not mask a real defect):
  `go test -count=1 ./internal/openapi` passes (the per-operation goldens added in `53ee328`).

## Done When
The four resolved-but-unpruned `normal` issues are removed from `issues.md`; the two design-blocked
`normal`s, the human-blocked `critical`, and all `low`s remain; no source/test/template file changed; and
`mise run check` is green.

## 2026-06-23 — Prune the 4 resolved-but-unpruned issues and surface the human/design gate

**Done:** Deleted the four `normal` issues from `issues.md` whose fixes had already shipped — the `/`
realm-index sub-item issue (all four sub-items CLOSED), the two M-API OpenAPI contract-accuracy issues
(phantom `verify` `index` param; `checkpoint` `text/plain`), and the M-API umbrella issue — after
re-verifying at the served-spec seam (YAML source + JSON twin + per-operation goldens) that the OpenAPI
fixes are genuinely in place. This is backlog hygiene only: zero source/test/template files touched. The
residual open set is now exactly the human-blocked `critical` (M-UI exit sign-off) + the two
design-blocked `normal`s (WASM signature half, realm-index Anchor honesty) + the `low`s — the clear
marker that the loop is out of autonomous code-closable work.

**Files changed:**
- `.claude/context/issues.md`: removed four whole `## …` issue sections (realm-index sub-item; phantom
  `index` param; `text/plain` checkpoint; the "No machine-readable API contract" umbrella). Each deletion
  ran header-through-blank-line-before-next-header, leaving no orphan body lines. The `low` healthz-503
  issue that sat between two pruned sections is preserved; both `---` separators and the M-Deploy
  HTML-comment block are intact.

**Verification:** `mise run check` → green (build + vet + test across all 30 packages, all `ok`);
`gofmt -l .` empty. Per-criterion:
- [x] Four pruned headers absent — all four negation greps print PASS (`recent declarers checked`,
  `advertises a phantom`, `serves \`application/octet-stream\``, `No machine-readable API contract`).
- [x] Two design-blocked `normal`s survive — `never checks the checkpoint signature against the hub` AND
  `Anchor column is per-hub` both present.
- [x] Human-blocked `critical` survives — `Browse the log →` present.
- [x] No source/test/template changed — `git status --porcelain | grep -vE 'context/(issues|next)\.md$'`
  exits non-zero (only `issues.md` is dirty; `next.md` was already committed by `define-next`).
- [x] `mise run check` green; `gofmt -l .` empty.
- [x] Served OpenAPI still accurate — `go test -count=1 ./internal/openapi` passes (per-operation
  goldens from `53ee328`). The served `verify` op declares only `Domain`+`iscc_id` (no `index`); the
  `checkpoint` 200 content type is `application/octet-stream` — confirmed in BOTH the YAML source and the
  JSON twin before pruning, so the prune did not mask a real defect.

**Next:** The loop is out of autonomous code-closable work — no open milestone Verify criterion, the
lone `critical` is human-blocked on the M-UI exit sign-off, and both remaining `normal`s are
design-blocked (each issue body says "needs a design pass"). `define-next` should NOT re-attempt any of
those three in code. The only honest forward motions are: (a) wait for the human M-UI sign-off / the two
design passes; or (b) if a future slice naturally touches one of the `low`-flagged files, fold in a
locality `low` THEN (never manufacture one to stay busy — auto-memory `loop-stalls-on-human-blocked-done`).

**Notes:**
- Scope discipline clean: doc/backlog-only edit, zero source/test/template files; the ≤3 code-file
  budget was unused. Nothing from `## Not In Scope` touched — no `.go`/`.html`/`openapi.{yaml,json}`
  edit, the human-blocked `critical` and the two design-blocked `normal`s left OPEN, all `low`s intact
  (25 issue headers remain).
- Oracle/conformance gate is N/A this iteration — no signature, RFC-6962, Merkle, did:web, fsck, OTS, or
  proof code touched. The OpenAPI golden run was a read-only re-verification that the contract the prune
  relied on is accurate, not a code change.
- The correctness-relevant fact this step relied on (the OpenAPI contract is the accurate machine
  surface) was re-verified at the served-spec seam — YAML lines for verify (`220-251`) + checkpoint
  (`252-262`) and the JSON twin via a `json.load` probe — not taken on faith from `state.md`.
- No new issue filed; no `low` manufactured. The residual open set is itself the loop-blocked marker, so
  no "loop is blocked" prose entry was added to `issues.md` (would duplicate `state.md`'s "Next
  Milestone" and risk drift).

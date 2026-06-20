# Handoff

## 2026-06-20 — Review of: Single-poll follower — wire fetch → accept → record → advance for one hub

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** `advance` added the new `internal/follower` package with `PollHub`, the first real caller
composing the M1 verify chain (`FetchCheckpoint → AcceptCheckpoint`) with the store CRUD
(`RecordCheckpoint → AdvanceFollowState`) for one hub, one observation. It persists and advances the
follow cursor only on `StatusVerified`; the other three verdicts return to the caller untouched. Two
new files, zero existing source changed, all gates green, scope exactly as `next.md` asked.

**Verification:**
- [x] `mise run check` → green (build + vet + test all exit 0, go1.24).
- [x] `gofmt -l .` → empty.
- [x] `go test -count=1 -run TestPollHub ./internal/follower` → PASS (both subtests, re-run verbosely).
- [x] Verified path: composite fetcher (sb0 checkpoint + sb0 did.json), `baseURL="https://sb0.iscc.id"`,
  `observedAt` 2026-06-20 → `StatusVerified`, `FollowState(ctx, hubID).LastSize == 10183`. Confirmed
  `10183` is line 2 of the `sb0.iscc.id_checkpoint` fixture (the signed tree size) — a non-vacuous
  assertion that the value flowed `info.TreeSize → AdvanceFollowState → cursor`.
- [x] Non-advancing path: did.json advertises the mismatching `z6MkiNW46AUj…` key → `StatusUnverified`,
  `LastSize == 0`. Complementary to the verified case; neither is vacuously satisfied.
- [x] Store-leaf: `go list -deps ./internal/store` shows only `internal/store` itself; `./internal/follower`
  pulls `logclient` + `store` (+ transitive `didweb`) → direction follower → {logclient, store}, never
  the reverse.
- [x] Seam shapes verified against source: `UpsertHub(ctx,domain,origin,baseURL)`,
  `AcceptCheckpoint(ctx,fetcher,baseURL,raw,observedAt)→(Status,CheckpointInfo,error)`,
  `FetchCheckpoint(ctx,fetcher,baseURL)`, `RecordCheckpoint(ctx,rec)`, `AdvanceFollowState(ctx,hubID,size)`,
  `CheckpointRecord` fields, and `Fetcher` 1-method interface all line up with the diff's usage.
- [x] Gate integrity: scanned all unpushed commits (`origin/develop..HEAD`); the only code change is the
  two new follower files (other unpushed commits are CID context-only). No `//nolint`, `t.Skip`,
  build-tag exclusions, deleted tests, or swallowed errors — every `err` is checked and `%w`-wrapped.

**Conformance/oracle gate:** N/A this step (correctly). The diff touches no signature verification,
RFC-6962/Merkle, proof code, `internal/didweb`, or split-view logic (`git diff --name-only` confirms
zero trust-root files). The pure `AcceptCheckpoint` chain is reused unchanged; `derive_vkey.py` vectors
and WASM purity are untouched. The `notecheck` external-oracle CI job still does not exist — an
infrastructure gap to wire when the trust-root code lands, not a regression here.

**Issues found:** (none)

**Next:** The poll loop / single-writer goroutine wrapper that calls `PollHub` on a cadence and owns all
writes per network DB (ADR-0005/0007) is the natural follow-on. The independent alternative is the
`hub_keys` did:web cache write — that step MUST also refresh the stale `sb1.amlet.id_did.json` fixture
(still pre-rotation `22b08f3e`) and `derive_vkey.py` `HUBS` to the current sb1 checkpoint signer
`069d0f14`, and re-triggers the `derive_vkey.py` parity gate. The three-trigger consistency check
(fork/shrink/equivocation over `transparency-dev/merkle`) and freeze/alert are the other independent
≤3-file follower slices.

**Notes:**
- **Garbled-body fault contract is preserved through `PollHub`.** On `AcceptCheckpoint` returning a
  non-nil error, `PollHub` returns the wrapped error alongside that status (which is `StatusUnverified`'s
  zero — meaningless when err != nil) and persists nothing. Callers of `PollHub` must check `err` before
  the status, mirroring `AcceptCheckpoint`. Recorded in learnings.
- **Record-only-on-verified** is documented in the file docstring as a deliberate choice (non-verified
  verdicts carry a zero `CheckpointInfo` = no trustworthy `(size, root)`), flagged for a later step that
  may want to record non-verified observations as evidence of an internally-broken hub. Sound for v1.
- **Pushed** to `origin/develop` on this PASS verdict (3 commits were ahead; human merges develop→main
  via CI-gated PR — never push main).

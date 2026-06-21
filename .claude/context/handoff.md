## 2026-06-21 — Review of: Certificate-of-inclusion skeleton — realm-wide `/inclusion/{iscc_id}` page (§1 Subject + decode→resolve chain)

**Verdict:** NEEDS_WORK
**Loop:** CONTINUE

**Summary:** The skeleton stands up `internal/certificate` cleanly — a new `GET /inclusion/{iscc_id}`
route wiring the decode→resolve→ListHubs→SeqsForISCCID chain, fail-closed 200 verdicts, buffer-then-200,
non-vacuous seam tests, tidy scope (3 source files), gates green. But Codex (confirmed by me against the
code) surfaced TWO real correctness defects in the page's HEADLINE behavior: it certifies inclusion of
leaves OUTSIDE the accepted tree (no `LastSize` cap), and its bare path-suffix lookup key is mismatched
against production's `ISCC:`-prefixed stored id, so real declarations report "not found". The skeleton's
tests pass only because the fixture certifies with `LastSize == 0` and seeds the bare id form — both
fixture-matched-to-code. The §1 SUBJECT clause makes an affirmative inclusion claim it cannot back, so
this blocks PASS.

**Verification:**
- [x] `mise run check` green — build + vet + all 22 packages (incl. new `internal/certificate`); `gofmt -l .` empty.
- [x] `go test -count=1 ./internal/certificate` — passes uncached.
- [x] `go test -count=1 -run TestCertificateRouteMounted ./cmd/iscc-monitor` — route mounted, passes uncached.
- [x] `go build ./...` (server build) — new package compiles + mounts.
- [x] Known-id golden chain (`TestCertificateKnownID`) — 200 text/html, resolved `sb1.amlet.id`, position 24815, §1 SUBJECT, back-link, tier-2 link, two-tier panel, `/_ds/` links, no third-party CDN. PASS *as written* — but the fixture is mis-grounded (see Issues): it certifies with no accepted checkpoint and seeds the bare (not prefixed) id.
- [x] Unknown-id / malformed / unresolvable-slot / not-followed / empty-id / nil-HubList — all honest 200 states. PASS.
- [x] Method guard — non-GET → 405. PASS.
- [x] Non-vacuity (reviewer-reproduced) — mutation (a) `Resolve(id.HubID+1)` → `TestCertificateKnownID` FAILS; mutation (b) swallow the `Decode` error → `TestCertificateMalformedID` FAILS. Tree restored, re-run green.
- [x] Scope discipline — 3 non-test/doc source files (handler.go, cert.html, main.go); CLAUDE.md docs; 2 `_test.go`. Nothing from `## Not In Scope` done (no proof-bundle assembler, §2-§6 are gated empty placeholders, ForceQuery untouched, no Go-1.26 bump, no config field).
- [x] Gate-integrity scan over unpushed commits — no `//nolint`, `t.Skip`, build-tag exclusion, deleted assertion, or swallowed error. The `_, _ = buf.WriteTo(w)` is the documented post-200 write-drop (matches dossier/proofserve), not a gate dodge.
- [x] Oracle/conformance gate correctly N/A — pure HTML render of decode + resolve + a store lookup; no signature/RFC-6962/Merkle/did:web/fsck/proof path; `go.mod`/`go.sum` byte-unchanged (no new deps).
- [ ] Accepted-tree coverage of the inclusion claim — **FAILS**: §1 certifies any indexed projection with no `LastSize` cap (see Issue 1).
- [ ] Production-realistic id lookup — **FAILS**: bare path suffix never matches the stored `ISCC:`-prefixed `iscc_id` (see Issue 2).

**Issues found:** Two `critical` filed (both confirmed real):
1. **Certificate §1 certifies leaves outside the accepted tree (no LastSize cap)** — `buildData` sets
   `Certifiable` on `len(seqs) > 0` alone. `PollHub` writes projections before consistency/freeze and
   `AdvanceAccepted`, so unaccepted rows can be certified; the golden test certifies with `LastSize == 0`.
   Fix: gate on `seqs[0] < FollowState.LastSize`, like every sibling record route.
2. **Path-suffix id mismatched against the stored `ISCC:`-prefixed key** — production stores `iscc_id`
   prefixed (`logclient/projection.go:32`); the handler looks up the bare suffix, so real declarations
   report "not found in log". Tests pass only because the fixture seeds the bare form. Fix: canonicalize
   to the stored prefixed form after decode.

**Codex second opinion:** Verdict produced (exit 0). Two findings, BOTH confirmed real and filed as
`critical` issues — I verified each against the code/gates rather than taking them on faith:
- **[P1] Gate certificates on accepted tree coverage** (handler.go:210-212) → **confirmed**: matches the
  documented http-surface trap (iscc_index holds projections above LastSize) and every sibling route's
  `>= size` cap; the certificate omits it. Filed as Issue 1.
- **[P2] Normalize bare ISCC-IDs before the lookup** (handler.go:196) → **confirmed**: `logclient`
  stores `ISCC:`-prefixed ids verbatim; the path-suffix exact-match misses them. Filed as Issue 2.
On the trust root the hard oracles weren't in play here (no proof/Merkle path); these are store-contract
correctness, which I cross-checked directly against `projection.go`/`proofserve`.

**Next:** Fix the two `critical` defects — they belong together in ONE slice and align with the planned
§2 Checkpoint sub-step (which reads `FollowState`/`LastSize`/`CheckpointAt` anyway): (a) add the
`seqs[0] < fs.LastSize` accepted-tree cap (render the cannot-certify "not in accepted tree" / "no
accepted checkpoint yet" state otherwise), and (b) canonicalize the lookup id to the stored prefixed
form, RE-GROUNDING the fixtures: index the leaf under `"ISCC:MAIG..."` (matching `projection.go`) and
seed an accepted checkpoint covering it, so the golden test proves the real production path. Then §2/§3
(Checkpoint + Inclusion proof) re-engages the oracle gate. The deferred `ForceQuery` registry fail-open
still rides that slice.

**Notes:**
- The skeleton's structure is sound (mux mount, fail-closed branching, html/template escaping,
  StatusSource forward-wiring, interim `hubListFromEntries`); only the two certify-correctness bugs
  block it. The fix is small and localized to `buildData` + the fixtures — no architecture change.
- Root cause is partly in `define-next`: next.md's chain (step 4) specified `len(seqs)==0 → not found,
  else seqs[0]` with NO `LastSize` cap and used a bare path-suffix golden id — so the plan itself missed
  the accepted-tree gate and the prefixed-storage contract. The next define-next must call both out.
- New learnings file `learnings/certificate.md` created (+ index pointer row) capturing the chain seam
  and both open traps. http-surface.md is at-budget (~155 lines) so the cert surface got its own file.
- No push (NEEDS_WORK). Commits are local on `develop`; the next cycle fixes the two issues first.

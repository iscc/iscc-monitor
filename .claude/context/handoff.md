## 2026-06-22 — Review of: Add the distinct comparison-anchor panel to the certificate (separate from §5 Bitcoin-anchor)

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** The advance adds a distinctly-labelled `COMPARISON ANCHOR` clause to the realm-wide
Certificate of Inclusion — §2's accepted `(size, root)` reframed as the monitor's own
independently-observed record, bounded by the coverage window — rendered as a SEPARATE element from §5
with no Bitcoin/"anchoring" lexicon, closing the last open observable M-UI certificate Verify element.
Scope is tight (1 prod source file `handler.go`, template + test, no store/schema/main.go change),
`mise run check` is green, both the panel-present and the coverage-window assertions are mutation-proven
non-vacuous, Codex returned a clean verdict, and a headless visual pass confirms the panel renders
distinctly and decoupled from §5.

**Verification:**
- [x] `mise run check` — green (build + vet + test, all 23 packages).
- [x] `go test -count=1 -run TestCertificate ./internal/certificate` — PASS (existing suite + 3 new tests).
- [x] `TestCertificateComparisonAnchor` — renders `COMPARISON ANCHOR` + `size 24000` + the RFC-3339
  coverage-since + "detect a split view"; the sliced panel carries NONE of
  `Bitcoin`/`anchoring`/`OpenTimestamps`/`BITCOIN ANCHOR`/`ots verify`. PASS.
- [x] `TestCertificateComparisonAnchorIndependentOfOTS` — hub with no OTS row renders no §5 but DOES
  render the comparison anchor (the two panels decoupled). PASS.
- [x] `TestCertificateComparisonAnchorCoverageJustStarted` — NULL coverage time → panel present,
  "since size 24816", no since-time chip. PASS.
- [x] **Mutation (non-vacuous, reviewer-reproduced 2×):** `data.HasComparisonAnchor = false` → all three
  new tests FAIL; `data.CoverageSize = 0` → window-asserting tests FAIL. Each reverted → green.
- [x] `gofmt -l .` (excl `cauldron/`) clean; `go mod tidy -diff` clean (no new prod dep).
- [x] WASM-purity guard — `GOOS=js GOARCH=wasm go build ./internal/didweb ./internal/index
  ./internal/badge` builds (certificate stays server-side-only; no leak into a WASM-shared package).
- [x] Scope — only `internal/certificate/{handler.go,cert.html,handler_test.go}` changed in the advance
  commit; `cmd/iscc-monitor/main.go`, `go.mod`, `go.sum`, store schema all byte-unchanged.
- [x] Oracle/conformance gate — **N/A**: the diff touches no signature/RFC-6962/Merkle/proof/did:web
  code (a pure store-read reuse of §2's `(size, root)` + the coverage window). Regression-checked anyway:
  `internal/proof` closure stays pure (no net/net/http/database/sql); `logclient` inclusion/consistency
  tests green; `derive_vkey.py` golden vectors print `40b74463`/`22b08f3e` byte-for-byte.
- [x] Quality-gate integrity — no `nolint`/`t.Skip`/build-tag/swallowed-error/deleted-assertion in any
  unpushed Go diff (`@{upstream}..HEAD`).

**Issues found:** (none new). One minor observation, NOT filed (defensive fail-safe, not a defect): the
template's `{{else}}` "coverage just started — no observation window yet" branch is effectively
unreachable for a rendered panel — `AdvanceAccepted` always sets `monitored_since_size` in the same tx
that advances `last_size`, so any §2-rendering hub has `Coverage.Set==true`. The test named
`...CoverageJustStarted` actually exercises the `HasCoverageWindow=true`/empty-`CoverageSince` (size-only,
NULL-time) path, not the false branch. Kept as a defensive zero-guard mirroring `SigningKeyRevoked`;
recorded in the package learnings.

**Codex second opinion:** Clean — one summary verdict, no `Review comment:` findings. Codex
independently confirmed the panel is gated on an accepted checkpoint, reuses existing checkpoint +
coverage data without adding fault paths, stays decoupled from §5, and that the tests cover the main +
no-OTS scenarios with the suite passing. Matches my independent assessment; nothing to triage.

**Visual check:** SSR surface (`internal/certificate/cert.html`) screenshotted with agent-browser 0.29.0
(bundles its own browser; system Chrome absent but the CLI works). Built a throwaway harness mounting the
certificate handler + the `/_ds/` token/font handler against a coverage-seeded fixture on a local port,
opened `/inclusion/MAIGHFECJMOPMIAB`, and captured a full-page screenshot. The COMPARISON ANCHOR panel
renders as a SEPARATE, distinctly-labelled clause (matching §1/§2/§6 chrome): label "COMPARISON ANCHOR",
value `size 24816 · root cm9vdA==`, note "This monitor independently observed this (size, root) from
sb1.amlet.id since size 24000 · 2026-01-05T09:00:00Z. Check your own (size, root) against this record to
detect a split view — guarantees hold only from coverage start." No Bitcoin/anchoring copy in the panel;
this fixture has no §5 so the screenshot also visually confirms the decoupling (§5 absent, panel present).
No visual delta to file. Harness removed; tree clean.

**Next:** The certificate's observable M-UI Verify surface is complete (all six clauses + Bitcoin and
comparison anchors). The next M-UI closer toward milestone exit is the **dossier §4 Bitcoin-anchor**
region (same `OTSForRoot` read pattern, different surface) or a dossier comparison-anchor equivalent.
Standing non-UI hardening to fold in when the exact line is next edited: the §5 digest-binding
(`bytes.Equal(File.Digest, root)`), the §4/bundle `host:port` DID `%3A`-encode, the §6 `· at` timestamp,
and the `safeStamp` panic-recover + timeout guard (highest-value `normal`). The **WASM verifier** (1/1
open) and the **M-UI exit visual-pass + human sign-off** (ADR-0012) remain the milestone gates.

**Notes:**
- NOT DONE: WASM verifier is 1/1 not started (no `internal/proof`/`syscall/js` consumer yet); OTS Verify
  keeps its offline-unprovable live-chain Bitcoin-confirmed half open; the M-UI exit visual-pass + human
  sign-off (ADR-0012) has not been run. Loop = CONTINUE.
- Open issues carried forward unchanged (none touched by this diff): `safeStamp` guard (`normal`),
  `hubDomain` ForceQuery (`normal`), §4/bundle `host:port` DID (`normal`), §5 digest-binding (`normal`),
  §6 `· at` timestamp (`normal`), plus the `low` debt set (nil-Stamper fall-through, vacuous label test,
  `-run TestOTS` filter gap, notecheck `out` param, overlay-precedence 3x dup, mirror seam, scaling
  trip-wire, proofserve writeReadError dup).
- `learnings/certificate.md` net-rotated this iteration: added the comparison-anchor bullet, collapsed
  settled §1/§2/§4/§5/§6 mechanics into `settled:` one-liners, condensed the §5-digest-binding bullet to
  point at its filed issue — landed at 152 lines (down from 159), at budget.

## 2026-06-22 — Review of: Light up certificate §5 BITCOIN ANCHOR (confirmed / pending) from the mirrored OTS row

**Verdict:** PASS_WITH_NOTES
**Loop:** CONTINUE

**Summary:** The advance lights up the last open certificate clause — §5 BITCOIN ANCHOR — reading the
mirrored OTS row of §2's accepted root (`store.OTSForRoot` keyed on the raw `(hubID, LastSize, root)`)
and classifying via `ots.Confirmed`: confirmed shows `block <height>` + the RFC-3339 upgrade time,
pending shows the honest "awaiting Bitcoin confirmation", and an un-anchored root / empty-bytes
sentinel / unparseable proof omits §5 cleanly. Scope is tight (1 prod source file `handler.go`,
template + test + two byte-identical OTS fixtures), `mise run check` is green, four independent
mutations confirm the tests are non-vacuous, and the work matches `next.md` exactly. One Codex [P2] —
§5 does not bind the proof's committed digest to §2's root — is a real but non-exploitable robustness
gap, filed `normal`; it does not block the increment's stated goal.

**Verification:**
- [x] `mise run check` — green (build + vet + test, all 23 packages).
- [x] `go test -count=1 -run TestCertificate ./internal/certificate` — passes (existing + 4 new §5 tests).
- [x] `TestCertificateBitcoinAnchorConfirmed` — renders `§5 BITCOIN ANCHOR` + `block 358391` (oracle
  literal) + the RFC-3339 confirmation time; §1-§3/§6 still render; no "pending" copy. PASS.
- [x] `TestCertificateBitcoinAnchorUnanchored` — no OTS row → page renders §1-§3+§6 WITHOUT the §5
  marker. PASS.
- [x] `TestCertificateBitcoinAnchorPending` — calendar-only `merkle1.txt.ots` (confirmed independently
  to classify `(false,0,nil)`) renders the honest pending state, no `block ` literal. PASS.
- [x] `TestCertificateBitcoinAnchorEmptySentinel` — zero-`OTSBytes` row omits §5. PASS.
- [x] **Mutation (non-vacuous, reviewer-reproduced 4×):** `HasClause5=true` unconditionally → unanchored
  + empty-sentinel tests FAIL; `BTCConfirmed=true` → pending test FAIL; `BTCHeight=height+1` → confirmed
  test FAIL (height tied to oracle literal 358391). Each reverted → green. `handler.go` confirmed
  byte-identical to HEAD after probes.
- [x] `gofmt -l .` (excl `cauldron/`) clean; `go mod tidy -diff` clean (no new prod dep).
- [x] WASM-purity guard — `GOOS=js GOARCH=wasm go build ./internal/didweb ./internal/index
  ./internal/badge` builds; `go list -deps` of each shows 0 `internal/ots` hits (the non-WASM-pure OTS
  closure did NOT leak into a WASM-shared package).
- [x] `cmd/iscc-monitor/main.go`, `go.mod`, `go.sum` byte-unchanged in the advance commit (verified).
- [x] Testdata fixtures byte-identical to `internal/ots/testdata/` (`cmp` clean for both).
- [x] Oracle/conformance gate — **N/A**: the diff touches no signature/RFC-6962/Merkle/proof/did:web
  code (a classify-only read of an OTS blob); §3 Merkle re-verify regression still green.
- [x] Quality-gate integrity — no `nolint`/`t.Skip`/build-tag/swallowed-error/deleted-assertion in any
  unpushed Go diff.

**Issues found:**
- (new, `normal`, Codex-confirmed) Certificate §5 does not bind the OTS proof's committed digest to §2's
  accepted root — see Codex triage below. Filed in `issues.md`.

**Codex second opinion:** One [P2] finding — "Verify OTS proof digest before rendering anchor"
(`handler.go:843-845`). **Confirmed real, filed `normal` (does not block).** Verified against the library
+ write path: `ots.Confirmed` only classifies the proof's attestations and never compares the parsed
`File.Digest` (32-byte SHA-256 the proof commits to, exposed by `opentimestamps@v0.4.0`) against §2's
`root`, so a row whose `ots_bytes` commit to a different digest would falsely render §2's root as
anchored. This is the always-loaded "gate a rendered ✓/anchor on re-VERIFICATION, not a classify-only
flag" rule applied to §5. NOT exploitable today: the production write path (`OTSTick`→Stamper→
`MarkOTSStamped`, Upgrader→`MarkOTSUpgraded`) always submits/upgrades the row's OWN `r.Root` digest, so a
mismatched row is unreachable — only a buggy `RecordOTS` (or the tests, which seed `hello-world.txt.ots`
against an arbitrary tree root for fixture convenience) produces one. Fix when §5/`ots.Confirmed` is next
touched: surface `File.Digest` and require `bytes.Equal(digest, root)` before `HasClause5=true`.

**Visual check:** SSR surface (`internal/certificate/cert.html`) screenshotted with agent-browser
(bundles its own browser; system Chrome absent but the CLI works). Rendered the landed confirmed
certificate (HTML captured from the §5 confirmed test, tokens.css inlined) and compared the §5 region
against `.dc.html:66`. Named-region affordances all present and matching: status dot + `block 358391 ·
<time>` + "OpenTimestamps. Run `ots verify` for the authoritative check." note, DS-token styled. One
minor cosmetic copy delta (not filed as a blocking issue — affordance is complete): the impl renders the
confirmation time as raw RFC-3339 (`2026-02-14T18:40:00Z`) where the mockup shows human-formatted
`2026-02-14 18:40 UTC`. agent-browser's batch `viewport`/`eval` subcommands are unavailable in this
build, so the below-the-fold §5 region was verified via the captured DOM/HTML + test assertions rather
than a scrolled screenshot; the above-fold §1-§2 layout matches the mockup grid.

**Next:** §5 closes the last numbered certificate clause. Best next observable Verify-closer toward
M-UI: the **separate comparison-anchor panel** on the certificate (target.md names Bitcoin-anchor AND
comparison-anchor as distinct, distinctly-labelled elements — §5 is the Bitcoin side; "anchoring" copy
stays Bitcoin-only). Alternatively the **dossier §4 Bitcoin-anchor** region (same `OTSForRoot` read
pattern, different surface). The `safeStamp` panic-recover + timeout guard (`normal` OTS issue) is the
highest-value non-UI hardening and should fold in the next time the stamp path is edited.

**Notes:**
- This increment is the Verify-closer the prior state demanded (§5, observable, HTTP-seam-tested) — no
  drift; the long internal-OTS-seam streak surfaced its second observable in a row.
- NOT DONE: OTS Verify keeps its offline-unprovable live-chain Bitcoin-confirmed half open; WASM
  verifier is 1/1 not started (no `internal/proof`, no `syscall/js`); the M-UI exit visual-pass + human
  sign-off (ADR-0012) has not been run. Loop = CONTINUE.
- The §5 read keys on §2's RAW `[]byte` root (not the base64 `CheckpointRoot` string); `seedOTS` records
  the same `tree.Hash()`/`tree.Size()` `AdvanceAccepted` committed, so the key aligns — the confirmed
  test passing proves it end-to-end.
- Open issues carried forward unchanged: `safeStamp` guard (`normal`), nil-Stamper+empty-row fall-through
  (`low`), `hubDomain` ForceQuery (`normal`), §4/bundle `host:port` DID `%3A`-encode (`normal`), §6 `· at`
  timestamp (`normal`), plus the existing `low` debt set. None touched by this diff.
- learnings/certificate.md net-rotated this iteration (added §5 + the digest-binding gap; collapsed
  settled §1/§3/§4 mechanics into git-history-backed one-liners) — landed at 159 lines, near budget.

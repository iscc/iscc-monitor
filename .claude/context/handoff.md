## 2026-06-22 — Light up certificate §5 BITCOIN ANCHOR (confirmed / pending) from the mirrored OTS row

**Done:** Certificate §5 BITCOIN ANCHOR now renders from the mirrored OpenTimestamps row of the §2
accepted root: a Bitcoin-confirmed proof shows the confirming block height (+ confirmation time), a
calendar-only proof shows the honest "pending" state, and an un-anchored root (no OTS row, the
empty-bytes sentinel, or an unparseable proof) omits §5 entirely — never an error. This closes the last
open numbered certificate clause (`HasClause5` was declared but never assigned `true`).

**Files changed:**
- `internal/certificate/handler.go`: imported `internal/ots`; added view-model fields `BTCConfirmed
  bool`, `BTCHeight int64`, `BTCConfirmedAt string`; added the §5 read in `buildData` inside the
  `if data.HasClause2` guard (after §4) — `st.OTSForRoot(ctx, hub.HubID, hub.LastSize, root)` keyed on
  the raw `[]byte` root, classifying a non-empty proof via `ots.Confirmed`; updated the package /
  `certData` / `buildData` docstrings for §5. (1 source file, within the ≤3 budget.)
- `internal/certificate/cert.html`: filled the empty §5 `clause-value` with a status dot + confirmed
  (`block N · <time>`) / pending markup + the "OpenTimestamps … run `ots verify`" note; added
  `.clause-status` / `.clause-dot` / `.clause-dot-confirmed` CSS (DS tokens only, no CDN).
- `internal/certificate/handler_test.go`: added `otsFixture`/`seedOTS` helpers and four §5 tests
  (Confirmed, Pending, Unanchored, EmptySentinel).
- `internal/certificate/testdata/`: new — `hello-world.txt.ots` (confirmed, oracle height 358391) and
  `merkle1.txt.ots` (calendar-only/pending), copied byte-identical from `internal/ots/testdata/`
  (hermetic; no cross-package testdata read at runtime).

**Verification:** `mise run check` → green (build + vet + test, all 23 packages). Per-criterion:
- [x] `go test -count=1 -run TestCertificate ./internal/certificate` passes (existing + 4 new §5 tests).
- [x] `TestCertificateBitcoinAnchorConfirmed`: certifiable id with a confirmed OTS row renders
  `§5 BITCOIN ANCHOR` + `block 358391` (the external `ots verify` oracle literal) + the RFC-3339
  confirmation time; §1-§3/§6 still render; no "pending" copy.
- [x] `TestCertificateBitcoinAnchorUnanchored`: certifiable id with NO OTS row renders the page WITHOUT
  the `§5 BITCOIN ANCHOR` marker; §1-§3+§6 unaffected.
- [x] `TestCertificateBitcoinAnchorPending` (extra): calendar-only proof renders the honest pending
  state, no `block ` literal.
- [x] `TestCertificateBitcoinAnchorEmptySentinel` (extra): a row with zero `OTSBytes` omits §5.
- [x] **Mutation (non-vacuous, reproducible):** forcing `data.HasClause5 = true` unconditionally inside
  the §5 `HasClause2` guard → `TestCertificateBitcoinAnchorUnanchored` FAILS (rendered §5 + the
  download bundle for the un-anchored fixture); reverting restores green. (To re-run: insert
  `data.HasClause5 = true` right after the `OTSForRoot` err-check in `buildData`.)
- [x] `gofmt -l .` (excl `cauldron/`) clean; `go mod tidy -diff` clean (no new prod dep — `internal/ots`
  and `internal/store` were already in the module).
- [x] WASM-purity guard: `GOOS=js GOARCH=wasm go build ./internal/didweb ./internal/index
  ./internal/badge` builds; `go list -deps` of each shows 0 hits on `internal/ots` (the non-WASM-pure
  closure did NOT leak into a WASM-shared package — certificate is server-side only).
- [x] `cmd/iscc-monitor/main.go`, `go.mod`, `go.sum` byte-unchanged (the §5 read uses the `st` the
  handler already holds; no new wiring).

**Next:** Remaining OTS / certificate closers, in priority order:
1. **Separate comparison-anchor panel** on the certificate (target.md names Bitcoin-anchor AND
   comparison-anchor as distinct, distinctly-labelled elements; "anchoring" copy is Bitcoin-only). §5 is
   the Bitcoin side; the comparison-anchor element is still unbuilt.
2. **`safeStamp` guard + nil-Stamper guard-order fix** (open `normal`+`low` OTS issues) — fold in BEFORE
   the stamp path (`internal/otsclient`/`internal/follower`) runs against a live calendar.
3. **Dossier §4 Bitcoin-anchor** region — a different surface, same OTS read pattern.
The "upgrades to Bitcoin-confirmed" live-chain half stays open offline (needs a live calendar + real BTC
confirmation; offline-unprovable). §5 renders whatever the mirrored row already holds.

**Notes:**
- Oracle gate: §5 reuses the already-mutation-proven `ots.Confirmed` (pinned to the OTS ecosystem's own
  bundled vectors; height 358391 is ground truth, not derived here). The certificate test asserts that
  oracle literal end-to-end through the rendered page, so the §5 surface is non-vacuously tied to the
  oracle. No signature/RFC-6962/Merkle/did:web/proof code was touched.
- Three honest fail-closed states implemented exactly per next.md: (1) miss OR empty-`OTSBytes` sentinel
  → no §5; (2) `ots.Confirmed` parse error → SILENT decline (never 500); (3) only a genuine `OTSForRoot`
  DB fault → 500 (buffer-then-200, like every other clause).
- The §5 read keys on §2's raw `[]byte` root (NOT the base64 `CheckpointRoot` string) — the confirmed
  test passing confirms the `(hubID, LastSize, tree.Hash())` key aligns with what `AdvanceAccepted`
  committed and what `seedOTS` records.
- The pending vector (`merkle1.txt.ots`) gives a real calendar-only `(false, 0, nil)` classification, so
  the pending branch is asserted against a genuine fixture, not just at the template-string level.
- Out of scope, untouched (per next.md Not-In-Scope): comparison-anchor panel, dossier §4, the
  `safeStamp` guard, the §4/bundle `did:web:host:port` `%3A`-encode bug, the proof-bundle `ots?` member,
  `main.go`. None fixed; the open issues carry forward unchanged.
- CSS: §5 uses DS tokens (`--iscc-lime-green` for the confirmed dot, `--radius-pill` for the circle,
  `--text-faint` for the pending/muted dot). No `--radius-full` token exists; used `--radius-pill`
  (9999px). No CDN/external URL added.

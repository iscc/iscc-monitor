## 2026-06-22 — Add the distinct comparison-anchor panel to the certificate (separate from §5 Bitcoin-anchor)

**Done:** Added a distinctly-labelled `COMPARISON ANCHOR` panel to the realm-wide Certificate of
Inclusion — the monitor's independently-observed record of the §2 accepted `(size, root)` this hub
showed THIS monitor, bounded by the coverage window — rendered as a SEPARATE element from §5, carrying
no "Bitcoin"/"anchoring"/"OpenTimestamps" copy. This closes the last open observable M-UI certificate
Verify element (target.md: Bitcoin-anchor and comparison-anchor are separate, distinctly-labelled).

**Files changed:**
- `internal/certificate/handler.go`: added `HasComparisonAnchor` + `HasCoverageWindow` gate flags and
  `CoverageSize uint64` / `CoverageSince string` fields to `certData`; populated them inside the existing
  `HasClause2` guard in `buildData` from `followedHub`'s `HubSummary.Coverage` (no new store read, reuses
  the already-encoded `CheckpointSize`/`CheckpointRoot`); updated the package doc, the `certData` doc, and
  the `buildData` step list (new step 9b). Flagged the mockup deviation in the field + package docstrings.
- `internal/certificate/cert.html` (template, not counted): rendered the `COMPARISON ANCHOR` clause between
  §5 and §6, reusing `.clause`/`.clause-mono`/`.clause-note` (page-scoped, no new CSS/CDN). Coverage copy
  is "since size N · <RFC-3339>" when the window has a time, "since size N" when time is NULL, and the
  honest "coverage just started" state when no window is set.
- `internal/certificate/handler_test.go` (test, not counted): added `fixtureStoreCovered` (sets the
  coverage start with a non-zero time via `SetCoverage` BEFORE `AdvanceAccepted`, the only way to get a
  coverage time given `AdvanceAccepted` writes a NULL time for a zero-`ObservedAt` record), the
  `comparisonAnchorPanel` slicer, and three tests: `TestCertificateComparisonAnchor` (label + window +
  no-Bitcoin-copy-in-panel), `TestCertificateComparisonAnchorIndependentOfOTS` (no §5 OTS row → §5 absent
  but comparison anchor present — the two are decoupled), `TestCertificateComparisonAnchorCoverageJustStarted`
  (NULL coverage time → window states size only, no since-time chip).

**Verification:** `mise run check` → green (build + vet + test, all packages). Per criterion:
- `go test -count=1 -run TestCertificate ./internal/certificate` — PASS (existing suite + 3 new tests).
- `TestCertificateComparisonAnchor` — asserts `COMPARISON ANCHOR` label + `size 24000` + the RFC-3339
  coverage-since + "detect a split view" affordance; slices the panel and asserts it contains NONE of
  `Bitcoin`/`anchoring`/`OpenTimestamps`/`BITCOIN ANCHOR`/`ots verify` — proving the two anchor panels are
  separate, distinctly-labelled elements. PASS.
- `TestCertificateComparisonAnchorIndependentOfOTS` — hub with no OTS row renders no §5 but DOES render the
  comparison anchor (the two are decoupled). PASS.
- `TestCertificateComparisonAnchorCoverageJustStarted` — NULL coverage time → panel present, "since size
  24816", no time chip (honest no-window state, ADR-0001). PASS.
- **Mutation (non-vacuous, reproduced):** forcing `data.HasComparisonAnchor = false` → all three new tests
  FAIL (no `COMPARISON ANCHOR` marker); reverting → green. `handler.go` confirmed restored to `= true`.
- `gofmt -l internal/ cmd/` empty; `go mod tidy -diff` clean (no new prod dep).
- WASM-purity guard — `GOOS=js GOARCH=wasm go build ./internal/didweb ./internal/index ./internal/badge`
  builds (certificate stays server-side-only; no leak into a WASM-shared package).

**Next:** With all six numbered clauses + both anchor panels (Bitcoin + comparison) now present, the
certificate's observable M-UI Verify surface is complete. The next M-UI closer toward milestone exit is the
**dossier §4 Bitcoin-anchor** region (same `OTSForRoot` read pattern, different surface — see the prior
review's `Next:`), or a dossier comparison-anchor equivalent. The standing non-UI hardening items remain:
the §5 digest-binding (`bytes.Equal(File.Digest, root)`), the §4/bundle `host:port` DID `%3A`-encode, and
the §6 `· at` timestamp — fold each in the next time that exact line is edited. The **WASM verifier** (1/1
open) and the **M-UI exit visual-pass + human sign-off** (ADR-0012) are the remaining milestone gates.

**Notes:**
- **Scope:** 1 prod file of substance (`handler.go`), within the ≤3 budget. No store schema / new query —
  the coverage window rides out of the existing `followedHub` `ListHubs` scan (`HubSummary.Coverage`); §2
  already loaded the `(size, root)`. No new error path: the panel renders inside `HasClause2` and cannot
  500 on its own. None of the earmarked `normal` fixes were touched (the comparison anchor adds a new clause
  and edits none of those lines), per next.md's "keep the diff tight".
- **Mockup deviation (design-parity flag):** the certificate mockup (`.dc.html`) has NO comparison-anchor
  element — target.md mandates it beyond the mockup. Flagged in the package doc + the `HasComparisonAnchor`
  field docstring (one evergreen sentence each), not silently dropped. Placed as its own numbered-clause row
  with a distinct `COMPARISON ANCHOR` heading (never "§5"/"BITCOIN ANCHOR").
- **Coverage-time fixture nuance:** `AdvanceAccepted` writes `monitored_since_size` with a NULL time for a
  zero-`ObservedAt` record, so the existing fixtures yield `Coverage.Set == true` with a zero `Since`. The
  new `fixtureStoreCovered` calls `SetCoverage` with an explicit time BEFORE `AdvanceAccepted` (whose
  set-once UPDATE then no-ops) to seed a real coverage time — the test covers both the with-time and
  NULL-time paths.
- **Glossary discipline:** the panel copy uses "Comparison anchor" framing (the monitor's own observation,
  the split-view check), never "witness" (reserved for the deferred M7 cosigner role) and never any
  "anchoring"/Bitcoin lexicon (verified by the panel-slice assertion). It reads as the monitor's account
  (Tier 1, verifiable cache), consistent with the existing two-tier honesty panel.
- **Oracle/conformance gate — N/A:** this diff touches no signature/RFC-6962/Merkle/proof/did:web code —
  the comparison anchor reuses §2's already-read `(size, root)` and the coverage window, no crypto path.
  §3's Merkle re-verify regression still green.

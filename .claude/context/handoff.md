## 2026-06-21 — Certificate-of-inclusion skeleton — realm-wide `/inclusion/{iscc_id}` page (§1 Subject + decode→resolve chain)

**Done:** Stood up a new `internal/certificate` package serving `GET /inclusion/{iscc_id}`: it
decodes the self-describing ISCC-IDv1 (`index.Decode`), resolves the issuing hub's domain via the
Hub-List (`registry.HubList.Resolve`), finds that hub's store row (`ListHubs`), and looks up the id's
indexed leaf seqs (`SeqsForISCCID`, ADR-0008), then renders an Evidence-Ledger certificate whose §1
SUBJECT clause + subject banner (subject id, resolved hub domain, position seqs[0]) are real. Every
malformed / unresolvable / not-followed / not-in-log id renders the documented "cannot certify" 200
state, never a 5xx. This is the first real caller wiring both the decoder and the resolver.

**Files changed:**
- `internal/certificate/handler.go` (new): the realm-wide handler + the decode→resolve→store-lookup
  chain (`buildData`), fail-closed (every cannot-certify branch is a 200; only a DB/template fault is a
  500, detected via buffer-then-200). Accepts `(*registry.HubList, *store.Store, StatusSource)`;
  `StatusSource` taken for forward-compat wiring (not consulted by the §1 skeleton). Exports
  `PathPrefix = "/inclusion/"`.
- `internal/certificate/cert.html` (new): embedded DS-shell template (dossier shell verbatim — `/_ds/`
  links, self-hosted fonts, NO CDN body; unquoted `[data-status=…]` n/a since no badge inlined yet),
  rendering the mockup landmarks: chrome + `verify ↗ monitor.iscc.codes` tier-2 link, `← Realm index`
  back-link, certificate head, subject banner, §1 SUBJECT clause, two-tier honesty panel, disabled
  "Download proof bundle (coming soon)" placeholder, static "Verify independently →" link, "Cite as /
  verifiable cache" footer. §2–§6 are `{{if .HasClauseX}}` placeholders that render nothing.
- `cmd/iscc-monitor/main.go` (modify): mount `certificate.Handler` at the `/inclusion/` subtree in
  `buildMux`; build the interim `*registry.HubList` from realm entries (slot i = entry i,
  `hubListFromEntries`) — no config/env change, matches the testnet fixture (sb0=slot0, sb1=slot1).
  `buildMux` / `serveMetrics` gained a `*registry.HubList` param.
- `CLAUDE.md` (docs): added the `GET /inclusion/<iscc_id>` route to the HTTP-surface bullet list.
- `internal/certificate/handler_test.go` (new test): golden HTTP-seam tests (see Verification).
- `cmd/iscc-monitor/main_test.go` (test): `TestCertificateRouteMounted` (route mounted + interim
  Hub-List wired end-to-end through `buildMux`); updated the 6 existing `buildMux` calls for the new param.

**Verification:** `mise run check` → green (build + vet + all 22 packages, including the new
`internal/certificate`; `gofmt -l .` empty). Per-criterion:
- Known-id golden chain — `TestCertificateKnownID`: `GET /inclusion/MAIGHFECJMOPMIAB` → 200 text/html;
  body has the id, the **resolved** `sb1.amlet.id`, position 24815, `§1 SUBJECT`, `← Realm index`,
  `monitor.iscc.codes`, the two-tier panel, the `/_ds/` links, no third-party CDN host. PASS.
- Unknown-id (resolves, not in log) — `TestCertificateNotInLog` → 200 "not found in log". PASS.
- Malformed-id — `TestCertificateMalformedID` (`NOTANISCCID`) → 200 "not a valid ISCC-ID". PASS.
- Unresolvable slot — `TestCertificateUnresolvableSlot` (`MEIGHFECJMOPMIAC`, hub_id 2) → 200 "not found
  in this realm". PASS. (Plus `TestCertificateResolvedButNotFollowed`, `TestCertificateEmptyID`,
  `TestCertificateNilHubList`.)
- Method guard — `TestCertificateNonGET` → 405. PASS.
- Non-vacuity — `TestCertificateResolvedDomainTracksHubList`: remapping slot 1 → sb0.iscc.id renders
  sb0.iscc.id (not sb1), proving Resolve is load-bearing. I also reproduced two mutations: (a)
  `Resolve(id.HubID+1)` → `TestCertificateKnownID` + `TestCertificateRouteMounted` FAIL; (b) bypassing
  the `Decode` error → `TestCertificateMalformedID` FAILS. Tree restored, re-run green.

**Next:** The same arc continues with the §2/§3 clause sub-step (Checkpoint + Inclusion proof). That
is where the **oracle/conformance gate RE-ENGAGES** — `define-next` should call it out: the §3 inclusion
proof reuses `proofserve`'s `InclusionProofFromTiles` against the accepted tree (LastSize / CheckpointAt)
and the reviewer must mutation-prove the served proof non-vacuous (corrupt a hash / bump leafIndex → the
test FAILS), with `notecheck`/golden-vector parity considered. The §2/§3 slice is also the natural home
for the deferred `ForceQuery` fail-open fix in `registry.go` (it consumes the resolved domain). The
proof-bundle assembler + the Download action remain a distinct later oracle-gated step.

**Notes:**
- **Interim Hub-List wiring (flag for review):** `hubListFromEntries` maps realm.txt order to slots
  (slot i = entry i). Production has no real Hub-List document path yet; `realm.txt` is line-based
  domains, NOT the YAML Hub-List. This is the KISS interim called for in `next.md` (no new env var) and
  matches the testnet fixture exactly. A real Hub-List source is its own decision; documented with a
  TODO in `hubListFromEntries`. The `*registry.HubList` is hand-built from the exported `HubList`/`Hub`
  shape (`HubID *uint16`), so `registry.go` and `internal/config` are untouched.
- **Scope:** 3 non-test/non-doc files (handler.go + cert.html created, main.go modified); CLAUDE.md is
  docs; two `_test.go` are tests. Nothing from `## Not In Scope` touched: no proof-bundle assembler
  (button is a disabled placeholder), §2–§6 are empty gated placeholders, the `ForceQuery` fix is left
  for the §2/§3 slice, no ADR-0011 Go-1.26 bump (local go1.24.13), no WASM tier-2 result, no config field.
- **Oracle/conformance gate correctly N/A here:** pure HTML render of decode + registry resolve + a
  store `SeqsForISCCID` lookup — no signature / RFC-6962 / Merkle / proof / did:web path, no new crypto,
  `go.mod`/`go.sum` byte-unchanged. It APPLIES to the §3 proof sub-step (see Next).
- **Resolved-domain link discipline:** §1 derives `<domain>` from the real resolve, never hardcoded;
  later clauses linking into `/<domain>/log/...` should derive Origin the same way ("Origin =
  `<domain>/log`", learnings index).
- No new dependencies; `internal/certificate` imports only `index`, `registry`, `store` + stdlib.

## 2026-06-22 — Review of: Percent-encode the port when building a hub's `did:web:` DID on certificate §4 and the proof bundle

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** The advance added a two-line local pure helper `didWeb(domain)` to
`internal/certificate/handler.go` that `%3A`-encodes the FIRST colon (the resolver's exact idiom,
`strings.Replace(domain, ":", "%3A", 1)`) and routed BOTH DID-building sites — the §4 `SigningKeyDID`
and the proof-bundle `Hub.DID` — through it, so a `host:port` hub now names the same host its key
resolved from while a no-port domain round-trips byte-identical. Scope is exemplary: exactly one
production file (`handler.go`) + two test files, no new imports, nothing from `## Not In Scope` touched.
Both sites are mutation-proven; the §4 + bundle DID `host:port` `normal` is fully closed.

**Verification:**
- [x] `mise run check` — green, all 27 packages build + vet + test.
- [x] `gofmt -l .` — empty (clean).
- [x] `mise exec -- go test -count=1 -run 'TestCertificateSigningKey|TestCertificateProofBundle' ./internal/certificate` — `ok`.
- [x] §4 mutation: reverting `data.SigningKeyDID = didWeb(data.Domain)` → `"did:web:" + data.Domain` FAILS `TestCertificateSigningKeyDIDPortEncoded`; restored byte-identical (`git diff` empty).
- [x] Bundle mutation: reverting `DID: didWeb(data.Domain)` → `"did:web:" + data.Domain` FAILS `TestCertificateProofBundleDIDPortEncoded` (`bundle hub.did = "did:web:localhost:8443", want "...%3A8443"`); restored byte-identical.
- [x] Clean-domain regression: `did:web:sb1.amlet.id` still renders exactly on §4 (`…DIDCleanDomain`) and bundle (`TestCertificateProofBundle`) — no spurious encoding.
- [x] Correctness vs the resolver: `didWeb` → `did:web:host%3Aport`, which `didweb.DocumentURL` `url.PathUnescape`s back to a single `host:port` host segment → `https://host:port/.well-known/did.json` — the same host the key resolved from. Round-trip verified against `internal/didweb/url.go`.
- [x] No remaining unencoded `"did:web:" + domain` concat sites in production code (grep); the only other site, `logclient/didresolve.go:102`, already used the idiom.
- [x] Gate-circumvention scan over all 3 unpushed commits (`@{upstream}..HEAD`) — no `//nolint`, `t.Skip`, build-tag exclusion, deleted assertion, or swallowed error in added lines. Diff is purely additive.
- [x] Conformance/oracle gate — N/A: render-string fix only, touches no signature/Merkle/proof-verify/did:web-derivation/fork-shrink-equivocation code. `internal/certificate` is server-side only (no WASM-purity concern).
- [x] Test-fixture review: `fixtureStoreTiled`'s new `switch` keeps sb0/sb1 callers byte-identical (explicit `case`s) and the `default` branch genuinely drives a `host:port` domain through the real `UpsertHub`→Resolve→followedHub chain (resolved from `hostPortHubList` slot 1, not hardcoded) — non-tautological.

**Issues found:** (none new). Resolved + deleted the `normal` "Certificate §4 AND the proof bundle build `did:web:` + raw domain" issue after mutation-verifying both sites are encoded.

**Codex second opinion:** Clean — "The change consistently routes both certificate DID render sites through the same host:port encoding behavior used by the resolver, and the added regression coverage exercises both the HTML and bundle paths. I found no introduced correctness issues." Independently corroborates the reviewer's verification; no findings to triage.

**Visual check:** n/a — no visual-region/layout/chrome/affordance change. The diff alters only the rendered `did:web:` STRING VALUE, and only for a `host:port` hub, which no testnet/live fixture produces; the served certificate page is byte-identical for clean domains. (`agent-browser` present but Chrome not found in this environment — moot, as there is no delta to capture.)

**Next:** Code-closable `normal`s on the certificate/registry surface remain, in handoff-named order: (1) the `hubDomain` ForceQuery fail-open in `internal/registry/registry.go` (add `|| u.ForceQuery` to the line-188 reject — separate file, same fail-quietly-on-clean-fixtures class); (2) the §5 OTS-digest-binding `bytes.Equal` gap; (3) the tier-2 honesty-copy overstatement (`cert.html:465`). The §6 `· at` timestamp needs a store schema column (larger). Otherwise the front-of-queue **WASM-verifier signature half** (state.md "Next Milestone") is design-first / STOP-candidate (browser did:web resolution) — do a design pass first; do NOT loosen `verifier.html`'s "hub-signed root" copy.

**Notes:**
- KISS held perfectly: a two-line local helper, no shared export, no `didweb` refactor (exactly `## Not In Scope`). The choice of `strings.Replace(..., 1)` over `url.PathEscape` is correct — escaping the whole domain would over-encode `.` in hostnames; only the single port colon is load-bearing.
- The remaining milestone Verify criteria are human-blocked (Pages repo-Settings custom-domain enablement, `normal`) or offline-unprovable (OTS Bitcoin-confirmed half needs a live calendar + real BTC confirmation), so the loop continues to close productive `normal`s.
- 3 commits ahead of `origin/develop`; pushing on PASS. The known `Pages` workflow failure on develop is the human-blocked custom-domain step, not a code regression.

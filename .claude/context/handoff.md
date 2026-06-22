## 2026-06-22 — Percent-encode the port when building a hub's `did:web:` DID on certificate §4 and the proof bundle

**Done:** Added a tiny local pure helper `didWeb(domain)` in `internal/certificate/handler.go` that
`%3A`-encodes the FIRST colon (`host:port` → `did:web:host%3Aport`, reusing the resolver's
`strings.Replace(domain, ":", "%3A", 1)` idiom) and routed BOTH DID-building sites through it — the
proof-bundle `bundle.Hub.DID` and the §4 `data.SigningKeyDID`. A `host:port` hub now names the same
host the signing key resolved from on both Tier-1 self-verifiable surfaces; a no-port domain
round-trips byte-identical.

**Files changed:**
- `internal/certificate/handler.go`: new local `didWeb(domain)` helper after `PathPrefix`; both
  `"did:web:" + data.Domain` sites (bundle ~line 565, §4 ~line 886) now call `didWeb(data.Domain)`;
  the §4 `SigningKeyDID` field docstring and the `bundleHub` docstring updated to state the port is
  `%3A`-encoded (evergreen). No new import (`strings` already present).
- `internal/certificate/handler_test.go`: generalized `fixtureStoreTiled` to register a non-testnet
  `indexDomain` as its own hub (default branch, byte-identical for sb0/sb1 callers); added
  `hostPortHubList()` (slot 1 → `https://localhost:8443`); added `TestCertificateSigningKeyDIDPortEncoded`
  (§4 renders `did:web:localhost%3A8443`, NOT `did:web:localhost:8443`) and
  `TestCertificateSigningKeyDIDCleanDomain` (no-port `did:web:sb1.amlet.id` unchanged).
- `internal/certificate/bundle_test.go`: added `TestCertificateProofBundleDIDPortEncoded` (the served
  `.bundle` `Hub.DID` is `did:web:localhost%3A8443` while `hub.domain` stays verbatim `localhost:8443`).

**Verification:** `mise run check` → green (all 27 packages build + vet + test; `gofmt -l .` empty).
- [x] `mise exec -- go test -run 'TestCertificateSigningKey|TestCertificateProofBundle' ./internal/certificate` → `ok`.
- [x] `host:port` asserts `did:web:host%3Aport` on BOTH §4 and the bundle — **PASS**.
- [x] Mutation, §4 site: reverting `didWeb(data.Domain)` → `"did:web:" + data.Domain` fails
  `TestCertificateSigningKeyDIDPortEncoded`; restored byte-identical — **PASS**.
- [x] Mutation, bundle site: same revert fails `TestCertificateProofBundleDIDPortEncoded`
  (`hub.did = "did:web:localhost:8443"`); restored byte-identical — **PASS**.
- [x] Clean-domain regression: `did:web:sb1.amlet.id` still renders exactly (no spurious encoding) on
  §4 (`…DIDCleanDomain`) and the bundle (existing `TestCertificateProofBundle` line 92) — **PASS**.
- Conformance/oracle gate: N/A — render-string fix only, touches no signature/Merkle/proof-verify/
  did:web-derivation/fork-shrink-equivocation code (`next.md` confirmed). `internal/certificate` is
  server-side only, so no WASM-purity concern.

**Next:** The handoff-named code-closable `normal`s on the certificate/registry surface remain: the
`hubDomain` ForceQuery fail-open in `internal/registry/registry.go` (same fail-quietly-on-clean-fixtures
class, separate file), the §5 OTS-digest-binding `bytes.Equal` gap, the §6 `· at` timestamp (needs a
store schema column), and the tier-2 honesty-copy overstatement. Otherwise resume the front-of-queue
**WASM-verifier signature half** (state.md "Next Milestone") — flagged **design-first / STOP-candidate**
(browser-side did:web resolution); do a design pass first and do NOT loosen `verifier.html`'s
"hub-signed root" copy. The remaining milestone Verify criteria are human-blocked (Pages
repo-Settings) or offline-unprovable (OTS Bitcoin-confirmed).

**Notes:**
- KISS held: the helper is two lines, local to `handler.go` (no new shared export / no `didweb`
  refactor, per `## Not In Scope`). Did NOT `url.PathEscape` the whole domain (would over-encode `.`
  in hostnames); `strings.Replace(..., 1)` replaces only the load-bearing port colon.
- The one test-fixture change worth review: `fixtureStoreTiled` gained a `default` switch branch so a
  non-testnet `indexDomain` registers its own hub. Existing sb0/sb1 callers are byte-identical
  (explicit `case`s); the new branch is exercised only by the two host:port tests. This was the
  minimal way to drive a `host:port` domain through the real decode→Resolve→followedHub chain (the
  `host:port` is genuinely resolved from `hostPortHubList`'s slot-1 URL, not hardcoded in the page).
- IPv6-literal hosts are out of scope (the realm fixture is hostnames, per `next.md`); only the single
  `host:port` colon is load-bearing.
- The other two surfaces from `## Not In Scope` (registry ForceQuery, §5 OTS digest) were left
  untouched to keep this step to one production file.

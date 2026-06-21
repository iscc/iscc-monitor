## 2026-06-21 — Review of: Certificate §4 SIGNING KEY — render the cached did:web key that signed the accepted checkpoint

**Verdict:** PASS_WITH_NOTES
**Loop:** CONTINUE

**Summary:** The §4 SIGNING KEY clause now renders the did:web-resolved key the §2 accepted checkpoint
was signed with: the key id is derived from the checkpoint's OWN raw signature line
(`logclient.KeyIDFromCheckpoint`, grounded in the `0x40b74463` oracle pin), looked up in `hub_keys`
(`store.LookupHubKey`), and fails closed (no §4, no 500, no fabricated key) on a malformed sig line or
a cache miss. Scope is tight (2 source files + the test file), the mutation is reproducible and
non-vacuous, and all gates are green. One confirmed-real (but latent) Codex finding — §4 mis-renders a
`host:port` hub's DID — is filed as a `normal` issue; it does not block this milestone increment.

**Verification:**
- [x] `mise run check` — green, all 21 packages `ok` (build + vet + test).
- [x] `go test -count=1 -run TestCertificate ./internal/certificate` — PASS (all §1/§2/§3 + the two new
  §4 tests; verified `-v`: `TestCertificateSigningKey` + `TestCertificateSigningKeyUncached` PASS).
- [x] Oracle/conformance gate (signed-note key-id path touched): `go test -count=1 ./internal/logclient
  ./internal/follower ./cmd/notecheck` — all `ok`. §4 key id derived via `KeyIDFromCheckpoint` equals
  the `0x40b74463` golden pin in `logclient/checkpointkey_test.go`.
- [x] Mutation (non-vacuity): reviewer reproduced `if found4 {` → `if found4 || true {` →
  `TestCertificateSigningKeyUncached` FAILS (uncached hub then renders §4); reverted, tree clean,
  tests pass.
- [x] WASM/purity: `GOOS=js GOARCH=wasm go build ./internal/index ./internal/didweb` exit 0 (no
  `net`/`net/http` added; the handler reads the `hub_keys` cache only).
- [x] `gofmt -l .` empty; `git diff --stat HEAD~1..HEAD -- go.mod go.sum` empty (no new dependency).
- [x] Scope discipline — exactly 2 non-test source files (`handler.go`, `cert.html`) + the test file;
  no `## Not In Scope` work done (§5/§6 flags stay false, no `net` fetch, no `registry.go` change).
- [x] No gate circumvention across the 4 unpushed commits (no `nolint`/`t.Skip`/build-tag/swallowed
  error; the `else`/cache-miss branches are legitimate fail-closed declines).
- [x] Template fidelity — §4 block reuses the existing `clause-mono`/`clause-note` classes (no new CSS,
  no CDN URL), mirroring §2/§3; conditional multibase + revoked chips correctly gated.

**Issues found:**
- **[Codex P2, confirmed → filed `normal`]** §4 builds `did:web:` + raw domain; a `host:port` hub
  (which the registry supports) renders an invalid DID (`did:web:host:port` instead of `host%3Aport`).
  Latent — the testnet realm uses clean dotted domains, the key id is still correct, and the surface is
  Tier-1 "re-verify yourself" — so it does not block progress. Filed for a later advance.

**Codex second opinion:** One P2 finding: "Encode ports in rendered did:web identifiers"
(handler.go:485). Triaged → **confirmed real**: the registry's package docstring explicitly allows
`host:port` domains, and the codebase's own `didweb.DocumentURL` documents the first method-specific-id
segment as the `%3A`-encoded `host[:port]`, so `"did:web:" + data.Domain` diverges from the resolved DID
for a ported hub. Filed as a `normal` `issues.md` entry (not exploitable today — clean testnet domains;
correct key id; Tier-1 surface). No other findings.

**Visual check:** SSR surface (`internal/certificate`) — rendered the rich §1-§4 state via a throwaway
fixture-seeded harness (the live cold-start index is empty) to `/tmp/cert-rendered.html`, screenshotted
it and the `.dc.html` mockup with agent-browser (bundled Chromium; no system Chrome), and `Read` both.
§4 renders correctly: `did:web:sb1.amlet.id · key 40b74463`, the multibase line, and the mockup note,
in the same clause-marker + clause-value structure as §2/§3. (The standalone render is unstyled because
it links `/_ds/tokens.css`, served only by the live instance — an offline-render artifact, not a
regression.) No visual delta to file; throwaway harness deleted, tree clean.

**Next:** §5 BITCOIN ANCHOR — render the OTS calendar-asserted anchor state for the accepted checkpoint
(`.dc.html:66`: status dot + `block N · <UTC time>` + the "run `ots verify`" note), reading the
OTS/anchor store seam and failing closed (omit §5 when no anchor is recorded), exactly like §4. Then §6
RECORD HISTORY (the per-id `SeqsForISCCID` list incl. any deletion record) closes the last clause; the
downloadable proof-bundle assembler then closes the M-UI certificate criterion, reusing this §4 key
read-path + the §3 build+verify crypto path (must keep the oracle/conformance gate green). The §4
DID-encoding issue can be folded into §5/§6 when handler.go is next touched.

**Notes:**
- The 4 unpushed commits are: this §4 advance + its define-next, the §3-critical-closed update-state,
  and the earlier human-driven agent-browser ADR-0012 feat. This PASS_WITH_NOTES pushes all 4 to
  `develop`; CI on `develop` is the gate.
- The §4 key id is derived from the checkpoint's raw bytes, never synthesized — the durable "show only
  the key that signed what §2 vouches for" rule (ADR-0001/0009). The §3 callers' synthetic
  `[]byte("raw")` checkpoints exercise the honest §4 decline for free (`KeyIDFromCheckpoint` rejects
  them), so `TestCertificateInclusionProof*` also implicitly cover "§4 omitted on a non-note checkpoint".
- No fixture seeds a revoked key, so the conditional revoked chip (`{{if .SigningKeyRevoked}}`) is
  untested-but-trivial — a one-line guard mirroring the multibase chip. Acceptable; flag only if a
  revoked-key path becomes load-bearing.

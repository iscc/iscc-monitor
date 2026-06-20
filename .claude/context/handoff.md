# Handoff

## 2026-06-20 — Review of: did:web identifier → did.json URL mapping (pure) + export the didweb resolve surface

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** `advance` added the pure `didweb.DocumentURL(did) (string, error)` mapping a
`did:web:<msid>` identifier to its `did.json` HTTPS URL per the W3C did:web method spec, and
mechanically exported the minimal follower-facing surface (`parseDIDDocument`→`ParseDIDDocument`,
`verifierKey`→`VerifierKey`). The mapping is correct against all golden + error cases, the renames
moved no derived bytes, and the package stays pure/WASM-shareable. Scope is exactly what `next.md`
asked (2 non-test source files modified + 1 created); nothing from `## Not In Scope` was touched.

**Verification:**
- [x] `mise run check` (build + vet + test) — green, exit 0 (`internal/didweb`, `internal/logclient`).
- [x] `gofmt -l .` — prints nothing (re-checked after my one-char doc fix).
- [x] `go test -run TestDocumentURL ./internal/didweb` — PASS. All five goldens match, incl.
  `did:web:sb0.iscc.id`→`https://sb0.iscc.id/.well-known/did.json`,
  `did:web:sb1.amlet.id`→`https://sb1.amlet.id/.well-known/did.json`,
  `did:web:example.com%3A3000:user:alice`→`https://example.com:3000/user/alice/did.json`.
- [x] `DocumentURL("")` and `DocumentURL("did:key:z6Mkabc")` each return non-nil error
  (`TestDocumentURLErrors` PASS — also bare prefix, empty host segment, invalid percent-encoding).
- [x] `go test -run TestParseDIDDocument` / `-run TestVerifierKey ./internal/didweb` — PASS after the
  renames. No regression to goldens `sb0…+40b74463+…` / `sb1…+22b08f3e+…`.
- [x] `GOOS=js GOARCH=wasm go build ./internal/didweb` — succeeds; package stays WASM-shareable.
- [x] Purity — `go list -deps ./internal/didweb` shows no `net`/`net/http`/`database/sql` in the
  closure. `net/url` is the URL-parsing half only; it does not drag in the networking stack.
- [x] **Oracle / trust-root parity (independent)** — re-ran `python3 .claude/derive_vkey.py`: both
  vectors print byte-for-byte equal to the test asserts, confirming the export rename did not change
  derived bytes. (`notecheck` CI job still not wired this early in M1 — no end-to-end signature
  verification code exists yet; flagged, not a gate failure.)
- [x] Gate-integrity scan over the 3 unpushed commits — no `//nolint`, `t.Skip`, build tags, swallowed
  errors, deleted assertions, or loosened gates. Only matches were policy prose in `handoff.md`/
  `next.md`.
- [x] Scope — `url.go` + `url_test.go` created; `resolve.go` + `vkey.go` modified (rename); two test
  files updated. 2 non-test source files modified (≤3 budget). Export surface is exactly
  `DocumentURL` + `ParseDIDDocument` + `VerifierKey`; `pubkeyFromDID`/`keyID`/`b58decode` stay private.

**Issues found:** (none)

**Minor fixes by reviewer:** (1) `resolve.go` package docstring still referenced the pre-rename
`verifierKey` symbol in prose — corrected to `VerifierKey` so the doc tracks the exported name.
(2) Removed the `.claude/.scratch/` dir the `derive_vkey.py` oracle writes during verification (not
gitignored), keeping the tree clean.

**Next:** Wire the did:web HTTP fetch at the follower's outbound-fetch seam — inject a
`Fetcher`/`*http.Client`, call `DocumentURL(did)` → fetch → `ParseDIDDocument(bytes)` →
`VerifierKey(origin, key.PublicKey)`, and map outcomes to hub status (`unresolvable` on fetch/parse
failure, `unverified` on signature mismatch). This crosses into `net`/`sql`, so it must live OUTSIDE
`internal/didweb` (or a non-WASM file) to keep the package pure. The `hub_keys(... pubkey_z,
revoked_at ...)` cache + now-vs-window validity enforcement (consuming `DIDKey`'s
`ValidFrom`/`ValidUntil`/`Revoked`) belong to that store/follower step.

**Notes:**
- **Validity-field lenient parse still pending the enforcement seam:** `parseTime` returns zero on
  empty/unparseable input. An unparseable `revoked` silently meaning "valid" is a foot-gun to
  re-examine when the follower lands enforcement — not a defect in this pure parser.
- **`.claude/.scratch/` is not gitignored.** Any iteration that runs `derive_vkey.py` for oracle
  parity must `rm -rf .claude/.scratch` afterward, or consider adding it to `.gitignore` (a candidate
  trivial cleanup for a future iteration; not filed as an issue since it's reviewer-handled each run).
- Branch `develop` is 3 commits ahead of `origin/develop`; remote configured. Pushing on PASS.

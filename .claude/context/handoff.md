# Handoff

## 2026-06-20 — did:web identifier → did.json URL mapping (pure) + export the didweb resolve surface

**Done:** Added the pure `didweb.DocumentURL(did string) (string, error)` that maps a
`did:web:<method-specific-id>` identifier to its `did.json` HTTPS URL per the W3C did:web method spec
(colon-separated segments, percent-decoded, `/.well-known/did.json` vs `/<path>/did.json`). Promoted
the minimal resolve surface across the package boundary by mechanically renaming
`parseDIDDocument`→`ParseDIDDocument` and `verifierKey`→`VerifierKey`; `pubkeyFromDID`/`keyID`/
`b58decode` stay unexported. No `net/http`, no fetch, no status mapping (deferred to the next step).

**Files changed:**
- `internal/didweb/url.go` (new): pure `DocumentURL` mapping; imports only `fmt`, `net/url`
  (`PathUnescape`), `strings`. WASM-shareable.
- `internal/didweb/resolve.go`: `parseDIDDocument`→`ParseDIDDocument` (signature, docstring, three
  wrapped-error prefixes `parseDIDDocument:`→`ParseDIDDocument:`). Logic unchanged.
- `internal/didweb/vkey.go`: `verifierKey`→`VerifierKey` (signature + docstring). Derivation bytes
  unchanged.
- `internal/didweb/url_test.go` (new): table-driven golden (sb0, sb1, port+path, path-only,
  port-only) + error tests (empty, wrong method, bare prefix, empty host, bad percent-encoding).
- `internal/didweb/resolve_test.go`, `internal/didweb/vkey_test.go`: updated call sites/comments to
  the exported names.

**Verification:** `mise run check` (build + vet + test) → green, exit 0
(`internal/didweb`, `internal/logclient`). `gofmt -l .` → prints nothing. Per-criterion:
- [x] `go test -run TestDocumentURL ./internal/didweb` PASS; all five golden cases match, including
  `did:web:sb0.iscc.id`→`https://sb0.iscc.id/.well-known/did.json`,
  `did:web:sb1.amlet.id`→`https://sb1.amlet.id/.well-known/did.json`, and
  `did:web:example.com%3A3000:user:alice`→`https://example.com:3000/user/alice/did.json`.
- [x] `DocumentURL("")` and `DocumentURL("did:key:z6Mkabc")` each return non-nil error (plus bare
  prefix, empty host, invalid percent-encoding).
- [x] `go test -run TestParseDIDDocument` / `-run TestVerifierKey` still PASS after the renames — no
  regression to the golden vectors `sb0…+40b74463+…` / `sb1…+22b08f3e+…`.
- [x] `GOOS=js GOARCH=wasm go build ./internal/didweb` succeeds — package stays WASM-shareable.
- [x] Oracle parity: re-ran `python3 .claude/derive_vkey.py` — both vectors print byte-for-byte equal
  to the test asserts; the rename did not change derived bytes. Removed the `.claude/.scratch/` the
  oracle writes (not gitignored).

**Next:** Wire the did:web HTTP fetch at the outbound-fetch seam (follower step): inject a
`Fetcher`/`*http.Client`, call `DocumentURL(did)` → fetch → `ParseDIDDocument(bytes)` →
`VerifierKey(origin, key.PublicKey)`, and map outcomes to hub status (`unresolvable` on fetch/parse
failure, `unverified` on signature mismatch). The `hub_keys(... pubkey_z, revoked_at ...)` cache and
now-vs-window validity enforcement (consuming the already-surfaced `ValidFrom`/`ValidUntil`/`Revoked`
from `DIDKey`) belong to that store/follower step. Note that step crosses into `net`/`sql`, so it
must live OUTSIDE `internal/didweb` (or in a non-WASM file) to keep this package pure.

**Notes:**
- **W3C did:web mapping ported from the method-spec rule in `next.md`** (no usable copy in
  `cauldron/`). Decode rule: first colon-segment is `host[:port]`, rest are path; each segment
  `url.PathUnescape`d; no path → `/.well-known/did.json`, path → `/<segs>/did.json`. The two live
  hubs have no path and no port, so they hit the `.well-known` branch.
- **Export surface widened to exactly `DocumentURL` + `ParseDIDDocument` + `VerifierKey`** (YAGNI per
  `next.md`); `pubkeyFromDID`/`keyID`/`b58decode` remain package-private in-package helpers.
- **Validity-field lenient parse still stands as flagged in the prior handoff**: `parseTime` returns
  zero on empty/unparseable input; the meaning of an unparseable/expired value is the follower's
  decision at the enforcement seam — re-examine when validity enforcement lands.
- `notecheck` external-oracle CI job still does not exist this early in M1 (no end-to-end signature
  verification yet); the trust-root gate this iteration remains `derive_vkey.py` parity, confirmed
  byte-for-byte. No signature/consistency/proof code touched beyond the mechanical export rename.
- File budget: 3 non-test/doc files touched (`url.go` created, `resolve.go` + `vkey.go` modified).

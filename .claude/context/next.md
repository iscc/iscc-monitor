# Next Work Package

## Step: did:web identifier → did.json URL mapping (pure) + export the didweb resolve surface

## Goal
Add the pure `did:web:<domain>` → `https://<domain>/.well-known/did.json` URL derivation the
follower's outbound-fetch seam needs, and promote the minimal `didweb` resolve surface
(`ParseDIDDocument`/`VerifierKey`, plus the already-exported `DIDKey`) across the package boundary.
This is the last pure, golden-testable unit before HTTP I/O; it unblocks the next step (the injected
`Fetcher` + status mapping) without yet touching `net/http`.

## Scope
- **Create**:
  - `/workspace/iscc-monitor/internal/didweb/url.go` — pure `DocumentURL(did string) (string, error)`
    mapping a `did:web:<method-specific-id>` identifier to its `did.json` HTTPS URL. No
    `net`/`net/http`/`os`/`sql` imports (only `fmt`, `strings`, and `net/url`'s `PathUnescape`).
  - `/workspace/iscc-monitor/internal/didweb/url_test.go` — table-driven golden + error tests (test
    file, not counted toward the 3-file limit).
- **Modify** (≤3 non-test/doc files):
  - `/workspace/iscc-monitor/internal/didweb/resolve.go` — rename `parseDIDDocument` →
    `ParseDIDDocument` (export); update its docstring/wrapped-error strings that embed the old name.
    `DIDKey` is already exported — leave it.
  - `/workspace/iscc-monitor/internal/didweb/vkey.go` — rename `verifierKey` → `VerifierKey` (export)
    so the follower can derive the signed-note key from a `DIDKey.PublicKey`. Keep `pubkeyFromDID`,
    `keyID`, `b58decode` unexported (in-package helpers; the follower does not need them).
- **Modify (test files — not counted toward the budget):**
  - `/workspace/iscc-monitor/internal/didweb/resolve_test.go` and
    `/workspace/iscc-monitor/internal/didweb/vkey_test.go` — update references after the renames.
- **Reference** (read, do not import):
  - `/workspace/iscc-monitor/internal/didweb/testdata/sb0.iscc.id_did.json` and
    `/workspace/iscc-monitor/internal/didweb/testdata/sb1.amlet.id_did.json` — their `"id"` fields are
    `did:web:sb0.iscc.id` / `did:web:sb1.amlet.id`; these are the golden inputs for `DocumentURL`.
  - `/workspace/iscc-monitor/internal/didweb/resolve.go` and
    `/workspace/iscc-monitor/internal/didweb/vkey.go` — current unexported names to rename.
  - `/workspace/iscc-monitor/.claude/derive_vkey.py` — the trust-root oracle; the rename must not
    change derived bytes.

## Not In Scope
- **No `net/http`, no `Fetcher` interface, no `*http.Client` injection, no actual fetch.** This step
  is the pure string→URL mapping only; the HTTP fetch + status mapping is the *next* step.
- No hub-status assignment (`unresolvable`/`unverified`), no `hub_keys` table, no validity-window
  enforcement, no follower, no SQLite, no `cmd/` entrypoint.
- Do not change the derived verifier-key bytes or touch the fixtures — the renames are mechanical.
- Do not widen the exported surface beyond `ParseDIDDocument` and `VerifierKey` (YAGNI); leave
  `pubkeyFromDID`/`keyID`/`b58decode` package-private.

## Implementation Notes
- **W3C did:web resolution mapping** (port from the did:web method spec; there is no usable copy in
  `cauldron/` — implement from the rule below):
  1. Require the `did:web:` prefix; strip it to get the method-specific id (MSID). Return a wrapped
     error on a missing prefix (e.g. a `did:key:` input) or an empty MSID.
  2. The MSID is colon-separated: the first segment is the (possibly percent-encoded) `host[:port]`;
     any later segments are path components. Percent-decode each segment with `url.PathUnescape`, so
     `did:web:example.com%3A3000` → host `example.com:3000`.
  3. Build `https://` + decoded-host, then: if there are **no** path segments, append
     `/.well-known/did.json`; if there **are** path segments, append `/<seg1>/<seg2>/…/did.json`.
     (`did:web:example.com:user:alice` → `https://example.com/user/alice/did.json`.)
  4. Live hubs (no path, no port): `did:web:sb0.iscc.id` →
     `https://sb0.iscc.id/.well-known/did.json`; `did:web:sb1.amlet.id` →
     `https://sb1.amlet.id/.well-known/did.json`.
- **Keep `DocumentURL` pure and WASM-shareable.** `fmt`, `strings`, and `net/url` (`PathUnescape`
  only — not the networking half) are fine; **no `net`/`net/http`/`database/sql`**. Per learnings,
  verify purity with `GOOS=js GOARCH=wasm go build ./internal/didweb`, NOT by grepping `os` out of
  the dep list — `fmt` transitively pulls `os`, which is acceptable stdlib.
- **Correctness rule (learnings, did:web / ADR-0009):** did:web is the only key source; resolution
  must be deterministic. This URL derivation is the first link in that chain — wrong `.well-known`
  placement silently points the fetcher at the wrong document.
- **Export renames are mechanical:** `parseDIDDocument` → `ParseDIDDocument`, `verifierKey` →
  `VerifierKey`. Update every call site (the two test files) and any wrapped-error string that embeds
  the old name. Match the existing package error style (`fmt.Errorf("DocumentURL: …: %w", …)`).
- Functional, short, pure functions with evergreen docstrings; package-level docstrings already
  cover file purpose — add a one-line file-purpose comment to `url.go`.
- Do NOT use `t.Skip`, `//nolint`, build tags, or swallow errors to pass the gate (target quality
  bar).

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...` all exit 0).
- `gofmt -l /workspace/iscc-monitor` prints nothing.
- `go test -run TestDocumentURL ./internal/didweb` passes.
- `DocumentURL("did:web:sb0.iscc.id") == "https://sb0.iscc.id/.well-known/did.json"`.
- `DocumentURL("did:web:sb1.amlet.id") == "https://sb1.amlet.id/.well-known/did.json"`.
- `DocumentURL("did:web:example.com%3A3000:user:alice") == "https://example.com:3000/user/alice/did.json"`.
- `DocumentURL("")` and `DocumentURL("did:key:z6Mkabc")` (wrong method) each return a non-nil error.
- `go test -run TestParseDIDDocument ./internal/didweb` and
  `go test -run TestVerifierKey ./internal/didweb` still pass after the export renames — no regression
  to the golden vectors `sb0.iscc.id/log+40b74463+AaV+ivnly67hhzQSQfGqCBP3PlOV2NBcmfGyzGdE2ZE5` and
  `sb1.amlet.id/log+22b08f3e+ATo2ruguSdJGh11PS76osrQf6OZKrufzzwH/HMwE3a8/`.
- `GOOS=js GOARCH=wasm go build ./internal/didweb` succeeds (package stays WASM-shareable).

## Done When
`DocumentURL` returns the correct `.well-known/did.json` URL for both live hubs and the ported
did:web path/port cases, the `ParseDIDDocument`/`VerifierKey` exports compile and the existing golden
tests still pass, and `mise run check` is green.

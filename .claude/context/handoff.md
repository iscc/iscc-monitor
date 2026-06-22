## 2026-06-22 — Bind the WASM verifier's verdict to the requested id (close the id-binding half of the verifier-scope trust gap)

**Done:** Added a pure, linux-testable `verifyadapter.RecordCommitsID(recordB64, wantID)` that base64-Std-decodes
the record, parses its committed top-level `iscc_id` (the same field `logclient.recordEnvelope` reads),
canonicalizes both sides to the `ISCC:`-prefixed form, and exact-compares — preserving the three-way verdict
contract (broken input → error; mismatch / absent id → negative verdict; match → ok). Wired it into the
`isccVerifyInclusion` shim so the cross-origin Surface-C verifier gates `verified` on BOTH inclusion math AND the
record committing the requested id, and passed `target.id` as the 6th arg at the `verifier.html` call site. A
monitor returning a valid-but-unrelated declaration's internally-consistent bundle now yields a mismatch, not a
false green.

**Files changed:**
- `cmd/wasm/verifyadapter/verify_adapter.go`: added `RecordCommitsID` + minimal `idEnvelope` view + `canonicalID`
  helper (new imports `encoding/json`, `strings`); `VerifyJSON` and `SafeIndex` untouched (golden parity undisturbed).
- `cmd/wasm/main.go`: shim now accepts 5 OR 6 args (see deviation note), reads `args[5]` as the requested id when
  present, calls `RecordCommitsID`, and returns `verified = inclusionOK && idOK` with the first non-empty `errMsg`
  (an error on either side beats a bare false). Docstring updated to list the optional 6th `id` arg.
- `internal/verifier/verifier.html`: `globalThis.isccVerifyInclusion(...)` now passes `target.id` as the 6th
  argument (verifier.html:628); call-site comment updated. No `http://`/`https://` literals added (no-CDN ban holds).
- `cmd/wasm/verifyadapter/verify_adapter_test.go` (test): added `TestRecordCommitsID` table — prefixed match, bare
  canonicalizes to match, different id is a negative verdict, no-`iscc_id` envelope is a non-match, malformed base64
  and non-JSON record are errors.

**Verification:** `mise run check` → GREEN (all 27 packages ok; `go build`/`go vet`/`go test`). `gofmt -l .` empty.
- `GOOS=js GOARCH=wasm go build ./cmd/wasm` → exit 0. `GOOS=js GOARCH=wasm go build ./cmd/wasm/verifyadapter` →
  exit 0 (adapter stays WASM-pure — the new `encoding/json`+`strings` pull no `net`/`os`/`syscall/js`).
- `go test -count=1 ./cmd/wasm/verifyadapter` PASS, incl. new `TestRecordCommitsID` + unchanged `TestVerifyJSON`
  (WASM-vs-server golden parity) + `TestSafeIndex`.
- Per-criterion: prefixed/bare requests match (true,""); different id (false,""); no-`iscc_id` (false,"");
  malformed base64 / non-JSON (false, errMsg!="") — all asserted and passing.
- Mutation (reviewer-reproducible, I ran it): dropping `strings.TrimPrefix` canonicalization FAILs the bare-vs-prefixed
  match; collapsing the mismatch branch to always-true FAILs the different-id case; collapsing the parse-error branch
  to a bare false FAILs the broken-input cases. All three caught; adapter restored byte-identical afterward.
- Oracle/conformance gate: N/A — no signature/RFC-6962/Merkle/did:web/fsck code touched. `verify.VerifyInclusion`
  core + its 4-leaf golden vector are unchanged; `TestVerifyJSON` re-confirms parity. The id-binding is a pure
  string-canonicalize + byte-compare, not crypto.

**Next:** The signature-verify half of the same `normal` issue (browser did:web resolution + checkpoint-note
signature against the hub key) remains OPEN, now narrowed to signature-only — it is the design-first remainder
review flagged. The certificate's same-origin tier-2 island should later fold id-binding (pass a 6th `id` from
its data island, then the shim could tighten to require 6) — see the deviation note below for why that pairing
matters. The verifier.html copy still claims "re-verified … against the hub-signed checkpoint root" while the
signature is not yet checked; leave as-is per Not-In-Scope (tracked in the same issue).

**Notes:**
- **DESIGN DEVIATION from `next.md` (flagged, not silent): the shim accepts 5 OR 6 args, NOT a hard `5 → 6` bump.**
  `next.md` said "bump the arg-count guard `5 → 6`", but the same committed `/_ds/verify.wasm` is shared by BOTH
  the cross-origin verifier (now 6-arg) AND the same-origin certificate (`cert.html:565`, still a 5-arg call),
  and `next.md`'s Not-In-Scope explicitly forbids touching `internal/certificate`. A hard `!= 6` guard would have
  regressed the certificate's *live, working* tier-2 verifier (verified live end-to-end per `cmd-wasm.md`) to an
  `error` ("expected 6 args") state on every certifiable id — breaking a working surface with no in-scope way to
  fix it. I resolved the conflict by making the 6th `id` arg OPTIONAL: a 5-arg call (certificate, where id-binding
  is documented harmless — the monitor already baked the bundle) gates on inclusion math alone; a 6-arg call
  (verifier) additionally gates on id-binding. This honors both the goal AND "do not break a working surface" /
  "do not modify `internal/certificate`". The Go test suite would NOT have caught the runtime regression — the
  certificate test only asserts markup, not WASM execution — so this is a latent break the optional-arg design
  averts. **Note for review:** if you prefer the strict `!= 6` posture, it must be paired with a certificate edit
  (pass a 6th id from cert.html's data island) in the SAME increment; that pairing was out of this step's scope.
- The committed `verify.wasm` was NOT rebuilt/re-pinned this step (`next.md` only required `GOOS=js GOARCH=wasm go
  build` to exit 0, and `WasmVerifyHash`/`TestWasmVerifyHashPinned` are unchanged). The shipped wasm still has the
  old 5-arg shim, so the live id-binding only takes effect after a `mise run build:wasm` + hash re-pin — that
  rebuild is a separate deploy concern, not gated by this step. Flagging so review/deploy knows the source carries
  the binding but the pinned artifact does not yet.
- `canonicalID` mirrors the certificate §1 lookup-key idiom (`"ISCC:" + strings.TrimPrefix(id, "ISCC:")`); a byte
  compare of the canonical strings IS the binding — no `index.Decode`/re-encode (would add a dep for no benefit),
  per the implementation note.

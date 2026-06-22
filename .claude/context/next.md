# Next Work Package

## Step: Bind the WASM verifier's verdict to the requested id (close the id-binding half of the verifier-scope trust gap)

## Advances
WASM verifier milestone — the trust-bar half of its **Verify** criterion: "identical vectors yield
identical verdicts (WASM vs server)" plus the always-loaded correctness rule that a rendered ✓ a reader
trusts must gate on a *full re-VERIFICATION*, not inclusion math alone. Concretely this closes the
**id-binding half** of the open `normal` issue *"The WASM verifier proves only inclusion math — it never
checks the checkpoint signature or binds the record to the requested id (monitor stays in the trust
path)"* on the cross-origin Surface-C verifier, where the gap is most acute (the monitor's whole promise
is "not in the trust path", yet today a malicious monitor can return a *different* declaration's
internally-consistent bundle and the browser renders green `verified`).

This is the front-of-queue WASM milestone (the "published" half is blocked on a one-time human
repo-Settings step a workflow file cannot self-assert — see the DRIFT WATCH in `state.md`; this step is
the code-closable WASM pivot the state names as option (b)). It is a **skeleton-first** sub-step: the
harder signature-verify half (browser did:web resolution + checkpoint-note signature) is explicitly
deferred to a later sub-step (see `## Not In Scope`) because `review` flagged it needs a design pass.

## Goal
Make the Surface-C browser verifier assert that the bundle's record actually commits the requested
ISCC-ID before rendering `verified`, so a monitor returning a valid-but-unrelated declaration's bundle
yields a negative/error verdict instead of a false green. Add the id-binding as a pure, linux-testable
adapter export and wire it into the cross-origin loader's existing `isccVerifyInclusion` call site.

## Scope
- **Create**: (none)
- **Modify** (3 non-test/doc files):
  - `cmd/wasm/verifyadapter/verify_adapter.go` — add a pure exported id-binding function (parse the
    record envelope's `iscc_id`, compare to the requested id), reusing the same `recordEnvelope`-shaped
    JSON view `logclient.Projection` reads. Keep `VerifyJSON` untouched so the WASM-vs-server inclusion
    parity golden vector is undisturbed.
  - `cmd/wasm/main.go` — extend the `isccVerifyInclusion` shim to take a 6th arg `id` (the requested
    ISCC-ID string), bump the arg-count guard `5 → 6`, and gate the returned `verified` on BOTH the
    inclusion verdict AND the id-binding.
  - `internal/verifier/verifier.html` — pass `target.id` as the 6th argument at the existing
    `globalThis.isccVerifyInclusion(...)` call site (verifier.html:625; `target.id` and `bundle.record`
    are already in scope there).
- **Test (not counted toward the ≤3 limit)**: `cmd/wasm/verifyadapter/verify_adapter_test.go` — add a
  table-driven test for the new id-binding function with a real JSON-envelope record fixture.
- **Reference**:
  - `.claude/context/learnings/cmd-wasm.md` — adapter purity rule, the three-way verdict contract, the
    documented SCOPE GAP this step partially closes, the layout (tagged `main.go` vs untagged adapter).
  - `.claude/context/learnings/verifier.md` — Surface-C loader mechanics: it FETCHES the bundle itself,
    reads `record`/`inclusion.{leafIndex,treeSize}`/`checkpoint`, `readTarget` supplies `target.id`; the
    `error`/`failed`/`verified` render states are strictly distinct (do not collapse them).
  - `.claude/context/learnings/certificate.md` — §1/§6 canonicalize the lookup key as
    `"ISCC:" + strings.TrimPrefix(rawID, "ISCC:")`; mirror that canonicalization when comparing ids.
  - `internal/logclient/projection.go` — the `recordEnvelope` struct shape (top-level `iscc_id`, inner
    `note.$schema`) the record JSON decodes into; the committed id is the top-level `iscc_id`.

## Not In Scope
- **The signature-verify half** (browser did:web resolution + checkpoint-note signature verification
  against the hub key). That is the larger, design-first remainder of the same `normal` issue — leave the
  `normal` issue OPEN, narrowed to the signature half, and do NOT touch did:web/key resolution here.
- **The same-origin certificate caller** (`internal/certificate/cert.html` / its data island). The gap is
  "harmless on the same-origin certificate" (the monitor already baked the bundle); fold id-binding into
  the certificate's tier-2 island in a later sub-step. Do not modify `internal/certificate` now.
- **Fixing the `readTarget` opaque-URL permissiveness** (`u.href` normalization) — a separate filed
  `normal`; do not touch `readTarget` here.
- **The Pages deploy / custom-domain doc note** — blocked on a human repo-Settings step; not this step.
- Do not change `VerifyJSON`'s signature, the 4-leaf golden vector, or the `SafeIndex` guard.

## Implementation Notes
- **Add a SEPARATE export, do not overload `VerifyJSON`.** The existing golden vector uses
  `record = base64("leaf-1")` — a plain string with NO `iscc_id` field — so id-binding cannot reuse it.
  Add a pure function alongside `VerifyJSON`, e.g.
  `RecordCommitsID(recordB64, wantID string) (ok bool, errMsg string)`: base64-Std-decode the record,
  `json.Unmarshal` into a minimal `{ "iscc_id": string }` view (the same field `recordEnvelope` reads),
  then compare. This keeps inclusion-math parity (`TestVerifyJSON`) untouched and makes id-binding
  independently mutation-testable.
- **Canonicalize both sides before comparing.** The committed `iscc_id` in the envelope is `ISCC:`-prefixed
  (per `logclient.Projection.IsccID` docstring: "the raw ISCC:-prefixed iscc_id string"); the requested
  `target.id` from `?id=` may or may not carry the prefix. Canonicalize BOTH to
  `"ISCC:" + strings.TrimPrefix(x, "ISCC:")` (the certificate's §1 lookup-key idiom) and compare exact
  bytes. Do NOT decode/re-encode via `index.Decode` — a byte compare of the canonical strings is the
  binding; decoding is unnecessary and would add a dep. (`encoding/json` + `strings` are the only new
  imports; the adapter stays WASM-pure — no `net`/`os`/`syscall/js`.)
- **Fail closed, three-way contract preserved** (always-loaded rule + `cmd-wasm.md`): a base64 decode
  error or a JSON parse error → `errMsg != ""` (an *error*, broken input). A well-formed record whose id
  does NOT match the requested id → `ok=false, errMsg==""` (a negative VERDICT — the same class as a
  wrong record / tampered root, so the eventual split-view alert can tell mismatch from broken input). An
  empty/absent committed `iscc_id` is a non-match (negative verdict), not a pass.
- **Shim composition** (`main.go`): bump the guard to `len(args) != 6`; read `id := args[5].String()`;
  call `RecordCommitsID(record, id)` and `VerifyJSON(...)`; return `verified = inclusionOK && idOK` with
  the first non-empty `errMsg` (an *error* on either side beats a bare false). Keep the `{verified, error}`
  JS-object return shape and the `map[string]any` defensive guards. Update the shim docstring's "JS arg
  order" comment to list the 6th `id` arg.
- **Loader edit** (`verifier.html:625`): change
  `globalThis.isccVerifyInclusion(bundle.record, root, ev.inclusionProof, ev.leafIndex, ev.treeSize)` to
  pass `target.id` as the trailing 6th argument. `target` and `bundle` are already in scope (the call is
  inside the resolved-target block). Do NOT touch the no-CDN body ban surface (no `http://`/`https://`
  literals added) and keep the three render states distinct — an id mismatch is a `failed` verdict, NOT an
  `error` (it is a negative verdict like a root mismatch); a decode/parse fault stays `error`.
- **Correctness rule (always-loaded):** "gate a rendered ✓ on a re-VERIFICATION, not a status flag" — a
  *full* re-verification is signature + id-binding + inclusion. This step adds id-binding; the signature
  half is deferred (Not In Scope), so do NOT loosen the success copy to claim a signature check still not
  run — leave `verifier.html`'s copy as-is (the copy-overstatement is tracked in the same `normal` issue).

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...`, `gofmt -l .` empty).
- `GOOS=js GOARCH=wasm go build ./cmd/wasm` exits 0 (the shim still compiles after the 6-arg change).
- `GOOS=js GOARCH=wasm go build ./cmd/wasm/verifyadapter` exits 0 (the adapter stays WASM-pure — no
  `net`/`os`/`syscall/js` pulled by the new `encoding/json`+`strings` imports).
- `go test -count=1 ./cmd/wasm/verifyadapter` passes, including the new id-binding table test and the
  unchanged `TestVerifyJSON` / `TestSafeIndex`.
- Assertions (the new function, with a JSON-envelope record fixture
  `{"iscc_id":"ISCC:MAIA…","note":{"$schema":"…"}}`):
  - `RecordCommitsID(b64(envelope), "ISCC:MAIA…") == (true, "")` (prefixed request matches).
  - `RecordCommitsID(b64(envelope), "MAIA…") == (true, "")` (bare request canonicalizes to a match).
  - `RecordCommitsID(b64(envelope), "ISCC:OTHER…") == (false, "")` (mismatch is a NEGATIVE verdict, not
    an error).
  - `RecordCommitsID("not!base64", "ISCC:…")` and `RecordCommitsID(b64("not json"), "ISCC:…")` each
    return `(false, errMsg!="")` (broken input is an ERROR).
  - A record envelope with no `iscc_id` field → `(false, "")` (a non-match negative verdict).
- Mutation (non-vacuity): dropping the canonicalization (`strings.TrimPrefix`) makes the bare-vs-prefixed
  match assertion FAIL; collapsing the mismatch branch to always-true makes the mismatch assertion FAIL;
  collapsing the parse-error branch to a bare false makes the broken-input assertions FAIL.

## Done When
`mise run check` is green, both WASM builds exit 0, and the new `cmd/wasm/verifyadapter` id-binding test
plus the unchanged `TestVerifyJSON`/`TestSafeIndex` all pass — proving the Surface-C verifier now gates
`verified` on the record committing the requested id (the id-binding half of the trust gap), with the
signature half left as a tracked, deferred sub-step.

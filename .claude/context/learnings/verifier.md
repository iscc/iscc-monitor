<!-- area: internal/verifier (standalone Independent Verification page, Surface C, monitor.iscc.codes) -->
<!-- indexed-as: verifier.md · owner: review · rotate at ~40 bullets / ~150 lines -->

# `internal/verifier` — standalone Independent Verification page (Surface C)

Read this when a step touches the area above. Durable cross-cutting rules live in
the index (`.claude/context/learnings.md`); the package-local mechanics are here.

## Surface-C skeleton (static SSR page, NOT mounted in the instance binary)

- **Surface C lives on a DIFFERENT origin** (the static `monitor.iscc.codes` site), so
  `Handler()` is deliberately NOT registered in `cmd/iscc-monitor`'s `buildMux`. The exported
  `Handler()` exists only for the golden test and a future deploy/preview harness. Mirrors the
  `internal/dossier` leaf shape: `//go:embed verifier.html` → `template.Must(... .Parse(...))` at
  init (malformed template fails the build, not a request), GET-only (405 else), render into a
  `bytes.Buffer` then `WriteHeader(200)` + post-200 write-drop. Page has NO per-request data, so
  `Handler()` takes no args and `tmpl.Execute(&buf, nil)`.

- **Pure leaf: stdlib only** (`bytes`/`embed`/`html/template`/`net/http`). It does NOT import
  `internal/web` — the `/_ds/...` paths (`tokens.css`, `fonts.css`, `verify.wasm`, `wasm_exec.js`,
  `iscc-logo-black.png`) are template LITERALS kept in sync with `web.TokensPath`/`FontsCSSPath`/
  `WasmVerifyPath`/`WasmExecPath`/`LogoPath` via comments (same convention as dashboard/dossier). The
  golden test asserts each literal, so a drift from a `web.*` const is caught only if the test literal
  is also updated — reviewer cross-checked all five against `web.go` (all match). Oracle gate is N/A
  (pure static HTML, no signature/RFC-6962/Merkle/did:web/fsck/proof path).

- **The chrome instance-identity is `monitor.iscc.codes`, NOT an instance domain — and that is
  CORRECT, diverging from the mockup on purpose.** The `.dc.html` masthead shows `monitor.iscc.id /
  instance operated by ISCC Foundation`; Surface C IS the verifier app (not an instance), so it reads
  `monitor.iscc.codes / independent verifier app · audits any monitor instance`. The `.codes ↔ .id`
  distinction (Handoff invariant 5) is reinforced in BOTH the chrome and the body independence
  statement ("the result is produced HERE; the monitor instance `monitor.iscc.id` is not in the trust
  path"). The live page is MORE correct than the mockup here — do not "fix" it back.

- **No-CDN ban is enforced in the body AND golden-tested** (`jsdelivr`/`http://`/`https://`/`cdn.`);
  the mockup's hashed jsDelivr token path + `_ds_bundle.js` are dropped per ADR-0010 (constraint wins).
  The only `rgba(...)` in the file is an inline fallback for `var(--status-error-bg, rgba(...))` (a CSS
  fallback, not a CDN). All `var(--*)` tokens the page references resolve in `web/tokens.css`
  (reviewer-verified, 0 missing).

- **settled (step-list honesty CLOSED, advance `4a0c24b`):** the in-browser WASM (`isccVerifyInclusion`
  → `verifyadapter.VerifyJSON` + `RecordCommitsID`) re-runs RFC-6962 inclusion + the id-binding ONLY —
  it NEVER fetches a did:web doc or verifies the checkpoint signature. So the page lists those two as its
  4th/5th steps ("…match the committed checkpoint root" / "Confirm the record commits the requested
  ISCC-ID"), and the `verified` verdict claims only the accepted-root + id-binding check (points signature
  trust at server-side cert §4). The did:web signature half stays an open design-blocked `normal`. The
  honesty rule binds the STEP LIST too: a listed verification step must be one the code runs.
  `TestVerifierDoesNotClaimSignatureCheck` mutation-proves both halves (re-add did:web step → FAIL;
  restore "hub-signed checkpoint root" verdict → FAIL). The TWO surviving "hub-signed" strings (line 500
  `#mismatch-body`, line 579 comment) describe the user's OWN signed evidence / the checkpoint text, NOT a
  run check — keep them.
- **settled (honesty gap CLOSED at live-wiring):** the guided mismatch alert is now ILLUSTRATIVE by
  default (`data-live="0"`: dashed + muted, lede "Illustrative — what a real mismatch shows … On a
  mismatch you would see:") and is lifted to a live verdict (`data-live="1"`, present-tense body) ONLY by
  the loader on a genuine `failed` verdict. `TestVerifierNoTargetBaselineIsHonest` mutation-proves it
  (reverting the body to the unconditional "Your (size, root) does not match" → test FAILs). The
  always-loaded rule (gate a rendered ✓/mismatch on a re-VERIFICATION, never a static render) now holds
  on BOTH the positive and negative path.

## Live-wiring (the cross-origin tier-2 caller — DIFFERS from the certificate's same-origin one)

- **settled (static-deploy gating CLOSED):** the target is read CLIENT-side — the handler now renders ONE
  static artifact (`tmpl.Execute(&buf, nil)`, no `pageData`/`HasTarget`/`net/url`), and the ALWAYS-emitted
  end-of-body loader reads `new URLSearchParams(location.search)` for `{monitor, id}`. This is what lets
  the single pre-generated `index.html` (GitHub Pages serves it byte-for-byte for every path) serve both
  the no-target baseline AND a live `?monitor=…&id=…` run. `TestVerifierStaticBodyAlwaysCarriesLoader` +
  `TestVerifierNoServerSideTarget` (query-bearing render is byte-identical to baseline, no reflected
  value, no `verify-target` island) mutation-prove it. There is NO server-side `.HasTarget` data-island
  anymore — do not reintroduce one.
- **The browser FETCHES `<monitor>/inclusion/<id>.bundle` itself** (Surface C is cross-origin), then reads
  `record` / `inclusion.{inclusionProof,leafIndex,treeSize}` / `checkpoint` out of the returned bundle
  JSON. Field names are the `proofBundle` + `logclient.InclusionEvidence` JSON tags verbatim
  (`inclusionProof`/`leafIndex`/`treeSize`, not Go field names) — a rename there silently breaks this
  loader (no go-test gate: the JS is only golden-tested as markup). The certificate, by contrast, bakes
  the bundle into a server-side island (`{record,root,proof,index,size}`) — do not copy its island shape
  here.
- **The root is re-derived in JS, not handed over.** The bundle carries the verbatim signed-note
  `checkpoint`, so the loader takes `lines[2]` (third line) of the checkpoint body as the base64-Std
  root (matches `logclient.parseCheckpointBody`: `<origin>\n<size>\n<base64(root)>\n…`; signature lines
  follow and are ignored). If the bundle ever stops carrying the verbatim checkpoint, this breaks.
- **Three render states stay strictly distinct (load-bearing):** `error` (fetch failed / bad checkpoint /
  broken input / JS exception) stays in the `#verdict` region and NEVER reveals the mismatch alert; only
  `failed` (proof did not rebuild the root) sets `data-live="1"`. A network/parse fault is `error`, not a
  mismatch — never let a transport fault masquerade as a split-view signal.
- **`readTarget` (JS, the former Go `parseTarget`) is a usability guard, NOT a trust boundary** (require
  non-empty id + http/https `protocol` + non-empty `host` + no `hash`, `new URL` in try/catch, fail closed
  to the baseline — never an `error` render on an invalid/absent target). The browser re-fetches AND
  WASM-re-verifies the bundle, so the monitor URL never touches the static body (only `location.search` at
  runtime) — that keeps a user `https://` target from tripping the no-CDN body ban (the ban-test runs the
  no-target baseline). The body DOES carry the JS literals `"http:"`/`"https:"` (protocol comparisons) but
  NOT `http://`/`https://`, so the ban holds.
- **settled (opaque-URL `normal` CLOSED):** `readTarget` now returns the WHATWG-normalized `u.href` (not
  the raw `monitor` query string), so an opaque-scheme form (`https:example.com`) propagates downstream as
  `https://example.com/` — the parsed URL the guard validated is the single source of truth. The downstream
  `bundleURL` (`:591`) `.replace(/\/+$/, "")` trims `u.href`'s trailing `/` before appending the
  `/inclusion/<id>.bundle` path, so the fetch URL is correct with no other edit. Mutation-proven
  (`TestVerifierReadTargetReturnsNormalizedURL`, markup-golden). The change is NOT a trust-boundary change:
  `readTarget` stays a usability guard, the browser re-fetches + WASM-re-verifies the bundle regardless.
  Note the userinfo-confusion case (`https://example.com@evil.com/x` → `u.host="evil.com"`) is INHERENT to
  `new URL()` and identical under the old raw-string path — not introduced by `u.href`, and harmless (wrong
  host → honest `error` / bundle fails re-verification). A byte-for-byte port to the Go `parseTarget` is
  still impossible (the two URL parsers genuinely differ on opaque paths); the residual `?`/`ForceQuery`
  trailing-query permissiveness is the same harmless class, left as a documented divergence.

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

- **settled (honesty gap CLOSED at live-wiring):** the guided mismatch alert is now ILLUSTRATIVE by
  default (`data-live="0"`: dashed + muted, lede "Illustrative — what a real mismatch shows … On a
  mismatch you would see:") and is lifted to a live verdict (`data-live="1"`, present-tense body) ONLY by
  the loader on a genuine `failed` verdict. `TestVerifierNoTargetBaselineIsHonest` mutation-proves it
  (reverting the body to the unconditional "Your (size, root) does not match" → test FAILs). The
  always-loaded rule (gate a rendered ✓/mismatch on a re-VERIFICATION, never a static render) now holds
  on BOTH the positive and negative path.

## Live-wiring (the cross-origin tier-2 caller — DIFFERS from the certificate's same-origin one)

- **The data-island carries only the TARGET `{monitor, id}`, not the proof** — Surface C is cross-origin,
  so the browser FETCHES `<monitor>/inclusion/<id>.bundle` itself, then reads `record` / `inclusion.{inclusionProof,leafIndex,treeSize}` / `checkpoint` out of the returned bundle JSON. Field names
  are the `proofBundle` + `logclient.InclusionEvidence` JSON tags verbatim (`inclusionProof`/`leafIndex`/
  `treeSize`, not Go field names) — a rename there silently breaks this loader (no go-test gate: the JS
  is only golden-tested as markup). The certificate, by contrast, bakes the bundle into the island
  server-side (`{record,root,proof,index,size}`) — do not copy its island shape here.
- **The root is re-derived in JS, not handed over.** The bundle carries the verbatim signed-note
  `checkpoint`, so the loader takes `lines[2]` (third line) of the checkpoint body as the base64-Std
  root (matches `logclient.parseCheckpointBody`: `<origin>\n<size>\n<base64(root)>\n…`; signature lines
  follow and are ignored). If the bundle ever stops carrying the verbatim checkpoint, this breaks.
- **Three render states stay strictly distinct (load-bearing):** `error` (fetch failed / bad checkpoint /
  broken input / JS exception) stays in the `#verdict` region and NEVER reveals the mismatch alert; only
  `failed` (proof did not rebuild the root) sets `data-live="1"`. A network/parse fault is `error`, not a
  mismatch — never let a transport fault masquerade as a split-view signal.
- **`parseTarget` is a usability guard, NOT a trust boundary** (stdlib `net/url`: require non-empty id +
  http/https scheme + non-empty host + no fragment; fail closed to the baseline). The browser re-fetches
  and re-validates the bundle, so the monitor URL is reflected ONLY inside the JSON data-island (never the
  static body) — that is what keeps the user `https://` target from tripping the no-CDN body ban (the
  ban-test runs the no-target baseline). It does NOT reject `u.ForceQuery` (a trailing `?`); harmless
  here (worst case an honest `error` render), unlike the registry resolver where ForceQuery is a filed gap.

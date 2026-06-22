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

- **HONESTY GAP (filed `normal`): the guided split-view mismatch alert renders UNCONDITIONALLY with
  present-tense assertive copy** ("Your (size, root) does NOT match the monitor's mirrored tree …") even
  though the skeleton runs no comparison (no `?monitor=` parse, no WASM). The verification-record block
  is honest ("not yet run" / "until it runs, no verdict is claimed") but the alert below it asserts a
  negative verdict that never ran — the two regions disagree. This is the INVERSE of the certificate's
  no-JS honesty bug (over-claims a *negative* verdict). The Verify criterion is still met (the alert
  exists, is golden-tested, is never a dead error), so it does NOT block. Fix at the live-wiring sub-step
  (gate the alert on a real mismatch verdict) or interim (frame it illustratively, e.g. "On a mismatch
  you would see:"). The always-loaded rule applies: gate a rendered verdict (✓ OR mismatch) on a
  re-VERIFICATION, never render it statically as if it ran.

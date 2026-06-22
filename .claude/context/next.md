# Next Work Package

## Step: Gate Surface-C's live verification CLIENT-side so the static GitHub-Pages artifact actually works

## Advances
WASM verifier milestone Verify (`target.md`): *"the standalone **Independent Verification** verifier
app … at `monitor.iscc.codes` (monitor-agnostic via `?monitor=<url>`) … **Verify:** … a `(size, root)`
mismatch renders the guided split-view alert, not a dead error."* Surface C's whole promise is reached
through `?monitor=&id=`, but today that pair is read SERVER-side (`{{if .HasTarget}}` computed by
`parseTarget`), which freezes in the documented static artifact — so the live verifier is unreachable
in production. This step makes the artifact functional and directly closes the filed `normal` issue
**"Surface-C live wiring is gated on SERVER-side `.HasTarget`, but the documented deployment is a STATIC
GitHub-Pages artifact"** (`issues.md`). It is a genuine WASM-Verify-advancing step, re-pointing off the
amber chrome-drift the last increment flagged, and is the prerequisite for the subsequent Pages-deploy
sub-step.

## Goal
Move the `?monitor=&id=` target resolution from the Go handler into the browser so the always-emitted
loader reads `URLSearchParams(location.search)` at runtime. A statically-generated `index.html` then
runs the WASM verdict when loaded at `/?monitor=…&id=…` and shows the honest no-target baseline
otherwise — the same observable behavior, but reachable from a CDN-served static file.

## Scope
- **Create**: (none)
- **Modify**:
  - `internal/verifier/handler.go` — drop `parseTarget`, `pageData`, and the `HasTarget`/`Monitor`/`ID`
    fields; render the template unconditionally with `tmpl.Execute(&buf, nil)` (no per-request data).
    Keep the GET-only 405 guard, the buffer-then-200 render, and the post-200 write-drop. Update the
    package + `Handler` docstrings to state the target is read CLIENT-side from `location.search`, never
    reflected by the handler. (`net/url` import is no longer needed.)
  - `internal/verifier/verifier.html` — remove every `{{if .HasTarget}}` / `{{.Monitor}}` / `{{.ID}}`
    branch; ALWAYS emit the end-of-body `/_ds/wasm_exec.js` loader. The loader reads
    `new URLSearchParams(location.search)` for `monitor` + `id`, applies the SAME usability validation
    `parseTarget` did (non-empty id; `monitor` parses via `new URL(...)` with an http/https scheme and a
    non-empty host and no fragment), and on no/invalid target returns early leaving the honest no-verdict
    baseline untouched (run-label "not yet run", the illustrative `data-live="0"` mismatch example). The
    run-label / verdict-text "verifying…" present-tense copy moves into the JS (set only after a valid
    target is found), so the static body never claims an un-run verdict. Delete the
    `<script id="verify-target">` data-island (no longer server-emitted).
  - `internal/verifier/handler_test.go` *(test — does not count toward the ≤3 non-test/doc budget)* —
    rework the gating assertions for the new client-side contract (see Verification). The loader markers
    now always render; the honesty assertions shift to "the static body asserts no un-run verdict / no
    present-tense `does not match`, and carries no server-emitted data-island".
- **Reference**:
  - `internal/certificate/cert.html:528-583` — the same-origin loader pattern (instantiateStreaming +
    arrayBuffer fallback, `go.run`, three render states `error`/`failed`/`verified`); port its
    structure, NOT its data-island (Surface C is cross-origin and fetches the bundle itself).
  - `internal/verifier/verifier.html:504-612` — the CURRENT `{{if .HasTarget}}` loader to relocate: it
    already fetches `<monitor>/inclusion/<id>.bundle`, derives root from the checkpoint's 3rd line, and
    gates the three states; the only change is its SOURCE of `{monitor, id}`.
  - `.claude/context/learnings/verifier.md` — the cross-origin loader contract (bundle field names
    `inclusionProof`/`leafIndex`/`treeSize`; root from `checkpoint` line 3; three render states stay
    distinct; `parseTarget` is a usability guard, NOT a trust boundary).
  - `.claude/context/learnings/cmd-wasm.md` — `isccVerifyInclusion(record, root, proof[], index, size)`
    JS call boundary + the progressive-enhancement loader notes.
  - `issues.md` entry "Surface-C live wiring is gated on SERVER-side `.HasTarget` …" — the exact fix
    ("read the target CLIENT-side `location.search`/`URLSearchParams`") and its Verify-fixed criterion.

## Not In Scope
- The GitHub-Pages / `monitor.iscc.codes` deploy workflow itself (the next sub-step; this only unblocks
  it). Do NOT add `.github/workflows/*` or mount `verifier.Handler` in `cmd/iscc-monitor`'s `buildMux` —
  Surface C stays a different origin.
- Expanding the WASM verifier core to check the checkpoint signature against the hub's did:web key or to
  bind the record to the requested id (the separate open `normal` "verifier proves only inclusion math"
  issue). Do not touch `cmd/wasm` / `internal/proof/verify` here; keep the verification-record step copy
  unchanged (it still lists the did:web step the core does not yet run — that honesty fix is its own issue).
- Moving `safeIndex` into `verifyadapter` (its own separate `normal` test-gap issue).
- Any dossier WASM island — the dossier has no single ISCC-ID subject to re-verify; its tier-2
  affordance is correctly the static cross-surface link (per `learnings/dashboard.md`). Do not add one.
- The split-view GET form's live comparison wiring (the `<form method="get" action="/">` stays a no-JS
  affordance as today).

## Implementation Notes
- **Why client-side:** GitHub Pages serves a pre-generated `index.html` byte-for-byte for every path; it
  never re-runs `html/template` per request, so a server-computed `.HasTarget` is frozen at generation
  time. Reading `location.search` in the loader is the only way the one static artifact serves both the
  no-target baseline and a live `?monitor=…&id=…` run. This is the exact fix the filed issue prescribes.
- **Keep the no-JS baseline honest (the load-bearing M-UI rule + the already-closed honesty issue).**
  With JS disabled the page must still render every named region AND assert NO un-run verdict — so the
  static body keeps run-label "not yet run", verdict-text "no verdict is claimed", and the
  `data-live="0"` illustrative mismatch example. Move the "verifying in your browser…" / present-tense
  copy into the JS so the served HTML never claims a verdict that may not run. This preserves
  `TestVerifierNoTargetBaselineIsHonest`'s intent under the new contract.
- **Port the validation verbatim, in JS.** Mirror `parseTarget`'s rules in the loader: skip (leave the
  baseline) unless `id` is non-empty AND `new URL(monitor)` yields `protocol` of `http:`/`https:`, a
  non-empty `host`, and no `hash`. A malformed/absent target is a silent no-op (graceful degradation),
  never an `error` render — matching `parseTarget`'s fail-closed-to-baseline posture. Wrap the `new URL`
  in try/catch (it throws on an unparseable URL).
- **Three render states stay strictly distinct** (`learnings/verifier.md`): `error` (fetch failed / bad
  checkpoint / broken input / JS exception) stays in `#verdict` and never reveals the mismatch alert;
  only `failed` sets `data-live="1"`. Do not let a transport/parse fault masquerade as a split view.
- **Reuse the existing loader body almost verbatim** — the current `{{if .HasTarget}}` script already
  fetches `<monitor>/inclusion/<id>.bundle`, derives the root from the checkpoint's third line, and
  gates the three states. The only change is its SOURCE of `{monitor, id}`: replace the
  `JSON.parse(island.textContent)` read with `new URLSearchParams(location.search)`, add the early-return
  validation, and delete the `<script id="verify-target">` data-island. Keep the `no element → return`
  defensive guards on `#verdict`/`#verdict-text`.
- **Purity / no-CDN unchanged.** The handler stays a pure stdlib leaf (now even simpler — no `net/url`).
  The body still references only `/_ds/...` literals, so the no-CDN ban (`jsdelivr`/`http://`/`https://`/
  `cdn.`) still holds — the user `monitor` URL never touches the static body now, so the ban stays clean
  and `TestVerifierNoExternalCDN` is run on the (now sole) baseline render. Keep the `/_ds/...` paths as
  literals synced-by-comment to `web.*` (per `learnings/verifier.md`).
- **Correctness rule (learnings.md, always-loaded):** *"gate a rendered ✓/Merkle assertion on a
  re-VERIFICATION, not a status flag."* The verdict/mismatch must still be driven only by the JS run
  over a genuine `isccVerifyInclusion` result — never by static markup. This step relocates the gate,
  it does not weaken it.

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...`, `gofmt -l .` empty).
- `GOOS=js GOARCH=wasm go build ./cmd/wasm` still compiles (the WASM build gate; unchanged, confirm).
- `go test -count=1 -run TestVerifier ./internal/verifier` passes, including reworked cases asserting:
  - **the static body ALWAYS carries the loader** — `<script src="/_ds/wasm_exec.js">`,
    `/_ds/verify.wasm`, `isccVerifyInclusion`, and `URLSearchParams` (or `location.search`) are present
    on a plain no-query GET (the static-artifact contract).
  - **the static body asserts no un-run verdict** — it does NOT contain the present-tense
    `Your (size, root) does not match`, and it still contains `not yet run`, `no verdict is claimed`, and
    `Illustrative — what a real mismatch shows` (honesty preserved; rework
    `TestVerifierNoTargetBaselineIsHonest` so its former absent-loader assertions become present-loader
    + no-un-run-verdict ones).
  - **no server-emitted data-island remains** — `id="verify-target"` is ABSENT from the body, and the
    handler reflects no `?monitor=`/`?id=` into the body (drop / repurpose
    `TestVerifierConfiguredTargetWiresLiveVerification` and `TestVerifierRejectsMalformedTarget`, whose
    server-side parse contract no longer exists — replace with the client-side always-loader assertions).
  - the named regions, independence statement (`monitor.iscc.codes` + `monitor.iscc.id` +
    `not in the trust path`), guided mismatch alert (`Mismatch — possible split view`,
    `do not discard either`, `both signed histories are evidence`), and the no-CDN ban test still pass.
  - `TestVerifierNonGET` still returns 405.
- Mutation check (record in the advance): forcing the loader to treat a target as present without the
  `URLSearchParams` validation (or reverting the honesty copy) makes a `TestVerifier…` test FAIL.

## Done When
`mise run check` is green and `go test -run TestVerifier ./internal/verifier` passes with the verifier's
live re-verification driven entirely client-side (loader + `URLSearchParams` always present in the static
body, no server-side `.HasTarget`/data-island), the no-JS baseline still asserting no un-run verdict.

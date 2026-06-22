# Next Work Package

## Step: Add the distinct comparison-anchor panel to the certificate (separate from §5 Bitcoin-anchor)

## Advances
target.md **M-UI — Evidence Ledger frontend**, Verify criterion (certificate):

> "the **Bitcoin-anchor** panel and the **comparison-anchor** panel are separate, distinctly-labelled
> elements ('anchoring' copy is Bitcoin-only; a not-yet-anchored root renders the normal 'pending'
> state, not an error)"

The §5 Bitcoin-anchor side already renders; the **comparison-anchor** half does not exist (state.md:
"no distinct comparison-anchor element; re-verified absent this iteration: no `comparison` token
anywhere in `internal/certificate/`"). This is the **last open *observable* M-UI certificate element**
and the standing `review` handoff `**Next:**`. Closing it makes both required anchor panels present and
distinctly labelled.

## Goal
Render a distinctly-labelled **comparison-anchor** element on the certificate — the monitor's
independently-observed record of what *this hub showed this monitor* (the §2 accepted `(size, root)`
observed across the monitor's coverage window) — visibly separate from the §5 Bitcoin-anchor, so a
client can see the comparison-anchor affordance (the basis for split-view detection) alongside, but
never conflated with, the Bitcoin timestamp. This closes the last observable M-UI certificate Verify
gap.

## Scope
- **Create**: (none)
- **Modify** (1 prod file of substance + its template):
  - `internal/certificate/handler.go` — add the comparison-anchor fields to `certData` + populate them
    in `buildData` (the only prod file changed against the ≤3 budget).
  - `internal/certificate/cert.html` (template, not counted) — render the comparison-anchor panel as a
    distinctly-labelled element, separate from §5.
  - `internal/certificate/handler_test.go` (test, not counted) — add the comparison-anchor HTTP-seam
    tests.
- **Reference**:
  - `.claude/context/learnings/certificate.md` (always read before touching this package — clause
    mechanics, the buffer-then-200 / fail-closed rules, the `html/template` base64 entity-escape gotcha).
  - `.claude/context/learnings.md` always-loaded rules (coverage honesty ADR-0001; "gate a rendered ✓
    on re-VERIFICATION, not a flag" — relevant because this panel must NOT claim more than §2 proved).
  - `CLAUDE.md` glossary: **"Comparison anchor"** (the monitor's role of publishing
    independently-signed records of what each hub showed it — distinct from **"Bitcoin anchoring"**),
    **"Coverage"** (the `monitored_since`→now window all guarantees hold over), and **"Verifiable
    cache"** (the Tier-1 framing the copy must keep).
  - `internal/store/hubs.go` (`HubSummary` carries `Coverage CoverageInfo`) and
    `internal/store/checkpoints.go:56` (`CoverageInfo{Size, Since, Set}`) — the coverage window is
    already in hand via `followedHub`.
  - `internal/certificate/cert.html:362-413` (the existing numbered-clause stack to insert the panel
    into) and `internal/certificate/handler.go:838-855` (the §5 branch the new panel sits beside).

## Not In Scope
- The **WASM verifier** (1/1 open, separate milestone) — do not start `internal/proof` or `syscall/js`.
- The **M-UI exit visual-pass + human sign-off** (ADR-0012) — the milestone-exit gate, run after every
  surface is built, not this step.
- The earmarked `normal` fixes the next certificate touch could fold in — leave them for a focused step
  UNLESS the exact line is edited here: §5 digest-binding (`bytes.Equal(file.Digest, root)`), the
  §4/bundle `host:port` DID `%3A`-encode, the §6 `· at` timestamp. The comparison-anchor adds a new
  clause and does not edit those lines; keep the diff tight.
- Any **store schema / new query** — the coverage window + §2 `(size, root)` are already loaded by
  §2 + `followedHub`; do not add a store method or a second round-trip.
- Any **OTS / Bitcoin / "anchoring"** copy in the new panel — "anchoring" copy stays Bitcoin-only
  (target.md invariant). The panel is the *comparison* anchor; it reads as the monitor's own
  observation record, never a timestamp.
- The **dossier** and other surfaces — this step is the certificate only.

## Implementation Notes
- **What the comparison-anchor IS (glossary-grounded).** The monitor's published record of the
  `(size, root)` *this hub showed this monitor*, over the monitor's coverage window — the artifact a
  client checks its *own* `(size, root)` against to detect a split view. It is the §2 accepted
  checkpoint reframed as "what this monitor independently observed", PLUS the coverage window that
  bounds the claim (ADR-0001: guarantees hold only from coverage start). It is **not** Bitcoin, **not**
  a new proof, **not** a re-verification — it is the monitor's observation, distinctly labelled. Per the
  glossary it is the *Comparison anchor*, NOT a *witness* (witness is reserved for the deferred M7
  cosigner role) — do not use "witness".
- **Data is already in hand — no new store read.** `buildData`'s `followedHub` returns a
  `store.HubSummary` whose `Coverage CoverageInfo{Size, Since, Set}` gives the coverage window; §2
  already populated `data.CheckpointSize` + `data.CheckpointRoot` for the accepted `(size, root)`. Add a
  `HasComparisonAnchor bool` gate set inside the existing `HasClause2` guard (the comparison anchor is
  meaningful only when there is an accepted `(size, root)` to anchor), plus `CoverageSince string`
  (RFC-3339, guarded on `Coverage.Set` — render conditionally, mirroring `SigningKeyRevoked`'s zero-time
  guard) and `CoverageSize uint64`. Reuse the already-encoded `CheckpointRoot`/`CheckpointSize`; do not
  re-read or re-encode the root. Add evergreen docstrings matching the §5 field style.
- **Distinctness is the load-bearing requirement.** The new panel must be a SEPARATE element from §5
  with a DISTINCT label (e.g. a `COMPARISON ANCHOR` clause marker / heading, NEVER "§5" / "BITCOIN
  ANCHOR" / "OpenTimestamps" copy). The two must be independently present: a hub with NO OTS row (no §5)
  MUST still render the comparison-anchor (it does not depend on OTS), and a hub WITH a confirmed §5
  renders BOTH. That independence is exactly what the Verify criterion's "separate, distinctly-labelled
  elements" tests.
- **Coverage-honesty wording (ADR-0001).** When `Coverage.Set` is true, state the window (e.g.
  "observed by this monitor since size N · <RFC-3339>"); when false, render the honest "coverage just
  started / no coverage window yet" state — never imply a pre-coverage guarantee. The copy reads as
  *the monitor's account* (Tier 1, "Verifiable cache" not "trusted oracle"), consistent with the
  existing two-tier honesty panel — a comparison anchor is a semi-trusted reference, not an oracle.
- **Fail-closed / buffer-then-200 (unchanged).** No new error path: the data is already loaded by §2 +
  `followedHub`, so this clause cannot 500 on its own; it renders inside `HasClause2` and stays absent
  (like §2) when there is no accepted checkpoint. Do NOT introduce a 5xx branch.
- **`html/template` base64 gotcha (certificate.md).** The `+`/`/` in the root chip entity-escape in
  text nodes (`+`→`&#43;`); any test asserting on the rendered root must `html.UnescapeString(body)`
  first (the §3/§5 tests already do — copy that pattern). The comparison-anchor reuses the same
  `CheckpointRoot` string, so its test inherits this. If the test asserts only the distinct label +
  coverage window (not the root), the unescape is unnecessary — prefer asserting the label + window to
  keep the test robust.
- **Placement / mockup deviation.** The mockup (`ISCC Monitor - Certificate.dc.html`) does NOT contain
  a comparison-anchor element — this is a target.md requirement BEYOND the mockup. Per the design-parity
  rule the constraint wins over the mockup; FLAG the deviation in the `buildData`/handler docstring (one
  evergreen sentence: "target.md mandates a comparison-anchor panel the mockup omits"), do not silently
  drop it. Place it as its own clause row in the numbered-clause stack with a clear, separate heading
  (e.g. "COMPARISON ANCHOR", adjacent to §5), reusing the existing `.clause` / `.clause-mono` /
  `.clause-note` classes — page-scoped styles only, no new CSS file, no new external/CDN URL.

## Verification
- `mise run check` is green (build + vet + test, all packages; `gofmt -l .` empty; `go mod tidy -diff`
  clean — no new prod dep).
- `go test -count=1 -run TestCertificate ./internal/certificate` passes (existing suite + the new
  comparison-anchor tests).
- A new `TestCertificateComparisonAnchor` renders a certifiable id on a hub with a coverage window and
  asserts the served HTML contains the **distinct comparison-anchor label string** AND the coverage
  window (size + since), while that element contains NO "Bitcoin"/"anchoring"/"OpenTimestamps" copy —
  proving the two anchor panels are separate and distinctly labelled.
- A test asserts the comparison-anchor renders for a hub with **NO OTS row** (so §5 is omitted) — i.e.
  the comparison-anchor is independent of the Bitcoin anchor (the two panels are decoupled).
- **Mutation (non-vacuous):** forcing `HasComparisonAnchor = false` (or removing the template block)
  makes `TestCertificateComparisonAnchor` FAIL; reverting restores green. (Record this for review.)
- `GOOS=js GOARCH=wasm go build ./internal/didweb ./internal/index ./internal/badge` still builds
  (certificate stays server-side-only; no non-WASM-pure closure leaked into a WASM-shared package).

## Done When
`mise run check` is green and the certificate renders a distinctly-labelled comparison-anchor element
that is separate from and independent of the §5 Bitcoin-anchor (with "anchoring" copy staying
Bitcoin-only), proven by the new `TestCertificateComparisonAnchor` test and its mutation.

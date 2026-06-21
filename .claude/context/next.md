# Next Work Package

## Step: Certificate-of-inclusion skeleton — realm-wide `/inclusion/{iscc_id}` page (§1 Subject clause + decode→resolve chain)

## Advances
M-UI (Evidence Ledger frontend) — the one remaining open Verify criterion:

> "the **realm-wide certificate** (`/inclusion/{iscc_id}`, keyed by the self-describing ISCC-IDv1 —
> decode realm + 12-bit `hub_id`, resolve the issuing hub via the registry) for a known id renders the
> numbered evidence clauses (subject + position; checkpoint `(size, root)`; inclusion proof; signing
> key; anchor state; full per-id record history incl. any deletion) and offers a **downloadable proof
> bundle** … while an unknown id renders the documented 'not found in log' state (200, never 5xx) …"

This is the **largest** remaining M-UI slice and the only one that re-engages the oracle/conformance
gate (the proof-bundle assembler). Per the skeleton-first rule it cannot land in one ≤3-file step, so
this step lays the **verifiable skeleton**: the realm-wide route + the decode→resolve→store-lookup
chain + the **§1 Subject** clause + the documented invalid-id / unknown-id 200 states. It closes the
"decode realm + hub_id, resolve the issuing hub via the registry … unknown id renders the documented
'not found in log' state (200, never 5xx)" portion of the criterion and **wires the already-built
`index.Decode` + `registry.Resolve`** (state.md: both built, resolver "not yet wired into any caller").
The §2–§6 clauses and the proof-bundle assembler are explicit `## Not In Scope` sub-steps for later
iterations so the loop continues the same arc rather than switching to an unrelated refactor.

This is milestone work (not a self-filed backlog drain): the two iterations before this were
trust-root-adjacent resolver hardening with no Verify-closing surface; state.md's Convergence note
explicitly warns against a third consecutive resolver-polish iteration on an unwired leaf, so we push
onto the certificate page proper.

## Goal
Stand up a new `internal/certificate` package serving `GET /inclusion/{iscc_id}`: decode the
self-describing ISCC-IDv1, resolve the issuing hub's domain via the Hub-List, look up that hub's store
row, and render an Evidence-Ledger certificate page whose **§1 Subject** clause + subject banner are
real. A malformed id, an id that resolves to no listed slot / no followed hub, or an id with no indexed
leaf renders the documented "not found in log" / "cannot certify" state at **200**, never 5xx. This
wires the decoder and resolver into their first real caller and lays the package the remaining clauses +
proof-bundle assembler extend.

## Scope
- **Create**: `internal/certificate/handler.go` — the realm-wide certificate handler (§1 Subject
  skeleton).
- **Create**: `internal/certificate/cert.html` — the embedded Evidence-Ledger certificate template
  (document chrome + `← Realm index` back-link + certificate head + subject banner + §1 Subject clause +
  the two-tier honesty panel; §2–§6 are `{{if}}`-gated placeholders that render nothing yet).
- **Modify**: `cmd/iscc-monitor/main.go` — mount `certificate.Handler` at the realm-wide subtree
  `/inclusion/` in `buildMux`, building the `*registry.HubList` once at startup (see Implementation
  Notes for the minimal interim wiring that avoids a config change this step).
- **Modify (docs)**: `CLAUDE.md` — add the `GET /inclusion/{iscc_id}` route to the HTTP-surface bullet
  list (near line 83, the `GET /<domain>/log/verify` entry).
- **Create (test)**: `internal/certificate/handler_test.go` — golden HTTP-seam tests (see Verification).
- **Reference**:
  - `/workspace/iscc-monitor/.claude/design/ISCC Monitor - Certificate.dc.html` — the authoritative
    mockup (landmark regions: document chrome, `← Realm index` back-link, certificate head, subject
    banner, numbered clauses §1–§6, two-tier honesty panel, Download-proof-bundle + "Verify
    independently →" actions, "Cite as / verifiable cache" footer).
  - `/workspace/iscc-monitor/internal/dossier/handler.go` +
    `/workspace/iscc-monitor/internal/dossier/dossier.html` — the established new-page pattern (embed +
    `template.Must` + `badge.Source` association, buffer-then-200, StatusSource overlay, the
    DS-token/font shell). Copy its shell + the coverage-honesty discipline.
  - `/workspace/iscc-monitor/internal/proofserve/handler.go` `serveRecord` / `serveVerify` — the
    `iscc_id → seqs` lookup (`SeqsForISCCID`, ADR-0008 one-to-many) and the accepted-tree (`LastSize`)
    guards the later clauses reuse; the skeleton needs only subject + position, but read these so the
    view-model is forward-compatible.
  - `/workspace/iscc-monitor/internal/index/iscc.go` +
    `/workspace/iscc-monitor/internal/index/iscc_test.go` — `Decode`; the golden vector
    `MAIGHFECJMOPMIAB` decodes to `{Realm:0, HubID:1}`.
  - `/workspace/iscc-monitor/internal/registry/registry.go` — `ParseHubList` /
    `HubList.Resolve(hubID uint16) (domain, ok)`;
    `/workspace/iscc-monitor/internal/registry/testdata/testnet.yaml` (slot 0 → `sb0.iscc.id`, slot 1 →
    `sb1.amlet.id`).
  - `/workspace/iscc-monitor/internal/store/hubs.go` — `HubSummary` (`HubID/Domain/Origin/LastSize/...`)
    + `ListHubs` for the domain→hubID lookup (same read dossier uses).
  - `/workspace/iscc-monitor/cmd/iscc-monitor/main.go` — `buildMux` / `registerHubs` (where the route
    mounts and where the realm entries already live).
  - `/workspace/iscc-monitor/.claude/context/learnings/registry.md`,
    `/workspace/iscc-monitor/.claude/context/learnings/index.md`,
    `/workspace/iscc-monitor/.claude/context/learnings/http-surface.md` — package pitfalls (resolver
    `ForceQuery` fail-open; decoder fail-closed nibbles; the `iscc_id`/seq/accepted-tree contract; the
    mux-mount + CSS-literal + buffer-then-200 SSR traps).

## Not In Scope
- **The proof-bundle assembler** (`{checkpoint, inclusion/consistency proof, record bytes, hub key,
  ots?}`) and the **Download proof bundle** action wiring — this is the oracle-gated half; it is its own
  later step (the reviewer must mutation-prove the served bundle's inclusion proof non-vacuous +
  `notecheck`/golden-vector parity). The skeleton's button renders as a disabled/placeholder element.
- **Clauses §2 Checkpoint, §3 Inclusion proof, §4 Signing key, §5 Bitcoin anchor, §6 Record history** —
  later sub-steps. Render them as `{{if .HasClauseX}}`-gated placeholders that emit nothing this step,
  so the template grows without rework. (§5 Bitcoin anchor in particular waits on the OTS milestone — a
  not-yet-anchored root must render the "pending" state, never an error.)
- **The `ForceQuery` fail-open fix** in `registry.go` (`hubDomain`, line 188) — the open `normal` issue.
  Folding it would push to 4 non-test/doc files. It rides the **next** certificate slice (§2/§3, which
  also touch the resolved domain), exactly as the review/state recommend; the testnet fixture uses clean
  `https://host` urls so resolution is correct for this skeleton.
- **The ADR-0011 Go 1.26 + iscc-lib bump** — toolchain-gated (local is go1.24.13; mise cannot provision
  Go 1.26 this iteration; confirmed via `go version`). Do NOT flip `go.mod`'s `go` directive.
- **The WASM tier-2 "verify in your browser" result** — lands in the WASM milestone; the tier-1/tier-2
  honesty panel + the static "Verify independently →" link must be present and visually distinct, but the
  tier-2 *result* is not computed here.
- **A new `internal/config` field / env var** — keep the Hub-List wiring interim (see Notes); a config
  change is its own decision, not this skeleton's.

## Implementation Notes
- **Route shape: realm-wide subtree, not per-hub.** Mount `certificate.Handler` at the exact subtree
  `"/inclusion/"` in `buildMux` (alongside `/`, `/metrics`, `/healthz`, `/_ds/`). `http.ServeMux`
  subtree matching gives the handler paths like `/inclusion/MAIGHFECJMOPMIAB`; extract the id with
  `strings.TrimPrefix(r.URL.Path, "/inclusion/")`. A bare `/inclusion/` (empty id) → the documented
  "no id / not found" 200 state. Non-GET → 405 (mirror dossier). This subtree is disjoint from every
  `/<domain>/log/` mirror subtree and every `/<domain>` dossier exact mount, so it never shadows them —
  the per-hub JSON proof route stays `/<domain>/log/inclusion` (a DIFFERENT mount; do not touch it).
- **Decode→resolve→lookup chain (the trust root of this path):**
  1. `id, err := index.Decode(rawID)` — a decode error → the documented invalid-id 200 state (never 5xx).
  2. `domain, ok := hubList.Resolve(id.HubID)` — `!ok` (unknown slot) → 200 "not found in this realm".
     (`Resolve` resolves on `HubID`; the skeleton may surface `id.Realm` in §1 copy but resolves on the
     slot.)
  3. Find the store row for `domain` via `st.ListHubs(ctx)` (same read dossier uses) → its `HubID`. A
     resolved domain with no followed hub → 200 "hub not followed by this monitor".
  4. `seqs, err := st.SeqsForISCCID(ctx, hubID, rawID)` (ADR-0008 one-to-many); `len(seqs)==0` → the
     documented "not found in log" 200 state. The subject position is `seqs[0]` (ascending; the
     deterministic default, matching `serveVerify`).
- **Fail-closed / coverage-honesty discipline (ADR-0001):** every "cannot certify" branch is a **200**
  with an honest explanation, never a 5xx and never a fabricated proof. A genuine infra fault (a
  `ListHubs` / `SeqsForISCCID` DB error) is the only 500. Render into a `bytes.Buffer` first so a
  template/store error is a 500 BEFORE any 200 is committed (copy the dossier/proofserve pattern).
- **Hub-List wiring without a config change (KISS, interim):** production has no Hub-List document path
  yet, and `realm.txt` is line-based domains (a DIFFERENT format — do NOT conflate it with the YAML
  Hub-List). For THIS skeleton, build the `*registry.HubList` in `cmd/iscc-monitor` from the parsed
  realm entries by assigning slot i = entry i in `realm.txt` order — this matches the testnet fixture
  (sb0=slot0, sb1=slot1) and needs no new env var. Pass it into
  `certificate.Handler(hubList *registry.HubList, st *store.Store, statuses certificate.StatusSource)`
  so swapping in a real Hub-List source later is a one-line change. Document this interim mapping with a
  clear TODO and flag it so `review` weighs it. If hand-constructing a `*registry.HubList` in `cmd/` is
  awkward (the `Hubs` slice needs `*uint16` HubIDs), prefer a tiny exported `registry` constructor over
  a config change — but keep both `registry.go`'s and config's existing surface untouched if at all
  possible (the route mount + handler are the load-bearing change).
- **Template / DS shell:** copy the dossier shell verbatim (DS `<link>`s to `/_ds/...`, self-hosted
  fonts, NO external CDN URL in the body — the `noExternalCDN` ban is asserted at the seam). Use the
  **UNQUOTED** `[data-status=verified]` CSS attribute form if you inline any badge color block (the
  http-surface CSS-literal trap). Render the mockup's landmark regions: document chrome header (ISCC
  logo + "Trust & Transparency Monitor" mark + instance-identity block + `verify ↗ monitor.iscc.codes`
  tier-2 link), `← Realm index` back-link, certificate head (ref + issued + instance), subject banner
  ("<id> is included in the transparency log of <hub> at position N"), **§1 Subject** clause, the
  two-tier honesty panel, the "Cite as / verifiable cache" footer. §2–§6 are gated placeholders.
- **html/template, not text/template** — the id, domain, and seq auto-escape. Associate `badge.Source`
  into the set if you render the hub's status badge (mirror dossier's
  `template.Must(t.Parse(badge.Source))`).
- **Oracle/conformance gate is N/A for THIS step** (pure HTML render of decode + resolve + a store
  `SeqsForISCCID` lookup; no signature / RFC-6962 / Merkle / proof / did:web path, no new crypto). It
  APPLIES to the later proof-bundle sub-step — call that out so the next `define-next` re-engages it.
- **Correctness rules in play (learnings index):** "Origin = `<domain>/log`" (the certificate links into
  `/<domain>/log/...` for later clauses — derive, never hardcode the bare domain); "`iscc_id → seq` is
  one-to-many, verification is schema-agnostic (ADR-0008)" (subject defaults to `seqs[0]`, interpret
  nothing); "Coverage honesty (ADR-0001)" (every cannot-certify branch is an honest 200).

## Verification
- `mise run check` is green (`go build ./... && go vet ./... && go test ./...`; `gofmt -l .` empty).
- `go test -count=1 ./internal/certificate` passes uncached.
- `go build ./...` (the server build) passes — the new package compiles and mounts.
- **Known-id golden chain (seam test):** with a fixture store holding the testnet hubs and at least one
  indexed leaf for a known id under the hub at slot 1 (`sb1.amlet.id`), `GET
  /inclusion/MAIGHFECJMOPMIAB` returns `200 text/html`, and the body contains: the subject id
  (`MAIGHFECJMOPMIAB` or `ISCC:MAIGHFECJMOPMIAB`), the **resolved** hub domain `sb1.amlet.id`, the
  position (`seqs[0]`), the **§1 SUBJECT** clause marker, the `← Realm index` back-link, the
  `verify ↗ monitor.iscc.codes` tier-2 link, and the two-tier honesty panel — and contains **no**
  external CDN URL outside the same-origin `/_ds/` links (a `noExternalCDN`-style assert over the body).
- **Unknown-id (resolves but not in log):** `GET /inclusion/<known-realm-id-with-no-indexed-leaf>`
  returns `200 text/html` with the documented "not found in log" state, never 4xx/5xx.
- **Malformed-id:** `GET /inclusion/NOTANISCCID` returns `200 text/html` with the documented invalid-id
  state, never 4xx/5xx (a decode error is a verdict, not a fault).
- **Unresolvable slot:** an id whose `hub_id` slot is not in the Hub-List returns the documented "not in
  this realm" 200 state.
- **Method guard:** a non-GET to `/inclusion/...` returns 405.
- **Non-vacuity (reviewer-reproducible):** the known-id test must FAIL if `Resolve` is stubbed to return
  the wrong domain or `Decode` is bypassed — assert on the *resolved* `sb1.amlet.id` (derived from the
  real decode→resolve chain), not on a literal the template could carry unconditionally.

## Done When
`internal/certificate` serves `GET /inclusion/{iscc_id}` mounted in `buildMux`, the decode→resolve→
store-lookup chain renders a real §1 Subject clause + subject banner for the known golden id and an
honest 200 state for every malformed / unresolvable / not-in-log id, CLAUDE.md lists the new route, and
all Verification checks pass.

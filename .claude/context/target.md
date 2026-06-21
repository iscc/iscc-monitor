# Target — iscc-monitor v1

> Authoritative specs: `.claude/prd/0001-iscc-monitor-v1.md`, `.claude/plans/cosmic-baking-octopus.md`,
> `.claude/adr/0001`–`0011`, glossary in `CLAUDE.md`. Where this file and an ADR/PRD disagree, the
> ADR/PRD wins. This file is the *fixed target* the CID loop advances toward — the desired end-state
> plus the bar every increment is verified against.

## Stack (locked — ADR-0003, ADR-0011)

Go 1.26 (bumped from 1.24 to consume the iscc-lib Go codec — ADR-0011; the config bump + dependency
add is the active increment in `issues.md`), `CGO_ENABLED=0`, module `github.com/iscc/iscc-monitor`.
**Reuse, do not reimplement** — the transparency stack: `transparency-dev/tessera` (`client`, `api`,
`api/layout`, `fsck`), `transparency-dev/merkle` (`rfc6962`, `proof`), `transparency-dev/formats`,
`golang.org/x/mod/sumdb/note`, `nbd-wtf/opentimestamps`, `modernc.org/sqlite`; **and the ISCC codec**:
`github.com/iscc/iscc-lib/packages/go` (pure-Go, `CGO_ENABLED=0`, conformance-tested vs `iscc-core`;
pinned v0.5.0) is the canonical ISCC en/decoder for generic ISO 24138 / Version-0 codec work.
**Carve-out (ADR-0011):** ISCC-IDv1 (`MainType=ID`, `Version=1`) is **not** in iscc-lib yet — its
`decodeHeader` rejects `Version>0` — so `internal/index.Decode` stays the **interim** in-repo port
until upstream ships it (migration trigger = [iscc/iscc-lib#43](https://github.com/iscc/iscc-lib/issues/43)
+ the `internal/index` tripwire test). Read-only vendored reference copies live under `cauldron/`
(gitignored) — port/oracle against them, never import them.

**Frontend (locked — ADR-0010).** The v1 web surfaces follow the **Evidence Ledger** direction on the
**ISCC Design System v2**; the build source of truth is
`.claude/design/ISCC Monitor - Developer Handoff.dc.html` (+ the `_ds/` token bundle), subordinate to
the ADRs/PRD (where they disagree, the ADR/PRD wins — flag it). Server-rendered (`html/template` +
`go:embed`), **no-JS baseline** for the dashboard & log browser; WASM re-verification is additive. DS
tokens + Readex Pro / JetBrains Mono fonts are **self-hosted** (embedded, no runtime CDN). Hub status is
conveyed by **icon + label + silhouette** (grayscale-safe), never hue alone.

## Quality bar (the gate — non-negotiable)

`mise run check` is green: `go build ./...`, `go vet ./...`, `go test ./...` all pass and `gofmt -l .`
is empty. Testing is **integration-first at the outbound-fetch seam** (PRD "Testing Decisions"):
drive follower → store → API against fixtures and assert on *observable outputs* (SQLite state,
`/metrics`, REST / proof-bundle responses, verifier verdicts) — never on follower internals. Pure
units (`proof/verify`, `didweb`) get table-driven golden-vector tests, with the two live testnet
hubs `sb0.iscc.id/log` and `sb1.amlet.id/log` as golden vectors (`.claude/derive_vkey.py` is the
reference). **Never weaken a gate to pass** — no `//nolint`, `t.Skip`, swallowed errors, or build-tag
exclusions to dodge checks. Fix the root cause.

**Oracle / conformance gate (crypto path — PRD "External oracle gates").** Any change to signature
verification, RFC-6962 / Merkle consistency, or proof code must pass the monitor's **conformance
tests** before a PASS verdict — they run as ordinary `go test` / `mise run check` once the package
exists and encode: golden-vector parity (`derive_vkey.py` vectors), `fsck` root-rebuild over the
`SQLiteFetcher`, and the inclusion cross-check against the hub's own `IsccLogInclusionProof`. The
**fully-independent** external oracle `notecheck` (a separate binary fed checkpoint text) is built and
**shelled out in CI** — not `go run` from gitignored `cauldron/` (those are module-less `main.go`
files with external deps: no local compile path). Independence honesty: `notecheck` and the
hub-computed `IsccLogInclusionProof` are the true external oracles; the ported `runfsck` hasher /
`fsck`-over-`SQLiteFetcher` is an in-process structural **self-check** (it shares the monitor's code, so
it can share its bugs). The LLM reviewer is **not** ground truth for crypto correctness — a
green-but-wrong verify (e.g. one accepting a malformed proof) must not ship on `mise run check` + an
LLM PASS alone. A conformance/oracle regression is a gate failure.

**Coverage (ratified decision — no percentage gate).** There is **deliberately no coverage-percentage
gate**. This overrides the general "every project needs tests / 100 % coverage" expectation (and
iscc-lib's 100 % gate) in favour of *this* project's PRD "Testing Decisions": correctness is proven by
the seam-based integration tests + golden vectors + the external oracles above — not by a line-coverage
number, which would force brittle tests on wiring / `main` and tempt the internal-detail assertions the
PRD forbids. Revisit only if the testing strategy itself changes.

## Milestones (advance in ADR-0004 / ADR-0010 order)

### M1 — Read-only Monitor  `[not started]`

config + realm registry (domains only) + did:web resolution + `vkey`/`origin` (golden-tested) +
per-hub follower (verify Ed25519 signed-note + the three-trigger consistency check
fork/shrink/equivocation, persist, freeze) + per-network SQLite + coverage tracking
(`monitored_since`) + structured logs + `/metrics`. Survives restart.
**Verify:** `origin("https://sb0.iscc.id") == "sb0.iscc.id/log"`; `verifierKey` byte-matches
`derive_vkey.py` for both hubs; synthetic fork/shrink/equivocation each → correct `violations.kind` +
`frozen=1` + exactly one alert + other hubs unaffected + evidence survives restart.

### M2 — Aggregator  `[not started]`

tiles + entry bundles as SQLite BLOBs (partial-tile discipline: `is_full` only at `width==256`) +
`SQLiteFetcher` implementing `client.Fetcher` so `fsck.New(...)` runs on the local store +
`iscc_index` (schema-agnostic; `iscc_id → seq` one-to-many; stores raw `note.$schema`) +
`inclusion`/`consistency`/`entries` served from the local store via `ProofBuilder` (never re-hitting
the hub). **Verify:** `fsck` rebuilds each accepted root from the `SQLiteFetcher`; computed inclusion
proof matches the hub's `evidence.IsccLogInclusionProof` for sampled `iscc_id`s.

### M3 — Trust API + dashboard  `[not started]`

REST surface (CORS on every public GET) + `verify-for-me` + server-rendered dashboard (status,
coverage, lag, violations, OTS) + log browser + raw tlog-tiles mirror at canonical paths.
**Verify** (all asserted at the HTTP seam against fixtures — observable outputs, never handler
internals): every public GET carries `Access-Control-Allow-Origin: *`; `GET
/<domain>/log/verify?iscc_id=<known-id>` returns the documented `verify-for-me` JSON verdict (hub
status + checkpoint `(size, root)` + inclusion result), and a malformed/unknown id returns the
documented non-verified verdict, never a 5xx; `GET /` returns `200 text/html` listing **every** realm
hub with its glossary status + coverage window (golden-tested on a fixture store); `GET
/<domain>/log/` (log browser) returns `200 text/html` exposing the mirrored checkpoint `(size, root)`
with links into `entries`/proofs.

### M-UI — Evidence Ledger frontend  `[not started]`

Dress and extend M3's functional SSR surfaces into the **Evidence Ledger** design (ADR-0010): embed the
ISCC Design System v2 tokens + self-hosted fonts; render every hub via a five-status `HubStatusBadge`
template partial (icon + label + silhouette); and add the screens M3 left functional-only — **hub
dossier** (incl. the frozen Exhibit), **log-browser record list** (paginated over `iscc_index`) +
**single record**, and the **certificate of inclusion** (HTML proof result + downloadable proof
bundle). Make the full five-status taxonomy (`verified`/`unresolvable`/`unverified`/`frozen`/`inactive`)
store-provable so the badge renders it honestly. SSR + no-JS is the hard baseline; WASM is the next
milestone. Each surface is built to **named-region parity** with its specific mockup — the served HTML
must carry that mockup's landmark regions, affordances, and cross-surface navigation (the design-parity
bar below), so the surfaces resemble the design and form one navigable whole, not just pass behaviorally.
**Verify** (all asserted at the HTTP seam against fixtures — observable no-JS HTML, never handler
internals; the rendering path's oracle gate is N/A, but the proof-bundle assembler shares the crypto
path and MUST keep the conformance/oracle gate green):
every SSR surface (`/` realm index, hub dossier, log-browser record list, single record, certificate)
returns `200 text/html`, embeds the DS tokens + self-hosted fonts with **no external CDN URL in the
body**, and is **complete with JavaScript disabled** (content in the served HTML, not script-gated);
`HubStatusBadge` renders **all five** statuses each with a distinct text label **and** a distinct
inline-SVG silhouette (golden-tested per status), with `frozen` rendered as the categorically-distinct
**Exhibit** (violation kind + detected-at + "do not trust new state", non-dismissable), visibly
different markup from the `unresolvable`/`unverified` caution; the coverage window (`monitored_since`
size + RFC-3339 time, or an explicit "coverage just started" / "no coverage yet") shows for **every**
hub on the index + dossier and a pre-coverage state never renders as a guarantee (ADR-0001); the
**Bitcoin-anchor** panel and the **comparison-anchor** panel are separate, distinctly-labelled elements
("anchoring" copy is Bitcoin-only; a not-yet-anchored root renders the normal "pending" state, not an
error); the log-browser record list paginates via plain links (`?from=…[&n=…]`, no-JS), newest-first,
each row links to its single-record page, and an empty log renders the informative empty state (200);
the single-record page renders `declaration`, `deletion` (a new record — original preserved) **and an
unknown `note.$schema`** without erroring; the **realm-wide certificate** (`/inclusion/{iscc_id}`, keyed
by the self-describing ISCC-IDv1 — decode realm + 12-bit `hub_id`, resolve the issuing hub via the
registry) for a known id renders the numbered evidence clauses (subject + position; checkpoint
`(size, root)`; inclusion proof; signing key; anchor state; full per-id record history incl. any
deletion) and offers a **downloadable proof bundle** `{checkpoint, inclusion/consistency proof, record
bytes, hub key, ots?}`, while an unknown id renders the documented "not found in log" state (200, never
5xx) and the tier-1 ("the monitor reports") vs tier-2 ("verify in
your browser") affordance is present and visually distinct (the tier-2 result itself lands in the WASM
milestone); status stays legible in grayscale + colorblind-safe (icon+label+silhouette, not hue);
`mise run check` green; any new store status-derivation stays a leaf read (store keeps no
`net/http`/web dependency).

**Design parity (named-region — the layout/affordance bar, ADR-0010 handoff).** Beyond behaving
correctly, each SSR surface must render the **landmark regions and affordances of its specific mockup**,
asserted at the HTTP seam by the presence of those regions in the served **no-JS** HTML — a region
checklist, **never a pixel diff**. The mockup is **authoritative for layout and affordances** but
**subordinate to the hard constraints above** (no-JS / no-CDN / self-hosted fonts+tokens / grayscale-safe
/ store-leaf): where a mockup conflicts — it links `fonts.css` at jsDelivr and ships a JS component
runtime — the constraint wins (we self-host + SSR; an interactive control becomes its plain-link/`GET`-form
equivalent) and the deviation is flagged, not silently dropped. Visual polish below the named-region
level (exact spacing, weight, "feel") is **not** an autonomous per-iteration pass/fail gate, but is no longer left unverified: when a diff touches an SSR surface, the `review` step runs a headless **visual pass** with `agent-browser` (ADR-0012) — it screenshots the changed surface and its `.dc.html` mockup and files the visual deltas as `issues.md` entries (generative, best-effort; it degrades gracefully and never stalls the loop when the browser is unavailable). The residual aesthetic judgement is a **mandatory M-UI exit gate**: M-UI does not reach DONE until every surface has passed that visual pass and a human has signed off, deviations filed as issues. Two cross-cutting
requirements hold on every surface:
- **Document chrome + instance identity.** Every surface carries the shared handoff header: the ISCC logo
  + "Trust & Transparency Monitor" mark, the **instance-identity** block (this instance's domain +
  operator + realm), and the **`verify ↗ monitor.iscc.codes`** tier-2 link — legible instance identity
  (handoff invariant 9) and the tier-1/tier-2 split present on the page itself.
- **Navigation closure.** The `/` → hub dossier → log browser → single record → certificate chain is
  fully traversable with **JavaScript disabled**: forward links *and* `←` breadcrumb back-links both
  present, so the index is enterable and no surface is a dead end (this is what the realm-index→dossier
  row link, today absent, must restore).

Per-surface landmark regions (each keyed to its mockup file under `.claude/design/`):
- **`/` realm index** — `ISCC Monitor - Realm Index.dc.html`: the **claim-lookup hero** foregrounded
  *above* the register (ISO-24138 eyebrow + plain-language explainer + ISCC-ID lookup → certificate); the
  **realm register** table (`#` · hub·domain · coverage-since + size · checkpoint size · Bitcoin-anchor
  dot+label · status badge) with a "N hubs followed & mirrored" count, **every row a link to that hub's
  dossier**, and frozen rows visibly tinted; the coverage footnote.
- **hub dossier** — `ISCC Monitor - Hub Dossier.dc.html`: `← Realm index` back-link; the trust-document
  head (eyebrow "Hub dossier", hub name + domain, `md` `HubStatusBadge`, "Compiled by <instance> ·
  <time>"); the numbered sections **§1 Identity** (did:web), **§2 Coverage** (since + size), **§3 Latest
  checkpoint** (size + time), **§4 Bitcoin anchor** (dot + label + "run `ots verify`"); the **§5
  observation log**; the frozen **Exhibit** (required above) above the sections; the two actions ("Prove
  an ISCC-ID in this hub →" certificate, "Browse the log →").
- **log browser / record list** — `ISCC Monitor - Log Browser.dc.html`: `← <hub> dossier` back-link; the
  head (eyebrow "Log browser", hub name, "<domain> · N records mirrored"); the **plain-link pager**
  (newer/older + "seq X–Y of Z" range, disabled at the ends — the no-JS `?from=…` form of the mockup's
  buttons); the record table (`Seq · Type · ISCC-ID · Logged`) with a per-row **type badge**
  (declaration/deletion/unknown) and **every row a link to its single record**; the append-only footnote.
- **single record** — `ISCC Monitor - Single Record.dc.html`: `← Log browser` breadcrumb; the older/newer
  record **link** stepper; the record head (eyebrow "Log record · seq N", id, type badge); the gated
  **deletion** and **unknown-type** notices; the field grid (ISCC-ID, type + `note.$schema`, hub,
  position, logged, **raw bytes** block); the actions ("Prove this record's inclusion →", "Back to list").
- **certificate of inclusion** — `ISCC Monitor - Certificate.dc.html`: `← Realm index` back-link; the
  certificate head (ref + issued + instance) and subject banner ("<id> is included in … at position N");
  the numbered clauses **§1 Subject · §2 Checkpoint (size, root) · §3 Inclusion proof
  (leaf→siblings→root) · §4 Signing key (did:web) · §5 Bitcoin anchor · §6 Record history** (incl. any
  deletion); the **two-tier honesty** panel (Tier 1 "the monitor's account" vs Tier 2 "you verify
  yourself"); the **Download proof bundle** primary action + "Verify independently →" link; the "Cite as
  / verifiable cache" footer.

The `HubStatusBadge` partial follows `HubStatusBadge.dc.html`; Surface C's verifier app follows
`ISCC Monitor - Independent Verification.dc.html` in the WASM milestone below.

### WASM verifier upgrade  `[not started]`

`internal/proof/verify` → `GOOS=js GOARCH=wasm`, lazy-loaded progressive enhancement that elevates the
Evidence Ledger's **tier-2** result ("your browser verified…") on the M-UI certificate/dossier, plus the
standalone **Independent Verification** verifier app (`ISCC Monitor - Independent Verification.dc.html`,
Surface C) at `monitor.iscc.codes` (monitor-agnostic
via `?monitor=<url>`); reproducible build + published hash + SRI pin (ADR-0003, ADR-0010). **Verify:**
identical vectors yield identical verdicts (WASM vs server); the verifier artifact hash matches the
published value; a `(size, root)` mismatch renders the guided split-view alert, not a dead error.

### OTS / Bitcoin anchoring  `[not started]`

stamp each distinct observed root daily (`UNIQUE(hub, tree_size, root)`) + background upgrade loop
(pending → Bitcoin-confirmed) + serve `.ots`; **never blocks the follower**. **Verify:** a stamped
root upgrades to Bitcoin-confirmed and the served `.ots` verifies with the standard `ots` client.

### M7 — DEFERRED (out of v1)

multi-monitor gossip + cosigning (C2SP witness cosignatures) + witness endpoint
(`tlog-witness add-checkpoint`) + cross-monitor split-view comparison. Out of scope; never blocks DONE.

## Done When

Every v1 milestone (M1 → M2 → M3 → M-UI → WASM → OTS) meets its **Verify** criteria with `mise run
check` green and no open `critical` or `normal` issue in `issues.md`. M7 is explicitly out of scope.

# Next Work Package

## Step: Percent-encode the port when building a hub's `did:web:` DID on the certificate §4 clause and the proof bundle

## Advances
This step does **not** close an unmet milestone Verify criterion directly — every remaining one is
human-blocked, offline-unprovable, or design-first/STOP-candidate:
- WASM "the verifier artifact … [is published at its] published value" — **human-blocked** (the live
  Pages deploy needs a one-time repo-Settings step; a workflow file cannot self-enable Pages).
- WASM signature-half (browser did:web resolution + checkpoint-note signature verify) — the handoff
  `**Next:**` flags this **design-first / STOP-candidate**; building it blind is exactly what the
  review warned against (do a design pass first, do not loosen `verifier.html`'s "hub-signed root" copy).
- OTS "a stamped root upgrades to Bitcoin-confirmed" — **offline-unprovable** (needs a live calendar
  + real BTC confirmation).
- M-UI exit visual-pass + human sign-off (ADR-0012) — **human-gated**.

It therefore closes a `normal` issue per the protocol's "weigh `normal` against the gap" rule, in the
exact productive code-closable direction the handoff named (`§5`/`host:port` DID encode / SSR-parity).
It preempts because it is a **trust-root-adjacent correctness defect on a Tier-1 self-verifiable
surface**: the certificate §4 clause and the downloadable proof bundle both name a `did:web:` DID that,
for a `host:port` hub, denotes a **different** key-resolution target than the key was actually resolved
from — a false identity claim a client follows to fetch the wrong did.json. Issue: **"Certificate §4 AND
the proof bundle build `did:web:` + raw domain, mis-rendering a `host:port` hub's DID"** (`normal`).

## Goal
Make the certificate §4 signing-key DID and the proof-bundle `Hub.DID` correctly percent-encode a hub's
port colon (`host:port` → `did:web:host%3Aport`), reusing the resolver's existing encoding idiom, so the
DID denotes the same host the key was resolved from on both Tier-1 self-verifiable surfaces.

## Scope
- **Create**: (none)
- **Modify**: `internal/certificate/handler.go` (the two `"did:web:" + data.Domain` sites: the proof
  bundle at line 553 and §4 at line 874 — fix BOTH together via one shared local helper). **One
  production file.**
- **Reference**:
  - `.claude/context/learnings/certificate.md` (§4 DID bug + the bundle's inherited copy; "Any surface
    building a DID from a domain must `%3A`-encode the port"). Read before editing.
  - `.claude/context/learnings/didweb.md` ("did:web colon must be percent-encoded before `DocumentURL`
    (`strings.Replace(host, ":", "%3A", 1)`)" — the exact idiom to reuse).
  - `internal/didweb/url.go` (`DocumentURL` docstring: "first segment is the percent-encoded host[:port]").
  - `internal/certificate/handler.go:549-564` (bundle) and `:867-880` (§4) for the two call sites;
    `handler.go:261` + `:411` for the docstrings to keep evergreen.

## Not In Scope
- The `hubDomain` ForceQuery fail-open in `internal/registry/registry.go` (separate `normal` issue,
  separate file — keep this step to one file).
- The certificate §5 OTS-digest-binding `normal` (different clause, different fix).
- Adding a `%3A`-encode helper to a shared package or refactoring `didweb` to export an encoder — keep
  the helper local to `handler.go` (KISS; the resolver's idiom is two lines, not worth a new export).
- Any WASM / signature-half / Pages / OTS-Bitcoin work (all blocked per `## Advances`).
- The §6 `· at` timestamp, the certificate tier-2 honesty-copy overstatement, or the `/` sub-region
  deltas (each its own later `normal`).

## Implementation Notes
- Add a tiny local pure helper in `handler.go`, e.g.
  `func didWeb(domain string) string { return "did:web:" + strings.Replace(domain, ":", "%3A", 1) }`,
  and use it at BOTH sites: replace `"did:web:" + data.Domain` at line 553 (`bundle.Hub.DID`) and at
  line 874 (`data.SigningKeyDID`). `strings` is already imported (`handler.go:91`) — no new import.
- Use `strings.Replace(domain, ":", "%3A", 1)` (replace only the FIRST colon), the **exact** idiom the
  resolver uses (`learnings/didweb.md`: `strings.Replace(host, ":", "%3A", 1)`) — do NOT hand-roll a
  different encoder, and do NOT `url.PathEscape` the whole domain (that would over-encode `.`-bearing
  hostnames; a no-port domain like `sb0.iscc.id` must round-trip byte-identical to today's
  `did:web:sb0.iscc.id`).
  Correctness rule (always-loaded `learnings.md` + `didweb.md`): a `host:port` did:web id splits at the
  colon into a path segment unless the colon is `%3A`-encoded, so the DID would otherwise denote a
  different document than the key resolved from.
- Update the §4 field docstring at `handler.go:261` ("`SigningKeyDID` is … `\"did:web:\" + Domain`") and
  the bundle DID docstring near `:411` ("`\"did:web:\" + domain`") to state the port is `%3A`-encoded —
  keep the comments evergreen and matching the new behavior (CLAUDE.md "evergreen comments").
- Edge cases: a clean `sb0.iscc.id` (no colon) is unchanged; an IPv6-literal host is not a realm concern
  (the fixture realm is hostnames); only the single `host:port` colon is load-bearing.
- This is a render-string fix only — it touches NO signature / Merkle / proof-verify code, so the
  conformance/oracle gate is N/A (no `derive_vkey.py` / `fsck` / inclusion path). `internal/certificate`
  is server-side only (`internal/ots` is already in its closure), so there is no WASM-purity concern.

## Verification
- `mise run check` is green (build + vet + test all 27 packages, `gofmt -l .` empty).
- `mise exec -- go test -count=1 -run 'TestCertificateSigningKey|TestCertificateProofBundle' ./internal/certificate` passes.
- A new (or extended) test seeds a hub whose `data.Domain == "localhost:8443"` (a `host:port` form) and
  asserts the rendered §4 DID is `did:web:localhost%3A8443` (and the served `.bundle` `Hub.DID` is the
  same), NOT `did:web:localhost:8443`; reverting the `%3A` encode at either site makes that test FAIL
  (mutation-provable, both sites).
- A clean-domain regression case asserts a no-port hub (`sb0.iscc.id`) still renders exactly
  `did:web:sb0.iscc.id` (no spurious encoding) on both §4 and the bundle.

## Done When
`mise run check` is green and a new `host:port` test asserts `did:web:host%3Aport` on both the §4
clause and the proof bundle (mutation-failing when either `%3A` encode is reverted), with the clean-domain
case unchanged.

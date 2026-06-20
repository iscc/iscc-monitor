---
status: accepted
---

# Three trust dimensions, and an in-browser verifier for non-technical users

Verifying the monitor is not one trust question but three orthogonal ones, with
different parties in the trust path:

| Claim | Proof | Trust path | Trustless? |
|---|---|---|---|
| **Inclusion** — R is committed in hub H at root Rt | hub-signed checkpoint + Merkle proof, verified by the client | the **hub's** key only; monitor is a verifiable cache | yes, if verified client-side |
| **Time / anti-rewrite** — this root existed before block B | OTS / Bitcoin anchor, `ots verify` | nobody (Bitcoin PoW) | yes |
| **Equivocation** — H showed everyone the same history | client compares its `(size,root)` vs the monitor's tree | the **monitor** as reference | partial, until gossip (M7) |

The monitor is therefore designed to be a **verifiable cache, not a trusted
oracle**: fetching data from it is not the same as trusting its verdict. A
"verify-for-me" REST endpoint trusts the monitor; a self-contained proof bundle
the client verifies itself does not. OTS and gossip strengthen the two dimensions
where the monitor would otherwise be trusted; client-side verification removes it
from the inclusion path entirely.

**Decision.** The primary audience includes non-technical business/legal users who
open monitor.iscc.id to read the trust situation and browse logs. For that
audience, an **in-browser (WASM) verifier is in v1 scope**, because it is the only
form client-side verification can take for someone who will not run a CLI —
without it they are merely trusting the monitor's rendered badges. It is built on
the shared pure `internal/proof/verify` package (identical logic server-side, CLI,
and WASM) and sequenced as a **progressive enhancement** on the M3 server-rendered
dashboard (readable first; "verified in your browser" after). The CLI/library +
self-contained bundles ship too — for auditors, automation, and to *attest* the
WASM build.

## Consequences

- **The monitor can equivocate about its own verifier.** It serves both the data
  and the WASM, so it could serve an auditor the honest verifier and a victim a
  tampered one that always renders "✓". "An auditor attested the verifier" is only
  meaningful if the bytes a user loads are provably the audited bytes. Therefore
  the verifier MUST be (1) a **reproducible build**, (2) its hash **published
  independently of the monitor** (e.g. in iscc-hub and/or the auditor's
  attestation), and (3) **SRI-pinned** on the page. Strongest option: serve the
  verifier from an **independent origin** that only calls the monitor's data API,
  so the monitor structurally cannot swap it. This shrinks the web user's trust
  assumption to a check-once fact about a fixed artifact.
- Keeps Go (ADR rationale: proof-code reuse + one verifier codebase to native +
  WASM). The "static binary" benefit is secondary.
- `internal/proof/verify` must stay pure (no net/os/sqlite) — it is the single
  shared core across server, CLI, and WASM.

## Verifier hosting (resolved 2026-06-20)

Same-origin SRI is insufficient — a malicious monitor serves both the page and a
matching `integrity=` attribute, so it defends only against CDN corruption, not a
malicious origin. The verifier is therefore hosted on an **independent origin**:
**`monitor.iscc.codes`** (GitHub Pages built from `iscc/iscc-monitor`, following
the `*.iscc.codes` repo-docs convention), separate from any monitor *instance*.

- **Monitor-agnostic, one verifier for the federation:** the verifier takes a
  `?monitor=<url>` parameter and verifies data from *any* instance (the Foundation's
  `monitor.iscc.id`, or third-party monitors on their own domains). You audit one
  artifact; it works against every monitor; no monitor serves it, so none can
  tamper with it.
- **Provenance:** Pages-from-repo ties the deployed WASM to a public commit + public
  build — the reproducible-build + published-hash requirement, with no extra infra.
- **`.codes` = code / `.id` = instance:** a monitor instance's dashboard stays on
  its own domain and links "verify independently →" to
  `monitor.iscc.codes/?monitor=<that instance>`.
- **Two requirements:** (1) the monitor's public read API + mirror must be
  **CORS-enabled** (`Access-Control-Allow-Origin: *`; data is public + self-
  verifying) so the cross-origin verifier can fetch it; (2) the canonical verifier
  URL must be **well-known / Foundation-published** — the residual (standard) trust
  assumption is that a user reaches the genuine `monitor.iscc.codes`, not a link
  from a possibly-malicious dashboard.


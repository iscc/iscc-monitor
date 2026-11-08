---
status: accepted
---

# Defer the independent verifier app; monitor.iscc.codes hosts the documentation

ADR-0003 put an in-browser verifier on an independent origin, `monitor.iscc.codes`, so anyone could
re-check a monitor instance's proofs without trusting that instance's own pages. Its first version
(Surface C: `internal/verifier`, rendered into a static site by `cmd/verifier-site` and deployed by
`.github/workflows/pages.yml`) shipped but was not usable:

- every link to it (the certificate's "Verify independently in your browser" button, the masthead and
  the footer) opened the bare site without the `?monitor=&id=` parameters it needs, so it never ran;
- the page offered no field to enter an ISCC-ID or a monitor;
- its split-view "Compare" form submitted a size and root that no code read;
- it could not take the proof bundle a user downloads from a certificate.

**Decision.** Remove the verifier app for v0.1.0 rather than repair it in place.

- Delete `internal/verifier`, `cmd/verifier-site`, `.github/workflows/pages.yml` and
  `.github/pages/CNAME`, and remove every link to the app from the shared chrome and the certificate.
- Keep the certificate's in-browser re-check (the tier-2 panel) and the WASM checker it loads
  (`cmd/wasm`, `/_ds/verify.wasm` and its pinned hash). It runs on the instance and involves no other
  domain. The checker's optional ISCC-ID argument, which only the app passed, stays, so a future
  verifier can use it without changing the pinned artifact.
- `monitor.iscc.codes` will host this application's documentation.
- A verifier app returns later, on a different domain and redesigned: links that carry the monitor
  and the ISCC-ID so it runs on arrival, a form to enter them, and verification of a downloaded bundle
  file without contacting any monitor. The deleted code is in the history up to commit `1e3573f`.

**Consequences.**

- ADR-0003's independent-origin verifier, PRD stories 5, 9 and 10, and target.md's Surface C are
  deferred. ADR-0003's other trust dimensions and the proof bundle are unchanged.
- The masthead no longer carries a tier-2 link. The certificate keeps its two-tier honesty panel; its
  Tier 2 is the downloadable proof bundle and the in-browser re-check.
- Deleting the workflow does not unpublish the site already deployed at `monitor.iscc.codes`.
  Replacing it with the documentation, or unpublishing it in the repository's Pages settings, is a
  separate manual step.
- The Surface C mockups in `.claude/design/` stay as reference for the redesign.

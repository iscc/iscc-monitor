---
status: accepted
---

# Frontend visual verification uses agent-browser (headless), not an interactive browser MCP

ADR-0010 adopted the **Evidence Ledger** frontend and gave M-UI a **design-parity bar** asserted at
the HTTP seam — a *named-region checklist, never a pixel diff*. That bar verifies that a surface's
**landmark regions and affordances** are present in the served no-JS HTML, but it cannot see a
*rendered* page: visual fidelity below the named-region level (spacing, weight, layout, "feel") was
left as an unverified human checkpoint, so the served surfaces can pass every behavioural check while
looking nothing like the mockups (observed on the realm index: bare 4-column ledger, no claim-lookup
hero, no document chrome, rows not links). The CID loop needs a way to *look* at what it renders.

The obvious tool — an interactive browser-extension MCP (`claude-in-chrome`) — does **not** fit: it
needs a desktop Chrome with the extension and interactive site permissions, so it is unavailable in
this dev container and in scheduled/cron loop runs (`/goal`, `/loop`). A visual check that only works
when a human is at the keyboard cannot be part of the autonomous gate.

## Decision

**Adopt `vercel-labs/agent-browser` as the loop's frontend visual-verification tool.** It is a headless
Rust CLI (client-daemon; Chrome-for-Testing via CDP; CLI / MCP / batch interfaces) that runs in the
container, CI, and serverless with no desktop — the same place the loop runs. Pinned at **v0.29.0**.

- **Install (devcontainer).** `npm install -g agent-browser@0.29.0` in the `Dockerfile` (alongside the
  existing Claude Code / Codex global installs; the npm package bundles the **version-matched** skill,
  which the bare github-release binary does not), then `agent-browser install --with-deps` to bake
  Chrome for Testing + its Linux system libraries (`libnspr4`/`libnss3`/`libgbm1`/GTK/… — Chrome fails
  to launch without them) into the image so rebuilds need no re-download. node 22 emits an `EBADENGINE`
  warning (the package declares `node >=24`) but installs and runs the prebuilt binary fine.
- **Skill.** The bundled Claude Code skill is vendored into `.claude/skills/agent-browser/SKILL.md`
  (a `hidden: true` discovery stub that points agents at `agent-browser skills get core` for the
  version-matched usage guide). Re-vendor it whenever the pin moves.
- **Loop integration.** The `review` step (which already has `Bash` + `Read`) shells out to the CLI:
  when a diff touches an SSR surface (`internal/dashboard|dossier|web|certificate` or a template), it
  launches the instance against fixtures that exercise the rich states, screenshots the changed surface
  and its `.claude/design/*.dc.html` mockup, vision-compares the PNGs itself, and files the visual
  deltas as `issues.md` entries — **generative**, the same triage pattern as the Codex second opinion.
- **Role — generative, not a flaky autonomous pass/fail.** An LLM's screenshot-similarity verdict is
  non-deterministic, so it produces deviation *issues*, it does not set a per-iteration verdict. The
  loop gate stays **named-region parity** (stable, HTTP-seam). The residual aesthetic judgement is a
  **mandatory M-UI exit gate**: M-UI does not reach DONE until every surface has passed the visual pass
  and a human has signed off (`target.md`).
- **Graceful degradation (hard rule).** If the CLI/Chrome is unavailable (cron run, install missing,
  launch failure), the visual step records "visual check skipped — <reason>" in the handoff and the
  loop continues. It is best-effort; it MUST NOT stall the loop — the same rule as the Codex step.

## Consequences

- **Visual fidelity is verified where the loop runs.** Headless, container/CI/cron-safe; no interactive
  desktop, no extension, no host browser. Works in autonomous and interactive runs alike.
- **One more global npm tool + a heavier image.** Chrome for Testing (~180 MB) + its system libs are
  baked into the devcontainer image. Acceptable for a dev image; the download is cached in the layer,
  not repeated per rebuild.
- **CLI ⇄ skill version-match is a maintenance edge.** Skills "ship with the CLI"; the vendored
  `.claude/skills/agent-browser/SKILL.md` and the `agent-browser@0.29.0` pin must move together.
- **Not `claude-in-chrome`.** That interactive MCP stays available for ad-hoc, human-driven browser
  work in an interactive session, but is never wired into the loop gate (it cannot run headless here).
- **Pixel-diff regression is still out of scope.** A deterministic headless-Chrome + golden-screenshot
  diff in CI remains a future option for stable per-iteration gating once surfaces are near-final; this
  ADR deliberately starts with generative review + a human exit gate (YAGNI).

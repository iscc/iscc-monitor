---
name: build
description: Advance iscc-monitor by exactly one CID-loop iteration (update-state → define-next → advance → review). Use when explicitly asked to "run the build loop", "advance the project / one CID iteration", on /build, or when an automation (/goal, /loop, a scheduled task) drives autonomous building. NOT for one-off edits, questions, or reviewing unrelated work.
---

Run **one CID-loop iteration** — Continuous Iterative Development. You are the **orchestrator**
(replacing iscc-lib's `tools/cid.py`): you do no implementation yourself. You spawn four **roles**,
each as an independent fresh-context subagent, in order, and route their results. The roles are
defined in `.claude/agents/` and read their own context from `.claude/context/`; you only launch them
and read the files they commit.

The point of four separate subagents is **role isolation**: the role that implements (`advance`) must
never be the role that reviews it (`review`). Do not collapse them or do their work inline.

## Steps

1. **Preflight.** Confirm the working directory is the iscc-monitor repo (`.claude/context/target.md`
   exists). Run `git status --short` to see the starting state — untracked design files under
   `.claude/` and `cauldron/` are expected; that is fine. **Ensure the loop is on the `develop`
   branch** — the loop never commits to `main`. If `develop` does not exist, create it
   (`git checkout -b develop`; on a repo with no commits yet this simply starts `develop`). *Done when*
   you have confirmed the repo, are on `develop`, and noted the git state.

2. **update-state.** Spawn the `update-state` agent (Agent tool, `subagent_type: "update-state"`,
   prompt: "Run your role for this CID iteration."). When it returns, read `.claude/context/state.md`.
   - If `state.md` shows `## Status: DONE` → **stop the iteration**, emit `LOOP: DONE`, and report.
     Do not spawn the other roles.
   *Done when* `state.md` reflects this iteration's assessment, or you have emitted `LOOP: DONE`.

3. **define-next.** Spawn the `define-next` agent (`subagent_type: "define-next"`, same prompt). When
   it returns, read `.claude/context/next.md` and confirm it defines exactly one work package with
   runnable Verification. *Done when* `next.md` holds one scoped, verifiable step.

4. **advance.** Spawn the `advance` agent (`subagent_type: "advance"`, same prompt). It implements,
   tests, runs `mise run check`, writes `handoff.md`, and commits. *Done when* the `advance` subagent
   has returned and committed (or written a blocker handoff).

5. **review.** Spawn the `review` agent (`subagent_type: "review"`, same prompt). It independently
   inspects the diff, re-runs the gates, sets the verdict, and pushes on PASS. When it returns, read
   `.claude/context/handoff.md`. *Done when* `handoff.md` carries a `**Verdict:**` and a `**Loop:**`
   line.

6. **Route the loop signal** from `handoff.md`:
   - `**Loop:** DONE` → the target is met. Emit `LOOP: DONE`.
   - `**Loop:** STOP` (a `HUMAN REVIEW REQUESTED` was raised) → **halt**. Emit `LOOP: STOP` and surface
     the exact reason to the human. This is the one stop the autonomous loop keeps; do not try to
     resolve a human-only decision yourself.
   - `**Loop:** CONTINUE` (the normal case, **including a NEEDS_WORK verdict** — the next iteration
     retries) → emit `LOOP: CONTINUE`.
   *Done when* you have emitted exactly one `LOOP:` signal.

7. **Report.** End your output with this block verbatim (so a `/goal` evaluator and the human can read
   the outcome without opening files):

   ```
   CID iteration complete.
   Verdict: <PASS | PASS_WITH_NOTES | NEEDS_WORK>
   Loop:    <CONTINUE | DONE | STOP>
   Step:    <the next.md step title>
   Note:    <one line — what landed, or why it stopped>
   ```

## Rules

- **Spawn, don't do.** Never implement, scope, or assess inline — that destroys role isolation. Launch
  the subagents and read what they commit.
- **One iteration per invocation.** Looping across iterations is the driver's job (`/goal`, `/loop`,
  or a scheduled task) — see `AUTOMATION.md`. Do not loop inside this skill.
- **Run roles in order** and stop early on `LOOP: DONE` (step 2) or `LOOP: STOP` (step 6). Do not
  spawn a later role after a stop signal.
- **Stay out of the agents' files.** They own `.claude/context/*`; you only read them to route.
- If a subagent dies or returns nothing, report it as a failed iteration (`LOOP: STOP`, Note: which
  role died) rather than papering over it.

See `AUTOMATION.md` (next to this file) for driving the loop unattended.

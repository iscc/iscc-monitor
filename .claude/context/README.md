# CID Context Pack (`.claude/context/`)

Shared working memory for the **CID loop** (Continuous Iterative Development) — the autonomous
agent loop that advances iscc-monitor in small, verified increments. One invocation of the
`build` skill runs one iteration: `update-state → define-next → advance → review`.

This pack keeps each role's prompt small and unambiguous. The authoritative *specs* live
elsewhere (`.claude/prd`, `.claude/plans`, `.claude/adr`, the `CLAUDE.md` glossary); these files
hold only the loop's moving state.

## Files

| File           | Owner                         | Update style                         | Purpose                                                      |
| -------------- | ----------------------------- | ------------------------------------ | ----------------------------------------------------------- |
| `target.md`    | human                         | curated                              | Desired end-state + the quality bar + milestones M1→M7      |
| `state.md`     | `update-state`                | overwrite                            | Honest snapshot of what exists at HEAD (records assessed-at)|
| `next.md`      | `define-next`                 | overwrite                            | Exactly one small, verifiable work package                  |
| `handoff.md`   | `advance` then `review`       | overwrite                            | Inter-role report + the review verdict + the loop signal    |
| `learnings.md` | `review`                      | slim index — promote durable rules   | Always-loaded durable rules + a pointer table into `learnings/` |
| `learnings/`   | `review`                      | per-package detail, capped + rotated | Progressively-loaded pitfalls; a role Reads only the file its step touches |
| `issues.md`    | humans + agents               | append-only; `review` deletes resolved | Lightweight backlog `define-next` can prioritize          |

## Hygiene

- **Specs live in `.claude/{prd,plans,adr}`**, not here — link, don't copy.
- **`state.md` is evidence + gaps**, not a re-description of met targets.
- **Issues stay actionable**: "what is wrong + where + how to verify".
- **Learnings is an index + detail dir, not one growing file.** `learnings.md` holds only durable,
  cross-cutting rules + a pointer table; per-package pitfalls live in `learnings/<name>.md` and a role
  Reads only the file its step touches. Promote a finding to the index **only if** it stays true with
  its package deleted *and* is needed even when a step does not touch that package.
- **Rotation, not unbounded append.** Record the forward-looking pitfall, not the verification
  ceremony (that lives in the handoff + commit). Cap each detail file at ~40 bullets / ~150 lines and
  the index at ~120 lines; on overflow, net-reduce — collapse settled notes to a one-line `settled:`
  summary. Git history keeps the rest.

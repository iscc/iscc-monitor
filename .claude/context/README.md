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
| `learnings.md` | `review`                      | append-only (prune for signal)       | High-signal pitfalls, patterns, verified conventions        |
| `issues.md`    | humans + agents               | append-only; `review` deletes resolved | Lightweight backlog `define-next` can prioritize          |

## Hygiene

- **Specs live in `.claude/{prd,plans,adr}`**, not here — link, don't copy.
- **`state.md` is evidence + gaps**, not a re-description of met targets.
- **Issues stay actionable**: "what is wrong + where + how to verify".
- Prune `learnings.md` when it drifts past signal; git history keeps the rest.

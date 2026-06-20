Run the `/build` skill to advance iscc-monitor by exactly one CID-loop iteration. Print the report
block (`Verdict:` / `Loop:`) at the end.

- If the report shows `Loop: DONE`, say so in one line and stop scheduling further iterations.
- If it shows `Loop: STOP`, summarize the `HUMAN REVIEW REQUESTED` reason and hand back — do not start
  another iteration.
- Otherwise (`Loop: CONTINUE`, including a `NEEDS_WORK` verdict), continue: the next iteration retries.

Do not start new work outside the CID loop. See `.claude/skills/build/AUTOMATION.md`.

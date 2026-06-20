# Driving the CID loop unattended

The `build` skill runs **one** iteration. Building the whole project means running it many times. This
is the deliberate split that replaces iscc-lib's `tools/cid.py`:

- **The skill** = one verified increment (`update-state → define-next → advance → review`).
- **The driver** = whatever re-runs the skill until the target is `DONE`.
- **`mise`** = the quality gates only. It does **not** loop — Claude does.

The autonomous policy is **fully autonomous** (your choice): `review` pushes on PASS and the loop
auto-advances across milestone boundaries. It stops only on `Loop: DONE`, on `Loop: STOP` (a genuine
`HUMAN REVIEW REQUESTED` — a design/API/spec decision only you should make), or on a dead subagent.
Each iteration ends by printing a report block whose last lines are `Verdict:` and `Loop:` — the
signal every driver below keys off.

Because the monitor **is** the trust root, an extra gate guards autonomy on the crypto path. Any
change to signature / consistency / proof code must pass the monitor's **conformance tests** before
`review` will PASS and push — these encode the PRD's oracle comparisons (golden vectors from
`derive_vkey.py`; `fsck` root-rebuild over the `SQLiteFetcher`; the inclusion cross-check against the
hub's own `IsccLogInclusionProof`) and run as ordinary `go test` / `mise run check` once the package
exists. The **fully-independent** checks are wired in **CI**, not `go run` from gitignored `cauldron/`
(those are module-less `main.go` files needing external deps — there is no local compile path): build
and shell out to `notecheck` in CI for signature-verdict parity, and run the hub-receipt cross-check
(the hub computed `IsccLogInclusionProof`, not the monitor). Be honest about independence: the ported
`runfsck` leaf hasher / `fsck`-over-`SQLiteFetcher` is a structural **self-check** inside the monitor's
own process (if the port shares a bug it agrees with the bug) — useful, but `notecheck` and the hub
receipt are the true external oracles. This is a *gate* (a fail → `NEEDS_WORK` → retry), not a human
stop, so the loop stays unattended.

**Branch model.** The loop runs on the **`develop`** branch and never commits or pushes `main`
(matching iscc-lib). The skill's preflight ensures you are on `develop` before any role runs;
`review` pushes `develop` on PASS; a human merges `develop → main` via a CI-gated PR. This keeps the
default branch human-gated even while the loop runs unattended.

> **Why the evaluator needs that block.** A `/goal` check reads only what Claude *surfaced in the
> conversation* — it does not run commands or open files. The skill therefore prints `Loop: DONE` /
> `STOP` / `CONTINUE` in plain text so the driver can decide without inspecting the repo.

## Three drivers (pick by how unattended you need it)

| Driver                    | Re-runs when            | Needs session open | Survives restart        | Local files (`cauldron/`) |
| ------------------------- | ----------------------- | ------------------ | ----------------------- | ------------------------- |
| **`/goal`**               | previous turn finishes  | yes                | yes (`--resume`)        | yes                       |
| **`/loop`**               | a time interval elapses | yes                | restored if < 7 days    | yes                       |
| **Routine / Desktop / CI**| a schedule fires        | no                 | yes (durable)           | see "cauldron" caveat     |

*"Survives restart" for `/goal` and `/loop` means they **re-attach when you manually resume** the
session (`--resume` / `--continue`) within the expiry window — **not** that they keep running while the
session is closed. Both stop firing the instant the session/terminal closes ("tasks only fire while
Claude Code is running and idle"). Only the Routine / Desktop / CI row runs truly unattended.*

### A. `/goal` — drive until the milestone target holds (recommended for interactive bring-up)

`/goal` keeps starting a new turn after each one finishes until a condition holds. Make the condition
the build loop, and key it off the printed signal:

```
/goal Run the /build skill to advance exactly one CID iteration, then repeat. Print the report block each time. The goal is MET when the latest /build report shows "Loop: DONE" or "Loop: STOP" — on STOP, summarize the HUMAN REVIEW REQUESTED reason and hand back to me. Otherwise keep going. Stop after 40 turns regardless.
```

- Setting the goal starts the first turn immediately — no separate prompt needed.
- **`/goal` is model-driven, not a deterministic re-invoker.** Each new turn's directive is the
  *condition text*; whether Claude actually calls `/build` again that turn is the model's choice. The
  condition is phrased "Run the /build skill … then repeat" precisely to make that reliable, and the
  skill is model-invocable so it *can* fire — but if you want a **guaranteed** re-invocation on a
  cadence, use `/loop /build` (driver B), which re-issues the command itself.
- The `stop after 40 turns` clause bounds a runaway loop; raise/lower it to taste. It is also the
  loop's only runaway bound — see "What the simplification drops" below.
- Pair with **auto mode** (configured via the auto-mode / permission-mode settings) so each
  iteration's tool calls run without per-tool approval — otherwise the loop blocks on the first edit.
- Headless, one shot to completion: `claude -p "/goal Run /build until the latest report shows Loop: DONE or Loop: STOP; stop after 40 turns."` (Ctrl+C interrupts.)
- Resuming a session with `--resume`/`--continue` restores an unfinished goal (the turn/token counters reset).
- `/goal` requires the workspace trust dialog accepted and hooks enabled (it is a Stop hook under the hood).

### B. `/loop` — re-run on an interval

`/loop` re-runs a prompt on a fixed cadence (session-scoped; auto-expires after 7 days; fires only
while the session is idle; `Esc` stops it):

```
/loop 20m /build
```

Each interval invokes the `/build` skill once. Because `loop.md` (below) exists, a **bare** `/loop`
also runs the build loop at a Claude-chosen cadence:

```
/loop
```

Use `/goal` when you want to stop *the moment the target is met*; use `/loop` when you want a steady
heartbeat (e.g. advance one step every 20 minutes while you watch).

### C. Durable scheduling — runs without an open session

`/goal` and `/loop` die when the session closes. For truly unattended building, use a durable
scheduler that **re-issues the `/build` command directly** on a cron — one deterministic iteration per
fire. Prefer this over scheduling a `/goal` condition: `/goal` re-invocation is the model's choice each
turn (above), so for an *unattended security build* the deterministic `/build`-per-fire path is safer —
it guarantees the four-role cycle runs and cannot be satisfied by a turn that merely *says*
`Loop: CONTINUE`.

- **Desktop scheduled task** (local machine) — has access to the gitignored `cauldron/` reference and
  your local `go`/`mise`; persists across restarts; 1-minute minimum. **Best fit for this repo.**
- **GitHub Actions** (`schedule` trigger) — must `checkout` the sibling `iscc-hub`/`tessera` repos (or
  un-ignore `cauldron/`) so the reference resolves in CI; commits/pushes from the runner.
- **Cloud Routine** — runs on Anthropic infrastructure on a **fresh clone with no local files**, so it
  will **not** see the gitignored `cauldron/` reference. Use it only if you first vendor the reference
  into the repo (drop the `cauldron/` line from `.gitignore`) or have the routine fetch the sibling
  repos in a setup step. 1-hour minimum interval.

A durable nightly job fires the `/build` command itself, e.g. a Desktop scheduled task whose prompt is
the bare command (one iteration per fire; runs the next iteration on the next fire):

```
/build
```

Schedule it at a modest interval (e.g. every 30–60 min during the build window); each fire advances
one verified increment and prints `Verdict:`/`Loop:` for morning triage. When a fire reports
`Loop: DONE` or `Loop: STOP`, disable the task. Keep the interval ≥ the time one iteration takes —
there is no per-subagent timeout (see "What the simplification drops"), so a hung iteration must be
caught by you, not the scheduler.

## Recommended progression

1. **Bootstrap interactively.** Run `/build` once by hand. Watch the four roles. Confirm `mise run
   check` goes green after the module is created and the first golden test lands.
2. **Supervise.** Once an iteration looks healthy, drive it while you watch: `/goal` (stops the moment
   the target holds) or `/loop /build` (deterministic heartbeat). Turn auto mode on so each iteration
   runs without per-tool approval.
3. **Graduate to unattended.** When you trust it, schedule a **Desktop scheduled task firing `/build`**
   (the deterministic command, not a `/goal` condition) so it advances without an open session. Each
   morning, triage any `Loop: STOP` it parked for you.

## What still stops the loop (by design)

- `Loop: DONE` — every v1 milestone (M1 → OTS) meets its `target.md` **Verify** criteria, gates green.
- `Loop: STOP` — a `HUMAN REVIEW REQUESTED`: a design deviation, a backward-incompatible API change, a
  questionable `target.md`, or a risky non-obvious failure. The loop hands these to you instead of
  guessing — the one safeguard kept even in fully-autonomous mode. Remove it only if you accept the
  loop committing its own resolution to such decisions.
- A dead/empty subagent — reported as a failed iteration rather than silently skipped.

A `NEEDS_WORK` verdict does **not** stop the loop: the next iteration retries it.

## What the simplification drops (vs iscc-lib's `tools/cid.py`)

Replacing the Python orchestrator with a skill + a Claude-Code driver removes a few `cid.py`
properties on purpose — named here so each is a choice, not an oversight:

- **Per-role runaway timeout.** `cid.py` killed a hung role at a wall-clock cap (1200–3000 s). The
  skill has **no per-subagent kill switch**; the only runaway bound is the `/goal` `stop after N
  turns` clause (and your `Esc` / `Ctrl-C`). The turn-cap counts driver turns, **not** subagent
  wall-clock — so in the unattended path a single hung subagent stalls the whole run until you notice.
  Each iteration also spawns **four `opus`/`xhigh` subagents in series** (your "4 independent
  subagents" choice) — real cost and latency per iteration. Mitigate: schedule `/build`-per-fire at an
  interval ≥ one iteration's typical time, keep any `/goal` turn-cap small, and check unattended runs
  periodically rather than fully fire-and-forget.
- **The IDLE terminal state.** `cid.py` also stopped on IDLE (only low-priority work left). This loop
  **deliberately omits IDLE** — the signal set stays `CONTINUE`/`DONE`/`STOP`, and the turn-cap bounds
  no-op iterations instead. If near-completion runs waste budget, lower the turn-cap or add a
  "stop when only low-priority issues remain" clause to your `/goal` condition.
- **Structured iteration log, `meta-improve` self-tuning, Codex cross-review, per-agent memory.** Git
  history + the Claude transcript replace the JSONL log; the other three are out of this simplified
  scope. Cross-iteration knowledge lives in `.claude/context/learnings.md`.

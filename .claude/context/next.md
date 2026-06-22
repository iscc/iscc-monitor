# Next Work Package

## Step: Add a public-facing root `README.md`

## Advances
The `target.md` **Done When** gate, which now reads:

> Every v1 milestone … meets its **Verify** criteria with `mise run check` green and no open
> `critical` or `normal` issue in `issues.md`, **and a public-facing root `README.md` exists** — a
> human-facing project overview + build/run instructions + pointers to the specs, distinct from the
> agent-facing `CLAUDE.md` and the CID context pack's `.claude/context/README.md`.

It also closes the `normal` issue **"No public-facing root `README.md` — the project has no human-facing
front door"** (`issues.md`). This is the last code/doc-closable "Done When" requirement and `state.md`
names it as the immediate next work. DONE is unreachable while it is absent, so this is the correct
milestone-advancing step — a doc artifact the target explicitly mandates, not a self-filed polish detour.

## Goal
Create a tracked root `README.md` so a human landing on the repo (GitHub, a fresh clone, the GHCR
image's "source" link) gets an honest project overview + a copy-pasteable build/run snippet + pointers
to the specs — closing the final `target.md` "Done When" gate.

## Scope
- **Create**: `README.md` (repo root) — the only file this step creates.
- **Modify**: none (no Go source; the ≤3 non-test/doc budget is untouched).
- **Reference**:
  - `/workspace/iscc-monitor/CLAUDE.md` — the project-overview prose + the authoritative "Running a
    local dev instance" env table and run snippet to LINK (do **not** copy the full table).
  - `/workspace/iscc-monitor/deploy/OPERATING.md` — the operator/deploy doc to point at for production
    (GHCR image, volumes, reverse proxy); the README is the front door, OPERATING.md is the deep dive.
  - `/workspace/iscc-monitor/mise.toml` — confirm the exact gate command (`mise run check`,
    `tasks.check` = `go build ./... && go vet ./... && go test ./...`) and `mise run fmt`.
  - `.claude/context/learnings/config.md` — the config contract: `ISCC_MONITOR_DB` + `ISCC_MONITOR_REALM`
    are BOTH required (no safe image/Go default for `DB`); intervals default `NORMAL=5m`/`FROZEN=1h`; the
    masthead identity keys are optional display strings. Use this to keep the snippet honest.

## Not In Scope
- Do **not** duplicate the full `ISCC_MONITOR_*` env-var table — link `CLAUDE.md` "Running a local dev
  instance" as the single authoritative source so the two never drift (the issue mandates this).
- Do **not** touch `CLAUDE.md`, `deploy/OPERATING.md`, `mise.toml`, or any Go source — README only.
- Do **not** fix the open `normal`/`low` issues in passing (Compose volume-prep chown, `publish.yml`
  ref-guard, Node-20 action bumps, the proofserve masthead slice, the shared `Resolve` leaf) — each is
  its own later step.
- Do **not** add CI/Markdown-lint tooling, or badges that reference infra not yet public (e.g. a GHCR
  pull badge while the package's visibility is still an iscc-infra step). Plain prose + working links.

## Implementation Notes
Keep it human-facing and evergreen (no "new/improved", no changelog-style wording — CLAUDE.md comment
discipline). Suggested sections, all grounded in the references above:

1. **What it is** — the independent **Trust & Transparency** service for the ISCC-Hub network: follows
   every hub's tlog-tiles transparency log, verifies Ed25519 signed-note signatures + RFC-6962
   consistency, mirrors the logs as SQLite BLOBs, and publishes verifiable evidence. Frame it honestly
   as a **verifiable cache**, *not* a trusted oracle (CLAUDE.md glossary: clients re-verify what the
   monitor serves against the hub's signature + Merkle math). One tight sentence distinguishing **split
   view** (detected by comparison) from a **self-consistency violation** (detected autonomously → freeze)
   is welcome but optional — keep it short.
2. **Stack** — Go 1.26, `CGO_ENABLED=0`, a single static binary `cmd/iscc-monitor`, configured entirely
   through environment variables. Mention the reuse posture briefly (the transparency-dev stack +
   `iscc-lib` Go codec) but link the ADRs rather than re-explaining.
3. **Build & run** — a copy-pasteable fenced `sh` block that actually starts the binary against the
   **testnet realm**. Mirror CLAUDE.md's known-good snippet; the canonical form is:
   ```sh
   go build -o /tmp/iscc-monitor ./cmd/iscc-monitor
   ISCC_MONITOR_DB=/tmp/monitor-dev.db \
   ISCC_MONITOR_REALM=internal/registry/testdata/realm.txt \
   ISCC_MONITOR_NORMAL=30s \
   ISCC_MONITOR_ADDR=0.0.0.0:41464 \
   /tmp/iscc-monitor
   ```
   After the first poll the HTTP surface is live (`GET /` dashboard, `/healthz`, `/metrics`). Note the
   two **required** vars (`ISCC_MONITOR_DB`, `ISCC_MONITOR_REALM`); do not imply a default DB path —
   there is none (`config.md`). For the full var list, link CLAUDE.md "Running a local dev instance".
4. **Quality gate** — `mise run check` (build + vet + test) must stay green; formatting via
   `mise run fmt` / `gofmt -l .`. Use the exact task names from `mise.toml`.
5. **Deployment** — one line: a container image is published to GHCR; point to `deploy/OPERATING.md`
   for the operator contract (volume path, backups, non-root uid, reverse-proxy, `/metrics` exposure).
   Do not restate that doc's substance.
6. **Specs / pointers** — link `.claude/prd/` and `.claude/adr/` (ADR-0001..0013) and the glossary in
   `CLAUDE.md`. State plainly that `CLAUDE.md` is **agent-facing** project instructions and
   `.claude/context/README.md` is the **CID-loop-internal** context pack — so THIS README is the human
   front door (the "distinct from" the target requires).

Relative links in the README must resolve from the repo root (e.g. `./CLAUDE.md`,
`./deploy/OPERATING.md`, `.claude/adr/`). Every linked path is confirmed present: `CLAUDE.md`,
`deploy/OPERATING.md`, `.claude/prd/0001-iscc-monitor-v1.md`, `.claude/adr/0001…0013`,
`internal/registry/testdata/realm.txt`.

## Verification
- `test -f /workspace/iscc-monitor/README.md` exits 0 (the file exists at the repo root).
- `mise run check` is green (a doc-only change must not regress the gate — `go build`/`vet`/`test` pass;
  no `.go` file changed).
- `gofmt -l /workspace/iscc-monitor` is empty (no Go file touched).
- `grep -q 'mise run check' /workspace/iscc-monitor/README.md` — the README names the quality gate.
- `grep -q 'cmd/iscc-monitor' /workspace/iscc-monitor/README.md` — the README shows the build/run path.
- `grep -qi 'verifiable cache' /workspace/iscc-monitor/README.md` — the honest framing is present.
- No dead relative links: for each path the README references
  (`CLAUDE.md`, `deploy/OPERATING.md`, `.claude/prd`, `.claude/adr`), `test -e <path>` exits 0.

## Done When
`README.md` exists at the repo root with a project overview, a runnable build/run snippet, the
`mise run check` gate, and working links to `CLAUDE.md` + the spec dirs; all Verification checks pass and
`mise run check` stays green.

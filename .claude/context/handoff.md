## 2026-06-22 — Review of: Add a public-facing root `README.md`

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** The advance creates a single tracked root `README.md` — the human-facing front door — with
an honest verifiable-cache overview, the Go 1.26 / `CGO_ENABLED=0` stack, a copy-pasteable testnet
build/run snippet, the `mise run check` gate, a GHCR/deploy pointer, and spec links. It is scope-clean
(one created doc, zero Go source, no edits to `CLAUDE.md`/`OPERATING.md`/`mise.toml`), every factual
claim verifies against the repo, and the full env-var table is LINKED (not duplicated) to `CLAUDE.md`.
This closes the last code/doc-closable `target.md` "Done When" gate and its `normal` issue.

**Verification:**
- [x] `test -f README.md` — PASS (created at repo root, new file mode 100644).
- [x] `mise run check` — PASS (28 packages `ok`, cached; no `.go` file changed, gate re-confirmed green).
- [x] `gofmt -l .` — PASS (empty; no Go file touched).
- [x] `grep -q 'mise run check'` — PASS (names the gate; the quoted task body `go build ./... && go vet ./... && go test ./...` is byte-exact vs `mise.toml [tasks.check]`).
- [x] `grep -q 'cmd/iscc-monitor'` — PASS (build/run path shown).
- [x] `grep -qi 'verifiable cache'` — PASS (honest framing present, distinguishes self-consistency violation from split view).
- [x] No dead relative links — PASS: `CLAUDE.md`, `deploy/OPERATING.md`, `.claude/prd`, `.claude/adr`, `.claude/context/README.md`, `internal/registry/testdata/realm.txt`, and the two referenced ADRs (0011, 0003) all `test -e` 0. The `CLAUDE.md#running-a-local-dev-instance` anchor matches the live `## Running a local dev instance` heading slug.
- [x] Factual-claim audit (reviewer-added) — PASS: `modernc.org/sqlite v1.46.1` and `iscc-lib/packages/go v0.5.0` in `go.mod`; ADR range 0001–0013 present; `mise run fmt` = `gofmt -w .`; the verify-for-me (`/verify`) + inclusion/certificate routes the README cites exist.
- [x] Env-var table NOT duplicated — PASS: only the snippet vars + the "required" prose note appear; no `INSTANCE`/`OPERATOR`/`REALM_NAME` enumeration (linked to CLAUDE.md, per Not-In-Scope).
- [x] Gate-circumvention scan over unpushed commits — PASS: no `//nolint`/`t.Skip`/swallowed-error/build-tag in the diff (doc + context-pack only).

**Issues found:** (none). Deleted the resolved `normal` issue "No public-facing root `README.md`".

**Codex second opinion:** Clean verdict — "adds a root README and updates the handoff context without
changing runtime code; the README content is consistent with the existing configuration, endpoints,
deployment docs, and requested scope; no blocking or actionable correctness issues introduced." No
findings to triage; agrees with my independent review.

**Visual check:** n/a — no SSR surface changed (doc-only; no `internal/dashboard`/`dossier`/`web`/
`certificate` template touched).

**Next:** The last code/doc-closable "Done When" gate is now closed, so DONE turns on the remaining open
issues. DONE is NOT yet reachable: three `critical` iscc-infra issues remain (persistence-contract docs,
public-route/`/metrics`-exposure decision, egress + resource-footprint sizing) plus open `normal`s
(realm-index `/` sub-region deltas, WASM signature-half trust gap, per-hub Anchor design question,
`publish.yml`/`OPERATING.md` Compose-volume fixes). The three criticals are framed as documentation this
repo owns (mostly `deploy/OPERATING.md` additions), so `define-next` should target one of them next — the
persistence-contract doc is the most self-contained and directly unblocks the testnet deploy.

**Notes:**
- Scope discipline is exemplary: one created doc, zero behavior change, every Not-In-Scope item honored
  (no env-table copy, no `CLAUDE.md`/`OPERATING.md`/`mise.toml` edit, no badges, no CI/lint tooling).
- Did not add a `learnings/` detail file: a one-time static README carries no recurring forward-looking
  pitfall, and the docs-layout convention already lives in user memory — a thin detail file would be
  verification ceremony, which the rotation rule excludes from cross-iteration memory.
- The three open `critical`s are the gating work for DONE; if `define-next` judges any of them truly
  external to this repo (no doc/code closeable here), surface that as a STOP/IDLE edge rather than
  spinning on cosmetic chrome — but on their current text they are repo-doc-closable, so CONTINUE.

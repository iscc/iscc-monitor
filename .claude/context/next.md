# Next Work Package

## Step: Ship the canonical `deploy/realm-testnet.txt` and bake it into the image

## Advances
M-Deploy Verify criterion (target.md, the M-Deploy block):

> a **canonical realm document** lives at a fixed non-testdata path (e.g. under `deploy/`), is baked
> into the image at a documented path, and `registry.Parse` accepts it (test); and CLAUDE.md's env
> table lists the masthead identity keys …

Closes the realm-doc half of the `critical` issue **"Provide a canonical, mountable testnet realm
file (not testdata) + the instance identity env values"** (the identity-key half is already done —
CLAUDE.md's env table lists `ISCC_MONITOR_INSTANCE` / `OPERATOR` / `REALM_NAME`). This is review's
explicit `**Next:**` (the smallest remaining code-closable M-Deploy slice) and the Dockerfile already
forward-references this exact path (`Dockerfile:44` calls `deploy/realm-testnet.txt` "a later M-Deploy
slice").

## Goal
Replace the testdata realm path baked into the production image with a canonical, mountable
`deploy/realm-testnet.txt`, so the deploy contract no longer reaches into Go testdata. The image bakes
this file at the documented `/etc/iscc-monitor/realm.txt`, `registry.Parse` accepts it under test, and
CLAUDE.md documents the canonical path.

## Scope
- **Create**: `deploy/realm-testnet.txt` — the canonical testnet realm document. Same content as
  `internal/registry/testdata/realm.txt` (the two real testnet hubs `sb0.iscc.id` + `sb1.amlet.id`,
  domains-only, ADR-0009), with a header comment that it is the canonical mountable/bakeable realm doc.
- **Create (test, not counted)**: `internal/registry/deploy_test.go` — a golden test that reads
  `../../deploy/realm-testnet.txt` and asserts `Parse` returns exactly the two expected `Entry` values.
- **Modify**: `Dockerfile` — repoint the bake `COPY` line (currently
  `COPY internal/registry/testdata/realm.txt /etc/iscc-monitor/realm.txt`, `Dockerfile:45`) to
  `COPY deploy/realm-testnet.txt /etc/iscc-monitor/realm.txt`, and update the adjacent comment
  (lines 43-44) so it no longer calls the canonical file "a later M-Deploy slice".
- **Modify (doc)**: `CLAUDE.md` — in the env table (lines 42-43), document `deploy/realm-testnet.txt` as the
  canonical mountable/baked realm doc and that the image bakes it at `/etc/iscc-monitor/realm.txt`; keep the
  existing testdata note framed as the in-tree test fixture.
- **Reference**:
  - `.claude/context/learnings/registry.md` — `Parse` is the pure domains-only leaf; fails closed on
    URL-shaped lines; preserves order, no dedupe; `BaseURL = "https://" + Domain`.
  - `internal/registry/registry_test.go` — `TestParseGolden` is the pattern to mirror (read fixture →
    `Parse` → `reflect.DeepEqual` against the two-`Entry` `want`).
  - `internal/registry/testdata/realm.txt` — the exact current content to copy.
  - `cmd/iscc-monitor/main_test.go:36` — the established `filepath.Join("..", "..", …)` repo-root-relative
    read idiom for a test that reads a file outside its package dir.
  - `Dockerfile` lines 43-45 — the bake `COPY` and its forward-reference comment.

## Not In Scope
- Do NOT touch `internal/registry/testdata/realm.txt` — it stays the in-package test fixture; the canonical
  `deploy/` file is a separate artifact (the two can have identical content without one importing the other).
- Do NOT write the operability/deployment doc (volume path, backup unit, egress, `/metrics` exposure,
  migration policy) — that is the NEXT M-Deploy slice and folds in the other three `critical` infra asks.
- Do NOT write the root `README.md` — separate "Done When" slice.
- Do NOT `go:embed` the realm into the binary or change `cmd/iscc-monitor` wiring — the realm is supplied
  via the `ISCC_MONITOR_REALM` path env (mount or bake), not embedded.
- Do NOT change `registry.Parse` itself, the `Entry` shape, or any follower/store wiring.
- Do NOT add the `workflow_dispatch` ref-guard fold-in (no workflow file is touched this step).

## Implementation Notes
- The realm content is **domains-only** (ADR-0009): one hub domain per line, `#` comments and blank lines
  dropped, no scheme, no path. Copy the two real testnet hubs verbatim from
  `internal/registry/testdata/realm.txt`: `sb0.iscc.id` and `sb1.amlet.id`. A URL-shaped line (`https://…`
  or a `/path`) makes `Parse` fail closed naming the bad line — keep the file plain domains.
- The new `deploy_test.go` mirrors `TestParseGolden`: `os.ReadFile(filepath.Join("..", "..", "deploy",
  "realm-testnet.txt"))`, then `Parse`, then `reflect.DeepEqual` against
  `[]Entry{{Domain: "sb0.iscc.id", BaseURL: "https://sb0.iscc.id"}, {Domain: "sb1.amlet.id", BaseURL:
  "https://sb1.amlet.id"}}`. Name it `TestParseCanonicalDeployRealm`. Note `Parse`'s contract:
  order preserved, no dedupe — assert the exact ordered two-entry slice, not a set.
- This test is **non-vacuous** because it reads the actual `deploy/` file, not a string literal — if the
  canonical file is deleted, malformed, URL-shaped, or drifts from the two intended hubs, the test fails.
  (Reverting the `deploy/` file content to a URL-shaped line makes `Parse` error → test FAIL; dropping a
  hub makes the `DeepEqual` FAIL.)
- Dockerfile: this is a `COPY` source-path change plus a comment edit; the `golang:1.26.4` build stage and
  distroless final stage are unchanged. Keep the documented baked path `/etc/iscc-monitor/realm.txt`
  identical so the CI `docker` `/healthz` smoke (which baked-realm-starts the container) keeps passing.
- Docker is CI-only on this host (per the last review) — the Dockerfile `COPY` change cannot be locally
  `docker build`-verified, but is mechanically a one-line source-path swap to an existing tracked file; the
  CI `docker` job is the real oracle and it bakes + boots + polls `/healthz`.
- Relevant Correctness rule (learnings.md): registry is the domains-only ADR-0009 leaf — the realm registry
  advertises domains only, no keys; keep the canonical file domains-only.

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...`, `gofmt -l .` empty).
- `go test -run TestParseCanonicalDeployRealm ./internal/registry` passes (reads the real
  `deploy/realm-testnet.txt`, asserts the two ordered entries).
- `deploy/realm-testnet.txt` exists at the repo root under `deploy/`, is domains-only, and
  `registry.Parse` accepts it with exactly two entries (`sb0.iscc.id`, `sb1.amlet.id`).
- `grep -q 'COPY deploy/realm-testnet.txt /etc/iscc-monitor/realm.txt' Dockerfile` succeeds, and the
  Dockerfile no longer references `internal/registry/testdata/realm.txt` on the baked-realm `COPY` line
  (`grep 'COPY .*testdata/realm.txt' Dockerfile` returns no match).
- CLAUDE.md's env table names `deploy/realm-testnet.txt` as the canonical realm doc baked at
  `/etc/iscc-monitor/realm.txt`.

## Done When
`deploy/realm-testnet.txt` exists, the image bakes it at `/etc/iscc-monitor/realm.txt`, CLAUDE.md
documents the canonical path, and `mise run check` + `go test -run TestParseCanonicalDeployRealm
./internal/registry` both pass.

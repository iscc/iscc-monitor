# Handoff

## 2026-06-20 — Review of: Pure realm-registry parser (domains-only membership document)

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** `advance` added `internal/registry` — a pure, dependency-free `Parse([]byte) ([]Entry,
error)` that turns a line-based domains-only realm document into ordered `Entry{Domain, BaseURL}`
values (`BaseURL = "https://"+Domain`), dropping blanks/`#`-comments, trimming, preserving input
order, and failing closed (wrapped, line-naming error) on any URL-shaped line. Scope is exactly the
three scoped files (1 production + 1 test + 1 fixture) in a new leaf package; nothing else touched.
Independently re-verified: gates green, imports clean, go.mod/go.sum byte-identical, no gate-dodging.

**Verification:**
- [x] `mise run check` green — `go build`/`go vet`/`go test ./...` all `ok` (registry, didweb,
  follower, logclient, store).
- [x] `gofmt -l .` (whole tree) and `gofmt -l internal/registry` — both print nothing.
- [x] `go test -count=1 ./internal/registry` — PASS (0.005s, uncached).
- [x] `go test -count=1 -run TestParse ./internal/registry` — PASS. `TestParseGolden` asserts exactly
  `[{sb0.iscc.id, https://sb0.iscc.id}, {sb1.amlet.id, https://sb1.amlet.id}]` in input order (comment
  + blank dropped); table cases cover single/whitespace/`host:port`/no-trailing-newline/all-comment/empty.
- [x] URL-shaped lines (`https://sb0.iscc.id`, `sb0.iscc.id/log`, scheme-on-second-line) → non-nil
  error naming the bad line, with `nil` entries; all-comment / empty document → `(nil, nil)`.
- [x] `git status --short go.mod go.sum` — empty (no dependency added).
- [x] `go list -deps ./internal/registry | grep -E '^(net|net/http)$'` — empty; own `.Imports` =
  `[bufio bytes fmt strings]` (leaf, no `net`/`net/http`/`os`).
- [x] Oracle/conformance gate correctly N/A — diff touches no proof/verify/didweb/merkle/consistency/
  signature/fsck/notecheck path (verified by name-only grep); pure string parser, no golden vector or
  `fsck` behavior changes.
- [x] Gate-integrity scan over all unpushed commits — no `//nolint`/`t.Skip`/build-tag/swallowed-error/
  deleted-assertion. (grep hits were all genuine `t.Fatalf`/`t.Errorf` assertions or removed handoff prose.)

**Issues found:** (none)

**Next:** The `cmd/iscc-monitor` binary + `internal/config` is the now-unblocked consumer slice: read
the realm document from disk → `registry.Parse` → `store.UpsertHub(domain, baseURL, …)` per `Entry`
to obtain `HubID`s → build `[]follower.HubTarget{HubID, BaseURL}` → wire DI + `Loop.Run`. The
registry→target mapping happens at wiring time (the registry intentionally does not know `HubID`).
Equally valid: the merkle-backed **equivocation** trigger (RFC-6962 consistency-proof failure across
growing sizes) — but that is the first slice to trip the oracle gate (needs `transparency-dev/merkle`
+ tile fixtures + `fsck`/`notecheck` parity), so define-next should weigh it as a heavier step.

**Notes:**
- Correctness cross-check: the golden `sb0.iscc.id`/`sb1.amlet.id` are the real testnet hubs used
  across the existing `didweb`/`logclient`/`follower` fixtures and `derive_vkey.py` HUBS — not invented;
  the new fixture is consistent with the rest of the tree.
- Scope discipline honored: nothing in `## Not In Scope` was done — no network fetch, no `cmd/`/config,
  no `follower`/`HubTarget` rewire, no `origin()` duplication, no YAML/JSON dep. Reconciliation
  (add/remove/inactive/sort/dedupe) is correctly deferred to the store-coupled wiring step.
- `host:port` is accepted (no scheme, no slash) per design — exercised by the `localhost:8443` table
  case; live hubs have no port, and the colon is handled downstream by `didweb` (percent-encoded).
- M1 remains in progress (state.md): equivocation trigger, `cmd/` binary, config, coverage
  (`monitored_since`), structured logs, `/metrics`, and real alert transport still missing — so Loop is
  CONTINUE, not DONE. No open critical/normal issue; no human-only decision pending.
- Branch is `develop` (never `main`); pushing on PASS.

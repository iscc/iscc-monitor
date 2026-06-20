# Handoff

## 2026-06-20 — Pure realm-registry parser (domains-only membership document)

**Done:** Added `internal/registry` — a pure, dependency-free `Parse([]byte) ([]Entry, error)` that
turns a line-based domains-only realm-membership document into ordered `Entry{Domain, BaseURL}`
values (`BaseURL = "https://" + Domain`). It ignores blank lines and `#` comments, trims surrounding
whitespace, preserves input order, and fails closed on a URL-shaped line (scheme `://` or path `/`)
with a wrapped error naming the offending line. No I/O, no new dependency.

**Files changed:**
- `internal/registry/registry.go`: package + `Entry` type + `Parse`; imports exactly `{bufio, bytes,
  fmt, strings}` (stdlib leaf, no `net`/`net/http`/`os`).
- `internal/registry/registry_test.go`: golden fixture test + table-driven parse cases (single,
  comments/blanks, whitespace, `host:port`, no-trailing-newline, all-comment, empty) + error cases
  (scheme, path, scheme-on-second-line) asserting the bad line is named and entries are nil on error.
- `internal/registry/testdata/realm.txt`: golden fixture — header comment, `sb0.iscc.id`, blank line,
  `sb1.amlet.id`.

**Verification:** `mise run check` → green (`go build`/`go vet`/`go test ./...` all `ok`; registry
ran uncached `0.006s`). Per-criterion:
- [x] `gofmt -l internal/registry` — empty.
- [x] `go test -count=1 ./internal/registry` — PASS.
- [x] `go test -count=1 -run TestParse ./internal/registry` — PASS; `TestParseGolden` asserts exactly
  `[{sb0.iscc.id, https://sb0.iscc.id}, {sb1.amlet.id, https://sb1.amlet.id}]` in input order
  (comment + blank dropped).
- [x] URL-shaped lines (`https://sb0.iscc.id`, `sb0.iscc.id/log`) → non-nil error naming the bad
  line; all-comment / empty document → `(nil slice, nil err)`.
- [x] `git status --short go.mod go.sum` — empty (no dependency added; `go.mod`/`go.sum`
  byte-identical).
- [x] `go list -deps ./internal/registry | grep -E '^(net|net/http)$'` — empty; own `.Imports` =
  `[bufio bytes fmt strings]` (leaf, no net stack).

**Next:** The `cmd/iscc-monitor` binary + `internal/config` is the now-unblocked slice that consumes
this: read the realm document from disk → `registry.Parse` → `store.UpsertHub(domain, baseURL, …)`
per `Entry` to get `HubID`s → build `[]follower.HubTarget{HubID, BaseURL}` → wire DI + `Loop.Run`.
Note `UpsertHub` returns the `HubID` the loop needs, so the registry→target mapping happens at wiring
time (the registry intentionally does not know about `HubID`). Alternatively the merkle-backed
**equivocation** trigger or the `hub_keys` did:web cache write (both trip the oracle gate, own steps).

**Notes:**
- Conformance/oracle gate is correctly N/A: this is a pure string parser — `git diff --name-only`
  touches no proof/verify/didweb/merkle/consistency/signature path, and no golden vector or `fsck`
  behavior changes.
- Scope honored: 1 production file + 1 test file + 1 fixture, all in the new leaf package; nothing in
  `follower`/`logclient`/`store` touched, no `cmd/`/`config`/fetch/`origin()` started (all explicitly
  Not In Scope). The parser produces `BaseURL` only; the follower already derives origin + verifier
  key from it inside `PollHub`, so no duplication of `logclient.origin`.
- Reconciliation (add/remove/inactive, sort, dedupe) is deliberately omitted — it is store-coupled and
  belongs with the wiring step (Not In Scope here). `Parse` returns entries in raw input order with no
  dedupe.
- `host:port` is accepted (no scheme, no slash) per Implementation Notes — live hubs have no port, so
  this is covered only by the `localhost:8443` table case; the colon is handled downstream by
  `didweb` (percent-encoded before `DocumentURL`).
- Branch is `develop` (never `main`).

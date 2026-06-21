## 2026-06-21 — Hub-List parser + `(realm, hub_id) → domain` resolver in `internal/registry`

**Done:** Added a pure, golden-tested iscc-hub Hub-List parser (`ParseHubList`) plus `Hub`/`HubList`
types and a `Resolve(hubID uint16) (domain, ok)` lookup to `internal/registry` — purely additive,
leaving the existing domains-only `Parse`/`Entry` path and `realm.txt` wiring untouched. This lands the
next link in the realm-wide certificate chain: a decoded `(realm, hub_id)` from `internal/index` now
maps to the issuing hub's domain.

**Files changed:**
- `internal/registry/registry.go`: added imports `net/url` + `gopkg.in/yaml.v3`; `Hub{HubID uint16;
  URL string; Active bool}`, `HubList{Version int; Network string; Hubs []Hub}` with `yaml:"…"` tags
  (no pubkey field — ADR-0009); `ParseHubList([]byte) (*HubList, error)` (fail-closed on invalid YAML /
  hub_id > 4095 / duplicate hub_id / empty-or-path-bearing url; empty `hubs:` is valid); `Resolve`
  method (linear scan; inactive resolves, unknown misses); unexported `hubDomain` helper.
- `internal/registry/testdata/testnet.yaml`: golden realm-0 fixture (`hub_id 0 → https://sb0.iscc.id`,
  `hub_id 1 → https://sb1.amlet.id`, both `active: true`; a `pubkey:` line on hub 0 to prove it is
  parsed-and-ignored). These are the real testnet hubs (match `realm.txt` / `derive_vkey.py`).
- `internal/registry/hublist_test.go`: table-driven golden + failure tests; expected domains hard-coded
  ground truth, not re-derived from the parser.
- `go.mod` / `go.sum`: promoted `gopkg.in/yaml.v3 v3.0.1` to a direct `require` via `go mod tidy`.

**Verification:** `mise run check` → green (build + vet + all 20 packages PASS; `gofmt -l .` empty).
Per-criterion:
- [x] `go test -count=1 ./internal/registry` passes uncached (existing 3 `Parse` tests + 7 new
  Hub-List golden/failure tests).
- [x] `ParseHubList(testnet.yaml)`; `Resolve(0) == ("sb0.iscc.id", true)` and
  `Resolve(1) == ("sb1.amlet.id", true)` (hard-coded).
- [x] `Resolve(9)` (unknown slot) → `("", false)`.
- [x] Inactive hub resolves (`("paused.example", true)`); `hub_id 4095` resolves (full 12-bit range).
- [x] Malformed docs fail closed with non-nil error + nil list: invalid YAML, `hub_id 4096`,
  duplicate `hub_id`, empty url, path-bearing url.
- [x] `pubkey` parsed-and-ignored (Hub struct is exactly `{HubID, URL, Active}`).
- [x] `GOOS=js GOARCH=wasm go build ./internal/registry` succeeds; dep closure has `net/url` +
  `net/netip` (both pure, WASM-safe) but **no** `net`/`net/http`/`database/sql`/`os` directly.
- [x] `go mod tidy` consistent; second run is a no-op; `yaml.v3` is a direct require.

**Next:** The `/inclusion/{iscc_id}` realm-wide HTML certificate page + the downloadable proof-bundle
assembler — the chain is now complete: `index.Decode(iscc_id) → (realm, hub_id)` →
`registry.HubList.Resolve(hub_id) → domain` → `store.SeqsForISCCID` over `/<domain>/log/…`. That slice
**re-engages the oracle/conformance gate** (it touches proof-bundle assembly). It also needs the
realm→network selection (realm 0 → `testnet.yaml`) and a way to load the Hub-List bytes — which is the
live-wiring step deliberately deferred here (see Notes).

**Notes:**
- **Migration deliberately not done (in scope per next.md "Not In Scope").** This step is additive and
  reversible: `cmd/iscc-monitor/main.go`, `internal/config`, the dashboard, and `realm.txt` /
  `Parse` / `Entry` are all untouched. Swapping the live registry source from domains-only `realm.txt`
  to the Hub-List is a separate, backward-incompatible step that `state.md` + prior `review` flag as
  possibly warranting a STOP for human sign-off — that decision is **not forced here**.
- **URL→domain via `net/url.Parse(...).Host`** (chosen over `strings.TrimPrefix`): handles any scheme,
  strips paths, and fails closed on a host-less url (bare path → `Host == ""` → "no host" error). It is
  stdlib + WASM-safe (pulls `net/url`/`net/netip`, not `net`/`net/http`). Documented on `hubDomain`.
- `Resolve` returns the **domain** (host), not the url, because the certificate page needs it for the
  `/<domain>/log/…` mount and it stays consistent with `Entry.Domain`.
- **Empty `hubs:` is valid** (mirrors `Parse`'s all-comment→empty behavior); a zero-hub list parses and
  every slot then misses. Tested.
- `go.sum` gained test-only transitive checksums of yaml.v3 (`kr/pretty`, `kr/text`,
  `rogpeppe/go-internal`, `gopkg.in/check.v1`) — required for `go mod tidy` consistency, not built into
  the binary.
- **Oracle/conformance gate correctly N/A here:** pure leaf, no signature-verify / RFC-6962 / Merkle /
  proof code touched. `go.mod`/`go.sum` changed only to add the pure-Go `yaml.v3` direct require. The
  gate re-engages at the proof-bundle assembler (next step).
- **Pre-existing unrelated working-tree edits** (not mine, not committed by me): `.claude/context/
  target.md`, `.claude/prd/0001-iscc-monitor-v1.md`, `CLAUDE.md`, and a new untracked
  `.claude/adr/0011-iscc-lib-codec-dependency.md`. These were present in the tree from another role/
  session; I committed only my implementation + tests + this handoff per protocol. `review` may want to
  reconcile them — ADR-0011 in particular suggests a planned iscc-lib codec dependency that could affect
  the `internal/index` decoder strategy.

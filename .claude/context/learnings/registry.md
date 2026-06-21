<!-- area: internal/registry -->
<!-- indexed-as: registry.md · owner: review · rotate at ~40 bullets / ~150 lines -->

# `internal/registry` — realm membership leaf

Read this when a step touches the area above. Durable cross-cutting rules live in
the index (`.claude/context/learnings.md`); the package-local mechanics are here.

## Realm registry (`internal/registry`)

- **`Parse([]byte) ([]Entry, error)` is the pure domains-only membership leaf (ADR-0009).** Line-based,
  drops blank/`#`-comment lines, trims, preserves input order (no sort/dedupe — reconciliation is the
  store-coupled wiring step's job). Fails closed on URL-shaped lines: `://` (scheme) or `/` (path) →
  wrapped error naming the bad line; returns `nil` entries alongside the error. Imports are exactly
  `{bufio bytes fmt strings}` — verified no `net`/`net/http`/`os` in the closure (oracle gate correctly
  N/A: no proof/verify/didweb/merkle path touched, go.mod/go.sum byte-identical). `Entry.BaseURL =
  "https://"+Domain`; the follower derives origin+vkey from BaseURL inside `PollHub`, so the registry
  intentionally carries no key field and never reaches for the package-private `logclient.origin`.
- **The golden fixture's `sb0.iscc.id`/`sb1.amlet.id` are the real testnet hubs**, consistent with the
  existing `didweb`/`logclient`/`follower` fixtures and `derive_vkey.py`'s HUBS — not invented. The
  registry→`HubTarget` mapping is deferred to wiring (needs `HubID` from `store.UpsertHub`), so
  `Loop`/`HubTarget`/`cmd/` stay untouched here, as scoped.

## Hub-List parser (`ParseHubList` / `HubList.Resolve`, ADR-0010)

- **Pure additive leaf beside `Parse`.** `ParseHubList([]byte) (*HubList, error)` parses the iscc-hub
  `{version, network, hubs:[{hub_id, url, active, pubkey?}]}` YAML; `Resolve(hubID uint16) (domain, ok)`
  maps the embedded 12-bit slot (0-4095, NOT the `store.UpsertHub` surrogate PK) to the issuing hub's
  domain. `pubkey` is parsed-and-ignored (no field — keys come from did:web, ADR-0009). Inactive hubs
  still resolve (`ok == true`; the badge conveys inactivity); `ok == false` is reserved for an unknown
  slot. Stays WASM-shareable: `yaml.v3` + `net/url` are pure (`net/url` pulls `net/netip`, NOT
  `net`/`net/http`); `os` appears only transitively via `fmt` (the documented purity nuance) — prove
  with `GOOS=js GOARCH=wasm go build`, not by grepping `os`.
- **`hubDomain` is fail-OPEN on a scheme'd path-bearing url — known gap, see issues.md.** It only checks
  `u.Host == ""`, so `https://host/path` parses and `Resolve` returns `host`, dropping the path —
  despite next.md/docstring/handoff all claiming path-bearing is rejected. The scheme-LESS test case
  (`host/path`) is rejected via the *no-host* branch, masking the real gap. A genuine path-bearing
  reject needs a `u.Path != ""` check. Treat any url-host-extraction helper as fail-open until it
  rejects path/query/fragment AND a scheme'd-path test proves it.
- **YAML zero-value fail-open: a missing `hub_id` decodes to slot 0** (a plain `uint16` cannot tell
  absent from `0`). Same trap as any required scalar YAML field — presence-track (`*uint16` /
  `yaml.Node` / custom `UnmarshalYAML`) if absence must fail closed. See issues.md.
- **`KnownFields(false)`** is deliberate: unknown keys (notably `pubkey`, future fields) are tolerated,
  not rejected — the parse-and-ignore contract. Do not flip to `true` without re-deciding `pubkey`.

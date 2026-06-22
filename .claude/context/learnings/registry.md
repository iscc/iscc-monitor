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
- **settled (landed): three fail-opens hardened.** `hubDomain` now rejects
  `u.Path != "" || u.RawQuery != "" || u.ForceQuery || u.Fragment != ""` after the host check (a
  scheme'd `https://host/log` is rejected not host-stripped; a bare trailing `?`/`https://host?` —
  `ForceQuery==true` with `RawQuery==""` — is rejected, not round-tripped into the domain);
  `Hub.HubID` is now `*uint16` so an absent `hub_id` is rejected ("hub_id is required") instead of
  decoding to slot 0. All mutation-proven non-vacuous; trailing-slash `https://host/` rejected (path
  `/`). `Resolve(uint16)(string,bool)` signature unchanged; the public `Hub` shape changed
  (`uint16`→`*uint16`, in-package reader only).
- **`url.Parse` fail-open trap is wider than the obvious three fields.** Beyond `Path`/`RawQuery`/
  `Fragment`, `ForceQuery` is load-bearing: `u.RawQuery != ""` does NOT catch `https://host?` (now
  guarded). The empty fragment `https://host#` Go drops on round-trip (harmless), and `Opaque` is
  unreachable for an `https://`-scheme'd host, so neither needed a guard. General rule: enumerate
  `url.URL`'s shape-carrying fields (`Path RawQuery ForceQuery Fragment Opaque User`), not just the
  obvious three, before claiming "host only".
- **YAML required-scalar fail-open (general): a missing `hub_id` decodes to the zero value** (a plain
  `uint16` cannot tell absent from `0`). Presence-track required scalar YAML fields with `*T` /
  `yaml.Node` / custom `UnmarshalYAML`. (Landed: `HubID *uint16`.)
- **`KnownFields(false)`** is deliberate: unknown keys (notably `pubkey`, future fields) are tolerated,
  not rejected — the parse-and-ignore contract. Do not flip to `true` without re-deciding `pubkey`.

# Next Work Package

## Step: Hub-List parser + `(realm, hub_id) → domain` resolver in `internal/registry`

## Advances
M-UI (Evidence Ledger frontend) Verify criterion — the **realm-wide certificate**:

> "the **realm-wide certificate** (`/inclusion/{iscc_id}`, keyed by the self-describing
> ISCC-IDv1 — decode realm + 12-bit `hub_id`, resolve the issuing hub via the registry) …"

The certificate trust-root decoder (`internal/index.Decode`) is complete and PASS; it yields
`{Realm, HubID 0-4095, Timestamp}`. This step lands the **next link in that chain**: the
`(realm, hub_id) → hub domain` resolver ADR-0010 names ("Hub-id resolution adopts the iscc-hub
Hub-List"). It is the intermediate slice the handoff `**Next:**` and `state.md` both name as the
prerequisite before the `/inclusion/{iscc_id}` HTML page + proof-bundle assembler can be built.

## Goal
Add a pure, golden-tested Hub-List parser to `internal/registry` that reads the iscc-hub
`hubs/<network>.yaml` schema (`{version, network, hubs:[{hub_id, url, active, pubkey?}]}`) and a
`Resolve(hubID) → (domain, ok)` lookup, so a decoded `(realm, hub_id)` from `internal/index` can be
mapped to the issuing hub's domain. This unblocks the certificate page without yet touching the live
realm-file wiring.

## Scope
- **Modify**: `internal/registry/registry.go` (add `ParseHubList`, the `Hub`/`HubList` types, and a
  `Resolve` method — **purely additive**; leave the existing `Parse`/`Entry` domains-only path and its
  package docstring intact).
- **Modify**: `go.mod` (promote `gopkg.in/yaml.v3` to a direct dependency — it is already in `go.sum`
  transitively; run `go mod tidy` so `go.mod`/`go.sum` stay consistent).
- **Create**: `internal/registry/testdata/testnet.yaml` (golden Hub-List fixture for realm 0:
  `hub_id 0 → https://sb0.iscc.id`, `hub_id 1 → https://sb1.amlet.id`, both `active: true`; these are
  the existing testnet hubs, consistent with `realm.txt` and `derive_vkey.py`). *(testdata + tests are
  excluded from the 3-file budget.)*
- **Create**: `internal/registry/hublist_test.go` (table-driven golden + failure-mode tests).
- **Reference**:
  - `/workspace/iscc-monitor/.claude/adr/0010-evidence-ledger-frontend.md` (lines 83-119 — the
    ISCC-IDv1 layout, the `hubs/<network>.yaml` schema, `pubkey` deprecated/ignored, realm→network map).
  - `/workspace/iscc-monitor/.claude/context/learnings/registry.md` (the domains-only `Parse` leaf
    conventions this must sit beside — fail-closed, no I/O, import-clean).
  - `/workspace/iscc-monitor/.claude/context/learnings/index.md` (the decoder's `{Realm, HubID,
    Timestamp}` output shape the resolver consumes; HubID is 0-4095).
  - `/workspace/iscc-monitor/internal/registry/registry.go` (existing `Parse`/`Entry` style to match).
  - `/workspace/iscc-monitor/.claude/hublist-schema-proposal.md` (the SUPERSEDED `valid_from` key
    proposal — confirms `pubkey` is ignored; do NOT implement the `keys` list).

## Not In Scope
- **Do NOT migrate the live wiring.** Leave `cmd/iscc-monitor/main.go`, `internal/config`, and the
  dashboard callers reading `realm.txt` via `Parse` exactly as they are. Swapping the live registry
  source from domains-only `realm.txt` to the Hub-List is a **separate, backward-incompatible step**
  that `state.md` + the latest `review` flag as possibly warranting a STOP for human sign-off — keep
  this step additive and reversible so that decision is not forced here. (registry is consumed only by
  `cmd/iscc-monitor/main.go` + its test, verified — but the migration still changes config + the
  public-ish realm-file contract.)
- **Do NOT remove `realm.txt`, `Parse`, or `Entry`.** They stay until the migration step.
- **Do NOT build the `/inclusion/{iscc_id}` HTML page or the proof-bundle assembler** — they are the
  next slices and re-engage the oracle/conformance gate (out of scope for this pure-leaf step).
- **Do NOT implement the `keys`/`valid_from` rotation list** (SUPERSEDED by ADR-0009 — did:web is the
  key source; `pubkey` in the Hub-List is deprecated and ignored).
- **Do NOT add a `mainnet.yaml` fixture or cross-network/realm-selection logic** beyond what one
  network needs; the realm→network selection lives in the wiring/cmd step, not this leaf.

## Implementation Notes
- **Keep `internal/registry` a pure, WASM-shareable leaf.** `ParseHubList([]byte) (*HubList, error)`
  (or `(HubList, error)`) parses bytes already in hand — **no file I/O** (the caller reads the file,
  matching `Parse`). After adding `yaml.v3`, verify the import closure still has no `net`/`net/http`/
  `database/sql`/`os` directly (yaml.v3 is pure-Go; `os` may appear transitively via `fmt`, which is
  fine — see the didweb purity nuance in `learnings.md`).
- **Types.** `Hub{ HubID uint16; URL string; Active bool }` (HubID is the embedded 12-bit field 0-4095,
  NOT the surrogate `hubs.hub_id` PK from `store.UpsertHub`). `HubList{ Version int; Network string;
  Hubs []Hub }`. Use `yaml:"…"` struct tags. **Ignore `pubkey`** — do not add a field for it (ADR-0009;
  it would invite the wrong key source).
- **Resolver returns the domain, not the URL.** The certificate page needs the **domain** for the
  `/<domain>/log/…` mount, so `(hl *HubList) Resolve(hubID uint16) (domain string, ok bool)` should
  strip the scheme (host of the URL — `strings.TrimPrefix(url, "https://")`, or `net/url.Parse` then
  `.Host`; both stdlib + WASM-safe, `net/url` does NOT pull `net/http`). Pick one and document it. A
  linear scan over `Hubs` is fine.
- **Inactive hubs still resolve.** Per ADR-0010 the monitor follows/mirrors inactive hubs and the badge
  marks them `inactive` — prefer resolving an `active: false` hub (returning its domain, `ok == true`)
  rather than treating it as "not found"; let the downstream badge convey inactivity. Document this and
  test it (a resolved inactive hub). `ok == false` is reserved for an **unknown** slot.
- **Fail closed, like `Parse`.** Reject a malformed document with a wrapped error naming the fault:
  invalid YAML, a `hub_id` outside 0-4095, a duplicate `hub_id`, or a `url` that is empty/path-bearing.
  Return `nil`/zero alongside the error (the `Parse` convention). Decide and test the empty-`hubs:`
  boundary: prefer mirroring `Parse`'s all-comment→empty-slice behavior (a document with zero hubs is
  valid, not an error).
- **Authoring the golden fixture.** No `hubs/<network>.yaml` exists in `cauldron/` to copy — author
  `testnet.yaml` from the ADR-0010 schema (lines 96-99): `version: 1`, `network: testnet`, `hub_id 0` =
  `https://sb0.iscc.id`, `hub_id 1` = `https://sb1.amlet.id`, both `active: true`. These hubs match the
  existing `realm.txt` / `derive_vkey.py` HUBS, so the fixture is grounded, not invented (note this in
  the test, like the registry learnings note the realm-file hubs are real). You MAY include a
  `pubkey:` line in the fixture (to prove it is parsed-and-ignored), but assert no `pubkey` is read into
  any type.
- **Test ground-truth, not the symbol** (per the index/registry learnings and the open `low` vacuous-
  test issue): assert `Resolve(1) == ("sb1.amlet.id", true)` against a **hard-coded** expected domain
  string, not a value re-derived from the parser. Add a small synthetic in-test YAML string exercising a
  `hub_id` like `4095` (pins the full 12-bit range) and an unknown-slot miss (`ok == false`), plus a
  `hub_id: 4096` reject (over-range) and a duplicate-`hub_id` reject.

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...`, `gofmt -l .` empty).
- `go test -count=1 ./internal/registry` passes uncached (existing `Parse` tests + the new Hub-List
  golden/failure tests).
- `ParseHubList(testnet.yaml)` then `Resolve(0) == ("sb0.iscc.id", true)` and
  `Resolve(1) == ("sb1.amlet.id", true)` (hard-coded expectations).
- `Resolve(<unknown slot, e.g. 9>)` returns `("", false)` — the documented not-found path.
- A malformed Hub-List (invalid YAML, or `hub_id > 4095`, or a duplicate `hub_id`) returns a non-nil
  error and a nil/zero list.
- `GOOS=js GOARCH=wasm go build ./internal/registry` succeeds (still WASM-shareable; no
  `net/http`/`database/sql` in the closure).
- `go mod tidy` leaves `go.mod`/`go.sum` consistent (a second `go mod tidy` is a no-op); `yaml.v3` is a
  direct `require`.

## Done When
`internal/registry` parses the iscc-hub Hub-List YAML and resolves a 12-bit `hub_id` to its hub domain
(inactive resolves; unknown misses) with golden + fail-closed tests, `mise run check` is green, and the
live `realm.txt` wiring is untouched — all Verification criteria pass.

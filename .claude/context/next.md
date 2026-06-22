# Next Work Package

## Step: Move the three instance-identity env keys into the `internal/config` leaf

## Advances
Closes the `normal` issue **"Instance-identity env keys read inline in main.go, not validated via
internal/config; CLAUDE.md env docs lack the three new keys"** (issues.md), which the latest `review`
handoff names as the explicit `**Next:**`:

> The config-leaf env move (the explicit NEXT sub-step, closes the open `normal`): move the three
> identity env keys (`ISCC_MONITOR_INSTANCE` / `ISCC_MONITOR_OPERATOR` / `ISCC_MONITOR_REALM_NAME`)
> from `main.go`'s inline `identity()` into `internal/config`'s `optional(get, key, fallback)` leaf
> (ratifying the realm-name key name) and add them to CLAUDE.md's env table — a focused `config.go` +
> `main.go` + CLAUDE.md change.

This is the smaller, cleaner first leg of the masthead-identity arc (the arc serves the M-UI
"Document chrome + instance identity" cross-cutting Verify requirement). The remaining milestone-level
Verify criteria are human-blocked (WASM "published" half — Pages repo-settings step) or design-first
(WASM signature half, OTS Bitcoin-confirmed, per-hub Anchor honesty) per `state.md`, so this `normal`
issue is the strongest code-only candidate and is explicitly queued as `**Next:**`. Preferred over
re-polishing already-met surfaces.

## Goal
Make `internal/config` the single validated source for the three optional masthead-identity strings so
every SSR masthead (current and the future proofserve trio) draws from one validated place instead of
`os.Getenv` calls in `main.go`, and document the keys in CLAUDE.md's env table. This ratifies the
`ISCC_MONITOR_REALM_NAME` key name in the config leaf (the human realm NAME, distinct from the required
realm-document PATH `ISCC_MONITOR_REALM`).

## Scope
- **Create**: (none)
- **Modify** (2 production files + 1 doc file):
  - `internal/config/config.go` — add `Instance`, `Operator`, `RealmName` string fields to `Config`;
    add the three key constants (`keyInstance`/`keyOperator`/`keyRealmName`, same literals main.go uses
    today); read each via the existing `optional(get, key, "")` helper inside `Load`. Update the package
    doc comment's "Configuration keys" block + the `Config` struct doc to list the three optional keys.
  - `cmd/iscc-monitor/main.go` — delete the `keyInstance`/`keyOperator`/`keyRealmName` const block
    (lines 281-296) and the `os.Getenv`-based `identity()` body (lines 298-309); rebuild
    `dashboard.Identity` from the `cfg` fields. `identity` is called once at `main.go:150` inside `run()`
    where `cfg` is already in scope — make it `identity(cfg config.Config) dashboard.Identity` and pass
    `cfg`. (`os` stays imported — still used for `os.ReadFile`/`os.LookupEnv`/`os.Exit`/`os.Stderr`.)
  - `CLAUDE.md` — add the three keys to the "Running a local dev instance" env-var bullet list
    (after line 47, the `ISCC_MONITOR_ADDR` bullet), each `(optional)`, noting `ISCC_MONITOR_REALM_NAME`
    is the human realm NAME distinct from the required realm-document PATH `ISCC_MONITOR_REALM`.
- **Reference**:
  - `.claude/context/learnings/config.md` — the pure-leaf rules (imports exactly `{fmt time}`,
    map-backed fake test pattern, the `optional` no-validation-beyond-default contract that `keyAddr`
    already uses).
  - `internal/config/config_test.go` — the existing `fromMap` fake + golden/defaults/table structure to
    extend (not a budget file).
  - `cmd/iscc-monitor/main.go:281-309` — the existing inline `keyInstance`/`keyOperator`/`keyRealmName`
    consts + `identity()` to delete, and the `keyRealmName` rationale comment (288-291) to port into
    config.
  - `internal/dashboard/handler.go:94-105` — the `Identity{Instance, Operator, Realm}` struct main builds
    (its `Realm` field is fed from config's `RealmName`).

## Not In Scope
- **Do NOT thread identity into the proofserve trio** (`browser.html`, `records.html`, `record.html`).
  That is the next slice and the trigger to consolidate the duplicated fallback consts (the `low`). This
  step is the config move only.
- Do NOT fold the duplicated `instanceFallback`/`operatorFallback` consts + a shared `Resolve` into one
  leaf (the `low` issue) — that lands with the proofserve slice.
- Do NOT rename `dashboard.Identity.Realm` or touch any handler's fail-safe defaulting — the handlers
  keep applying the static placeholder on an empty field; config carries empty strings when keys unset.
- Do NOT add validation/normalization to the three identity values — they are free-form display strings;
  `optional(get, key, "")` is the right (no-validation) helper, matching `keyAddr`.
- Do NOT import `dashboard` from `config` (that would invert the pure-leaf dependency / risk a cycle).
  `config` stays `{fmt time}`-only; `main.go` keeps building `dashboard.Identity` from the `Config` fields.

## Implementation Notes
- **Keep `internal/config` a pure leaf** (learnings index + `config.md`): the three new fields are plain
  `string`s read through the injected `get` closure via the existing `optional` helper — NO new imports,
  NO `os`/`net`/`dashboard`. The package must still import exactly `{fmt time}`.
- Use `optional(get, key, "")` for all three (empty-string fallback). This matches `keyAddr`'s pattern
  but with an empty default, so an unset key stays `""` and the dashboard/dossier/certificate handlers
  apply their own static-masthead fail-safe — do NOT duplicate that fallback copy in config.
- Port the `keyRealmName` rationale comment from `main.go:288-291` into config: it is the human realm
  NAME ("ISCC mainnet"), deliberately distinct from the already-required `ISCC_MONITOR_REALM` (the
  realm-document filesystem PATH); overloading the path var would leak a filename into the ledger
  subtitle. This step *ratifies* that key name in the config leaf.
- `identity()` in `main.go` becomes `identity(cfg config.Config) dashboard.Identity` returning
  `dashboard.Identity{Instance: cfg.Instance, Operator: cfg.Operator, Realm: cfg.RealmName}`; its single
  caller at `main.go:150` passes `cfg`. Rewrite both doc comments (the const block is gone; `identity`
  is now config-sourced) to describe the current state, not the move (evergreen-comment rule).
- Extend `config_test.go`: `TestLoadGolden` adds the three keys to its input map and the three fields to
  `want` (full round-trip). `TestLoadDefaults` (minimal input) asserts the three fields are `""` when the
  keys are absent. Add a `TestLoad` table case for a partially-set identity (e.g. only `keyInstance` set,
  the others `""`) so per-field independence is non-vacuous.
- **No crypto correctness rule applies** (pure startup-value parsing — no signature/Merkle/did:web/proof
  path). Oracle gate is N/A; `go.mod`/`go.sum`/`schema.sql` stay byte-identical (state this in the verdict).

## Verification
- `mise run check` is green (build + vet + `go test ./...` + `gofmt -l .` empty outside `cauldron/`).
- `go test -count=1 -run TestLoad ./internal/config` passes (golden round-trip incl. the three identity
  fields, the absent→`""` default, and the partial-set case).
- `go test -count=1 ./cmd/iscc-monitor` passes (the `identity(cfg)` signature compiles and wires at the
  `main.go:150` call site).
- Assertion: with `ISCC_MONITOR_INSTANCE` / `ISCC_MONITOR_OPERATOR` / `ISCC_MONITOR_REALM_NAME` set,
  `config.Load(...)` returns a `Config` whose `Instance`/`Operator`/`RealmName` equal those values;
  with the keys absent, those three fields are `""`.
- Assertion: `go list -deps github.com/iscc/iscc-monitor/internal/config` adds no new import — the
  package's direct imports stay `{fmt time}` (no `os`/`dashboard`/`net`).
- Mutation check (advance runs + reverts byte-clean): forcing `optional(get, keyInstance, "")` to return
  a literal makes `TestLoadGolden`/`TestLoadDefaults` FAIL; restore, `git diff` clean.

## Done When
`internal/config` parses the three identity keys into typed `Config` fields, `main.go`'s `identity(cfg)`
builds `dashboard.Identity` from those fields (no `os.Getenv` for identity), CLAUDE.md documents the
three keys with the realm-name-vs-path distinction, and all Verification checks pass.

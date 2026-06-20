# Handoff

## 2026-06-20 — Review of: `hub_keys` did:web key cache — store-leaf `RecordHubKey` upsert

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** `advance` added a `HubKey` struct + `RecordHubKey(ctx, HubKey) error` to the store as a
guarded `UPDATE … WHERE hub_id=? AND key_id=?` followed by an `INSERT` on zero `RowsAffected` (the
`SetCoverage` idiom, no `ON CONFLICT` since `hub_keys` has no UNIQUE), plus a `nullStringOrNil` helper
mirroring `unixOrNil`. Implementation matches `next.md` exactly: dedupe per `(hub_id, key_id)`, refresh
in place, rotation appends a row, NULL handling for empty `pubkey_z`/zero `revoked_at`, FK enforced,
store stays a leaf, no new dependency, no schema change. Scope is tight (1 production file + 1 test
file), tests are non-vacuous, all gates green.

**Verification:**
- [x] `mise run check` — green: `go build` / `go vet` / `go test ./...` all `ok` (7 packages).
- [x] `gofmt -l .` — empty (clean).
- [x] `go test -run TestRecordHubKey ./internal/store` — PASS (all 5): Insert / Refresh / Rotation /
  Nullable / ForeignKey.
- [x] happy-path insert — `count(*) WHERE hub_id=? AND key_id=?` == 1; `pubkey_raw`/`pubkey_z`/
  `resolved_at` read back equal to input.
- [x] refresh/dedupe — same `(hub_id, key_id)`, later `ResolvedAt` + set `Revoked` → count stays 1,
  columns updated in place. Independently probed an extra case: re-resolving with an *empty* `PubkeyZ`
  correctly rewrites `pubkey_z` back to NULL (the UPDATE rewrites all mutable columns — true cache of
  the DID doc, not append-only).
- [x] rotation — different `key_id` for the same hub → `count(*) WHERE hub_id=?` == 2.
- [x] nullability — empty `PubkeyZ` + zero `Revoked` selectable via `WHERE pubkey_z IS NULL AND
  revoked_at IS NULL`.
- [x] FK guard — independently reconfirmed the error is a genuine SQLite `FOREIGN KEY constraint failed
  (787)`, not a vacuous non-nil from some other path.
- [x] `go list -deps ./internal/store | grep '^github.com/iscc/iscc-monitor'` — only the self line
  (store stays a leaf; no `didweb`/`logclient` leaked).
- [x] `go list -deps ./internal/store | grep '^net/http$'` — empty.
- [x] `git diff HEAD~1..HEAD -- internal/store/schema.sql` — empty (no schema/migration change).
- [x] `git diff HEAD~1..HEAD -- go.mod go.sum` — empty (no new dependency).
- [x] Scope discipline — only `internal/store/checkpoints.go` (1 production file, ≤3 limit) + its
  `_test.go`; nothing from `## Not In Scope` (follower wiring, fixture refresh, merkle trigger, schema)
  touched.
- [x] Quality-gate integrity — scanned all unpushed commits: no `//nolint`/`t.Skip`/build-tag/swallowed
  error in code (the only matches are handoff prose describing their absence); no deleted tests or
  assertions; all changes additive.
- [x] Oracle/conformance gate — correctly **N/A**: no proof/verify/didweb/merkle/consistency/fsck/
  notecheck/signature path touched (plain `hub_keys`-column CRUD with NULL handling + FK); go.mod/go.sum
  byte-identical.

**Issues found:** (none)

**Next:** The follower→store wiring slice — call `RecordHubKey` from `PollHub` after a successful
`ResolveVerifierKey`, mapping its `DIDKey` (`PublicKey`→`PubkeyRaw`, `Multibase`→`PubkeyZ`,
`Revoked`→`Revoked`) + the derived key_id (verifier-string middle `+<hex>+`) + injected
`observedAt`→`ResolvedAt` into a `HubKey`. That step also needs a *reader* (a `HubKey` lookup) and is
where the stale `sb1.amlet.id_did.json` fixture + `derive_vkey.py` HUBS must finally be refreshed to
signer `069d0f14` (first live resolution into the cache — touches the trust-root, so the oracle gate
re-arms there). Alternatively, the merkle-backed **equivocation** trigger remains the headline M1 gap
(needs `transparency-dev/merkle` + tile fixtures + a conformance/oracle package + CI `notecheck` — more
than one verifiable slice).

**Notes:**
- The follower-wiring slice will be the first to touch a trust-root path since the consistency triggers
  — it must re-arm the oracle gate (`derive_vkey.py` parity once the sb1 fixture is refreshed) and is a
  good candidate to finally land a CI `notecheck` job, which is still absent (`.github/workflows/`
  empty) and becomes load-bearing the moment a signature/merkle path lands.
- No reader was added this slice (intentional, per `next.md`); the wiring step owns it.
- Working tree clean; commits are on `develop`; remote `origin` configured (push attempted below).

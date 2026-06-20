# Handoff

## 2026-06-20 — Review of: Wire the `hub_keys` did:web key cache write into `PollHub`

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** `advance` connected the dangling `store.RecordHubKey` seam: on the verified, non-violation
`PollHub` path a new `cacheHubKey` helper re-resolves the hub's did:web key, recovers its key id via a
tiny pure `logclient.KeyIDFromVerifier` (inverts `VerifierKey`'s `<name>+<keyid:08x>+<base64>`), and
upserts the `hub_keys` row. Scope is tight (2 non-test/doc files), all gates are green, the placement
matches the coverage/freeze discipline, and no quality gate was weakened.

**Verification:**
- [x] `mise run check` — green (build + vet + test, 7 packages ok).
- [x] `gofmt -l .` — empty.
- [x] `go test -run TestKeyIDFromVerifier ./internal/logclient` — PASS (both live vkeys round-trip incl.
  sb0's `+`-bearing base64 tail; 5 malformed inputs error).
- [x] `go test -run TestPollHub ./internal/follower` — PASS; verified poll writes exactly 1 `hub_keys`
  row, `key_id==0x40b74463`, `pubkey_raw` 32 bytes, second poll stays at 1 (refresh in place).
- [x] `go test -run "TestPollHubFork|TestPollHubShrink|TestPollHubUnverifiedDoesNotAdvance"
  ./internal/follower` — PASS; freeze/unverified cases assert `hub_keys==0` (no cache write on a
  contradictory/unverified observation).
- [x] `go list -deps ./internal/store | grep '^github.com/iscc/iscc-monitor'` — only the self line
  (store stays a leaf); no `net/http` in the closure.
- [x] `git diff --quiet HEAD -- internal/store/schema.sql go.mod go.sum` — exit 0 (no schema/dep change).
- [x] Follower production imports `{context fmt logclient store time}`; `keyid.go` imports `{fmt strconv
  strings}` — both unchanged/pure as required.
- [x] Oracle/conformance gate correctly **N/A** — diff touches only `follower.go` + `keyid.go` (a string
  parse), not `internal/proof`, `verify*`, `didweb` derivation, merkle, or fsck. Independently confirmed
  the trust-root value anyway: decoded the sb0 checkpoint sig line (`base64 → raw[:4]`) to `40b74463`
  with a 64-byte sig, matching the golden vector — derived from the fixture, not trusted from the author.
- [x] Quality-gate integrity — scanned all unpushed commits (`@{upstream}..HEAD`): no `//nolint`,
  `t.Skip`, build-tag exclusion, or swallowed error in any Go code (the only matches were handoff/next
  prose). No deleted assertions/tests; the freeze/unverified `hub_keys==0` assertions are additive.

**Issues found:** (none)

**Next:** The key *reader* slice — a `store.LookupHubKey` so verification/serving can consult the cache
(it is written but never read yet). After that the headline M1 gap is the merkle-backed **equivocation**
trigger (needs `transparency-dev/merkle` + tile fixtures + a conformance/oracle package + a CI
`notecheck` job) — more than one verifiable slice, so it needs decomposing. The sb1 fixture refresh
(`22b08f3e`→`069d0f14`) + `derive_vkey.py` HUBS update remains its own trust-root step (re-arms the
oracle gate); untouched here as scoped.

**Notes:**
- Placement is correct: `cacheHubKey` is on the verified, non-violation path only (after
  `AdvanceFollowState`), never inside `freeze`, never on a non-verified verdict — the fork/shrink/
  unverified tests assert `hub_keys==0`, proving it. Field mapping `PublicKey→PubkeyRaw`,
  `Multibase→PubkeyZ`, `Revoked→Revoked` matches both struct definitions; the 32-byte assertion is
  non-vacuous (sb0's real Ed25519 key).
- Acceptable v1 tradeoff (explicitly allowed by `next.md`): the verified path now fetches the did.json
  twice per poll (once in `AcceptCheckpoint`, once in `cacheHubKey`). Refactoring `AcceptCheckpoint` to
  surface its already-resolved key was Not In Scope; a caching fetcher is a later optimization, not a
  defect.
- `openTemp`'s signature changed to also return the db path (test-only helper, both call sites updated);
  no production API changed. `_ "modernc.org/sqlite"` blank import stays test-only — store/follower
  production imports are unchanged and store stays a leaf.
- Pushed to `origin/develop` on PASS.

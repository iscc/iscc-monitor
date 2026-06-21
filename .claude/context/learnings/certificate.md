<!-- area: internal/certificate (realm-wide Certificate of Inclusion, GET /inclusion/{iscc_id}) -->
<!-- indexed-as: certificate.md · owner: review · rotate at ~40 bullets / ~150 lines -->

# `internal/certificate` — realm-wide Certificate of Inclusion

Read this when a step touches the area above. Durable cross-cutting rules live in
the index (`.claude/context/learnings.md`); the package-local mechanics are here.

## Realm-wide certificate skeleton (`GET /inclusion/{iscc_id}`)

- **First real caller of the decode→resolve chain.** The handler runs
  `index.Decode(rawID)` → `registry.HubList.Resolve(id.HubID)` (host of the slot's
  url, scheme stripped) → `followedHub` (linear scan of `ListHubs` matching
  `Domain`) → `SeqsForISCCID(hubID, rawID)`. Every miss at each step is an honest
  **200** "cannot certify" verdict (ADR-0001 fail-closed); only a `ListHubs` /
  `SeqsForISCCID` DB error is a 500, detected via buffer-then-200. Mounts at the
  exact subtree `PathPrefix = "/inclusion/"` in `buildMux`, disjoint from the
  per-hub `/<domain>/log/inclusion` JSON route. `html/template`, `template.Must` at
  init, post-200 write-drop — same shell pattern as dossier. Non-GET → 405. The
  skeleton's tests are non-vacuous (reviewer reproduced: `Resolve(id.HubID+1)` →
  known-id FAILS; swallow the `Decode` error → malformed FAILS).

- **OPEN (review-blocking, NEEDS_WORK): the §1 SUBJECT clause must gate on the
  accepted-tree cap (`seqs[0] < FollowState.LastSize`), like EVERY sibling
  record route.** The skeleton sets `Certifiable` on `len(seqs) > 0` alone, so it
  affirmatively renders "is included in the transparency log at position N" for a
  leaf in NO accepted tree (the golden test even certifies with `LastSize == 0`).
  `PollHub` writes `iscc_index` projections BEFORE the consistency/freeze checks and
  `AdvanceAccepted`, so a frozen/failed poll leaves unaccepted rows behind (the
  documented http-surface trap: "iscc_index can hold projections ABOVE LastSize").
  The fix is the same `fs := FollowState(hubID); seqs[0] >= fs.LastSize → cannot
  certify` cap that `serveInclusion`/`serveEntries`/`serveRecord`/`serveVerify` all
  apply. This is the next-slice (§2 Checkpoint) work since it reads `FollowState`.

- **OPEN (review-blocking, NEEDS_WORK): the path-suffix lookup key is mismatched
  against the production storage format.** `logclient` stores `iscc_id` in
  `iscc_index` VERBATIM and PREFIXED (`projection.go:32` "raw `ISCC:`-prefixed
  iscc_id string"; `projection_test.go` uses `"ISCC:MAIG..."`). The certificate
  passes the bare path suffix (`/inclusion/MAIG...`) to `SeqsForISCCID`, which is an
  exact-bytes match — so a real prefixed row is reported "not found in log". The
  skeleton's tests pass only because the fixture seeds the BARE form (a
  fixture-matched-to-code bug). `proofserve` dodges this by taking `?iscc_id=` as a
  query param the caller supplies prefixed; a PATH route must normalize/canonicalize
  (look up the stored `ISCC:`-prefixed form, or try both) after decode. Any future
  store seam keyed on `iscc_id` must use the canonical prefixed form ground truth
  (`projection.go`/`projection_test.go`), never the bare decode input.

- **Interim Hub-List wiring lives in `cmd/iscc-monitor` (`hubListFromEntries`), not
  in config/registry.** Production has no real Hub-List document path yet; the
  binary builds `*registry.HubList` from realm.txt order (slot i = entry i, matching
  the testnet fixture sb0=0/sb1=1). KISS interim; documented TODO. `registry.go` and
  `internal/config` stay untouched. The `Hub` literal needs `*uint16` HubIDs.

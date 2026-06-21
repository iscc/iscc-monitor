## 2026-06-21 — Make the certificate §1 inclusion claim sound — accepted-tree cap + ISCC:-prefixed lookup

**Done:** Closed the two open `critical` certificate defects in ONE slice: `buildData` now (a)
canonicalizes the lookup id to the stored `ISCC:`-prefixed form before `SeqsForISCCID`, and (b) gates
the affirmative inclusion claim on the accepted-tree cap (`len(seqs) > 0 && seqs[0] < LastSize`),
rendering honest cannot-certify states ("no accepted checkpoint yet" / "not in accepted tree")
otherwise. `LastSize` is carried out of the existing `ListHubs` scan via `followedHub` (now returns the
matched `store.HubSummary`) with no second store round-trip. Fixtures re-grounded to the production wire
format (prefixed id + an accepted checkpoint via `AdvanceAccepted`).

**Files changed:**
- `internal/certificate/handler.go` (only non-test source file, 1 of ≤3): `followedHub` returns
  `store.HubSummary` (carries `HubID` + `LastSize`); `buildData` builds `lookupID := "ISCC:" +
  strings.TrimPrefix(rawID, "ISCC:")`, looks up the prefixed form, and applies the accepted-tree cap
  before setting `Certifiable`. Package/`buildData`/`certData.Certifiable` doc comments updated to match.
- `internal/certificate/handler_test.go` (test, not counted): `fixtureStore` now indexes under
  `"ISCC:"+indexedID` and seeds an accepted checkpoint (`LastSize = seq+1`); new
  `fixtureStoreUnaccepted` helper takes an explicit `LastSize`. Added `TestCertificatePrefixedLookup`
  (bare + prefixed request both certify) and `TestCertificateUnacceptedLeaf` (above-accepted-tree +
  no-checkpoint subtests render cannot-certify). Updated `TestCertificateKnownID` docstring.
- `cmd/iscc-monitor/main_test.go` (test, not counted): re-grounded `TestCertificateRouteMounted` —
  indexes the golden id as `"ISCC:MAIGHFECJMOPMIAB"` and accepts a checkpoint at size 24816 — so the
  end-to-end mux route still certifies under the new (correct) gating.

**Verification:** `mise run check` → green (build + vet + all 22 packages; `gofmt -l .` empty).
- `go test -count=1 ./internal/certificate` → pass uncached.
- `TestCertificateKnownID` → pass: bare-suffix request certifies the prefixed leaf within the accepted
  tree, position 24815, §1 SUBJECT, sb1.amlet.id.
- `TestCertificateUnacceptedLeaf` → pass (both subtests): seq >= LastSize → "not in accepted tree";
  LastSize == 0 → "no accepted checkpoint yet"; neither shows the subject banner.
- `TestCertificatePrefixedLookup` → pass: `/inclusion/MAIGHFECJMOPMIAB` and
  `/inclusion/ISCC:MAIGHFECJMOPMIAB` both certify the same leaf.
- Mutation checks (reviewer-reproducible, tree restored after each):
  (a) removing the `seqs[0] >= LastSize` / `LastSize == 0` cap → `TestCertificateUnacceptedLeaf` FAILS
  (renders the certifiable banner);
  (b) reverting `lookupID` to bare `rawID` → `TestCertificateKnownID` + `TestCertificatePrefixedLookup`
  FAIL ("not found in log"). Both non-vacuous.

**Next:** §1 is now sound — proceed to the §2 Checkpoint clause (`HasClause2`), which reads
`FollowState`/`CheckpointAt(hubID, LastSize)` to render the accepted `(size, root)` the cap already
keys on. The accepted-checkpoint plumbing this slice introduced (the `HubSummary.LastSize` carry) is
the same data §2 displays, so it should reuse it. After §2, §3 Inclusion proof re-engages the
oracle/conformance gate (the proof bundle must be mutation-proven non-vacuous against the hub's
`IsccLogInclusionProof`).

**Notes:**
- The oracle/conformance gate is still N/A this slice: it remains a pure HTML render of decode +
  registry resolve + store reads (no signature/RFC-6962/Merkle/did:web/fsck/proof path). `go.mod`/
  `go.sum` byte-unchanged (no new deps). The gate APPLIES starting at §3.
- No template edit was needed (as next.md predicted): both new cannot-certify states route through the
  existing `cert.html` `{{else}}` not-found branch via `data.Reason`. `data.Domain = domain` is set on
  every post-resolve branch so the page names the hub.
- `cmd/iscc-monitor/main_test.go` was an unavoidable third file to touch, but it is a TEST file
  (scope counts non-test source files; only `handler.go` is source). It was a pre-existing
  fixture-matched-to-code instance of the SAME two bugs (bare id, no accepted checkpoint) — re-grounding
  it to ground truth was required to keep `mise run check` green and is exactly the fix this slice is
  about.
- The frozen-hub edge case needs no separate branch: a frozen hub's `LastSize` is its last *accepted*
  size (freeze stops advance, ADR-0006), so the `seqs[0] < LastSize` cap already caps a frozen hub at
  its accepted window. Documented in the code comment.
- The two `critical` issues in `issues.md` are now closed by this change (review should delete them).
- `learnings/certificate.md` has two OPEN (review-blocking) bullets describing exactly these fixes;
  they can be marked resolved.

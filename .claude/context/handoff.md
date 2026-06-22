## 2026-06-22 — Bind the §5 OTS proof's committed digest to §2's accepted root before rendering BITCOIN ANCHOR

**Done:** Added a digest-bound classifier `ots.ConfirmedFor(otsBytes, root)` that fail-closes (wrapped
error) unless the parsed proof's committed SHA-256 digest equals the caller's root, then certificate §5
now calls `ots.ConfirmedFor(rec.OTSBytes, root)` instead of the digest-agnostic `ots.Confirmed`. A
mis-stamped (root, proof) row now declines §5 silently (no fabricated "block N"), exactly like an
unparseable proof. `Confirmed` and `ConfirmedFor` share one parse (`recoverParse`) and one classify
(new private `classify` holding the attestation + `>MaxInt64` overflow logic).

**Files changed:**
- `internal/ots/ots.go`: added `ConfirmedFor` (parse → `bytes.Equal(file.Digest, root)` gate → shared
  `classify`); extracted the attestation+overflow logic into a private `classify(file)` called by both
  `Confirmed` and `ConfirmedFor` (one parse, one overflow guard); imported `bytes`; updated the package
  docstring to describe the two-classifier-on-one-core design and the binding.
- `internal/certificate/handler.go`: §5 now calls `ots.ConfirmedFor(rec.OTSBytes, root)` (the in-scope
  `root` is §2's raw `CheckpointAt` bytes); the existing `cerr == nil` guard already declines a mismatch
  silently — no new branch, no new 500. Updated the §5 inline docstring and the three buildData/certData
  doc references (`handler.go:40`, `:299`, `:341-343`, `:642-646`, `:919-923`) to state the binding.
- `internal/ots/ots_test.go` (test): added `TestOTSConfirmedForDigestBound` (confirmed/pending under the
  fixture's own committed digest; mismatch declines with non-nil err; parse error) + a `mustHex` helper.
  The digest literals (`03ba20…`, `d32fee…`) are the fixtures' own committed SHA-256 (verified via
  `opentimestamps.File.Digest`), pinned with a ground-truth comment like the heights.
- `internal/certificate/handler_test.go` (test): added `seedOTSAtRoot` (records an OTS row at an
  arbitrary root; `seedOTS` now delegates to it at `tree.Hash()`) and `otsFixtureDigest`. Reworked
  `TestCertificateBitcoinAnchorConfirmed/Pending` to drive §2's accepted root to the fixture's committed
  digest (`fixtureStoreTiled` acceptedRoot override) and seed the row at that same digest, so the binding
  holds and §5 renders; these no longer assert §3 (a forced `acceptedRoot != tree.Hash()` makes §3
  decline — documented in the test). Added `TestCertificateBitcoinAnchorDigestMismatch` (row seeded at
  `tree.Hash()`, which the bundled proof never commits to → §5 OMITTED, §1-§3+§6 still render).

**Verification:** `mise run check` → green (build + vet + `go test ./...`, all 28 packages ok; `gofmt -l
.` empty).
- `go test -run TestOTSConfirmedFor -v ./internal/ots` → PASS (lists confirmed/pending/mismatch/parse
  subcases).
- `go test -run TestCertificateBitcoinAnchor -v ./internal/certificate` → PASS (Confirmed, Pending,
  DigestMismatch, Unanchored, EmptySentinel).
- Mutation 1 (ots): removing the `bytes.Equal` gate from `ConfirmedFor` → `…DigestBound/mismatch_declines`
  FAILS ("want digest-mismatch error, got nil"). Restored byte-identical.
- Mutation 2 (certificate): reverting §5 to `ots.Confirmed(rec.OTSBytes)` → `…DigestMismatch` FAILS
  (renders §5 for the mismatched row). Restored byte-identical.
- WASM-pure invariant: `go list -deps` of `internal/{didweb,index,badge}` + `internal/proof/verify` each
  show 0 hits on `internal/ots`; all four still `GOOS=js GOARCH=wasm go build`. No `go.mod`/`go.sum` drift.

**Next:** Two §5/§6-adjacent `normal`s remain in the named-order queue: (1) the §6 `· at` timestamp
`normal` needs a store schema column on `RecordRow` (larger, store-touching — define carefully). (2) the
certificate tier-2 honesty-copy overstatement (`cert.html:465`). The §5 OTS-digest-binding `normal` this
step closed should be deleted from issues.md by `review` after independent mutation re-verification. The
front-of-queue WASM-verifier signature half remains design-first / a STOP-candidate (browser did:web
resolution).

**Notes:**
- Scope: 2 non-test/doc files (`ots.go`, `handler.go`) within budget, plus 2 test files. Nothing from
  `## Not In Scope` touched — `.ots` fixtures unchanged (byte-verbatim oracle), no OTS write path
  (`otsloop.go`/`otsclient`/`store/ots.go`), `Confirmed`'s signature + its `.ots`-route / upgrade-loop
  callers (`otsclient/client.go:109`) unchanged.
- Followed plan option (A): confirmed/pending tests now drive `acceptedRoot == fixture digest`, which
  intentionally diverges from the mirrored `tree.Hash()`, so §3 declines in those two tests — they no
  longer assert §3 (covered by the clean-tree §3 tests + the new `…DigestMismatch` which keeps a clean
  tree so §3 renders). The `…DigestMismatch` test reuses the existing `seedOTS`-at-`tree.Hash()` seam as
  exactly the mis-stamped case, as the plan noted.
- The `Confirmed` function docstring was left intact: it still accurately describes `Confirmed`'s observable
  behavior even though the classification detail now physically lives in the shared `classify` helper. The
  package docstring carries the new two-classifier description.
- No backward-incompatible API change: `Confirmed` is unchanged; `ConfirmedFor` is purely additive.

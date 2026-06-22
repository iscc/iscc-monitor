# Next Work Package

## Step: Bind the §5 OTS proof's committed digest to §2's accepted root before rendering BITCOIN ANCHOR

## Advances
Closes the front-of-queue code-closable `normal` issue **"Certificate §5 BITCOIN ANCHOR does not bind
the OTS proof's committed digest to §2's accepted root"** (issues.md:21). This preempts further
milestone work because it is the named-next item in `state.md` "Next Milestone" step 2 and the latest
`review` handoff ("Next") — and the milestone Verify criterion it serves is the M-UI **certificate
Bitcoin-anchor** clause: *"the **Bitcoin-anchor** panel … a not-yet-anchored root renders the normal
'pending' state, not an error"* (target.md M-UI). The always-loaded learnings rule applies directly:
*"On a self-verifiable surface, gate a rendered ✓/Merkle assertion on a re-VERIFICATION, not a status
flag. A built proof is not a verified proof."* Today §5 renders "block N" for **any** parseable
Bitcoin attestation without checking the proof actually commits to §2's root — a false anchor claim on
a Tier-1 surface. DONE still requires 0 `normal`; this drains one.

## Goal
Make §5 BITCOIN ANCHOR fail-closed: render the anchor only when the stored OTS proof's committed
SHA-256 digest equals §2's accepted root, so a mis-stamped (root-key, proof-digest) row declines §5
exactly like an unparseable proof instead of falsely vouching that §2's root is Bitcoin-confirmed.

## Scope
- **Modify**: `internal/ots/ots.go` — add a digest-bound classifier `ConfirmedFor(otsBytes, root
  []byte) (confirmed bool, height int64, err error)` that parses once (reusing `recoverParse` + the
  `>MaxInt64` height guard) and **fail-closes with a wrapped error when `!bytes.Equal(file.Digest,
  root)`** before classifying attestations. Keep `Confirmed` as the digest-agnostic primitive (still
  used by the `.ots` route / upgrade loop, which already key on the row's own root); share ONE parse +
  ONE overflow guard between the two (don't duplicate the recover/overflow logic). Update the package
  + function docstrings to describe the binding. (1 non-test/doc file.)
- **Modify**: `internal/certificate/handler.go` — at §5 (the `if found && len(rec.OTSBytes) > 0`
  block, handler.go:929-940) replace `ots.Confirmed(rec.OTSBytes)` with `ots.ConfirmedFor(rec.OTSBytes,
  root)` (the `root` var already in scope is §2's raw `CheckpointAt` bytes). A digest mismatch now
  returns a non-nil `cerr`, so the existing `if … cerr == nil` guard already declines §5 silently — no
  new branch, no new 500 path. Update the §5 docstring (handler.go:904-923) to state the binding. (2nd
  non-test/doc file.)
- **Tests** (NOT counted in the 3-file budget): `internal/ots/ots_test.go` (digest-binding unit; keep
  the confirmed/pending positive coverage here, where the fixture digest is known ground truth) and
  `internal/certificate/handler_test.go` (rework the §5 confirmed/pending render tests + add the
  mismatch-decline test). See Implementation Notes — the existing certificate §5 fixtures seed a row
  root that does NOT equal the proof digest, so they MUST change.
- **Reference**:
  - `.claude/context/learnings/certificate.md` (§5 mechanics; the KNOWN-GAP note on the §5 bullet is
    exactly this issue) and `.claude/context/learnings/ots.md` (parse/recover/overflow-guard traps;
    the "fixtures ARE the `ots verify` oracle" rule).
  - `internal/certificate/handler.go:904-940` (§5 read), `internal/ots/ots.go:58-93` (`Confirmed` +
    `recoverParse`).
  - `github.com/nbd-wtf/opentimestamps@v0.4.0/ots.go:59-62` — `File.Digest []byte` is the 32-byte
    SHA-256 the proof commits to (confirmed present in the module cache).
  - `internal/certificate/handler_test.go:1481-1624` (`otsFixture`, `seedOTS`, the §5
    confirmed/pending/unanchored tests) and `internal/ots/ots_test.go:38-113` (`TestOTSConfirmed*`).

## Not In Scope
- Do NOT regenerate or fabricate `.ots` fixtures. The bundled `hello-world.txt.ots` (committed digest
  `03ba204e50d126e4674c005e04d82e84c21366780af1f43bd54a37816b6ab340`, height 358391) and
  `merkle1.txt.ots` (committed digest `d32fee9a827f5a0d580f80beb7edce662dd99fcd6591e4ef8a6244403df0b7c9`,
  pending) are the external `ots verify` oracle — keep them byte-verbatim. Bind the **test's row root**
  to the fixture's digest, never the other way around (don't invent a proof for a Merkle root).
- Do NOT touch the OTS write path (`follower/otsloop.go`, `otsclient`, `store/ots.go`) — production
  always stamps the row's own `r.Root`, so a real production row is already digest-bound; this step
  only hardens the READ/render gate. The nil-Stamper `low` (otsloop.go:144) stays untouched.
- Do NOT change `Confirmed`'s signature or its `.ots`-route / upgrade-loop callers — they legitimately
  key on the row's own root and don't need the binding.
- Do NOT address the §6 `· at` timestamp `normal` or the tier-2 honesty-copy `normal` (separate steps).

## Implementation Notes
- **`ConfirmedFor` shape.** Parse via the existing `recoverParse` (the library PANICS on malformed
  input — keep that guard, ots.md). After a successful parse, `if !bytes.Equal(file.Digest, root) {
  return false, 0, fmt.Errorf("ots.ConfirmedFor: proof digest %x does not commit to root %x", …) }` —
  fail-closed, same wrapped-error contract as a parse failure, so the certificate's `cerr == nil`
  guard treats a mismatch identically to an unparseable proof (silent §5 decline, never 500). Then run
  the existing attestation classification (the `>MaxInt64` guard stays BEFORE the int64 cast). Import
  `bytes`. `internal/ots` stays NOT WASM-pure (it already isn't) — no new WASM concern.
- **Keep ONE parse / ONE overflow guard.** Refactor so `Confirmed` and `ConfirmedFor` share the parse
  + classify; the only difference is `ConfirmedFor` interposes the `bytes.Equal` digest check between
  parse and classify. Suggested: a private `classify(file *opentimestamps.File) (bool, int64, error)`
  holding the attestation + overflow logic, called by both after their respective parse/gate.
- **ots unit test (positive coverage lands HERE, where the digest is ground truth).** Add
  `TestOTSConfirmedForDigestBound`:
  - `ConfirmedFor(hello-world.txt.ots, <its digest 03ba20…>)` → `(true, 358391, nil)` (pin the digest
    literal with a comment that it is the file's committed SHA-256, ground truth like the height).
  - `ConfirmedFor(merkle1.txt.ots, <its digest d32fee…>)` → `(false, 0, nil)` (pending, bound).
  - `ConfirmedFor(hello-world.txt.ots, <any other 32-byte root>)` → `(false, 0, non-nil err)` (mismatch
    declines). **Mutation anchor:** removing the `bytes.Equal` gate makes this mismatch subcase return
    `(true, 358391, nil)` and FAIL — keep it non-vacuous.
- **certificate §5 tests MUST change** — the load-bearing rework. Today
  `TestCertificateBitcoinAnchorConfirmed/Pending` seed the row at `tree.Hash()` (handler_test.go:1506)
  while the fixture commits to its OWN digest, so once the binding lands those tests FAIL (the proof
  digest never equals the Merkle root). Two options — prefer (A):
  - **(A) Add a `seedOTSAtRoot(t, st, domain, treeSize, root, otsBytes, upgradedAt)` helper** that
    records the OTS row at an ARBITRARY root (the fixture's committed digest) instead of `tree.Hash()`,
    and drive §2 with an accepted checkpoint whose root EQUALS that digest — `fixtureStoreTiled` already
    takes an `acceptedRoot` override (handler_test.go:569). With accepted root == fixture digest, §2
    renders that root, the row keys on it, the digest binds, and §5 renders "block 358391". NOTE: a
    forced `acceptedRoot != tree.Hash()` makes §3 decline (mirror root ≠ accepted root) — that's fine
    for a §5-focused test (assert §5 renders; do NOT assert §3). Keep ONE existing clean-tree test that
    still exercises §1-§4+§6 together (it will no longer render §5, since its real fixtures mismatch —
    drop its §5 assertions, or make it the mismatch-decline test below).
  - **(B)** If (A)'s §3-decline coupling is awkward, move the "renders block 358391" positive assertion
    entirely to the ots unit above and let the certificate confirmed/pending tests assert §5 **declines**
    on the real fixtures (the honest post-binding behavior, digest ≠ tree root).
  - Either way, ADD `TestCertificateBitcoinAnchorDigestMismatch`: seed a confirmed proof whose digest
    does NOT match §2's root (the current `seedOTS`-at-`tree.Hash()` is already exactly this case) and
    assert §5 is OMITTED (no "§5 BITCOIN ANCHOR", no "block "), while §1-§3+§6 still render.
    **Mutation:** reverting §5's call to `ots.Confirmed(rec.OTSBytes)` renders §5 for this mismatched
    row and FAILS the test.
- **Correctness rule (learnings index):** "gate a rendered ✓/anchor on a re-VERIFICATION, not a
  classify-only flag" — the binding upgrades `ots.Confirmed`'s classify-only verdict into a true
  verification that the proof commits to the asserted root. Fail-closed per ADR-0001: a mismatch
  declines, never 5xx, never a fabricated anchor.
- **Oracle/conformance honesty (ots.md):** the bundled `.ots` files remain the external `ots verify`
  oracle; heights (358391) and digests stay ground-truth literals, never derived from monitor code.

## Verification
- `mise run check` is green (build + vet + `go test ./...`, all packages; `gofmt -l .` empty).
- `go test -count=1 -run TestOTSConfirmedFor ./internal/ots` passes; `-v` lists the digest-bound
  confirmed, pending, and mismatch-decline subcases.
- `go test -count=1 -run TestCertificateBitcoinAnchor ./internal/certificate` passes (the reworked
  confirmed/pending render tests + the new `…DigestMismatch` decline test).
- Mutation (manual, record in handoff): removing `bytes.Equal(file.Digest, root)` from `ConfirmedFor`
  makes the `TestOTSConfirmedForDigestBound` mismatch subcase FAIL; reverting §5's call to
  `ots.Confirmed(rec.OTSBytes)` makes `TestCertificateBitcoinAnchorDigestMismatch` FAIL.
- No WASM-pure leaf (`internal/{didweb,index,badge}`, `internal/proof/verify`) imports `internal/ots`:
  `go list -deps` of each shows 0 hits on `internal/ots` (unchanged; `ConfirmedFor` adds no importer).

## Done When
`mise run check` is green, the ots `ConfirmedFor` digest-binding unit and the reworked + new
certificate §5 tests all pass, and the two mutations above each fail their guarding test — so §5
renders the Bitcoin anchor only for a proof that provably commits to §2's accepted root.

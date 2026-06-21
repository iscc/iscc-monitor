<!-- area: cmd/notecheck (main.go) -->
<!-- indexed-as: notecheck.md · owner: review · rotate at ~40 bullets / ~150 lines -->

# `cmd/notecheck` — external signature-parity oracle

Read this when a step touches the area above. Durable cross-cutting rules live in
the index (`.claude/context/learnings.md`); the package-local mechanics are here.

## notecheck oracle port (`cmd/notecheck`)

- **The external signature-parity oracle now has an in-repo compile path** — `cmd/notecheck/main.go`
  is a faithful port of `cauldron/iscc-hub/conformance/notecheck/main.go` (gitignored, module-less)
  into `package main` under the monitor module. Reviewer diffed it line-for-line against the reference:
  same two-verifier composition (`f_note "…/formats/note".NewVerifier` + `x/mod/sumdb/note.Open`), same
  strict reject `len(n.Sigs)==0 || len(n.UnverifiedSigs)!=0`, same exit codes (2 setup / 1 verify / 0
  ok). The reject is byte-identical to `internal/logclient/verify.go` — the load-bearing parity
  invariant. Reference printed `OK <name> <size>` in its doc but `OK <name>` in code; the port matches
  the CODE (and `next.md`), confirmed end-to-end (`OK sb0.iscc.id/log`).
- **`go mod tidy` only re-classified `transparency-dev/formats` from `// indirect` to a direct require
  (additive, byte-stable, tidy-idempotent).** It was already in the module graph (entered via
  `tessera/fsck`); the direct `cmd/notecheck` import promotes it. `git diff --exit-code -- go.mod
  go.sum` is clean after a second tidy, `go mod verify` → all verified, directive stays `go 1.24.0`,
  no `toolchain`. go.sum was NOT touched.
- **Oracle gate APPLIES (this IS the external signature oracle) and is satisfied + mutation-proven.**
  Short-circuiting `run` to always-accept makes BOTH negative tests FAIL
  (`RejectsCorruptedBody` + `BadVKey`), so a green-but-wrong oracle cannot ship. Trust-root value
  confirmed from ground truth, not the author: `derive_vkey.py` reproduces the embedded `sb0VKey`
  byte-for-byte (`…+40b74463+…`), and independently decoding the fixture's sig line in Python yields
  signer `sb0.iscc.id/log`, keyhash `40b74463`, 64-byte ed25519 sig — matching the vkey's middle field.
- **The corrupted-body negative exercises the `note.Open failed` path, NOT the strict-reject branch.**
  Flipping a base64 char on the sig line makes `note.Open` itself return "no verifiable signatures"
  (the sig no longer validates → `UnknownVerifierError` → empty `Sigs`), so control never reaches
  `len(n.Sigs)==0 || len(n.UnverifiedSigs)!=0`. The strict-reject branch is covered only by being
  byte-identical to the reference + `verify.go` (correct for v1 single-sig). It first becomes
  independently exercisable at M7 (a cosigner sig line landing in `UnverifiedSigs` with a real verified
  hub sig) — the slice that wires a witness cosig must add a test hitting that exact branch.
- **`run(vkey, in, out)`'s `out io.Writer` param is vestigial — `run` returns the name and `main` does
  the `OK` print to `os.Stdout` directly, so `out` is never written.** `go vet` does not flag unused
  function parameters, so the gate can't catch it; it matches the literal signature `next.md` specified.
  Harmless (the test passes a throwaway buffer), filed `low`. If `run` is ever reused, either drop `out`
  or have it print `OK %s` itself so the signature stops lying.

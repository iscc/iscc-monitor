# Handoff

## 2026-06-21 — Review of: Port `notecheck` into the monitor module as `cmd/notecheck`

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** `advance` ported the fully-independent signature-parity oracle from gitignored
`cauldron/iscc-hub/conformance/notecheck/main.go` into `cmd/notecheck` inside the monitor module,
giving it a real in-repo compile path. The port is faithful (reviewer diffed it line-for-line against
the reference), scoped to exactly 3 files (1 src + 1 test + a tidy-only `go.mod`), and the negative
tests are mutation-proven non-vacuous. All gates green; the trust-root value is confirmed from ground
truth, not the author.

**Verification:**
- [x] `mise run check` — green (build + vet + test, all 11 packages `ok`, incl. `cmd/notecheck`).
- [x] `gofmt -l cmd/notecheck/` prints nothing; `gofmt -l .` clean outside gitignored `cauldron/`.
- [x] `go test -count=1 -run TestNotecheck ./cmd/notecheck` PASS uncached (golden accept +
      corrupted-body reject + bad-vkey exit-2).
- [x] Built binary end-to-end: `/tmp/notecheck --vkey "sb0…E2ZE5" < testdata/live/sb0.iscc.id_checkpoint`
      → `OK sb0.iscc.id/log`, exit 0.
- [x] Corrupted checkpoint rejected: `sed 's/QLdEY/QLdEZ/' … | /tmp/notecheck --vkey …` →
      `note.Open failed: note has no verifiable signatures`, exit 1. Missing `--vkey` → exit 2;
      `--vkey garbage` → exit 2 (`key has 1 parts, expected 3`).
- [x] `go mod tidy && git diff --exit-code -- go.mod go.sum` clean (idempotent after the formats
      direct-require promotion); `grep -c '^go 1.24.0$'` == 1, `grep -c '^toolchain'` == 0;
      `go mod verify` → all modules verified.
- [x] **Oracle gate (APPLIES — this IS the external signature oracle):** `python3 .claude/derive_vkey.py`
      reproduces the embedded `sb0VKey` byte-for-byte
      (`sb0.iscc.id/log+40b74463+AaV+ivnly67hhzQSQfGqCBP3PlOV2NBcmfGyzGdE2ZE5`); independently decoding
      the fixture's sig line in Python yields signer `sb0.iscc.id/log`, keyhash `40b74463`, 64-byte
      ed25519 sig — matching the vkey's middle field. `.claude/.scratch/` removed; tree clean.
- [x] **Mutation-proof:** short-circuiting `run` to always-accept makes BOTH `RejectsCorruptedBody`
      and `BadVKey` FAIL (reverted) — a green-but-wrong oracle cannot ship.
- [x] **Gate integrity:** scanned all unpushed commits — no `//nolint`, `t.Skip`, swallowed-error,
      build-tag, or deleted-assertion patterns. The reference's flat-`main` → `main`+pure-`run` split
      is an idiomatic test seam `next.md` asked for, not a dodge.
- [x] **Scope:** exactly `cmd/notecheck/main.go` (1 non-test src) + `main_test.go` + `go.mod`. Nothing
      from `## Not In Scope` touched — no `.github/workflows/`, no `mise.toml` wiring, no `internal/`
      test, no follower/inclusion/tile-writer/fixture/`derive_vkey.py`/did.json edit.

**Issues found:**
- (low, filed) `run(vkey, in, out io.Writer)`'s `out` param is vestigial — `run` returns the name and
  `main` prints `OK %s` to `os.Stdout` itself, so `out` is never written. Matches the literal signature
  `next.md` specified; harmless and `go vet`-clean (unused params aren't flagged). Left as-is (editing
  it touches the signature + test call sites for no behavior gain); filed `low` for the next time `run`
  is touched.

**Next:** Add `.github/workflows/` with the `notecheck` signature-parity CI job (the sole open `normal`
issue — now UNBLOCKED by this slice). It only needs `go build ./cmd/notecheck` then shells the binary
out against a captured checkpoint: assert `OK <name>` + exit 0, and exit 1 on a corrupted one, plus a
`mise run check` job. CI doing `go build ./...` on a fresh checkout must account for gitignored
`cauldron/` (never reaches CI, but `./cmd/notecheck` sidesteps it). After CI lands: the inclusion
cross-check vs the hub's `IsccLogInclusionProof` (the second half of M2's Verify bar).

**Notes:**
- The corrupted-body negative exercises the `note.Open failed` path, NOT the strict-reject branch
  (`len(n.Sigs)==0 || len(n.UnverifiedSigs)!=0`) — flipping a sig byte makes `note.Open` itself return
  "no verifiable signatures" before that branch is reached. The strict-reject branch is covered only by
  being byte-identical to the reference + `internal/logclient/verify.go` (correct for v1 single-sig); it
  first becomes independently exercisable at M7 when a witness cosig lands in `UnverifiedSigs`. The M7
  cosigner slice must add a test hitting that branch.
- WASM purity is N/A — this is a `cmd/` binary importing `os`/`flag`/`io`, never part of the
  `internal/proof`/`didweb` WASM-shared seam.
- Per protocol, did NOT `go run` the `cauldron/` binaries (module-less, no local compile path); the
  reference was read for line-for-line comparison only.

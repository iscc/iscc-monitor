# Handoff

## 2026-06-20 — Review of: Enforce the CID 1.0 key validity window (pure `DIDKey.ValidAt`)

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** `internal/didweb/validity.go` adds a pure `func (k DIDKey) ValidAt(now time.Time) bool`
that enforces the CID 1.0 wall-clock window — half-open `[ValidFrom, ValidUntil)`, revoked at/after
`Revoked`, each zero field meaning "no constraint" so a zero-value `DIDKey` is always valid. It takes
`now` as an argument (deterministic, never reads the clock), imports only `time` (WASM-pure), and is
backed by a 12-case table test pinning every boundary at exact instants. Scope is exactly what
`next.md` asked: one new source file, one new test file, and a doc-only repoint of the `DIDKey`
comment. Every verification criterion passes and I re-derived the boundary behavior with an
independent throwaway test rather than trusting the author's table.

**Verification:**
- [x] `mise run check` (build + vet + test) — green, exit 0 on a freshly cleared test cache.
- [x] `gofmt -l internal/didweb` and `gofmt -l .` — both empty.
- [x] `go test -run TestValidAt ./internal/didweb` — PASS (all 12 subtests + `TestValidAtZeroKeyNow`).
- [x] Zero-value `DIDKey{}.ValidAt(time.Now())` → `true` (matches the live "currently valid" fixtures).
- [x] `ValidUntil=2020-01-01` @2026-06-20 → `false`; `ValidFrom=2030-01-01` → `false`;
  `Revoked=2020-01-01` → `false`. Re-confirmed with an independent reviewer test.
- [x] Half-open boundary: `ValidUntil=T` → `ValidAt(T)==false`, `ValidAt(T-1ns)==true`;
  `ValidFrom=T` → `ValidAt(T)==true` (inclusive lower bound); `Revoked=T` → `ValidAt(T)==false`.
- [x] `GOOS=js GOARCH=wasm go build ./internal/didweb` exits 0 (`validity.go` imports only `time`).
- [x] Trust-root/oracle gate — N/A by construction (no crypto/Merkle/`SQLiteFetcher`/`internal/proof`
  touched), but verified anyway: `derive_vkey.py` still prints both golden vectors byte-exact
  (`sb0…+40b74463+…`, `sb1…+22b08f3e+…`); `.claude/.scratch/` cleaned up afterward.
- [x] Gate integrity — across all unpushed commits, no `//nolint` / `t.Skip` / build-tag exclusion /
  swallowed error / loosened gate / deleted assertion in source (matches are prose-only in context md).
- [x] sb1 fixture / `derive_vkey.py` `HUBS` correctly left at the pre-rotation `22b08f3e` (out of
  scope this step; pinned to the `hub_keys` step).

**Issues found:**
- Filed `[review]` `normal`: `parseTime` fails *open* on a malformed (non-empty, unparseable)
  validity timestamp — it maps to the zero time, which `ValidAt` reads as "no constraint". Correctly
  deferred from this pure-predicate step (the advance/define-next docs flagged it); recorded in
  `issues.md` to be decided at the `hub_keys`/fixture step (a garbled `revoked` arguably should
  degrade to not-`verified`, not be treated as unconstrained).

**Minor fix applied (review):** aligned the now-stale `verificationMethod` doc comment in `resolve.go`
("the follower may later enforce") to point at `DIDKey.ValidAt`, matching the authoritative `DIDKey`
comment. Doc-only; gofmt/build/vet re-confirmed clean.

**Next:** Wire the SQLite `hub_keys` cache + the follower's checkpoint-acceptance path
(`ResolveVerifierKey` → `VerifyCheckpoint`), calling `DIDKey.ValidAt(observedAt)`: an in-window
matching key → `verified`; an out-of-window matching key → **not-`verified`** (rotation/revocation),
*distinct* from `ErrUnverified → unverified` and `ErrUnresolvable → unresolvable`. That step also
refreshes the sb1 did.json fixture + `derive_vkey.py` `HUBS` to the current key (`069d0f14`) and
should resolve the `parseTime` fail-open issue above. The three-trigger RFC-6962 consistency check
(fork/shrink/equivocation via `transparency-dev/merkle`) is the step after.

**Notes:**
- M1 is only *partially* met: the verification primitives (did:web chain, signed-note checkpoint
  verify, now the validity predicate) are pure and golden-tested, but there is still **no follower, no
  SQLite store, no binary entrypoint, no consistency check, no coverage/metrics**. Loop stays
  CONTINUE; nowhere near v1 DONE.
- **No CI / `notecheck` job exists yet** (`.github/workflows/` still absent). The signature-parity
  oracle these steps feed is the natural CI gate; flag for whoever adds the workflow. CI must also
  exclude the gitignored `cauldron/` trees (a fresh `go build ./...` over them needs stub `go.mod`s).
- `net/url` appears in the *package* import closure (from `url.go`), not from `validity.go`; per
  learnings it does not pull `net`/`net/http`, and the WASM build is green — not a purity regression.
- Pushed to `origin/develop` on PASS (remote configured). Branch: `develop`; never push `main`.

---
status: accepted
---

# ISCC en/decoding reuses the iscc-lib Go binding; ISCC-IDv1 is an interim in-repo port

Extends the "reuse, do not reimplement" principle of ADR-0003 (Go stack) from the transparency
stack to the **ISCC codec**. The monitor must not own a second, hand-rolled copy of security- and
conformance-critical ISCC en/decoding when the ISCC Foundation already ships a maintained,
conformance-tested Go implementation.

## Decision

**Adopt `github.com/iscc/iscc-lib/packages/go` as the canonical Go implementation of ISCC
en/decoding** for everything it covers. It is a **pure Go** reimplementation (not an FFI wrapper
around the Rust core, unlike the Python/Node/Java bindings), builds with `CGO_ENABLED=0`, and is
conformance-tested against the `iscc-core` reference vectors. Pinned at **v0.5.0**. The monitor
**consumes** it for generic ISO 24138 / Version-0 codec work — decode / decompose / validate any
ISCC-CODE or ISCC-UNIT it encounters (e.g. ISCC-CODEs embedded in declaration notes on the
single-record and certificate surfaces). The monitor does **not** generate codes (it follows hub
logs, it does not fingerprint media), so the `Gen*CodeV0` surface is unused.

**Bump the locked toolchain from Go 1.24 to Go 1.26** (ADR-0003 / `target.md` stack). `iscc-lib`'s
`packages/go/go.mod` declares `go 1.26.1`, so consuming it requires a toolchain `>= 1.26.1`. The
bump (`mise.toml`, `go.mod`, CI, devcontainer) plus the dependency add is the adoption increment,
filed in `issues.md`.

**ISCC-IDv1 stays an interim in-repo port** (`internal/index/iscc.go`), **not** delegated to
iscc-lib — for two verified reasons:

1. **iscc-lib cannot decode ISCC-IDv1 today.** `packages/go/codec.go`'s `decodeHeader` hard-rejects
   any `Version > 0` (`fmt.Errorf("iscc: invalid Version: %d", versionVal)`). An ISCC-IDv1 has
   `Version = 1`, so `IsccDecode` rejects every real ISCC-IDv1 with *"invalid Version: 1"*. There is
   no ISCC-ID-specific function in the Go API.
2. **ISCC-IDv1 is not part of ISO 24138** and is still evolving; its authoritative codec lives in
   `iscc-core`'s `iscc_id.py` (mirrored hub-side in `iscc-hub`), which `internal/index/iscc.go`
   already documents itself as a port of.

`internal/index.Decode` is therefore reframed from "the codec" to "the **interim** Go port until
iscc-lib ships ISCC-IDv1". Its package doc and golden/mutation tests are unchanged; it remains a
pure, WASM-shareable leaf, fail-closed on all four header nibbles.

## Migration trigger

When the iscc-lib Go binding ships ISCC-IDv1 (`MainType = ID`, `Version = 1`) encode/decode at
parity with `iscc-core`'s `iscc_id.py`, **replace `internal/index.Decode` with the iscc-lib call
and delete the port.** The trigger is tracked two ways:

- **Upstream issue [iscc/iscc-lib#43](https://github.com/iscc/iscc-lib/issues/43)** requests exactly
  this support, with the bit layout and a conformance vector.
- **An executable tripwire** in `internal/index`: a parity test that imports iscc-lib and asserts it
  *currently rejects* a known ISCC-IDv1 (`Version = 1`). When upstream lands support, that assertion
  flips **red** and tells the next loop iteration to migrate. This is also the one real, honest use
  that keeps the dependency live (the monitor has no other generic-codec call yet) and stays inside
  the project's oracle-gate testing philosophy.

## Consequences

- **One source of truth for the ISO 24138 codec.** No second hand-rolled base32 / varnibble / header
  codec to drift from the standard; conformance is iscc-lib's responsibility, verified against
  `iscc-core`.
- **Toolchain moves to Go 1.26.** A two-minor-version jump from 1.24; all current deps are pure-Go
  and `CGO_ENABLED=0`-clean, as are iscc-lib's (`zeebo/blake3`, `golang.org/x/text`,
  `klauspost/cpuid`). The static-binary / cross-compile guarantee is preserved.
- **`go 1.26.1` patch-pin caveat.** iscc-lib's `go` directive is patch-level (`1.26.1`), unusually
  strict for a library; it forces consumers to exactly `>= 1.26.1`. Acceptable given the bump, but
  worth raising upstream to relax to `go 1.26`.
- **Maturity risk pinned.** iscc-lib is marked experimental in the ISCC ecosystem; the dependency is
  version-pinned (v0.5.0) and the conformance/golden gates catch a regression on upgrade.
- **Coverage gate unchanged.** `target.md` already declines iscc-lib's 100%-coverage gate in favour
  of this project's seam/oracle testing; adopting the library does not import its coverage policy.
- **ADR-0003 stack note amended:** the locked stack is Go 1.26 (was 1.24) and the reuse list now
  includes the ISCC codec, not only the transparency stack.

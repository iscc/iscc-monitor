## 2026-06-23 — Dossier §1 stops claiming "Key resolved from did:web" on the `unresolvable` path

**Done:** On the `unresolvable` overlay path the dossier §1 Identity no longer asserts "Key resolved
from did:web:<domain>" (which contradicted the same page's "signing key is unresolved" caution). Added
a pure view-model flag `KeyUnresolved` (set `status == "unresolvable"`, one source of truth alongside
`ShowCaution`) and gated the §1 `section-value` text to render neutral "Key source: did:web:<domain>"
when unresolved, keeping the mockup's "Key resolved from did:web:<domain>" copy verbatim for every other
status. No new store read; §1 stays static-derived off the already-resolved status string.

**Files changed:**
- `internal/dossier/handler.go`: added `KeyUnresolved bool` to `dossierData` (+ doc comment paragraph
  explaining the §1-honesty gate); set it `status == "unresolvable"` in `buildData` at the line that
  also sets `ShowCaution`.
- `internal/dossier/dossier.html`: gated the §1 `section-value` (line 533) with
  `{{if .KeyUnresolved}}Key source: …{{else}}Key resolved from …{{end}}`, keeping
  `<span class="mono">did:web:{{.Domain}}</span>` in BOTH arms so the domain always shows.
- `internal/dossier/handler_test.go`: added `TestDossierUnresolvedKeyWordingHonesty` driving a
  store-verified hub with `fakeStatusSource{id:"unresolvable"}` (asserts NO "Key resolved from", DOES
  contain "Key source:", still shows `did:web:sb0.iscc.id`) + a sibling verified-path case (nil source)
  asserting the mockup's `Key resolved from <span class="mono">did:web:sb0.iscc.id</span>` copy is
  unchanged and NOT over-gated to "Key source:".

**Verification:** `mise run check` → green (build + vet + test across all 30 packages, `gofmt -l .`
empty). Per-criterion:
- [x] `mise run check` green; gofmt clean.
- [x] `go test -count=1 -run TestDossier ./internal/dossier` PASS (existing suite unaffected).
- [x] NEW `TestDossierUnresolvedKeyWordingHonesty` PASS.
- [x] Mutation 1 (revert §1 gate → template always "Key resolved from"): unresolvable assert FAILS;
  reverted → PASS. (Re-runnable: restore the gate.)
- [x] Mutation 2 (over-gate: `KeyUnresolved: true` in buildData): verified-sibling assert FAILS;
  reverted → PASS. Guards both directions.
- Oracle/conformance gate: **N/A** — pure SSR template + view-model flag off the already-resolved
  status; no signature, RFC-6962, Merkle, did:web resolution, fsck, or proof path touched; no new
  store read; `go.mod`/`go.sum` untouched.

**Next:** The dossier §1 honesty `normal` is now code-closed. Remaining open `normal`s are
design-first / design-blocked (the WASM cross-origin signature half; the realm-index per-checkpoint
Anchor honesty) and the lone `critical` is human-blocked on the M-UI exit sign-off. Steer `define-next`
either to a design-note pass on one of the design-blocked `normal`s, or to the bookkeeping prune of the
now-stale M-API issue entries (resolved by ancestor commit `53ee328`) — but that is an
`update-state`/`review` job, not advance work. If no non-blocked code-closable work remains, the loop
may be at the human-blocked stall (memory note "loop-stalls-on-human-blocked-done").

**Notes:**
- Scope held to exactly 3 files (2 prod + 1 test), all in next.md scope. The `unverified` §1 wording is
  deliberately left as "Key resolved from" — `unverified` means the did:web doc DID fetch/parse (a key
  resolved) and only the signature matched no listed key, so "resolved" stays honest; the gate is
  `unresolvable`-ONLY per next.md "Not In Scope".
- The `KeyUnresolved` flag is set at the same site as `ShowCaution` (one source of truth off the
  already-resolved `status`), so the §1 verb and the caution copy can never drift out of sync.
- `learnings/dossier.md` lines 34-40 record this as an open `normal` (Codex P2, reviewer-confirmed) —
  review should mark it CLOSED and update that bullet (advance does not write learnings).
- No remote push state changed; the only working-tree changes are the three committed files + this
  handoff.

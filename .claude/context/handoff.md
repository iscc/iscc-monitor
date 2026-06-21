## 2026-06-21 — Review of: Hub-List parser + `(realm, hub_id) → domain` resolver in `internal/registry`

**Verdict:** PASS_WITH_NOTES
**Loop:** CONTINUE

**Summary:** The advance adds a pure, golden-tested iscc-hub Hub-List parser (`ParseHubList`) +
`Hub`/`HubList` types + a `Resolve(hubID) → (domain, ok)` lookup to `internal/registry`, purely
additive (the domains-only `Parse`/`Entry`/`realm.txt` path is untouched, as scoped). The codec, types,
golden fixture, and tests are correct; all gates are green and I mutation-proved four guards
non-vacuous. But Codex surfaced — and I confirmed — two **fail-open** gaps in URL/field handling: a
scheme'd path-bearing url is accepted (contradicting the explicit next.md/docstring/handoff claim that
path-bearing is rejected), and a missing `hub_id` silently becomes slot 0. Neither blocks progress
(additive leaf, live wiring deferred, fixture clean), so this lands with both filed as `normal`
follow-ups.

**Verification:**
- [x] `mise run check` green — build + vet + all 20 packages PASS.
- [x] `gofmt -l .` empty.
- [x] `go test -count=1 ./internal/registry` passes uncached (3 existing `Parse` tests + 7 new Hub-List
  golden/failure tests).
- [x] `Resolve(0) == ("sb0.iscc.id", true)` and `Resolve(1) == ("sb1.amlet.id", true)` (hard-coded
  ground truth, real testnet hubs).
- [x] `Resolve(9)` unknown slot → `("", false)`; inactive hub resolves; `hub_id 4095` resolves.
- [x] Malformed docs fail closed (invalid YAML / `hub_id 4096` / duplicate `hub_id` / empty url /
  scheme-less bare-path url) with non-nil error + nil list.
- [~] "Path-bearing url rejected" — **partially false.** Only the *scheme-less* `host/path` is rejected
  (via the no-host branch). A *scheme'd* `https://host/path` is ACCEPTED, path silently dropped
  (reviewer-confirmed). next.md/docstring/handoff claim full path-bearing rejection; the test's
  `"path-bearing url"` case masks the gap. Filed `normal`.
- [x] `GOOS=js GOARCH=wasm go build ./internal/registry` succeeds; closure has `net/url`+`net/netip`,
  no `net`/`net/http`/`database/sql` (`os` only transitively via `fmt`, the documented nuance).
- [x] `go mod tidy` idempotent (second run a no-op; `go.mod`/`go.sum` byte-identical); `yaml.v3` is a
  direct require.
- [x] Mutation proof (reviewer-run): weakening the over-range guard, disabling duplicate detection,
  making `Resolve` ignore `hubID`, and making `hubDomain` skip host-stripping each make a targeted test
  FAIL. The tests are tied to hard-coded ground truth, not the symbol. Restored clean.
- [x] Quality-gate integrity: scanned all 3 unpushed commits — no `nolint`/`t.Skip`/build-exclude/
  swallowed-error/deleted-test/loosened-gate.

**Oracle/conformance gate:** correctly N/A — pure registry leaf, no signature-verify / RFC-6962 /
Merkle / proof code touched. `go.mod`/`go.sum` changed only to promote the pure-Go `yaml.v3` to a
direct require (+ its test-only transitive checksums). Gate re-engages at the proof-bundle assembler.

**Issues found:**
1. (`normal`, filed) `hubDomain` accepts a scheme'd path-bearing url (`https://host/path` → `host`,
   path dropped) — a fail-open against the explicitly-claimed contract; the guarding test only covers
   the scheme-less case so it gives false confidence.
2. (`normal`, filed) A Hub-List entry omitting `hub_id` silently becomes slot 0 (YAML zero-value
   fail-open) — beyond next.md's named rejects, but the same fail-open class.

**Codex second opinion:** Two `[P2]` findings, **both reviewer-confirmed real** and filed:
- P2 #1 (path-bearing url accepted, registry.go:169-172) — confirmed: `hubDomain("https://sb0.iscc.id/log")`
  returns `("sb0.iscc.id", nil)`. Matches issue #1 above.
- P2 #2 (missing `hub_id` → slot 0, registry.go:80) — confirmed: an entry with only `url`/`active`
  parses with `HubID == 0` and `Resolve(0)` returns it. Matches issue #2 above.
No dismissals. Both are correctness gaps in the new hub-resolution path, not on the signature/Merkle
trust root, so no oracle conflict.

**Next:** Two viable threads for define-next to weigh:
- **(a) Harden the two fail-open gaps** (issues above) — a one-line `u.Path` check + a scheme'd-path
  test, and presence-tracking for `hub_id`. Small, well-scoped, hardens a trust-root-adjacent resolver
  before it is wired. Good "close the loop on this leaf" slice.
- **(b) Sequence the human-filed ADR-0011 work** (adopt the iscc-lib Go codec + bump toolchain to Go
  1.26, `normal`, in `issues.md`) — flagged as foundational and "before more M-UI feature work." It
  needs a Go 1.26 toolchain present (local is 1.24); confirm mise can provision it first.
- Then the deferred `/inclusion/{iscc_id}` HTML certificate page + proof-bundle assembler (re-engages
  the oracle gate). I'd do (a) first (cheap, and it's the leaf just landed), then weigh (b) vs the
  certificate page.

**Notes:**
- **Pre-existing uncommitted working-tree edits remain** (not from this advance): `.claude/context/
  target.md`, `.claude/plans/cosmic-baking-octopus.md`, `.claude/prd/0001-iscc-monitor-v1.md`,
  `CLAUDE.md`, and the new untracked `.claude/adr/0011-iscc-lib-codec-dependency.md`, plus a human-filed
  `normal` ADR-0011 issue already in `issues.md`. These look like a human/other-session planning pass
  (ADR-0011 = adopt iscc-lib + Go 1.26). I did **not** touch or commit them — they are outside this
  increment and `update-state`/`define-next` should reconcile them. Worth a human glance: ADR-0011 may
  change the `internal/index` decoder strategy (migrate to iscc-lib once iscc-lib#43 lands), which would
  retire the hand-rolled port.
- **`hubDomain` only succeeds for URLs with a scheme** (or `//host`): a scheme-less bare host fails
  "no host". The ADR-0010 schema uses full `https://host` urls, so the fixture is consistent — but this
  means the parser is strict-scheme, worth keeping in mind when the live Hub-List is authored.
- **`KnownFields(false)`** is the intended parse-and-ignore contract (tolerates `pubkey` + future
  keys); don't flip it without re-deciding `pubkey`.
- The three open `low` test-hardening/architecture issues remain loop-skipped and untouched.

# Handoff

## 2026-06-20 — Review of: did:web HTTP resolver at the outbound-fetch seam (base URL → verifier key, `unresolvable` on failure)

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** `advance` added `internal/logclient/didresolve.go` — a 1-method `Fetcher` seam, an
unexported `httpFetcher` (via `NewHTTPFetcher`) mirroring Tessera's fetch shape, an exported
`ErrUnresolvable` sentinel, and `ResolveVerifierKey(ctx, fetcher, baseURL)` wiring
`origin → did:web:<host%3Aport> → DocumentURL → Fetch → ParseDIDDocument → VerifierKey`, collapsing
every failure to `ErrUnresolvable`. Purely additive; `internal/didweb` untouched and still WASM-pure.
Wiring, error mapping, and golden-vector parity all verified independently. Clean, in-scope, no gate
games.

**Verification:**
- [x] `mise run check` (build + vet + test) — green, exit 0 (re-run, not just from handoff).
- [x] `gofmt -l /workspace/iscc-monitor` — prints nothing.
- [x] `go test -run TestResolveVerifierKey ./internal/logclient` — PASS (all subtests verbose-run).
- [x] sb0 fixture → `sb0.iscc.id/log+40b74463+AaV+ivnly67hhzQSQfGqCBP3PlOV2NBcmfGyzGdE2ZE5` for
  `https://sb0.iscc.id`; fetched URL `https://sb0.iscc.id/.well-known/did.json` (full string asserted).
- [x] sb1 fixture → `sb1.amlet.id/log+22b08f3e+ATo2ruguSdJGh11PS76osrQf6OZKrufzzwH/HMwE3a8/` for
  `https://sb1.amlet.id` (full string asserted).
- [x] Fetch error / 404 (`os.ErrNotExist`) / malformed JSON / no-assertionMethod / empty base URL each
  satisfy `errors.Is(err, ErrUnresolvable)` (asserted with `errors.Is`).
- [x] `GOOS=js GOARCH=wasm go build ./internal/didweb` succeeds; `go list -deps ./internal/didweb` has
  no `net`/`net/http`/`database/sql` (purity intact — `net/http` isolated to `logclient`).
- [x] **Trust-root oracle gate:** re-ran `python3 .claude/derive_vkey.py` — both golden vectors print
  byte-exact (`sb0…+40b74463…`, `sb1…+22b08f3e…`), matching the test assertions. Scratch dir cleaned.
- [x] `internal/logclient/testdata/{sb0,sb1}` fixtures are byte-identical to the `internal/didweb`
  source fixtures (`diff` clean) — no drift in the trust-root inputs.
- [x] Gate-integrity scan of unpushed range (`@{upstream}..HEAD`): no `//nolint`, `t.Skip`, build
  tags, deleted assertions, or loosened gates. The single `_ = resp.Body.Close()` is the idiomatic
  deferred close, not a swallowed gate error.
- [x] Scope: exactly 1 non-test source file + 1 test file + 2 fixtures; nothing from `## Not In Scope`
  leaked (no signature verification, no SQLite, no follower loop, no external deps).

**Issues found:** (none)

**Next:** Wire Ed25519 signed-note signature verification: parse a hub-signed checkpoint and verify it
against the key `ResolveVerifierKey` returns, mapping a signature matching no listed key to status
`unverified` (distinct from `unresolvable`). This needs an actual checkpoint fixture and a
`note.Verifier`/signed-note parser, and pairs with the `hub_keys` SQLite cache + CID 1.0
validity-window enforcement (consuming `DIDKey.ValidFrom`/`ValidUntil`/`Revoked`, currently
parsed-but-unenforced). Capturing a real sb0/sb1 checkpoint into `testdata/live/` is a prerequisite.

**Notes:**
- **No CI configured** (`.github/workflows/` absent). The `notecheck` external-oracle signature-parity
  job does not exist yet — correct at this stage, since no end-to-end signature-verification code is in
  tree. The trust-root gate at this point is `derive_vkey.py` parity, re-run and green. Flag for
  whoever wires the GitHub workflow (and: a fresh `go build ./...` over the gitignored `cauldron/`
  trees breaks — CI must exclude them; see learnings).
- **Host re-derivation** uses `TrimSuffix(origin(baseURL), "/log")` rather than a second host parser —
  reuse, not duplication, matching `next.md`'s "one host-extraction helper" intent. Sound.
- **did:web colon percent-encoding** (`strings.Replace(host, ":", "%3A", 1)`) is required only for the
  `httptest` random-port path (live hubs have no port) and matches the existing
  `did:web:example.com%3A3000` golden. Documented inline. The `httptest` golden correctly asserts only
  the `<addr>/log` origin prefix (the key embeds the fixture's origin, not the fetch host).
- Validity-window fields (`ValidFrom`/`ValidUntil`/`Revoked`) remain parsed-but-unenforced — correctly
  out of scope here; enforcement belongs to the store/follower step.
- Branch `develop`, remote `origin` configured; pushing on PASS.

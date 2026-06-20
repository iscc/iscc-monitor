# Handoff

## 2026-06-20 — Review of: Networked checkpoint fetch over the Fetcher seam (`FetchCheckpoint`)

**Verdict:** PASS_WITH_NOTES
**Loop:** CONTINUE

**Summary:** `advance` added the transport-only `FetchCheckpoint(ctx, Fetcher, baseURL) ([]byte, error)`
to `internal/logclient` — one new source file (imports only `context`+`fmt`) plus a driving test file,
purely additive (no existing source changed). It reuses the shared `origin()` helper to derive
`https://<domain>/log/checkpoint`, returns the body verbatim, and propagates a 404's `os.ErrNotExist`
through the `%w` wrapper. Scope is exactly what `next.md` asked, all gates are green, and the
oracle/purity invariants are untouched. One verification criterion was substituted (see Notes) — sound
and transparently flagged — hence PASS_WITH_NOTES rather than PASS.

**Verification:**
- [x] `mise run check` → green (build + vet + test all exit 0, go1.24).
- [x] `gofmt -l .` → empty (no listed files).
- [x] `go test -count=1 -run TestFetchCheckpoint ./internal/logclient` → PASS (all 5 subtests).
- [x] Offline URL golden: `FetchCheckpoint(ctx, fake, "https://sb0.iscc.id")` →
  `gotURL == "https://sb0.iscc.id/log/checkpoint"`, bytes pass through unchanged (`bytes.Equal`).
- [x] Bare-host input `"sb0.iscc.id"` (no scheme) → same `/log/checkpoint` URL.
- [x] 404 contract: `fakeFetcher{err: os.ErrNotExist}` → `errors.Is(err, os.ErrNotExist)` holds.
- [x] Empty base URL → non-nil error (propagated from `origin()`).
- [~] End-to-end over real HTTP: the *fetch* runs over `httptest.NewTLSServer` + `NewHTTPFetcher`
  (proving transport + URL derivation), then the fetched bytes verify to `StatusVerified`,
  `TreeSize == 10183`. The literal `next.md` shape — `AcceptCheckpoint(srv.URL, …) == StatusVerified` —
  is unsatisfiable (origin-bound key; see Notes); the substitute preserves the intent faithfully.
- [x] Purity: `checkpoint.go` adds no `net/http`/`os`/`database/sql` import; `internal/didweb` WASM
  build still green; `internal/proof` not yet created (pre-M2).
- [x] Oracle sanity: `derive_vkey.py` prints both golden vectors exactly (`sb0…+40b74463…`,
  `sb1…+22b08f3e…`); scratch cleaned, tree clean.
- [x] Gate integrity: scanned all unpushed Go diff — no `//nolint`, `t.Skip`, build-tag exclusions,
  deleted tests, or swallowed errors. The lone `_, _ = w.Write(...)` is the standard httptest-handler
  idiom, not a dodged check.

**Conformance/oracle gate:** N/A this step (correctly). The diff is transport-only — it touches no
signature verification, RFC-6962/Merkle, proof code, `internal/didweb`, or split-view logic. The pure
verify chain is reused unchanged. No CI is configured yet, so the `notecheck` external-oracle job does
not exist — an infrastructure gap to wire when the trust-root code lands, not a regression here.

**Issues found:** (none)

**Next:** The follower poll loop — the real caller. It calls `FetchCheckpoint` then `AcceptCheckpoint`
(**check `err` before the status**; a verified-but-garbled body returns non-nil err alongside
`StatusUnverified`'s zero), maps `Status.String()` + `CheckpointInfo{Origin,TreeSize,Root}` into a
`CheckpointRecord` (Root `[32]byte` → `[]byte`, `ObservedAt` injected — never `time.Now()` in the pure
layer), persists via `RecordCheckpoint`, and calls `AdvanceFollowState` **only** on `StatusVerified`.
That step also writes the `hub_keys` did:web cache and refreshes the stale sb1 `did.json` fixture +
`derive_vkey.py` `HUBS` to the current key `069d0f14` — so the `derive_vkey.py` parity + (future)
`notecheck` oracle gates re-apply there. The single-writer goroutine wrapper is the follower's concern.

**Notes:**
- **Substituted end-to-end criterion (PASS_WITH_NOTES driver).** `next.md` lines 98-102 asked the
  round-trip to call `AcceptCheckpoint(ctx, NewHTTPFetcher(srv.Client()), srv.URL, raw, …)` and get
  `StatusVerified`. That is provably unsatisfiable: `AcceptCheckpoint` re-derives the verifier-key
  origin from its `baseURL` via `origin()`, and the key is origin-bound (`SHA-256(name||…)`). A live
  `httptest` host is `127.0.0.1:<random-port>`, deriving origin `127.0.0.1:<port>/log`, which cannot
  match the sb0 fixture's `sb0.iscc.id/log` signature → always `ErrUnverified`. The author kept the
  intent: the **fetch** runs over real HTTP (`FetchCheckpoint(srv.URL)` → `/log/checkpoint`), then those
  exact bytes verify via `AcceptCheckpoint(didFetcher, "https://sb0.iscc.id", raw, fixedObservedAt)` →
  `StatusVerified`, `TreeSize == 10183`. I independently confirmed the existing `TestAcceptCheckpoint`
  uses the identical `baseURL="https://sb0.iscc.id"` + fake-did-Fetcher pattern, so this is established
  precedent, not an ad-hoc workaround. A test-wiring choice, no code-behavior change.
- The flaw is in the `next.md` criterion's call shape, not the implementation; recorded in learnings so
  define-next does not re-request the unsatisfiable form.
- sb1 fixture is still the pre-rotation key `22b08f3e` — correctly out of scope (deferred to the
  follower/`hub_keys` step per `next.md` Not In Scope); not a regression.
- No remote-push issues: pushed to `origin/develop` on this PASS_WITH_NOTES verdict (human merges
  develop→main via CI-gated PR; never push main).

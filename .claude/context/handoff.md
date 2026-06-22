## 2026-06-22 — Review of: Render certificate timestamps in UTC (`.UTC().Format`) to green the gate on non-UTC hosts

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** The advance normalizes all four `.Format(time.RFC3339)` call sites in
`internal/certificate/handler.go` to `.UTC().Format(time.RFC3339)` (§4 `SigningKeyRevoked` + bundle key
`Revoked` + §5 `BTCConfirmedAt` + Comparison-Anchor `CoverageSince`) and adds one host-independent
regression test. Scope is exactly one production file (within the ≤3 budget) plus its test; the change is
minimal, correct, and mutation-proven non-vacuous. `mise run check` is now green on UTC, America/New_York,
and Asia/Kolkata — the gate that was environment-dependently RED on the parent is cleared.

**Verification:**
- [x] `mise run check` green — all 27 packages `ok` (host TZ = UTC).
- [x] `TZ=America/New_York go test -count=1 ./internal/certificate` — passes (parent `HEAD~1` handler.go
  reproduced 2 FAILs under this TZ; confirmed in a throwaway checkout, then restored).
- [x] `TZ=UTC go test -count=1 ./internal/certificate` — passes (no regression on UTC hosts).
- [x] `TZ=Asia/Kolkata` (half-hour offset) — passes (extra robustness check).
- [x] New test FAILS on a reverted `.UTC()` — mutation-verified BOTH pinned sites: reverting `:1013`
  (`BTCConfirmedAt`) OR `:1040` (`CoverageSince`) makes `TestCertificateRendersTimestampsInUTC` FAIL on a
  UTC host (proving it is non-vacuous, unlike the two pre-existing TZ-sensitive tests). Handler restored
  byte-clean after each mutation.
- [x] `gofmt -l` empty on the two touched files AND across the whole tree; `go vet ./internal/certificate`
  clean.
- [x] All four `.Format(time.RFC3339)` sites in the handler are `.UTC()`-normalized — grep confirms no
  bare `.Format(time.RFC3339)` remains (and no other `.Format(` call sites exist).
- [x] Scope discipline — 1 prod file + 1 test file; nothing in `## Not In Scope` touched (no SIGTERM,
  Dockerfile, version-stamp, deploy realm doc, format-humanizing, or other SSR surfaces).
- [x] Oracle gate **N/A** — pure timestamp-rendering change; no signature / RFC-6962 / Merkle / did:web /
  proof / `go.mod` / `go.sum` / `schema.sql` path touched (`git diff --name-only` = the 2 cert files).
- [x] Gate-integrity scan of all unpushed commits — no `//nolint` / `t.Skip` / build-tag exclusion /
  swallowed error / deleted assertion in any added line.

**Issues found:** (none) — no new defects. Pruned two resolved entries:
- Deleted the resolved *"Certificate renders … timestamps in LOCAL time"* `normal` issue (this slice fixed
  it; verified end-to-end + mutation-proven).
- Pruned the *"Pages custom domain not bound by artifact CNAME"* `normal` issue — it was marked RESOLVED
  end-to-end / live-verified in-place by `update-state` with an explicit "prune it" note (live apex serves
  200s; deploy branch policy fixed). Residual doc-note ask is subsumed by the open README + M-Deploy work.

**Codex second opinion:** Clean — "The change consistently normalizes the certificate timestamp render
points to UTC and adds a regression test that exercises non-UTC local time behavior. I did not identify
any introduced correctness issues." No findings to triage; it independently ran the same TZ tests
(America/New_York + Asia/Kolkata) and saw them pass, matching my verification.

**Visual check:** n/a — no visual SSR surface change. The diff only changes a rendered timestamp's zone
(`…+01:00` → `…Z`); identical chrome/layout, and the data values were already correct. No `.dc.html`
mockup comparison is warranted for a zone-of-an-RFC-3339-string fix.

**Next:** The gate is green on any host TZ, so the queued M-Deploy `critical` work can now be verified
cleanly. Strongest first slice: **trap SIGTERM in `run()`** — `signal.NotifyContext(context.Background(),
os.Interrupt, syscall.SIGTERM)` in `cmd/iscc-monitor/main.go:133`. It is a 1-file change closing a
`critical` ops issue (containers/orchestrators stop via SIGTERM, not SIGINT) and is the cleanest unblock
before the Dockerfile/GHCR step so `docker stop` drains the store cleanly. The multi-stage Dockerfile +
GHCR publish workflow (with the `-ldflags` git-SHA build stamp on `/healthz` or `GET /version`) is the
larger `critical` follow-on; `deploy/realm-testnet.txt` + the public `README.md` are cheap independent
slices. M-Deploy is 0/Verify with 6 `critical` ops issues open.

**Notes:**
- The fix is consistent with the federation-wide convention — dashboard/dossier/log-browser already render
  RFC-3339 in UTC `Z`; this was the cert handler diverging. Recorded the rule in `learnings/certificate.md`
  as the forward-looking trap ("every rendered timestamp MUST be `.UTC().Format(...)`; the store's
  `time.Unix` read-back re-wraps in `time.Local`, so a bare `.Format` leaks the host offset and only CI's
  UTC runner stays green"). Net-reduced that file (collapsed three settled blocks) but it sits at ~179
  lines — at the rotation-budget ceiling for an 8-clause surface; next cert review should keep collapsing
  settled mechanics rather than letting it grow.
- Two of the four fixed sites (`:653`/`:962`, both `key.Revoked`) have NO fixture exercising the
  §4/bundle revoked path, so reverting their `.UTC()` fails no test — this is pre-existing untested-debt,
  not new (a future §4-revoked fixture would pin them). `next.md` deliberately directed fixing all four for
  uniform locale-independence; correct call.
- The test forces `time.Local` to a fixed UTC+1 zone (scoped, `t.Cleanup`-restored) rather than seeding a
  `FixedZone` instant, because the store's `time.Unix` read-back strips a seeded location — the only way to
  make the offset non-UTC on every host. The cert suite has no `t.Parallel`, so the per-process
  `time.Local` swap is safe. Reasonable, well-documented deviation from the literal `next.md` sketch.
- Pushed to `origin/develop` on PASS.

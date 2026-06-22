## 2026-06-22 — Thread config-driven instance identity into the certificate masthead

**Done:** Made the certificate-of-inclusion masthead (`/inclusion/{iscc_id}`) render this deployment's
configured `dashboard.Identity` (instance domain + operator/realm line) instead of the static `monitor
instance` placeholder, completing the third (and final SSR-masthead) leg of the identity arc so the
dashboard / dossier / certificate mastheads are now byte-identical. An unconfigured binary (a
zero-value `Identity`) falls back to today's exact static copy.

**Files changed:**
- `internal/certificate/handler.go`: added a 4th `id dashboard.Identity` arg to `Handler`; ported the
  `instanceFallback`/`operatorFallback` consts + `resolveIdentity` helper VERBATIM from the dossier
  (with the "MUST stay byte-identical" comment); added `Instance`/`Operator` fields to `certData`;
  resolve `id` once at the top of `Handler` and set the two fields on the value `buildData` RETURNS, on
  BOTH the HTML branch and the `.bundle` branch (so every honest 200 carries the masthead regardless of
  `buildData`'s `certData{}` literal early returns). Renamed the `.bundle` branch's local `id` to
  `bareID` to avoid shadowing the new param.
- `internal/certificate/cert.html`: replaced the static `<span class="chrome-instance">monitor
  instance</span>` (line 391) with the two-line `chrome-identity` block ported byte-for-byte from
  dashboard/dossier (`{{.Instance}}`/`{{.Operator}}`); added the `.chrome-identity` + `.chrome-operator`
  CSS rules (rule bodies now byte-identical across all three mastheads). The certificate footer phrase
  "issued by this monitor instance" (line 419) is untouched (pre-existing, unrelated).
- `cmd/iscc-monitor/main.go`: forwarded the already-constructed `id` into the certificate mount
  (`certificate.Handler(hubList, st, m, id)`); `buildMux`'s own signature is unchanged.
- `internal/certificate/handler_test.go` (test): added `TestCertificateRendersInstanceIdentity`
  (populated + zero-value paths, exercised on a cannot-certify/malformed id so the masthead is tested on
  the honest-200 path); added the `dashboard` import; updated 32 existing `Handler(...)` calls for the
  new 4th arg.
- `internal/certificate/bundle_test.go` (test): added the `dashboard` import; updated 5 `Handler(...)`
  calls for the new 4th arg.

**Verification:** `mise run check` → ALL GREEN (build + vet + `go test ./...`, all 27 packages ok;
`gofmt -l .` empty outside `cauldron/`).
- `mise run check` green — pass.
- `go test -count=1 -run TestCertificate ./internal/certificate` — pass (existing tests under the new
  4-arg signature + the new identity test).
- `TestCertificateRendersInstanceIdentity`: populated `Identity{Instance:"monitor.example.test",
  Operator:"operated by Example Org · example net"}` renders BOTH literals AND the masthead placeholder
  is ABSENT; zero-value `Identity{}` renders the masthead fallback + the generic operator fallback —
  pass.
- Mutation A (run + reverted byte-clean): replacing `{{.Instance}}` in cert.html with the literal
  placeholder → test FAILS. Restored.
- Mutation B (run + reverted byte-clean): forcing `resolveIdentity` to `instance, operator = "", ""` →
  test FAILS. Restored. Tree byte-clean, HEAD unchanged after both.
- `go test -count=1 ./cmd/iscc-monitor` — pass (new `certificate.Handler(...,id)` signature compiles).
- No-CDN ban intact: cert.html still contains `monitor.iscc.codes` (3x); the diff introduces no
  `cdn.`/`jsdelivr`/`unpkg`/`googleapis`/`http://` token.
- Byte-identical-chrome rule verified: the `.chrome-identity`/`.chrome-instance`/`.chrome-operator` CSS
  rule bodies AND the `chrome-actions` masthead block are now byte-identical across dashboard.html,
  dossier.html, and cert.html.
- **Oracle gate: N/A** — pure HTML render of masthead strings; no signature / RFC-6962 / Merkle /
  did:web / fsck / proof / store path touched. `go.mod`/`go.sum`/`schema.sql` byte-identical (not in the
  diff). Scope: exactly 3 production files + 2 test files.

**Next:** The config-leaf env move — move the three identity env keys
(`ISCC_MONITOR_INSTANCE`/`ISCC_MONITOR_OPERATOR`/`ISCC_MONITOR_REALM_NAME`) from `main.go`'s inline
`identity()` into `internal/config`'s `optional(get, key, fallback)` leaf (ratifying the realm-name key
name) and add them to CLAUDE.md's env table. That is a focused `config.go` + `main.go` + CLAUDE.md
change and CLOSES the config-move `normal` issue (deliberately deferred this slice to stay ≤3 prod
files, per next.md Not-In-Scope). After that: the proofserve mastheads (`browser.html`, `records.html`,
`record.html`); `internal/verifier` stays EXCLUDED (its `.codes` chrome is the verifier-app identity).
Folding `Identity` + the now-three duplicated fallback consts + one exported `Resolve` into a shared
leaf would close the dup `low` once the arc reaches all surfaces.

**Notes:**
- **Substring collision found & handled in the test:** the certificate has a pre-existing footer phrase
  "issued by this monitor instance · verifiable cache…" (cert.html:419), so a bare
  `strings.Contains(body, "monitor instance")` placeholder-absence check (as the dossier test uses)
  matches the footer and is vacuous here. The new test instead pins the masthead element specifically
  (`chrome-instance">monitor instance`) for both the negative (configured) and positive (fallback)
  assertions — so it is non-vacuous against the masthead, not the footer. The footer phrase was left
  untouched (out of scope; not an identity claim).
- This is the third VERBATIM copy of the `instanceFallback`/`operatorFallback` consts + `resolveIdentity`
  (dashboard → dossier → certificate). Commented "MUST stay byte-identical"; the consolidation into one
  shared resolve leaf remains the tracked `low` (fold when the masthead arc finishes all surfaces).
- The certificate masthead `.chrome-instance` CSS comment was replaced with the dossier's accurate one
  ("config-driven from the masthead Identity"); the `.chrome-verify` comment was left as-is (the review
  noted only the explanatory CSS comment may differ per file). The dashboard.html's stale
  ".chrome-identity" comment ("static copy in this skeleton") is still the tracked `low` — out of scope
  (would be a 4th prod file).
- The unconfigured certificate masthead now renders a SECOND line (the operator fallback) it did not
  show before — intended per the byte-identical-chrome rule (same change the dossier slice made).
- `internal/certificate` already imports `net/http` (it is an HTTP handler, not a WASM-shared pure
  package), so adding the `internal/dashboard` import introduces no purity regression; no import cycle
  (dashboard does not import certificate/dossier, mirroring the dossier slice).

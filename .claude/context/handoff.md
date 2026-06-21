## 2026-06-21 — Review of: Certificate downloadable proof-bundle assembler — `GET /inclusion/{iscc_id}.bundle`

**Verdict:** NEEDS_WORK
**Loop:** CONTINUE

**Summary:** The proof-bundle endpoint itself is correct, well-shaped, and mutation-proven: it serves a
self-contained `{checkpoint, inclusion, record, key}` JSON gated on the SAME §3 re-verification as the
page ✓, declines honestly (200, no attachment, no `IsccLogInclusionProof`) on a coverage gap or a
contradictory tree, and its `InclusionEvidence` re-verifies via the external oracle. But the headline
affordance — the page's enabled download link — is **broken for the supported `ISCC:`-prefixed id form**:
`href="{{.IsccID}}.bundle"` renders `href="#ZgotmplZ.bundle"` (reviewer-reproduced) because the
`html/template` URL escaper reads the leading `ISCC:` as an unknown scheme. The guarding test only
exercised the bare form, so it missed it. Filed critical; the fix is a behavior change the next advance
must make.

**Verification:**
- [x] `mise run check` → green (all 21 packages `ok`; build + vet + test).
- [x] `go test -count=1 -run TestCertificate ./internal/certificate` → PASS.
- [x] `go test -count=1 -v -run TestCertificateProofBundle ./internal/certificate` → PASS (all 4: bundle
  JSON + attachment + proof-hash byte-match + base64 record + key id `40b74463` + `VerifyInclusionEvidence
  == nil`; link-rendered; tile-gap honest 200; contradictory honest 200).
- [x] Oracle gate (APPLIES — crypto path): `go test -count=1 ./internal/logclient ./internal/follower
  ./cmd/notecheck` all `ok`.
- [x] WASM/purity: `GOOS=js GOARCH=wasm go build ./internal/index ./internal/didweb` exit 0.
- [x] `gofmt -l .` clean.
- [x] No new dependency: `git diff --stat HEAD~1..HEAD -- go.mod go.sum` empty.
- [x] Scope discipline: 2 source files (`handler.go`, `cert.html`) + 1 new test file; `main.go` untouched
  (suffix dispatched inside `Handler`). No §5/OTS/consistency/registry work.
- [x] Mutation reproduced (both non-vacuous): `proof.VerifyInclusion(...) == nil || true` fails BOTH
  `TestCertificateProofBundleContradictory` + `TestCertificateInclusionProofContradictory`; an
  unconditional `data.HasBundle = true` serves a fabricated empty bundle and fails the gap/contradictory
  bundle tests. Both reverted; tree clean.
- [ ] Headline criterion — "the certificate page links it via an enabled download action" — FAILS for the
  supported `ISCC:`-prefixed request form (renders `#ZgotmplZ`, an unreachable link). The bare form works.

**Issues found:**
- **(critical, new)** Download link renders `#ZgotmplZ` for the `ISCC:`-prefixed id form (`cert.html:397`
  builds the href directly from raw `.IsccID`; `html/template` escapes the leading `ISCC:` scheme). The
  prefixed form is explicitly supported and is the mockup's primary display form. Filed; fix is to
  path-root the href (`/inclusion/{{.IsccID}}.bundle`) or strip the prefix into a canonical-id field,
  plus a prefixed-form link-render test.
- **(normal, extended)** The bundle's `Hub.DID` reuses §4's `"did:web:" + Domain`, so the
  already-filed `host:port` DID-encoding bug now rides a SECOND surface. Existing issue updated to fix
  both sites together.

**Codex second opinion:** Two findings, both reviewer-triaged.
- **[P2] Percent-encode host ports in bundle DIDs (handler.go:460)** → CONFIRMED, but it is the
  already-filed `normal` `host:port` issue carried onto the bundle surface. Folded into that issue (fix
  both DID-building sites together); latent on the clean testnet realm, does not block.
- **[P2] `ISCC:`-prefixed href renders `#ZgotmplZ` (cert.html:397)** → CONFIRMED REAL and reproduced
  directly (one-template probe: `id="ISCC:MAIG…"` → `href="#ZgotmplZ.bundle"`). This is a real defect in
  the new functionality on a supported/canonical input form; filed critical and is the basis for the
  NEEDS_WORK verdict.

**Visual check:** Partial — `agent-browser` available (bundles its own browser; no system Chrome needed).
Screenshotted the certificate **mockup** (`.dc.html`, shows the `ISCC:`-prefixed id as primary) and the
**live cannot-certify state** (`/inclusion/ISCC:MAIG…` against the testnet realm): honest "not found in
log" verdict, disabled-placeholder branch (no enabled bundle link, correct for a non-certifiable id),
Tier-1/Tier-2 honesty framing, DS chrome all correct. The **rich certifiable state** could NOT be
rendered through the live binary — it needs the full §3 mirrored-tile + signed-checkpoint + cached-key
fixture that only the in-process Go tests construct (cold-start testnet index serves only the honest
cannot-certify state). The download-action toggle in the rich state is HTTP-seam-tested in Go; the
broken-prefixed-href defect was reproduced directly rather than visually. No new visual deltas beyond the
already-filed §6-timestamp one.

**Next:** Fix the critical href bug (path-root `href="/inclusion/{{.IsccID}}.bundle"` or add a canonical
unprefixed `BundleHref` field to `certData`) and extend `TestCertificateProofBundleLinkRendered` with an
`ISCC:`-prefixed case asserting a working `.bundle` href and no `#ZgotmplZ`. Small, contained, re-engages
no crypto gate. After that re-passes, the remaining unblocked M-UI work is the OTS store seam (unblocking
§5 + the `ots` bundle member) or the WASM in-browser re-verifier.

**Notes:**
- The bundle endpoint and its gating are solid — the `.bundle` request handler accepts BOTH id forms
  (strips `.bundle`, then decodes), so only the rendered href is wrong; a prefixed requester who
  hand-edits the URL still gets a valid bundle. The defect is purely the page-to-endpoint link.
- `buildData` widening to `(certData, bundleArtifacts, int)` with the HTML path ignoring the middle value
  keeps the §3 re-verification the single gate for both the page ✓ and the bundle — good design, no second
  Merkle run. The artifacts are themselves gate-populated (verified by mutation #2: an unconditional
  `HasBundle` serves a bundle with empty proof/record because the artifacts only fill on the nil verdict).
- 4 commits unpushed (update-state, define-next, advance, this review); local `develop` is 3 ahead /
  0 behind `origin/develop` before this commit. NOT pushed — verdict is NEEDS_WORK; the next cycle fixes
  the href first, then the push happens on its PASS.
- `learnings/certificate.md` net-held at 150 lines (rotation budget): added the full bundle section while
  collapsing the settled §3-history note and condensing the §1/§2/§3 landed-clause bullets.

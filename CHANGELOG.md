# Changelog

All notable changes to iscc-monitor are recorded here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and versions follow
[Semantic Versioning](https://semver.org/spec/v2.0.0.html). While the version is below 1.0.0, a minor
release may include breaking changes.

## [0.1.0] - 2026-11-06

First versioned release. It packages the service that has followed the ISCC testnet and mainnet hubs
from `monitor-test.iscc.io` and `monitor.iscc.io`.

### Added

- **Log following and verification:** follows each hub's tlog-tiles log from the authoritative Hub-List,
  verifies checkpoint signatures against the hub's did:web key, and checks RFC-6962 consistency. A
  self-consistency violation freezes the hub and preserves the evidence.
- **Mirror:** stores each hub's tiles and entry bundles in one SQLite file per network and serves them at
  canonical tlog-tiles paths, so a log stays verifiable if its hub goes offline.
- **Proofs and verdicts:** computed inclusion and consistency proofs, a downloadable proof bundle, and a
  verify-for-me endpoint that is flagged as the weaker path compared with verifying a bundle yourself.
- **ISCC-ID input:** certificates, proofs and verdicts accept an ISCC-ID in any letter case, with an optional
  `ISCC:` or `did:iscc:` prefix. Previously the proof and verify-for-me endpoints reported an ID without the
  prefix as not found, even when it was in the log. Proof bundles now always name the ID in its canonical
  `ISCC:` form ([#1](https://github.com/iscc/iscc-monitor/issues/1)).
- **Web interface:** server-rendered dashboard, hub dossiers, log and record browsers, and a Certificate of
  Inclusion for any ISCC-ID in the realm. These pages work without JavaScript, and no page loads anything
  from a CDN. The design follows the ISCC brand guidelines (edition 01). A certificate that proves inclusion
  carries the ISCC Trust Seal, naming its ISCC-ID and issue date.
- **Footer and operator co-branding:** every page shows the running version, linked to its release or
  commit, and the open-source credit. An operator can add its name, website, a logo shown beside the ISCC
  mark, and footer links such as an imprint or privacy policy, all through environment variables
  ([Branding](https://monitor.iscc.codes/operate/branding/)).
- **In-browser verification:** a reproducible, hash-pinned WebAssembly verifier that re-checks a
  certificate's inclusion proof in the visitor's browser.
- **Bitcoin anchoring:** OpenTimestamps proofs for checkpoint roots. The realm register and hub pages show
  each hub's last confirmed anchor and the entries it covers. A certificate shows the earliest confirmed
  anchor that covers its record, with a download of that anchor's proof and the `ots verify` command to
  check it, and the proof bundle carries the evidence that ties the record to that anchor.
  `checkpoint.ots?size=N` serves the proof of an earlier root. The monitor submits new roots to the
  calendar and checks them for confirmation at startup and then hourly (`ISCC_MONITOR_OTS_INTERVAL`), and
  logs one summary line per pass ([#6](https://github.com/iscc/iscc-monitor/issues/6)).
- **API and operations:** OpenAPI 3.1 contract with hosted reference docs, Prometheus metrics, health and
  build-version endpoints, and a non-root container image.
- **Documentation:** a documentation site at [monitor.iscc.codes](https://monitor.iscc.codes) with guides to
  reading a certificate, verifying a proof bundle with a short Python script, checking a Bitcoin anchor and
  comparing log views, an explanation of the trust model, the operator guide and the API reference.
- **Project licensing:** Apache License 2.0, with third-party terms collected in
  [`THIRD_PARTY_NOTICES.txt`](https://github.com/iscc/iscc-monitor/blob/main/THIRD_PARTY_NOTICES.txt) and shipped in the image.
- **Release process:** tagged releases publish `:vX.Y.Z` and `:latest` images and a GitHub Release.
- **Vulnerability scanning:** `govulncheck` runs on every push and pull request, weekly, and before a
  release.

### Security

- Builds now use Go 1.26.10. This fixes 19 standard-library vulnerabilities that the previously deployed
  Go 1.26.4 build could reach, in `net/http`, `net/url`, `net/textproto`, `crypto/tls`, `crypto/x509`,
  `encoding/asn1` and `html/template`.
- Updated `golang.org/x/mod`, `golang.org/x/crypto`, `golang.org/x/sys` and OpenTelemetry. The monitor did
  not call the affected code in these modules.

### Known limitations

- **Rollback across a schema change:** a binary refuses a database migrated by a newer version, so rolling
  back across a migration means restoring the pre-upgrade snapshot, which loses what was observed since.
- **New hubs need a restart:** Hub-List changes apply hourly, but following a newly added hub still requires
  a restart ([#5](https://github.com/iscc/iscc-monitor/issues/5)).
- **Bitcoin anchoring selection:** every newly accepted root is stamped, rather than the newest root once a
  day as ADR-0004 specifies ([#6](https://github.com/iscc/iscc-monitor/issues/6)).
- **In-browser check scope:** the certificate's in-browser re-check verifies the inclusion proof only, not
  the hub's signature on the checkpoint, which the server verifies, or the Bitcoin anchor, which
  `ots verify` checks.
- **No standalone verifier:** a verifier app on an independent domain, which re-checks proofs without
  relying on any monitor's own pages, is planned for a later release.
- **Split views:** detected only when a client compares its view against a monitor. Monitor gossip and
  witness cosigning are not included.
- **Baked testnet realm:** the image's fallback realm document names hosts that no longer serve, so set
  `ISCC_MONITOR_REALM` to the Hub-List URL.
- **Reverse proxy expected:** the HTTP server sets no connection timeouts, so run it behind a proxy such as
  Caddy rather than exposing it directly.

[0.1.0]: https://github.com/iscc/iscc-monitor/releases/tag/v0.1.0

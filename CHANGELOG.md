# Changelog

All notable changes to iscc-monitor are recorded here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and versions follow
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2026-11-08

First release. The ISCC Monitor follows the transparency log of every hub in an ISCC network, checks it,
keeps a copy of it and publishes evidence that anyone can verify without trusting the monitor.

### Added

- Follows every hub in the network's Hub-List. Checks each checkpoint signature against the hub's did:web
  key and the RFC 6962 consistency of its log. A hub that rewrites its history is frozen, and the evidence
  is kept.
- Mirrors each log in one SQLite file per network and serves it at the standard tlog-tiles paths.
- A Certificate of Inclusion for any ISCC-ID, with an in-browser re-check of the inclusion proof, and a
  proof bundle that anyone can verify without the monitor.
- Bitcoin anchoring of accepted log states with OpenTimestamps.
- A dashboard, hub pages and a log browser that work without JavaScript and load nothing from third
  parties.
- Operator co-branding: name, website, logo and footer links.
- An HTTP API described in OpenAPI 3.1, Prometheus metrics and a container image at
  `ghcr.io/iscc/iscc-monitor`.
- Documentation at [monitor.iscc.codes](https://monitor.iscc.codes).

### Known limitations

- A hub added to the Hub-List is followed only after a restart.
- The monitor timestamps every accepted root, not only the newest root once a day.
- A split view is found only when a client compares its view with a monitor. Monitor gossip and witness
  cosigning are not part of this release.
- The fallback realm built into the image names retired testnet hosts. Set `ISCC_MONITOR_REALM` to a
  Hub-List URL.
- The HTTP server sets no connection timeouts. Run it behind a reverse proxy.

[0.1.0]: https://github.com/iscc/iscc-monitor/releases/tag/v0.1.0

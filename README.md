# iscc-monitor

An independent Trust & Transparency service for the ISCC-Hub network. It follows the
[tlog-tiles](https://c2sp.org/tlog-tiles) transparency log of every hub, checks the signatures and the
RFC 6962 consistency of each log, keeps a mirror of each log and publishes evidence that anyone can check
without trusting the monitor.

Documentation: **<https://monitor.iscc.codes>**

## Build and run

You need Go 1.26. The task shortcuts also need [mise](https://mise.jdx.dev/).

```sh
go build -o iscc-monitor ./cmd/iscc-monitor

ISCC_MONITOR_DB=./monitor-test.db \
ISCC_MONITOR_REALM=https://raw.githubusercontent.com/iscc/iscc-hub/refs/heads/main/hubs/testnet.yaml \
ISCC_MONITOR_NORMAL=30s \
ISCC_MONITOR_ADDR=127.0.0.1:41464 \
./iscc-monitor
```

Then open <http://127.0.0.1:41464>. [Configuration](https://monitor.iscc.codes/operate/configuration/) lists
all settings.

## Quality gate

```sh
mise run check   # go build, go vet and go test for every package
mise run fmt     # gofmt
```

## Deployment

The container image is `ghcr.io/iscc/iscc-monitor`. The
[operator guide](https://monitor.iscc.codes/operate/deploy/) covers the image tags, storage, the reverse proxy,
backups, upgrades and branding.

## Specifications

- [`.claude/prd/`](.claude/prd/): the product requirements.
- [`.claude/adr/`](.claude/adr/): the architecture decision records.
- [`.claude/design/`](.claude/design/): the design of the web pages.
- [`CLAUDE.md`](CLAUDE.md): instructions for coding agents, with the project glossary.

## License

Licensed under the [Apache License, Version 2.0](LICENSE). Copyright 2026 ISCC Foundation; see
[`NOTICE`](NOTICE). Third-party software and assets keep their own licenses, collected in
[`THIRD_PARTY_NOTICES.txt`](THIRD_PARTY_NOTICES.txt). The license does not grant rights to the ISCC
name or logo.

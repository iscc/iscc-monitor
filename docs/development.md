---
title: Development
description: Build, test and contribute to the ISCC Monitor, and build this documentation
icon: lucide/code
---

# Development

The monitor is a Go program with no C dependencies. The source is at
[github.com/iscc/iscc-monitor](https://github.com/iscc/iscc-monitor).
{ .iscc-lead }

## Build and run

You need Go 1.26. The task shortcuts also need [mise](https://mise.jdx.dev/).

```bash
git clone https://github.com/iscc/iscc-monitor
cd iscc-monitor
go build -o iscc-monitor ./cmd/iscc-monitor

ISCC_MONITOR_DB=./monitor-test.db \
ISCC_MONITOR_REALM=https://raw.githubusercontent.com/iscc/iscc-hub/refs/heads/main/hubs/testnet.yaml \
ISCC_MONITOR_NORMAL=30s \
ISCC_MONITOR_ADDR=127.0.0.1:41464 \
./iscc-monitor
```

Then open <http://127.0.0.1:41464>. [Configuration](operate/configuration.md) lists all settings.

## Quality gate

```bash
mise run check   # go build, go vet and go test for every package
mise run fmt     # gofmt
```

`mise run check` must pass before a change is merged. CI runs it with `govulncheck` on every push and pull request.

## Project layout

| Path                   | Content                                                              |
| ---------------------- | -------------------------------------------------------------------- |
| `cmd/iscc-monitor`     | The monitor binary                                                   |
| `cmd/wasm`             | The in-browser inclusion verifier, built to `internal/web/verify.wasm` |
| `internal/`            | Follower, store, proof serving, certificate and web pages            |
| `internal/openapi`     | The OpenAPI 3.1 contract. A test fails when it and the routes drift apart |
| `deploy/`              | The fallback realm document built into the image                     |
| `docs/`                | This documentation site                                              |
| `.claude/`             | Product requirements, architecture decision records and design specs |

## Documentation

This site uses [Zensical](https://zensical.org) and the ISCC theme
[zensical-iscc](https://github.com/iscc/zensical-iscc). The tooling needs [uv](https://docs.astral.sh/uv/).

```bash
mise run docs:build   # build into site/
mise run docs:check   # check the built site against the theme's rules
mise run docs:serve   # preview on http://127.0.0.1:41465
```

Pages are Markdown files under `docs/`, listed in `nav` in `zensical.toml`. Each push to `main` that changes the
documentation publishes it to monitor.iscc.codes.

## Releases

Releases follow [Semantic Versioning](https://semver.org). They stay below 1.0.0 until the scope of the first major
version is reached. A release is a tag on `main`. It publishes the `:vX.Y.Z` container image and a GitHub Release.
The [changelog](reference/changelog.md) describes each release.

## License

The monitor is licensed under the [Apache License 2.0](https://github.com/iscc/iscc-monitor/blob/main/LICENSE),
copyright ISCC Foundation. Third-party components keep their own licenses, collected in
[`THIRD_PARTY_NOTICES.txt`](https://github.com/iscc/iscc-monitor/blob/main/THIRD_PARTY_NOTICES.txt). The license
grants no rights to the ISCC name or logo.

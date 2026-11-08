---
title: Configuration
description: Every environment variable the ISCC Monitor reads, with defaults and validation rules
icon: lucide/sliders-horizontal
---

# Configuration

The monitor reads its configuration from environment variables at startup and checks them first. If a value is
invalid, the monitor does not start and names the variable at fault.
{ .iscc-lead }

## Required

`ISCC_MONITOR_DB`
:   Path to the SQLite database file of the network, for example `/data/monitor.db`. The monitor creates the file
    on the first start. Use one file per network. Never point two running instances at the same file.

`ISCC_MONITOR_REALM`
:   The source of the network's hubs: the URL or file path of an
    [iscc-hub Hub-List](https://github.com/iscc/iscc-hub/tree/main/hubs).

    | Network | Hub-List URL                                                                       |
    | ------- | ---------------------------------------------------------------------------------- |
    | Mainnet | `https://raw.githubusercontent.com/iscc/iscc-hub/refs/heads/main/hubs/mainnet.yaml` |
    | Testnet | `https://raw.githubusercontent.com/iscc/iscc-hub/refs/heads/main/hubs/testnet.yaml` |

    The monitor fetches a URL again every hour, so changed hub entries apply without a restart. A newly added hub
    is followed after a restart. The image default is a built-in testnet list whose hosts no longer serve, so
    always set this variable.

## Timing

Intervals use Go's duration syntax, such as `30s`, `5m` or `1h`, and must be positive.

| Variable                    | Default | Meaning                                                                              |
| --------------------------- | ------- | ------------------------------------------------------------------------------------ |
| `ISCC_MONITOR_NORMAL`       | `5m`    | Poll interval for each hub                                                           |
| `ISCC_MONITOR_FROZEN`       | `1h`    | Poll interval for a frozen hub, for evidence. Must be at least `ISCC_MONITOR_NORMAL` |
| `ISCC_MONITOR_OTS_INTERVAL` | `1h`    | Interval for sending new roots to the OpenTimestamps calendar and checking pending roots for Bitcoin confirmation |

The OpenTimestamps pass also runs once at startup. However short the interval, the monitor asks the calendar
about one root at most once an hour. While the root stays unconfirmed, the wait doubles, up to once a day.

## Network

`ISCC_MONITOR_ADDR`
:   The HTTP listen address. Default `:9464`. Keep it on the internal network; see
    [Reverse proxy](deploy.md#reverse-proxy).

## Masthead

All optional. Without them, the masthead shows neutral placeholder text.

`ISCC_MONITOR_INSTANCE`
:   The domain of the instance, for example `monitor.example.com`.

`ISCC_MONITOR_OPERATOR`
:   The line below the domain. Default: "independent Trust & Transparency service · ISCC-Hub network".

`ISCC_MONITOR_REALM_NAME`
:   The display name of the network, for example `ISCC mainnet`. It appears in the title of the hub list on the
    start page.

## Operator co-branding

All optional. [Branding](branding.md) shows them in use.

`ISCC_MONITOR_OPERATOR_NAME`
:   The organization that runs the instance. It appears as "Operated by ..." in the footer and as the text
    alternative of the logo. Required when `ISCC_MONITOR_OPERATOR_URL` or `ISCC_MONITOR_OPERATOR_LOGO` is set.

`ISCC_MONITOR_OPERATOR_URL`
:   The operator's website, an absolute `http` or `https` URL. The name and the logo link to it.

`ISCC_MONITOR_OPERATOR_LOGO`
:   Path to the operator's logo: `.svg`, `.png`, `.webp` or `.jpg`, at most 256 KiB, of the type its extension
    names. The monitor reads it once at startup and serves it at `/_ds/operator-logo`.

`ISCC_MONITOR_FOOTER_LINKS`
:   Footer links, such as an imprint or a privacy policy, as `Label=URL` pairs separated by `|`. Each URL is
    `http`, `https` or `mailto:`.

    ```text
    Imprint=https://example.com/imprint|Privacy=https://example.com/privacy
    ```

## Example

A local instance for testnet with a short poll interval:

```bash
ISCC_MONITOR_DB=./monitor-test.db \
ISCC_MONITOR_REALM=https://raw.githubusercontent.com/iscc/iscc-hub/refs/heads/main/hubs/testnet.yaml \
ISCC_MONITOR_NORMAL=30s \
ISCC_MONITOR_ADDR=127.0.0.1:41464 \
iscc-monitor
```

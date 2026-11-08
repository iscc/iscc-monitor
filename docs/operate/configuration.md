---
title: Configuration
description: Every environment variable the ISCC Monitor reads, with defaults and validation rules
icon: lucide/sliders-horizontal
---

# Configuration

The monitor reads its whole configuration from environment variables at startup. It checks them before doing
anything else and refuses to start with a message naming the variable at fault.
{ .iscc-lead }

## Required

`ISCC_MONITOR_DB`
:   Path to the SQLite database file of the network, for example `/data/monitor.db`. The file is created on the
    first start. Use one file per network, and never point two running instances at the same file.

`ISCC_MONITOR_REALM`
:   Where the network's hubs come from: the URL or file path of an
    [iscc-hub Hub-List](https://github.com/iscc/iscc-hub/tree/main/hubs).

    | Network | Hub-List URL                                                                       |
    | ------- | ---------------------------------------------------------------------------------- |
    | Mainnet | `https://raw.githubusercontent.com/iscc/iscc-hub/refs/heads/main/hubs/mainnet.yaml` |
    | Testnet | `https://raw.githubusercontent.com/iscc/iscc-hub/refs/heads/main/hubs/testnet.yaml` |

    A URL is fetched again every hour, so changed hub entries apply without a restart; a newly added hub is
    followed after a restart. The image sets this variable to a built-in testnet list whose hosts no longer serve,
    so always set it.

## Timing

Intervals use Go's duration syntax, such as `30s`, `5m` or `1h`, and must be positive.

| Variable                    | Default | Meaning                                                                                      |
| --------------------------- | ------- | -------------------------------------------------------------------------------------------- |
| `ISCC_MONITOR_NORMAL`       | `5m`    | How often each hub is polled                                                                 |
| `ISCC_MONITOR_FROZEN`       | `1h`    | How often a frozen hub is polled for evidence; must be at least `ISCC_MONITOR_NORMAL`          |
| `ISCC_MONITOR_OTS_INTERVAL` | `1h`    | How often new roots go to the OpenTimestamps calendar and pending ones are checked for Bitcoin confirmation |

The OpenTimestamps pass also runs once at startup. However short the interval, the calendar is asked about any one
root at most once an hour, backing off up to once a day while it stays unconfirmed.

## Network

`ISCC_MONITOR_ADDR`
:   The HTTP listen address. Default `:9464`. Keep it on the internal network; see
    [Reverse proxy](deploy.md#reverse-proxy).

## Masthead

All optional. Without them, the masthead shows neutral placeholder text.

`ISCC_MONITOR_INSTANCE`
:   The instance's domain, shown in the masthead, for example `monitor.example.com`.

`ISCC_MONITOR_OPERATOR`
:   The line beneath the domain. Default: "independent Trust & Transparency service · ISCC-Hub network".

`ISCC_MONITOR_REALM_NAME`
:   The network's display name, shown in the title of the start page's hub list, for example `ISCC mainnet`.

## Operator co-branding

All optional; [Branding](branding.md) shows them in use.

`ISCC_MONITOR_OPERATOR_NAME`
:   The organization running the instance: "Operated by ..." in the footer and the logo's text alternative.
    Required when `ISCC_MONITOR_OPERATOR_URL` or `ISCC_MONITOR_OPERATOR_LOGO` is set.

`ISCC_MONITOR_OPERATOR_URL`
:   The operator's website, an absolute `http` or `https` URL. It links the name and the logo.

`ISCC_MONITOR_OPERATOR_LOGO`
:   Path to the operator's logo file: `.svg`, `.png`, `.webp` or `.jpg`, at most 256 KiB, and of the type its
    extension names. Read once at startup and served by the instance at `/_ds/operator-logo`.

`ISCC_MONITOR_FOOTER_LINKS`
:   Footer links such as an imprint or a privacy policy: `Label=URL` pairs separated by `|`, each URL `http`,
    `https` or `mailto:`.

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

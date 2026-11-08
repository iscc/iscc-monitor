---
title: HTTP API
description: The routes of an ISCC Monitor instance, its machine API and the OpenAPI contract
icon: lucide/braces
---

# HTTP API

Every instance serves its web pages and a machine API from the same host. All routes are `GET`, need no
authentication and allow cross-origin requests (`Access-Control-Allow-Origin: *`).
{ .iscc-lead }

## Interactive reference

Each instance hosts an interactive reference of its machine API at `/docs`, generated from its OpenAPI 3.1
contract, with a console to try requests:

- Mainnet: <https://monitor.iscc.io/docs>
- Testnet: <https://monitor-test.iscc.io/docs>

The contract itself is at `/openapi.json` and `/openapi.yaml`.

## Machine API

`<hub>` is the domain of a hub the instance follows, such as `iscc.id`. ISCC-ID parameters accept any letter case,
with or without the `ISCC:` or `did:iscc:` prefix.

### Evidence

| Route                                  | Returns                                                                                  |
| -------------------------------------- | ---------------------------------------------------------------------------------------- |
| `/inclusion/<iscc_id>.bundle`          | The [proof bundle](proof-bundle.md) for an ISCC-ID, as a JSON download                     |
| `/<hub>/log/inclusion?iscc_id=<id>`    | Inclusion proof of the ISCC-ID's record against the accepted checkpoint; `&index=<n>` picks a record when several name the ID |
| `/<hub>/log/consistency?from=<size>`   | Consistency proof from any earlier size to the accepted size                              |
| `/<hub>/log/verify?iscc_id=<id>`       | *verify-for-me*: the monitor's own verdict, the weaker path                               |

### Mirror

These serve the hub's log as the monitor mirrored it, at the paths of the
[tlog-tiles](https://c2sp.org/tlog-tiles) specification, so standard tlog-tiles clients can read a log from the
monitor instead of the hub.

| Route                                  | Returns                                                                     |
| -------------------------------------- | --------------------------------------------------------------------------- |
| `/<hub>/log/checkpoint`                | The hub-signed checkpoint the monitor accepted last                          |
| `/<hub>/log/tile/<path>`               | A hash tile or entry bundle, such as `tile/0/000` or `tile/entries/000`       |
| `/<hub>/log/entries?index=<n>`         | The bytes of the single record at position `n`                               |
| `/<hub>/log/checkpoint.ots[?size=<n>]` | The OpenTimestamps proof of the accepted root, or of the root accepted at size `n` |

### Operations

| Route       | Returns                                                            |
| ----------- | ------------------------------------------------------------------ |
| `/healthz`  | Liveness and database readiness                                    |
| `/version`  | The running build                                                  |
| `/metrics`  | Prometheus metrics; often not public, see [Monitoring](../operate/monitoring.md#metrics) |

## Web pages

The pages work without JavaScript and load nothing from other sites. They are not part of the OpenAPI contract.

| Route                     | Page                                                          |
| ------------------------- | ------------------------------------------------------------- |
| `/`                       | Start page: ISCC-ID lookup and every hub with its status        |
| `/inclusion/<iscc_id>`    | [Certificate of Inclusion](../guides/certificate.md)           |
| `/<hub>`                  | A hub's page: status, coverage, checkpoints and Bitcoin anchors |
| `/<hub>/log/`             | The hub's mirrored log                                          |
| `/<hub>/log/records`      | The hub's records, newest first                                 |
| `/<hub>/log/record?index=<n>` | One record                                                  |
| `/docs`                   | Interactive API reference                                       |

## Examples

Verdict from the monitor:

```bash
curl "https://monitor.iscc.io/iscc.id/log/verify?iscc_id=ISCC:MEIGKV5N6NTVF4AB"
```

```json
{
  "iscc_id": "ISCC:MEIGKV5N6NTVF4AB",
  "hub_status": "verified",
  "tree_size": 220,
  "root": "ILlsIpVPzNH4BGP8hM2I+roOSakYq+qyx67H+21yIq0=",
  "leaf_index": 0,
  "included": true,
  "verified": true,
  "reason": ""
}
```

An ISCC-ID the monitor cannot verify still returns `200`, with `"verified": false` and a `reason`.

The bytes of that record:

```bash
curl "https://monitor.iscc.io/iscc.id/log/entries?index=0"
```

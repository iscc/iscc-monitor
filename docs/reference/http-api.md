---
title: HTTP API
description: The routes of an ISCC Monitor instance, its machine API and the OpenAPI contract
icon: lucide/braces
---

# HTTP API

Every instance serves its web pages and a machine API from the same host. All routes are `GET`. They need no
authentication and allow cross-origin requests (`Access-Control-Allow-Origin: *`).
{ .iscc-lead }

## Interactive reference

Each instance has an interactive reference of its machine API at `/docs`, with a console to try requests. It is
generated from the OpenAPI 3.1 contract at `/openapi.json` and `/openapi.yaml`.

- Mainnet: <https://monitor.iscc.io/docs>
- Testnet: <https://monitor-test.iscc.io/docs>

## Machine API

`<hub>` is the domain of a hub that the instance follows, such as `iscc.id`. ISCC-ID parameters accept any letter
case, with or without the `ISCC:` or `did:iscc:` prefix.

### Evidence

| Route                                  | Returns                                                                                  |
| -------------------------------------- | ---------------------------------------------------------------------------------------- |
| `/inclusion/<iscc_id>.bundle`          | The [proof bundle](proof-bundle.md) for an ISCC-ID, as a JSON download                     |
| `/<hub>/log/inclusion?iscc_id=<id>`    | Inclusion proof of the ISCC-ID's record against the accepted checkpoint. If several records name the ID, `&index=<n>` picks one |
| `/<hub>/log/consistency?from=<size>`   | Consistency proof from an earlier size to the accepted size                               |
| `/<hub>/log/verify?iscc_id=<id>`       | verify-for-me: the monitor's own verdict, the weaker path                                 |

For the consistency route, `from` can be any size from 0 to the accepted size, with or without a recorded checkpoint
at that size. A larger `from` returns `400`. `404` means no accepted checkpoint yet, or a tile is not mirrored.

### Mirror

These routes serve the monitor's mirror of the hub's log at [tlog-tiles](https://c2sp.org/tlog-tiles) paths. Standard
tlog-tiles clients can read the log from the monitor instead of the hub.

| Route                                  | Returns                                                                     |
| -------------------------------------- | --------------------------------------------------------------------------- |
| `/<hub>/log/checkpoint`                | The last hub-signed checkpoint that the monitor accepted                     |
| `/<hub>/log/tile/<path>`               | A hash tile or entry bundle, such as `tile/0/000` or `tile/entries/000`       |
| `/<hub>/log/entries?index=<n>`         | The bytes of the single record at position `n`                               |
| `/<hub>/log/checkpoint.ots[?size=<n>]` | The OpenTimestamps proof of the accepted root, or of the root accepted at size `n` |

### Operations

| Route       | Returns                                                            |
| ----------- | ------------------------------------------------------------------ |
| `/healthz`  | Liveness and database readiness                                    |
| `/version`  | The running build                                                  |
| `/metrics`  | Prometheus metrics. Often not public, see [Monitoring](../operate/monitoring.md#metrics) |

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

Get the monitor's verdict:

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

If the monitor cannot verify an ISCC-ID, it still returns `200`, with `"verified": false` and a `reason`.

Get the bytes of that record:

```bash
curl "https://monitor.iscc.io/iscc.id/log/entries?index=0"
```

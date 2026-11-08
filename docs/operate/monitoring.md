---
title: Monitoring
description: Health checks, Prometheus metrics and log lines of an ISCC Monitor instance
icon: lucide/activity
---

# Monitoring

An instance reports its health, its build, the state of each hub and events that need a person. It uses two small
JSON endpoints, Prometheus metrics and its log.
{ .iscc-lead }

## Health and version

`GET /healthz`
:   `200` when the process is live and the database is ready, `503` when the database is not. Use it for
    container health checks and uptime monitoring.

`GET /version`
:   The running build, for example `{"version":"v0.1.0+ac65974"}`. Check it instead of trusting a floating image
    tag.

## Metrics

`GET /metrics` serves Prometheus text format:

| Metric                             | Type    | Labels             | Meaning                                                                   |
| ---------------------------------- | ------- | ------------------ | ------------------------------------------------------------------------- |
| `iscc_monitor_hub_status`          | gauge   | `hub_id`, `status` | 1 for the hub's current status, 0 for the others                          |
| `iscc_monitor_last_observed_at`    | gauge   | `hub_id`           | Unix time of the hub's latest poll that reached a verdict                 |
| `iscc_monitor_poll_failures_total` | counter | `hub_id`           | Failed polls: the hub was unreachable or sent a broken response           |
| `iscc_monitor_violations_total`    | counter | `hub_id`, `kind`   | Self-consistency violations, by `kind`: `shrink`, `fork` or `equivocation` |

`status` is one of the five [hub statuses](../concepts/how-it-works.md#hub-status). `hub_id` is the hub's number
in this instance's database, given in the order the instance first followed the hubs. It is not necessarily the
`hub_id` in the Hub-List.

Alerts to set:

- Any increase of `iscc_monitor_violations_total`, or `iscc_monitor_hub_status{status="frozen"} == 1`. A hub
    rewrote its history, and a person must look at the evidence.
- `iscc_monitor_hub_status{status=~"unresolvable|unverified"} == 1` for longer than a few polls.
- `time() - iscc_monitor_last_observed_at` well above the poll interval.
- A rising rate of `iscc_monitor_poll_failures_total`.

### Public or private

Metrics carry no secrets, but they are operator telemetry, not evidence. The monitor has one listener for all
routes, so you decide at the reverse proxy whether `/metrics` is public. The
[Caddy example](deploy.md#reverse-proxy) refuses it.

## Log

The monitor writes one line per event to standard error, in logfmt (`key=value`). The lines to know:

| Message                                            | Level | When                                                                        |
| -------------------------------------------------- | ----- | --------------------------------------------------------------------------- |
| `hub frozen`                                       | WARN  | A hub's log contradicted itself. `kind` names how. Investigate              |
| `poll hub failed`                                  | ERROR | One poll of one hub failed. Occasional failures are normal                  |
| `ots pass`                                         | INFO  | Each OpenTimestamps pass: roots `stamped`, `confirmed`, still `pending` and `failed`, and the time of the `next` pass |
| `ots pass had failures`                            | ERROR | At least one root failed. `err` has the first error. The next pass tries again |
| `realm Hub-List refreshed`                         | INFO  | The hourly Hub-List refresh applied                                         |
| `realm refresh failed; keeping last good Hub-List` | ERROR | The monitor could not fetch the Hub-List. The previous one stays in use     |

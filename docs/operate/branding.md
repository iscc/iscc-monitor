---
title: Branding
description: Show your organization's name, logo and legal links on an ISCC Monitor instance
icon: lucide/badge
---

# Branding

Every page has the Deep Navy ISCC masthead and footer. The footer shows the running version and the ISCC
Foundation's open-source credit. You can add your own identity beside them without a rebuild.
{ .iscc-lead }

## What you can add

| Item              | Where it shows                                                                  |
| ----------------- | ------------------------------------------------------------------------------- |
| Organization name | Footer, as "Operated by ..."                                                    |
| Website           | Link on the name and the logo                                                   |
| Logo              | Masthead, after an "Operated by" label, on a small white plate                  |
| Footer links      | Footer. Many jurisdictions require an imprint or a privacy policy on a public site |
| Instance domain   | Masthead, with an operator line beneath it                                      |

The ISCC mark, the colors and the open-source credit always stay. Co-branding does not re-theme the monitor.

## Example

Mount the logo read-only and point the settings at it:

```yaml title="compose.yaml (excerpt)"
services:
  iscc-monitor:
    environment:
      ISCC_MONITOR_INSTANCE: monitor.example.com
      ISCC_MONITOR_OPERATOR_NAME: Example GmbH
      ISCC_MONITOR_OPERATOR_URL: https://example.com/
      ISCC_MONITOR_OPERATOR_LOGO: /etc/iscc-monitor/operator-logo.svg
      ISCC_MONITOR_FOOTER_LINKS: "Imprint=https://example.com/imprint|Privacy=https://example.com/privacy"
    volumes:
      - ./operator-logo.svg:/etc/iscc-monitor/operator-logo.svg:ro
```

[Configuration](configuration.md#operator-co-branding) describes each setting.

## Logo guidelines

- Use a transparent background.
- Draw the logo for a light background. The masthead is dark, but the logo sits on a white plate.
- The masthead shows the logo at most 28 pixels tall and 160 pixels wide. Pick proportions that read at that size.
- Keep the file under 256 KiB. SVG stays sharp at every size.

The monitor reads the logo once at startup and serves it itself. Pages load nothing from another origin.

## When the instance refuses to start

The monitor checks the settings at startup. It exits with a message if:

- a URL is malformed,
- a logo or URL is set without a name,
- the logo is missing, larger than 256 KiB, or not the image type its extension names.

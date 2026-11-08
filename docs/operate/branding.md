---
title: Branding
description: Show your organization's name, logo and legal links on an ISCC Monitor instance
icon: lucide/badge
---

# Branding

Every page of an instance shows the Deep Navy ISCC masthead and footer, with the running version and the ISCC
Foundation's open-source credit in the footer. An operator adds its own identity beside them, without a rebuild.
{ .iscc-lead }

## What you can add

- Your organization's **name**, shown as "Operated by ..." in the footer.
- A link to your **website**.
- Your **logo** in the masthead, after an "Operated by" label, on a small white plate.
- **Footer links**, such as an imprint or a privacy policy, which a public site needs in many jurisdictions.
- The instance's **domain** and a line beneath it in the masthead.

The ISCC mark, the colours and the open-source credit stay as they are: co-branding names the operator, it does not
re-theme the monitor.

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

- Give it a transparent background.
- Draw it for a light background: the masthead is dark, but the logo sits on a white plate.
- Choose proportions that still read at 28 pixels tall and 160 pixels wide.
- Keep it under 256 KiB. SVG stays sharp at every size.

The monitor reads the logo once at startup and serves it from the instance itself, so pages still load nothing from
another origin.

## When the instance refuses to start

The settings are checked at startup. The monitor exits with a message if a URL is malformed, a logo or URL is set
without a name, or the logo is missing, larger than 256 KiB or not the image type its extension names.

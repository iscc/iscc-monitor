# ISCC Design System

Brand & UI design system for the **ISCC — International Standard Content Code** (ISO 24138:2024): an open, content-derived identifier and fingerprint for digital content of any media type. An ISCC is *computed from the content itself* — no registry, no signup — and decomposes into five similarity-preserving layers (Meta · Semantic · Content · Data · Instance) ordered from abstract to concrete.

This project gives design agents the real ISCC look-and-feel: brand colors, type, fonts, logos, reusable React primitives, and a pixel-faithful recreation of the **ISCC Generator** web app.

---

## Sources

Everything here is lifted from official ISCC sources — not invented. Reader may not have access; recorded for provenance:

- **Brand guidelines:** `iscc/iscc-skills` → `iscc-brand-guidelines/skills/iscc-brand-guidelines/SKILL.md` (colors, type, logo usage). The same skill was also attached by the user as `uploads/iscc-brand-guidelines.skill`.
- **Generator web app (UI kit source of truth):** `github.com/iscc/iscc-web` → `frontend/` (Vue 3 + SCSS). Recreated faithfully: `App.vue`, `main.scss`, and components `AppHeader`, `IntakeCard`, `EducationStrip`, `ResultCard`, `UnitField`, `lib/iscc.ts`, `lib/icons.ts`, `lib/unit-copy.ts`.
- **Assets:** logo (black/white), favicon, white wordmark and `grain.png` texture — copied from the two repos above into `assets/`.
- **Authority on ISCC concepts:** the ISCC expert tool + `iscc.io` / `iscc.codes` / `iscc.foundation`.

> ⚠️ Note: a separate org also uses the acronym "ISCC" (International Sustainability & Carbon Certification, `iscc-system.org`). That is **unrelated** — this system is the *International Standard Content Code* only.

---

## Content Fundamentals — how ISCC writes

- **Voice:** precise but practical, empowering, trustworthy, open and neutral. Demystify the standard; "here's how" over abstract theory. Professional, never evangelical.
- **Person:** addresses the reader as **you** ("Drop a file and read its code", "Files stay private to **you**"). Speaks plainly about what the product does.
- **Casing:** sentence case for headings and UI ("What you'll get — one code, five layers"). **ALL-CAPS mono** is reserved for technical readout labels and status (`ISCC-UNITS · 4 OF 4`, `DONE IN 1.2 S`, `EXPERIMENTAL`, `SEMANTIC ON`).
- **Layer names:** Title-Case hyphenated — Meta-Code, Semantic-Code, Content-Code, Data-Code, Instance-Code. The five together are "ISCC-UNITS".
- **Tone devices:** confident metaphors grounded in fact ("The **DNA** of your digital content", "what the file says about itself", "the exact bytes"). Honest qualifiers, not hype — the Semantic layer is openly flagged *experimental · not plain ISO 24138*.
- **Emoji:** **none.** Avoid emoji in all brand copy.
- **Numbers/units:** concrete and technical — byte sizes (`2.4 MB`), bit counts (`64 bit`), Hamming/percentage matches, elapsed seconds. Don't pad with vanity stats.
- **Examples of real copy:** "An ISCC is a fingerprint generated *from the content itself* — no registry, no signup." · "Add a title and description and re-decode — watch how only the META field changes." · "One flipped bit anywhere changes it completely — there is no 'similar' on this layer, only identical or not."

---

## Visual Foundations

- **Palette:** ISCC Blue `#0054b2` is primary; Deep Navy `#123663` anchors dark surfaces and technical readouts; Bright Yellow `#ffc300` is the single accent (the brand divider, the "Copy" button, axis rules); Coral Red `#f56169` carries the live ISCC code bar **and** error states; Lime Green `#a6db50` is success + the Instance layer. Two extended blues (Sky `#4596f5`, Cyan `#7ac2f7`) fill in. See `tokens/colors.css`.
- **The five-layer color system** is core brand IP: Meta `#7ac2f7` → Semantic `#4596f5` → Content `#0054b2` → Data `#123663` → Instance `#a6db50`, laid out left-to-right as an *abstract → concrete* axis. Reuse these exact mappings whenever ISCC units appear.
- **Type:** Readex Pro (sans) for everything human-facing — notably **light (300)** for body and lead copy, **600/700** for headings, with negative tracking on large headlines. JetBrains Mono for every code, ID, unit label and status readout. See `tokens/typography.css`.
- **Backgrounds:** a warm off-white **canvas `#fbf7f2`** (not pure white) underneath floating white cards. Hero is full-bleed ISCC Blue; readout strips are full-bleed Deep Navy. No gradients as fills (one subtle multi-stop only on the scan-line and the unit axis bar).
- **Grain texture:** the signature surface treatment — a 256px `grain.png` tiled with `background-blend-mode: overlay` over colored panels (header, hero, footer, code bars, navy readout). Apply via the `.iscc-grain` utility. It is fine paper tooth, never heavy.
- **Corners:** soft, not pill-everything. Controls `0.5rem`, fields `0.65rem`, cards `0.75rem`, the intake instrument `1rem`. Status pills, chips and the compare CTA are fully rounded. The geometric logo mark stays **square** (circle + square with a gap).
- **Cards:** white, 1px `#e9ecef` border, soft **navy-tinted** shadows (`0 10px 24px -4px rgba(18,54,99,.14)`). The hero intake card lifts harder (`shadow-instrument`). No colored left-border accent cards.
- **Shadows:** all tinted toward navy, never neutral gray or black. Four steps: sm · card · raised · instrument.
- **Borders & dividers:** hairline `#e9ecef`; inputs use `#ced4da` and turn ISCC Blue on focus (1px → no glow on inputs; selected unit fields get a yellow `0 0 0 3px` ring).
- **Animation:** purposeful and technical, never decorative bounce. Linear/eased fades; the decode sequence has a cyan **scan-line sweep**, per-bit **flicker** while hashing, and a pulsing status dot. Everything collapses under `prefers-reduced-motion`. Easing is `ease`/`ease-in-out`, ~.15s for UI transitions.
- **Hover/press:** links lighten to white on dark, darken to `#004695` on blue; icon buttons gain an ISCC-Blue border + text; unit fields lift `translateY(-2px)`. Disabled = `opacity .45`.
- **Transparency & blur:** light-on-dark chrome uses `rgba(255,255,255,0.08–0.25)` borders/fills on navy & blue; the page-wide drop catcher dims to `rgba(18,54,99,0.62)`. No backdrop-blur in the brand.
- **Imagery vibe:** cool, technical, content-agnostic. Thumbnails are square with `0.5rem` radius; when absent, a glyph tile stands in. The product is about *codes*, so the hero "imagery" is literally the code bar and bit strips.
- **Layout:** centered `1200px` max container, generous vertical rhythm. Header and footer are fixed-tone navy bars that mirror each other (mark + yellow divider + product label; legal + dev exits).

---

## Iconography

- **System:** a small **feather-style stroke icon set** — 24×24 viewBox, `1.8` stroke width, round caps/joins, `currentColor`. Lifted verbatim from `iscc-web/lib/icons.ts` into `ui_kits/generator/generator-lib.jsx` (`ICONS` map + `UiIcon`).
- **Inventory:** upload, download, trash, copy, check, check-circle, x, code, compare, arrow-right/down, chevron-right, shield, image, type, search, file, film, music, github, plus, alert. Media-type glyphs (image/film/music/type/file) stand in for missing thumbnails.
- **Format:** inline SVG paths (no icon font, no PNG icons, no sprite). When a new icon is needed, match the feather stroke style — or substitute from **Feather / Lucide** (same geometry) and flag it.
- **Emoji:** never used as icons or anywhere in brand copy.
- **Logo mark** is not an icon — it's the geometric circle+square symbol; use `assets/favicon.png` at small sizes, the full wordmark otherwise.

---

## Index / Manifest

**Root**
- `styles.css` — the single entry point consumers link (an `@import` manifest only).
- `tokens/` — `fonts.css` (`@font-face`), `colors.css`, `typography.css`, `spacing.css`, `base.css` (resets, `.iscc-grain`, keyframes).
- `assets/` — `logo-black.png`, `logo-white.png`, `iscc-logo-white.png` (app wordmark), `favicon.png`, `grain.png`.
- `SKILL.md` — Agent-Skill manifest (download & use in Claude Code).

**Components** (`window.ISCCDesignSystem_<hash>` once compiled)
- `components/core/Button` — 5-variant brand action button.
- `components/forms/TextField`, `components/forms/Switch` — inputs, textarea, code field, toggle.
- `components/layout/Card` — light/inset/navy/brand surfaces + elevation.
- `components/feedback/Badge` — mono status pills & flag chips.
- `components/data/UnitBadge` (+ `ISCC_UNITS`), `components/data/IsccCode` — the ISCC-specific primitives.
- `components/navigation/Tabs` — underline tab bar.

**UI Kit**
- `ui_kits/generator/` — the full **ISCC Generator** app (`index.html` + `generator-lib.jsx`, `chrome.jsx`, `result.jsx`, `app.jsx`). Drop a file / paste text / decode a code and watch the live five-layer readout. Also registered as a Starting Point.

**Design System tab** — every `*.card.html` (foundations under `guidelines/`, component demos beside each component) is tagged `@dsCard` and renders automatically, grouped Colors · Type · Spacing · Brand · Components · Generator.

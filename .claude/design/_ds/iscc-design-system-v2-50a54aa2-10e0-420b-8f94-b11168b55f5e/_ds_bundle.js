/* @ds-bundle: {"format":3,"namespace":"ISCCDesignSystem_50a54a","components":[{"name":"Button","sourcePath":"components/core/Button.jsx"},{"name":"IsccCode","sourcePath":"components/data/IsccCode.jsx"},{"name":"ISCC_UNITS","sourcePath":"components/data/UnitBadge.jsx"},{"name":"UnitBadge","sourcePath":"components/data/UnitBadge.jsx"},{"name":"Badge","sourcePath":"components/feedback/Badge.jsx"},{"name":"Switch","sourcePath":"components/forms/Switch.jsx"},{"name":"TextField","sourcePath":"components/forms/TextField.jsx"},{"name":"Card","sourcePath":"components/layout/Card.jsx"},{"name":"Tabs","sourcePath":"components/navigation/Tabs.jsx"}],"sourceHashes":{"components/core/Button.jsx":"bb1b73212985","components/data/IsccCode.jsx":"89c8f9dd218b","components/data/UnitBadge.jsx":"edb17078af12","components/feedback/Badge.jsx":"0446f0198b29","components/forms/Switch.jsx":"e0cd4442ebfc","components/forms/TextField.jsx":"28887495a668","components/layout/Card.jsx":"fffc5a27f3cf","components/navigation/Tabs.jsx":"7f45fec000f5","ui_kits/generator/app.jsx":"14fefc2d2ff4","ui_kits/generator/chrome.jsx":"5627db99f55e","ui_kits/generator/generator-lib.jsx":"e5325ba802d4","ui_kits/generator/result.jsx":"6762e13dd563"},"inlinedExternals":[],"unexposedExports":[]} */

(() => {

const __ds_ns = (window.ISCCDesignSystem_50a54a = window.ISCCDesignSystem_50a54a || {});

const __ds_scope = {};

(__ds_ns.__errors = __ds_ns.__errors || []);

// components/core/Button.jsx
try { (() => {
function _extends() { return _extends = Object.assign ? Object.assign.bind() : function (n) { for (var e = 1; e < arguments.length; e++) { var t = arguments[e]; for (var r in t) ({}).hasOwnProperty.call(t, r) && (n[r] = t[r]); } return n; }, _extends.apply(null, arguments); }
/**
 * ISCC primary action button. Maps to the brand's filled-blue CTA, the
 * outline "compare" pill, the navy/yellow copy button and quiet ghost links.
 */
function Button({
  variant = "primary",
  size = "md",
  pill = false,
  disabled = false,
  type = "button",
  style,
  children,
  ...rest
}) {
  const sizes = {
    sm: {
      padding: "0.45rem 0.8rem",
      fontSize: "var(--text-sm)",
      gap: "0.4rem"
    },
    md: {
      padding: "0.55rem 1rem",
      fontSize: "var(--text-sm)",
      gap: "0.45rem"
    },
    lg: {
      padding: "0.7rem 1.25rem",
      fontSize: "var(--text-base)",
      gap: "0.55rem"
    }
  };
  const variants = {
    primary: {
      background: "var(--iscc-blue)",
      color: "var(--iscc-white)",
      border: "1.5px solid transparent"
    },
    secondary: {
      background: "transparent",
      color: "var(--iscc-blue)",
      border: "1.5px solid var(--iscc-blue)"
    },
    ghost: {
      background: "transparent",
      color: "var(--text-muted)",
      border: "1.5px solid transparent"
    },
    accent: {
      background: "var(--iscc-bright-yellow)",
      color: "var(--iscc-deep-navy)",
      border: "1.5px solid transparent"
    },
    "on-dark": {
      background: "transparent",
      color: "var(--iscc-white)",
      border: "1px solid rgba(255,255,255,0.35)"
    }
  };
  return /*#__PURE__*/React.createElement("button", _extends({
    type: type,
    disabled: disabled,
    style: {
      display: "inline-flex",
      alignItems: "center",
      justifyContent: "center",
      ...sizes[size],
      fontFamily: "var(--font-sans)",
      fontWeight: "var(--weight-semibold)",
      lineHeight: 1,
      borderRadius: pill ? "var(--radius-pill)" : "var(--radius-sm)",
      cursor: disabled ? "not-allowed" : "pointer",
      opacity: disabled ? 0.45 : 1,
      whiteSpace: "nowrap",
      transition: "background-color .15s ease, border-color .15s ease, opacity .15s ease",
      ...variants[variant],
      ...style
    }
  }, rest), children);
}
Object.assign(__ds_scope, { Button });
})(); } catch (e) { __ds_ns.__errors.push({ path: "components/core/Button.jsx", error: String((e && e.message) || e) }); }

// components/data/IsccCode.jsx
try { (() => {
function _extends() { return _extends = Object.assign ? Object.assign.bind() : function (n) { for (var e = 1; e < arguments.length; e++) { var t = arguments[e]; for (var r in t) ({}).hasOwnProperty.call(t, r) && (n[r] = t[r]); } return n; }, _extends.apply(null, arguments); }
/**
 * Renders an ISCC code (or any ISCC-ID) in the brand's monospace treatment.
 * - `inline`  navy mono on off-white, for IDs inside text and tables
 * - `bar`     the full-width coral code bar from the decoder readout, with copy
 */
function IsccCode({
  code = "",
  variant = "inline",
  onDark = false,
  copyable = false,
  style,
  ...rest
}) {
  const [copied, setCopied] = React.useState(false);
  const copy = async () => {
    try {
      await navigator.clipboard?.writeText(code);
      setCopied(true);
      setTimeout(() => setCopied(false), 1600);
    } catch (e) {/* clipboard unavailable */}
  };
  if (variant === "bar") {
    return /*#__PURE__*/React.createElement("div", _extends({
      style: {
        display: "flex",
        alignItems: "center",
        gap: "0.9rem",
        ...style
      }
    }, rest), /*#__PURE__*/React.createElement("div", {
      className: "iscc-grain",
      style: {
        flex: 1,
        minWidth: 0,
        background: "var(--iscc-coral-red)",
        borderRadius: "var(--radius-sm)",
        padding: "0.7rem 1rem",
        fontFamily: "var(--font-mono)",
        fontSize: "clamp(9px, 1.6vw, 16px)",
        fontWeight: "var(--weight-light)",
        letterSpacing: "var(--tracking-tight)",
        color: "var(--iscc-white)",
        whiteSpace: "nowrap",
        overflow: "hidden",
        textOverflow: "ellipsis"
      }
    }, code), copyable && /*#__PURE__*/React.createElement("button", {
      type: "button",
      onClick: copy,
      style: {
        flexShrink: 0,
        background: "var(--iscc-bright-yellow)",
        color: "var(--iscc-deep-navy)",
        border: "none",
        borderRadius: "var(--radius-sm)",
        padding: "0.5rem 0.95rem",
        fontFamily: "var(--font-sans)",
        fontSize: "var(--text-sm)",
        fontWeight: "var(--weight-semibold)",
        cursor: "pointer"
      }
    }, copied ? "Copied" : "Copy"));
  }
  return /*#__PURE__*/React.createElement("span", _extends({
    onClick: copyable ? copy : undefined,
    title: copyable ? copied ? "Copied" : "Click to copy" : undefined,
    style: {
      display: "inline-block",
      fontFamily: "var(--font-mono)",
      fontSize: "var(--text-xs)",
      letterSpacing: "0.03em",
      color: onDark ? "var(--iscc-white)" : "var(--iscc-deep-navy)",
      background: onDark ? "rgba(255,255,255,0.08)" : "var(--iscc-off-white)",
      border: onDark ? "1px solid rgba(255,255,255,0.16)" : "1px solid var(--border-subtle)",
      borderRadius: "var(--radius-xs)",
      padding: "0.2rem 0.5rem",
      wordBreak: "break-all",
      cursor: copyable ? "pointer" : "default",
      ...style
    }
  }, rest), code);
}
Object.assign(__ds_scope, { IsccCode });
})(); } catch (e) { __ds_ns.__errors.push({ path: "components/data/IsccCode.jsx", error: String((e && e.message) || e) }); }

// components/data/UnitBadge.jsx
try { (() => {
function _extends() { return _extends = Object.assign ? Object.assign.bind() : function (n) { for (var e = 1; e < arguments.length; e++) { var t = arguments[e]; for (var r in t) ({}).hasOwnProperty.call(t, r) && (n[r] = t[r]); } return n; }, _extends.apply(null, arguments); }
/** The five ISCC-UNIT layers, abstract → concrete, with their brand identity. */
const ISCC_UNITS = {
  meta: {
    label: "META",
    color: "var(--iscc-unit-meta)",
    accent: "var(--iscc-unit-meta-accent)",
    tint: "rgba(122,194,247,0.18)",
    meaning: "metadata similarity"
  },
  semantic: {
    label: "SEMANTIC",
    color: "var(--iscc-unit-semantic)",
    accent: "var(--iscc-unit-semantic-accent)",
    tint: "rgba(69,150,245,0.13)",
    meaning: "semantic similarity · experimental"
  },
  content: {
    label: "CONTENT",
    color: "var(--iscc-unit-content)",
    accent: "var(--iscc-unit-content-accent)",
    tint: "rgba(0,84,178,0.10)",
    meaning: "syntactic similarity"
  },
  data: {
    label: "DATA",
    color: "var(--iscc-unit-data)",
    accent: "var(--iscc-unit-data-accent)",
    tint: "rgba(18,54,99,0.10)",
    meaning: "data similarity"
  },
  instance: {
    label: "INSTANCE",
    color: "var(--iscc-unit-instance)",
    accent: "var(--iscc-unit-instance-accent)",
    tint: "rgba(166,219,80,0.18)",
    meaning: "exact match · checksum"
  }
};

/**
 * A labeled chip for one ISCC-UNIT layer. Square color swatch + mono label,
 * with an optional one-line meaning. Use to annotate codes, legends and docs.
 */
function UnitBadge({
  kind = "content",
  showMeaning = false,
  onDark = false,
  style,
  ...rest
}) {
  const u = ISCC_UNITS[kind] || ISCC_UNITS.content;
  return /*#__PURE__*/React.createElement("span", _extends({
    style: {
      display: "inline-flex",
      alignItems: "center",
      gap: "0.5rem",
      padding: "0.3rem 0.6rem",
      borderRadius: "var(--radius-xs)",
      background: onDark ? "rgba(255,255,255,0.07)" : u.tint,
      border: onDark ? "1px solid rgba(255,255,255,0.16)" : `1px solid ${u.color}`,
      ...style
    }
  }, rest), /*#__PURE__*/React.createElement("span", {
    style: {
      width: 10,
      height: 10,
      borderRadius: 3,
      background: u.color,
      border: onDark ? "1px solid rgba(255,255,255,0.5)" : "none",
      flexShrink: 0
    }
  }), /*#__PURE__*/React.createElement("span", {
    style: {
      fontFamily: "var(--font-mono)",
      fontSize: "var(--text-2xs)",
      fontWeight: "var(--weight-bold)",
      letterSpacing: "0.12em",
      color: onDark ? "var(--iscc-white)" : u.accent
    }
  }, u.label), showMeaning && /*#__PURE__*/React.createElement("span", {
    style: {
      fontFamily: "var(--font-sans)",
      fontSize: "var(--text-2xs)",
      fontWeight: "var(--weight-light)",
      color: onDark ? "rgba(255,255,255,0.65)" : "var(--text-muted)"
    }
  }, u.meaning));
}
Object.assign(__ds_scope, { ISCC_UNITS, UnitBadge });
})(); } catch (e) { __ds_ns.__errors.push({ path: "components/data/UnitBadge.jsx", error: String((e && e.message) || e) }); }

// components/feedback/Badge.jsx
try { (() => {
function _extends() { return _extends = Object.assign ? Object.assign.bind() : function (n) { for (var e = 1; e < arguments.length; e++) { var t = arguments[e]; for (var r in t) ({}).hasOwnProperty.call(t, r) && (n[r] = t[r]); } return n; }, _extends.apply(null, arguments); }
/**
 * Status pill / chip. The brand uses mono uppercase pills with a leading dot
 * for processing state (UPLOADING…, DONE IN 1.2 S, FAILED) and softer chips
 * for flags like "SEMANTIC ON" / "EXPERIMENTAL".
 */
function Badge({
  tone = "neutral",
  dot = false,
  mono = false,
  pulse = false,
  style,
  children,
  ...rest
}) {
  const tones = {
    info: {
      bg: "rgba(0,84,178,0.07)",
      bd: "rgba(0,84,178,0.30)",
      fg: "var(--iscc-blue)",
      dot: "var(--iscc-blue)"
    },
    success: {
      bg: "rgba(166,219,80,0.18)",
      bd: "rgba(166,219,80,0.55)",
      fg: "#4d6e14",
      dot: "#79a832"
    },
    error: {
      bg: "rgba(245,97,105,0.10)",
      bd: "rgba(245,97,105,0.45)",
      fg: "#b3434a",
      dot: "var(--iscc-coral-red)"
    },
    warning: {
      bg: "rgba(255,195,0,0.16)",
      bd: "rgba(255,195,0,0.55)",
      fg: "#7a5c00",
      dot: "var(--iscc-bright-yellow)"
    },
    neutral: {
      bg: "var(--iscc-off-white)",
      bd: "var(--iscc-border)",
      fg: "var(--text-muted)",
      dot: "var(--text-faint)"
    }
  };
  const t = tones[tone] || tones.neutral;
  return /*#__PURE__*/React.createElement("span", _extends({
    style: {
      display: "inline-flex",
      alignItems: "center",
      gap: "0.45rem",
      padding: "0.32rem 0.8rem",
      borderRadius: "var(--radius-pill)",
      background: t.bg,
      border: `1px solid ${t.bd}`,
      color: t.fg,
      fontFamily: mono ? "var(--font-mono)" : "var(--font-sans)",
      fontSize: mono ? "var(--text-2xs)" : "var(--text-xs)",
      fontWeight: "var(--weight-semibold)",
      letterSpacing: mono ? "var(--tracking-wide)" : "0.02em",
      textTransform: mono ? "uppercase" : "none",
      whiteSpace: "nowrap",
      lineHeight: 1.4,
      ...style
    }
  }, rest), dot && /*#__PURE__*/React.createElement("span", {
    style: {
      width: 8,
      height: 8,
      borderRadius: "50%",
      background: t.dot,
      flexShrink: 0,
      animation: pulse ? "iscc-pulse-dot 1.4s ease-in-out infinite" : "none"
    }
  }), children);
}
Object.assign(__ds_scope, { Badge });
})(); } catch (e) { __ds_ns.__errors.push({ path: "components/feedback/Badge.jsx", error: String((e && e.message) || e) }); }

// components/forms/Switch.jsx
try { (() => {
function _extends() { return _extends = Object.assign ? Object.assign.bind() : function (n) { for (var e = 1; e < arguments.length; e++) { var t = arguments[e]; for (var r in t) ({}).hasOwnProperty.call(t, r) && (n[r] = t[r]); } return n; }, _extends.apply(null, arguments); }
/**
 * On/off switch in the brand style — turns ISCC Blue when on. Used for the
 * Generator's "Semantic code" and "Granular simprints" toggles.
 */
function Switch({
  checked = false,
  onChange,
  disabled = false,
  id,
  style,
  ...rest
}) {
  return /*#__PURE__*/React.createElement("button", _extends({
    type: "button",
    role: "switch",
    id: id,
    "aria-checked": checked,
    disabled: disabled,
    onClick: () => !disabled && onChange?.(!checked),
    style: {
      position: "relative",
      width: 38,
      height: 22,
      flexShrink: 0,
      borderRadius: "var(--radius-pill)",
      border: "none",
      padding: 0,
      cursor: disabled ? "not-allowed" : "pointer",
      opacity: disabled ? 0.5 : 1,
      background: checked ? "var(--iscc-blue)" : "var(--border-control)",
      transition: "background-color .18s ease",
      ...style
    }
  }, rest), /*#__PURE__*/React.createElement("span", {
    style: {
      position: "absolute",
      top: 2,
      left: checked ? 18 : 2,
      width: 18,
      height: 18,
      borderRadius: "50%",
      background: "var(--iscc-white)",
      boxShadow: "0 1px 3px rgba(0,0,0,0.25)",
      transition: "left .18s ease"
    }
  }));
}
Object.assign(__ds_scope, { Switch });
})(); } catch (e) { __ds_ns.__errors.push({ path: "components/forms/Switch.jsx", error: String((e && e.message) || e) }); }

// components/forms/TextField.jsx
try { (() => {
function _extends() { return _extends = Object.assign ? Object.assign.bind() : function (n) { for (var e = 1; e < arguments.length; e++) { var t = arguments[e]; for (var r in t) ({}).hasOwnProperty.call(t, r) && (n[r] = t[r]); } return n; }, _extends.apply(null, arguments); }
/**
 * Text input / textarea / monospace code input in the brand field style:
 * off-white inset, soft border that turns ISCC Blue on focus.
 */
function TextField({
  as = "input",
  mono = false,
  invalid = false,
  style,
  ...rest
}) {
  const Tag = as === "textarea" ? "textarea" : "input";
  const [focus, setFocus] = React.useState(false);
  return /*#__PURE__*/React.createElement(Tag, _extends({
    onFocus: e => {
      setFocus(true);
      rest.onFocus?.(e);
    },
    onBlur: e => {
      setFocus(false);
      rest.onBlur?.(e);
    },
    style: {
      width: "100%",
      boxSizing: "border-box",
      display: "block",
      padding: as === "textarea" ? "0.75rem 0.9rem" : "0.6rem 0.9rem",
      minHeight: as === "textarea" ? "7.5rem" : undefined,
      resize: as === "textarea" ? "vertical" : undefined,
      fontFamily: mono ? "var(--font-mono)" : "var(--font-sans)",
      fontSize: mono ? "var(--text-sm)" : "var(--text-sm)",
      fontWeight: "var(--weight-light)",
      lineHeight: "var(--leading-normal)",
      color: "var(--text-heading)",
      background: "var(--surface-inset)",
      border: `1px solid ${invalid ? "var(--iscc-coral-red)" : focus ? "var(--iscc-blue)" : "var(--border-control)"}`,
      borderRadius: "var(--radius-md)",
      outline: "none",
      transition: "border-color .15s ease",
      ...style
    }
  }, rest));
}
Object.assign(__ds_scope, { TextField });
})(); } catch (e) { __ds_ns.__errors.push({ path: "components/forms/TextField.jsx", error: String((e && e.message) || e) }); }

// components/layout/Card.jsx
try { (() => {
function _extends() { return _extends = Object.assign ? Object.assign.bind() : function (n) { for (var e = 1; e < arguments.length; e++) { var t = arguments[e]; for (var r in t) ({}).hasOwnProperty.call(t, r) && (n[r] = t[r]); } return n; }, _extends.apply(null, arguments); }
/**
 * Surface container. The brand floats white cards on a warm canvas with soft,
 * navy-tinted shadows. `tone="dark"` gives the navy readout panel.
 */
function Card({
  tone = "light",
  elevation = "card",
  padded = true,
  style,
  children,
  ...rest
}) {
  const tones = {
    light: {
      background: "var(--surface-card)",
      color: "var(--text-body)",
      border: "1px solid var(--border-subtle)"
    },
    inset: {
      background: "var(--surface-inset)",
      color: "var(--text-body)",
      border: "1px solid var(--border-subtle)"
    },
    dark: {
      background: "var(--surface-dark)",
      color: "var(--text-on-dark)",
      border: "1px solid transparent"
    },
    brand: {
      background: "var(--surface-brand)",
      color: "var(--text-on-dark)",
      border: "1px solid transparent"
    }
  };
  const shadows = {
    none: "none",
    sm: "var(--shadow-sm)",
    card: "var(--shadow-card)",
    raised: "var(--shadow-raised)",
    instrument: "var(--shadow-instrument)"
  };
  return /*#__PURE__*/React.createElement("div", _extends({
    style: {
      borderRadius: "var(--radius-lg)",
      boxShadow: shadows[elevation],
      padding: padded ? "var(--space-6)" : 0,
      overflow: "hidden",
      ...tones[tone],
      ...style
    }
  }, rest), children);
}
Object.assign(__ds_scope, { Card });
})(); } catch (e) { __ds_ns.__errors.push({ path: "components/layout/Card.jsx", error: String((e && e.message) || e) }); }

// components/navigation/Tabs.jsx
try { (() => {
function _extends() { return _extends = Object.assign ? Object.assign.bind() : function (n) { for (var e = 1; e < arguments.length; e++) { var t = arguments[e]; for (var r in t) ({}).hasOwnProperty.call(t, r) && (n[r] = t[r]); } return n; }, _extends.apply(null, arguments); }
/**
 * Underline tab bar matching the Generator's intake tabs: equal columns,
 * a bottom border, and an ISCC-Blue active indicator + label.
 */
function Tabs({
  tabs = [],
  value,
  onChange,
  style,
  ...rest
}) {
  return /*#__PURE__*/React.createElement("div", _extends({
    role: "tablist",
    style: {
      display: "grid",
      gridTemplateColumns: `repeat(${tabs.length || 1}, 1fr)`,
      borderBottom: "1px solid var(--border-subtle)",
      ...style
    }
  }, rest), tabs.map(tab => {
    const id = typeof tab === "string" ? tab : tab.id;
    const label = typeof tab === "string" ? tab : tab.label;
    const icon = typeof tab === "string" ? null : tab.icon;
    const active = id === value;
    return /*#__PURE__*/React.createElement("button", {
      key: id,
      role: "tab",
      "aria-selected": active,
      onClick: () => onChange?.(id),
      style: {
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        gap: "0.45rem",
        padding: "0.8rem 0",
        background: "none",
        border: "none",
        borderBottom: `3px solid ${active ? "var(--iscc-blue)" : "transparent"}`,
        color: active ? "var(--iscc-blue)" : "var(--text-muted)",
        fontFamily: "var(--font-sans)",
        fontSize: "var(--text-sm)",
        fontWeight: active ? "var(--weight-semibold)" : "var(--weight-medium)",
        cursor: "pointer",
        transition: "color .15s ease, border-color .15s ease"
      }
    }, icon, /*#__PURE__*/React.createElement("span", null, label));
  }));
}
Object.assign(__ds_scope, { Tabs });
})(); } catch (e) { __ds_ns.__errors.push({ path: "components/navigation/Tabs.jsx", error: String((e && e.message) || e) }); }

// ui_kits/generator/app.jsx
try { (() => {
/* Generator UI kit — orchestrator. Owns the specimen feed and fakes the
   upload → decode → done lifecycle so the readout animates like the real app. */

function GeneratorApp() {
  const [specimens, setSpecimens] = React.useState([]);
  const seq = React.useRef(0);
  const update = (id, patch) => setSpecimens(list => list.map(s => s.id === id ? {
    ...s,
    ...patch
  } : s));
  const SAMPLES = {
    file: {
      label: "mountain-sunset.jpg",
      glyph: "image",
      facts: "image/jpeg · 2.4 MB · 1920×1280"
    },
    text: {
      label: "Pasted text",
      glyph: "type",
      facts: "text/plain · 1,204 characters"
    },
    code: {
      label: "Decoded ISCC",
      glyph: "search",
      facts: "KED5-72P4-AOF5-K6QX · 4 units"
    }
  };
  const onGenerate = ({
    kind,
    semantic,
    granular
  }) => {
    const id = ++seq.current;
    const sample = SAMPLES[kind];
    const startedAt = performance.now();
    const kinds = semantic ? ["meta", "semantic", "content", "data", "instance"] : ["meta", "content", "data", "instance"];
    const base = {
      id,
      kind,
      label: sample.label,
      glyph: sample.glyph,
      facts: sample.facts,
      semantic: kind === "code" ? false : semantic,
      granular,
      status: kind === "file" ? "uploading" : "processing",
      progress: 0,
      explain: null,
      elapsed: null,
      startedAt
    };
    setSpecimens(list => [base, ...list]);
    const finish = () => {
      const explain = fakeExplain(id, kind === "code" ? ["meta", "content", "data", "instance"] : kinds);
      update(id, {
        status: "done",
        explain,
        elapsed: (performance.now() - startedAt) / 1000 + 0.3
      });
    };
    if (kind === "file") {
      let p = 0;
      const up = setInterval(() => {
        p += 18 + Math.random() * 14;
        if (p >= 100) {
          p = 100;
          clearInterval(up);
          update(id, {
            progress: 100,
            status: "processing"
          });
          setTimeout(finish, 1100);
        } else update(id, {
          progress: Math.floor(p)
        });
      }, 130);
    } else if (kind === "code") {
      setTimeout(finish, 650);
    } else {
      setTimeout(finish, 1300);
    }
  };
  const onRemove = id => setSpecimens(list => list.filter(s => s.id !== id));
  return /*#__PURE__*/React.createElement("div", {
    style: {
      minHeight: "100vh",
      display: "flex",
      flexDirection: "column"
    }
  }, /*#__PURE__*/React.createElement(AppHeader, null), /*#__PURE__*/React.createElement(Hero, {
    onGenerate: onGenerate
  }), /*#__PURE__*/React.createElement("main", {
    style: {
      flex: 1,
      padding: "2.5rem 0 3.5rem"
    }
  }, /*#__PURE__*/React.createElement("div", {
    style: {
      maxWidth: "var(--container-max)",
      margin: "0 auto",
      padding: "0 1.5rem"
    }
  }, specimens.length === 0 ? /*#__PURE__*/React.createElement(EducationStrip, null) : /*#__PURE__*/React.createElement("div", {
    style: {
      display: "flex",
      flexDirection: "column",
      gap: "1.5rem"
    }
  }, specimens.map(s => /*#__PURE__*/React.createElement(ResultCard, {
    key: s.id,
    specimen: s,
    onRemove: () => onRemove(s.id)
  }))))), /*#__PURE__*/React.createElement(AppFooter, null));
}
ReactDOM.createRoot(document.getElementById("root")).render(/*#__PURE__*/React.createElement(GeneratorApp, null));
})(); } catch (e) { __ds_ns.__errors.push({ path: "ui_kits/generator/app.jsx", error: String((e && e.message) || e) }); }

// ui_kits/generator/chrome.jsx
try { (() => {
/* Generator UI kit — page chrome: header, hero + intake instrument, education
   strip and footer. Faithful recreation of iscc/iscc-web. */

function AppHeader() {
  return /*#__PURE__*/React.createElement("header", {
    className: "iscc-grain",
    style: {
      background: "var(--iscc-deep-navy)"
    }
  }, /*#__PURE__*/React.createElement("div", {
    style: {
      maxWidth: "var(--container-max)",
      margin: "0 auto",
      padding: "0.85rem 1.5rem",
      display: "flex",
      alignItems: "center",
      justifyContent: "space-between"
    }
  }, /*#__PURE__*/React.createElement("div", {
    style: {
      display: "flex",
      alignItems: "center",
      gap: "0.9rem"
    }
  }, /*#__PURE__*/React.createElement("img", {
    src: "../../assets/iscc-logo-white.png",
    alt: "ISCC logo",
    style: {
      height: "1.9rem",
      width: "auto",
      display: "block"
    }
  }), /*#__PURE__*/React.createElement("div", {
    style: {
      width: 3,
      height: "1.6rem",
      background: "var(--iscc-bright-yellow)",
      borderRadius: 1
    }
  }), /*#__PURE__*/React.createElement("span", {
    style: {
      color: "#fff",
      fontWeight: 300,
      letterSpacing: "0.1em",
      fontSize: "1.05rem"
    }
  }, "GENERATOR")), /*#__PURE__*/React.createElement("div", {
    style: {
      display: "flex",
      alignItems: "center",
      gap: "1.5rem"
    }
  }, /*#__PURE__*/React.createElement("a", {
    href: "#",
    style: {
      color: "rgba(255,255,255,0.78)",
      fontSize: "0.85rem",
      fontWeight: 300,
      textDecoration: "none"
    }
  }, "What is an ISCC?"), /*#__PURE__*/React.createElement("a", {
    href: "#",
    style: {
      display: "inline-flex",
      alignItems: "center",
      gap: "0.45rem",
      color: "#fff",
      fontSize: "0.85rem",
      fontWeight: 500,
      textDecoration: "none",
      border: "1px solid rgba(255,255,255,0.35)",
      borderRadius: "0.5rem",
      padding: "0.45rem 0.8rem"
    }
  }, /*#__PURE__*/React.createElement(UiIcon, {
    name: "code",
    size: 14
  }), /*#__PURE__*/React.createElement("span", null, "API Docs")))));
}
function IntakeCard({
  onGenerate
}) {
  const [tab, setTab] = React.useState("file");
  const [semantic, setSemantic] = React.useState(false);
  const [granular, setGranular] = React.useState(false);
  const [text, setText] = React.useState("");
  const [codeInput, setCodeInput] = React.useState("");
  const [drag, setDrag] = React.useState(false);
  const TABS = [{
    id: "file",
    label: "Media file",
    icon: "upload"
  }, {
    id: "text",
    label: "Plain text",
    icon: "type"
  }, {
    id: "code",
    label: "ISCC code",
    icon: "search"
  }];
  const tabBtn = t => /*#__PURE__*/React.createElement("button", {
    key: t.id,
    onClick: () => setTab(t.id),
    style: {
      display: "flex",
      alignItems: "center",
      justifyContent: "center",
      gap: "0.45rem",
      padding: "0.8rem 0",
      fontSize: "0.82rem",
      fontWeight: tab === t.id ? 600 : 500,
      color: tab === t.id ? "var(--iscc-blue)" : "var(--text-muted)",
      background: "none",
      border: "none",
      borderBottom: `3px solid ${tab === t.id ? "var(--iscc-blue)" : "transparent"}`,
      cursor: "pointer",
      transition: "color .15s, border-color .15s"
    }
  }, /*#__PURE__*/React.createElement(UiIcon, {
    name: t.icon,
    size: 15
  }), /*#__PURE__*/React.createElement("span", null, t.label));
  const goBtn = (label, enabled, onClick) => /*#__PURE__*/React.createElement("button", {
    disabled: !enabled,
    onClick: onClick,
    style: {
      display: "inline-flex",
      alignItems: "center",
      gap: "0.45rem",
      background: "var(--iscc-blue)",
      color: "#fff",
      fontSize: "0.82rem",
      fontWeight: 600,
      padding: "0.55rem 1rem",
      border: "none",
      borderRadius: "0.5rem",
      cursor: enabled ? "pointer" : "not-allowed",
      opacity: enabled ? 1 : 0.45,
      flexShrink: 0
    }
  }, /*#__PURE__*/React.createElement("span", null, label), /*#__PURE__*/React.createElement(UiIcon, {
    name: "arrow-right",
    size: 14
  }));
  const codeValid = codeInput.trim().length >= 12;
  return /*#__PURE__*/React.createElement("div", {
    style: {
      background: "#fff",
      borderRadius: "1rem",
      boxShadow: "var(--shadow-instrument)",
      overflow: "hidden"
    }
  }, /*#__PURE__*/React.createElement("div", {
    style: {
      display: "grid",
      gridTemplateColumns: "1fr 1fr 1fr",
      borderBottom: "1px solid var(--border-subtle)"
    }
  }, TABS.map(tabBtn)), /*#__PURE__*/React.createElement("div", {
    style: {
      padding: "1.35rem 1.5rem 1.5rem"
    }
  }, tab === "file" && /*#__PURE__*/React.createElement("div", {
    onClick: () => onGenerate({
      kind: "file",
      semantic,
      granular
    }),
    onDragOver: e => {
      e.preventDefault();
      setDrag(true);
    },
    onDragLeave: () => setDrag(false),
    onDrop: e => {
      e.preventDefault();
      setDrag(false);
      onGenerate({
        kind: "file",
        semantic,
        granular
      });
    },
    style: {
      border: drag ? "2px solid var(--iscc-blue)" : "2px dashed var(--border-control)",
      borderRadius: "0.65rem",
      background: drag ? "rgba(0,84,178,0.07)" : "var(--surface-inset)",
      padding: "2.1rem 1.25rem",
      display: "flex",
      flexDirection: "column",
      alignItems: "center",
      gap: "0.6rem",
      cursor: "pointer",
      boxShadow: drag ? "0 0 0 4px rgba(0,84,178,0.14)" : "none",
      transition: "border-color .15s, background-color .15s, box-shadow .15s"
    }
  }, /*#__PURE__*/React.createElement("div", {
    style: {
      width: "2.9rem",
      height: "2.9rem",
      borderRadius: "0.65rem",
      background: "rgba(0,84,178,0.08)",
      color: "var(--iscc-blue)",
      display: "flex",
      alignItems: "center",
      justifyContent: "center"
    }
  }, /*#__PURE__*/React.createElement(UiIcon, {
    name: "upload",
    size: 22
  })), /*#__PURE__*/React.createElement("div", {
    style: {
      fontSize: "0.95rem",
      fontWeight: 600,
      color: "var(--text-heading)"
    }
  }, "Drag & drop media files"), /*#__PURE__*/React.createElement("div", {
    style: {
      fontSize: "0.76rem",
      fontWeight: 300,
      color: "var(--text-muted)"
    }
  }, "or ", /*#__PURE__*/React.createElement("span", {
    style: {
      color: "var(--iscc-blue)",
      fontWeight: 500,
      textDecoration: "underline",
      textUnderlineOffset: 2
    }
  }, "browse"), " \u2014 multiple files OK, ISCC generates on add")), tab === "text" && /*#__PURE__*/React.createElement("div", null, /*#__PURE__*/React.createElement("textarea", {
    value: text,
    onChange: e => setText(e.target.value),
    placeholder: "Paste or type plain text \u2014 an article, a post, a chapter\u2026",
    style: {
      width: "100%",
      boxSizing: "border-box",
      height: "7.5rem",
      resize: "none",
      fontSize: "0.82rem",
      fontWeight: 300,
      lineHeight: 1.6,
      padding: "0.75rem 0.9rem",
      border: "1px solid var(--border-control)",
      borderRadius: "0.65rem",
      background: "var(--surface-inset)",
      color: "var(--text-heading)",
      outline: "none",
      display: "block",
      fontFamily: "var(--font-sans)"
    }
  }), /*#__PURE__*/React.createElement("div", {
    style: {
      display: "flex",
      alignItems: "center",
      justifyContent: "space-between",
      gap: "0.75rem",
      marginTop: "0.65rem"
    }
  }, /*#__PURE__*/React.createElement("span", {
    style: {
      fontFamily: "var(--font-mono)",
      fontSize: "0.69rem",
      fontWeight: 300,
      color: "var(--text-faint)"
    }
  }, "UTF-8 \xB7 ", text.length.toLocaleString(), " characters"), goBtn("Generate ISCC", text.trim().length > 0, () => onGenerate({
    kind: "text",
    semantic,
    granular
  })))), tab === "code" && /*#__PURE__*/React.createElement("div", null, /*#__PURE__*/React.createElement("input", {
    value: codeInput,
    onChange: e => setCodeInput(e.target.value),
    spellCheck: false,
    placeholder: "ISCC:KECWN77F73NA44D7BNDLJCRJ3M2YQGCK\u2026",
    style: {
      width: "100%",
      boxSizing: "border-box",
      fontFamily: "var(--font-mono)",
      fontSize: "0.78rem",
      letterSpacing: "-0.01em",
      padding: "0.75rem 0.9rem",
      border: "1px solid var(--border-control)",
      borderRadius: "0.65rem",
      background: "var(--surface-inset)",
      color: "var(--text-heading)",
      outline: "none"
    }
  }), /*#__PURE__*/React.createElement("div", {
    style: {
      display: "flex",
      alignItems: "center",
      justifyContent: "space-between",
      gap: "0.75rem",
      marginTop: "0.65rem"
    }
  }, /*#__PURE__*/React.createElement("div", {
    style: {
      display: "flex",
      alignItems: "center",
      gap: "0.5rem"
    }
  }, /*#__PURE__*/React.createElement("div", {
    style: {
      width: 8,
      height: 8,
      borderRadius: "50%",
      background: codeInput.trim() === "" ? "var(--border-control)" : codeValid ? "var(--iscc-lime-green)" : "var(--iscc-coral-red)"
    }
  }), /*#__PURE__*/React.createElement("span", {
    style: {
      fontFamily: "var(--font-mono)",
      fontSize: "0.69rem",
      fontWeight: 300,
      color: "var(--text-faint)"
    }
  }, codeInput.trim() === "" ? "validates as you type — prefix optional" : codeValid ? "ready to decode" : "not a valid ISCC yet")), goBtn("Decode code", codeValid, () => onGenerate({
    kind: "code",
    semantic: false,
    granular: false
  }))), /*#__PURE__*/React.createElement("div", {
    style: {
      marginTop: "0.85rem",
      background: "var(--surface-inset)",
      border: "1px solid var(--border-subtle)",
      borderRadius: "0.65rem",
      padding: "0.75rem 0.9rem",
      fontSize: "0.76rem",
      fontWeight: 300,
      lineHeight: 1.55,
      color: "var(--text-muted)"
    }
  }, "No upload needed \u2014 the code itself carries its units. You get the same readout: unit fields, bits and per-layer explanations.")), tab !== "code" && /*#__PURE__*/React.createElement("div", {
    style: {
      marginTop: "1.1rem",
      display: "flex",
      flexDirection: "column",
      gap: "0.75rem"
    }
  }, /*#__PURE__*/React.createElement(ToggleRow, {
    checked: semantic,
    onChange: setSemantic,
    label: "Semantic code",
    experimental: true,
    desc: "Adds a 5th unit from ML embeddings of meaning. Slower; not a plain ISO 24138 code."
  }), /*#__PURE__*/React.createElement(ToggleRow, {
    checked: granular,
    onChange: setGranular,
    label: "Granular simprints",
    desc: "Per-chunk fingerprints of text \u2014 enables matching fragments inside a document."
  }))));
}
function ToggleRow({
  checked,
  onChange,
  label,
  desc,
  experimental
}) {
  return /*#__PURE__*/React.createElement("div", {
    style: {
      display: "flex",
      alignItems: "flex-start",
      gap: "0.75rem"
    }
  }, /*#__PURE__*/React.createElement(Switch, {
    checked: checked,
    onChange: onChange
  }), /*#__PURE__*/React.createElement("div", null, /*#__PURE__*/React.createElement("div", {
    style: {
      display: "flex",
      alignItems: "center",
      gap: "0.5rem",
      fontSize: "0.82rem",
      fontWeight: 600,
      color: "var(--text-heading)"
    }
  }, /*#__PURE__*/React.createElement("span", null, label), experimental && /*#__PURE__*/React.createElement("span", {
    style: {
      fontSize: "0.62rem",
      fontWeight: 600,
      letterSpacing: "0.06em",
      color: "#7a5c00",
      background: "rgba(255,195,0,0.25)",
      border: "1px solid rgba(255,195,0,0.6)",
      padding: "0.1rem 0.45rem",
      borderRadius: "9999px"
    }
  }, "EXPERIMENTAL")), /*#__PURE__*/React.createElement("div", {
    style: {
      fontSize: "0.76rem",
      fontWeight: 300,
      color: "var(--text-muted)",
      marginTop: "0.1rem"
    }
  }, desc)));
}

/** Minimal switch (the kit re-implements it locally to stay self-contained). */
function Switch({
  checked,
  onChange
}) {
  return /*#__PURE__*/React.createElement("button", {
    type: "button",
    role: "switch",
    "aria-checked": checked,
    onClick: () => onChange(!checked),
    style: {
      position: "relative",
      width: 38,
      height: 22,
      flexShrink: 0,
      borderRadius: "9999px",
      border: "none",
      padding: 0,
      cursor: "pointer",
      background: checked ? "var(--iscc-blue)" : "var(--border-control)",
      transition: "background-color .18s",
      marginTop: "1px"
    }
  }, /*#__PURE__*/React.createElement("span", {
    style: {
      position: "absolute",
      top: 2,
      left: checked ? 18 : 2,
      width: 18,
      height: 18,
      borderRadius: "50%",
      background: "#fff",
      boxShadow: "0 1px 3px rgba(0,0,0,0.25)",
      transition: "left .18s"
    }
  }));
}
function Hero({
  onGenerate
}) {
  return /*#__PURE__*/React.createElement("section", {
    className: "iscc-grain",
    style: {
      background: "var(--iscc-blue)",
      padding: "2.75rem 0 3.25rem"
    }
  }, /*#__PURE__*/React.createElement("div", {
    style: {
      maxWidth: "var(--container-max)",
      margin: "0 auto",
      padding: "0 1.5rem"
    }
  }, /*#__PURE__*/React.createElement("div", {
    className: "hero-grid",
    style: {
      display: "grid",
      gridTemplateColumns: "1.15fr 1fr",
      gap: "3rem",
      alignItems: "start"
    }
  }, /*#__PURE__*/React.createElement("div", null, /*#__PURE__*/React.createElement(IntakeCard, {
    onGenerate: onGenerate
  })), /*#__PURE__*/React.createElement("div", {
    style: {
      color: "#fff"
    }
  }, /*#__PURE__*/React.createElement("div", {
    style: {
      display: "inline-flex",
      alignItems: "center",
      gap: "0.5rem",
      background: "rgba(255,255,255,0.12)",
      border: "1px solid rgba(255,255,255,0.25)",
      color: "#fff",
      padding: "0.35rem 0.8rem",
      borderRadius: "9999px",
      fontSize: "0.75rem",
      fontWeight: 500,
      marginBottom: "1.1rem"
    }
  }, /*#__PURE__*/React.createElement(UiIcon, {
    name: "check-circle",
    size: 13
  }), /*#__PURE__*/React.createElement("span", null, "ISO 24138:2024")), /*#__PURE__*/React.createElement("h1", {
    style: {
      margin: "0 0 1rem",
      fontSize: "clamp(1.8rem, 3.4vw, 2.5rem)",
      fontWeight: 700,
      lineHeight: 1.12,
      letterSpacing: "-0.015em",
      color: "#fff"
    }
  }, "The ", /*#__PURE__*/React.createElement("span", {
    style: {
      color: "var(--iscc-bright-yellow)"
    }
  }, "DNA"), " of your digital content", /*#__PURE__*/React.createElement("span", {
    style: {
      color: "var(--iscc-bright-yellow)"
    }
  }, ".")), /*#__PURE__*/React.createElement("p", {
    style: {
      margin: "0 0 1.35rem",
      color: "rgba(255,255,255,0.94)",
      fontSize: "0.94rem",
      fontWeight: 300,
      lineHeight: 1.6,
      maxWidth: "26rem"
    }
  }, "An ISCC is a fingerprint generated ", /*#__PURE__*/React.createElement("b", {
    style: {
      fontWeight: 600
    }
  }, "from the content itself"), " \u2014 no registry, no signup. Drop a file and read its code layer by layer."), /*#__PURE__*/React.createElement("ul", {
    style: {
      listStyle: "none",
      margin: 0,
      padding: 0,
      display: "flex",
      flexDirection: "column",
      gap: "0.55rem"
    }
  }, /*#__PURE__*/React.createElement(HeroPoint, {
    icon: "shield",
    color: "var(--iscc-lime-green)"
  }, "Files stay private to you and are ", /*#__PURE__*/React.createElement("b", {
    style: {
      fontWeight: 500,
      color: "#fff"
    }
  }, "auto-deleted after one hour")), /*#__PURE__*/React.createElement(HeroPoint, {
    icon: "image",
    color: "var(--iscc-light-cyan)"
  }, "Text, image, audio & video \u2014 anything else still gets a Data + Instance code"))))));
}
function HeroPoint({
  icon,
  color,
  children
}) {
  return /*#__PURE__*/React.createElement("li", {
    style: {
      display: "flex",
      gap: "0.55rem",
      alignItems: "flex-start",
      color: "rgba(255,255,255,0.85)",
      fontSize: "0.82rem",
      fontWeight: 300
    }
  }, /*#__PURE__*/React.createElement("span", {
    style: {
      color,
      marginTop: 2
    }
  }, /*#__PURE__*/React.createElement(UiIcon, {
    name: icon,
    size: 15,
    strokeWidth: 2.2
  })), /*#__PURE__*/React.createElement("span", null, children));
}
function AppFooter() {
  const legal = ["Privacy Policy", "Cookie Policy", "Imprint", "Disclaimer"];
  return /*#__PURE__*/React.createElement("footer", {
    className: "iscc-grain",
    style: {
      background: "var(--iscc-deep-navy)",
      marginTop: "auto"
    }
  }, /*#__PURE__*/React.createElement("div", {
    style: {
      maxWidth: "var(--container-max)",
      margin: "0 auto",
      padding: "0.9rem 1.5rem",
      display: "flex",
      alignItems: "center",
      justifyContent: "space-between",
      gap: "1.5rem",
      flexWrap: "wrap"
    }
  }, /*#__PURE__*/React.createElement("div", {
    style: {
      display: "flex",
      alignItems: "center",
      gap: "0.9rem"
    }
  }, /*#__PURE__*/React.createElement("img", {
    src: "../../assets/iscc-logo-white.png",
    alt: "ISCC",
    style: {
      height: "1.1rem",
      opacity: 0.9
    }
  }), /*#__PURE__*/React.createElement("span", {
    style: {
      color: "rgba(255,255,255,0.55)",
      fontSize: "0.8rem",
      fontWeight: 300
    }
  }, "Copyright \xA9 2022\u20132026 ISCC Foundation")), /*#__PURE__*/React.createElement("div", {
    style: {
      display: "flex",
      alignItems: "center",
      gap: "1.5rem",
      flexWrap: "wrap"
    }
  }, legal.map(l => /*#__PURE__*/React.createElement("a", {
    key: l,
    href: "#",
    style: {
      color: "rgba(255,255,255,0.7)",
      fontSize: "0.8rem",
      fontWeight: 300,
      textDecoration: "none"
    }
  }, l)), /*#__PURE__*/React.createElement("div", {
    style: {
      width: 1,
      height: "0.9rem",
      background: "rgba(255,255,255,0.25)"
    }
  }), /*#__PURE__*/React.createElement("a", {
    href: "#",
    style: {
      display: "inline-flex",
      alignItems: "center",
      gap: "0.4rem",
      color: "rgba(255,255,255,0.85)",
      fontSize: "0.8rem",
      fontWeight: 500,
      textDecoration: "none"
    }
  }, /*#__PURE__*/React.createElement(UiIcon, {
    name: "github",
    size: 13
  }), /*#__PURE__*/React.createElement("span", null, "Source Code")))));
}
function EducationStrip() {
  const SAMPLE = "ISCC:KED572P4AOF5K6QXQA4T6OJD5UGX7UBPFW2TVQNTHBCKFRFCANCZARQ4K6NSFZQSH4GQ";
  const CARDS = [{
    kind: "meta",
    name: "Meta-Code",
    code: "META-NONE-V0",
    compares: "Metadata similarity",
    from: "From embedded title & description"
  }, {
    kind: "semantic",
    name: "Semantic-Code",
    code: "SEMANTIC-V0",
    compares: "Semantic similarity",
    from: "From ML embeddings of meaning — experimental"
  }, {
    kind: "content",
    name: "Content-Code",
    code: "CONTENT-IMAGE-V0",
    compares: "Syntactic similarity",
    from: "From perceptual features per media type"
  }, {
    kind: "data",
    name: "Data-Code",
    code: "DATA-NONE-V0",
    compares: "Data similarity",
    from: "From the raw bitstream"
  }, {
    kind: "instance",
    name: "Instance-Code",
    code: "INSTANCE-V0",
    compares: "Exact match",
    from: "Cryptographic checksum — data integrity"
  }];
  return /*#__PURE__*/React.createElement("section", {
    style: {
      background: "#fff",
      borderRadius: "0.75rem",
      padding: "2.2rem 2.2rem 2.4rem",
      boxShadow: "var(--shadow-raised)"
    }
  }, /*#__PURE__*/React.createElement("div", {
    style: {
      display: "flex",
      alignItems: "baseline",
      justifyContent: "space-between",
      gap: "1.5rem",
      marginBottom: "1.3rem",
      flexWrap: "wrap"
    }
  }, /*#__PURE__*/React.createElement("h2", {
    style: {
      margin: 0,
      fontSize: "1.3rem",
      fontWeight: 600,
      color: "var(--iscc-deep-navy)"
    }
  }, "What you'll get \u2014 one code, five layers"), /*#__PURE__*/React.createElement("span", {
    style: {
      fontSize: "0.82rem",
      fontWeight: 300,
      color: "var(--text-muted)"
    }
  }, "Each unit is a similarity fingerprint of a different layer of your content.")), /*#__PURE__*/React.createElement("div", {
    className: "iscc-grain",
    style: {
      background: "var(--iscc-coral-red)",
      borderRadius: "0.5rem",
      padding: "0.8rem 1.1rem",
      fontFamily: "var(--font-mono)",
      fontSize: "clamp(7px, 1.25vw, 16px)",
      fontWeight: 300,
      color: "#fff",
      letterSpacing: "-0.01em",
      textAlign: "center",
      whiteSpace: "nowrap",
      overflow: "hidden"
    }
  }, SAMPLE), /*#__PURE__*/React.createElement("div", {
    style: {
      display: "flex",
      justifyContent: "center",
      margin: "0.25rem 0",
      color: "var(--text-faint)"
    }
  }, /*#__PURE__*/React.createElement(UiIcon, {
    name: "arrow-down",
    size: 18
  })), /*#__PURE__*/React.createElement("div", {
    className: "edu-grid",
    style: {
      display: "grid",
      gridTemplateColumns: "repeat(5, 1fr)",
      gap: "0.75rem"
    }
  }, CARDS.map(c => /*#__PURE__*/React.createElement("div", {
    key: c.kind,
    style: {
      borderRadius: "0.5rem",
      overflow: "hidden",
      border: "1px solid var(--border-subtle)",
      background: "#fff"
    }
  }, /*#__PURE__*/React.createElement("div", {
    style: {
      height: 6,
      background: UNIT_STYLE[c.kind].color
    }
  }), /*#__PURE__*/React.createElement("div", {
    style: {
      padding: "0.55rem 0.75rem",
      background: UNIT_STYLE[c.kind].tint
    }
  }, /*#__PURE__*/React.createElement("div", {
    style: {
      fontSize: "0.78rem",
      fontWeight: 600,
      color: "var(--iscc-deep-navy)"
    }
  }, c.name), /*#__PURE__*/React.createElement("div", {
    style: {
      fontFamily: "var(--font-mono)",
      fontSize: "0.62rem",
      color: "rgba(18,54,99,0.65)",
      marginTop: "0.1rem"
    }
  }, c.code)), /*#__PURE__*/React.createElement("div", {
    style: {
      padding: "0.6rem 0.75rem"
    }
  }, /*#__PURE__*/React.createElement("div", {
    style: {
      fontSize: "0.74rem",
      fontWeight: 500,
      color: "var(--text-body)"
    }
  }, c.compares), /*#__PURE__*/React.createElement("div", {
    style: {
      fontSize: "0.71rem",
      fontWeight: 300,
      color: "var(--text-muted)",
      marginTop: "0.2rem",
      lineHeight: 1.45
    }
  }, c.from))))), /*#__PURE__*/React.createElement("div", {
    style: {
      marginTop: "1.1rem",
      display: "flex",
      alignItems: "center",
      gap: "0.9rem"
    }
  }, /*#__PURE__*/React.createElement("span", {
    style: {
      fontSize: "0.69rem",
      fontWeight: 500,
      letterSpacing: "0.1em",
      color: "var(--text-muted)",
      textTransform: "uppercase"
    }
  }, "Abstract & persistent"), /*#__PURE__*/React.createElement("div", {
    style: {
      flex: 1,
      height: 4,
      borderRadius: 2,
      background: "var(--iscc-bright-yellow)"
    }
  }), /*#__PURE__*/React.createElement("span", {
    style: {
      fontSize: "0.69rem",
      fontWeight: 500,
      letterSpacing: "0.1em",
      color: "var(--text-muted)",
      textTransform: "uppercase"
    }
  }, "Concrete & volatile")), /*#__PURE__*/React.createElement("div", {
    style: {
      marginTop: "1.75rem",
      background: "var(--surface-inset)",
      border: "1px solid var(--border-subtle)",
      borderRadius: "0.65rem",
      padding: "1rem 1.25rem",
      display: "flex",
      alignItems: "center",
      justifyContent: "space-between",
      gap: "1.25rem",
      flexWrap: "wrap"
    }
  }, /*#__PURE__*/React.createElement("div", {
    style: {
      display: "flex",
      alignItems: "center",
      gap: "0.9rem"
    }
  }, /*#__PURE__*/React.createElement("div", {
    style: {
      width: "2.4rem",
      height: "2.4rem",
      borderRadius: "0.5rem",
      background: "rgba(0,84,178,0.08)",
      color: "var(--iscc-blue)",
      display: "flex",
      alignItems: "center",
      justifyContent: "center",
      flexShrink: 0
    }
  }, /*#__PURE__*/React.createElement(UiIcon, {
    name: "code",
    size: 18
  })), /*#__PURE__*/React.createElement("div", null, /*#__PURE__*/React.createElement("div", {
    style: {
      fontSize: "0.88rem",
      fontWeight: 600,
      color: "var(--text-heading)"
    }
  }, "Everything on this page is one REST call away"), /*#__PURE__*/React.createElement("div", {
    style: {
      fontFamily: "var(--font-mono)",
      fontSize: "0.72rem",
      fontWeight: 300,
      color: "var(--text-muted)",
      marginTop: "0.2rem"
    }
  }, "POST /api/v1/iscc \xB7 GET /api/v1/explain/{iscc} \xB7 POST /api/v1/simprint"))), /*#__PURE__*/React.createElement("a", {
    href: "#",
    style: {
      display: "inline-flex",
      alignItems: "center",
      gap: "0.45rem",
      background: "var(--iscc-blue)",
      color: "#fff",
      fontSize: "0.82rem",
      fontWeight: 600,
      textDecoration: "none",
      borderRadius: "0.5rem",
      padding: "0.6rem 1rem"
    }
  }, "Open API docs")));
}
Object.assign(window, {
  AppHeader,
  AppFooter,
  Hero,
  EducationStrip
});
})(); } catch (e) { __ds_ns.__errors.push({ path: "ui_kits/generator/chrome.jsx", error: String((e && e.message) || e) }); }

// ui_kits/generator/generator-lib.jsx
try { (() => {
/* Generator UI kit — shared library: icons, the ISCC-UNIT palette, deterministic
   code/bit helpers and per-layer copy. Lifted from iscc/iscc-web frontend.
   Exposes everything on window for the other babel scripts. */

const ICONS = {
  upload: '<path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4M17 8l-5-5-5 5M12 3v12"/>',
  download: '<path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4M7 10l5 5 5-5M12 15V3"/>',
  trash: '<path d="M3 6h18M8 6V4a1 1 0 011-1h6a1 1 0 011 1v2m3 0v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6"/>',
  copy: '<rect x="9" y="9" width="13" height="13" rx="2"/><path d="M5 15H4a2 2 0 01-2-2V4a2 2 0 012-2h9a2 2 0 012 2v1"/>',
  check: '<path d="M20 6L9 17l-5-5"/>',
  "check-circle": '<path d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"/>',
  x: '<path d="M18 6L6 18M6 6l12 12"/>',
  code: '<path d="M16 18l6-6-6-6M8 6l-6 6 6 6"/>',
  compare: '<path d="M8 3L4 7l4 4M4 7h16M16 21l4-4-4-4M20 17H4"/>',
  "arrow-right": '<path d="M5 12h14M12 5l7 7-7 7"/>',
  "arrow-down": '<path d="M12 5v14M5 12l7 7 7-7"/>',
  "chevron-right": '<path d="M9 18l6-6-6-6"/>',
  shield: '<path d="M12 2l8 4v6c0 5-3.5 8.5-8 10-4.5-1.5-8-5-8-10V6l8-4z"/>',
  image: '<rect x="3" y="3" width="18" height="18" rx="2"/><circle cx="8.5" cy="8.5" r="1.5"/><path d="M21 15l-5-5L5 21"/>',
  type: '<path d="M4 7V4h16v3M9 20h6M12 4v16"/>',
  search: '<circle cx="11" cy="11" r="8"/><path d="M21 21l-4.35-4.35"/>',
  file: '<path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"/><path d="M14 2v6h6"/>',
  film: '<rect x="2" y="2" width="20" height="20" rx="2.2"/><path d="M7 2v20M17 2v20M2 12h20M2 7h5M2 17h5M17 17h5M17 7h5"/>',
  music: '<path d="M9 18V5l12-2v13"/><circle cx="6" cy="18" r="3"/><circle cx="18" cy="16" r="3"/>',
  github: '<path d="M9 19c-5 1.5-5-2.5-7-3m14 6v-3.87a3.37 3.37 0 00-.94-2.61c3.14-.35 6.44-1.54 6.44-7A5.44 5.44 0 0020 4.77 5.07 5.07 0 0019.91 1S18.73.65 16 2.48a13.38 13.38 0 00-7 0C6.27.65 5.09 1 5.09 1A5.07 5.07 0 005 4.77a5.44 5.44 0 00-1.5 3.78c0 5.42 3.3 6.61 6.44 7A3.37 3.37 0 009 18.13V22"/>',
  plus: '<path d="M12 5v14M5 12h14"/>',
  alert: '<circle cx="12" cy="12" r="9"/><path d="M12 8v4M12 16h.01"/>'
};

/** Feather-style stroke icon (24×24 viewBox). */
function UiIcon({
  name,
  size = 16,
  strokeWidth = 1.8,
  style
}) {
  return /*#__PURE__*/React.createElement("svg", {
    width: size,
    height: size,
    viewBox: "0 0 24 24",
    fill: "none",
    stroke: "currentColor",
    strokeWidth: strokeWidth,
    strokeLinecap: "round",
    strokeLinejoin: "round",
    style: {
      display: "inline-block",
      flexShrink: 0,
      ...style
    },
    dangerouslySetInnerHTML: {
      __html: ICONS[name] || ""
    }
  });
}
const UNIT_STYLE = {
  meta: {
    label: "META",
    color: "#7ac2f7",
    accent: "#1f86c9",
    tint: "rgba(122,194,247,0.18)",
    meaning: "metadata similarity"
  },
  semantic: {
    label: "SEMANTIC",
    color: "#4596f5",
    accent: "#2e7ad6",
    tint: "rgba(69,150,245,0.13)",
    meaning: "semantic similarity · experimental"
  },
  content: {
    label: "CONTENT",
    color: "#0054b2",
    accent: "#0054b2",
    tint: "rgba(0,84,178,0.10)",
    meaning: "syntactic similarity"
  },
  data: {
    label: "DATA",
    color: "#123663",
    accent: "#123663",
    tint: "rgba(18,54,99,0.10)",
    meaning: "data similarity"
  },
  instance: {
    label: "INSTANCE",
    color: "#a6db50",
    accent: "#6d9c2a",
    tint: "rgba(166,219,80,0.18)",
    meaning: "exact match · checksum"
  }
};
const BASE32 = "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567";

/** Deterministic pseudo-random base32 string driving the decode animation. */
function scramble(length, seed) {
  let state = seed * 2654435761 + 1013904223 >>> 0;
  let out = "";
  for (let i = 0; i < length; i++) {
    state = state * 1103515245 + 12345 >>> 0;
    out += BASE32[(state >>> 16) % 32];
  }
  return out;
}

/** Deterministic '0'/'1' bit texture from a string seed. */
function seedBits(seed, length = 64) {
  let state = 0;
  for (let i = 0; i < seed.length; i++) state = state * 31 + seed.charCodeAt(i) >>> 0;
  let out = "";
  for (let i = 0; i < length; i++) {
    state = state * 1103515245 + 12345 >>> 0;
    out += state >>> 16 & 1 ? "1" : "0";
  }
  return out;
}
function formatBytes(bytes) {
  if (!Number.isFinite(bytes) || bytes < 0) return "";
  if (bytes < 1024) return `${bytes} B`;
  let value = bytes,
    unit = "B";
  for (const next of ["KB", "MB", "GB", "TB"]) {
    if (value < 1024) break;
    value /= 1024;
    unit = next;
  }
  return `${value >= 100 ? value.toFixed(0) : value.toFixed(value >= 10 ? 1 : 2)} ${unit}`;
}

/** Per-layer explanation copy (condensed from iscc-web unit-copy.ts). */
const UNIT_COPY = {
  meta: {
    title: "Meta-Code — what the file says about itself",
    body: "A similarity hash over the embedded title and description — the metadata, not the content. Renaming the file changes nothing; editing its embedded title moves these bits.",
    tip: "Add a title and description and re-decode — watch how only the META field changes."
  },
  semantic: {
    title: "Semantic-Code — what the work means",
    body: "Derived from ML embeddings of meaning — not wording or pixels. A translation or paraphrase keeps these bits close while the Content-Code drifts. Not part of plain ISO 24138.",
    tip: "Drop a paraphrased copy and compare: Semantic stays close while Content moves."
  },
  content: {
    title: "Content-Code — what the work looks like",
    body: "Derived from perceptual features of the decoded content — not the bytes. Re-encoding, format conversion or light edits barely move these bits.",
    tip: "Drop a re-encoded copy and compare — watch how few CONTENT bits flip."
  },
  data: {
    title: "Data-Code — how the file is stored",
    body: "A similarity hash over the raw bitstream, independent of media type. It tracks storage, not meaning, and is robust against small inserts and deletes.",
    tip: "Save the same content in another format: Data diverges while Content holds."
  },
  instance: {
    title: "Instance-Code — the exact bytes",
    body: "A cryptographic checksum of the file as-is. One flipped bit anywhere changes it completely — there is no “similar” here, only identical or not.",
    tip: "Change a single byte and re-decode: this checksum flips entirely while Content barely moves."
  }
};

/** Fabricate a plausible decoded specimen for the demo. */
function fakeExplain(seed, kinds) {
  const units = kinds.map((kind, i) => ({
    kind,
    code: scramble(16, seed * 17 + i * 101),
    bits: seedBits(`${seed}-${kind}`, 64)
  }));
  const iscc = "ISCC:" + scramble(kinds.length >= 5 ? 68 : 64, seed * 7);
  return {
    iscc,
    units
  };
}
Object.assign(window, {
  ICONS,
  UiIcon,
  UNIT_STYLE,
  scramble,
  seedBits,
  formatBytes,
  UNIT_COPY,
  fakeExplain
});
})(); } catch (e) { __ds_ns.__errors.push({ path: "ui_kits/generator/generator-lib.jsx", error: String((e && e.message) || e) }); }

// ui_kits/generator/result.jsx
try { (() => {
/* Generator UI kit — the decoder ResultCard: card head, navy readout strip with
   selectable ISCC-UNIT fields and bit strips, and the per-layer detail panel. */

function UnitField({
  kind,
  code,
  bits,
  selected,
  state,
  onSelect
}) {
  const s = UNIT_STYLE[kind];
  const ZERO = "rgba(255,255,255,0.25)";
  const QUEUED = "rgba(255,255,255,0.08)";
  let cells;
  if (state === "queued") cells = Array.from({
    length: 64
  }, () => ({
    color: QUEUED,
    anim: false
  }));else if (state === "hashing") cells = Array.from(seedBits(`noise-${kind}`), b => ({
    color: b === "1" ? s.color : ZERO,
    anim: true
  }));else cells = bits ? Array.from(bits, b => ({
    color: b === "1" ? s.color : ZERO,
    anim: false
  })) : [];
  const tag = state === "queued" ? "queued" : state === "hashing" ? "hashing…" : selected ? "64 bit · selected" : "64 bit";
  return /*#__PURE__*/React.createElement("button", {
    type: "button",
    disabled: state !== "ready",
    "aria-pressed": selected,
    onClick: onSelect,
    style: {
      display: "block",
      width: "100%",
      textAlign: "left",
      background: "rgba(255,255,255,0.07)",
      border: `2px solid ${selected ? "var(--iscc-bright-yellow)" : "rgba(255,255,255,0.16)"}`,
      borderRadius: "0.65rem",
      padding: "0.8rem 0.9rem 0.85rem",
      cursor: state === "ready" ? "pointer" : "default",
      boxShadow: selected ? "0 0 0 3px rgba(255,195,0,0.3), 0 8px 20px rgba(0,0,0,0.35)" : "none",
      transition: "border-color .15s, box-shadow .15s, transform .15s"
    }
  }, /*#__PURE__*/React.createElement("div", {
    style: {
      display: "flex",
      alignItems: "center",
      justifyContent: "space-between",
      gap: "0.5rem"
    }
  }, /*#__PURE__*/React.createElement("div", {
    style: {
      display: "flex",
      alignItems: "center",
      gap: "0.45rem"
    }
  }, /*#__PURE__*/React.createElement("span", {
    style: {
      width: 10,
      height: 10,
      borderRadius: 3,
      background: s.color,
      border: "1px solid rgba(255,255,255,0.5)",
      flexShrink: 0
    }
  }), /*#__PURE__*/React.createElement("span", {
    style: {
      fontFamily: "var(--font-mono)",
      fontSize: "0.69rem",
      fontWeight: 700,
      letterSpacing: "0.12em",
      color: "#fff"
    }
  }, s.label)), /*#__PURE__*/React.createElement("span", {
    style: {
      fontFamily: "var(--font-mono)",
      fontSize: "0.62rem",
      color: state === "hashing" ? "var(--iscc-bright-yellow)" : "rgba(255,255,255,0.6)",
      whiteSpace: "nowrap"
    }
  }, tag)), /*#__PURE__*/React.createElement("div", {
    style: {
      fontFamily: "var(--font-mono)",
      fontSize: "0.69rem",
      color: "rgba(255,255,255,0.85)",
      marginTop: "0.4rem",
      letterSpacing: "0.01em",
      whiteSpace: "nowrap",
      overflow: "hidden",
      textOverflow: "ellipsis"
    }
  }, code), cells.length > 0 && /*#__PURE__*/React.createElement("div", {
    style: {
      display: "flex",
      gap: 1,
      marginTop: "0.45rem"
    }
  }, cells.map((c, i) => /*#__PURE__*/React.createElement("div", {
    key: i,
    style: {
      width: "100%",
      maxWidth: 6,
      height: 18,
      borderRadius: 1,
      flexShrink: 1,
      background: c.color,
      animation: c.anim ? `iscc-bit-flick ${(0.35 + i * 53 % 50 / 100).toFixed(2)}s steps(2,end) ${(-(i * 37 % 80) / 100).toFixed(2)}s infinite` : "none"
    }
  }))), /*#__PURE__*/React.createElement("div", {
    style: {
      fontSize: "0.66rem",
      fontWeight: 300,
      color: "rgba(255,255,255,0.65)",
      marginTop: "0.5rem"
    }
  }, s.meaning));
}
function UnitDetail({
  kind
}) {
  const s = UNIT_STYLE[kind];
  const c = UNIT_COPY[kind];
  return /*#__PURE__*/React.createElement("div", {
    style: {
      padding: "1.25rem 1.75rem",
      borderTop: "1px solid var(--border-subtle)"
    }
  }, /*#__PURE__*/React.createElement("div", {
    style: {
      display: "flex",
      alignItems: "center",
      gap: "0.6rem",
      marginBottom: "0.6rem"
    }
  }, /*#__PURE__*/React.createElement("span", {
    style: {
      width: 12,
      height: 12,
      borderRadius: 3,
      background: s.color
    }
  }), /*#__PURE__*/React.createElement("span", {
    style: {
      fontWeight: 600,
      fontSize: "0.95rem",
      color: "var(--text-heading)"
    }
  }, c.title)), /*#__PURE__*/React.createElement("p", {
    style: {
      margin: "0 0 0.7rem",
      fontSize: "0.84rem",
      fontWeight: 300,
      lineHeight: 1.6,
      color: "var(--text-body)"
    }
  }, c.body), /*#__PURE__*/React.createElement("div", {
    style: {
      display: "flex",
      gap: "0.55rem",
      alignItems: "flex-start",
      background: s.tint,
      border: `1px solid ${s.color}`,
      borderRadius: "0.5rem",
      padding: "0.6rem 0.8rem"
    }
  }, /*#__PURE__*/React.createElement("span", {
    style: {
      color: s.accent,
      marginTop: 1
    }
  }, /*#__PURE__*/React.createElement(UiIcon, {
    name: "arrow-right",
    size: 13,
    strokeWidth: 2.4
  })), /*#__PURE__*/React.createElement("span", {
    style: {
      fontSize: "0.78rem",
      fontWeight: 400,
      color: "var(--text-body)"
    }
  }, /*#__PURE__*/React.createElement("b", {
    style: {
      fontWeight: 600
    }
  }, "Try it."), " ", c.tip)));
}
function ResultCard({
  specimen,
  onRemove
}) {
  const working = specimen.status === "uploading" || specimen.status === "processing";
  const [tick, setTick] = React.useState(0);
  const [selected, setSelected] = React.useState("content");
  const [rawOpen, setRawOpen] = React.useState(false);
  const [copied, setCopied] = React.useState(false);
  React.useEffect(() => {
    if (!working) return;
    const t = setInterval(() => setTick(x => x + 1), 120);
    return () => clearInterval(t);
  }, [working]);
  const pendingKinds = specimen.semantic ? ["meta", "semantic", "content", "data", "instance"] : ["meta", "content", "data", "instance"];
  const units = specimen.explain ? specimen.explain.units : [];
  const codeText = working ? `ISCC:${scramble(specimen.semantic ? 68 : 64, tick)}` : specimen.explain ? specimen.explain.iscc : "";
  const pill = (() => {
    if (specimen.status === "uploading") return {
      tone: "info",
      text: `UPLOADING · ${specimen.progress}%`,
      pulse: true
    };
    if (specimen.status === "processing") return {
      tone: "info",
      text: specimen.kind === "code" ? "DECODING CODE…" : "DECODING · GENERATING ISCC-UNITS…",
      pulse: true
    };
    if (specimen.status === "done") return {
      tone: "success",
      text: specimen.kind === "code" ? "DECODED" : `DONE IN ${specimen.elapsed.toFixed(1)} S`
    };
    return {
      tone: "error",
      text: "FAILED"
    };
  })();
  const copy = async () => {
    try {
      await navigator.clipboard?.writeText(specimen.explain.iscc);
      setCopied(true);
      setTimeout(() => setCopied(false), 1600);
    } catch (e) {}
  };
  const unitsLabel = working ? "ISCC-UNITS · SELF-DESCRIBING · DECODING…" : `ISCC-UNITS · ${units.length} OF ${units.length} · CLICK A FIELD TO INSPECT ITS LAYER`;
  return /*#__PURE__*/React.createElement("article", {
    style: {
      background: "#fff",
      border: "1px solid var(--border-subtle)",
      borderRadius: "0.75rem",
      boxShadow: "var(--shadow-card)",
      overflow: "hidden"
    }
  }, /*#__PURE__*/React.createElement("div", {
    style: {
      padding: "1.1rem 1.75rem 1rem",
      display: "flex",
      alignItems: "center",
      justifyContent: "space-between",
      gap: "1rem",
      flexWrap: "wrap"
    }
  }, /*#__PURE__*/React.createElement("div", {
    style: {
      display: "flex",
      alignItems: "center",
      gap: "0.8rem",
      minWidth: 0
    }
  }, /*#__PURE__*/React.createElement("div", {
    style: {
      width: "2.4rem",
      height: "2.4rem",
      borderRadius: "0.5rem",
      background: "var(--surface-inset)",
      border: "1px solid var(--border-default)",
      color: "var(--iscc-deep-navy)",
      display: "flex",
      alignItems: "center",
      justifyContent: "center",
      flexShrink: 0
    }
  }, /*#__PURE__*/React.createElement(UiIcon, {
    name: specimen.glyph,
    size: 18
  })), /*#__PURE__*/React.createElement("div", {
    style: {
      minWidth: 0
    }
  }, /*#__PURE__*/React.createElement("div", {
    style: {
      fontSize: "0.92rem",
      fontWeight: 600,
      color: "var(--text-heading)",
      wordBreak: "break-all"
    }
  }, specimen.label), /*#__PURE__*/React.createElement("div", {
    style: {
      fontFamily: "var(--font-mono)",
      fontSize: "0.69rem",
      fontWeight: 300,
      color: "var(--text-muted)",
      marginTop: "0.1rem"
    }
  }, specimen.facts))), /*#__PURE__*/React.createElement("div", {
    style: {
      display: "flex",
      alignItems: "center",
      gap: "0.6rem",
      flexWrap: "wrap"
    }
  }, specimen.semantic && specimen.kind !== "code" && /*#__PURE__*/React.createElement("span", {
    style: {
      fontSize: "0.66rem",
      fontWeight: 600,
      letterSpacing: "0.04em",
      color: "#7a5c00",
      background: "rgba(255,195,0,0.16)",
      border: "1px solid rgba(255,195,0,0.55)",
      borderRadius: "9999px",
      padding: "0.3rem 0.7rem"
    }
  }, "SEMANTIC ON"), /*#__PURE__*/React.createElement(StatusPill, pill), /*#__PURE__*/React.createElement("button", {
    onClick: onRemove,
    title: "Remove",
    style: {
      width: "2rem",
      height: "2rem",
      borderRadius: "0.5rem",
      border: "1px solid var(--border-default)",
      background: "none",
      color: "#495057",
      display: "inline-flex",
      alignItems: "center",
      justifyContent: "center",
      cursor: "pointer"
    }
  }, /*#__PURE__*/React.createElement(UiIcon, {
    name: "trash",
    size: 15
  })))), /*#__PURE__*/React.createElement("div", {
    className: "iscc-grain",
    style: {
      background: "var(--iscc-deep-navy)",
      padding: "1.25rem 1.75rem 1.5rem",
      position: "relative",
      overflow: "hidden"
    }
  }, specimen.status === "processing" && /*#__PURE__*/React.createElement("div", {
    style: {
      position: "absolute",
      left: 0,
      right: 0,
      top: 0,
      height: 2,
      background: "linear-gradient(to right, transparent, #7ac2f7 30%, #7ac2f7 70%, transparent)",
      boxShadow: "0 0 14px 3px rgba(122,194,247,0.45)",
      animation: "iscc-scan-sweep 2.1s ease-in-out infinite alternate",
      zIndex: 4
    }
  }), /*#__PURE__*/React.createElement("div", {
    style: {
      display: "flex",
      alignItems: "center",
      gap: "0.9rem"
    }
  }, /*#__PURE__*/React.createElement("div", {
    className: "iscc-grain",
    style: {
      flex: 1,
      minWidth: 0,
      background: "var(--iscc-coral-red)",
      borderRadius: "0.5rem",
      padding: "0.7rem 1rem",
      fontFamily: "var(--font-mono)",
      fontSize: "clamp(7px, 1.9vw, 17px)",
      fontWeight: 300,
      letterSpacing: "-0.01em",
      color: working ? "rgba(255,255,255,0.45)" : "#fff",
      whiteSpace: "nowrap",
      overflow: "hidden"
    }
  }, codeText), /*#__PURE__*/React.createElement("button", {
    disabled: working,
    onClick: copy,
    style: {
      display: "inline-flex",
      alignItems: "center",
      gap: "0.45rem",
      background: "var(--iscc-bright-yellow)",
      color: "var(--iscc-deep-navy)",
      fontSize: "0.78rem",
      fontWeight: 600,
      padding: "0.5rem 0.95rem",
      border: "none",
      borderRadius: "0.5rem",
      cursor: working ? "not-allowed" : "pointer",
      opacity: working ? 0.45 : 1,
      flexShrink: 0
    }
  }, /*#__PURE__*/React.createElement(UiIcon, {
    name: copied ? "check" : "copy",
    size: 14
  }), /*#__PURE__*/React.createElement("span", null, copied ? "Copied" : "Copy"))), /*#__PURE__*/React.createElement("div", {
    style: {
      display: "flex",
      alignItems: "center",
      gap: "0.65rem",
      marginTop: "0.9rem",
      marginBottom: "0.5rem"
    }
  }, /*#__PURE__*/React.createElement("span", {
    style: {
      fontFamily: "var(--font-mono)",
      fontSize: "0.6rem",
      fontWeight: 600,
      letterSpacing: "0.16em",
      color: "rgba(255,255,255,0.6)"
    }
  }, unitsLabel), /*#__PURE__*/React.createElement("div", {
    style: {
      flex: 1,
      height: 1,
      background: "rgba(255,255,255,0.15)"
    }
  })), /*#__PURE__*/React.createElement("div", {
    style: {
      display: "grid",
      gridTemplateColumns: "repeat(auto-fit, minmax(170px, 1fr))",
      gap: "0.65rem"
    }
  }, working ? pendingKinds.map(k => /*#__PURE__*/React.createElement(UnitField, {
    key: k,
    kind: k,
    code: scramble(16, tick * 31),
    bits: null,
    selected: false,
    state: specimen.status === "uploading" ? "queued" : "hashing"
  })) : units.map(u => /*#__PURE__*/React.createElement(UnitField, {
    key: u.kind,
    kind: u.kind,
    code: u.code,
    bits: u.bits,
    selected: u.kind === selected,
    state: "ready",
    onSelect: () => setSelected(u.kind)
  }))), /*#__PURE__*/React.createElement("div", {
    style: {
      display: "flex",
      alignItems: "center",
      gap: "0.75rem",
      marginTop: "1rem"
    }
  }, /*#__PURE__*/React.createElement("span", {
    style: {
      fontFamily: "var(--font-mono)",
      fontSize: "0.6rem",
      fontWeight: 500,
      letterSpacing: "0.14em",
      color: "rgba(255,255,255,0.55)"
    }
  }, "ABSTRACT & PERSISTENT"), /*#__PURE__*/React.createElement("div", {
    style: {
      flex: 1,
      height: 2,
      background: "rgba(255,195,0,0.6)"
    }
  }), /*#__PURE__*/React.createElement("span", {
    style: {
      fontFamily: "var(--font-mono)",
      fontSize: "0.6rem",
      fontWeight: 500,
      letterSpacing: "0.14em",
      color: "rgba(255,255,255,0.55)"
    }
  }, "CONCRETE & VOLATILE"))), specimen.status === "done" && units.some(u => u.kind === selected) && /*#__PURE__*/React.createElement(UnitDetail, {
    kind: selected
  }), specimen.status === "done" && /*#__PURE__*/React.createElement("div", {
    style: {
      display: "flex",
      alignItems: "center",
      justifyContent: "space-between",
      gap: "1rem",
      padding: "0.8rem 1.75rem",
      borderTop: "1px solid #f1f3f5",
      background: "var(--surface-2)",
      flexWrap: "wrap"
    }
  }, /*#__PURE__*/React.createElement("button", {
    onClick: () => setRawOpen(!rawOpen),
    style: {
      display: "inline-flex",
      alignItems: "center",
      gap: "0.45rem",
      background: "none",
      border: "none",
      color: "var(--text-muted)",
      fontSize: "0.78rem",
      fontWeight: 500,
      cursor: "pointer",
      padding: 0
    }
  }, /*#__PURE__*/React.createElement("span", {
    style: {
      display: "inline-flex",
      transform: rawOpen ? "rotate(90deg)" : "none",
      transition: "transform .15s"
    }
  }, /*#__PURE__*/React.createElement(UiIcon, {
    name: "chevron-right",
    size: 13
  })), /*#__PURE__*/React.createElement("span", null, "Raw result JSON")), /*#__PURE__*/React.createElement("button", {
    style: {
      display: "inline-flex",
      alignItems: "center",
      gap: "0.5rem",
      background: "none",
      border: "1.5px solid var(--iscc-blue)",
      color: "var(--iscc-blue)",
      fontSize: "0.78rem",
      fontWeight: 600,
      padding: "0.45rem 0.9rem",
      borderRadius: "9999px",
      cursor: "pointer"
    }
  }, /*#__PURE__*/React.createElement(UiIcon, {
    name: "compare",
    size: 14
  }), /*#__PURE__*/React.createElement("span", null, "Compare with another file"))), rawOpen && specimen.status === "done" && /*#__PURE__*/React.createElement("pre", {
    style: {
      margin: 0,
      padding: "1rem 1.75rem",
      fontSize: "0.72rem",
      lineHeight: 1.6,
      color: "var(--text-body)",
      background: "#fff",
      borderTop: "1px solid #f1f3f5",
      maxHeight: "20rem",
      overflow: "auto",
      fontFamily: "var(--font-mono)"
    }
  }, JSON.stringify({
    iscc: specimen.explain.iscc,
    units: specimen.explain.units.map(u => ({
      unit: u.code,
      kind: u.kind
    }))
  }, null, 2)));
}
function StatusPill({
  tone,
  text,
  pulse
}) {
  const tones = {
    info: {
      bg: "rgba(0,84,178,0.07)",
      bd: "rgba(0,84,178,0.3)",
      fg: "var(--iscc-blue)",
      dot: "var(--iscc-blue)"
    },
    success: {
      bg: "rgba(166,219,80,0.18)",
      bd: "rgba(166,219,80,0.55)",
      fg: "#4d6e14",
      dot: "#79a832"
    },
    error: {
      bg: "rgba(245,97,105,0.1)",
      bd: "rgba(245,97,105,0.45)",
      fg: "#b3434a",
      dot: "var(--iscc-coral-red)"
    }
  };
  const t = tones[tone];
  return /*#__PURE__*/React.createElement("span", {
    style: {
      display: "inline-flex",
      alignItems: "center",
      gap: "0.45rem",
      borderRadius: "9999px",
      padding: "0.32rem 0.8rem",
      background: t.bg,
      border: `1px solid ${t.bd}`,
      color: t.fg,
      fontFamily: "var(--font-mono)",
      fontSize: "0.66rem",
      fontWeight: 600,
      letterSpacing: "0.06em"
    }
  }, /*#__PURE__*/React.createElement("span", {
    style: {
      width: 8,
      height: 8,
      borderRadius: "50%",
      background: t.dot,
      animation: pulse ? "iscc-pulse-dot 1.4s ease-in-out infinite" : "none"
    }
  }), text);
}
Object.assign(window, {
  ResultCard
});
})(); } catch (e) { __ds_ns.__errors.push({ path: "ui_kits/generator/result.jsx", error: String((e && e.message) || e) }); }

__ds_ns.Button = __ds_scope.Button;

__ds_ns.IsccCode = __ds_scope.IsccCode;

__ds_ns.ISCC_UNITS = __ds_scope.ISCC_UNITS;

__ds_ns.UnitBadge = __ds_scope.UnitBadge;

__ds_ns.Badge = __ds_scope.Badge;

__ds_ns.Switch = __ds_scope.Switch;

__ds_ns.TextField = __ds_scope.TextField;

__ds_ns.Card = __ds_scope.Card;

__ds_ns.Tabs = __ds_scope.Tabs;

})();

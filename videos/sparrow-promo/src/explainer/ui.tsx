import React from "react";
import { AbsoluteFill, interpolate, useCurrentFrame } from "remotion";
import { C, F } from "../theme";
import { useReveal } from "./timing";

// Shared explainer primitives. Canvas is 1920x1080; the bottom band
// (y > 896) is reserved for captions — keep primary content above it.
export const SAFE_BOTTOM = 880;

export const card: React.CSSProperties = {
  background: "rgba(255,255,255,0.78)",
  border: "1px solid rgba(11,15,20,0.12)",
  borderRadius: 20,
  boxShadow: "0 16px 44px rgba(11,15,20,0.09)",
  fontFamily: F.display,
  color: C.ink,
};

// Scene kicker: "NIGHT ONE" eyebrow + optional digital clock, top-left.
export const NightKicker: React.FC<{ label: string; clock?: string; at: number; tone?: "day" | "night" | "dawn" }> = ({
  label,
  clock,
  at,
  tone = "day",
}) => {
  const r = useReveal();
  const s = r(at);
  const color = tone === "night" ? C.cream : C.ink;
  const accent = tone === "dawn" ? C.coral : tone === "night" ? C.teal : C.coral;
  return (
    <div
      style={{
        position: "absolute",
        top: 56,
        left: 80,
        display: "flex",
        alignItems: "center",
        gap: 22,
        opacity: s,
        transform: `translateY(${(1 - s) * 10}px)`,
      }}
    >
      <div
        style={{
          display: "flex",
          alignItems: "center",
          gap: 14,
          fontFamily: F.mono,
          fontSize: 24,
          fontWeight: 700,
          letterSpacing: "0.18em",
          color,
        }}
      >
        <span style={{ width: 14, height: 14, borderRadius: 4, background: accent, boxShadow: `0 0 14px ${accent}` }} />
        {label}
      </div>
      {clock && (
        <div
          style={{
            fontFamily: F.mono,
            fontSize: 34,
            fontWeight: 700,
            letterSpacing: "0.08em",
            color,
            background: tone === "night" ? "rgba(244,242,237,0.10)" : "rgba(11,15,20,0.06)",
            border: `1.5px solid ${tone === "night" ? "rgba(244,242,237,0.25)" : "rgba(11,15,20,0.12)"}`,
            borderRadius: 12,
            padding: "6px 18px",
          }}
        >
          {clock}
        </div>
      )}
    </div>
  );
};

// Dark code/terminal panel matching the promo's navy.
export const Terminal: React.FC<{
  title?: string;
  style?: React.CSSProperties;
  fontSize?: number;
  children: React.ReactNode;
}> = ({ title = "terminal", style, fontSize = 26, children }) => (
  <div
    style={{
      background: C.navy,
      borderRadius: 20,
      boxShadow: "0 26px 64px rgba(11,15,20,0.28)",
      overflow: "hidden",
      fontFamily: F.mono,
      color: "#E8E6E1",
      ...style,
    }}
  >
    <div
      style={{
        display: "flex",
        alignItems: "center",
        gap: 8,
        padding: "14px 20px",
        background: C.navyElev,
        fontSize: 17,
        color: "rgba(232,230,225,0.55)",
      }}
    >
      {["#FF5F57", "#FEBC2E", "#28C840"].map((c) => (
        <span key={c} style={{ width: 12, height: 12, borderRadius: 99, background: c }} />
      ))}
      <span style={{ marginLeft: 10 }}>{title}</span>
    </div>
    <div style={{ padding: "22px 28px", fontSize, lineHeight: 1.55 }}>{children}</div>
  </div>
);

// Character-by-character type-on between frames `from` and `to`.
export const TypeLine: React.FC<{
  text: string;
  from: number;
  to: number;
  color?: string;
  style?: React.CSSProperties;
  caret?: boolean;
}> = ({ text, from, to, color, style, caret = true }) => {
  const frame = useCurrentFrame();
  const chars = Math.floor(
    interpolate(frame, [from, to], [0, text.length], { extrapolateLeft: "clamp", extrapolateRight: "clamp" }),
  );
  const showCaret = caret && frame >= from && frame <= to + 6;
  if (frame < from) return null;
  return (
    <span style={{ color, ...style }}>
      {text.slice(0, chars)}
      {showCaret && <span style={{ opacity: Math.floor(frame / 8) % 2 === 0 ? 1 : 0.3, color: C.coral }}>|</span>}
    </span>
  );
};

// Word-by-word blur-in of a line, starting at `at` and spread over `span` frames.
export const WordsIn: React.FC<{
  text: string;
  at: number;
  span?: number;
  style?: React.CSSProperties;
  emphasis?: string[];
  emphasisColor?: string;
}> = ({ text, at, span = 14, style, emphasis = [], emphasisColor = C.coral }) => {
  const r = useReveal();
  const words = text.split(" ");
  return (
    <span style={style}>
      {words.map((w, i) => {
        const s = r(at + (i * span) / Math.max(1, words.length - 1), 18);
        const hot = emphasis.some((e) => w.replace(/[.,?!:]/g, "") === e);
        return (
          <span
            key={i}
            style={{
              display: "inline-block",
              marginRight: "0.26em",
              opacity: s,
              filter: `blur(${(1 - s) * 8}px)`,
              transform: `translateY(${(1 - s) * 14}px)`,
              color: hot ? emphasisColor : undefined,
            }}
          >
            {w}
          </span>
        );
      })}
    </span>
  );
};

// Small rounded chip. `tone` picks the palette.
export const Chip: React.FC<{
  children: React.ReactNode;
  tone?: "teal" | "coral" | "red" | "ink" | "ghost";
  size?: number;
  style?: React.CSSProperties;
}> = ({ children, tone = "ink", size = 20, style }) => {
  const pal = {
    teal: { fg: C.tealDeep, bg: "rgba(0,173,216,0.10)", bd: "rgba(0,173,216,0.45)" },
    coral: { fg: C.coralText, bg: "rgba(249,115,22,0.10)", bd: "rgba(249,115,22,0.5)" },
    red: { fg: C.red, bg: "rgba(229,72,77,0.10)", bd: "rgba(229,72,77,0.5)" },
    ink: { fg: C.ink, bg: "rgba(255,255,255,0.7)", bd: "rgba(11,15,20,0.16)" },
    ghost: { fg: "rgba(11,15,20,0.5)", bg: "rgba(11,15,20,0.04)", bd: "rgba(11,15,20,0.10)" },
  }[tone];
  return (
    <span
      style={{
        display: "inline-flex",
        alignItems: "center",
        gap: 8,
        fontFamily: F.mono,
        fontSize: size,
        fontWeight: 600,
        color: pal.fg,
        background: pal.bg,
        border: `1.5px solid ${pal.bd}`,
        borderRadius: 999,
        padding: `${Math.round(size * 0.35)}px ${Math.round(size * 0.8)}px`,
        whiteSpace: "nowrap",
        ...style,
      }}
    >
      {children}
    </span>
  );
};

// The hero object of the film: a parcel. Coral box with a tape stripe.
export const Parcel: React.FC<{ size?: number; color?: string; style?: React.CSSProperties; glow?: number }> = ({
  size = 64,
  color = C.coral,
  style,
  glow = 0,
}) => (
  <svg
    width={size}
    height={size}
    viewBox="0 0 64 64"
    style={{ filter: glow > 0 ? `drop-shadow(0 0 ${glow}px ${color})` : undefined, ...style }}
  >
    <path d="M8 22 L32 10 L56 22 L56 48 L32 60 L8 48 Z" fill={color} opacity={0.92} />
    <path d="M8 22 L32 34 L56 22" fill="none" stroke="rgba(255,255,255,0.7)" strokeWidth={2.5} />
    <path d="M32 34 L32 60" stroke="rgba(0,0,0,0.18)" strokeWidth={2.5} />
    <path d="M20 16 L44 28" stroke="rgba(255,255,255,0.55)" strokeWidth={6} strokeLinecap="round" />
  </svg>
);

// A receiving dock: a card with a shutter that is open (green) or closed (red).
export const Dock: React.FC<{
  name: string;
  sub?: string;
  state?: "idle" | "open" | "closed" | "paused" | "skipped";
  width?: number;
  style?: React.CSSProperties;
  badge?: React.ReactNode;
}> = ({ name, sub, state = "idle", width = 420, style, badge }) => {
  const lamp =
    state === "open" ? C.teal : state === "closed" ? C.red : state === "paused" ? "#F5A524" : "rgba(11,15,20,0.25)";
  const dim = state === "skipped";
  return (
    <div
      style={{
        ...card,
        width,
        padding: "18px 22px",
        display: "flex",
        alignItems: "center",
        gap: 18,
        opacity: dim ? 0.45 : 1,
        filter: dim ? "grayscale(0.8)" : undefined,
        borderColor: state === "closed" ? "rgba(229,72,77,0.5)" : state === "open" ? "rgba(0,173,216,0.5)" : undefined,
        ...style,
      }}
    >
      {/* shutter icon */}
      <svg width={54} height={54} viewBox="0 0 54 54">
        <rect x={4} y={8} width={46} height={42} rx={6} fill="rgba(11,15,20,0.06)" stroke="rgba(11,15,20,0.25)" strokeWidth={2} />
        {state === "closed" || state === "paused"
          ? [16, 24, 32, 40].map((y) => <line key={y} x1={8} y1={y} x2={46} y2={y} stroke={lamp} strokeWidth={3} opacity={0.8} />)
          : <rect x={10} y={16} width={34} height={30} rx={3} fill={state === "open" ? "rgba(0,173,216,0.18)" : "rgba(11,15,20,0.04)"} />}
        <circle cx={27} cy={5} r={4} fill={lamp} style={{ filter: `drop-shadow(0 0 6px ${lamp})` }} />
      </svg>
      <div style={{ flex: 1, minWidth: 0 }}>
        <div style={{ fontFamily: F.display, fontSize: 26, fontWeight: 700, color: C.ink }}>{name}</div>
        {sub && (
          <div
            style={{
              fontFamily: F.mono,
              fontSize: 18,
              color: "rgba(11,15,20,0.55)",
              marginTop: 3,
              whiteSpace: "nowrap",
              overflow: "hidden",
              textOverflow: "ellipsis",
            }}
          >
            {sub}
          </div>
        )}
      </div>
      {badge}
    </div>
  );
};

export const Fill: React.FC<{ children: React.ReactNode; style?: React.CSSProperties }> = ({ children, style }) => (
  <AbsoluteFill style={style}>{children}</AbsoluteFill>
);

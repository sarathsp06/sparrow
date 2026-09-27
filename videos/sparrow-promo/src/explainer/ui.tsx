import React from "react";
import { AbsoluteFill, interpolate, useCurrentFrame } from "remotion";
import { C, F } from "../theme";
import { useReveal } from "./timing";

// Shared explainer primitives. Canvas is 1920x1080; the bottom band
// (y > 896) is reserved for captions — keep primary content above it.
export const SAFE_BOTTOM = 880;

export const STEPS = ["Push", "Fan-out", "Build request", "Send & classify", "Back off"] as const;

// The consistent stage for the five-step run: a rail across the top showing
// where we are in the event's journey. `active` is 0-based.
export const StepRail: React.FC<{ active: number }> = ({ active }) => {
  const r = useReveal();
  const fill = r(4, 26);
  return (
    <div
      style={{
        position: "absolute",
        top: 58,
        left: 0,
        right: 0,
        display: "flex",
        justifyContent: "center",
        gap: 14,
        fontFamily: F.mono,
        fontSize: 19,
        letterSpacing: "0.06em",
      }}
    >
      {STEPS.map((s, i) => {
        const done = i < active;
        const cur = i === active;
        const color = cur ? C.coralText : done ? C.tealDeep : "rgba(11,15,20,0.38)";
        return (
          <React.Fragment key={s}>
            {i > 0 && (
              <div style={{ alignSelf: "center", width: 46, height: 2, background: "rgba(11,15,20,0.12)", position: "relative" }}>
                <div
                  style={{
                    position: "absolute",
                    inset: 0,
                    background: C.teal,
                    transformOrigin: "left",
                    transform: `scaleX(${i < active ? 1 : i === active ? fill : 0})`,
                  }}
                />
              </div>
            )}
            <div
              style={{
                display: "flex",
                alignItems: "center",
                gap: 10,
                padding: "8px 16px",
                borderRadius: 999,
                color,
                background: cur ? "rgba(249,115,22,0.10)" : "transparent",
                border: `1.5px solid ${cur ? "rgba(249,115,22,0.55)" : "transparent"}`,
                fontWeight: cur ? 700 : 500,
              }}
            >
              <span
                style={{
                  width: 26,
                  height: 26,
                  borderRadius: 999,
                  display: "grid",
                  placeItems: "center",
                  fontSize: 14,
                  color: cur || done ? "#fff" : "rgba(11,15,20,0.45)",
                  background: cur ? C.coral : done ? C.teal : "rgba(11,15,20,0.08)",
                  boxShadow: cur ? "0 0 12px rgba(249,115,22,0.5)" : "none",
                }}
              >
                {done ? "✓" : i + 1}
              </span>
              {s}
            </div>
          </React.Fragment>
        );
      })}
    </div>
  );
};

// Big step headline under the rail: "Step 2" eyebrow + title.
export const StepTitle: React.FC<{ n: number; title: string; at?: number }> = ({ n, title, at = 0 }) => {
  const r = useReveal();
  const s = r(at);
  return (
    <div
      style={{
        position: "absolute",
        top: 140,
        left: 120,
        fontFamily: F.display,
        color: C.ink,
        opacity: s,
        transform: `translateY(${(1 - s) * 18}px)`,
        filter: `blur(${(1 - s) * 6}px)`,
      }}
    >
      <div style={{ fontFamily: F.mono, fontSize: 22, letterSpacing: "0.18em", color: C.coralText, fontWeight: 700 }}>
        STEP {n}
      </div>
      <div style={{ fontSize: 76, fontWeight: 700, letterSpacing: "-0.03em", lineHeight: 1.05 }}>{title}</div>
    </div>
  );
};

export const card: React.CSSProperties = {
  background: "rgba(255,255,255,0.72)",
  border: "1px solid rgba(11,15,20,0.12)",
  borderRadius: 18,
  boxShadow: "0 14px 38px rgba(11,15,20,0.08)",
  fontFamily: F.display,
  color: C.ink,
};

// Dark code/terminal panel matching the promo's navy.
export const Terminal: React.FC<{
  title?: string;
  style?: React.CSSProperties;
  children: React.ReactNode;
}> = ({ title = "terminal", style, children }) => (
  <div
    style={{
      background: C.navy,
      borderRadius: 18,
      boxShadow: "0 24px 60px rgba(11,15,20,0.28)",
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
        padding: "14px 18px",
        background: C.navyElev,
        fontSize: 16,
        color: "rgba(232,230,225,0.55)",
      }}
    >
      {["#FF5F57", "#FEBC2E", "#28C840"].map((c) => (
        <span key={c} style={{ width: 12, height: 12, borderRadius: 99, background: c }} />
      ))}
      <span style={{ marginLeft: 10 }}>{title}</span>
    </div>
    <div style={{ padding: "22px 26px", fontSize: 24, lineHeight: 1.55 }}>{children}</div>
  </div>
);

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

// Hand-drawn style underline that sweeps in under an element.
export const Underline: React.FC<{ at: number; width: number; color?: string; style?: React.CSSProperties }> = ({
  at,
  width,
  color = C.coral,
  style,
}) => {
  const frame = useCurrentFrame();
  const p = interpolate(frame, [at, at + 14], [0, 1], { extrapolateLeft: "clamp", extrapolateRight: "clamp" });
  return (
    <svg width={width} height={24} style={{ position: "absolute", overflow: "visible", ...style }}>
      <path
        d={`M 4 14 Q ${width * 0.3} 4, ${width * 0.55} 12 T ${width - 4} 10`}
        fill="none"
        stroke={color}
        strokeWidth={7}
        strokeLinecap="round"
        pathLength={1}
        strokeDasharray={1}
        strokeDashoffset={1 - p}
      />
    </svg>
  );
};

export const Fill: React.FC<{ children: React.ReactNode; style?: React.CSSProperties }> = ({ children, style }) => (
  <AbsoluteFill style={style}>{children}</AbsoluteFill>
);

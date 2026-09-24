import React from "react";
import { AbsoluteFill, Easing, interpolate, spring, useCurrentFrame, useVideoConfig } from "remotion";
import { C, F } from "../theme";
import { Kicker } from "../components/Kicker";

// Condensed from the website's own comparison table
// (docs/src/content/docs/getting-started/why-sparrow.mdx, "Sparrow vs the Landscape").
// yes = full check, part = partial/caveat, no = cross, na = em dash (managed/SaaS, N/A).
type Mark = "yes" | "part" | "no" | "na";

const COLS = ["Sparrow", "Svix", "Convoy", "Hookdeck", "AWS SNS"];

const ROWS: { label: string; marks: Mark[] }[] = [
  { label: "Fully open source (MIT)", marks: ["yes", "part", "no", "part", "no"] },
  { label: "Self-hosted", marks: ["yes", "yes", "yes", "part", "no"] },
  { label: "PostgreSQL only \u2014 no Redis", marks: ["yes", "no", "no", "na", "na"] },
  { label: "HMAC + Ed25519 signing", marks: ["yes", "part", "part", "part", "no"] },
  { label: "Envelope-encrypted secrets", marks: ["yes", "part", "part", "part", "part"] },
  { label: "No per-message pricing", marks: ["yes", "no", "yes", "no", "no"] },
];

const MARK: Record<Mark, { glyph: string; color: string }> = {
  yes: { glyph: "\u2713", color: "#0E7490" },
  part: { glyph: "\u25D1", color: "#B45309" },
  no: { glyph: "\u2715", color: "#9F1239" },
  na: { glyph: "\u2014", color: "rgba(11,15,20,0.35)" },
};

const LABEL_W = 560;
const COL_W = 190;
const ROW_H = 86;
const TABLE_W = LABEL_W + COLS.length * COL_W;
const TABLE_X = (1920 - TABLE_W) / 2;
const TABLE_Y = 300;

export const Compare: React.FC = () => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();

  const pop = (delay: number) => {
    const s = spring({ frame: frame - delay, fps, config: { damping: 200 } });
    return { opacity: s, transform: `translateY(${(1 - s) * 18}px)` };
  };
  const titleIn = spring({ frame: frame - 8, fps, config: { damping: 200 } });

  return (
    <AbsoluteFill style={{ fontFamily: F.display, color: C.ink }}>
      <Kicker text="HOW IT COMPARES" />

      <div
        style={{
          position: "absolute",
          top: 150,
          width: "100%",
          textAlign: "center",
          fontSize: 72,
          fontWeight: 700,
          letterSpacing: "-0.035em",
          opacity: titleIn,
          transform: `translateY(${(1 - titleIn) * 20}px)`,
        }}
      >
        All of it, in the open.
      </div>

      {/* Sparrow column highlight */}
      <div
        style={{
          position: "absolute",
          left: TABLE_X + LABEL_W,
          top: TABLE_Y - 64,
          width: COL_W,
          height:
            (64 + ROWS.length * ROW_H + 16) *
            interpolate(frame, [20, 46 + ROWS.length * 12], [0.12, 1], {
              extrapolateLeft: "clamp",
              extrapolateRight: "clamp",
              easing: Easing.inOut(Easing.cubic),
            }),
          background: "linear-gradient(180deg, rgba(0,173,216,0.16), rgba(0,173,216,0.06))",
          border: "1.5px solid rgba(0,173,216,0.5)",
          borderRadius: 16,
          boxShadow: "0 0 40px rgba(0,173,216,0.18)",
          opacity: interpolate(frame, [20, 30], [0, 1], { extrapolateLeft: "clamp", extrapolateRight: "clamp" }),
        }}
      />

      {/* Header row */}
      {COLS.map((col, c) => (
        <div
          key={col}
          style={{
            position: "absolute",
            left: TABLE_X + LABEL_W + c * COL_W,
            top: TABLE_Y - 48,
            width: COL_W,
            textAlign: "center",
            fontFamily: F.mono,
            fontSize: 21,
            fontWeight: c === 0 ? 700 : 400,
            ...pop(24 + c * 5),
          }}
        >
          {col}
        </div>
      ))}

      {/* Rows */}
      {ROWS.map((row, r) => (
        <div
          key={row.label}
          style={{
            position: "absolute",
            left: TABLE_X,
            top: TABLE_Y + r * ROW_H,
            width: TABLE_W,
            height: ROW_H,
            display: "flex",
            alignItems: "center",
            borderBottom: r < ROWS.length - 1 ? "1px solid rgba(11,15,20,0.1)" : "none",
            ...pop(46 + r * 12),
          }}
        >
          <div style={{ width: LABEL_W, fontSize: 30, fontWeight: 500, paddingRight: 24 }}>
            {row.label}
          </div>
          {row.marks.map((m, c) => {
            const at = 46 + r * 12 + c * 3;
            const s = spring({ frame: frame - at, fps, config: { damping: 10, stiffness: 200 } });
            return (
              <div
                key={c}
                style={{
                  width: COL_W,
                  textAlign: "center",
                  fontSize: m === "yes" ? 38 : 32,
                  fontWeight: 700,
                  color: MARK[m].color,
                  scale: String(c === 0 ? 0.4 + s * 0.6 : 1),
                  opacity: c === 0 ? s : interpolate(frame, [at, at + 8], [0, 1], { extrapolateLeft: "clamp", extrapolateRight: "clamp" }),
                  textShadow: c === 0 ? `0 0 ${18 * s}px rgba(0,173,216,0.45)` : undefined,
                }}
              >
                {MARK[m].glyph}
              </div>
            );
          })}
        </div>
      ))}

      <div
        style={{
          position: "absolute",
          bottom: 84,
          width: "100%",
          textAlign: "center",
          fontFamily: F.mono,
          fontSize: 20,
          opacity: 0.55 * spring({ frame: frame - 140, fps, config: { damping: 200 } }),
        }}
      >
        {"\u25D1"} = partial / plan-dependent &#183; verify these claims in current vendor docs &#8212; this snapshot may be outdated
      </div>
    </AbsoluteFill>
  );
};

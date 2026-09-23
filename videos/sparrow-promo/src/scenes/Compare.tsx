import React from "react";
import { AbsoluteFill, interpolate, spring, useCurrentFrame, useVideoConfig } from "remotion";
import { C, F } from "../theme";

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
    <AbsoluteFill style={{ background: C.cream, fontFamily: F.display, color: C.ink }}>
      <div
        style={{
          position: "absolute",
          top: 64,
          left: 72,
          fontFamily: F.mono,
          fontSize: 22,
          letterSpacing: "0.16em",
          opacity: interpolate(frame, [0, 10], [0, 1], { extrapolateRight: "clamp" }),
        }}
      >
        <span style={{ color: C.coralText }}>&#10033;</span> HOW IT COMPARES
      </div>

      <div
        style={{
          position: "absolute",
          top: 150,
          width: "100%",
          textAlign: "center",
          fontSize: 64,
          fontWeight: 600,
          letterSpacing: "-0.02em",
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
          height: 64 + ROWS.length * ROW_H + 16,
          background: "rgba(0,173,216,0.09)",
          border: "1.5px solid rgba(0,173,216,0.45)",
          borderRadius: 16,
          ...pop(20),
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
          {row.marks.map((m, c) => (
            <div
              key={c}
              style={{
                width: COL_W,
                textAlign: "center",
                fontSize: m === "yes" ? 38 : 32,
                fontWeight: 700,
                color: MARK[m].color,
              }}
            >
              {MARK[m].glyph}
            </div>
          ))}
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

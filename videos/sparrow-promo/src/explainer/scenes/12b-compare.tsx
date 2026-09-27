import React from "react";
import {
  AbsoluteFill,
  Sequence,
  interpolate,
  useCurrentFrame,
  useVideoConfig,
  spring,
  staticFile,
  Easing,
} from "remotion";
import { Audio } from "@remotion/media";
import { C, F } from "../../theme";
import { Kicker } from "../../components/Kicker";
import { cueFrame, useReveal } from "../timing";

const ID = "12b-compare";

const COLS = ["Sparrow", "Svix", "Convoy", "Hookdeck", "DIY"];

type CellData = {
  text: string;
  // "sparrow" = teal bold, "muted" = dimmed, "coral" = coral emphasis, "normal" = default
  style: "sparrow" | "muted" | "normal" | "coral";
};

const ROWS: { label: string; cells: CellData[] }[] = [
  {
    label: "Fully open source (MIT)",
    cells: [
      { text: "Yes", style: "sparrow" },
      { text: "Partial", style: "normal" },
      { text: "No (Elastic 2.0)", style: "muted" },
      { text: "Partial", style: "normal" },
      { text: "Yes", style: "normal" },
    ],
  },
  {
    label: "Core infrastructure",
    cells: [
      { text: "PostgreSQL only", style: "sparrow" },
      { text: "PostgreSQL + Redis", style: "normal" }, // will be coral-emphasized at c1
      { text: "PostgreSQL + Redis", style: "normal" }, // will be coral-emphasized at c1
      { text: "SaaS", style: "normal" }, // will be coral-emphasized at c1
      { text: "Varies", style: "muted" },
    ],
  },
  {
    label: "Webhook signing",
    cells: [
      { text: "HMAC-SHA256 + Ed25519", style: "sparrow" },
      { text: "HMAC-SHA256", style: "normal" },
      { text: "HMAC-SHA256", style: "normal" },
      { text: "HMAC-SHA256", style: "normal" },
      { text: "Manual", style: "muted" },
    ],
  },
  {
    label: "Prebuilt integrations",
    cells: [
      { text: "Recipes", style: "sparrow" },
      { text: "Limited", style: "muted" },
      { text: "Limited", style: "muted" },
      { text: "Yes", style: "normal" },
      { text: "Manual", style: "muted" },
    ],
  },
  {
    label: "Consumer portal",
    cells: [
      { text: "Yes, self-hosted", style: "sparrow" },
      { text: "Yes", style: "normal" },
      { text: "Yes", style: "normal" },
      { text: "Yes", style: "normal" },
      { text: "No", style: "muted" },
    ],
  },
  {
    label: "Self-monitoring alerts",
    cells: [
      { text: "System events + email", style: "sparrow" },
      { text: "Operational webhooks", style: "normal" },
      { text: "Alert configs", style: "normal" },
      { text: "Issue alerts", style: "normal" },
      { text: "Manual", style: "muted" },
    ],
  },
];

// Layout constants
const LABEL_W = 340;
const COL_W = 260;
const ROW_H = 76;
const TABLE_W = LABEL_W + COLS.length * COL_W;
const TABLE_X = (1920 - TABLE_W) / 2;
const TABLE_Y = 230;
const HEADER_Y = TABLE_Y - 52;

// Cell colour resolver
const cellColor = (
  style: CellData["style"],
  coralOverride: boolean,
  tealPulse: number,
): { color: string; fontWeight: number; opacity: number } => {
  if (coralOverride) {
    return { color: C.coral, fontWeight: 700, opacity: 1 };
  }
  switch (style) {
    case "sparrow":
      return {
        color: interpolate(tealPulse, [0, 1], [0, 1], {
          extrapolateLeft: "clamp",
          extrapolateRight: "clamp",
        })
          ? C.teal
          : C.tealDeep,
        fontWeight: 700,
        opacity: 1,
      };
    case "muted":
      return { color: "rgba(11,15,20,0.38)", fontWeight: 400, opacity: 0.7 };
    case "coral":
      return { color: C.coral, fontWeight: 600, opacity: 1 };
    default:
      return { color: C.ink, fontWeight: 500, opacity: 0.85 };
  }
};

export const Compare: React.FC = () => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();
  const r = useReveal();

  const c0 = cueFrame(ID, 0);
  const c1 = cueFrame(ID, 1);
  // -- c1 word offsets (dur = 3.846s) --
  // "Svix and Convoy add Redis. Hookdeck's core is SaaS."
  // Redis emphasis lands ~1.2s in, SaaS ~3.0s in
  const redisEmphAt = cueFrame(ID, 1, 1.2);
  const saasEmphAt = cueFrame(ID, 1, 2.8);

  // -- c2 word offsets (dur = 5.84s) --
  // "Sparrow is fully MIT, with dual signing, recipes, a portal and self-monitoring."
  const mitPulseAt = cueFrame(ID, 2, 0.6);
  const signingAt = cueFrame(ID, 2, 1.6);
  const recipesAt = cueFrame(ID, 2, 2.4);
  const portalAt = cueFrame(ID, 2, 3.2);
  const monitorAt = cueFrame(ID, 2, 4.0);

  // Row reveal frames: rows 0-1 at c1, rows 2-5 staggered across c2
  const rowRevealAt = [
    c1,
    c1 + 8,
    signingAt,
    recipesAt,
    portalAt,
    monitorAt,
  ];

  // Coral emphasis on "PostgreSQL + Redis" cells (row 1, cols 1 & 2) and "SaaS" (row 1, col 3)
  const redisEmph = interpolate(frame, [redisEmphAt, redisEmphAt + 14], [0, 1], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
    easing: Easing.out(Easing.cubic),
  });
  const saasEmph = interpolate(frame, [saasEmphAt, saasEmphAt + 14], [0, 1], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
    easing: Easing.out(Easing.cubic),
  });

  // MIT pulse on Sparrow cell row 0
  const mitPulseRaw = spring({
    frame: frame - mitPulseAt,
    fps,
    config: { damping: 200 },
    durationInFrames: 30,
  });

  // Sparrow column highlight band
  const bandIn = interpolate(frame, [c0, c0 + 22], [0, 1], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
    easing: Easing.out(Easing.cubic),
  });
  const bandGrow = interpolate(frame, [c1, c1 + 60], [0.15, 1], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
    easing: Easing.inOut(Easing.cubic),
  });

  // Header columns
  const headerIn = (ci: number) =>
    spring({
      frame: frame - (c0 + 6 + ci * 4),
      fps,
      config: { damping: 200 },
      durationInFrames: 22,
    });

  // Footnote
  const footnoteIn = r(rowRevealAt[5] + 16);

  return (
    <AbsoluteFill style={{ fontFamily: F.display, color: C.ink }}>
      {/* Kicker */}
      <Kicker text="HOW IT COMPARES" delay={c0} />

      {/* Sparrow column highlight band */}
      <div
        style={{
          position: "absolute",
          left: TABLE_X + LABEL_W,
          top: HEADER_Y - 16,
          width: COL_W,
          height:
            (68 + ROWS.length * ROW_H + 20) * bandGrow,
          background:
            "linear-gradient(180deg, rgba(0,173,216,0.14), rgba(0,173,216,0.05))",
          border: "1.5px solid rgba(0,173,216,0.45)",
          borderRadius: 14,
          boxShadow: "0 0 36px rgba(0,173,216,0.15)",
          opacity: bandIn,
        }}
      />

      {/* Header row */}
      {COLS.map((col, ci) => {
        const s = headerIn(ci);
        return (
          <div
            key={col}
            style={{
              position: "absolute",
              left: TABLE_X + LABEL_W + ci * COL_W,
              top: HEADER_Y,
              width: COL_W,
              textAlign: "center",
              fontFamily: F.mono,
              fontSize: 21,
              fontWeight: ci === 0 ? 700 : 400,
              letterSpacing: "0.02em",
              color: ci === 0 ? C.tealDeep : C.ink,
              opacity: s,
              transform: `translateY(${(1 - s) * 14}px)`,
            }}
          >
            {col}
          </div>
        );
      })}

      {/* Table rows */}
      {ROWS.map((row, ri) => {
        const revealAt = rowRevealAt[ri];
        const rowS = spring({
          frame: frame - revealAt,
          fps,
          config: { damping: 200 },
          durationInFrames: 22,
        });

        return (
          <div
            key={row.label}
            style={{
              position: "absolute",
              left: TABLE_X,
              top: TABLE_Y + ri * ROW_H,
              width: TABLE_W,
              height: ROW_H,
              display: "flex",
              alignItems: "center",
              borderBottom:
                ri < ROWS.length - 1
                  ? "1px solid rgba(11,15,20,0.08)"
                  : "none",
              opacity: rowS,
              transform: `translateY(${(1 - rowS) * 16}px)`,
            }}
          >
            {/* Row label */}
            <div
              style={{
                width: LABEL_W,
                fontSize: 23,
                fontWeight: 500,
                paddingRight: 20,
                lineHeight: 1.25,
              }}
            >
              {row.label}
            </div>

            {/* Cells */}
            {row.cells.map((cell, ci) => {
              // Determine coral override for infrastructure row
              const isInfraRow = ri === 1;
              const isRedisCell = isInfraRow && (ci === 1 || ci === 2);
              const isSaasCell = isInfraRow && ci === 3;
              const coralOverride =
                (isRedisCell && redisEmph > 0.5) ||
                (isSaasCell && saasEmph > 0.5);

              // Teal pulse for MIT row Sparrow cell
              const tealPulse =
                ri === 0 && ci === 0 ? mitPulseRaw : 0;

              const { color, fontWeight, opacity } = cellColor(
                cell.style,
                coralOverride,
                tealPulse,
              );

              // Per-cell stagger
              const cellDelay = revealAt + 4 + ci * 3;
              const cellS = spring({
                frame: frame - cellDelay,
                fps,
                config: { damping: 200 },
                durationInFrames: 20,
              });

              return (
                <div
                  key={ci}
                  style={{
                    width: COL_W,
                    textAlign: "center",
                    fontFamily: ci === 0 ? F.mono : F.display,
                    fontSize: 20,
                    fontWeight,
                    color,
                    opacity: opacity * cellS,
                    lineHeight: 1.25,
                    padding: "0 8px",
                    textShadow:
                      ci === 0 && cell.style === "sparrow"
                        ? `0 0 ${14 * cellS}px rgba(0,173,216,0.3)`
                        : coralOverride
                          ? `0 0 12px rgba(249,115,22,0.35)`
                          : undefined,
                  }}
                >
                  {cell.text}
                </div>
              );
            })}
          </div>
        );
      })}

      {/* Footnote */}
      <div
        style={{
          position: "absolute",
          bottom: 1080 - 860,
          width: "100%",
          textAlign: "center",
          fontFamily: F.mono,
          fontSize: 17,
          color: "rgba(11,15,20,0.42)",
          opacity: footnoteIn * 0.7,
          transform: `translateY(${(1 - footnoteIn) * 8}px)`,
        }}
      >
        Directional snapshot {"—"} vendors change packaging often;
        verify against their docs.
      </div>

      {/* SFX: subtle ping on header reveal */}
      <Sequence from={c0 + 6}>
        <Audio src={staticFile("sfx/ping.mp3")} volume={0.12} />
      </Sequence>
    </AbsoluteFill>
  );
};

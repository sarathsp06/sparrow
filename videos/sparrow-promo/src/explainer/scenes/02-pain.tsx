import React from "react";
import {
  AbsoluteFill,
  Easing,
  Img,
  interpolate,
  random,
  Sequence,
  spring,
  staticFile,
  useCurrentFrame,
  useVideoConfig,
} from "remotion";
import { Audio } from "@remotion/media";
import { C, F } from "../../theme";
import { cueFrame, cueEndFrame } from "../timing";

// Frame 2 — The DIY stack collapses
// Reproduces the promo OldWay: chip board piles up, stress wobble, everything
// but PostgreSQL drops out, lockup lands. Retimed to VO cues. Cream backdrop
// is provided by the persistent Backdrop (this scene is transparent).
//
// Changes vs promo:
// - Subtitle: "webhook delivery you run yourself" (not "a self-hosted webhook delivery service")
// - Chips: MIT licensed · Go + PostgreSQL · OpenAPI 3.1 (not the promo's three)
// - Headline: "DIY means you manage all of this yourself." replaced by
//   the chip-board building on c0 VO names

const ID = "02-pain";
const c0 = cueFrame(ID, 0); // "So teams build it: a Redis queue, retry crons, …"
const c0End = cueEndFrame(ID, 0);
// c1: "And events still quietly get lost." — stress wobble peaks here
const c2 = cueFrame(ID, 2); // "With Sparrow, it's just Postgres."
const c3 = cueFrame(ID, 3); // "Sparrow: webhook delivery you run yourself."

// Chips: same set as promo OldWay. Named chips from c0 VO land on their words;
// the rest fill in between.
type Chip = {
  label: string;
  x: number;
  y: number;
  rot: number;
  keep?: boolean;
  /** Fractional offset into c0 when this chip's name is spoken (0–1). */
  voiceOffset?: number;
};

// The VO says: "a Redis queue, retry crons, signing code, dashboards, alerts."
// Map named chips to approximate word positions in the cue text.
const CHIPS: Chip[] = [
  { label: "Redis · queues", x: 480, y: 280, rot: -3, voiceOffset: 0.18 },
  { label: "client SDKs", x: 860, y: 288, rot: -1 },
  { label: "RabbitMQ · retries", x: 1180, y: 272, rot: 2 },
  { label: "rate limiting", x: 330, y: 420, rot: 2 },
  { label: "payload audit logs", x: 640, y: 428, rot: 1 },
  { label: "PostgreSQL", x: 1020, y: 420, rot: 0, keep: true },
  { label: "dead-letter queue", x: 1310, y: 428, rot: -2 },
  { label: "retry cron jobs", x: 420, y: 558, rot: -2, voiceOffset: 0.36 },
  { label: "consumer dashboard", x: 850, y: 566, rot: -2, voiceOffset: 0.7 },
  { label: "key rotation", x: 1250, y: 560, rot: 3 },
  { label: "HMAC signing code", x: 560, y: 690, rot: 2, voiceOffset: 0.52 },
  { label: "monitoring + alerts", x: 1030, y: 696, rot: 1, voiceOffset: 0.88 },
];

const PG_W = 212;
const LOCKUP_CHIPS = ["MIT licensed", "Go + PostgreSQL", "OpenAPI 3.1"];

export const Pain: React.FC = () => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();

  // --- Timing ---
  const c0Dur = c0End - c0;

  // Chip entrance: named chips land at their voiceOffset; others fill gaps.
  // Sort chips by their entrance frame for the counter.
  const chipEntranceFrames = CHIPS.map((c, i) => {
    if (c.voiceOffset !== undefined) {
      return c0 + Math.round(c.voiceOffset * c0Dur);
    }
    // Unnamed chips fill in uniformly across the cue
    const unnamedIndex = CHIPS.filter(
      (ch, j) => ch.voiceOffset === undefined && j <= i,
    ).length - 1;
    const unnamedTotal = CHIPS.filter(
      (ch) => ch.voiceOffset === undefined,
    ).length;
    return c0 + Math.round(((unnamedIndex + 0.5) / unnamedTotal) * c0Dur);
  });

  // Payoff: everything but PostgreSQL drops at c2
  const payoff = spring({
    frame: frame - c2,
    fps,
    config: { damping: 200 },
  });

  // Stress wobble builds through c1
  const stress = interpolate(frame, [c0 + 30, c2], [0, 1], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
    easing: Easing.in(Easing.quad),
  });

  // PostgreSQL chip glides to center
  const glide = interpolate(frame, [c2, c2 + 30], [0, 1], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
    easing: Easing.bezier(0.65, 0, 0.35, 1),
  });

  // Board fades out for lockup
  const boardOut = interpolate(frame, [c3, c3 + 12], [1, 0], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
  });

  // "With Sparrow, it's just [PostgreSQL]" line
  const justPgIn = payoff;

  // Headline fades out before payoff
  const headlineOut = 1 - payoff;

  const shownCount = chipEntranceFrames.filter((f) => frame >= f).length;

  return (
    <AbsoluteFill style={{ overflow: "hidden" }}>
      {/* Board layer — fades out for lockup */}
      <AbsoluteFill
        style={{
          opacity: boardOut,
          scale: String(1 - (1 - boardOut) * 0.06),
        }}
      >
        {/* Headline: "DIY means you manage all of this yourself." + counter */}
        <div
          style={{
            position: "absolute",
            top: 104,
            width: "100%",
            textAlign: "center",
            fontFamily: F.serif,
            fontSize: 72,
            color: C.ink,
            opacity:
              interpolate(frame, [c0, c0 + 12], [0, 1], {
                extrapolateRight: "clamp",
                extrapolateLeft: "clamp",
              }) * headlineOut,
          }}
        >
          DIY means you manage <em>all of this</em> yourself.
          <span
            style={{
              marginLeft: 22,
              fontFamily: F.mono,
              fontSize: 26,
              verticalAlign: "middle",
              color: C.cream,
              background: C.coralDeep,
              borderRadius: 999,
              padding: "6px 16px",
              opacity: shownCount ? 1 : 0,
            }}
          >
            {shownCount}
          </span>
        </div>

        {/* Chip board */}
        {CHIPS.map((c, i) => {
          const at = chipEntranceFrames[i];
          const s = spring({
            frame: frame - at,
            fps,
            config: { damping: 200 },
          });
          const wobble =
            Math.sin(frame * 0.9 + i * 1.7) * stress * 4;
          const jx = Math.sin(frame * 1.3 + i) * stress * 3;

          let dx = jx;
          let dy = 0;
          let rot = c.rot + wobble;
          let op = s;

          if (c.keep) {
            dx =
              (960 - PG_W / 2 - c.x) * glide + jx * (1 - glide);
            dy = (480 - c.y) * glide;
            rot =
              c.rot * (1 - glide) + wobble * (1 - glide);
          } else {
            const fallStart =
              c2 + random(`fall-${i}`) * 10;
            const ft = Math.max(0, frame - fallStart);
            dy = 2.6 * ft * ft;
            dx += (random(`drift-${i}`) - 0.5) * ft * 14;
            rot += (random(`spin-${i}`) - 0.5) * ft * 5;
            op =
              s *
              interpolate(ft, [0, 18], [1, 0], {
                extrapolateRight: "clamp",
              });
          }

          return (
            <div
              key={c.label}
              style={{
                position: "absolute",
                left: c.x,
                top: c.y,
                padding: "16px 28px",
                borderRadius: 14,
                background: c.keep
                  ? "rgba(0,173,216,0.12)"
                  : "rgba(255,255,255,0.88)",
                border: c.keep
                  ? "2px solid rgba(0,173,216,0.6)"
                  : "1px solid rgba(11,15,20,0.16)",
                boxShadow: c.keep
                  ? `0 10px 26px rgba(11,15,20,0.08), 0 0 ${40 * payoff}px rgba(0,173,216,0.35)`
                  : "0 10px 26px rgba(11,15,20,0.08)",
                fontFamily: F.mono,
                fontSize: 27,
                fontWeight: c.keep ? 700 : 400,
                color: C.ink,
                whiteSpace: "nowrap",
                opacity: op,
                translate: `${dx}px ${dy}px`,
                rotate: `${rot}deg`,
                scale: String(
                  s * (1 + (c.keep ? payoff * 0.25 : 0)),
                ),
              }}
            >
              {c.label}
            </div>
          );
        })}

        {/* "With Sparrow, it's just [PostgreSQL]" line */}
        <div
          style={{
            position: "absolute",
            bottom: 170,
            width: "100%",
            textAlign: "center",
            fontFamily: F.serif,
            fontSize: 66,
            color: C.ink,
            opacity: justPgIn,
            translate: `0px ${(1 - justPgIn) * 20}px`,
          }}
        >
          With <span style={{ color: C.teal }}>Sparrow</span>,
          it&rsquo;s just{" "}
          <em style={{ color: C.tealDeep }}>PostgreSQL</em>.
        </div>
      </AbsoluteFill>

      {/* Lockup: logo + wordmark + subtitle + chips */}
      <AbsoluteFill
        style={{ justifyContent: "center", alignItems: "center" }}
      >
        <div style={{ textAlign: "center" }}>
          {/* Logo */}
          <Img
            src={staticFile("sparrow-logo.svg")}
            style={{
              width: 150,
              height: 120,
              opacity: interpolate(
                frame,
                [c3 + 8, c3 + 14],
                [0, 1],
                {
                  extrapolateLeft: "clamp",
                  extrapolateRight: "clamp",
                },
              ),
              translate: `${interpolate(
                frame,
                [c3 + 8, c3 + 30],
                [-600, 0],
                {
                  extrapolateLeft: "clamp",
                  extrapolateRight: "clamp",
                  easing: Easing.bezier(0.16, 1, 0.3, 1),
                },
              )}px ${interpolate(
                frame,
                [c3 + 8, c3 + 30],
                [-160, 0],
                {
                  extrapolateLeft: "clamp",
                  extrapolateRight: "clamp",
                  easing: Easing.bezier(0.16, 1, 0.3, 1),
                },
              )}px`,
              rotate: `${interpolate(
                frame,
                [c3 + 8, c3 + 34],
                [-25, 0],
                {
                  extrapolateLeft: "clamp",
                  extrapolateRight: "clamp",
                  easing: Easing.bezier(0.16, 1, 0.3, 1),
                },
              )}deg`,
            }}
          />

          {/* Wordmark: "Sparrow" letter-by-letter */}
          <div
            style={{
              fontFamily: F.display,
              fontWeight: 700,
              fontSize: 156,
              letterSpacing: "-0.045em",
              color: C.ink,
              lineHeight: 1.05,
            }}
          >
            {"Sparrow".split("").map((ch, i) => {
              const s = spring({
                frame: frame - (c3 + 16 + i * 2),
                fps,
                config: { damping: 200 },
              });
              return (
                <span
                  key={i}
                  style={{
                    display: "inline-block",
                    opacity: s,
                    translate: `0px ${(1 - s) * 60}px`,
                  }}
                >
                  {ch}
                </span>
              );
            })}
          </div>

          {/* Subtitle: "webhook delivery you run yourself" */}
          <div
            style={{
              marginTop: 10,
              fontFamily: F.display,
              fontWeight: 500,
              fontSize: 42,
              letterSpacing: "-0.015em",
              color: C.ink,
              opacity: interpolate(
                frame,
                [c3 + 26, c3 + 40],
                [0, 1],
                {
                  extrapolateLeft: "clamp",
                  extrapolateRight: "clamp",
                },
              ),
              filter: `blur(${interpolate(
                frame,
                [c3 + 26, c3 + 40],
                [8, 0],
                {
                  extrapolateLeft: "clamp",
                  extrapolateRight: "clamp",
                },
              )}px)`,
            }}
          >
            webhook delivery you run yourself
          </div>

          {/* Badge chips: MIT licensed · Go + PostgreSQL · OpenAPI 3.1 */}
          <div
            style={{
              marginTop: 30,
              display: "flex",
              gap: 14,
              justifyContent: "center",
            }}
          >
            {LOCKUP_CHIPS.map((t, i) => {
              const s = spring({
                frame: frame - (c3 + 38 + i * 6),
                fps,
                config: { damping: 200 },
              });
              return (
                <span
                  key={t}
                  style={{
                    fontFamily: F.mono,
                    fontSize: 24,
                    letterSpacing: "0.02em",
                    color: C.ink,
                    padding: "10px 20px",
                    borderRadius: 999,
                    border: "1px solid rgba(11,15,20,0.16)",
                    background: "rgba(255,255,255,0.6)",
                    opacity: s,
                    translate: `0px ${(1 - s) * 14}px`,
                  }}
                >
                  {t}
                </span>
              );
            })}
          </div>
        </div>
      </AbsoluteFill>

      {/* SFX: impact when the stack drops */}
      <Sequence from={c2}>
        <Audio src={staticFile("sfx/impact-bass-1.mp3")} volume={0.22} />
      </Sequence>
    </AbsoluteFill>
  );
};

import React from "react";
import {
  AbsoluteFill,
  Easing,
  Img,
  interpolate,
  random,
  spring,
  staticFile,
  useCurrentFrame,
  useVideoConfig,
} from "remotion";
import { C, F } from "../theme";

// 8s beat: the DIY webhook stack piles up and starts to wobble, then everything
// but Postgres drops out of frame ("all you actually need is Postgres"), and the
// Sparrow lockup lands.
const PAYOFF = 112; // non-Postgres chips fall, Postgres glides to center
const SWAP = 152; // board out, lockup in

type Chip = { label: string; x: number; y: number; rot: number; keep?: boolean };

const CHIPS: Chip[] = [
  { label: "Redis · queues", x: 480, y: 280, rot: -3 },
  { label: "client SDKs", x: 860, y: 288, rot: -1 },
  { label: "RabbitMQ · retries", x: 1180, y: 272, rot: 2 },
  { label: "rate limiting", x: 330, y: 420, rot: 2 },
  { label: "payload audit logs", x: 640, y: 428, rot: 1 },
  { label: "PostgreSQL", x: 1020, y: 420, rot: 0, keep: true },
  { label: "dead-letter queue", x: 1310, y: 428, rot: -2 },
  { label: "retry cron jobs", x: 420, y: 558, rot: -2 },
  { label: "consumer dashboard", x: 850, y: 566, rot: -2 },
  { label: "key rotation", x: 1250, y: 560, rot: 3 },
  { label: "HMAC signing code", x: 560, y: 690, rot: 2 },
  { label: "monitoring + alerts", x: 1030, y: 696, rot: 1 },
];
const PG_W = 212; // approx rendered width of the PostgreSQL chip

export const OldWay: React.FC = () => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();

  const boardOut = interpolate(frame, [SWAP, SWAP + 12], [1, 0], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
  });
  const payoff = spring({ frame: frame - PAYOFF, fps, config: { damping: 200 } });
  // Stress builds as chips accumulate, peaks right before the payoff.
  const stress = interpolate(frame, [30, PAYOFF], [0, 1], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
    easing: Easing.in(Easing.quad),
  });
  const glide = interpolate(frame, [PAYOFF, PAYOFF + 30], [0, 1], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
    easing: Easing.bezier(0.65, 0, 0.35, 1),
  });
  const shownCount = CHIPS.filter((_, i) => frame >= 16 + i * 7).length;

  return (
    <AbsoluteFill>
      <AbsoluteFill style={{ opacity: boardOut, scale: String(1 - (1 - boardOut) * 0.06) }}>
        <div
          style={{
            position: "absolute",
            top: 104,
            width: "100%",
            textAlign: "center",
            fontFamily: F.serif,
            fontSize: 72,
            color: C.ink,
            opacity: interpolate(frame, [0, 12], [0, 1], { extrapolateRight: "clamp" }) * (1 - payoff),
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

        {CHIPS.map((c, i) => {
          const at = 16 + i * 7;
          const s = spring({ frame: frame - at, fps, config: { damping: 13, stiffness: 170 } });
          const wobble = Math.sin(frame * 0.9 + i * 1.7) * stress * 4;
          const jx = Math.sin(frame * 1.3 + i) * stress * 3;

          let dx = jx;
          let dy = 0;
          let rot = c.rot + wobble;
          let op = s;
          if (c.keep) {
            dx = (960 - PG_W / 2 - c.x) * glide + jx * (1 - glide);
            dy = (480 - c.y) * glide;
            rot = c.rot * (1 - glide) + wobble * (1 - glide);
          } else {
            const fallStart = PAYOFF + random(`fall-${i}`) * 10;
            const ft = Math.max(0, frame - fallStart);
            dy = 2.6 * ft * ft;
            dx += (random(`drift-${i}`) - 0.5) * ft * 14;
            rot += (random(`spin-${i}`) - 0.5) * ft * 5;
            op = s * interpolate(ft, [0, 18], [1, 0], { extrapolateRight: "clamp" });
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
                background: c.keep ? "rgba(0,173,216,0.12)" : "rgba(255,255,255,0.88)",
                border: c.keep ? "2px solid rgba(0,173,216,0.6)" : "1px solid rgba(11,15,20,0.16)",
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
                scale: String(s * (1 + (c.keep ? payoff * 0.25 : 0))),
              }}
            >
              {c.label}
            </div>
          );
        })}

        <div
          style={{
            position: "absolute",
            bottom: 170,
            width: "100%",
            textAlign: "center",
            fontFamily: F.serif,
            fontSize: 66,
            color: C.ink,
            opacity: payoff,
            translate: `0px ${(1 - payoff) * 20}px`,
          }}
        >
          With <span style={{ color: C.teal }}>Sparrow</span>, it&rsquo;s just{" "}
          <em style={{ color: C.tealDeep }}>Sparrow + Postgres</em>.
        </div>
      </AbsoluteFill>

      <AbsoluteFill style={{ justifyContent: "center", alignItems: "center" }}>
        <div style={{ textAlign: "center" }}>
          <Img
            src={staticFile("sparrow-logo.svg")}
            style={{
              width: 150,
              height: 120,
              opacity: interpolate(frame, [SWAP + 8, SWAP + 14], [0, 1], {
                extrapolateLeft: "clamp",
                extrapolateRight: "clamp",
              }),
              translate: `${interpolate(frame, [SWAP + 8, SWAP + 30], [-600, 0], {
                extrapolateLeft: "clamp",
                extrapolateRight: "clamp",
                easing: Easing.bezier(0.16, 1, 0.3, 1),
              })}px ${interpolate(frame, [SWAP + 8, SWAP + 30], [-160, 0], {
                extrapolateLeft: "clamp",
                extrapolateRight: "clamp",
                easing: Easing.bezier(0.16, 1, 0.3, 1),
              })}px`,
              rotate: `${interpolate(frame, [SWAP + 8, SWAP + 34], [-25, 0], {
                extrapolateLeft: "clamp",
                extrapolateRight: "clamp",
                easing: Easing.spring({ damping: 9 }),
              })}deg`,
            }}
          />
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
              const s = spring({ frame: frame - (SWAP + 16 + i * 2), fps, config: { damping: 14, stiffness: 180 } });
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
          <div
            style={{
              marginTop: 10,
              fontFamily: F.display,
              fontWeight: 500,
              fontSize: 42,
              letterSpacing: "-0.015em",
              color: C.ink,
              opacity: interpolate(frame, [SWAP + 26, SWAP + 40], [0, 1], {
                extrapolateLeft: "clamp",
                extrapolateRight: "clamp",
              }),
              filter: `blur(${interpolate(frame, [SWAP + 26, SWAP + 40], [8, 0], {
                extrapolateLeft: "clamp",
                extrapolateRight: "clamp",
              })}px)`,
            }}
          >
            a self-hosted webhook delivery service
          </div>
          <div style={{ marginTop: 30, display: "flex", gap: 14, justifyContent: "center" }}>
            {["one Go binary", "one Postgres database", "open source"].map((t, i) => {
              const s = spring({ frame: frame - (SWAP + 38 + i * 6), fps, config: { damping: 200 } });
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
    </AbsoluteFill>
  );
};

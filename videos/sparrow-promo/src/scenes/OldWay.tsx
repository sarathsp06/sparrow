import React from "react";
import {
  AbsoluteFill,
  Img,
  interpolate,
  spring,
  staticFile,
  useCurrentFrame,
  useVideoConfig,
} from "remotion";
import { C, F } from "../theme";

// 8s beat: the DIY webhook stack piles up into a mess, everything but Postgres
// grays out ("all you actually need is Postgres"), then the Sparrow lockup lands.
const PAYOFF = 112; // non-Postgres chips fade, Postgres pops
const SWAP = 152; // board out, lockup in

type Chip = { label: string; x: number; y: number; rot: number; keep?: boolean };

const CHIPS: Chip[] = [
  { label: "Redis \u00B7 queues", x: 480, y: 280, rot: -3 },
  { label: "client SDKs", x: 860, y: 288, rot: -1 },
  { label: "RabbitMQ \u00B7 retries", x: 1180, y: 272, rot: 2 },
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
export const OldWay: React.FC = () => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();

  const boardOut = interpolate(frame, [SWAP, SWAP + 12], [1, 0], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
  });
  const lockup = spring({ frame: frame - (SWAP + 14), fps, config: { damping: 200 } });
  const monoline = interpolate(frame, [SWAP + 26, SWAP + 44], [0, 1], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
  });
  const payoff = spring({ frame: frame - PAYOFF, fps, config: { damping: 200 } });
  const payoffT = interpolate(frame, [PAYOFF, PAYOFF + 10], [0, 1], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
  });

  return (
    <AbsoluteFill style={{ background: C.cream }}>
      <AbsoluteFill style={{ opacity: boardOut, transform: `scale(${1 - (1 - boardOut) * 0.06})` }}>
        <div
          style={{
            position: "absolute",
            top: 110,
            width: "100%",
            textAlign: "center",
            fontFamily: "Georgia, serif",
            fontSize: 62,
            color: C.ink,
            opacity: interpolate(frame, [0, 12], [0, 1], { extrapolateRight: "clamp" }),
          }}
        >
          DIY means you manage all of this yourself.
        </div>

        {CHIPS.map((c, i) => {
          const s = spring({
            frame: frame - (16 + i * 7),
            fps,
            config: { damping: 13, stiffness: 170 },
          });
          const dim = c.keep ? 0 : payoffT;
          const keepPop = c.keep ? payoff : 0;
          return (
            <div
              key={c.label}
              style={{
                position: "absolute",
                left: c.x,
                top: c.y,
                padding: "16px 28px",
                borderRadius: 14,
                background: c.keep ? "rgba(0,173,216,0.1)" : "rgba(255,255,255,0.8)",
                border: c.keep
                  ? "2px solid rgba(0,173,216,0.55)"
                  : "1px solid rgba(11,15,20,0.16)",
                boxShadow: "0 10px 26px rgba(11,15,20,0.08)",
                fontFamily: F.mono,
                fontSize: 27,
                color: C.ink,
                whiteSpace: "nowrap",
                opacity: s * (1 - dim * 0.78),
                transform: `rotate(${c.rot}deg) scale(${s * (1 + keepPop * 0.12)})`,
                filter: dim ? `grayscale(${dim})` : undefined,
              }}
            >
              {c.label}
            </div>
          );
        })}

        <div
          style={{
            position: "absolute",
            bottom: 130,
            width: "100%",
            textAlign: "center",
            fontFamily: "Georgia, serif",
            fontSize: 58,
            color: C.ink,
            opacity: payoff,
            transform: `translateY(${(1 - payoff) * 20}px)`,
          }}
        >
          With <span style={{ color: C.teal }}>Sparrow</span>, it’s just <span style={{ color: "#0E7490" }}>Sparrow + Postgres</span>.
        </div>
      </AbsoluteFill>

      <AbsoluteFill style={{ justifyContent: "center", alignItems: "center" }}>
        <div
          style={{
            textAlign: "center",
            opacity: lockup,
            transform: `scale(${0.9 + lockup * 0.1})`,
          }}
        >
          <Img
            src={staticFile("sparrow-logo.svg")}
            style={{ width: 150, height: 120 }}
          />
          <div
            style={{
              fontFamily: F.display,
              fontWeight: 600,
              fontSize: 150,
              letterSpacing: "-0.03em",
              color: C.ink,
            }}
          >
            Sparrow
          </div>
          <div
            style={{
              marginTop: 10,
              fontFamily: F.display,
              fontWeight: 500,
              fontSize: 42,
              letterSpacing: "-0.01em",
              color: C.ink,
            }}
          >
            a self-hosted webhook delivery service
          </div>
          <div
            style={{
              marginTop: 28,
              fontFamily: F.mono,
              fontSize: 26,
              letterSpacing: "0.06em",
              color: C.ink,
              opacity: monoline * 0.8,
            }}
          >
            one Go binary &#183; one Postgres database &#183; open source
          </div>
        </div>
      </AbsoluteFill>
    </AbsoluteFill>
  );
};

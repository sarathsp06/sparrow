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

// Architecture finale: your environment (services + Sparrow + Postgres) → the outside world.
// Then the pull one-liner — README "Latest Docker release" + why-sparrow.mdx
// ("distroless ~15MB", "PostgreSQL is the only infrastructure requirement").
const CMD = "docker pull ghcr.io/sarathsp06/sparrow:latest";

const SERVICES = ["orders-api", "billing-api", "auth-api"];
const OUTSIDE = [
  { name: "webhooks", icon: "icons/webhook.svg" },
  { name: "Slack", icon: "icons/slack.svg" },
  { name: "email", icon: "icons/sendgrid.svg" },
  { name: "SMS", icon: "icons/twilio.svg" },
  { name: "push", icon: "icons/ntfy.svg" },
];

// Diagram geometry (top half of frame).
const BOX = { x: 330, y: 200, w: 800, h: 390 };
const SVC_X = 380;
const SVC_W = 240;
const SVC_TOP = 268;
const SVC_H = 74;
const SVC_GAP = 22;
const SPARROW = { x: 730, y: 258, w: 370, h: 274 };
const OUT_X = 1310;
const OUT_W = 270;
const OUT_H = 62;
const OUT_GAP = 16;
const OUT_TOP = 395 - (OUTSIDE.length * OUT_H + (OUTSIDE.length - 1) * OUT_GAP) / 2;

const node: React.CSSProperties = {
  position: "absolute",
  background: "rgba(255,255,255,0.75)",
  border: "1px solid rgba(11,15,20,0.14)",
  borderRadius: 14,
  boxShadow: "0 10px 26px rgba(11,15,20,0.06)",
  fontFamily: F.display,
  color: C.ink,
};

export const Close: React.FC = () => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();

  const pop = (delay: number) => {
    const s = spring({ frame: frame - delay, fps, config: { damping: 200 } });
    return { opacity: s, transform: `translateY(${(1 - s) * 16}px)` };
  };
  const sp = (delay: number) => spring({ frame: frame - delay, fps, config: { damping: 200 } });

  // Connector draw-ins.
  const inLines = interpolate(frame, [55, 75], [0, 1], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
  });
  const outLines = interpolate(frame, [80, 102], [0, 1], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
  });

  // Traveling pulses, staggered loops so the diagram stays alive.
  const pulseT = (start: number, dur: number) =>
    interpolate(frame, [start, start + dur], [0, 1], {
      extrapolateLeft: "clamp",
      extrapolateRight: "clamp",
    });

  const typedN = Math.round(
    interpolate(frame, [150, 208], [0, CMD.length], {
      extrapolateLeft: "clamp",
      extrapolateRight: "clamp",
    }),
  );
  const pillIn = sp(140);
  const subIn = sp(212);
  const footerIn = sp(228);
  const caretOn = Math.floor(frame / 12) % 2 === 0;

  const svcOut = (i: number) => ({
    x: SVC_X + SVC_W,
    y: SVC_TOP + i * (SVC_H + SVC_GAP) + SVC_H / 2,
  });
  const sparrowIn = { x: SPARROW.x, y: SPARROW.y + SPARROW.h / 2 };
  const sparrowOut = { x: SPARROW.x + SPARROW.w, y: SPARROW.y + SPARROW.h / 2 };
  const outY = (i: number) => OUT_TOP + i * (OUT_H + OUT_GAP) + OUT_H / 2;

  return (
    <AbsoluteFill style={{ background: C.cream }}>
      <div
        style={{
          position: "absolute",
          top: 64,
          left: 72,
          fontFamily: F.mono,
          fontSize: 22,
          letterSpacing: "0.16em",
          color: C.ink,
          opacity: interpolate(frame, [0, 10], [0, 1], { extrapolateRight: "clamp" }),
        }}
      >
        <span style={{ color: C.coralText }}>&#10033;</span> ONE BINARY. ONE DATABASE. SELF-HOSTED.
      </div>

      {/* Diagram connectors */}
      <svg width={1920} height={1080} style={{ position: "absolute", inset: 0 }}>
        <rect
          x={BOX.x}
          y={BOX.y}
          width={BOX.w}
          height={BOX.h}
          rx={22}
          fill="none"
          stroke="rgba(11,15,20,0.22)"
          strokeWidth={2}
          strokeDasharray="10 10"
          opacity={sp(6)}
        />
        {SERVICES.map((s, i) => {
          const o = svcOut(i);
          return (
            <line
              key={s}
              x1={o.x}
              y1={o.y}
              x2={o.x + (sparrowIn.x - o.x) * inLines}
              y2={o.y + (sparrowIn.y - o.y) * inLines}
              stroke="rgba(11,15,20,0.25)"
              strokeWidth={2.5}
              strokeDasharray="2 9"
              strokeLinecap="round"
            />
          );
        })}
        {OUTSIDE.map((o, i) => (
          <line
            key={o.name}
            x1={sparrowOut.x}
            y1={sparrowOut.y}
            x2={sparrowOut.x + (OUT_X - sparrowOut.x) * outLines}
            y2={sparrowOut.y + (outY(i) - sparrowOut.y) * outLines}
            stroke="rgba(11,15,20,0.25)"
            strokeWidth={2.5}
            strokeDasharray="2 9"
            strokeLinecap="round"
          />
        ))}
        {/* pulses: in from services, out to the world, two waves */}
        {[95, 190].map((start) =>
          SERVICES.map((s, i) => {
            const o = svcOut(i);
            const t = pulseT(start + i * 5, 22);
            if (frame < start + i * 5 || t >= 1) return null;
            return (
              <circle
                key={`${s}-${start}`}
                cx={o.x + (sparrowIn.x - o.x) * t}
                cy={o.y + (sparrowIn.y - o.y) * t}
                r={7}
                fill={C.coral}
              />
            );
          }),
        )}
        {[122, 217].map((start) =>
          OUTSIDE.map((o, i) => {
            const t = pulseT(start + i * 4, 20);
            if (frame < start + i * 4 || t >= 1) return null;
            return (
              <circle
                key={`${o.name}-${start}`}
                cx={sparrowOut.x + (OUT_X - sparrowOut.x) * t}
                cy={sparrowOut.y + (outY(i) - sparrowOut.y) * t}
                r={7}
                fill={C.teal}
              />
            );
          }),
        )}
      </svg>

      {/* Boundary labels */}
      <div
        style={{
          position: "absolute",
          top: BOX.y + 14,
          left: BOX.x + 26,
          fontFamily: F.mono,
          fontSize: 17,
          letterSpacing: "0.14em",
          color: C.ink,
          opacity: 0.55 * sp(6),
        }}
      >
        YOUR INFRASTRUCTURE
      </div>
      <div
        style={{
          position: "absolute",
          top: OUT_TOP - 42,
          left: OUT_X + 4,
          fontFamily: F.mono,
          fontSize: 17,
          letterSpacing: "0.14em",
          color: C.ink,
          opacity: 0.55 * sp(40),
        }}
      >
        OUTSIDE WORLD
      </div>

      {/* Service nodes */}
      {SERVICES.map((s, i) => (
        <div
          key={s}
          style={{
            ...node,
            left: SVC_X,
            top: SVC_TOP + i * (SVC_H + SVC_GAP),
            width: SVC_W,
            height: SVC_H,
            display: "flex",
            alignItems: "center",
            paddingLeft: 24,
            fontSize: 24,
            fontWeight: 500,
            ...pop(14 + i * 6),
          }}
        >
          {s}
        </div>
      ))}

      {/* Sparrow + Postgres */}
      <div
        style={{
          ...node,
          left: SPARROW.x,
          top: SPARROW.y,
          width: SPARROW.w,
          height: SPARROW.h,
          border: "2px solid rgba(0,173,216,0.5)",
          textAlign: "center",
          ...pop(34),
        }}
      >
        <div style={{ paddingTop: 30 }}>
          <Img
            src={staticFile("sparrow-logo.svg")}
            style={{ width: 70, height: 56, display: "block", margin: "0 auto 6px" }}
          />
          <div style={{ fontSize: 40, fontWeight: 600, letterSpacing: "-0.02em" }}>Sparrow</div>
          <div
            style={{
              display: "inline-block",
              marginTop: 20,
              maxWidth: SPARROW.w - 48,
              fontFamily: F.mono,
              fontSize: 16,
              lineHeight: 1.45,
              padding: "9px 20px",
              borderRadius: 999,
              border: "1px solid rgba(11,15,20,0.18)",
              background: "rgba(0,173,216,0.08)",
            }}
          >
            PostgreSQL &#8212; the only dependency
          </div>
        </div>
      </div>

      {/* Outside-world nodes */}
      {OUTSIDE.map((o, i) => (
        <div
          key={o.name}
          style={{
            ...node,
            left: OUT_X,
            top: OUT_TOP + i * (OUT_H + OUT_GAP),
            width: OUT_W,
            height: OUT_H,
            display: "flex",
            alignItems: "center",
            gap: 12,
            paddingLeft: 22,
            fontSize: 22,
            fontWeight: 500,
            ...pop(44 + i * 6),
          }}
        >
          <Img src={staticFile(o.icon)} style={{ width: 22, height: 22 }} />
          {o.name}
        </div>
      ))}

      {/* Pull one-liner */}
      <div
        style={{
          position: "absolute",
          top: 690,
          width: "100%",
          display: "flex",
          justifyContent: "center",
          opacity: pillIn,
          transform: `translateY(${(1 - pillIn) * 18}px)`,
        }}
      >
        <div
          style={{
            fontFamily: F.mono,
            fontSize: 30,
            background: C.navy,
            color: C.cream,
            padding: "20px 34px",
            borderRadius: 14,
            boxShadow: "0 16px 40px rgba(11,15,20,0.18)",
          }}
        >
          <span style={{ color: C.teal }}>$ </span>
          {CMD.slice(0, typedN)}
          <span style={{ opacity: caretOn && typedN < CMD.length ? 1 : 0 }}>&#9608;</span>
        </div>
      </div>

      <div
        style={{
          position: "absolute",
          top: 790,
          width: "100%",
          textAlign: "center",
          fontFamily: F.display,
          fontSize: 28,
          color: C.ink,
          opacity: 0.75 * subIn,
        }}
      >
        distroless image &#183; ~15 MB &#183; runs anywhere you can run one container
      </div>

      <div
        style={{
          position: "absolute",
          top: 856,
          width: "100%",
          textAlign: "center",
          fontFamily: F.mono,
          fontSize: 22,
          color: C.ink,
          opacity: 0.65 * footerIn,
        }}
      >
        Free &amp; open source &#183; MIT &#183; github.com/sarathsp06/sparrow
      </div>
    </AbsoluteFill>
  );
};

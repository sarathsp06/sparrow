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
import { Kicker } from "../components/Kicker";
import { Comet, Ripple, Wire } from "../components/Wire";

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
    <AbsoluteFill>
      <Kicker text="ONE BINARY. ONE DATABASE. SELF-HOSTED." />

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
        {SERVICES.map((s, i) => (
          <Wire key={s} id={`in-${i}`} a={svcOut(i)} b={sparrowIn} frame={frame} drawStart={55} drawEnd={75} />
        ))}
        {OUTSIDE.map((o, i) => (
          <Wire
            key={o.name}
            id={`out-${i}`}
            a={sparrowOut}
            b={{ x: OUT_X, y: outY(i) }}
            frame={frame}
            drawStart={80 + i * 2}
            drawEnd={102 + i * 2}
          />
        ))}
        {/* pulses: in from services, out to the world — keep looping so the diagram stays alive */}
        {[95, 170, 245].map((start) =>
          SERVICES.map((s, i) => (
            <Comet
              key={`${s}-${start}`}
              a={svcOut(i)}
              b={sparrowIn}
              start={start + i * 5}
              end={start + i * 5 + 22}
              frame={frame}
              r={7}
            />
          )),
        )}
        {[122, 197, 272].map((start) =>
          OUTSIDE.map((o, i) => (
            <React.Fragment key={`${o.name}-${start}`}>
              <Comet
                a={sparrowOut}
                b={{ x: OUT_X, y: outY(i) }}
                start={start + i * 4}
                end={start + i * 4 + 20}
                frame={frame}
                color={C.teal}
                r={7}
              />
              <Ripple x={OUT_X} y={outY(i)} frame={frame} at={start + i * 4 + 20} size={36} />
            </React.Fragment>
          )),
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
          translate: `0px ${(1 - pillIn) * 18}px`,
          scale: String(0.94 + pillIn * 0.06),
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
            boxShadow: `0 16px 40px rgba(11,15,20,0.18), 0 0 ${
              typedN >= CMD.length ? 36 : 0
            }px rgba(0,173,216,0.35)`,
            border: `1.5px solid rgba(0,173,216,${typedN >= CMD.length ? 0.6 : 0})`,
          }}
        >
          <span style={{ color: C.teal }}>$ </span>
          {CMD.slice(0, typedN)}
          <span style={{ opacity: caretOn && typedN < CMD.length ? 1 : 0 }}>&#9608;</span>
          <span
            style={{
              marginLeft: 18,
              color: C.teal,
              opacity: interpolate(frame, [212, 220], [0, 1], { extrapolateLeft: "clamp", extrapolateRight: "clamp" }),
            }}
          >
            &#10003;
          </span>
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

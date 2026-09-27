import React from "react";
import {
  AbsoluteFill,
  Img,
  interpolate,
  Sequence,
  staticFile,
  useCurrentFrame,
} from "remotion";
import { Audio } from "@remotion/media";
import { C, F } from "../../theme";
import { cueFrame, useReveal, ease } from "../timing";

// Frame 3 — Follow one event
// Match-cut from Frame 2's ending lockup: the Sparrow lockup is already
// present centred (no subtitle), then lifts and shrinks. The coral
// `order.created` pill with JSON preview pops in centre. Journey line draws
// with `push` / `delivered ✓` ends and five faint station dots.
//
// Transparent over the persistent cream Backdrop.

const ID = "03-meet";
const c0 = cueFrame(ID, 0); // "Let's follow one event, order.created,"
const c1 = cueFrame(ID, 1); // "from push, to delivered."

export const Meet: React.FC = () => {
  const frame = useCurrentFrame();
  const r = useReveal();

  // --- Lockup ---
  // Already visible at frame 0 (match-cut from Frame 2's lockup).
  // At c0 it lifts to the upper third and shrinks.
  const liftP = ease(frame, c0, c0 + 22);
  const lockupY = interpolate(liftP, [0, 1], [380, 140]);
  const lockupScale = interpolate(liftP, [0, 1], [1, 0.5]);

  // --- Event pill ---
  // Pops in at c0 with a slight delay (after "order.created" is spoken).
  // "order.created" appears ~60% into c0.
  const pillDelay = c0 + 16;
  const pillIn = r(pillDelay);

  // --- Journey line ---
  // Draws at c1: "from push, to delivered."
  const lineDrawP = ease(frame, c1, c1 + 20);

  const STATIONS = 5;
  const lineY = 620;
  const lineLeft = 440;
  const lineRight = 1480;
  const lineWidth = lineRight - lineLeft;

  return (
    <AbsoluteFill style={{ overflow: "hidden" }}>
      {/* Lockup: logo + wordmark (no subtitle — match-cut from Frame 2's end) */}
      <div
        style={{
          position: "absolute",
          top: lockupY,
          left: 0,
          right: 0,
          display: "flex",
          flexDirection: "column",
          alignItems: "center",
          transform: `scale(${lockupScale})`,
          transformOrigin: "center top",
        }}
      >
        {/* Logo */}
        <Img
          src={staticFile("sparrow-logo.svg")}
          style={{
            width: 100,
            height: 80,
          }}
        />

        {/* Wordmark */}
        <div
          style={{
            marginTop: 12,
            fontFamily: F.display,
            fontWeight: 700,
            fontSize: 120,
            lineHeight: 1.05,
            letterSpacing: "-0.04em",
            color: C.ink,
          }}
        >
          Sparrow
        </div>
      </div>

      {/* Event pill: order.created with JSON preview */}
      {frame >= pillDelay && (
        <div
          style={{
            position: "absolute",
            top: 400,
            left: "50%",
            transform: `translateX(-50%) translateY(${(1 - pillIn) * 24}px)`,
            opacity: pillIn,
            filter: `blur(${(1 - pillIn) * 8}px)`,
            display: "flex",
            flexDirection: "column",
            alignItems: "center",
            gap: 12,
          }}
        >
          {/* Pill */}
          <div
            style={{
              padding: "14px 32px",
              borderRadius: 999,
              background: C.coral,
              color: "#fff",
              fontFamily: F.mono,
              fontSize: 28,
              fontWeight: 700,
              letterSpacing: "0.02em",
              boxShadow: "0 8px 28px rgba(249,115,22,0.35)",
            }}
          >
            order.created
          </div>

          {/* JSON preview */}
          <div
            style={{
              padding: "12px 20px",
              borderRadius: 12,
              background: "rgba(11,15,20,0.06)",
              border: "1px solid rgba(11,15,20,0.1)",
              fontFamily: F.mono,
              fontSize: 17,
              color: "rgba(11,15,20,0.6)",
              lineHeight: 1.5,
              whiteSpace: "pre",
            }}
          >
            {`{ "order_id": "ord_123", "total": 49.99 }`}
          </div>
        </div>
      )}

      {/* Journey line: push → delivered */}
      {frame >= c1 && (
        <svg
          width={1920}
          height={200}
          style={{ position: "absolute", top: lineY - 30, left: 0 }}
        >
          {/* Main line */}
          <line
            x1={lineLeft}
            y1={50}
            x2={lineLeft + lineWidth * lineDrawP}
            y2={50}
            stroke={C.ink}
            strokeWidth={3}
            strokeLinecap="round"
            opacity={0.25}
          />

          {/* Five station dots */}
          {Array.from({ length: STATIONS }).map((_, i) => {
            const dotX =
              lineLeft + (lineWidth / (STATIONS + 1)) * (i + 1);
            const dotP = ease(frame, c1 + 6 + i * 3, c1 + 12 + i * 3);
            return (
              <circle
                key={i}
                cx={dotX}
                cy={50}
                r={5}
                fill={C.ink}
                opacity={dotP * 0.2}
              />
            );
          })}
        </svg>
      )}

      {/* Labels: "push" (left, coral) and "delivered ✓" (right, teal) */}
      {frame >= c1 && (
        <>
          <div
            style={{
              position: "absolute",
              top: lineY + 30,
              left: lineLeft - 10,
              fontFamily: F.mono,
              fontSize: 22,
              fontWeight: 700,
              letterSpacing: "0.08em",
              color: C.coralText,
              opacity: ease(frame, c1 + 2, c1 + 10),
            }}
          >
            push
          </div>
          <div
            style={{
              position: "absolute",
              top: lineY + 30,
              left: lineRight - 90,
              fontFamily: F.mono,
              fontSize: 22,
              fontWeight: 700,
              letterSpacing: "0.08em",
              color: C.tealDeep,
              opacity: ease(frame, c1 + 14, c1 + 22),
            }}
          >
            delivered ✓
          </div>
        </>
      )}

      {/* SFX: chime when the event pill pops in */}
      <Sequence from={pillDelay + 4}>
        <Audio src={staticFile("sfx/chime.mp3")} volume={0.18} />
      </Sequence>
    </AbsoluteFill>
  );
};

import React from "react";
import { AbsoluteFill, interpolate, Sequence, useCurrentFrame } from "remotion";
import { Audio } from "@remotion/media";
import { staticFile } from "remotion";
import { C, F } from "../../theme";
import { StepRail, StepTitle, card } from "../ui";
import { cueFrame, useReveal, ease } from "../timing";
import { Comet, Ripple } from "../../components/Wire";

const ID = "07-classify";

// The 10 error categories — repo-verified (docs/.../reference/error-classification.md)
const CATEGORIES = [
  { name: "success", retryable: null },
  { name: "server_error", retryable: true },
  { name: "timeout", retryable: true },
  { name: "connection_refused", retryable: true },
  { name: "network_error", retryable: true },
  { name: "rate_limited", retryable: true },
  { name: "client_error", retryable: false },
  { name: "dns_error", retryable: false },
  { name: "tls_error", retryable: false },
  { name: "unexpected_status", retryable: false },
] as const;

// Index arrays for retry/stop bin sorting


// Loose cloud positions for the 10 tokens (relative to a 900x340 box)
const CLOUD: { x: number; y: number }[] = [
  { x: 380, y: 10 },   // success
  { x: 50, y: 40 },    // server_error
  { x: 260, y: 60 },   // timeout
  { x: 500, y: 50 },   // connection_refused
  { x: 710, y: 30 },   // network_error
  { x: 140, y: 140 },  // rate_limited
  { x: 360, y: 150 },  // client_error
  { x: 570, y: 140 },  // dns_error
  { x: 740, y: 140 },  // tls_error
  { x: 180, y: 230 },  // unexpected_status
];

// Bin target positions (relative to bin box origins)
const RETRY_TARGETS: { x: number; y: number }[] = [
  { x: 20, y: 48 },
  { x: 170, y: 48 },
  { x: 20, y: 98 },
  { x: 170, y: 98 },
  { x: 90, y: 148 },
];
const STOP_TARGETS: { x: number; y: number }[] = [
  { x: 20, y: 48 },
  { x: 170, y: 48 },
  { x: 20, y: 98 },
  { x: 170, y: 98 },
];

// Map categories to their cloud index
const RETRYABLE_INDICES = [1, 2, 3, 4, 5]; // server_error, timeout, connection_refused, network_error, rate_limited
const TERMINAL_INDICES = [6, 7, 8, 9]; // client_error, dns_error, tls_error, unexpected_status

const Chip: React.FC<{
  name: string;
  x: number;
  y: number;
  opacity: number;
  scale?: number;
  highlight?: string;
}> = ({ name, x, y, opacity, scale = 1, highlight }) => (
  <div
    style={{
      position: "absolute",
      left: x,
      top: y,
      fontFamily: F.mono,
      fontSize: 19,
      fontWeight: 500,
      padding: "8px 16px",
      borderRadius: 10,
      background: highlight || "rgba(255,255,255,0.8)",
      border: `1.5px solid ${highlight ? "transparent" : "rgba(11,15,20,0.18)"}`,
      color: highlight ? "#fff" : C.ink,
      opacity,
      transform: `scale(${scale})`,
      whiteSpace: "nowrap",
    }}
  >
    {name}
  </div>
);

export const Classify: React.FC = () => {
  const frame = useCurrentFrame();
  const r = useReveal();

  const c0 = cueFrame(ID, 0);
  const c1 = cueFrame(ID, 1);
  const c2 = cueFrame(ID, 2);
  const c3 = cueFrame(ID, 3);
  const c4 = cueFrame(ID, 4);

  // Cloud box origin
  const cloudX = 480;
  const cloudY = 340;

  // Bin positions
  const retryBinX = 480;
  const retryBinY = 530;
  const stopBinX = 1120;
  const stopBinY = 530;

  // c0: request arrow animation
  const arrowProgress = ease(frame, c0, c0 + 20);
  const arrowEndX = 480 + arrowProgress * 500;

  // Endpoint card
  const endpointReveal = r(c0 + 8);

  // c1: tokens spray in — stagger over 18 frames
  const tokenReveals = CATEGORIES.map((_, i) =>
    r(c1 + i * 2, 18),
  );

  // Count ticks up
  const countVal = Math.min(
    10,
    Math.round(
      interpolate(frame, [c1, c1 + 20], [0, 10], {
        extrapolateLeft: "clamp",
        extrapolateRight: "clamp",
      }),
    ),
  );

  // c2: retryable tokens fly to left bin
  const retryBinReveal = r(c2, 20);
  const retryFly = RETRYABLE_INDICES.map((_, i) =>
    ease(frame, c2 + 4 + i * 4, c2 + 18 + i * 4),
  );

  // c3: terminal tokens fly to right bin
  const stopBinReveal = r(c3, 20);
  const stopFly = TERMINAL_INDICES.map((_, i) =>
    ease(frame, c3 + 4 + i * 4, c3 + 18 + i * 4),
  );

  // c4: success token goes to checkmark, stop bin locks
  const successFly = ease(frame, c4, c4 + 16);
  const lockReveal = r(c4 + 8, 18);

  // Compute interpolated positions for flying tokens
  const getTokenPos = (cloudIdx: number, targetX: number, targetY: number, flyProgress: number) => {
    const startX = cloudX + CLOUD[cloudIdx].x;
    const startY = cloudY + CLOUD[cloudIdx].y;
    return {
      x: interpolate(flyProgress, [0, 1], [startX, targetX], { extrapolateRight: "clamp", extrapolateLeft: "clamp" }),
      y: interpolate(flyProgress, [0, 1], [startY, targetY], { extrapolateRight: "clamp", extrapolateLeft: "clamp" }),
    };
  };

  return (
    <AbsoluteFill>
      <StepRail active={3} />
      <StepTitle n={4} title="Send & classify" at={c0} />

      {/* c0: Request arrow leaving toward endpoint */}
      {frame >= c0 && (
        <svg
          width={1920}
          height={1080}
          style={{ position: "absolute", inset: 0, pointerEvents: "none" }}
        >
          {/* Arrow line */}
          <line
            x1={480}
            y1={340}
            x2={arrowEndX}
            y2={340}
            stroke={C.coral}
            strokeWidth={3}
            opacity={arrowProgress}
          />
          {/* Arrowhead */}
          {arrowProgress > 0.3 && (
            <polygon
              points={`${arrowEndX},${340} ${arrowEndX - 14},${332} ${arrowEndX - 14},${348}`}
              fill={C.coral}
              opacity={arrowProgress}
            />
          )}
          {/* Comet along the arrow */}
          <Comet
            a={{ x: 480, y: 340 }}
            b={{ x: 980, y: 340 }}
            frame={frame}
            start={c0 + 2}
            end={c0 + 18}
            color={C.coral}
            r={8}
          />
          {/* Ripple at endpoint */}
          <Ripple x={980} y={340} frame={frame} at={c0 + 18} color={C.red} size={40} />
        </svg>
      )}

      {/* Endpoint card */}
      <div
        style={{
          position: "absolute",
          left: 980,
          top: 306,
          ...card,
          padding: "14px 24px",
          opacity: endpointReveal,
          transform: `translateX(${(1 - endpointReveal) * 20}px)`,
          display: "flex",
          alignItems: "center",
          gap: 12,
        }}
      >
        <div
          style={{
            width: 12,
            height: 12,
            borderRadius: "50%",
            background: C.red,
            boxShadow: `0 0 8px ${C.red}`,
          }}
        />
        <span style={{ fontFamily: F.mono, fontSize: 18, color: C.red }}>503</span>
      </div>

      {/* c1: ten category tokens in a loose cloud */}
      {frame >= c1 && CATEGORIES.map((cat, i) => {
        // After flying, tokens move to bin positions
        const isRetryable = RETRYABLE_INDICES.includes(i);
        const isTerminal = TERMINAL_INDICES.includes(i);
        const isSuccess = i === 0;

        let posX = cloudX + CLOUD[i].x;
        let posY = cloudY + CLOUD[i].y;
        const opacity = tokenReveals[i];
        const chipScale = 0.85 + tokenReveals[i] * 0.15;
        let highlight: string | undefined;

        if (isRetryable && frame >= c2) {
          const flyIdx = RETRYABLE_INDICES.indexOf(i);
          const fly = retryFly[flyIdx];
          const targetX = retryBinX + RETRY_TARGETS[flyIdx].x;
          const targetY = retryBinY + RETRY_TARGETS[flyIdx].y;
          const pos = getTokenPos(i, targetX, targetY, fly);
          posX = pos.x;
          posY = pos.y;
          if (fly > 0.5) highlight = C.tealDeep;
        }

        if (isTerminal && frame >= c3) {
          const flyIdx = TERMINAL_INDICES.indexOf(i);
          const fly = stopFly[flyIdx];
          const targetX = stopBinX + STOP_TARGETS[flyIdx].x;
          const targetY = stopBinY + STOP_TARGETS[flyIdx].y;
          const pos = getTokenPos(i, targetX, targetY, fly);
          posX = pos.x;
          posY = pos.y;
          if (fly > 0.5) highlight = C.red;
        }

        if (isSuccess && frame >= c4) {
          const targetX = 780;
          const targetY = 280;
          const startX = cloudX + CLOUD[0].x;
          const startY = cloudY + CLOUD[0].y;
          posX = interpolate(successFly, [0, 1], [startX, targetX], { extrapolateRight: "clamp", extrapolateLeft: "clamp" });
          posY = interpolate(successFly, [0, 1], [startY, targetY], { extrapolateRight: "clamp", extrapolateLeft: "clamp" });
          if (successFly > 0.5) highlight = C.teal;
        }

        return (
          <Chip
            key={cat.name}
            name={cat.name}
            x={posX}
            y={posY}
            opacity={opacity}
            scale={chipScale}
            highlight={highlight}
          />
        );
      })}

      {/* "10" count */}
      {frame >= c1 && (
        <div
          style={{
            position: "absolute",
            left: cloudX + 850,
            top: cloudY + 60,
            fontFamily: F.display,
            fontSize: 64,
            fontWeight: 700,
            color: C.coralText,
            opacity: ease(frame, c1 + 10, c1 + 22),
          }}
        >
          {countVal}
        </div>
      )}

      {/* c2: Retry bin */}
      {frame >= c2 - 4 && (
        <div
          style={{
            position: "absolute",
            left: retryBinX,
            top: retryBinY,
            width: 360,
            height: 210,
            borderRadius: 18,
            border: `2.5px solid ${C.teal}`,
            background: `rgba(0,173,216,0.06)`,
            opacity: retryBinReveal,
            transform: `translateY(${(1 - retryBinReveal) * 16}px)`,
          }}
        >
          <div
            style={{
              position: "absolute",
              top: -18,
              left: 20,
              fontFamily: F.mono,
              fontSize: 20,
              fontWeight: 700,
              color: C.teal,
              background: C.cream,
              padding: "2px 14px",
              borderRadius: 8,
            }}
          >
            {"↻"} retry
          </div>
        </div>
      )}

      {/* c3: Stop bin */}
      {frame >= c3 - 4 && (
        <div
          style={{
            position: "absolute",
            left: stopBinX,
            top: stopBinY,
            width: 360,
            height: 210,
            borderRadius: 18,
            border: `2.5px solid ${C.red}`,
            background: `rgba(229,72,77,0.06)`,
            opacity: stopBinReveal,
            transform: `translateY(${(1 - stopBinReveal) * 16}px)`,
          }}
        >
          <div
            style={{
              position: "absolute",
              top: -18,
              left: 20,
              fontFamily: F.mono,
              fontSize: 20,
              fontWeight: 700,
              color: C.red,
              background: C.cream,
              padding: "2px 14px",
              borderRadius: 8,
            }}
          >
            {"■"} stop
          </div>
          {/* Lock icon on c4 */}
          {frame >= c4 && (
            <div
              style={{
                position: "absolute",
                top: -18,
                right: 20,
                opacity: lockReveal,
                transform: `scale(${lockReveal})`,
              }}
            >
              <svg width={24} height={28} viewBox="0 0 22 28">
                <rect x={2} y={12} width={18} height={14} rx={3} fill={C.red} />
                <path d="M6 12 V8 a5 5 0 0 1 10 0 V12" stroke={C.red} strokeWidth={2.4} fill="none" />
              </svg>
            </div>
          )}
        </div>
      )}

      {/* c4: success checkmark destination */}
      {frame >= c4 && (
        <div
          style={{
            position: "absolute",
            left: 770,
            top: 270,
            width: 44,
            height: 44,
            borderRadius: "50%",
            background: C.teal,
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            color: "#fff",
            fontSize: 22,
            fontWeight: 700,
            opacity: successFly,
            transform: `scale(${0.6 + successFly * 0.4})`,
            boxShadow: `0 0 16px rgba(0,173,216,0.5)`,
          }}
        >
          {"✓"}
        </div>
      )}

      {/* SFX: impact when tokens sort */}
      <Sequence from={c2} name="classify-sort-sfx">
        <Audio src={staticFile("sfx/ping.mp3")} volume={0.15} />
      </Sequence>
    </AbsoluteFill>
  );
};

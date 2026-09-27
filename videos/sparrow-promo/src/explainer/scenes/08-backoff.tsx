import React from "react";
import { AbsoluteFill, Sequence, useCurrentFrame } from "remotion";
import { Audio } from "@remotion/media";
import { staticFile } from "remotion";
import { C, F } from "../../theme";
import { StepRail, StepTitle, card } from "../ui";
import { cueFrame, useReveal, ease } from "../timing";
import { Ripple } from "../../components/Wire";

const ID = "08-backoff";

// Timeline layout
const TL_Y = 520;
const TL_LEFT = 200;
const TL_RIGHT = 1720;
const TL_WIDTH = TL_RIGHT - TL_LEFT;

// Proportional gaps: 1m + 2m + 4m = 7 units total
// attempt 1 at t0, then gaps for retries
const TOTAL_UNITS = 7;
const attemptX = (unit: number) => TL_LEFT + (unit / TOTAL_UNITS) * TL_WIDTH;

const ATTEMPTS = [
  { unit: 0, label: "1", status: "fail" },
  { unit: 1, label: "2", status: "fail" },
  { unit: 3, label: "3", status: "fail" },
  { unit: 7, label: "4", status: "success" },
] as const;

const GAPS = [
  { from: 0, to: 1, label: "+1m" },
  { from: 1, to: 3, label: "+2m" },
  { from: 3, to: 7, label: "+4m" },
] as const;

export const Backoff: React.FC = () => {
  const frame = useCurrentFrame();
  const r = useReveal();

  const c0 = cueFrame(ID, 0);
  const c1 = cueFrame(ID, 1);
  const c2 = cueFrame(ID, 2);
  const c3 = cueFrame(ID, 3);
  const c4 = cueFrame(ID, 4);

  // c1 word offsets: "one minute" ~1.2s, "two" ~2.4s, "four" ~3.5s into the cue
  const oneAt = c1 + Math.round(1.2 * 30);
  const twoAt = c1 + Math.round(2.4 * 30);
  const fourAt = c1 + Math.round(3.5 * 30);
  const gapReveals = [
    ease(frame, oneAt - 4, oneAt + 10),
    ease(frame, twoAt - 4, twoAt + 10),
    ease(frame, fourAt - 4, fourAt + 10),
  ];

  // Timeline base draw
  const tlDraw = ease(frame, c1 - 4, c1 + 20);

  // Attempt markers: first appears at c1, others appear with their gap labels
  const attemptReveals = [
    r(c1, 18),
    r(oneAt + 6, 18),
    r(twoAt + 6, 18),
    0, // attempt 4 appears at c4
  ];

  // c2: formula card
  const formulaReveal = r(c2, 22);

  // c2: bracket pulse — a single subtle scale pulse
  const pulsePhase = ease(frame, c2 + 6, c2 + 30);
  const pulseScale = pulsePhase < 1 ? 1 + Math.sin(pulsePhase * Math.PI) * 0.06 : 1;

  // c3: 429 response chip
  const chip429Reveal = r(c3, 20);

  // c4: snooze arc + attempt 4
  const snoozeReveal = r(c4, 22);
  const attempt4Reveal = r(c4 + 14, 18);

  return (
    <AbsoluteFill>
      <StepRail active={4} />
      <StepTitle n={5} title="Back off" at={c0} />

      {/* Timeline base line */}
      <svg
        width={1920}
        height={1080}
        style={{ position: "absolute", inset: 0, pointerEvents: "none" }}
      >
        {/* Main horizontal line */}
        <line
          x1={TL_LEFT}
          y1={TL_Y}
          x2={TL_LEFT + tlDraw * TL_WIDTH}
          y2={TL_Y}
          stroke="rgba(11,15,20,0.22)"
          strokeWidth={3}
          strokeLinecap="round"
        />

        {/* Gap brackets + labels */}
        {GAPS.map((gap, i) => {
          const x1 = attemptX(gap.from);
          const x2 = attemptX(gap.to);
          const midX = (x1 + x2) / 2;
          const gapR = gapReveals[i];
          return (
            <g key={gap.label} opacity={gapR} transform={`scale(${pulseScale})`} style={{ transformOrigin: `${midX}px ${TL_Y}px` }}>
              {/* Bracket top */}
              <line x1={x1 + 16} y1={TL_Y - 26} x2={x2 - 16} y2={TL_Y - 26} stroke={C.coralText} strokeWidth={2} />
              <line x1={x1 + 16} y1={TL_Y - 26} x2={x1 + 16} y2={TL_Y - 16} stroke={C.coralText} strokeWidth={2} />
              <line x1={x2 - 16} y1={TL_Y - 26} x2={x2 - 16} y2={TL_Y - 16} stroke={C.coralText} strokeWidth={2} />
              {/* Label */}
              <text
                x={midX}
                y={TL_Y - 38}
                textAnchor="middle"
                fontFamily={F.mono}
                fontSize={22}
                fontWeight={700}
                fill={C.coralText}
              >
                {gap.label}
              </text>
            </g>
          );
        })}

        {/* Attempt markers: 1, 2, 3 */}
        {ATTEMPTS.slice(0, 3).map((att, i) => {
          const x = attemptX(att.unit);
          const aR = attemptReveals[i];
          return (
            <g key={att.label} opacity={aR}>
              {/* Vertical tick */}
              <line x1={x} y1={TL_Y - 8} x2={x} y2={TL_Y + 8} stroke={C.red} strokeWidth={3} />
              {/* X mark */}
              <text
                x={x}
                y={TL_Y + 40}
                textAnchor="middle"
                fontFamily={F.mono}
                fontSize={28}
                fontWeight={700}
                fill={C.red}
              >
                {"✕"}
              </text>
              {/* Attempt number */}
              <text
                x={x}
                y={TL_Y + 68}
                textAnchor="middle"
                fontFamily={F.mono}
                fontSize={17}
                fill="rgba(11,15,20,0.55)"
              >
                #{att.label}
              </text>
            </g>
          );
        })}

        {/* c4: Attempt 4 (success) */}
        {frame >= c4 && (() => {
          const x = attemptX(ATTEMPTS[3].unit);
          return (
            <g opacity={attempt4Reveal}>
              <line x1={x} y1={TL_Y - 8} x2={x} y2={TL_Y + 8} stroke={C.teal} strokeWidth={3} />
              <text
                x={x}
                y={TL_Y + 40}
                textAnchor="middle"
                fontFamily={F.mono}
                fontSize={28}
                fontWeight={700}
                fill={C.teal}
              >
                {"✓"}
              </text>
              <text
                x={x}
                y={TL_Y + 68}
                textAnchor="middle"
                fontFamily={F.mono}
                fontSize={17}
                fill="rgba(11,15,20,0.55)"
              >
                #4
              </text>
              <Ripple x={x} y={TL_Y} frame={frame} at={c4 + 16} color={C.teal} size={50} />
            </g>
          );
        })()}

        {/* c3-c4: snooze arc (dashed) jumping over from attempt 3 area */}
        {frame >= c3 && (() => {
          const arcStartX = attemptX(3) + 60;
          const arcEndX = attemptX(7) - 40;
          const arcMidX = (arcStartX + arcEndX) / 2;
          const arcY = TL_Y - 140;
          return (
            <g opacity={snoozeReveal}>
              <path
                d={`M ${arcStartX} ${TL_Y - 30} Q ${arcMidX} ${arcY}, ${arcEndX} ${TL_Y - 30}`}
                fill="none"
                stroke={C.tealDeep}
                strokeWidth={2.5}
                strokeDasharray="8 6"
                strokeLinecap="round"
              />
              {/* zZ label */}
              <text
                x={arcMidX}
                y={arcY - 6}
                textAnchor="middle"
                fontFamily={F.mono}
                fontSize={22}
                fontWeight={600}
                fill={C.tealDeep}
              >
                {"zZ"}
              </text>
            </g>
          );
        })()}
      </svg>

      {/* c3: 429 response chip — above the timeline on a side lane */}
      {frame >= c3 - 4 && (
        <div
          style={{
            position: "absolute",
            left: attemptX(3) + 80,
            top: TL_Y - 180,
            ...card,
            padding: "14px 22px",
            opacity: chip429Reveal,
            transform: `translateY(${(1 - chip429Reveal) * 14}px)`,
            display: "flex",
            alignItems: "center",
            gap: 14,
            borderColor: "rgba(249,115,22,0.35)",
          }}
        >
          <span style={{ fontFamily: F.mono, fontSize: 20, fontWeight: 700, color: C.coralText }}>429</span>
          <span style={{ fontFamily: F.mono, fontSize: 17, color: "rgba(11,15,20,0.6)" }}>
            Too Many Requests
          </span>
          <span style={{ fontFamily: F.mono, fontSize: 17, color: C.tealDeep, marginLeft: 8 }}>
            Retry-After: 30
          </span>
        </div>
      )}

      {/* c4: attempt counter showing 3/4 (not incremented) */}
      {frame >= c4 - 4 && (
        <div
          style={{
            position: "absolute",
            left: attemptX(5),
            top: TL_Y + 90,
            fontFamily: F.mono,
            fontSize: 20,
            color: C.tealDeep,
            opacity: snoozeReveal,
            background: "rgba(14,116,144,0.08)",
            padding: "6px 16px",
            borderRadius: 8,
          }}
        >
          attempts: 3 / 4
        </div>
      )}

      {/* c2: formula card */}
      {frame >= c2 - 4 && (
        <div
          style={{
            position: "absolute",
            left: 480,
            top: TL_Y + 110,
            ...card,
            padding: "22px 32px",
            opacity: formulaReveal,
            transform: `translateY(${(1 - formulaReveal) * 14}px)`,
          }}
        >
          <span style={{ fontFamily: F.mono, fontSize: 26, fontWeight: 600, color: C.ink }}>
            {"delay = 60s × 2"}
          </span>
          <span style={{ fontFamily: F.mono, fontSize: 18, fontWeight: 600, color: C.ink, verticalAlign: "super" }}>
            {"n–1"}
          </span>
          <span style={{ fontFamily: F.mono, fontSize: 22, color: "rgba(11,15,20,0.5)", marginLeft: 16 }}>
            {"(max 24h)"}
          </span>
        </div>
      )}

      {/* SFX */}
      <Sequence from={c4 + 16} name="backoff-success-sfx">
        <Audio src={staticFile("sfx/chime.mp3")} volume={0.18} />
      </Sequence>
    </AbsoluteFill>
  );
};

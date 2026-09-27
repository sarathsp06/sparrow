import React from "react";
import { AbsoluteFill, interpolate, Easing, Sequence, useCurrentFrame } from "remotion";
import { Audio } from "@remotion/media";
import { staticFile } from "remotion";
import { C, F } from "../../theme";
import { card } from "../ui";
import { cueFrame, useReveal, ease } from "../timing";
import { Kicker } from "../../components/Kicker";

const ID = "09-health";

// Health colors
const AMBER = "#F5A524";
const TEAL = C.teal;
const RED = C.red;

// Gauge geometry — semicircular arc centered at (cx, cy) with radius r
const GAUGE_CX = 700;
const GAUGE_CY = 620;
const GAUGE_R = 220;

// Angles: 180deg = left (0%), 0deg = right (100%). Clockwise from left.
// Segments: <80% (red), 80-90% (amber), 90-100% (teal)
const pctToAngle = (pct: number) => Math.PI * (1 - pct / 100);

const arcPath = (startPct: number, endPct: number, r: number) => {
  const a1 = pctToAngle(startPct);
  const a2 = pctToAngle(endPct);
  const x1 = GAUGE_CX + r * Math.cos(a1);
  const y1 = GAUGE_CY - r * Math.sin(a1);
  const x2 = GAUGE_CX + r * Math.cos(a2);
  const y2 = GAUGE_CY - r * Math.sin(a2);
  const largeArc = Math.abs(a2 - a1) > Math.PI ? 1 : 0;
  return `M ${x1} ${y1} A ${r} ${r} 0 ${largeArc} 1 ${x2} ${y2}`;
};

const needleAt = (pct: number) => {
  const a = pctToAngle(pct);
  return {
    x: GAUGE_CX + (GAUGE_R - 30) * Math.cos(a),
    y: GAUGE_CY - (GAUGE_R - 30) * Math.sin(a),
  };
};

// State badges
const STATES = {
  unknown: { color: "rgba(11,15,20,0.4)", bg: "rgba(11,15,20,0.06)" },
  healthy: { color: TEAL, bg: "rgba(0,173,216,0.1)" },
  degraded: { color: AMBER, bg: "rgba(245,165,36,0.1)" },
  unhealthy: { color: RED, bg: "rgba(229,72,77,0.1)" },
} as const;

export const Health: React.FC = () => {
  const frame = useCurrentFrame();
  const r = useReveal();

  const c0 = cueFrame(ID, 0);
  const c1 = cueFrame(ID, 1);
  const c2 = cueFrame(ID, 2);
  const c3 = cueFrame(ID, 3);
  const c4 = cueFrame(ID, 4);

  // Webhook card + badge state
  const cardReveal = r(c0, 22);

  // Determine current state
  let currentState: keyof typeof STATES = "unknown";
  if (frame >= c3) currentState = "unhealthy";
  else if (frame >= c2) currentState = "degraded";
  else if (frame >= c1) currentState = "healthy";

  const stateInfo = STATES[currentState];

  // Gauge segments draw in
  const tealSegDraw = ease(frame, c1, c1 + 18);
  const amberSegDraw = ease(frame, c2, c2 + 18);
  const redSegDraw = ease(frame, c3 + 6, c3 + 24);

  // Needle position: sweeps to target percentage
  const needlePct = (() => {
    if (frame < c1) return 50;
    if (frame < c2) return interpolate(frame, [c1, c1 + 20], [50, 96], { extrapolateLeft: "clamp", extrapolateRight: "clamp", easing: Easing.out(Easing.cubic) });
    if (frame < c3) return interpolate(frame, [c2, c2 + 20], [96, 86], { extrapolateLeft: "clamp", extrapolateRight: "clamp", easing: Easing.out(Easing.cubic) });
    return interpolate(frame, [c3, c3 + 20], [86, 72], { extrapolateLeft: "clamp", extrapolateRight: "clamp", easing: Easing.out(Easing.cubic) });
  })();

  const needlePos = needleAt(needlePct);
  const gaugeReveal = r(c1 - 4, 22);

  // c3: five failure dots
  const failDots = Array.from({ length: 5 }, (_, i) =>
    ease(frame, c3 + i * 4, c3 + i * 4 + 8),
  );

  // c4: email card
  const emailReveal = r(c4, 22);

  // Segment labels
  const tealLabelReveal = r(c1 + 10, 18);
  const amberLabelReveal = r(c2 + 10, 18);
  const redLabelReveal = r(c3 + 16, 18);

  return (
    <AbsoluteFill>
      <Kicker text="PER-WEBHOOK HEALTH" delay={c0} />

      {/* Webhook card */}
      <div
        style={{
          position: "absolute",
          left: 400,
          top: 150,
          ...card,
          padding: "22px 32px",
          opacity: cardReveal,
          transform: `translateY(${(1 - cardReveal) * 14}px)`,
          display: "flex",
          alignItems: "center",
          gap: 18,
          minWidth: 520,
        }}
      >
        <div>
          <div style={{ fontFamily: F.display, fontSize: 26, fontWeight: 600, color: C.ink }}>
            partner-acme
          </div>
          <div style={{ fontFamily: F.mono, fontSize: 16, color: "rgba(11,15,20,0.5)", marginTop: 4 }}>
            https://api.acme.example/hooks
          </div>
        </div>
        {/* State badge */}
        <div
          style={{
            marginLeft: "auto",
            fontFamily: F.mono,
            fontSize: 17,
            fontWeight: 600,
            padding: "6px 16px",
            borderRadius: 999,
            color: stateInfo.color,
            background: stateInfo.bg,
            border: `1.5px solid ${stateInfo.color}`,
          }}
        >
          {currentState}
        </div>
      </div>

      {/* Semicircular gauge */}
      <svg
        width={1920}
        height={1080}
        style={{ position: "absolute", inset: 0, pointerEvents: "none" }}
      >
        {/* Gauge track (faint) */}
        <path
          d={arcPath(0, 100, GAUGE_R)}
          fill="none"
          stroke="rgba(11,15,20,0.08)"
          strokeWidth={28}
          strokeLinecap="round"
          opacity={gaugeReveal}
        />

        {/* Teal segment: 90-100% */}
        {frame >= c1 && (
          <path
            d={arcPath(90, 90 + tealSegDraw * 10, GAUGE_R)}
            fill="none"
            stroke={TEAL}
            strokeWidth={28}
            strokeLinecap="round"
            opacity={0.85}
          />
        )}

        {/* Amber segment: 80-90% */}
        {frame >= c2 && (
          <path
            d={arcPath(80, 80 + amberSegDraw * 10, GAUGE_R)}
            fill="none"
            stroke={AMBER}
            strokeWidth={28}
            strokeLinecap="round"
            opacity={0.85}
          />
        )}

        {/* Red segment: 0-80% */}
        {frame >= c3 + 4 && (
          <path
            d={arcPath(0, redSegDraw * 80, GAUGE_R)}
            fill="none"
            stroke={RED}
            strokeWidth={28}
            strokeLinecap="round"
            opacity={0.85}
          />
        )}

        {/* Needle */}
        {frame >= c1 && (
          <g opacity={gaugeReveal}>
            <line
              x1={GAUGE_CX}
              y1={GAUGE_CY}
              x2={needlePos.x}
              y2={needlePos.y}
              stroke={C.ink}
              strokeWidth={4}
              strokeLinecap="round"
            />
            <circle cx={GAUGE_CX} cy={GAUGE_CY} r={10} fill={C.ink} />
            {/* Needle tip dot */}
            <circle cx={needlePos.x} cy={needlePos.y} r={6} fill={stateInfo.color} />
          </g>
        )}

        {/* Percentage display */}
        {frame >= c1 && (
          <text
            x={GAUGE_CX - 40}
            y={GAUGE_CY - 40}
            textAnchor="end"
            fontFamily={F.display}
            fontSize={52}
            fontWeight={700}
            fill={stateInfo.color}
            opacity={gaugeReveal}
          >
            {Math.round(needlePct)}%
          </text>
        )}
      </svg>

      {/* Segment labels */}
      {frame >= c1 && (
        <div
          style={{
            position: "absolute",
            left: GAUGE_CX + GAUGE_R + 40,
            top: GAUGE_CY - 190,
            fontFamily: F.mono,
            fontSize: 19,
            color: TEAL,
            opacity: tealLabelReveal,
            fontWeight: 600,
          }}
        >
          {"healthy > 90%"}
        </div>
      )}
      {frame >= c2 && (
        <div
          style={{
            position: "absolute",
            left: GAUGE_CX + GAUGE_R + 40,
            top: GAUGE_CY - 150,
            fontFamily: F.mono,
            fontSize: 19,
            color: AMBER,
            opacity: amberLabelReveal,
            fontWeight: 600,
          }}
        >
          {"degraded 80–90%"}
        </div>
      )}
      {frame >= c3 + 4 && (
        <div
          style={{
            position: "absolute",
            left: GAUGE_CX + GAUGE_R + 40,
            top: GAUGE_CY - 110,
            fontFamily: F.mono,
            fontSize: 19,
            color: RED,
            opacity: redLabelReveal,
            fontWeight: 600,
          }}
        >
          {"unhealthy < 80%"}
        </div>
      )}

      {/* c3: five consecutive failure dots */}
      {frame >= c3 && (
        <div
          style={{
            position: "absolute",
            left: GAUGE_CX - 100,
            top: GAUGE_CY + 30,
            display: "flex",
            gap: 16,
            alignItems: "center",
          }}
        >
          {failDots.map((d, i) => (
            <div
              key={i}
              style={{
                width: 28,
                height: 28,
                borderRadius: "50%",
                border: `2px solid ${RED}`,
                background: d > 0.5 ? `rgba(229,72,77,${d * 0.25})` : "transparent",
                display: "flex",
                alignItems: "center",
                justifyContent: "center",
                fontFamily: F.mono,
                fontSize: 16,
                fontWeight: 700,
                color: RED,
                opacity: d,
                transform: `scale(${0.6 + d * 0.4})`,
              }}
            >
              {"✕"}
            </div>
          ))}
          <span
            style={{
              fontFamily: F.mono,
              fontSize: 17,
              color: "rgba(11,15,20,0.5)",
              marginLeft: 8,
              opacity: failDots[4],
            }}
          >
            5 consecutive
          </span>
        </div>
      )}

      {/* c4: email alert card */}
      {frame >= c4 - 4 && (
        <div
          style={{
            position: "absolute",
            right: 120,
            top: 360,
            width: 440,
            ...card,
            padding: "28px 32px",
            opacity: emailReveal,
            transform: `translateX(${(1 - emailReveal) * 40}px)`,
            borderLeft: `4px solid ${RED}`,
          }}
        >
          <div style={{ fontFamily: F.mono, fontSize: 15, color: "rgba(11,15,20,0.45)", marginBottom: 8 }}>
            alert
          </div>
          <div style={{ fontFamily: F.mono, fontSize: 18, fontWeight: 600, color: C.ink, marginBottom: 10 }}>
            sparrow.webhook.health_changed
          </div>
          <div style={{ fontFamily: F.display, fontSize: 19, color: C.ink, lineHeight: 1.5 }}>
            <span style={{ fontWeight: 600 }}>partner-acme</span>:{" "}
            <span style={{ color: AMBER }}>degraded</span>{" → "}
            <span style={{ color: RED }}>unhealthy</span>
          </div>
          <div style={{ fontFamily: F.mono, fontSize: 15, color: "rgba(11,15,20,0.45)", marginTop: 12 }}>
            {"to: oncall@acme.example.com"}
          </div>

          {/* Email icon */}
          <div
            style={{
              position: "absolute",
              top: -16,
              right: 20,
              width: 36,
              height: 36,
              borderRadius: "50%",
              background: RED,
              display: "flex",
              alignItems: "center",
              justifyContent: "center",
            }}
          >
            <svg width={18} height={14} viewBox="0 0 18 14" fill="none">
              <rect x={0.5} y={0.5} width={17} height={13} rx={2} stroke="#fff" strokeWidth={1.2} />
              <path d="M1 1 L9 7.5 L17 1" stroke="#fff" strokeWidth={1.2} strokeLinecap="round" />
            </svg>
          </div>
        </div>
      )}

      {/* SFX */}
      <Sequence from={c3} name="health-alert-sfx">
        <Audio src={staticFile("sfx/impact-bass-1.mp3")} volume={0.15} />
      </Sequence>
    </AbsoluteFill>
  );
};

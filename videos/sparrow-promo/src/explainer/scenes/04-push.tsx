import React from "react";
import {
  AbsoluteFill,
  Audio,
  Sequence,
  interpolate,
  spring,
  staticFile,
  useCurrentFrame,
  useVideoConfig,
  Easing,
} from "remotion";
import { C, F } from "../../theme";
import { cueFrame, useReveal, ease } from "../timing";
import { StepRail, StepTitle, Terminal } from "../ui";

const ID = "04-push";

// Typing animation: reveals characters of `text` between frames `from` and `to`.
const TypeLine: React.FC<{
  text: string;
  from: number;
  to: number;
  color?: string;
  style?: React.CSSProperties;
}> = ({ text, from, to, color, style }) => {
  const frame = useCurrentFrame();
  const chars = Math.floor(
    interpolate(frame, [from, to], [0, text.length], {
      extrapolateLeft: "clamp",
      extrapolateRight: "clamp",
      easing: Easing.linear,
    }),
  );
  const showCaret = frame >= from && frame <= to + 6;
  return (
    <span style={{ color, ...style }}>
      {text.slice(0, chars)}
      {showCaret && (
        <span
          style={{
            opacity: Math.floor(frame / 8) % 2 === 0 ? 1 : 0.3,
            color: C.coral,
          }}
        >
          |
        </span>
      )}
    </span>
  );
};

// Small Postgres drum icon that receives the event.
const PgDrum: React.FC<{ at: number }> = ({ at }) => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();
  const s = spring({ frame: frame - at, fps, config: { damping: 200 } });
  const ripple = interpolate(frame, [at + 8, at + 32], [0, 1], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
    easing: Easing.out(Easing.cubic),
  });
  return (
    <div
      style={{
        position: "absolute",
        left: 520,
        top: 420,
        textAlign: "center",
        opacity: s,
        transform: `translateY(${(1 - s) * 16}px)`,
      }}
    >
      <svg width={100} height={80} viewBox="0 0 100 80">
        {/* Drum body */}
        <ellipse cx={50} cy={60} rx={40} ry={14} fill={C.tealDeep} opacity={0.25} />
        <rect x={10} y={20} width={80} height={40} fill={C.tealDeep} opacity={0.15} />
        <ellipse cx={50} cy={20} rx={40} ry={14} fill="none" stroke={C.tealDeep} strokeWidth={2.5} />
        <ellipse cx={50} cy={60} rx={40} ry={14} fill="none" stroke={C.tealDeep} strokeWidth={2.5} />
        <line x1={10} y1={20} x2={10} y2={60} stroke={C.tealDeep} strokeWidth={2.5} />
        <line x1={90} y1={20} x2={90} y2={60} stroke={C.tealDeep} strokeWidth={2.5} />
        {/* Ripple on store */}
        {frame >= at + 8 && ripple < 1 && (
          <ellipse
            cx={50}
            cy={40}
            rx={15 + ripple * 35}
            ry={6 + ripple * 12}
            fill="none"
            stroke={C.teal}
            strokeWidth={2.5 * (1 - ripple)}
            opacity={1 - ripple}
          />
        )}
      </svg>
      <div
        style={{
          fontFamily: F.mono,
          fontSize: 16,
          color: C.tealDeep,
          letterSpacing: "0.04em",
          marginTop: 4,
        }}
      >
        stored
      </div>
      <div
        style={{
          fontFamily: F.mono,
          fontSize: 14,
          color: C.tealDeep,
          opacity: 0.6,
        }}
      >
        Postgres
      </div>
    </div>
  );
};

export const Push: React.FC = () => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();
  const r = useReveal();

  const c0 = cueFrame(ID, 0);
  const c1 = cueFrame(ID, 1);
  const c2 = cueFrame(ID, 2);
  const c3 = cueFrame(ID, 3);

  // Terminal typing bounds
  const cmdLine1 = "curl -X POST /v1/consumers/acme/events?event=order.created";
  const cmdLine2 = `-d '{"payload":{...},"idempotency_key":"idem_ord_123"}'`;
  const typeLine1Start = c1;
  const typeLine1End = c1 + 40;
  const typeLine2Start = typeLine1End + 4;
  const typeLine2End = typeLine2Start + 30;

  // Response 1 pop
  const resp1Pop = spring({
    frame: frame - typeLine2End - 4,
    fps,
    config: { damping: 200 },
  });

  // Ghost re-type for idempotency (c2)
  const ghostStart = c2;
  const ghostEnd = c2 + 24;

  // Response 2 pop
  const resp2Pop = spring({
    frame: frame - (c3 + 4),
    fps,
    config: { damping: 200 },
  });

  // "No second delivery" stamp
  const stampPop = spring({
    frame: frame - (c3 + 22),
    fps,
    config: { damping: 200 },
  });

  // Underline matching event_id in both responses
  const matchLine = ease(frame, c3 + 12, c3 + 28);

  return (
    <AbsoluteFill>
      <StepRail active={0} />
      <StepTitle n={1} title="Push" at={c0} />

      {/* Postgres drum — appears with c1 */}
      <PgDrum at={c1 + 14} />

      {/* SFX: subtle ping on store */}
      <Sequence from={c1 + 22}>
        <Audio src={staticFile("sfx/ping.mp3")} volume={0.15} />
      </Sequence>

      {/* Terminal — right 60%, vertically centred in the diagram area */}
      <div
        style={{
          position: "absolute",
          top: 290,
          left: 680,
          width: 1140,
          opacity: r(c1),
          transform: `translateY(${(1 - r(c1)) * 18}px)`,
        }}
      >
        <Terminal title="terminal" style={{ minHeight: 420 }}>
          {/* Command typing */}
          <div>
            <span style={{ color: C.teal }}>$ </span>
            <TypeLine text={cmdLine1} from={typeLine1Start} to={typeLine1End} />
          </div>
          {frame >= typeLine2Start && (
            <div style={{ marginTop: 4 }}>
              <span style={{ color: "transparent" }}>{"  "}</span>
              <TypeLine text={cmdLine2} from={typeLine2Start} to={typeLine2End} />
            </div>
          )}

          {/* Response 1 */}
          {resp1Pop > 0.01 && (
            <div
              style={{
                marginTop: 18,
                opacity: resp1Pop,
                transform: `translateY(${(1 - resp1Pop) * 10}px)`,
              }}
            >
              <div>
                <span style={{ color: C.teal, fontWeight: 700 }}>201 Created</span>
              </div>
              <div style={{ marginTop: 6, color: "rgba(232,230,225,0.75)", fontSize: 21 }}>
                {"{"}&quot;
                <span
                  style={{
                    color: C.teal,
                    textDecoration: matchLine > 0 ? "underline" : "none",
                    textDecorationColor: C.coral,
                    textUnderlineOffset: "4px",
                  }}
                >
                  event_id
                </span>
                &quot;:&quot;
                <span
                  style={{
                    textDecoration: matchLine > 0 ? "underline" : "none",
                    textDecorationColor: C.coral,
                    textUnderlineOffset: "4px",
                  }}
                >
                  evt_7c1...
                </span>
                &quot;,&quot;duplicate&quot;:
                <span style={{ color: C.teal }}>false</span>
                {"}"}
              </div>
            </div>
          )}

          {/* Ghost re-type (c2) */}
          {frame >= ghostStart && (
            <div
              style={{
                marginTop: 22,
                opacity: ease(frame, ghostStart, ghostStart + 6),
              }}
            >
              <span style={{ color: C.teal }}>$ </span>
              <TypeLine
                text={cmdLine1}
                from={ghostStart + 2}
                to={ghostEnd}
                color="rgba(232,230,225,0.45)"
              />
            </div>
          )}
          {frame >= ghostStart + 4 && (
            <div
              style={{
                opacity: ease(frame, ghostStart + 4, ghostStart + 10),
              }}
            >
              <span style={{ color: "transparent" }}>{"  "}</span>
              <TypeLine
                text={cmdLine2}
                from={ghostEnd - 12}
                to={ghostEnd + 6}
                color="rgba(232,230,225,0.45)"
              />
            </div>
          )}

          {/* Response 2 (c3) — duplicate:true */}
          {resp2Pop > 0.01 && (
            <div
              style={{
                marginTop: 18,
                opacity: resp2Pop,
                transform: `translateY(${(1 - resp2Pop) * 10}px)`,
              }}
            >
              <div>
                <span style={{ color: C.teal, fontWeight: 700 }}>201 Created</span>
              </div>
              <div style={{ marginTop: 6, color: "rgba(232,230,225,0.75)", fontSize: 21 }}>
                {"{"}&quot;
                <span
                  style={{
                    color: C.teal,
                    textDecoration: matchLine > 0 ? "underline" : "none",
                    textDecorationColor: C.coral,
                    textUnderlineOffset: "4px",
                  }}
                >
                  event_id
                </span>
                &quot;:&quot;
                <span
                  style={{
                    textDecoration: matchLine > 0 ? "underline" : "none",
                    textDecorationColor: C.coral,
                    textUnderlineOffset: "4px",
                  }}
                >
                  evt_7c1...
                </span>
                &quot;,&quot;duplicate&quot;:
                <span
                  style={{
                    color: C.coral,
                    fontWeight: 700,
                    textShadow: `0 0 12px rgba(249,115,22,${0.4 * resp2Pop})`,
                  }}
                >
                  true
                </span>
                {"}"}
              </div>
            </div>
          )}
        </Terminal>

        {/* "No second delivery" stamp */}
        {stampPop > 0.01 && (
          <div
            style={{
              position: "absolute",
              right: 40,
              bottom: 30,
              display: "flex",
              alignItems: "center",
              gap: 10,
              fontFamily: F.mono,
              fontSize: 20,
              fontWeight: 700,
              color: C.coralText,
              background: "rgba(249,115,22,0.12)",
              border: `2px solid ${C.coral}`,
              borderRadius: 12,
              padding: "10px 20px",
              opacity: stampPop,
              transform: `scale(${0.85 + stampPop * 0.15})`,
            }}
          >
            <span style={{ fontSize: 24 }}>{"✕"}</span> no second delivery
          </div>
        )}
      </div>
    </AbsoluteFill>
  );
};

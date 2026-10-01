import React from "react";
import { AbsoluteFill, Sequence, interpolate, useCurrentFrame, staticFile, Easing } from "remotion";
import { Audio } from "@remotion/media";
import { C, F } from "../../theme";
import { cueFrame, wordFrame, useReveal } from "../timing";
import { WordsIn } from "../ui";

// Frame 10 — What it needs: one PostgreSQL drum, struck-through alternatives,
// an MIT stamp.

const ID = "10-needs";
const c0 = cueFrame(ID, 0);

const Strike: React.FC<{ frame: number; at: number; width: number }> = ({ frame, at, width }) => {
  const p = interpolate(frame, [at, at + 10], [0, 1], { extrapolateLeft: "clamp", extrapolateRight: "clamp", easing: Easing.out(Easing.cubic) });
  if (p <= 0) return null;
  return (
    <svg width={width + 24} height={12} style={{ position: "absolute", top: "50%", left: -12, marginTop: -6, pointerEvents: "none" }}>
      <line x1={0} y1={6} x2={(width + 24) * p} y2={6} stroke={C.coral} strokeWidth={5} strokeLinecap="round" />
    </svg>
  );
};

const Ghost: React.FC<{ label: string; frame: number; appearAt: number; strikeAt: number; width: number }> = ({ label, frame, appearAt, strikeAt, width }) => {
  const r = useReveal();
  const s = r(appearAt);
  const dim = interpolate(frame, [strikeAt, strikeAt + 14], [1, 0.4], { extrapolateLeft: "clamp", extrapolateRight: "clamp" });
  return (
    <div style={{ position: "relative", padding: "18px 34px", borderRadius: 16, background: "rgba(255,255,255,0.5)", border: "1px solid rgba(11,15,20,0.10)", fontFamily: F.mono, fontSize: 32, fontWeight: 600, color: C.ink, whiteSpace: "nowrap", opacity: s * dim, transform: `translateY(${(1 - s) * 14}px)`, width, textAlign: "center" }}>
      {label}
      <Strike frame={frame} at={strikeAt} width={width} />
    </div>
  );
};

export const Needs: React.FC = () => {
  const frame = useCurrentFrame();
  const r = useReveal();

  const drumAt = wordFrame(ID, 1, "just");
  const queueAt = wordFrame(ID, 1, "queue");
  const redisAt = wordFrame(ID, 1, "redis");
  const brokerAt = wordFrame(ID, 1, "broker");
  const mitAt = wordFrame(ID, 2, "mit");
  const runAt = wordFrame(ID, 2, "run");
  const forkAt = wordFrame(ID, 2, "fork");
  const shipAt = wordFrame(ID, 2, "ship");
  const pricingAt = wordFrame(ID, 2, "per-message");

  const drum = r(drumAt, 26);
  const queue = r(queueAt);
  const mit = r(mitAt, 24);

  const W = 420;
  const H = 300;
  const RY = 42;

  return (
    <AbsoluteFill>
      <div style={{ position: "absolute", top: 110, width: "100%", textAlign: "center" }}>
        <WordsIn text="So what does it need to run?" at={c0} span={12} emphasis={["run"]} style={{ fontFamily: F.display, fontSize: 92, fontWeight: 700, letterSpacing: "-0.03em", lineHeight: 1.1, color: C.ink }} />
      </div>

      {/* PostgreSQL drum */}
      <div style={{ position: "absolute", left: 220, top: 330, width: W, height: H + RY * 2, opacity: drum, transform: `scale(${0.85 + drum * 0.15}) translateY(${(1 - drum) * 20}px)` }}>
        <svg width={W} height={H + RY * 2} style={{ position: "absolute", inset: 0 }}>
          <rect x={0} y={RY} width={W} height={H} fill="rgba(0,173,216,0.10)" />
          <ellipse cx={W / 2} cy={RY + H} rx={W / 2} ry={RY} fill="rgba(0,173,216,0.08)" stroke={C.teal} strokeWidth={3} />
          <rect x={1} y={RY} width={W - 2} height={H} fill="rgba(0,173,216,0.10)" />
          <line x1={0} y1={RY} x2={0} y2={RY + H} stroke={C.teal} strokeWidth={3} />
          <line x1={W} y1={RY} x2={W} y2={RY + H} stroke={C.teal} strokeWidth={3} />
          <ellipse cx={W / 2} cy={RY} rx={W / 2} ry={RY} fill="rgba(0,173,216,0.18)" stroke={C.teal} strokeWidth={3.5} />
        </svg>
        <div style={{ position: "absolute", top: RY + 50, width: "100%", textAlign: "center", fontFamily: F.mono, fontSize: 40, fontWeight: 700, color: C.tealDeep }}>PostgreSQL</div>
        <div style={{ position: "absolute", top: RY + 130, left: 40, right: 40, height: 76, borderRadius: 14, background: `rgba(0,173,216,${0.08 + queue * 0.12})`, border: `2px solid rgba(0,173,216,${queue * 0.6})`, display: "grid", placeItems: "center", fontFamily: F.mono, fontSize: 26, fontWeight: 700, color: C.tealDeep, opacity: queue, transform: `translateY(${(1 - queue) * 10}px)` }}>
          River queue · in here
        </div>
      </div>

      {/* MIT stamp */}
      <div style={{ position: "absolute", left: 790, top: 380, display: "flex", flexDirection: "column", alignItems: "center", gap: 22, opacity: mit, transform: `scale(${0.85 + mit * 0.15}) rotate(${(1 - mit) * -8}deg)` }}>
        <div style={{ width: 230, height: 230, borderRadius: "50%", border: `4px solid ${C.teal}`, background: "rgba(0,173,216,0.08)", boxShadow: `0 0 ${30 * mit}px rgba(0,173,216,0.3)`, display: "flex", flexDirection: "column", alignItems: "center", justifyContent: "center", fontFamily: F.display }}>
          <div style={{ fontSize: 66, fontWeight: 700, color: C.tealDeep, lineHeight: 1 }}>MIT</div>
          <div style={{ fontSize: 22, fontWeight: 700, color: C.tealDeep, letterSpacing: "0.12em", marginTop: 6 }}>LICENSE</div>
        </div>
        <div style={{ display: "flex", gap: 14, fontFamily: F.mono, fontSize: 34, fontWeight: 700, color: C.ink }}>
          {[
            ["run", runAt],
            ["fork", forkAt],
            ["ship", shipAt],
          ].map(([t, at], i) => {
            const s = r(at as number);
            return (
              <React.Fragment key={t as string}>
                {i > 0 && <span style={{ color: "rgba(11,15,20,0.3)", opacity: s }}>·</span>}
                <span style={{ opacity: s, transform: `translateY(${(1 - s) * 10}px)`, display: "inline-block" }}>{t as string}</span>
              </React.Fragment>
            );
          })}
        </div>
      </div>

      {/* Ghosts */}
      <div style={{ position: "absolute", right: 200, top: 360, display: "flex", flexDirection: "column", gap: 22, alignItems: "flex-end" }}>
        <Ghost label="Redis" frame={frame} appearAt={drumAt + 8} strikeAt={redisAt} width={240} />
        <Ghost label="message broker" frame={frame} appearAt={drumAt + 12} strikeAt={brokerAt} width={380} />
        <Ghost label="worker fleet" frame={frame} appearAt={drumAt + 16} strikeAt={brokerAt + 8} width={300} />
        <Ghost label="per-message pricing" frame={frame} appearAt={pricingAt} strikeAt={pricingAt + 10} width={440} />
      </div>

      <Sequence from={mitAt}><Audio src={staticFile("sfx/impact-bass-1.mp3")} volume={0.22} /></Sequence>
      <Sequence from={redisAt}><Audio src={staticFile("sfx/whoosh-short.mp3")} volume={0.2} /></Sequence>
    </AbsoluteFill>
  );
};

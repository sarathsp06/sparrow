import React from "react";
import { AbsoluteFill, Img, Sequence, interpolate, useCurrentFrame, useVideoConfig, staticFile, Easing } from "remotion";
import { Audio } from "@remotion/media";
import { C, F } from "../../theme";
import { cueFrame, wordFrame, useReveal, ease } from "../timing";
import { Chip } from "../ui";

// Frame 13 — Run it. The GitHub URL is the hero of the last frame.

const ID = "13-cta";
const c0 = cueFrame(ID, 0);
const c1 = cueFrame(ID, 1);
const CMD = "$ docker compose up -d";

export const Cta: React.FC = () => {
  const frame = useCurrentFrame();
  const { durationInFrames } = useVideoConfig();
  const r = useReveal();

  const t1 = r(c0);
  const t2 = r(wordFrame(ID, 0, "one", 1));
  const eq = r(wordFrame(ID, 0, "postgres") + 10);
  const typeStart = c1;
  const typeEnd = c1 + 26;
  const typedN = Math.round(interpolate(frame, [typeStart, typeEnd], [0, CMD.length], { extrapolateLeft: "clamp", extrapolateRight: "clamp" }));
  const done = frame >= typeEnd;
  const urlIn = r(wordFrame(ID, 1, "delivering"));
  const fade = interpolate(frame, [durationInFrames - 24, durationInFrames], [1, 0], { extrapolateLeft: "clamp", extrapolateRight: "clamp", easing: Easing.out(Easing.cubic) });

  const Token: React.FC<{ s: number; label: string }> = ({ s, label }) => (
    <div style={{ padding: "24px 40px", borderRadius: 18, background: "rgba(255,255,255,0.78)", border: "1.5px solid rgba(11,15,20,0.14)", boxShadow: "0 14px 38px rgba(11,15,20,0.08)", fontFamily: F.display, fontSize: 40, fontWeight: 700, color: C.ink, opacity: s, transform: `translateY(${(1 - s) * 20}px)`, display: "flex", alignItems: "center", gap: 16 }}>
      <span style={{ fontFamily: F.mono, fontSize: 32, fontWeight: 700, color: C.teal }}>1 ×</span>
      {label}
    </div>
  );

  return (
    <AbsoluteFill style={{ opacity: fade }}>
      <div style={{ position: "absolute", top: 200, width: "100%", display: "flex", justifyContent: "center", alignItems: "center", gap: 34 }}>
        <Token s={t1} label="Go binary" />
        <span style={{ fontSize: 48, color: "rgba(11,15,20,0.35)", opacity: t2 }}>+</span>
        <Token s={t2} label="PostgreSQL" />
        <div style={{ display: "flex", alignItems: "center", gap: 16, opacity: eq, transform: `translateX(${(1 - eq) * 20}px)` }}>
          <span style={{ fontSize: 48, color: "rgba(11,15,20,0.35)" }}>=</span>
          <Img src={staticFile("sparrow-logo.svg")} style={{ width: 60, height: 48 }} />
          <span style={{ fontFamily: F.display, fontSize: 46, fontWeight: 700, color: C.ink, letterSpacing: "-0.02em" }}>Sparrow</span>
        </div>
      </div>

      <div style={{ position: "absolute", top: 400, width: "100%", display: "flex", justifyContent: "center" }}>
        <div style={{ fontFamily: F.mono, fontSize: 44, background: C.navy, color: "#E8E6E1", padding: "28px 52px", borderRadius: 20, boxShadow: `0 20px 48px rgba(11,15,20,0.22)${done ? ", 0 0 40px rgba(0,173,216,0.3)" : ""}`, border: `1.5px solid ${done ? "rgba(0,173,216,0.5)" : "rgba(255,255,255,0.06)"}`, opacity: ease(frame, c1 - 4, c1 + 4) }}>
          <span style={{ color: C.teal }}>{CMD.slice(0, 2)}</span>
          {CMD.slice(2, typedN)}
          <span style={{ opacity: Math.floor(frame / 12) % 2 === 0 && !done ? 1 : 0 }}>█</span>
          {done && <span style={{ marginLeft: 18, color: C.teal }}>✓</span>}
        </div>
      </div>

      <div style={{ position: "absolute", top: 580, width: "100%", textAlign: "center", opacity: urlIn, transform: `translateY(${(1 - urlIn) * 14}px)` }}>
        <div style={{ fontFamily: F.display, fontSize: 64, fontWeight: 700, letterSpacing: "-0.02em", color: C.ink }}>
          github.com/<span style={{ color: C.teal }}>sarathsp06/sparrow</span>
        </div>
        <div style={{ marginTop: 22, display: "flex", justifyContent: "center", gap: 14 }}>
          <Chip tone="teal" size={24}>MIT licensed</Chip>
          <Chip tone="ink" size={24}>Go + PostgreSQL</Chip>
          <Chip tone="ink" size={24}>OpenAPI 3.1</Chip>
        </div>
      </div>

      <Sequence from={typeStart}><Audio src={staticFile("sfx/typing.mp3")} volume={0.18} /></Sequence>
      <Sequence from={typeEnd + 4}><Audio src={staticFile("sfx/chime.mp3")} volume={0.25} /></Sequence>
    </AbsoluteFill>
  );
};

import React from "react";
import { AbsoluteFill, Img, Sequence, interpolate, staticFile, useCurrentFrame } from "remotion";
import { Audio } from "@remotion/media";
import { C, F } from "../../theme";
import { cueFrame, wordFrame, useReveal, ease } from "../timing";
import { card, Parcel } from "../ui";

// Frame 3 — Three nights
// Match-cut from Frame 2's lockup. The parcel `item.shipped` becomes the hero,
// then three chapter cards promise the arc: happy path, the dark night, the bug.

const ID = "03-three";
const c0 = cueFrame(ID, 0);
const c1 = cueFrame(ID, 1);

const CHAPTERS = [
  { n: "01", title: "The happy path", sub: "push → fan-out → signed delivery", tone: C.teal, at: wordFrame(ID, 1, "happy") },
  { n: "02", title: "The night the partner went dark", sub: "retries · health · one-shot recovery", tone: "#3B3F73", at: wordFrame(ID, 1, "night") },
  { n: "03", title: "The Tuesday you shipped a bug", sub: "re-push history · pause · portal", tone: C.coral, at: wordFrame(ID, 1, "tuesday") },
];

export const Three: React.FC = () => {
  const frame = useCurrentFrame();
  const r = useReveal();

  // Lockup lifts and shrinks at c0 (match-cut from Frame 2).
  const lift = ease(frame, c0, c0 + 22);
  const lockupY = interpolate(lift, [0, 1], [380, 60]);
  const lockupScale = interpolate(lift, [0, 1], [1, 0.36]);

  // Parcel hero pops when "item.shipped" is spoken, then floats up for the cards.
  const parcelAt = wordFrame(ID, 0, "item");
  const parcelIn = r(parcelAt, 26);
  const parcelUp = ease(frame, c1 - 6, c1 + 18);
  const parcelY = interpolate(parcelUp, [0, 1], [330, 190]);
  const parcelScale = interpolate(parcelUp, [0, 1], [1, 0.72]);

  return (
    <AbsoluteFill style={{ overflow: "hidden" }}>
      {/* Lockup (continuity from Frame 2) */}
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
        <Img src={staticFile("sparrow-logo.svg")} style={{ width: 100, height: 80 }} />
        <div style={{ marginTop: 12, fontFamily: F.display, fontWeight: 700, fontSize: 120, lineHeight: 1.05, letterSpacing: "-0.04em", color: C.ink }}>
          Sparrow
        </div>
      </div>

      {/* Parcel hero: icon + item.shipped pill + payload */}
      <div
        style={{
          position: "absolute",
          top: parcelY,
          left: 0,
          right: 0,
          display: "flex",
          flexDirection: "column",
          alignItems: "center",
          gap: 18,
          opacity: parcelIn,
          transform: `scale(${parcelScale * (0.9 + parcelIn * 0.1)})`,
          transformOrigin: "center top",
          filter: `blur(${(1 - parcelIn) * 8}px)`,
        }}
      >
        <Parcel size={150} glow={18} />
        <div
          style={{
            padding: "16px 40px",
            borderRadius: 999,
            background: C.coral,
            color: "#fff",
            fontFamily: F.mono,
            fontSize: 44,
            fontWeight: 700,
            letterSpacing: "0.01em",
            boxShadow: "0 12px 36px rgba(249,115,22,0.38)",
          }}
        >
          item.shipped
        </div>
        <div
          style={{
            padding: "12px 24px",
            borderRadius: 14,
            background: "rgba(11,15,20,0.06)",
            border: "1px solid rgba(11,15,20,0.1)",
            fontFamily: F.mono,
            fontSize: 24,
            color: "rgba(11,15,20,0.65)",
            whiteSpace: "pre",
            opacity: interpolate(parcelUp, [0, 1], [1, 0]),
          }}
        >
          {`{ "parcel": "pkg_88a1", "carrier": "dhl", "weight_kg": 2.4 }`}
        </div>
      </div>

      {/* Three chapter cards */}
      <div
        style={{
          position: "absolute",
          top: 470,
          left: 0,
          right: 0,
          display: "flex",
          justifyContent: "center",
          gap: 34,
        }}
      >
        {CHAPTERS.map((ch, i) => {
          const s = r(ch.at, 26);
          return (
            <div
              key={ch.n}
              style={{
                ...card,
                width: 540,
                height: 300,
                padding: "30px 34px",
                display: "flex",
                flexDirection: "column",
                justifyContent: "space-between",
                borderTop: `8px solid ${ch.tone}`,
                opacity: s,
                transform: `translateY(${(1 - s) * 40}px) rotate(${(1 - s) * (i - 1) * 3}deg)`,
              }}
            >
              <div style={{ fontFamily: F.mono, fontSize: 26, fontWeight: 700, color: ch.tone, letterSpacing: "0.12em" }}>
                NIGHT {ch.n}
              </div>
              <div style={{ fontFamily: F.display, fontSize: 44, fontWeight: 700, lineHeight: 1.08, letterSpacing: "-0.02em", color: C.ink }}>
                {ch.title}
              </div>
              <div style={{ fontFamily: F.mono, fontSize: 20, color: "rgba(11,15,20,0.55)" }}>{ch.sub}</div>
            </div>
          );
        })}
      </div>

      <Sequence from={parcelAt + 4}>
        <Audio src={staticFile("sfx/chime.mp3")} volume={0.18} />
      </Sequence>
    </AbsoluteFill>
  );
};

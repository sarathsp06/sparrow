import React from "react";
import { AbsoluteFill, interpolate, useCurrentFrame } from "remotion";
import { C, F } from "../theme";
import { FPS, FRAMES, sceneStart } from "./timing";

// Caption pill in the reserved bottom band. One cue at a time; words light
// up in proportion to their length across the cue (the cue *is* the clip,
// so timing is exact at cue level and close at word level).
const CUES = FRAMES.flatMap((f, i) =>
  f.cues.map((c) => {
    const from = sceneStart(i) + Math.round(c.start * FPS);
    return { text: c.text, from, to: from + Math.round(c.dur * FPS) };
  }),
);

export const Captions: React.FC<{ dark?: (frame: number) => boolean }> = ({ dark }) => {
  const frame = useCurrentFrame();
  const cue = CUES.find((c) => frame >= c.from - 2 && frame < c.to + 8);
  if (!cue) return null;

  const onDark = dark?.(frame) ?? false;
  const inOp = interpolate(frame, [cue.from - 2, cue.from + 4, cue.to + 2, cue.to + 8], [0, 1, 1, 0], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
  });

  const words = cue.text.split(" ");
  const weights = words.map((w) => w.length + 2);
  const total = weights.reduce((a, b) => a + b, 0);
  const progress = (frame - cue.from) / Math.max(1, cue.to - cue.from);
  let acc = 0;

  return (
    <AbsoluteFill style={{ pointerEvents: "none" }}>
      <div
        style={{
          position: "absolute",
          left: 0,
          right: 0,
          bottom: 64,
          display: "flex",
          justifyContent: "center",
          opacity: inOp,
          transform: `translateY(${(1 - inOp) * 8}px)`,
        }}
      >
        <div
          style={{
            maxWidth: 1400,
            padding: "14px 30px",
            borderRadius: 16,
            background: onDark ? "rgba(244,242,237,0.10)" : "rgba(11,15,20,0.82)",
            backdropFilter: "blur(8px)",
            fontFamily: F.display,
            fontSize: 38,
            fontWeight: 600,
            letterSpacing: "-0.01em",
            lineHeight: 1.25,
            textAlign: "center",
          }}
        >
          {words.map((w, i) => {
            const startW = acc / total;
            acc += weights[i];
            const spoken = progress >= startW;
            return (
              <span
                key={i}
                style={{
                  color: spoken ? C.cream : "rgba(244,242,237,0.42)",
                  textShadow: spoken ? "0 0 1px rgba(0,0,0,0.2)" : "none",
                }}
              >
                {w}
                {i < words.length - 1 ? " " : ""}
              </span>
            );
          })}
        </div>
      </div>
    </AbsoluteFill>
  );
};

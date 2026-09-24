import React from "react";
import { AbsoluteFill, Easing, Img, interpolate, staticFile, useCurrentFrame } from "remotion";

// The sparrow darts across the frame over a scene cut, riding the whoosh SFX.
// Place inside a <Sequence from={whooshFrame}>.
export const FlyBy: React.FC<{ y?: number; reverse?: boolean }> = ({ y = 420, reverse = false }) => {
  const frame = useCurrentFrame();
  const t = interpolate(frame, [0, 22], [0, 1], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
    easing: Easing.bezier(0.5, 0, 0.3, 1),
  });
  if (frame > 24) return null;
  const x = reverse ? 2100 - t * 2400 : -200 + t * 2400;
  const yy = y - Math.sin(t * Math.PI) * 120;
  const trail = [0.02, 0.045, 0.075, 0.11];

  return (
    <AbsoluteFill style={{ pointerEvents: "none" }}>
      {trail.map((d, i) => {
        const tt = Math.max(0, t - d);
        const tx = reverse ? 2100 - tt * 2400 : -200 + tt * 2400;
        const ty = y - Math.sin(tt * Math.PI) * 120;
        return (
          <Img
            key={d}
            src={staticFile("sparrow-logo.svg")}
            style={{
              position: "absolute",
              left: tx,
              top: ty,
              width: 120,
              height: 96,
              opacity: 0.22 - i * 0.05,
              filter: "blur(3px)",
              scale: reverse ? "-1 1" : "1 1",
            }}
          />
        );
      })}
      <Img
        src={staticFile("sparrow-logo.svg")}
        style={{
          position: "absolute",
          left: x,
          top: yy,
          width: 120,
          height: 96,
          rotate: `${Math.cos(t * Math.PI) * -12}deg`,
          scale: reverse ? "-1 1" : "1 1",
          filter: "drop-shadow(0 10px 18px rgba(11,15,20,0.25))",
        }}
      />
    </AbsoluteFill>
  );
};

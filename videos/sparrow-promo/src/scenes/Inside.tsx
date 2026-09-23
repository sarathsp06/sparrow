import React from "react";
import { AbsoluteFill, Img, interpolate, spring, staticFile, useCurrentFrame, useVideoConfig } from "remotion";
import { C, F } from "../theme";

// 2.5s beat: the live interactive architecture doc (arch-page.png is a capture of
// sarathsp06.github.io/sparrow/diagrams/sparrow-layered-architecture.html).
export const Inside: React.FC = () => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();

  const cardIn = spring({ frame, fps, config: { damping: 200 } });
  const capIn = spring({ frame: frame - 22, fps, config: { damping: 200 } });
  // Slow punch-in so the still feels alive.
  const zoom = interpolate(frame, [0, 75], [1.0, 1.05]);

  return (
    <AbsoluteFill style={{ background: C.cream }}>
      <div
        style={{
          position: "absolute",
          top: 64,
          left: 72,
          fontFamily: F.mono,
          fontSize: 22,
          letterSpacing: "0.16em",
          color: C.ink,
          opacity: interpolate(frame, [0, 10], [0, 1], { extrapolateRight: "clamp" }),
        }}
      >
        <span style={{ color: C.coralText }}>&#10033;</span> UNDER THE HOOD
      </div>

      <div
        style={{
          position: "absolute",
          top: 120,
          left: (1920 - 1400) / 2,
          width: 1400,
          height: 875,
          borderRadius: 18,
          overflow: "hidden",
          border: "1px solid rgba(11,15,20,0.14)",
          boxShadow: "0 24px 60px rgba(11,15,20,0.16)",
          opacity: cardIn,
          transform: `translateY(${(1 - cardIn) * 24}px)`,
        }}
      >
        <Img
          src={staticFile("arch-page.png")}
          style={{ width: "100%", transform: `scale(${zoom})`, transformOrigin: "50% 30%" }}
        />
      </div>

      <div
        style={{
          position: "absolute",
          bottom: 36,
          width: "100%",
          textAlign: "center",
          fontFamily: F.mono,
          fontSize: 24,
          color: C.ink,
          opacity: 0.75 * capIn,
        }}
      >
        explore it live &#8594; sarathsp06.github.io/sparrow
      </div>
    </AbsoluteFill>
  );
};

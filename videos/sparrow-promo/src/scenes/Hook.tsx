import React from "react";
import { AbsoluteFill, interpolate, spring, useCurrentFrame, useVideoConfig } from "remotion";
import { C, F } from "../theme";

const LINES = ["Your app emits events.", "The rest of your stack expects webhooks."];

export const Hook: React.FC = () => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();

  // Coral underline under "Health checks." draws after its line lands.
  const underline = interpolate(frame, [75, 95], [0, 1], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
  });

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
          // Legible on frame 1 for silent autoplay; quick settle only.
          opacity: interpolate(frame, [0, 6], [0.6, 1], { extrapolateRight: "clamp" }),
        }}
      >
        <span style={{ color: C.coralText }}>&#10033;</span> WEBHOOK DELIVERY
      </div>

      <AbsoluteFill style={{ justifyContent: "center", alignItems: "center" }}>
        <div style={{ textAlign: "center" }}>
          {LINES.map((line, i) => {
            const delay = 10 + i * 24;
            const s = spring({ frame: frame - delay, fps, config: { damping: 200 } });
            return (
              <div
                key={line}
                style={{
                  fontFamily: F.display,
                  fontWeight: 500,
                  fontSize: 88,
                  lineHeight: 1.22,
                  letterSpacing: "-0.02em",
                  color: C.ink,
                  opacity: s,
                  transform: `translateY(${(1 - s) * 30}px)`,
                }}
              >
                {line}
              </div>
            );
          })}
          <div
            style={{
              marginTop: 44,
              fontFamily: F.mono,
              fontSize: 30,
              color: C.coralText,
              opacity: interpolate(frame, [70, 84], [0, 1], {
                extrapolateLeft: "clamp",
                extrapolateRight: "clamp",
              }),
            }}
          >
            Retries. Signing. Health checks. Every time.
            <div
              style={{
                height: 6,
                marginTop: 10,
                background: C.coral,
                borderRadius: 3,
                transformOrigin: "left center",
                transform: `scaleX(${underline})`,
              }}
            />
          </div>
        </div>
      </AbsoluteFill>
    </AbsoluteFill>
  );
};

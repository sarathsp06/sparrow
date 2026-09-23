import React from "react";
import { AbsoluteFill, interpolate, spring, useCurrentFrame, useVideoConfig } from "remotion";
import { C, F } from "../theme";

// All five claims verbatim from README features / okf bundle.
const CARDS = [
  "At-least-once delivery",
  "HMAC + Ed25519 signing",
  "Encryption at rest (AES-256-GCM)",
  "Per-webhook health tracking",
];
const WIDE = "Embedded dashboard \u00B7 consumer portal \u00B7 OpenAPI-first";

const cardStyle: React.CSSProperties = {
  display: "flex",
  justifyContent: "center",
  alignItems: "center",
  background: "rgba(255,255,255,0.55)",
  border: "1px solid rgba(11,15,20,0.1)",
  borderRadius: 14,
  fontFamily: F.display,
  fontSize: 34,
  color: C.ink,
  boxShadow: "0 10px 30px rgba(11,15,20,0.06)",
};

export const Features: React.FC = () => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();

  const pop = (delay: number) => {
    const s = spring({ frame: frame - delay, fps, config: { damping: 200 } });
    return {
      opacity: s,
      transform: `translateY(${(1 - s) * 26}px)`,
    };
  };

  const tagIn = spring({ frame: frame - 110, fps, config: { damping: 14, stiffness: 160 } });

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
        <span style={{ color: C.coralText }}>&#10033;</span> INCLUDED, NOT EXTRA
      </div>

      <AbsoluteFill style={{ justifyContent: "center", alignItems: "center" }}>
        <div style={{ width: 1560 }}>
          <div
            style={{
              display: "grid",
              gridTemplateColumns: "1fr 1fr",
              gap: 28,
            }}
          >
            {CARDS.map((text, i) => (
              <div key={text} style={{ ...cardStyle, height: 150, ...pop(14 + i * 16) }}>
                {text}
              </div>
            ))}
          </div>
          <div style={{ display: "flex", justifyContent: "center", marginTop: 28 }}>
            <div
              style={{
                ...cardStyle,
                position: "relative",
                height: 130,
                padding: "0 70px",
                whiteSpace: "nowrap",
                ...pop(90),
              }}
            >
              {WIDE}
              <div
                style={{
                  position: "absolute",
                  top: -20,
                  right: 26,
                  background: C.coral,
                  color: C.cream,
                  fontFamily: F.display,
                  fontSize: 24,
                  fontWeight: 500,
                  padding: "10px 20px",
                  borderRadius: 999,
                  opacity: tagIn,
                  transform: `scale(${0.6 + tagIn * 0.4}) rotate(${(1 - tagIn) * -6}deg)`,
                }}
              >
                all built in
              </div>
            </div>
          </div>
        </div>
      </AbsoluteFill>
    </AbsoluteFill>
  );
};

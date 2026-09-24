import React from "react";
import { AbsoluteFill, useCurrentFrame } from "remotion";
import { C } from "../theme";

// Persistent paper backdrop shared by every cream scene: drifting dot grid,
// two slow-orbiting color glows, film grain and a soft vignette.
export const Backdrop: React.FC = () => {
  const frame = useCurrentFrame();
  const t = frame / 30;

  return (
    <AbsoluteFill style={{ background: C.cream, overflow: "hidden" }}>
      <AbsoluteFill
        style={{
          backgroundImage: "radial-gradient(rgba(11,15,20,0.11) 1.4px, transparent 1.6px)",
          backgroundSize: "36px 36px",
          backgroundPosition: `${t * 6}px ${t * -9}px`,
          maskImage: "radial-gradient(ellipse 75% 70% at 50% 50%, black 30%, transparent 100%)",
        }}
      />
      <div
        style={{
          position: "absolute",
          width: 900,
          height: 900,
          borderRadius: "50%",
          left: 1250 + Math.sin(t * 0.35) * 120,
          top: -380 + Math.cos(t * 0.3) * 80,
          background: "radial-gradient(circle, rgba(0,173,216,0.16), transparent 65%)",
        }}
      />
      <div
        style={{
          position: "absolute",
          width: 1000,
          height: 1000,
          borderRadius: "50%",
          left: -420 + Math.cos(t * 0.28) * 110,
          top: 560 + Math.sin(t * 0.33) * 90,
          background: "radial-gradient(circle, rgba(249,115,22,0.11), transparent 65%)",
        }}
      />
      <Grain />
      <AbsoluteFill
        style={{
          background: "radial-gradient(ellipse 85% 80% at 50% 50%, transparent 60%, rgba(11,15,20,0.08))",
        }}
      />
    </AbsoluteFill>
  );
};

// Static-per-2-frames SVG noise, so the paper texture shimmers like film.
export const Grain: React.FC<{ opacity?: number; blend?: React.CSSProperties["mixBlendMode"] }> = ({
  opacity = 0.07,
  blend = "multiply",
}) => {
  const frame = useCurrentFrame();
  const seed = Math.floor(frame / 2) % 50;
  return (
    <svg
      width={1920}
      height={1080}
      style={{ position: "absolute", inset: 0, opacity, mixBlendMode: blend }}
    >
      <filter id={`grain-${seed}`}>
        <feTurbulence type="fractalNoise" baseFrequency="0.85" numOctaves={2} seed={seed} />
        <feColorMatrix type="saturate" values="0" />
      </filter>
      <rect width="100%" height="100%" filter={`url(#grain-${seed})`} />
    </svg>
  );
};

// Wrap a scene with the backdrop when rendered as a standalone composition.
export const withBackdrop = (Scene: React.FC): React.FC => {
  const Wrapped: React.FC = () => (
    <AbsoluteFill>
      <Backdrop />
      <Scene />
    </AbsoluteFill>
  );
  return Wrapped;
};

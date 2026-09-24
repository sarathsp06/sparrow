import React from "react";
import {
  AbsoluteFill,
  Easing,
  Img,
  interpolate,
  spring,
  staticFile,
  useCurrentFrame,
  useVideoConfig,
} from "remotion";
import { C, F } from "../theme";
import { Kicker } from "../components/Kicker";

// 2.5s beat: the live interactive architecture doc (arch-page.png is a capture of
// sarathsp06.github.io/sparrow/diagrams/sparrow-layered-architecture.html),
// framed as a browser window with a slow Ken Burns pan across the diagram.
const URL = "sarathsp06.github.io/sparrow/diagrams/sparrow-layered-architecture.html";

export const Inside: React.FC = () => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();

  const cardIn = spring({ frame, fps, config: { damping: 20, stiffness: 120 } });
  const capIn = spring({ frame: frame - 22, fps, config: { damping: 200 } });

  return (
    <AbsoluteFill>
      <Kicker text="UNDER THE HOOD" />

      <div
        style={{
          position: "absolute",
          top: 118,
          left: (1920 - 1400) / 2,
          width: 1400,
          height: 860,
          borderRadius: 18,
          overflow: "hidden",
          background: "white",
          border: "1px solid rgba(11,15,20,0.14)",
          boxShadow: "0 40px 90px rgba(11,15,20,0.22)",
          opacity: cardIn,
          transform: `perspective(2000px) rotateX(${(1 - cardIn) * 18}deg) translateY(${(1 - cardIn) * 60}px)`,
        }}
      >
        <div
          style={{
            height: 52,
            display: "flex",
            alignItems: "center",
            gap: 10,
            padding: "0 20px",
            background: "#EDEBE6",
            borderBottom: "1px solid rgba(11,15,20,0.1)",
          }}
        >
          {["#FF5F57", "#FEBC2E", "#28C840"].map((c) => (
            <span key={c} style={{ width: 14, height: 14, borderRadius: "50%", background: c }} />
          ))}
          <div
            style={{
              marginLeft: 24,
              flex: 1,
              height: 32,
              borderRadius: 8,
              background: "white",
              display: "flex",
              alignItems: "center",
              padding: "0 16px",
              fontFamily: F.mono,
              fontSize: 16,
              color: "rgba(11,15,20,0.7)",
            }}
          >
            {URL}
          </div>
        </div>
        <div style={{ overflow: "hidden", height: 808 }}>
          <Img
            src={staticFile("arch-page.png")}
            style={{
              width: "100%",
              transformOrigin: "50% 30%",
              scale: String(
                interpolate(frame, [0, 75], [1.0, 1.14], { easing: Easing.inOut(Easing.quad) }),
              ),
              translate: `${interpolate(frame, [0, 75], [30, -40])}px ${interpolate(frame, [0, 75], [0, -20])}px`,
            }}
          />
        </div>
      </div>

      <div
        style={{
          position: "absolute",
          bottom: 30,
          width: "100%",
          textAlign: "center",
          fontFamily: F.mono,
          fontSize: 24,
          color: C.ink,
          opacity: 0.8 * capIn,
        }}
      >
        explore it live &#8594; sarathsp06.github.io/sparrow
      </div>
    </AbsoluteFill>
  );
};

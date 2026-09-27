import React from "react";
import {
  AbsoluteFill,
  Img,
  Sequence,
  interpolate,
  useCurrentFrame,
  useVideoConfig,
  staticFile,
  Easing,
} from "remotion";
import { Audio } from "@remotion/media";
import { C, F } from "../../theme";
import { cueFrame, useReveal, ease } from "../timing";

const ID = "12-cta";
const CMD = "$ docker compose up -d";

export const Cta: React.FC = () => {
  const frame = useCurrentFrame();
  const { durationInFrames } = useVideoConfig();
  const r = useReveal();

  const c0 = cueFrame(ID, 0);
  const c1 = cueFrame(ID, 1);

  // c0: two tokens slam in: "1 x Go binary" and "1 x PostgreSQL"
  const token1In = r(c0);
  const token2In = r(c0 + 8);
  const equalsIn = r(c0 + 16);

  // c1: terminal types docker compose command
  const typeStart = c1;
  const typeEnd = c1 + 28;
  const typedN = Math.round(
    interpolate(frame, [typeStart, typeEnd], [0, CMD.length], {
      extrapolateLeft: "clamp",
      extrapolateRight: "clamp",
    }),
  );
  const cmdDone = frame >= typeEnd;
  const caretOn = Math.floor(frame / 12) % 2 === 0;

  // Footer / lockup
  const footerIn = r(c1 + 14);

  // Exit fade: gentle fade to cream/ink in the last ~20 frames
  const fadeStart = durationInFrames - 24;
  const exitFade = interpolate(
    frame,
    [fadeStart, durationInFrames],
    [1, 0],
    { extrapolateLeft: "clamp", extrapolateRight: "clamp", easing: Easing.out(Easing.cubic) },
  );

  const TOKEN_Y = 300;
  const TOKEN_GAP = 40;

  return (
    <AbsoluteFill style={{ opacity: exitFade }}>
      {/* Token pair: "1 x Go binary" + "1 x PostgreSQL" */}
      <div
        style={{
          position: "absolute",
          top: TOKEN_Y,
          width: "100%",
          display: "flex",
          justifyContent: "center",
          alignItems: "center",
          gap: TOKEN_GAP,
        }}
      >
        {/* Go binary token */}
        <div
          style={{
            padding: "22px 38px",
            borderRadius: 16,
            background: "rgba(255,255,255,0.72)",
            border: "1.5px solid rgba(11,15,20,0.14)",
            boxShadow: "0 14px 38px rgba(11,15,20,0.08)",
            fontFamily: F.display,
            fontSize: 32,
            fontWeight: 600,
            color: C.ink,
            opacity: token1In,
            transform: `translateY(${(1 - token1In) * 20}px) scale(${0.92 + token1In * 0.08})`,
            display: "flex",
            alignItems: "center",
            gap: 14,
          }}
        >
          <span
            style={{
              fontFamily: F.mono,
              fontSize: 26,
              fontWeight: 700,
              color: C.teal,
            }}
          >
            1 {"×"}
          </span>
          Go binary
        </div>

        {/* Plus sign */}
        <div
          style={{
            fontFamily: F.display,
            fontSize: 42,
            fontWeight: 300,
            color: "rgba(11,15,20,0.35)",
            opacity: token2In,
          }}
        >
          +
        </div>

        {/* PostgreSQL token */}
        <div
          style={{
            padding: "22px 38px",
            borderRadius: 16,
            background: "rgba(255,255,255,0.72)",
            border: "1.5px solid rgba(11,15,20,0.14)",
            boxShadow: "0 14px 38px rgba(11,15,20,0.08)",
            fontFamily: F.display,
            fontSize: 32,
            fontWeight: 600,
            color: C.ink,
            opacity: token2In,
            transform: `translateY(${(1 - token2In) * 20}px) scale(${0.92 + token2In * 0.08})`,
            display: "flex",
            alignItems: "center",
            gap: 14,
          }}
        >
          <span
            style={{
              fontFamily: F.mono,
              fontSize: 26,
              fontWeight: 700,
              color: C.teal,
            }}
          >
            1 {"×"}
          </span>
          PostgreSQL
        </div>

        {/* = Sparrow lockup */}
        <div
          style={{
            display: "flex",
            alignItems: "center",
            gap: 16,
            opacity: equalsIn,
            transform: `translateX(${(1 - equalsIn) * 20}px)`,
          }}
        >
          <span
            style={{
              fontFamily: F.display,
              fontSize: 42,
              fontWeight: 300,
              color: "rgba(11,15,20,0.35)",
            }}
          >
            =
          </span>
          <Img
            src={staticFile("sparrow-logo.svg")}
            style={{ width: 44, height: 36 }}
          />
          <span
            style={{
              fontFamily: F.display,
              fontSize: 36,
              fontWeight: 700,
              color: C.ink,
              letterSpacing: "-0.02em",
            }}
          >
            Sparrow
          </span>
        </div>
      </div>

      {/* Terminal: docker compose up -d */}
      <div
        style={{
          position: "absolute",
          top: 470,
          width: "100%",
          display: "flex",
          justifyContent: "center",
        }}
      >
        <div
          style={{
            fontFamily: F.mono,
            fontSize: 34,
            background: C.navy,
            color: "#E8E6E1",
            padding: "24px 42px",
            borderRadius: 16,
            boxShadow: `0 20px 48px rgba(11,15,20,0.22)${
              cmdDone ? ", 0 0 40px rgba(0,173,216,0.3)" : ""
            }`,
            border: `1.5px solid ${
              cmdDone ? "rgba(0,173,216,0.5)" : "rgba(255,255,255,0.06)"
            }`,
            opacity: ease(frame, c1 - 4, c1 + 4),
            transform: `translateY(${(1 - ease(frame, c1 - 4, c1 + 4)) * 14}px)`,
          }}
        >
          <span style={{ color: C.teal }}>
            {CMD.slice(0, 2)}
          </span>
          {CMD.slice(2, typedN)}
          <span
            style={{
              opacity: caretOn && !cmdDone ? 1 : 0,
            }}
          >
            {"█"}
          </span>
          {cmdDone && (
            <span style={{ marginLeft: 18, color: C.teal }}>
              {"✓"}
            </span>
          )}
        </div>
      </div>

      {/* Footer: MIT licensed * github URL */}
      <div
        style={{
          position: "absolute",
          top: 590,
          width: "100%",
          textAlign: "center",
          fontFamily: F.mono,
          fontSize: 24,
          color: C.ink,
          opacity: 0.65 * footerIn,
          transform: `translateY(${(1 - footerIn) * 10}px)`,
        }}
      >
        MIT licensed {"·"} github.com/sarathsp06/sparrow
      </div>

      {/* SFX: typing on command */}
      <Sequence from={typeStart}>
        <Audio src={staticFile("sfx/typing.mp3")} volume={0.18} />
      </Sequence>

      {/* SFX: chime at the end */}
      <Sequence from={typeEnd + 4}>
        <Audio src={staticFile("sfx/chime.mp3")} volume={0.25} />
      </Sequence>
    </AbsoluteFill>
  );
};

import React from "react";
import {
  AbsoluteFill,
  Sequence,
  interpolate,
  useCurrentFrame,
  staticFile,
} from "remotion";
import { Audio } from "@remotion/media";
import { C, F } from "../../theme";
import { cueFrame, useReveal, ease } from "../timing";
import { Wire, Comet, Ripple } from "../../components/Wire";
import { WordsIn } from "../ui";

const ID = "11-takeaway";

// Geometry: mirrors Frame 1 but on cream ground.
// Service chip at left, endpoint card at right, dotted wire between.
const SVC = { x: 340, y: 380 };
const SVC_W = 200;
const SVC_H = 62;
const EP = { x: 1360, y: 380 };
const EP_W = 260;
const EP_H = 62;

const WIRE_A = { x: SVC.x + SVC_W, y: SVC.y + SVC_H / 2 };
const WIRE_B = { x: EP.x, y: EP.y + EP_H / 2 };

export const Takeaway: React.FC = () => {
  const frame = useCurrentFrame();
  const r = useReveal();

  const c0 = cueFrame(ID, 0);
  const c1 = cueFrame(ID, 1);
  const c2 = cueFrame(ID, 2);

  // c0: composition fades in — service chip, wire, endpoint, small question
  const sceneIn = r(c0);

  // c1: packet travels and lands with teal ripple + checkmark
  const packetStart = c1;
  const packetEnd = c1 + 26;

  // c2: headline word-reveals
  const headlineAt = c2;

  // Endpoint state: starts neutral, goes green when packet lands
  const epGreen = ease(frame, packetEnd - 4, packetEnd + 4);

  // "delivered * attempt 4" label
  const deliveredIn = r(packetEnd + 2);

  return (
    <AbsoluteFill>
      {/* Small question text above the diagram */}
      <div
        style={{
          position: "absolute",
          top: 280,
          width: "100%",
          textAlign: "center",
          fontFamily: F.mono,
          fontSize: 24,
          letterSpacing: "0.04em",
          color: "rgba(11,15,20,0.50)",
          opacity: sceneIn,
          transform: `translateY(${(1 - sceneIn) * 10}px)`,
        }}
      >
        So, where did order.created end up?
      </div>

      {/* Service chip: orders-api */}
      <div
        style={{
          position: "absolute",
          left: SVC.x,
          top: SVC.y,
          width: SVC_W,
          height: SVC_H,
          borderRadius: 14,
          background: "rgba(255,255,255,0.72)",
          border: "1px solid rgba(11,15,20,0.14)",
          boxShadow: "0 10px 26px rgba(11,15,20,0.06)",
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          fontFamily: F.mono,
          fontSize: 22,
          fontWeight: 500,
          color: C.ink,
          opacity: sceneIn,
          transform: `translateY(${(1 - sceneIn) * 14}px)`,
        }}
      >
        orders-api
      </div>

      {/* Endpoint card: partner API */}
      <div
        style={{
          position: "absolute",
          left: EP.x,
          top: EP.y,
          width: EP_W,
          height: EP_H,
          borderRadius: 14,
          background: interpolate(epGreen, [0, 1], [0, 1], {
            extrapolateLeft: "clamp",
            extrapolateRight: "clamp",
          })
            ? `rgba(${Math.round(255 - epGreen * 255)},${Math.round(255 - epGreen * 40)},${Math.round(255 - epGreen * 39)},0.72)`
            : "rgba(255,255,255,0.72)",
          border: `1.5px solid ${
            epGreen > 0.5
              ? `rgba(0,173,216,${0.3 + epGreen * 0.4})`
              : "rgba(11,15,20,0.14)"
          }`,
          boxShadow: epGreen > 0.5
            ? `0 10px 26px rgba(0,173,216,0.12)`
            : "0 10px 26px rgba(11,15,20,0.06)",
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          fontFamily: F.mono,
          fontSize: 22,
          fontWeight: 500,
          color: C.ink,
          opacity: sceneIn,
          transform: `translateY(${(1 - sceneIn) * 14}px)`,
        }}
      >
        partner API
      </div>

      {/* Wire + comet + ripple */}
      <svg
        width={1920}
        height={1080}
        style={{ position: "absolute", inset: 0, opacity: sceneIn }}
      >
        <Wire
          id="takeaway-wire"
          a={WIRE_A}
          b={WIRE_B}
          frame={frame}
          drawStart={c0}
          drawEnd={c0 + 22}
          color="rgba(11,15,20,0.22)"
          width={3}
        />
        <Comet
          a={WIRE_A}
          b={WIRE_B}
          frame={frame}
          start={packetStart}
          end={packetEnd}
          color={C.coral}
          r={10}
        />
        <Ripple
          x={WIRE_B.x}
          y={WIRE_B.y}
          frame={frame}
          at={packetEnd}
          color={C.teal}
          size={55}
        />
      </svg>

      {/* "delivered * attempt 4" label next to endpoint */}
      <div
        style={{
          position: "absolute",
          left: EP.x + EP_W / 2 - 100,
          top: EP.y + EP_H + 18,
          fontFamily: F.mono,
          fontSize: 19,
          fontWeight: 600,
          color: C.tealDeep,
          opacity: deliveredIn,
          transform: `translateY(${(1 - deliveredIn) * 8}px)`,
          display: "flex",
          alignItems: "center",
          gap: 10,
        }}
      >
        <span style={{ color: C.teal, fontSize: 22 }}>{"✓"}</span>
        delivered {"·"} attempt 4
      </div>

      {/* c2: Hero headline */}
      <div
        style={{
          position: "absolute",
          top: 560,
          width: "100%",
          textAlign: "center",
        }}
      >
        <WordsIn
          text="Push once. Sparrow does the rest."
          at={headlineAt}
          span={18}
          emphasis={["Sparrow"]}
          emphasisColor={C.teal}
          style={{
            fontFamily: F.display,
            fontSize: 96,
            fontWeight: 700,
            letterSpacing: "-0.03em",
            lineHeight: 1.1,
            color: C.ink,
          }}
        />
      </div>

      {/* SFX: ping on packet landing */}
      <Sequence from={packetEnd}>
        <Audio src={staticFile("sfx/ping.mp3")} volume={0.25} />
      </Sequence>
    </AbsoluteFill>
  );
};

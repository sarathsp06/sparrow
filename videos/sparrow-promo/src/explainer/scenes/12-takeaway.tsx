import React from "react";
import { AbsoluteFill, Sequence, staticFile, useCurrentFrame } from "remotion";
import { Audio } from "@remotion/media";
import { C, F } from "../../theme";
import { cueFrame, wordFrame, useReveal } from "../timing";
import { Chip, Dock, Parcel, WordsIn } from "../ui";
import { Wire, Comet, Ripple } from "../../components/Wire";

// Frame 12 — Callback: where did item.shipped end up? Every dock, every time.

const ID = "12-takeaway";
const c0 = cueFrame(ID, 0);
const c2 = cueFrame(ID, 2);

const SRC = { x: 160, y: 300 };
const DOCKS = [
  { name: "billing-svc", y: 180, at: wordFrame(ID, 1, "every", 0) },
  { name: "partner-dhl", y: 300, at: wordFrame(ID, 1, "every", 1) },
  { name: "crm-sync", y: 420, at: wordFrame(ID, 1, "even") },
];

export const Takeaway: React.FC = () => {
  const frame = useCurrentFrame();
  const r = useReveal();
  const inS = r(c0);

  return (
    <AbsoluteFill>
      <div style={{ position: "absolute", top: 90, width: "100%", textAlign: "center", fontFamily: F.mono, fontSize: 28, letterSpacing: "0.04em", color: "rgba(11,15,20,0.55)", opacity: inS }}>
        So, where did item.shipped end up?
      </div>

      {/* source */}
      <div style={{ position: "absolute", left: SRC.x, top: SRC.y - 40, display: "flex", flexDirection: "column", alignItems: "center", gap: 10, opacity: inS, transform: `translateY(${(1 - inS) * 14}px)` }}>
        <Parcel size={80} glow={10} />
        <Chip tone="ink" size={22}>warehouse-app</Chip>
      </div>

      <svg width={1920} height={1080} style={{ position: "absolute", inset: 0, pointerEvents: "none", overflow: "visible", opacity: inS }}>
        {DOCKS.map((d, i) => {
          const a = { x: SRC.x + 100, y: SRC.y };
          const b = { x: 1180, y: d.y + 48 };
          return (
            <React.Fragment key={d.name}>
              <Wire id={`t-${i}`} a={a} b={b} frame={frame} drawStart={c0 + i * 4} drawEnd={c0 + 22 + i * 4} color="rgba(11,15,20,0.22)" width={3} />
              <Comet a={a} b={b} frame={frame} start={d.at - 18} end={d.at} color={C.coral} r={10} />
              <Ripple x={b.x} y={b.y} frame={frame} at={d.at} color={C.teal} size={56} />
            </React.Fragment>
          );
        })}
      </svg>

      {DOCKS.map((d) => (
        <div key={d.name} style={{ position: "absolute", left: 1180, top: d.y, opacity: inS }}>
          <Dock name={d.name} width={560} state={frame >= d.at ? "open" : "idle"} badge={frame >= d.at ? <Chip tone="teal" size={20}>✓ delivered</Chip> : undefined} />
        </div>
      ))}

      <div style={{ position: "absolute", top: 640, width: "100%", textAlign: "center" }}>
        <WordsIn text="Push once. Sparrow does the rest." at={c2} span={18} emphasis={["Sparrow"]} emphasisColor={C.teal} style={{ fontFamily: F.display, fontSize: 104, fontWeight: 700, letterSpacing: "-0.03em", lineHeight: 1.1, color: C.ink }} />
      </div>

      {DOCKS.map((d) => (
        <Sequence key={d.name} from={d.at}><Audio src={staticFile("sfx/ping.mp3")} volume={0.2} /></Sequence>
      ))}
    </AbsoluteFill>
  );
};

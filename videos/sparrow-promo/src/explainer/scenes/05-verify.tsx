import React from "react";
import { AbsoluteFill, Sequence, staticFile, useCurrentFrame } from "remotion";
import { Audio } from "@remotion/media";
import { C, F } from "../../theme";
import { cueFrame, wordFrame, useReveal, ease } from "../timing";
import { Chip, Dock, NightKicker, Parcel } from "../ui";
import { Wire, Comet, Ripple } from "../../components/Wire";

// Frame 5 — Your customer's side. The parcel arrives at their dock; one copy-in
// file checks the three Standard Webhooks headers; bad parcels are turned away.

const ID = "05-verify";
const c0 = cueFrame(ID, 0);

const LANGS = [
  { l: "Python", at: wordFrame(ID, 1, "python") },
  { l: "TypeScript", at: wordFrame(ID, 1, "typescript") },
  { l: "Java", at: wordFrame(ID, 1, "java") },
  { l: "Ruby", at: wordFrame(ID, 1, "ruby") },
];
const MORE = ["Go", "Kotlin", "PHP", "Rust", "Elixir"];

const REJECTS = [
  { l: "wrong key", at: wordFrame(ID, 2, "wrong") },
  { l: "replayed", at: wordFrame(ID, 2, "replayed") },
  { l: "> 5 min old", at: wordFrame(ID, 2, "older") },
];

export const Verify: React.FC = () => {
  const frame = useCurrentFrame();
  const r = useReveal();

  const dockIn = r(c0);
  const arriveAt = wordFrame(ID, 0, "checks");
  const headersAt = wordFrame(ID, 0, "signature");
  const fileAt = wordFrame(ID, 1, "one");
  const moreAt = wordFrame(ID, 1, "five");
  const rejectedAt = wordFrame(ID, 2, "rejected");
  const rejected = r(rejectedAt);
  const verifiedAt = rejectedAt + 22;
  const verified = r(verifiedAt);

  const a = { x: 120, y: 330 };
  const b = { x: 1240, y: 330 };

  return (
    <AbsoluteFill>
      <NightKicker label="NIGHT ONE · THE OTHER SIDE" at={c0} />

      {/* The parcel travelling in */}
      <svg width={1920} height={1080} style={{ position: "absolute", inset: 0, pointerEvents: "none", overflow: "visible" }}>
        <Wire id="v-in" a={a} b={b} frame={frame} drawStart={c0} drawEnd={c0 + 20} color="rgba(11,15,20,0.22)" width={3} />
        <Comet a={a} b={b} frame={frame} start={c0 + 8} end={arriveAt} color={C.coral} r={10} />
        <Ripple x={b.x} y={b.y} frame={frame} at={arriveAt} color={C.teal} size={56} />
      </svg>
      {frame >= c0 + 8 && frame <= arriveAt && (
        <Parcel size={54} style={{ position: "absolute", left: a.x + (b.x - a.x) * ease(frame, c0 + 8, arriveAt) - 27, top: 330 - 60 }} />
      )}

      {/* Customer dock */}
      <div style={{ position: "absolute", left: 1240, top: 270, opacity: dockIn, transform: `translateX(${(1 - dockIn) * 30}px)` }}>
        <Dock
          name="your customer"
          sub="receiver · verifies every parcel"
          width={560}
          state={verified > 0.5 ? "open" : rejected > 0.5 ? "closed" : "idle"}
          badge={verified > 0.5 ? <Chip tone="teal" size={20}>✓ verified</Chip> : rejected > 0.5 ? <Chip tone="red" size={20}>✕ rejected</Chip> : undefined}
        />
      </div>

      {/* The three signed headers, large, under the wire */}
      <div style={{ position: "absolute", left: 120, top: 400, width: 1040, fontFamily: F.mono, fontSize: 28, lineHeight: 1.7 }}>
        {[
          ["webhook-id", "msg_9f2a3c…"],
          ["webhook-timestamp", "1790799840"],
          ["webhook-signature", "v1,K5oQ7r…  v1a,Zq1M…"],
        ].map(([k, v], i) => {
          const s = r(headersAt + i * 6);
          return (
            <div key={k} style={{ opacity: s, transform: `translateX(${(1 - s) * -16}px)` }}>
              <span style={{ color: C.coralText, fontWeight: 700 }}>{k}</span>
              <span style={{ color: "rgba(11,15,20,0.45)" }}>: </span>
              <span style={{ color: C.ink }}>{v}</span>
            </div>
          );
        })}
      </div>

      {/* One copy-in file */}
      {frame >= fileAt - 4 && (
        <div
          style={{
            position: "absolute",
            left: 120,
            top: 600,
            width: 760,
            opacity: r(fileAt),
            transform: `translateY(${(1 - r(fileAt)) * 16}px)`,
            background: C.navy,
            color: C.cream,
            borderRadius: 18,
            padding: "18px 26px",
            fontFamily: F.mono,
            fontSize: 22,
            lineHeight: 1.6,
            boxShadow: "0 22px 56px rgba(11,15,20,0.25)",
          }}
        >
          <div style={{ color: C.teal, fontSize: 18, marginBottom: 8 }}>sparrow_verify.py · one file, no package</div>
          <div><span style={{ color: "#c792ea" }}>from</span> sparrow_verify <span style={{ color: "#c792ea" }}>import</span> verify</div>
          <div>verify(headers, raw_body, secret)  <span style={{ color: "rgba(232,230,225,0.45)" }}># raises if it isn't ours</span></div>
        </div>
      )}

      {/* Language chips */}
      <div style={{ position: "absolute", left: 920, top: 600, width: 900, display: "flex", flexWrap: "wrap", gap: 12, alignContent: "flex-start" }}>
        {LANGS.map((x) => {
          const s = r(x.at);
          return (
            <div key={x.l} style={{ opacity: s, transform: `translateY(${(1 - s) * 12}px)` }}>
              <Chip tone="ink" size={24}>{x.l}</Chip>
            </div>
          );
        })}
        {MORE.map((x, i) => {
          const s = r(moreAt + i * 3);
          return (
            <div key={x} style={{ opacity: s, transform: `translateY(${(1 - s) * 12}px)` }}>
              <Chip tone="ghost" size={22}>{x}</Chip>
            </div>
          );
        })}
      </div>

      {/* Rejections slam onto the dock door */}
      <div style={{ position: "absolute", left: 1240, top: 410, display: "flex", gap: 12, flexWrap: "wrap", width: 560 }}>
        {REJECTS.map((x) => {
          const s = r(x.at);
          return (
            <div key={x.l} style={{ opacity: s, transform: `translateY(${(1 - s) * -14}px) rotate(${(1 - s) * -6}deg)` }}>
              <Chip tone="red" size={22}>✕ {x.l}</Chip>
            </div>
          );
        })}
      </div>

      <Sequence from={arriveAt}><Audio src={staticFile("sfx/ping.mp3")} volume={0.18} /></Sequence>
      <Sequence from={rejectedAt}><Audio src={staticFile("sfx/impact-bass-1.mp3")} volume={0.14} /></Sequence>
    </AbsoluteFill>
  );
};

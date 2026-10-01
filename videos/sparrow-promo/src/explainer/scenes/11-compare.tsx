import React from "react";
import { AbsoluteFill, Sequence, interpolate, useCurrentFrame, staticFile, Easing } from "remotion";
import { Audio } from "@remotion/media";
import { C, F } from "../../theme";
import { Kicker } from "../../components/Kicker";
import { cueFrame, wordFrame, useReveal } from "../timing";

// Frame 11 — How it compares (docs landing "How Sparrow stacks up", verbatim).

const ID = "11-compare";
const c0 = cueFrame(ID, 0);
const c1 = cueFrame(ID, 1);

const COLS = ["Sparrow", "Svix", "Convoy", "Hookdeck", "DIY"];
type Cell = { text: string; style: "sparrow" | "muted" | "normal" };
const ROWS: { label: string; cells: Cell[]; at: number }[] = [
  { label: "Fully open source (MIT)", at: c1, cells: [{ text: "Yes", style: "sparrow" }, { text: "Partial", style: "normal" }, { text: "No (Elastic 2.0)", style: "muted" }, { text: "Partial", style: "normal" }, { text: "Yes", style: "normal" }] },
  { label: "Core infrastructure", at: c1 + 8, cells: [{ text: "PostgreSQL only", style: "sparrow" }, { text: "PostgreSQL + Redis", style: "normal" }, { text: "PostgreSQL + Redis", style: "normal" }, { text: "SaaS", style: "normal" }, { text: "Varies", style: "muted" }] },
  { label: "Webhook signing", at: wordFrame(ID, 2, "signing"), cells: [{ text: "HMAC-SHA256 + Ed25519", style: "sparrow" }, { text: "HMAC-SHA256", style: "normal" }, { text: "HMAC-SHA256", style: "normal" }, { text: "HMAC-SHA256", style: "normal" }, { text: "Manual", style: "muted" }] },
  { label: "Prebuilt integrations", at: wordFrame(ID, 2, "recipes"), cells: [{ text: "Recipes", style: "sparrow" }, { text: "Limited", style: "muted" }, { text: "Limited", style: "muted" }, { text: "Yes", style: "normal" }, { text: "Manual", style: "muted" }] },
  { label: "Consumer portal", at: wordFrame(ID, 2, "portal"), cells: [{ text: "Yes, self-hosted", style: "sparrow" }, { text: "Yes", style: "normal" }, { text: "Yes", style: "normal" }, { text: "Yes", style: "normal" }, { text: "No", style: "muted" }] },
  { label: "Self-monitoring alerts", at: wordFrame(ID, 2, "self-monitoring"), cells: [{ text: "System events + email", style: "sparrow" }, { text: "Operational webhooks", style: "normal" }, { text: "Alert configs", style: "normal" }, { text: "Issue alerts", style: "normal" }, { text: "Manual", style: "muted" }] },
];

const LABEL_W = 400;
const COL_W = 290;
const ROW_H = 84;
const TABLE_W = LABEL_W + COLS.length * COL_W;
const X = (1920 - TABLE_W) / 2;
const Y = 250;
const HEADER_Y = Y - 60;

export const Compare: React.FC = () => {
  const frame = useCurrentFrame();
  const r = useReveal();

  const redisAt = wordFrame(ID, 1, "redis");
  const saasAt = wordFrame(ID, 1, "saas");
  const mitAt = wordFrame(ID, 2, "mit");
  const redis = interpolate(frame, [redisAt, redisAt + 12], [0, 1], { extrapolateLeft: "clamp", extrapolateRight: "clamp", easing: Easing.out(Easing.cubic) });
  const saas = interpolate(frame, [saasAt, saasAt + 12], [0, 1], { extrapolateLeft: "clamp", extrapolateRight: "clamp", easing: Easing.out(Easing.cubic) });
  const mit = r(mitAt, 30);
  const band = r(c0, 22);
  const bandGrow = interpolate(frame, [c1, c1 + 60], [0.15, 1], { extrapolateLeft: "clamp", extrapolateRight: "clamp", easing: Easing.inOut(Easing.cubic) });
  const foot = r(ROWS[5].at + 16);

  return (
    <AbsoluteFill style={{ fontFamily: F.display, color: C.ink }}>
      <Kicker text="HOW IT COMPARES" delay={c0} />

      <div style={{ position: "absolute", left: X + LABEL_W, top: HEADER_Y - 16, width: COL_W, height: (76 + ROWS.length * ROW_H + 20) * bandGrow, background: "linear-gradient(180deg, rgba(0,173,216,0.14), rgba(0,173,216,0.05))", border: "1.5px solid rgba(0,173,216,0.45)", borderRadius: 16, boxShadow: "0 0 36px rgba(0,173,216,0.15)", opacity: band }} />

      {COLS.map((col, ci) => {
        const s = r(c0 + 6 + ci * 4, 22);
        return (
          <div key={col} style={{ position: "absolute", left: X + LABEL_W + ci * COL_W, top: HEADER_Y, width: COL_W, textAlign: "center", fontFamily: F.mono, fontSize: 26, fontWeight: ci === 0 ? 700 : 500, color: ci === 0 ? C.tealDeep : C.ink, opacity: s, transform: `translateY(${(1 - s) * 14}px)` }}>
            {col}
          </div>
        );
      })}

      {ROWS.map((row, ri) => {
        const rowS = r(row.at, 22);
        return (
          <div key={row.label} style={{ position: "absolute", left: X, top: Y + ri * ROW_H, width: TABLE_W, height: ROW_H, display: "flex", alignItems: "center", borderBottom: ri < ROWS.length - 1 ? "1px solid rgba(11,15,20,0.08)" : "none", opacity: rowS, transform: `translateY(${(1 - rowS) * 16}px)` }}>
            <div style={{ width: LABEL_W, fontSize: 26, fontWeight: 600, paddingRight: 20, lineHeight: 1.2 }}>{row.label}</div>
            {row.cells.map((cell, ci) => {
              const coral = ri === 1 && ((ci === 1 || ci === 2) ? redis > 0.5 : ci === 3 ? saas > 0.5 : false);
              const color = coral ? C.coral : cell.style === "sparrow" ? (ri === 0 && mit > 0.5 ? C.teal : C.tealDeep) : cell.style === "muted" ? "rgba(11,15,20,0.4)" : C.ink;
              const cs = r(row.at + 4 + ci * 3, 20);
              return (
                <div key={ci} style={{ width: COL_W, textAlign: "center", fontFamily: ci === 0 ? F.mono : F.display, fontSize: ci === 0 ? 22 : 23, fontWeight: cell.style === "sparrow" || coral ? 700 : 500, color, opacity: cs, lineHeight: 1.2, padding: "0 8px", textShadow: coral ? "0 0 12px rgba(249,115,22,0.35)" : ci === 0 ? `0 0 ${14 * cs}px rgba(0,173,216,0.3)` : undefined }}>
                  {cell.text}
                </div>
              );
            })}
          </div>
        );
      })}

      <div style={{ position: "absolute", top: Y + ROWS.length * ROW_H + 34, width: "100%", textAlign: "center", fontFamily: F.mono, fontSize: 21, color: "rgba(11,15,20,0.55)", opacity: foot }}>
        Directional snapshot — vendors change packaging often; verify against their docs.
      </div>

      <Sequence from={c0 + 6}><Audio src={staticFile("sfx/ping.mp3")} volume={0.12} /></Sequence>
    </AbsoluteFill>
  );
};

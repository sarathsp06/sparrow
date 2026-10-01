import React from "react";
import { AbsoluteFill, Img, Sequence, staticFile, useCurrentFrame } from "remotion";
import { Audio } from "@remotion/media";
import { C, F } from "../../theme";
import { cueFrame, wordFrame, useReveal, ease } from "../timing";
import { NightKicker } from "../ui";

// Frame 9 — Seeing it: the embedded dashboard (a reconstruction of the Svelte
// UI) and an OpenTelemetry trace of one parcel, push to dock.

const ID = "09-seeing";
const c0 = cueFrame(ID, 0);
const c1 = cueFrame(ID, 1);

const NAV = [
  { label: "Webhooks", at: wordFrame(ID, 0, "webhooks") },
  { label: "Events", at: wordFrame(ID, 0, "events") },
  { label: "Deliveries", at: wordFrame(ID, 0, "deliveries") },
  { label: "Health", at: wordFrame(ID, 0, "health") },
];

const ROWS = [
  { name: "partner-dhl", url: "https://hooks.partner-dhl.example/in", health: "healthy" },
  { name: "billing-svc", url: "https://billing.internal/wh", health: "healthy" },
  { name: "crm-sync", url: "https://api.crm.example/hooks", health: "degraded" },
] as const;

const SPARK = [4, 7, 5, 8, 6, 9, 7, 10, 8, 11, 9, 8, 10, 12, 11, 9, 10, 13, 11, 10];

const hc = (h: string) => (h === "healthy" ? "#16875a" : h === "degraded" ? "#b3790a" : "#d5362f");

export const Seeing: React.FC = () => {
  const frame = useCurrentFrame();
  const r = useReveal();
  const browserIn = r(c0);
  const otelIn = r(c1);

  const W = 1380;
  const H = 540;
  const X = (1920 - W) / 2;
  const Y = 150;
  const SIDEBAR = 250;

  const spans = [
    { label: "POST /v1/…/events", start: 0, width: 0.96 },
    { label: "event.fanout", start: 0.07, width: 0.4 },
    { label: "webhook.deliver", start: 0.28, width: 0.56 },
    { label: "http.post partner-dhl", start: 0.4, width: 0.4 },
  ];

  return (
    <AbsoluteFill>
      <NightKicker label="SEEING IT · SAME BINARY" at={c0} />

      <div style={{ position: "absolute", left: X, top: Y, width: W, height: H, borderRadius: 20, overflow: "hidden", background: "#fff", border: "1px solid rgba(11,15,20,0.14)", boxShadow: "0 40px 90px rgba(11,15,20,0.22)", opacity: browserIn, transform: `translateY(${(1 - browserIn) * 40}px)` }}>
        <div style={{ height: 52, display: "flex", alignItems: "center", gap: 10, padding: "0 20px", background: "#EDEBE6", borderBottom: "1px solid rgba(11,15,20,0.10)" }}>
          {["#FF5F57", "#FEBC2E", "#28C840"].map((cc) => <span key={cc} style={{ width: 13, height: 13, borderRadius: "50%", background: cc }} />)}
          <div style={{ marginLeft: 20, flex: 1, height: 32, borderRadius: 8, background: "white", display: "flex", alignItems: "center", padding: "0 14px", fontFamily: F.mono, fontSize: 17, color: "rgba(11,15,20,0.6)" }}>
            localhost:8080 · SPARROW_SERVE_UI=true
          </div>
        </div>
        <div style={{ display: "flex", height: H - 52 }}>
          <div style={{ width: SIDEBAR, background: "#faf9f6", borderRight: "1px solid #e6e2d9", display: "flex", flexDirection: "column" }}>
            <div style={{ display: "flex", alignItems: "center", gap: 12, padding: "16px 18px", borderBottom: "1px solid #e6e2d9" }}>
              <Img src={staticFile("sparrow-logo.svg")} style={{ width: 30, height: 24 }} />
              <span style={{ fontFamily: F.display, fontWeight: 700, fontSize: 17, letterSpacing: "0.16em", color: "#1c1a16" }}>SPARROW</span>
            </div>
            <nav style={{ display: "flex", flexDirection: "column", gap: 4, padding: 12 }}>
              {NAV.map((item, i) => {
                const hp = r(item.at, 18);
                const active = i === 0;
                return (
                  <div key={item.label} style={{ padding: "11px 14px", borderRadius: 10, background: hp > 0.01 ? `rgba(0,173,216,${hp * 0.14})` : active ? "rgba(0,0,0,0.05)" : "transparent", border: `2px solid rgba(0,173,216,${hp * 0.6})`, fontFamily: F.display, fontSize: 20, fontWeight: hp > 0.5 || active ? 700 : 500, color: hp > 0.5 ? C.tealDeep : "#1c1a16" }}>
                    {item.label}
                  </div>
                );
              })}
            </nav>
            <div style={{ padding: "8px 12px" }}>
              <div style={{ padding: "10px 0", borderRadius: 10, background: "#ea9d2b", color: "#fff", textAlign: "center", fontFamily: F.display, fontSize: 18, fontWeight: 700 }}>+ Push Event</div>
            </div>
          </div>
          <div style={{ flex: 1, background: "#faf9f6", padding: "26px 30px" }}>
            <div style={{ display: "flex", justifyContent: "space-between", alignItems: "baseline", marginBottom: 16 }}>
              <span style={{ fontFamily: F.display, fontSize: 28, fontWeight: 700, color: "#1c1a16" }}>Webhooks</span>
              <span style={{ fontFamily: F.mono, fontSize: 15, color: "#6b6558" }}>consumer: acme</span>
            </div>
            <div style={{ background: "#fff", borderRadius: 12, border: "1px solid #e6e2d9", overflow: "hidden" }}>
              <div style={{ display: "flex", height: 44, alignItems: "center", padding: "0 20px", borderBottom: "1px solid #e6e2d9", fontFamily: F.mono, fontSize: 14, letterSpacing: "0.06em", color: "#6b6558", fontWeight: 600 }}>
                <span style={{ width: 220 }}>NAME</span><span style={{ width: 400 }}>URL</span><span style={{ width: 140 }}>HEALTH</span><span style={{ flex: 1, textAlign: "right" }}>DELIVERIES · 24H</span>
              </div>
              {ROWS.map((row) => (
                <div key={row.name} style={{ display: "flex", alignItems: "center", height: 66, padding: "0 20px", borderBottom: "1px solid rgba(230,226,217,0.6)", fontFamily: F.display, fontSize: 18, color: "#1c1a16" }}>
                  <span style={{ width: 220, fontWeight: 600 }}>{row.name}</span>
                  <span style={{ width: 400, fontFamily: F.mono, fontSize: 15, color: "#6b6558" }}>{row.url}</span>
                  <span style={{ width: 140 }}>
                    <span style={{ padding: "4px 12px", borderRadius: 999, fontFamily: F.mono, fontSize: 14, color: hc(row.health), background: `${hc(row.health)}1a` }}>● {row.health}</span>
                  </span>
                  <span style={{ flex: 1, display: "flex", justifyContent: "flex-end" }}>
                    <svg width={160} height={34}>
                      <polyline points={SPARK.map((v, i) => `${(i / (SPARK.length - 1)) * 160},${34 - (v / 13) * 30}`).join(" ")} fill="none" stroke={C.teal} strokeWidth={2.5} strokeLinejoin="round" />
                    </svg>
                  </span>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>

      {/* OpenTelemetry trace strip */}
      <div style={{ position: "absolute", left: 120, top: 730, width: 1680, opacity: otelIn, transform: `translateY(${(1 - otelIn) * 16}px)` }}>
        <div style={{ fontFamily: F.mono, fontSize: 18, fontWeight: 700, letterSpacing: "0.12em", color: C.tealDeep, marginBottom: 8 }}>OPENTELEMETRY · one trace, push to dock · traces + metrics + logs</div>
        {spans.map((sp, i) => {
          const w = ease(frame, c1 + i * 8, c1 + i * 8 + 20);
          return (
            <div key={sp.label} style={{ position: "relative", height: 30, opacity: r(c1 + i * 8, 18) }}>
              <span style={{ position: "absolute", left: sp.start * 1680, top: -2, fontFamily: F.mono, fontSize: 16, color: C.ink, opacity: 0.75 }}>{sp.label}</span>
              <div style={{ position: "absolute", left: sp.start * 1680, top: 18, width: sp.width * 1680 * w, height: 10, borderRadius: 4, background: i === 3 ? C.coral : `rgba(0,173,216,${0.55 + i * 0.12})` }} />
            </div>
          );
        })}
      </div>

      <Sequence from={c0}><Audio src={staticFile("sfx/whoosh-short.mp3")} volume={0.16} /></Sequence>
      <Sequence from={c1}><Audio src={staticFile("sfx/chime.mp3")} volume={0.14} /></Sequence>
    </AbsoluteFill>
  );
};

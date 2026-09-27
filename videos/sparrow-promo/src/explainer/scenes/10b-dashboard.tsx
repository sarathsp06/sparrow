import React from "react";
import {
  AbsoluteFill,
  Img,
  Sequence,
  useCurrentFrame,
  staticFile,
} from "remotion";
import { Audio } from "@remotion/media";
import { C, F } from "../../theme";
import { cueFrame, frameTiming, useReveal, ease } from "../timing";
import { card } from "../ui";

const ID = "10b-dashboard";

/* ─── Mock data ─────────────────────────────────────────────────────── */

const NAV_ITEMS = [
  { label: "Webhooks", icon: "M9 7a4 4 0 1 1 4 4l-2 3.5M15 17a4 4 0 1 1-4-4M7.5 13.5 5 17a4 4 0 1 0 4 2" },
  { label: "Events", icon: "M13 2 3 14h7v8l10-12h-7z" },
  { label: "Deliveries", icon: "M22 2 11 13M22 2l-7 20-4-9-9-4 20-7z" },
  { label: "Health", icon: "M3 12h4l2 6 4-14 3 10 2-2h3" },
];

// Derive word offsets proportionally from the cue text so they stay correct
// when timing.json is regenerated at a different speech rate.
const HIGHLIGHT_LABELS = ["webhooks,", "events,", "deliveries,", "health."];
const c1Cue = frameTiming(ID).cues[1];
const c1Words = c1Cue.text.split(" ");
const HIGHLIGHT_WORDS = HIGHLIGHT_LABELS.map((w) => {
  const idx = c1Words.findIndex((x) => x.toLowerCase() === w.toLowerCase());
  return {
    label: w.replace(/[.,]/, "").replace(/^./, (ch) => ch.toUpperCase()),
    offsetSec: (idx / Math.max(1, c1Words.length - 1)) * c1Cue.dur,
  };
});

const WEBHOOK_ROWS = [
  { name: "partner-acme", url: "https://api.acme.example/hooks", health: "healthy" as const },
  { name: "billing-svc", url: "https://billing.internal/wh", health: "healthy" as const },
  { name: "slack-ops", url: "https://hooks.slack.com/…", health: "degraded" as const },
];

/* Mini sparkline data — deterministic, illustrative only */
const SPARKLINE_POINTS = [4, 7, 5, 8, 6, 9, 7, 10, 8, 11, 9, 8, 10, 12, 11, 9, 10, 13, 11, 10];

/* ─── Helpers ───────────────────────────────────────────────────────── */

const healthColor = (h: "healthy" | "degraded" | "unhealthy") =>
  h === "healthy" ? "#16875a" : h === "degraded" ? "#b3790a" : "#d5362f";

const healthBg = (h: "healthy" | "degraded" | "unhealthy") =>
  h === "healthy" ? "rgba(22,135,90,0.10)" : h === "degraded" ? "rgba(179,121,10,0.10)" : "rgba(213,54,47,0.10)";

const Sparkline: React.FC<{ width: number; height: number; opacity: number }> = ({ width, height, opacity }) => {
  const max = Math.max(...SPARKLINE_POINTS);
  const min = Math.min(...SPARKLINE_POINTS);
  const pts = SPARKLINE_POINTS.map((v, i) => {
    const x = (i / (SPARKLINE_POINTS.length - 1)) * width;
    const y = height - ((v - min) / (max - min)) * (height - 4) - 2;
    return `${x},${y}`;
  });
  return (
    <svg width={width} height={height} style={{ opacity }}>
      <polyline
        points={pts.join(" ")}
        fill="none"
        stroke={C.teal}
        strokeWidth={2}
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  );
};

/* ─── Browser chrome ────────────────────────────────────────────────── */

const BrowserFrame: React.FC<{
  url: string;
  width: number;
  height: number;
  style?: React.CSSProperties;
  children: React.ReactNode;
}> = ({ url, width, height, style, children }) => (
  <div
    style={{
      width,
      height,
      borderRadius: 18,
      overflow: "hidden",
      background: "#ffffff",
      border: "1px solid rgba(11,15,20,0.14)",
      boxShadow: "0 40px 90px rgba(11,15,20,0.22)",
      ...style,
    }}
  >
    {/* Title bar */}
    <div
      style={{
        height: 48,
        display: "flex",
        alignItems: "center",
        gap: 10,
        padding: "0 18px",
        background: "#EDEBE6",
        borderBottom: "1px solid rgba(11,15,20,0.10)",
      }}
    >
      {["#FF5F57", "#FEBC2E", "#28C840"].map((c) => (
        <span key={c} style={{ width: 12, height: 12, borderRadius: "50%", background: c }} />
      ))}
      <div
        style={{
          marginLeft: 20,
          flex: 1,
          height: 30,
          borderRadius: 8,
          background: "white",
          display: "flex",
          alignItems: "center",
          padding: "0 14px",
          fontFamily: F.mono,
          fontSize: 14,
          color: "rgba(11,15,20,0.6)",
        }}
      >
        {url}
      </div>
    </div>
    {/* Content */}
    <div style={{ height: height - 48, overflow: "hidden", display: "flex" }}>
      {children}
    </div>
  </div>
);

/* ─── Main scene ────────────────────────────────────────────────────── */

export const Dashboard: React.FC = () => {
  const frame = useCurrentFrame();
  const r = useReveal();

  const c0 = cueFrame(ID, 0);
  const c1 = cueFrame(ID, 1);
  const c2 = cueFrame(ID, 2);
  const c3 = cueFrame(ID, 3);

  /* c0: log lines that blur/dim */
  const logIn = r(c0);
  const logDim = ease(frame, c0 + 20, c1, 1, 0.15);
  const logBlur = ease(frame, c0 + 20, c1, 0, 8);

  /* c1: browser frame rises in */
  const browserIn = r(c1);

  /* c1 highlights: each nav item / table header lights up in turn */
  const highlightProgress = (idx: number) => {
    const startFrame = cueFrame(ID, 1, HIGHLIGHT_WORDS[idx].offsetSec);
    return r(startFrame, 18);
  };

  /* "same binary" chip */
  const chipIn = r(c1 + 30);

  /* c2: portal card */
  const portalIn = r(c2);

  /* c3: OTel trace waterfall */
  const otelIn = r(c3);

  /* ── Layout constants ── */
  const BROWSER_W = 1140;
  const BROWSER_H = 580;
  const BROWSER_X = (1920 - BROWSER_W) / 2;
  const BROWSER_Y = 120;

  const SIDEBAR_W = 200;

  const PORTAL_W = 440;
  const PORTAL_X = 1920 - PORTAL_W - 120;
  const PORTAL_Y = 440;

  /* OTel trace strip */
  const TRACE_X = 160;
  const TRACE_Y = BROWSER_Y + BROWSER_H + 48;
  const TRACE_W = 1000;

  return (
    <AbsoluteFill>
      {/* ── c0: fading log tail ── */}
      <div
        style={{
          position: "absolute",
          top: 200,
          left: 120,
          fontFamily: F.mono,
          fontSize: 16,
          lineHeight: 1.9,
          color: "rgba(11,15,20,0.35)",
          opacity: logIn * logDim,
          filter: `blur(${logBlur}px)`,
          transform: `translateY(${(1 - logIn) * 14}px)`,
          whiteSpace: "pre",
        }}
      >
        {`{"level":"info","msg":"POST /v1/consumers/acme/events","status":201,"dur":"3ms"}
{"level":"info","msg":"delivery.attempt","dlv":"dlv_4e9…","attempt":1,"status":503}
{"level":"warn","msg":"delivery.retry_scheduled","dlv":"dlv_4e9…","delay":"60s"}
{"level":"info","msg":"delivery.attempt","dlv":"dlv_4e9…","attempt":2,"status":200}
{"level":"info","msg":"POST /v1/consumers/acme/events","status":201,"dur":"2ms"}`}
      </div>

      {/* ── c1: browser-framed dashboard ── */}
      <div
        style={{
          position: "absolute",
          left: BROWSER_X,
          top: BROWSER_Y,
          opacity: browserIn,
          transform: `translateY(${(1 - browserIn) * 40}px)`,
          filter: `blur(${(1 - browserIn) * 4}px)`,
        }}
      >
        <BrowserFrame url="localhost:8080" width={BROWSER_W} height={BROWSER_H}>
          {/* ── Sidebar ── */}
          <div
            style={{
              width: SIDEBAR_W,
              height: BROWSER_H - 48,
              background: "#faf9f6",
              borderRight: "1px solid #e6e2d9",
              display: "flex",
              flexDirection: "column",
              flexShrink: 0,
            }}
          >
            {/* Logo area */}
            <div
              style={{
                display: "flex",
                alignItems: "center",
                gap: 10,
                padding: "14px 16px",
                borderBottom: "1px solid #e6e2d9",
              }}
            >
              <div
                style={{
                  width: 32,
                  height: 32,
                  borderRadius: 8,
                  border: "1px solid #e6e2d9",
                  background: "#f4f2ed",
                  display: "grid",
                  placeItems: "center",
                  overflow: "hidden",
                }}
              >
                <Img src={staticFile("sparrow-logo.svg")} style={{ width: 20, height: 20 }} />
              </div>
              <div style={{ display: "flex", flexDirection: "column" }}>
                <span
                  style={{
                    fontFamily: F.display,
                    fontWeight: 700,
                    fontSize: 13,
                    letterSpacing: "0.18em",
                    color: "#1c1a16",
                  }}
                >
                  SPARROW
                </span>
                <span
                  style={{
                    fontFamily: F.mono,
                    fontSize: 9,
                    letterSpacing: "0.12em",
                    color: "#6b6558",
                    marginTop: 2,
                  }}
                >
                  webhook delivery
                </span>
              </div>
            </div>

            {/* Nav items */}
            <nav style={{ display: "flex", flexDirection: "column", gap: 2, padding: 10 }}>
              {NAV_ITEMS.map((item, idx) => {
                const isActive = idx === 0; // Webhooks is the active page
                const hp = highlightProgress(idx);
                const isHighlighted = hp > 0.01;
                const highlightBg = isHighlighted
                  ? `rgba(0,173,216,${hp * 0.12})`
                  : isActive
                    ? "rgba(0,0,0,0.05)"
                    : "transparent";
                const highlightBorder = isHighlighted
                  ? `2px solid rgba(0,173,216,${hp * 0.6})`
                  : "2px solid transparent";
                return (
                  <div
                    key={item.label}
                    style={{
                      display: "flex",
                      alignItems: "center",
                      gap: 10,
                      padding: "7px 12px",
                      borderRadius: 8,
                      background: highlightBg,
                      border: highlightBorder,
                      position: "relative",
                      transition: "none",
                    }}
                  >
                    {isActive && !isHighlighted && (
                      <span
                        style={{
                          position: "absolute",
                          left: 0,
                          top: 6,
                          bottom: 6,
                          width: 2,
                          borderRadius: 1,
                          background: "#ea9d2b",
                        }}
                      />
                    )}
                    <svg
                      width={16}
                      height={16}
                      viewBox="0 0 24 24"
                      fill="none"
                      stroke={isHighlighted ? C.teal : isActive ? "#1c1a16" : "#6b6558"}
                      strokeWidth={1.8}
                      strokeLinecap="round"
                      strokeLinejoin="round"
                    >
                      <path d={item.icon} />
                    </svg>
                    <span
                      style={{
                        fontFamily: F.display,
                        fontSize: 13,
                        fontWeight: isActive || isHighlighted ? 600 : 400,
                        color: isHighlighted ? C.teal : isActive ? "#1c1a16" : "#6b6558",
                      }}
                    >
                      {item.label}
                    </span>
                  </div>
                );
              })}
            </nav>

            {/* Push Event button */}
            <div style={{ padding: "8px 10px" }}>
              <div
                style={{
                  display: "flex",
                  alignItems: "center",
                  justifyContent: "center",
                  gap: 6,
                  padding: "8px 0",
                  borderRadius: 8,
                  background: "#ea9d2b",
                  color: "#fff",
                  fontFamily: F.display,
                  fontSize: 13,
                  fontWeight: 600,
                }}
              >
                <span style={{ fontSize: 16, lineHeight: 1 }}>+</span> Push Event
              </div>
            </div>

            <div style={{ flex: 1 }} />

            {/* Fleet health indicator */}
            <div
              style={{
                padding: "12px 16px",
                borderTop: "1px solid #e6e2d9",
                display: "flex",
                alignItems: "center",
                gap: 8,
                fontFamily: F.mono,
                fontSize: 11,
                color: "#6b6558",
              }}
            >
              <span
                style={{
                  width: 7,
                  height: 7,
                  borderRadius: "50%",
                  background: "#16875a",
                }}
              />
              All systems go
            </div>
          </div>

          {/* ── Main content area ── */}
          <div
            style={{
              flex: 1,
              background: "#faf9f6",
              padding: "24px 28px",
              display: "flex",
              flexDirection: "column",
              gap: 18,
              overflow: "hidden",
            }}
          >
            {/* Pulse bar across top */}
            <div
              style={{
                position: "absolute",
                top: 48,
                left: SIDEBAR_W,
                right: 0,
                height: 4,
                background: "linear-gradient(90deg, #16875a 70%, #b3790a 85%, #d5362f 100%)",
                opacity: 0.6,
              }}
            />

            {/* Page header */}
            <div
              style={{
                display: "flex",
                alignItems: "baseline",
                justifyContent: "space-between",
                marginTop: 4,
              }}
            >
              <span
                style={{
                  fontFamily: F.display,
                  fontSize: 22,
                  fontWeight: 600,
                  color: "#1c1a16",
                  letterSpacing: "-0.01em",
                }}
              >
                Webhooks
              </span>
              <span
                style={{
                  fontFamily: F.mono,
                  fontSize: 11,
                  color: "#6b6558",
                  letterSpacing: "0.04em",
                }}
              >
                consumer: acme
              </span>
            </div>

            {/* Webhooks table */}
            <div
              style={{
                background: "#ffffff",
                borderRadius: 10,
                border: "1px solid #e6e2d9",
                overflow: "hidden",
                boxShadow: "0 1px 3px rgba(0,0,0,0.04)",
              }}
            >
              {/* Table header */}
              <div
                style={{
                  display: "flex",
                  alignItems: "center",
                  height: 38,
                  padding: "0 18px",
                  borderBottom: "1px solid #e6e2d9",
                  fontFamily: F.mono,
                  fontSize: 11,
                  fontWeight: 600,
                  letterSpacing: "0.06em",
                  color: "#6b6558",
                }}
              >
                <div style={{ width: 160 }}>NAME</div>
                <div style={{ width: 240 }}>URL</div>
                <div style={{ width: 80, textAlign: "center" }}>HEALTH</div>
                <div style={{ flex: 1, textAlign: "right", paddingRight: 8 }}>DELIVERIES</div>
              </div>

              {/* Rows */}
              {WEBHOOK_ROWS.map((row) => (
                <div
                  key={row.name}
                  style={{
                    display: "flex",
                    alignItems: "center",
                    height: 48,
                    padding: "0 18px",
                    borderBottom: "1px solid rgba(230,226,217,0.5)",
                    fontFamily: F.display,
                    fontSize: 13,
                    color: "#1c1a16",
                  }}
                >
                  <div
                    style={{
                      width: 160,
                      fontWeight: 500,
                      overflow: "hidden",
                      textOverflow: "ellipsis",
                      whiteSpace: "nowrap",
                    }}
                  >
                    {row.name}
                  </div>
                  <div
                    style={{
                      width: 240,
                      fontFamily: F.mono,
                      fontSize: 11,
                      color: "#6b6558",
                      overflow: "hidden",
                      textOverflow: "ellipsis",
                      whiteSpace: "nowrap",
                    }}
                  >
                    {row.url}
                  </div>
                  <div style={{ width: 80, display: "flex", justifyContent: "center" }}>
                    <span
                      style={{
                        display: "inline-flex",
                        alignItems: "center",
                        gap: 5,
                        padding: "3px 10px",
                        borderRadius: 999,
                        fontFamily: F.mono,
                        fontSize: 11,
                        fontWeight: 500,
                        color: healthColor(row.health),
                        background: healthBg(row.health),
                      }}
                    >
                      <span
                        style={{
                          width: 6,
                          height: 6,
                          borderRadius: "50%",
                          background: healthColor(row.health),
                        }}
                      />
                      {row.health}
                    </span>
                  </div>
                  <div style={{ flex: 1, display: "flex", justifyContent: "flex-end", paddingRight: 8 }}>
                    <Sparkline width={100} height={28} opacity={0.8} />
                  </div>
                </div>
              ))}
            </div>
          </div>
        </BrowserFrame>
      </div>

      {/* ── "same binary" chip ── */}
      <div
        style={{
          position: "absolute",
          left: BROWSER_X + BROWSER_W / 2 - 160,
          top: BROWSER_Y + BROWSER_H + 10,
          display: "flex",
          alignItems: "center",
          gap: 10,
          fontFamily: F.mono,
          fontSize: 16,
          color: C.ink,
          opacity: chipIn,
          transform: `translateY(${(1 - chipIn) * 10}px)`,
        }}
      >
        <span
          style={{
            padding: "8px 18px",
            borderRadius: 10,
            background: "rgba(0,173,216,0.08)",
            border: "1px solid rgba(0,173,216,0.25)",
            color: C.tealDeep,
            fontWeight: 500,
          }}
        >
          same binary · SPARROW_SERVE_UI=true
        </span>
      </div>

      {/* ── c2: portal card ── */}
      <div
        style={{
          position: "absolute",
          left: PORTAL_X,
          top: PORTAL_Y,
          opacity: portalIn,
          transform: `translateX(${(1 - portalIn) * 30}px) translateY(${(1 - portalIn) * 10}px)`,
          filter: `blur(${(1 - portalIn) * 4}px)`,
        }}
      >
        <div
          style={{
            ...card,
            width: PORTAL_W,
            padding: 0,
            overflow: "hidden",
            boxShadow: "0 30px 70px rgba(11,15,20,0.18)",
          }}
        >
          {/* Portal header */}
          <div
            style={{
              padding: "16px 20px",
              borderBottom: "1px solid rgba(11,15,20,0.08)",
              display: "flex",
              alignItems: "center",
              justifyContent: "space-between",
            }}
          >
            <div style={{ display: "flex", flexDirection: "column", gap: 4 }}>
              <span
                style={{
                  fontFamily: F.display,
                  fontSize: 16,
                  fontWeight: 600,
                  color: "#1c1a16",
                }}
              >
                acme · consumer portal
              </span>
              <span
                style={{
                  fontFamily: F.mono,
                  fontSize: 11,
                  color: "#6b6558",
                }}
              >
                /portal#token=spt_…
              </span>
            </div>
            <span
              style={{
                width: 8,
                height: 8,
                borderRadius: "50%",
                background: "#16875a",
              }}
            />
          </div>

          {/* Portal webhook row */}
          <div
            style={{
              padding: "12px 20px",
              borderBottom: "1px solid rgba(11,15,20,0.06)",
              display: "flex",
              alignItems: "center",
              gap: 10,
            }}
          >
            <span
              style={{
                fontFamily: F.display,
                fontSize: 13,
                fontWeight: 500,
                color: "#1c1a16",
              }}
            >
              partner-acme
            </span>
            <span
              style={{
                display: "inline-flex",
                alignItems: "center",
                gap: 4,
                padding: "2px 8px",
                borderRadius: 999,
                fontFamily: F.mono,
                fontSize: 10,
                fontWeight: 500,
                color: "#16875a",
                background: "rgba(22,135,90,0.10)",
              }}
            >
              <span
                style={{
                  width: 5,
                  height: 5,
                  borderRadius: "50%",
                  background: "#16875a",
                }}
              />
              healthy
            </span>
            <span
              style={{
                marginLeft: "auto",
                fontFamily: F.mono,
                fontSize: 10,
                color: "#6b6558",
              }}
            >
              api.acme.example/hooks
            </span>
          </div>

          {/* "Add endpoint" button */}
          <div style={{ padding: "14px 20px" }}>
            <div
              style={{
                display: "inline-flex",
                alignItems: "center",
                gap: 6,
                padding: "7px 16px",
                borderRadius: 8,
                border: "1px solid #e6e2d9",
                fontFamily: F.display,
                fontSize: 13,
                fontWeight: 500,
                color: "#1c1a16",
                cursor: "default",
              }}
            >
              <span style={{ fontSize: 16, lineHeight: 1 }}>+</span> Add endpoint
            </div>
          </div>
        </div>
      </div>

      {/* ── c3: OTel trace waterfall ── */}
      <div
        style={{
          position: "absolute",
          left: TRACE_X,
          top: TRACE_Y,
          width: TRACE_W,
          opacity: otelIn,
          transform: `translateY(${(1 - otelIn) * 16}px)`,
          filter: `blur(${(1 - otelIn) * 4}px)`,
        }}
      >
        {/* Label */}
        <div
          style={{
            fontFamily: F.mono,
            fontSize: 14,
            fontWeight: 600,
            letterSpacing: "0.10em",
            color: C.tealDeep,
            marginBottom: 10,
          }}
        >
          OpenTelemetry · traces + metrics + logs
        </div>
        {/* Waterfall bars */}
        <div style={{ display: "flex", flexDirection: "column", gap: 5 }}>
          {([
            { label: "POST /v1/…/events", start: 0, width: 0.95 },
            { label: "event.fanout", start: 0.08, width: 0.42 },
            { label: "webhook.deliver", start: 0.28, width: 0.55 },
            { label: "http.post partner-acme", start: 0.38, width: 0.38 },
          ] as const).map((span, i) => {
            const barIn = r(c3 + i * 8, 18);
            const barWidth = ease(frame, c3 + i * 8, c3 + i * 8 + 20, 0, 1);
            return (
              <div
                key={span.label}
                style={{
                  position: "relative",
                  height: 28,
                  opacity: barIn,
                }}
              >
                {/* Span label */}
                <span
                  style={{
                    position: "absolute",
                    left: span.start * TRACE_W - 2,
                    top: 0,
                    fontFamily: F.mono,
                    fontSize: 12,
                    color: C.ink,
                    opacity: 0.7,
                    whiteSpace: "nowrap",
                    transform: "translateY(-1px)",
                  }}
                >
                  {span.label}
                </span>
                {/* Bar */}
                <div
                  style={{
                    position: "absolute",
                    left: span.start * TRACE_W,
                    top: 16,
                    width: span.width * TRACE_W * barWidth,
                    height: 10,
                    borderRadius: 3,
                    background:
                      i === 0
                        ? C.teal
                        : i === 3
                          ? C.coral
                          : `rgba(0,173,216,${0.5 + i * 0.1})`,
                    boxShadow:
                      i === 0
                        ? "0 2px 8px rgba(0,173,216,0.3)"
                        : "none",
                  }}
                />
              </div>
            );
          })}
        </div>
      </div>

      {/* ── SFX ── */}
      <Sequence from={c1}>
        <Audio src={staticFile("sfx/whoosh-short.mp3")} volume={0.18} />
      </Sequence>
      <Sequence from={c2}>
        <Audio src={staticFile("sfx/ping.mp3")} volume={0.18} />
      </Sequence>
      <Sequence from={c3}>
        <Audio src={staticFile("sfx/chime.mp3")} volume={0.15} />
      </Sequence>
    </AbsoluteFill>
  );
};

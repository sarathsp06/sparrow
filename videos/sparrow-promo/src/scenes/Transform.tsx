import React from "react";
import {
  AbsoluteFill,
  Img,
  interpolate,
  spring,
  staticFile,
  useCurrentFrame,
  useVideoConfig,
} from "remotion";
import { C, F } from "../theme";

// Payload transformation: Go-template recipes reshape one event for each destination.
// The seven recipes ship in satellites/recipes (docs/satellites/recipes.mdx).
const DESTS = [
  { label: "Slack", icon: "icons/slack.svg" },
  { label: "Discord", icon: "icons/discord.svg" },
  { label: "Email", icon: "icons/sendgrid.svg" },
  { label: "SMS", icon: "icons/twilio.svg" },
  { label: "Push", icon: "icons/ntfy.svg" },
  { label: "PagerDuty", icon: "icons/pagerduty.svg" },
  { label: "ClickHouse", icon: "icons/clickhouse.svg" },
];

export const Transform: React.FC = () => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();

  const pop = (delay: number) => {
    const s = spring({ frame: frame - delay, fps, config: { damping: 200 } });
    return { opacity: s, transform: `translateY(${(1 - s) * 20}px)` };
  };
  const arrow = interpolate(frame, [40, 60], [0, 1], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
  });
  const footnote = spring({ frame: frame - 160, fps, config: { damping: 200 } });
  const caption = spring({ frame: frame - 130, fps, config: { damping: 200 } });

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
        <span style={{ color: C.coralText }}>&#10033;</span> ONE EVENT, ANY SHAPE
      </div>

      <AbsoluteFill style={{ justifyContent: "center", alignItems: "center" }}>
        <div style={{ display: "flex", alignItems: "center", gap: 60 }}>
          {/* Source payload */}
          <div
            style={{
              background: C.navy,
              borderRadius: 16,
              padding: "34px 40px",
              fontFamily: F.mono,
              fontSize: 26,
              lineHeight: 1.8,
              color: C.cream,
              boxShadow: "0 24px 60px rgba(11,15,20,0.25)",
              ...pop(8),
            }}
          >
            <div style={{ color: "rgba(230,237,243,0.5)", fontSize: 19, marginBottom: 10 }}>
              order.created
            </div>
            <div>{"{"}</div>
            <div style={{ paddingLeft: 32 }}>
              <span style={{ color: C.teal }}>&quot;order_id&quot;</span>: &quot;ord_1&quot;,
            </div>
            <div style={{ paddingLeft: 32 }}>
              <span style={{ color: C.teal }}>&quot;amount&quot;</span>: 42
            </div>
            <div>{"}"}</div>
          </div>

          {/* Transform arrow */}
          <div style={{ textAlign: "center", opacity: arrow }}>
            <div
              style={{
                fontFamily: F.mono,
                fontSize: 19,
                color: C.ink,
                opacity: 0.6,
                marginBottom: 12,
              }}
            >
              Go template
              <br />
              transform
            </div>
            <svg width={150} height={30}>
              <line
                x1={0}
                y1={15}
                x2={150 * arrow - 14}
                y2={15}
                stroke={C.coral}
                strokeWidth={5}
                strokeLinecap="round"
              />
              <path
                d={`M ${150 * arrow - 18} 4 L ${150 * arrow} 15 L ${150 * arrow - 18} 26 Z`}
                fill={C.coral}
              />
            </svg>
          </div>

          {/* Destinations */}
          <div style={{ display: "flex", flexDirection: "column", gap: 16 }}>
            {DESTS.map((d, i) => (
              <div
                key={d.label}
                style={{
                  display: "flex",
                  alignItems: "center",
                  gap: 16,
                  fontFamily: F.display,
                  fontSize: 30,
                  fontWeight: 500,
                  color: C.ink,
                  background: "rgba(255,255,255,0.6)",
                  border: "1px solid rgba(11,15,20,0.12)",
                  borderRadius: 999,
                  padding: "12px 34px",
                  boxShadow: "0 8px 22px rgba(11,15,20,0.06)",
                  ...pop(56 + i * 12),
                }}
              >
                <Img src={staticFile(d.icon)} style={{ width: 30, height: 30 }} />
                {d.label}
              </div>
            ))}
            <div
              style={{
                display: "flex",
                alignItems: "center",
                gap: 16,
                fontFamily: F.display,
                fontSize: 30,
                fontWeight: 500,
                color: C.coralText,
                border: `2px dashed ${C.coral}`,
                borderRadius: 999,
                padding: "12px 34px",
                ...pop(56 + DESTS.length * 12),
              }}
            >
              <Img src={staticFile("icons/webhook-coral.svg")} style={{ width: 30, height: 30 }} />
              &#8230;or any HTTP API
            </div>
          </div>
        </div>
      </AbsoluteFill>

      <div
        style={{
          position: "absolute",
          bottom: 150,
          width: "100%",
          textAlign: "center",
          fontFamily: F.display,
          fontSize: 36,
          fontWeight: 500,
          color: C.ink,
          opacity: caption,
          transform: `translateY(${(1 - caption) * 16}px)`,
        }}
      >
        Same event. Different payloads. Still just webhooks underneath.
      </div>

      {/* CLI is just a helper */}
      <div
        style={{
          position: "absolute",
          bottom: 76,
          width: "100%",
          textAlign: "center",
          fontFamily: F.mono,
          fontSize: 22,
          color: "rgba(11,15,20,0.6)",
          opacity: footnote,
          transform: `translateY(${(1 - footnote) * 14}px)`,
        }}
      >
        $ sparrow use slack --event order.created
        <span style={{ opacity: 0.65 }}>&nbsp;&nbsp;&#8212; one command via the helper CLI</span>
      </div>
    </AbsoluteFill>
  );
};

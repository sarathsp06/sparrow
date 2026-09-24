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
import { Kicker } from "../components/Kicker";

// Payload transformation: Go-template recipes reshape one event for each destination.
// The seven recipes ship in satellites/recipes (docs/satellites/recipes.mdx).
// `fmt` is each recipe's target format, from its yaml `description`.
const DESTS = [
  { label: "Slack", icon: "icons/slack.svg", fmt: "Block Kit" },
  { label: "Discord", icon: "icons/discord.svg", fmt: "embeds" },
  { label: "Email", icon: "icons/sendgrid.svg", fmt: "SendGrid v3" },
  { label: "SMS", icon: "icons/twilio.svg", fmt: "Twilio Messages" },
  { label: "Push", icon: "icons/ntfy.svg", fmt: "ntfy topic" },
  { label: "PagerDuty", icon: "icons/pagerduty.svg", fmt: "Events API v2" },
  { label: "ClickHouse", icon: "icons/clickhouse.svg", fmt: "JSONEachRow" },
];
const DEST_AT = (i: number) => 56 + i * 12;

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
    <AbsoluteFill>
      <Kicker text="ONE EVENT, ANY SHAPE" />

      <AbsoluteFill style={{ justifyContent: "center", alignItems: "center", paddingBottom: 90 }}>
        <div style={{ display: "flex", alignItems: "center", gap: 48 }}>
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

          {/* Transform arrow — the template line is straight from satellites/recipes/slack.yaml */}
          <div style={{ textAlign: "center", opacity: arrow, width: 330 }}>
            <div
              style={{
                display: "inline-block",
                fontFamily: F.mono,
                fontSize: 21,
                color: C.cream,
                background: C.navy,
                borderRadius: 10,
                padding: "10px 16px",
                marginBottom: 14,
                boxShadow: "0 10px 24px rgba(11,15,20,0.18)",
              }}
            >
              <span style={{ color: C.coral }}>{"{{"}</span>.event_name <span style={{ color: C.teal }}>| json</span>
              <span style={{ color: C.coral }}>{"}}"}</span>
            </div>
            <div style={{ fontFamily: F.mono, fontSize: 18, color: C.ink, opacity: 0.6, marginBottom: 10 }}>
              Go template recipe
            </div>
            <svg width={330} height={30} style={{ overflow: "visible" }}>
              <line
                x1={0}
                y1={15}
                x2={330 * arrow - 14}
                y2={15}
                stroke={C.coral}
                strokeWidth={5}
                strokeLinecap="round"
              />
              <path
                d={`M ${330 * arrow - 18} 4 L ${330 * arrow} 15 L ${330 * arrow - 18} 26 Z`}
                fill={C.coral}
              />
              {DESTS.map((d, i) => {
                const t = interpolate(frame, [DEST_AT(i) - 12, DEST_AT(i)], [0, 1], {
                  extrapolateLeft: "clamp",
                  extrapolateRight: "clamp",
                });
                if (t <= 0 || t >= 1) return null;
                return (
                  <circle
                    key={d.label}
                    cx={t * 320}
                    cy={15}
                    r={10}
                    fill={C.cream}
                    stroke={C.coral}
                    strokeWidth={4}
                    style={{ filter: "drop-shadow(0 0 6px rgba(249,115,22,0.8))" }}
                  />
                );
              })}
            </svg>
          </div>

          {/* Destinations */}
          <div style={{ display: "flex", flexDirection: "column", gap: 11 }}>
            {DESTS.map((d, i) => {
              const flash = interpolate(frame, [DEST_AT(i), DEST_AT(i) + 16], [1, 0], {
                extrapolateLeft: "clamp",
                extrapolateRight: "clamp",
              });
              return (
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
                  border: `1px solid rgba(11,15,20,${0.12 + flash * 0.5})`,
                  borderRadius: 999,
                  padding: "9px 34px",
                  minWidth: 470,
                  boxShadow: `0 8px 22px rgba(11,15,20,0.06), 0 0 ${flash * 30}px rgba(249,115,22,${flash * 0.5})`,
                  ...pop(DEST_AT(i)),
                }}
              >
                <Img src={staticFile(d.icon)} style={{ width: 30, height: 30 }} />
                {d.label}
                <span
                  style={{
                    marginLeft: "auto",
                    fontFamily: F.mono,
                    fontSize: 18,
                    fontWeight: 400,
                    color: C.tealDeep,
                    opacity: interpolate(frame, [DEST_AT(i) + 6, DEST_AT(i) + 14], [0, 1], {
                      extrapolateLeft: "clamp",
                      extrapolateRight: "clamp",
                    }),
                  }}
                >
                  {d.fmt}
                </span>
              </div>
              );
            })}
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
                padding: "9px 34px",
                minWidth: 470,
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

import React from "react";
import {
  AbsoluteFill,
  Easing,
  interpolate,
  spring,
  useCurrentFrame,
  useVideoConfig,
} from "remotion";
import { C, F } from "../theme";
import { Grain } from "../components/Backdrop";
import { Kicker } from "../components/Kicker";

const LINES = [
  ["Your", "app", "emits", "events."],
  ["The", "rest", "of", "your", "stack", "expects", "webhooks."],
];
const PAIN = ["Retries.", "Signing.", "Health checks.", "Every time."];

// Illustrative DIY delivery log (no Sparrow claims) scrolling behind the headline.
const LOG = [
  ["POST https://partner-a.example/hooks", "503", "retry 1/5"],
  ["POST https://erp.internal/events", "timeout", "10s"],
  ["POST https://hooks.slack.com/services/…", "429", "rate limited"],
  ["POST https://billing.partner.example/wh", "500", "retry 2/5"],
  ["verify signature", "mismatch", "dropped"],
  ["POST https://crm.internal/webhooks", "ECONNRESET", "retry 3/5"],
  ["POST https://partner-b.example/in", "502", "retry 1/5"],
  ["dead-letter queue", "+1", "page on-call"],
];

export const Hook: React.FC = () => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();

  const underline = interpolate(frame, [104, 124], [0, 1], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
    easing: Easing.bezier(0.65, 0, 0.35, 1),
  });
  // Slow camera drift into the headline.
  const cam = interpolate(frame, [0, 160], [1, 1.05]);

  let wordIndex = 0;

  return (
    <AbsoluteFill style={{ background: C.navy, overflow: "hidden" }}>
      {/* Ghost log — the mess you'd otherwise own */}
      <AbsoluteFill
        style={{
          padding: "0 120px",
          translate: `0px ${160 - frame * 1.6}px`,
          maskImage: "linear-gradient(to bottom, transparent, black 25%, black 75%, transparent)",
        }}
      >
        {[...LOG, ...LOG, ...LOG].map(([req, code, note], i) => (
          <div
            key={i}
            style={{
              display: "flex",
              gap: 40,
              fontFamily: F.mono,
              fontSize: 26,
              lineHeight: "62px",
              color: "rgba(244,242,237,0.13)",
              whiteSpace: "nowrap",
              marginLeft: (i % 3) * 90,
            }}
          >
            <span>{req}</span>
            <span style={{ color: "rgba(229,72,77,0.4)" }}>{code}</span>
            <span>{note}</span>
          </div>
        ))}
      </AbsoluteFill>

      <AbsoluteFill
        style={{ background: "radial-gradient(ellipse 60% 50% at 50% 50%, rgba(24,22,21,0.92) 40%, rgba(24,22,21,0.4))" }}
      />

      <Kicker text="WEBHOOK DELIVERY" color={C.cream} instant />

      <AbsoluteFill style={{ justifyContent: "center", alignItems: "center", scale: String(cam) }}>
        <div style={{ textAlign: "center" }}>
          {LINES.map((words, li) => (
            <div
              key={li}
              style={{
                fontFamily: F.display,
                fontWeight: 600,
                fontSize: 80,
                lineHeight: 1.18,
                letterSpacing: "-0.035em",
                color: li === 0 ? "rgba(244,242,237,0.7)" : C.cream,
              }}
            >
              {words.map((w) => {
                const delay = 4 + wordIndex++ * 4;
                const s = spring({ frame: frame - delay, fps, config: { damping: 200 } });
                return (
                  <span
                    key={w + delay}
                    style={{
                      display: "inline-block",
                      marginRight: "0.25em",
                      // Frame 1 is already legible for silent autoplay.
                      opacity: Math.max(s, delay < 8 ? 0.6 : 0),
                      filter: `blur(${(1 - s) * 12}px)`,
                      translate: `0px ${(1 - s) * 28}px`,
                      color: w === "webhooks." ? C.coral : undefined,
                    }}
                  >
                    {w}
                  </span>
                );
              })}
            </div>
          ))}

          <div
            style={{
              marginTop: 54,
              display: "inline-flex",
              flexDirection: "column",
              alignItems: "stretch",
            }}
          >
            <div style={{ display: "flex", gap: 22, justifyContent: "center" }}>
              {PAIN.map((p, i) => {
                const s = spring({ frame: frame - (66 + i * 10), fps, config: { damping: 11, stiffness: 220 } });
                return (
                  <span
                    key={p}
                    style={{
                      fontFamily: F.mono,
                      fontSize: 32,
                      fontWeight: 500,
                      color: i === PAIN.length - 1 ? C.coral : "rgba(244,242,237,0.85)",
                      opacity: interpolate(s, [0, 0.3], [0, 1], { extrapolateRight: "clamp" }),
                      scale: String(interpolate(s, [0, 1], [1.6, 1])),
                      display: "inline-block",
                    }}
                  >
                    {p}
                  </span>
                );
              })}
            </div>
            <div
              style={{
                height: 6,
                marginTop: 14,
                background: `linear-gradient(90deg, ${C.coral}, ${C.coralDeep})`,
                borderRadius: 3,
                transformOrigin: "left center",
                scale: `${underline} 1`,
                boxShadow: "0 0 24px rgba(249,115,22,0.55)",
              }}
            />
          </div>
        </div>
      </AbsoluteFill>
      <Grain opacity={0.06} blend="screen" />
    </AbsoluteFill>
  );
};

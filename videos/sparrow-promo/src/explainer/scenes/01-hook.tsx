import React from "react";
import {
  AbsoluteFill,
  Easing,
  interpolate,
  spring,
  useCurrentFrame,
  useVideoConfig,
} from "remotion";
import { C, F } from "../../theme";
import { Grain } from "../../components/Backdrop";
import { Kicker } from "../../components/Kicker";
import { cueFrame, cueEndFrame } from "../timing";

// Frame 1 — Webhooks, again
// Reproduces the promo Hook's look exactly: dark navy ground, scrolling ghost
// log, radial dim, grain, Kicker "WEBHOOK DELIVERY" visible from frame 0.
// Retimed to VO cues instead of hardcoded frame numbers.

const ID = "01-hook";
const c0 = cueFrame(ID, 0); // "Your app emits events."
const c1 = cueFrame(ID, 1); // "The rest of your stack expects webhooks."
const c2 = cueFrame(ID, 2); // "Which means retries. Signing. Health checks. Every time."
const c2End = cueEndFrame(ID, 2);

const LINES = [
  ["Your", "app", "emits", "events."],
  ["The", "rest", "of", "your", "stack", "expects", "webhooks."],
];

const PAIN = ["Retries.", "Signing.", "Health checks.", "Every time."];

// Illustrative DIY delivery log scrolling behind the headline (same as promo).
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

  // --- Word reveals ---
  // Line 1 words reveal starting at c0, spread across the cue duration (~1s)
  const c0Dur = cueEndFrame(ID, 0) - c0;
  const line1WordGap = Math.max(2, Math.floor(c0Dur / LINES[0].length));

  // Line 2 words reveal starting at c1
  const c1Dur = cueEndFrame(ID, 1) - c1;
  const line2WordGap = Math.max(2, Math.floor(c1Dur / LINES[1].length));

  // Pain items reveal at c2, one per spoken word-group
  // "retries. signing. health checks. every time." — 4 items across ~3.7s
  const c2Dur = c2End - c2;
  const painGap = Math.max(6, Math.floor(c2Dur / PAIN.length));

  // Underline sweeps after all pain items are in
  const underlineStart = c2 + painGap * (PAIN.length - 1) + 8;
  const underline = interpolate(frame, [underlineStart, underlineStart + 20], [0, 1], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
    easing: Easing.bezier(0.65, 0, 0.35, 1),
  });

  // Slow camera drift into the headline
  const totalFrames = Math.ceil(8.499 * 30);
  const cam = interpolate(frame, [0, totalFrames], [1, 1.05]);

  return (
    <AbsoluteFill style={{ background: C.navy, overflow: "hidden" }}>
      {/* Ghost log — the mess you'd otherwise own */}
      <AbsoluteFill
        style={{
          padding: "0 120px",
          translate: `0px ${160 - frame * 1.6}px`,
          maskImage:
            "linear-gradient(to bottom, transparent, black 25%, black 75%, transparent)",
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
        style={{
          background:
            "radial-gradient(ellipse 60% 50% at 50% 50%, rgba(24,22,21,0.92) 40%, rgba(24,22,21,0.4))",
        }}
      />

      <Kicker text="WEBHOOK DELIVERY" color={C.cream} instant />

      <AbsoluteFill
        style={{
          justifyContent: "center",
          alignItems: "center",
          scale: String(cam),
        }}
      >
        <div style={{ textAlign: "center" }}>
          {/* Line 1: reveals at c0 */}
          <div
            style={{
              fontFamily: F.display,
              fontWeight: 600,
              fontSize: 80,
              lineHeight: 1.18,
              letterSpacing: "-0.035em",
              color: "rgba(244,242,237,0.7)",
            }}
          >
            {LINES[0].map((w, wi) => {
              const delay = c0 + wi * line1WordGap;
              const s = spring({
                frame: frame - delay,
                fps,
                config: { damping: 200 },
              });
              return (
                <span
                  key={`l0-${wi}`}
                  style={{
                    display: "inline-block",
                    marginRight: "0.25em",
                    opacity: s,
                    filter: `blur(${(1 - s) * 12}px)`,
                    translate: `0px ${(1 - s) * 28}px`,
                  }}
                >
                  {w}
                </span>
              );
            })}
          </div>

          {/* Line 2: reveals at c1, "webhooks." in coral */}
          <div
            style={{
              fontFamily: F.display,
              fontWeight: 600,
              fontSize: 80,
              lineHeight: 1.18,
              letterSpacing: "-0.035em",
              color: C.cream,
            }}
          >
            {LINES[1].map((w, wi) => {
              const delay = c1 + wi * line2WordGap;
              const s = spring({
                frame: frame - delay,
                fps,
                config: { damping: 200 },
              });
              return (
                <span
                  key={`l1-${wi}`}
                  style={{
                    display: "inline-block",
                    marginRight: "0.25em",
                    opacity: s,
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

          {/* Pain line: mono items reveal one per spoken word-group at c2 */}
          <div
            style={{
              marginTop: 54,
              display: "inline-flex",
              flexDirection: "column",
              alignItems: "stretch",
            }}
          >
            <div
              style={{
                display: "flex",
                gap: 22,
                justifyContent: "center",
              }}
            >
              {PAIN.map((p, i) => {
                const delay = c2 + i * painGap;
                const s = spring({
                  frame: frame - delay,
                  fps,
                  config: { damping: 200 },
                });
                return (
                  <span
                    key={p}
                    style={{
                      fontFamily: F.mono,
                      fontSize: 32,
                      fontWeight: 500,
                      color:
                        i === PAIN.length - 1
                          ? C.coral
                          : "rgba(244,242,237,0.85)",
                      opacity: s,
                      scale: String(interpolate(s, [0, 1], [1.15, 1])),
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

import React from "react";
import {
  AbsoluteFill,
  Audio,
  Sequence,
  spring,
  staticFile,
  useCurrentFrame,
  useVideoConfig,
} from "remotion";
import { C, F } from "../../theme";
import { cueFrame, useReveal, ease, frameTiming } from "../timing";
import { StepRail, StepTitle, card } from "../ui";
import { Wire, Comet, Ripple } from "../../components/Wire";

const ID = "06-sign";

// Envelope fields — standard webhook envelope (FLOWS.md section 5)
const ENVELOPE_FIELDS = [
  { key: "version", value: '"1"' },
  { key: "event_id", value: '"evt_7c1..."' },
  { key: "event_name", value: '"order.created"' },
  { key: "timestamp", value: '"2025-01-15T10:30:00Z"' },
  { key: "attempt", value: "1" },
  { key: "payload", value: '{ "order_id": "ord_123" }' },
];

// Custom body after Go template morph
const CUSTOM_BODY = [
  { key: "order", value: '"ord_123"' },
  { key: "amount", value: "49.99" },
  { key: "status", value: '"created"' },
];

// Method cycle tokens — proportional offsets within c3 cue
const METHODS = ["POST", "PUT", "PATCH", "DELETE"] as const;
const C3_DUR = frameTiming(ID).cues[3].dur;
// "Any method: POST, PUT, PATCH, DELETE." — 6 words, methods are words 2-5
const METHOD_OFFSETS = [
  (2 / 6) * C3_DUR,
  (3 / 6) * C3_DUR,
  (4 / 6) * C3_DUR,
  (5 / 6) * C3_DUR,
];

// Generic destinations (no brand logos) for c4
const DESTINATIONS = [
  { method: "POST", url: "hooks.partner.example" },
  { method: "PUT", url: "api.crm.example/orders/ord_123" },
  { method: "PATCH", url: "billing.internal/invoices" },
];

// Signature header fields for c5
const SIG_HEADERS = [
  { key: "webhook-id", value: "msg_7c1a2b..." },
  { key: "webhook-timestamp", value: "1737193800" },
  { key: "webhook-signature", value: "v1,K5oQ7r...=" },
];

// Card layout
const CARD_LEFT = 520;
const CARD_TOP = 280;
const CARD_W = 780;

export const Sign: React.FC = () => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();
  const r = useReveal();

  const c0 = cueFrame(ID, 0);
  const c1 = cueFrame(ID, 1);
  const c2 = cueFrame(ID, 2);
  const c3 = cueFrame(ID, 3);
  const c4 = cueFrame(ID, 4);
  const c5 = cueFrame(ID, 5);

  // Request card outline appears at c0
  const cardOutline = r(c0 + 8, 22);

  // c1: envelope body builds, then morphs to custom
  const envelopePop = r(c1);
  // "or your own shape" — proportional: ~6 words in, out of 11 words
  const c1dur = frameTiming(ID).cues[1].dur;
  const morphAt = cueFrame(ID, 1, (6 / 11) * c1dur);
  const templateChipPop = r(morphAt);
  const morphProgress = ease(frame, morphAt + 4, morphAt + 18);

  // c2: headers fill
  const headersPop = r(c2);
  const c2dur = frameTiming(ID).cues[2].dur;
  // "even an API token" at word 4/9, "encrypted at rest" at word 7/9
  const authAt = cueFrame(ID, 2, (4 / 9) * c2dur);
  const encryptedAt = cueFrame(ID, 2, (7 / 9) * c2dur);
  const authPop = r(authAt);
  const encryptTagPop = r(encryptedAt);

  // c3: method cycling
  const activeMethod = (() => {
    for (let i = METHODS.length - 1; i >= 0; i--) {
      if (frame >= cueFrame(ID, 3, METHOD_OFFSETS[i])) return i;
    }
    return -1;
  })();
  const methodPop = r(c3);
  // Settle on PUT after c3 ends
  const settledMethod = frame >= cueFrame(ID, 3, C3_DUR) ? 1 : activeMethod;
  const displayMethod = settledMethod >= 0 ? METHODS[settledMethod] : "POST";

  // c4: three destinations
  const destPops = DESTINATIONS.map((_, i) => r(c4 + 6 + i * 12));

  // c5: signature
  const sigPop = r(c5);
  const hmacPop = r(c5 + 20);
  const ed25519Pop = r(c5 + 30);
  const sealPop = spring({
    frame: frame - (c5 + 40),
    fps,
    config: { damping: 200 },
  });

  // Destination layout
  const DEST_LEFT = CARD_LEFT + CARD_W + 60;
  const DEST_TOP = 340;
  const DEST_GAP = 90;

  return (
    <AbsoluteFill>
      <StepRail active={2} />
      <StepTitle n={3} title="Build the request" at={c0} />

      {/* === Request card === */}
      <div
        style={{
          ...card,
          position: "absolute",
          left: CARD_LEFT,
          top: CARD_TOP,
          width: CARD_W,
          minHeight: 440,
          padding: 0,
          opacity: cardOutline,
          transform: `translateY(${(1 - cardOutline) * 14}px)`,
          overflow: "hidden",
        }}
      >
        {/* Method + URL line */}
        <div
          style={{
            padding: "18px 28px 14px",
            borderBottom: "1px solid rgba(11,15,20,0.08)",
            display: "flex",
            alignItems: "baseline",
            gap: 14,
            fontFamily: F.mono,
            fontSize: 22,
            fontWeight: 700,
            opacity: methodPop,
          }}
        >
          {/* Method token — cycles on c3 words */}
          <span
            style={{
              color: "#fff",
              background: C.tealDeep,
              borderRadius: 8,
              padding: "4px 14px",
              fontSize: 20,
              letterSpacing: "0.04em",
            }}
          >
            {frame >= c3 ? displayMethod : "POST"}
          </span>
          <span style={{ color: C.ink, fontSize: 20 }}>
            {frame >= c3 + Math.round(C3_DUR * 30)
              ? "https://api.crm.example/orders/ord_123"
              : "https://hooks.partner.example/webhooks"}
          </span>
        </div>

        {/* HEADERS section */}
        <div
          style={{
            padding: "14px 28px",
            borderBottom: "1px solid rgba(11,15,20,0.08)",
            opacity: headersPop,
          }}
        >
          <div
            style={{
              fontFamily: F.mono,
              fontSize: 13,
              color: "rgba(11,15,20,0.4)",
              letterSpacing: "0.1em",
              marginBottom: 10,
            }}
          >
            HEADERS
          </div>
          <div style={{ fontFamily: F.mono, fontSize: 20, lineHeight: 1.7, color: C.ink }}>
            {/* Content-Type — appears with c2 */}
            <div style={{ opacity: ease(frame, c2, c2 + 10) }}>
              <span style={{ color: C.tealDeep }}>Content-Type</span>: application/json
            </div>
            {/* X-Tenant */}
            <div style={{ opacity: ease(frame, c2 + 6, c2 + 14) }}>
              <span style={{ color: C.tealDeep }}>X-Tenant</span>: acme
            </div>
            {/* Authorization — encrypted */}
            <div style={{ opacity: authPop, display: "flex", alignItems: "center", gap: 10 }}>
              <span>
                <span style={{ color: C.tealDeep }}>Authorization</span>:{" "}
                Bearer {""}
              </span>
              <span
                style={{
                  fontFamily: F.mono,
                  fontSize: 20,
                  color: "rgba(11,15,20,0.35)",
                  letterSpacing: "0.12em",
                }}
              >
                {"--------"}
              </span>
              {/* Lock icon */}
              <svg width={16} height={20} viewBox="0 0 16 20" style={{ flexShrink: 0, marginLeft: 4 }}>
                <rect x={1} y={9} width={14} height={10} rx={2.5} fill={C.tealDeep} opacity={0.8} />
                <path d="M4 9 V6 a4 4 0 0 1 8 0 V9" stroke={C.tealDeep} strokeWidth={2} fill="none" />
              </svg>
            </div>
            {/* "encrypted at rest" tag */}
            {encryptTagPop > 0.01 && (
              <div
                style={{
                  display: "inline-flex",
                  alignItems: "center",
                  gap: 6,
                  fontFamily: F.mono,
                  fontSize: 14,
                  color: C.tealDeep,
                  background: "rgba(0,173,216,0.08)",
                  border: `1px solid rgba(0,173,216,0.3)`,
                  borderRadius: 999,
                  padding: "4px 12px",
                  marginTop: 6,
                  opacity: encryptTagPop,
                  transform: `translateY(${(1 - encryptTagPop) * 6}px)`,
                }}
              >
                encrypted at rest {"·"} AES-256-GCM
              </div>
            )}

            {/* c5: Signature headers */}
            {sigPop > 0.01 &&
              SIG_HEADERS.map((h, i) => {
                const hAt = c5 + 4 + i * 8;
                const hPop = ease(frame, hAt, hAt + 10);
                return (
                  <div key={h.key} style={{ opacity: hPop, marginTop: i === 0 ? 10 : 0 }}>
                    <span style={{ color: C.coral }}>{h.key}</span>: {h.value}
                  </div>
                );
              })}

            {/* Signing scheme chips */}
            {hmacPop > 0.01 && (
              <div
                style={{
                  display: "flex",
                  gap: 10,
                  marginTop: 10,
                  alignItems: "center",
                }}
              >
                <div
                  style={{
                    fontFamily: F.mono,
                    fontSize: 15,
                    color: C.tealDeep,
                    background: "rgba(0,173,216,0.10)",
                    border: `1.5px solid ${C.teal}`,
                    borderRadius: 999,
                    padding: "5px 14px",
                    opacity: hmacPop,
                    transform: `scale(${0.9 + hmacPop * 0.1})`,
                  }}
                >
                  HMAC-SHA256
                </div>
                {ed25519Pop > 0.01 && (
                  <div
                    style={{
                      fontFamily: F.mono,
                      fontSize: 15,
                      color: C.tealDeep,
                      background: "rgba(0,173,216,0.10)",
                      border: `1.5px solid ${C.teal}`,
                      borderRadius: 999,
                      padding: "5px 14px",
                      opacity: ed25519Pop,
                      transform: `scale(${0.9 + ed25519Pop * 0.1})`,
                    }}
                  >
                    Ed25519
                  </div>
                )}
                {/* Signed seal */}
                {sealPop > 0.01 && (
                  <div
                    style={{
                      display: "flex",
                      alignItems: "center",
                      gap: 6,
                      fontFamily: F.mono,
                      fontSize: 16,
                      fontWeight: 700,
                      color: C.tealDeep,
                      background: "rgba(0,173,216,0.12)",
                      border: `2px solid ${C.teal}`,
                      borderRadius: 12,
                      padding: "5px 14px",
                      opacity: sealPop,
                      transform: `scale(${0.85 + sealPop * 0.15})`,
                      boxShadow: `0 0 16px rgba(0,173,216,${0.3 * sealPop})`,
                    }}
                  >
                    <svg width={16} height={16} viewBox="0 0 18 18">
                      <circle cx={9} cy={9} r={8} fill="none" stroke={C.tealDeep} strokeWidth={2} />
                      <path d="M5 9.5 L8 12.5 L13 6.5" fill="none" stroke={C.tealDeep} strokeWidth={2} strokeLinecap="round" strokeLinejoin="round" />
                    </svg>
                    signed
                  </div>
                )}
              </div>
            )}
          </div>
        </div>

        {/* BODY section */}
        <div style={{ padding: "14px 28px 20px", opacity: envelopePop }}>
          <div
            style={{
              fontFamily: F.mono,
              fontSize: 13,
              color: "rgba(11,15,20,0.4)",
              letterSpacing: "0.1em",
              marginBottom: 8,
              display: "flex",
              alignItems: "center",
              gap: 10,
            }}
          >
            BODY
            {/* Go template chip */}
            {templateChipPop > 0.01 && (
              <span
                style={{
                  fontSize: 12,
                  color: C.coral,
                  background: "rgba(249,115,22,0.1)",
                  border: `1px solid rgba(249,115,22,0.35)`,
                  borderRadius: 999,
                  padding: "2px 10px",
                  opacity: templateChipPop,
                  transform: `translateX(${(1 - templateChipPop) * 12}px)`,
                  display: "inline-block",
                }}
              >
                {"{{ .payload.order_id }}"}
              </span>
            )}
          </div>
          <div
            style={{
              fontFamily: F.mono,
              fontSize: 20,
              lineHeight: 1.6,
              color: C.ink,
            }}
          >
            {morphProgress < 0.5 ? (
              // Envelope JSON
              <>
                <div>{"{"}</div>
                {ENVELOPE_FIELDS.map((f, i) => {
                  const fieldAt = c1 + 4 + i * 5;
                  const fieldPop = ease(frame, fieldAt, fieldAt + 8);
                  return (
                    <div
                      key={f.key}
                      style={{
                        paddingLeft: 20,
                        opacity: fieldPop,
                      }}
                    >
                      <span style={{ color: C.tealDeep }}>&quot;{f.key}&quot;</span>: {f.value}
                      {i < ENVELOPE_FIELDS.length - 1 ? "," : ""}
                    </div>
                  );
                })}
                <div>{"}"}</div>
              </>
            ) : (
              // Custom morphed JSON
              <>
                <div>{"{"}</div>
                {CUSTOM_BODY.map((f, i) => {
                  const fPop = ease(frame, morphAt + 6 + i * 4, morphAt + 14 + i * 4);
                  return (
                    <div
                      key={f.key}
                      style={{
                        paddingLeft: 20,
                        opacity: fPop,
                      }}
                    >
                      <span style={{ color: C.tealDeep }}>&quot;{f.key}&quot;</span>: {f.value}
                      {i < CUSTOM_BODY.length - 1 ? "," : ""}
                    </div>
                  );
                })}
                <div>{"}"}</div>
              </>
            )}
          </div>
        </div>
      </div>

      {/* === c4: Three generic destination endpoints === */}
      {DESTINATIONS.map((dest, i) => {
        const pop = destPops[i];
        const y = DEST_TOP + i * DEST_GAP;
        return (
          pop > 0.01 && (
            <div
              key={dest.url}
              style={{
                ...card,
                position: "absolute",
                left: DEST_LEFT,
                top: y,
                padding: "12px 20px",
                display: "flex",
                alignItems: "center",
                gap: 10,
                opacity: pop,
                transform: `translateX(${(1 - pop) * 24}px)`,
              }}
            >
              <span
                style={{
                  fontFamily: F.mono,
                  fontSize: 14,
                  fontWeight: 700,
                  color: "#fff",
                  background: C.tealDeep,
                  borderRadius: 6,
                  padding: "2px 8px",
                }}
              >
                {dest.method}
              </span>
              <span style={{ fontFamily: F.mono, fontSize: 17, color: C.ink }}>
                {dest.url}
              </span>
            </div>
          )
        );
      })}

      {/* c4: Dotted wires + comets from card to destinations */}
      <svg
        width={1920}
        height={1080}
        style={{ position: "absolute", inset: 0, pointerEvents: "none", overflow: "visible" }}
      >
        {DESTINATIONS.map((dest, i) => {
          const y = DEST_TOP + i * DEST_GAP + 24;
          const a = { x: CARD_LEFT + CARD_W, y: CARD_TOP + 220 };
          const b = { x: DEST_LEFT, y };
          const drawStart = c4 + 10 + i * 8;
          const drawEnd = c4 + 30 + i * 8;
          return (
            <React.Fragment key={dest.url}>
              <Wire
                id={`dest-${i}`}
                a={a}
                b={b}
                frame={frame}
                drawStart={drawStart}
                drawEnd={drawEnd}
                color="rgba(0,173,216,0.35)"
                width={2}
              />
              <Comet
                a={a}
                b={b}
                frame={frame}
                start={drawStart + 6}
                end={drawEnd + 4}
                color={C.coral}
                r={7}
              />
              <Ripple x={b.x} y={b.y} frame={frame} at={drawEnd + 4} color={C.teal} size={30} />
            </React.Fragment>
          );
        })}
      </svg>

      {/* SFX: seal stamp */}
      <Sequence from={c5 + 40}>
        <Audio src={staticFile("sfx/impact-bass-1.mp3")} volume={0.15} />
      </Sequence>
    </AbsoluteFill>
  );
};

import React from "react";
import {
  AbsoluteFill,
  Easing,
  interpolate,
  random,
  spring,
  useCurrentFrame,
  useVideoConfig,
} from "remotion";
import { C, F } from "../theme";
import { Kicker } from "../components/Kicker";

// Claims verbatim from README features / okf bundle. Each card carries a tiny
// illustrative demo; the signature header format is from
// internal/webhooks/client/request.go (Standard Webhooks: "v1," HMAC + "v1a," Ed25519).
const WIDE = "Embedded dashboard · consumer portal · OpenAPI-first";
const B64 = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";
const HEX = "0123456789abcdef";

const rnd = (seed: string, n: number, alphabet: string) =>
  Array.from({ length: n }, (_, i) => alphabet[Math.floor(random(`${seed}-${i}`) * alphabet.length)]).join("");

const useIn = (delay: number) => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();
  return spring({ frame: frame - delay, fps, config: { damping: 18, stiffness: 120 } });
};

const Card: React.FC<{ title: string; delay: number; children: React.ReactNode }> = ({
  title,
  delay,
  children,
}) => {
  const s = useIn(delay);
  return (
    <div
      style={{
        height: 218,
        padding: "30px 36px",
        display: "flex",
        flexDirection: "column",
        justifyContent: "space-between",
        background: "rgba(255,255,255,0.72)",
        border: "1px solid rgba(11,15,20,0.1)",
        borderRadius: 18,
        boxShadow: "0 14px 36px rgba(11,15,20,0.07)",
        fontFamily: F.display,
        color: C.ink,
        opacity: s,
        transform: `perspective(1400px) rotateX(${(1 - s) * 55}deg) translateY(${(1 - s) * 40}px)`,
        transformOrigin: "50% 100%",
      }}
    >
      <div style={{ fontSize: 36, fontWeight: 600, letterSpacing: "-0.02em" }}>{title}</div>
      <div style={{ fontFamily: F.mono, fontSize: 21 }}>{children}</div>
    </div>
  );
};

const Retry: React.FC<{ delay: number }> = ({ delay }) => {
  const frame = useCurrentFrame() - delay;
  const row = (at: number) =>
    interpolate(frame, [at, at + 8], [0, 1], { extrapolateLeft: "clamp", extrapolateRight: "clamp" });
  return (
    <div style={{ display: "flex", gap: 34 }}>
      <span style={{ opacity: row(14) }}>
        attempt 1 <span style={{ color: C.red, fontWeight: 700 }}>&#10005; 503</span>
      </span>
      <span style={{ opacity: row(14) * 0.5 }}>&#8594; backoff &#8594;</span>
      <span style={{ opacity: row(34), scale: String(0.9 + row(34) * 0.1) }}>
        attempt 2 <span style={{ color: C.tealDeep, fontWeight: 700 }}>&#10003; 200</span>
      </span>
    </div>
  );
};

const Signature: React.FC<{ delay: number }> = ({ delay }) => {
  const frame = useCurrentFrame() - delay;
  const hmac = rnd("hmac", 14, B64);
  const ed = rnd("ed", 14, B64);
  const full = `v1,${hmac}… v1a,${ed}…`;
  const n = Math.round(
    interpolate(frame, [12, 50], [0, full.length], { extrapolateLeft: "clamp", extrapolateRight: "clamp" }),
  );
  return (
    <div style={{ whiteSpace: "nowrap", overflow: "hidden" }}>
      <span style={{ opacity: 0.55 }}>webhook-signature: </span>
      <span style={{ color: C.tealDeep }}>{full.slice(0, n)}</span>
      <span style={{ opacity: n < full.length && Math.floor(frame / 6) % 2 === 0 ? 1 : 0 }}>&#9608;</span>
    </div>
  );
};

const Encrypt: React.FC<{ delay: number }> = ({ delay }) => {
  const frame = useCurrentFrame() - delay;
  const plain = "whsec_my-signing-secret";
  const cipher = rnd("cipher", plain.length, HEX);
  // Characters flip from plaintext to ciphertext left to right.
  const k = interpolate(frame, [20, 50], [0, plain.length], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
  });
  const text = plain
    .split("")
    .map((ch, i) => (i < k - 2 ? cipher[i] : i < k ? rnd(`s-${Math.floor(frame / 2)}-${i}`, 1, HEX) : ch))
    .join("");
  const locked = k >= plain.length;
  return (
    <div style={{ display: "flex", alignItems: "center", gap: 16 }}>
      <svg width={22} height={26} viewBox="0 0 22 26">
        <rect x={2} y={11} width={18} height={14} rx={3} fill={locked ? C.tealDeep : "rgba(11,15,20,0.4)"} />
        <path
          d={locked ? "M6 11 V7 a5 5 0 0 1 10 0 V11" : "M6 11 V7 a5 5 0 0 1 10 0 V5"}
          stroke={locked ? C.tealDeep : "rgba(11,15,20,0.4)"}
          strokeWidth={2.6}
          fill="none"
        />
      </svg>
      <span style={{ color: locked ? C.tealDeep : C.ink }}>{text}</span>
    </div>
  );
};

const Health: React.FC<{ delay: number }> = ({ delay }) => {
  const frame = useCurrentFrame() - delay;
  const bars = 26;
  return (
    <div style={{ display: "flex", alignItems: "flex-end", gap: 20 }}>
      <div style={{ display: "flex", alignItems: "flex-end", gap: 5, height: 40 }}>
        {Array.from({ length: bars }, (_, i) => {
          const h = 14 + random(`h-${i}`) * 26;
          const grow = interpolate(frame, [10 + i * 1.2, 18 + i * 1.2], [0, 1], {
            extrapolateLeft: "clamp",
            extrapolateRight: "clamp",
            easing: Easing.out(Easing.cubic),
          });
          const bad = i === 7 || i === 8;
          return (
            <div
              key={i}
              style={{
                width: 9,
                height: h * grow,
                borderRadius: 2,
                background: bad ? C.coral : C.tealDeep,
                opacity: 0.85,
              }}
            />
          );
        })}
      </div>
      <span
        style={{
          display: "inline-flex",
          alignItems: "center",
          gap: 8,
          padding: "4px 14px",
          borderRadius: 999,
          background: "rgba(14,116,144,0.12)",
          color: C.tealDeep,
          opacity: interpolate(frame, [44, 52], [0, 1], { extrapolateLeft: "clamp", extrapolateRight: "clamp" }),
        }}
      >
        <span
          style={{
            width: 10,
            height: 10,
            borderRadius: "50%",
            background: C.tealDeep,
            boxShadow: `0 0 0 ${3 + Math.sin(frame / 4) * 3}px rgba(14,116,144,0.2)`,
          }}
        />
        healthy
      </span>
    </div>
  );
};

export const Features: React.FC = () => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();
  const wide = useIn(90);
  const tagIn = spring({ frame: frame - 110, fps, config: { damping: 14, stiffness: 160 } });

  return (
    <AbsoluteFill>
      <Kicker text="INCLUDED, NOT EXTRA" />

      <AbsoluteFill style={{ justifyContent: "center", alignItems: "center", paddingTop: 30 }}>
        <div style={{ width: 1600 }}>
          <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 28 }}>
            <Card title="At-least-once delivery" delay={14}>
              <Retry delay={14} />
            </Card>
            <Card title="HMAC + Ed25519 signing" delay={30}>
              <Signature delay={30} />
            </Card>
            <Card title="Encryption at rest (AES-256-GCM)" delay={46}>
              <Encrypt delay={46} />
            </Card>
            <Card title="Per-webhook health tracking" delay={62}>
              <Health delay={62} />
            </Card>
          </div>
          <div style={{ display: "flex", justifyContent: "center", marginTop: 28 }}>
            <div
              style={{
                position: "relative",
                height: 120,
                padding: "0 70px",
                display: "flex",
                alignItems: "center",
                whiteSpace: "nowrap",
                background: C.navy,
                color: C.cream,
                borderRadius: 18,
                fontFamily: F.display,
                fontSize: 34,
                fontWeight: 500,
                boxShadow: "0 18px 40px rgba(11,15,20,0.2)",
                opacity: wide,
                translate: `0px ${(1 - wide) * 30}px`,
              }}
            >
              {WIDE}
              <div
                style={{
                  position: "absolute",
                  top: -20,
                  right: 26,
                  background: C.coral,
                  color: C.cream,
                  fontFamily: F.display,
                  fontSize: 24,
                  fontWeight: 600,
                  padding: "10px 20px",
                  borderRadius: 999,
                  boxShadow: "0 8px 20px rgba(249,115,22,0.4)",
                  opacity: tagIn,
                  scale: String(0.6 + tagIn * 0.4),
                  rotate: `${(1 - tagIn) * -12 + 3}deg`,
                }}
              >
                all built in
              </div>
            </div>
          </div>
        </div>
      </AbsoluteFill>
    </AbsoluteFill>
  );
};

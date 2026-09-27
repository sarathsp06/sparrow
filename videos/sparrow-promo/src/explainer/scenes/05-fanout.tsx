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
import { cueFrame, useReveal, ease } from "../timing";
import { StepRail, StepTitle, card } from "../ui";
import { Comet, Ripple, Wire } from "../../components/Wire";

const ID = "05-fanout";

const SUBS = [
  { name: "billing-svc", event: "order.created", labels: "", match: true },
  { name: "partner-acme", event: "order.created", labels: "", match: true },
  { name: "slack-ops", event: "order.created", labels: "", match: true },
  { name: "analytics", event: "order.created", labels: "env=staging", match: false },
];

// Layout
const PILL_X = 240;
const PILL_Y = 480;
const CARD_X = 960;
const CARD_W = 440;
const CARD_H = 90;
const CARD_GAP = 16;
const CARD_TOP = 340;

const cardY = (i: number) => CARD_TOP + i * (CARD_H + CARD_GAP);
const cardCenterY = (i: number) => cardY(i) + CARD_H / 2;

// Filter chip labels
const FILTERS = [
  { label: "event ✓", color: C.teal },
  { label: "consumer: acme ✓", color: C.teal },
  { label: "labels env=prod", color: C.coral },
];

export const FanOut: React.FC = () => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();
  const r = useReveal();

  const c0 = cueFrame(ID, 0);
  const c1 = cueFrame(ID, 1);
  const c2 = cueFrame(ID, 2);
  const c3 = cueFrame(ID, 3);

  // Grey-out the analytics card (env=staging mismatch)
  const greyOut = ease(frame, c2 + 50, c2 + 68);

  // Wire draw and comets (c3)
  const wireDrawStart = c3 + 4;
  const wireDrawEnd = c3 + 28;

  // Queue strip
  const queuePop = spring({
    frame: frame - (c3 + 36),
    fps,
    config: { damping: 200 },
  });

  return (
    <AbsoluteFill>
      <StepRail active={1} />
      <StepTitle n={2} title="Fan-out" at={c0} />

      {/* Event pill with label chip */}
      <div
        style={{
          position: "absolute",
          left: PILL_X - 80,
          top: PILL_Y - 55,
          opacity: r(c1),
          transform: `translateX(${(1 - r(c1)) * -20}px)`,
        }}
      >
        <div
          style={{
            ...card,
            display: "inline-flex",
            alignItems: "center",
            gap: 14,
            padding: "18px 28px",
            borderColor: C.coral,
            borderWidth: 2,
          }}
        >
          <div
            style={{
              width: 14,
              height: 14,
              borderRadius: 99,
              background: C.coral,
              boxShadow: `0 0 10px rgba(249,115,22,0.5)`,
            }}
          />
          <span style={{ fontFamily: F.mono, fontSize: 24, fontWeight: 700, color: C.ink }}>
            order.created
          </span>
        </div>
        {/* Label chip */}
        <div
          style={{
            display: "inline-flex",
            marginLeft: 14,
            alignItems: "center",
            gap: 6,
            fontFamily: F.mono,
            fontSize: 17,
            color: C.coralText,
            background: "rgba(249,115,22,0.10)",
            border: `1.5px solid rgba(249,115,22,0.4)`,
            borderRadius: 999,
            padding: "8px 16px",
            opacity: r(c1 + 6),
          }}
        >
          env=prod
        </div>
      </div>

      {/* Filter chips — reveal sequentially at c2 */}
      <div
        style={{
          position: "absolute",
          left: PILL_X - 80,
          top: PILL_Y + 30,
          display: "flex",
          gap: 12,
        }}
      >
        {FILTERS.map((f, i) => {
          const at = c2 + i * 20;
          const s = r(at);
          return (
            <div
              key={f.label}
              style={{
                fontFamily: F.mono,
                fontSize: 18,
                color: f.color,
                background:
                  f.color === C.coral
                    ? "rgba(249,115,22,0.10)"
                    : "rgba(0,173,216,0.10)",
                border: `1.5px solid ${f.color}`,
                borderRadius: 999,
                padding: "7px 16px",
                opacity: s,
                transform: `translateY(${(1 - s) * 10}px)`,
              }}
            >
              {f.label}
            </div>
          );
        })}
      </div>

      {/* Wires SVG layer */}
      <svg
        width={1920}
        height={1080}
        style={{ position: "absolute", inset: 0, pointerEvents: "none", overflow: "visible" }}
      >
        {SUBS.map((sub, i) => {
          if (!sub.match) return null;
          const a = { x: PILL_X + 180, y: PILL_Y };
          const b = { x: CARD_X, y: cardCenterY(i) };
          return (
            <React.Fragment key={sub.name}>
              <Wire
                id={`fan-${i}`}
                a={a}
                b={b}
                frame={frame}
                drawStart={wireDrawStart + i * 4}
                drawEnd={wireDrawEnd + i * 4}
                color="rgba(0,173,216,0.45)"
                width={2.5}
              />
              <Comet
                a={a}
                b={b}
                frame={frame}
                start={wireDrawStart + 8 + i * 6}
                end={wireDrawEnd + 12 + i * 6}
                color={C.coral}
                r={8}
              />
              <Ripple
                x={CARD_X}
                y={cardCenterY(i)}
                frame={frame}
                at={wireDrawEnd + 12 + i * 6}
                color={C.teal}
                size={40}
              />
            </React.Fragment>
          );
        })}
      </svg>

      {/* Subscription cards */}
      {SUBS.map((sub, i) => {
        const cardPop = r(c1 + 6 + i * 6);
        const isGrey = !sub.match && greyOut > 0;
        return (
          <div
            key={sub.name}
            style={{
              ...card,
              position: "absolute",
              left: CARD_X,
              top: cardY(i),
              width: CARD_W,
              height: CARD_H,
              display: "flex",
              alignItems: "center",
              padding: "0 24px",
              gap: 14,
              opacity: cardPop * (isGrey ? 1 - greyOut * 0.6 : 1),
              transform: `translateX(${(1 - cardPop) * 30}px)`,
              filter: isGrey ? `grayscale(${greyOut * 0.8})` : "none",
            }}
          >
            <div style={{ flex: 1 }}>
              <div
                style={{
                  fontFamily: F.display,
                  fontSize: 22,
                  fontWeight: 600,
                  color: C.ink,
                }}
              >
                {sub.name}
              </div>
              <div
                style={{
                  fontFamily: F.mono,
                  fontSize: 16,
                  color: "rgba(11,15,20,0.55)",
                  marginTop: 2,
                }}
              >
                {sub.event}
                {sub.labels ? ` · ${sub.labels}` : ""}
              </div>
            </div>
            {/* Match/mismatch indicator */}
            {!sub.match && greyOut > 0 && (
              <div
                style={{
                  width: 30,
                  height: 30,
                  borderRadius: 99,
                  background: C.red,
                  color: "#fff",
                  display: "grid",
                  placeItems: "center",
                  fontSize: 18,
                  fontWeight: 700,
                  opacity: greyOut,
                }}
              >
                {"✕"}
              </div>
            )}
          </div>
        );
      })}

      {/* Queue strip — "river · postgres" */}
      {queuePop > 0.01 && (
        <div
          style={{
            position: "absolute",
            left: CARD_X - 30,
            top: cardY(SUBS.length - 1) + CARD_H + 40,
            opacity: queuePop,
            transform: `translateY(${(1 - queuePop) * 14}px)`,
          }}
        >
          <div
            style={{
              display: "flex",
              alignItems: "center",
              gap: 12,
              fontFamily: F.mono,
              fontSize: 16,
              color: C.tealDeep,
              letterSpacing: "0.06em",
              marginBottom: 10,
            }}
          >
            <svg width={18} height={18} viewBox="0 0 18 18">
              <rect x={1} y={1} width={16} height={16} rx={3} fill={C.tealDeep} opacity={0.2} />
              <rect x={4} y={4} width={10} height={10} rx={2} fill={C.tealDeep} opacity={0.4} />
            </svg>
            river {"·"} postgres
          </div>
          <div style={{ display: "flex", gap: 10 }}>
            {[1, 2, 3].map((n) => {
              const tileAt = c3 + 40 + n * 6;
              const tilePop = spring({
                frame: frame - tileAt,
                fps,
                config: { damping: 200 },
              });
              return (
                <div
                  key={n}
                  style={{
                    ...card,
                    display: "inline-flex",
                    alignItems: "center",
                    gap: 8,
                    padding: "10px 18px",
                    fontFamily: F.mono,
                    fontSize: 17,
                    color: C.tealDeep,
                    opacity: tilePop,
                    transform: `scale(${0.85 + tilePop * 0.15})`,
                  }}
                >
                  <div
                    style={{
                      width: 10,
                      height: 10,
                      borderRadius: 99,
                      background: C.teal,
                    }}
                  />
                  delivery #{n}
                </div>
              );
            })}
          </div>
          {/* "no broker · no Redis" label */}
          <div
            style={{
              fontFamily: F.mono,
              fontSize: 15,
              color: "rgba(11,15,20,0.4)",
              marginTop: 10,
              letterSpacing: "0.04em",
              opacity: ease(frame, c3 + 66, c3 + 80),
            }}
          >
            no broker {"·"} no Redis
          </div>
        </div>
      )}

      {/* SFX: chime on fan-out */}
      <Sequence from={c3 + 12}>
        <Audio src={staticFile("sfx/chime.mp3")} volume={0.12} />
      </Sequence>
    </AbsoluteFill>
  );
};

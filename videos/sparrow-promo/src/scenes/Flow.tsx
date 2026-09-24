import React from "react";
import {
  AbsoluteFill,
  Easing,
  Img,
  interpolate,
  spring,
  staticFile,
  useCurrentFrame,
  useVideoConfig,
} from "remotion";
import { C, F } from "../theme";
import { Kicker } from "../components/Kicker";
import { Comet, Ripple, Wire } from "../components/Wire";

// Architecture centerpiece, following the repo's own explanation
// (docs/src/assets/diagrams/sparrow-layered-architecture.archify.json):
//   1 · Subscribe — consumers register webhooks + event subscriptions (internal/rest/webhook.go)
//   2 · Push — your services POST events (internal/rest/event.go)
//   3 · Deliver — River workers sign, POST, retry, track health (internal/webhooks/queue/webhook_worker.go)
// Story: your microservice ecosystem exists → Sparrow drops in → consumers can listen.

const SERVICES = [
  { name: "orders-api", big: true },
  { name: "billing-api", big: false },
  { name: "auth-api", big: false },
];

const RECEIVERS = [
  { name: "partner API", icon: "icons/webhook.svg", status: "webhook \u00B7 \u2713 signed", at: 200 },
  { name: "internal service", icon: "icons/webhook.svg", status: "webhook \u00B7 \u2713 delivered", at: 218 },
  { name: "Slack #ops", icon: "icons/slack.svg", status: "via recipe", at: 236 },
  { name: "email", icon: "icons/sendgrid.svg", status: "via recipe", at: 254 },
  { name: "SMS", icon: "icons/twilio.svg", status: "via recipe", at: 272 },
  { name: "push", icon: "icons/ntfy.svg", status: "via recipe", at: 290 },
];

// Layout constants (absolute px on 1920x1080).
const LEFT_X = 110;
const LEFT_W = 400;
const MID = { x: 745, y: 340, w: 430, h: 360 };
const RIGHT_X = 1400;
const RIGHT_W = 420;
const RIGHT_H = 96;
const RIGHT_GAP = 18;
const RIGHT_TOP = 540 - (RECEIVERS.length * RIGHT_H + (RECEIVERS.length - 1) * RIGHT_GAP) / 2;

// Your-services column: orders-api tall (with POST pill), two small siblings.
const SVC_TOPS = [300, 530, 640];
const SVC_HEIGHTS = [200, 90, 90];

const card: React.CSSProperties = {
  position: "absolute",
  background: "rgba(255,255,255,0.6)",
  border: "1px solid rgba(11,15,20,0.12)",
  borderRadius: 16,
  boxShadow: "0 12px 34px rgba(11,15,20,0.07)",
  fontFamily: F.display,
  color: C.ink,
};

const StepBadge: React.FC<{
  x: number;
  y: number;
  color: string;
  label: string;
  in_: number;
}> = ({ x, y, color, label, in_ }) => (
  <div
    style={{
      position: "absolute",
      left: x,
      top: y,
      fontFamily: F.mono,
      fontSize: 20,
      letterSpacing: "0.04em",
      color,
      background: "rgba(255,255,255,0.85)",
      border: `1.5px solid ${color}`,
      borderRadius: 999,
      padding: "8px 18px",
      opacity: in_,
      transform: `translateY(${(1 - in_) * 12}px)`,
    }}
  >
    {label}
  </div>
);

export const Flow: React.FC = () => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();

  const pop = (delay: number) => {
    const s = spring({ frame: frame - delay, fps, config: { damping: 200 } });
    return { opacity: s, transform: `translateY(${(1 - s) * 24}px)` };
  };
  const badge = (delay: number) =>
    spring({ frame: frame - delay, fps, config: { damping: 200 } });

  // Sparrow drops into the ecosystem from above.
  const drop = spring({ frame: frame - 45, fps, config: { damping: 16, mass: 0.9 } });
  const dropOpacity = interpolate(frame, [45, 55], [0, 1], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
  });

  const ordersOut = { x: LEFT_X + LEFT_W, y: SVC_TOPS[0] + SVC_HEIGHTS[0] / 2 };
  const midIn = { x: MID.x, y: MID.y + MID.h / 2 };
  const midOut = { x: MID.x + MID.w, y: MID.y + MID.h / 2 };
  const ry = (i: number) => RIGHT_TOP + i * (RIGHT_H + RIGHT_GAP) + RIGHT_H / 2;

  // Sparrow "processing" glow while the event is inside.
  const busy = interpolate(frame, [165, 180, 200, 220], [0, 1, 1, 0], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
  });
  const caption = spring({ frame: frame - 320, fps, config: { damping: 200 } });

  return (
    <AbsoluteFill>
      <Kicker text="DROP IT INTO YOUR STACK" />

      {/* Camera: slow push-in over the whole beat */}
      <AbsoluteFill style={{ scale: String(interpolate(frame, [0, 450], [1, 1.045])) }}>

      <div
        style={{
          position: "absolute",
          top: 262,
          left: 108,
          fontFamily: F.mono,
          fontSize: 18,
          letterSpacing: "0.14em",
          color: C.ink,
          opacity: 0.55 * badge(58),
        }}
      >
        YOUR INFRASTRUCTURE
      </div>

      {/* Connectors + pulses */}
      <svg
        width={1920}
        height={1080}
        style={{ position: "absolute", inset: 0, pointerEvents: "none", overflow: "visible" }}
      >
        {/* Environment boundary: your services + Sparrow live inside your infra */}
        <rect
          x={80}
          y={245}
          width={1135}
          height={545}
          rx={24}
          fill="rgba(255,255,255,0.18)"
          stroke="rgba(11,15,20,0.22)"
          strokeWidth={2}
          strokeDasharray="10 10"
          strokeDashoffset={frame * 0.4}
          opacity={badge(58)}
        />
        {/* Landing shockwave when Sparrow drops in */}
        <Ripple x={MID.x + MID.w / 2} y={MID.y + MID.h / 2} frame={frame} at={58} size={420} color={C.teal} />

        {/* 1 · subscribe: receivers → Sparrow (teal) */}
        {RECEIVERS.map((r, i) => (
          <Wire
            key={`sub-${r.name}`}
            id={`sub-${i}`}
            a={{ x: RIGHT_X, y: ry(i) }}
            b={midOut}
            frame={frame}
            drawStart={95 + i * 3}
            drawEnd={120 + i * 3}
            color="rgba(0,173,216,0.5)"
          />
        ))}
        {RECEIVERS.map((r, i) => (
          <Comet
            key={`subp-${r.name}`}
            a={{ x: RIGHT_X, y: ry(i) }}
            b={midOut}
            start={100 + i * 4}
            end={126 + i * 4}
            frame={frame}
            color={C.teal}
            r={7}
          />
        ))}

        {/* 2 · push: orders-api → Sparrow (coral) */}
        <Wire id="push" a={ordersOut} b={midIn} frame={frame} drawStart={140} drawEnd={158} />
        {[160, 350].map((s) => (
          <Comet key={s} a={ordersOut} b={midIn} start={s} end={s + 30} frame={frame} r={11} />
        ))}

        {/* 3 · deliver: Sparrow → receivers, reuses the subscribe corridor */}
        {RECEIVERS.map((r, i) =>
          [r.at - 22, r.at + 190].map((s) => (
            <React.Fragment key={`${r.name}-${s}`}>
              <Comet a={midOut} b={{ x: RIGHT_X, y: ry(i) }} start={s} end={s + 22} frame={frame} />
              <Ripple x={RIGHT_X} y={ry(i)} frame={frame} at={s + 22} color={C.coral} size={46} />
            </React.Fragment>
          )),
        )}
      </svg>

      {/* Your ecosystem of services */}
      {SERVICES.map((svc, i) => (
        <div
          key={svc.name}
          style={{
            ...card,
            left: LEFT_X,
            top: SVC_TOPS[i],
            width: LEFT_W,
            height: SVC_HEIGHTS[i],
            ...pop(6 + i * 7),
          }}
        >
          {svc.big ? (
            <div style={{ padding: "24px 30px" }}>
              <div style={{ fontFamily: F.mono, fontSize: 19, opacity: 0.55 }}>
                your services
              </div>
              <div style={{ fontSize: 32, fontWeight: 500, marginTop: 4 }}>{svc.name}</div>
              <div
                style={{
                  marginTop: 16,
                  display: "inline-block",
                  fontFamily: F.mono,
                  fontSize: 17,
                  background: C.navy,
                  color: C.cream,
                  padding: "10px 14px",
                  borderRadius: 8,
                  opacity: interpolate(frame, [130, 144], [0, 1], {
                    extrapolateLeft: "clamp",
                    extrapolateRight: "clamp",
                  }),
                }}
              >
                POST /v1/consumers/acme/events
              </div>
            </div>
          ) : (
            <div
              style={{
                padding: "0 30px",
                height: "100%",
                display: "flex",
                alignItems: "center",
                fontSize: 28,
                fontWeight: 500,
              }}
            >
              {svc.name}
            </div>
          )}
        </div>
      ))}

      {/* Sparrow drops in */}
      <div
        style={{
          ...card,
          left: MID.x,
          top: MID.y,
          width: MID.w,
          height: MID.h,
          border: `2px solid rgba(0,173,216,${0.35 + busy * 0.65})`,
          boxShadow: `0 12px 34px rgba(11,15,20,0.07), 0 0 ${40 * busy}px rgba(0,173,216,${
            0.35 * busy
          })`,
          background: "rgba(255,255,255,0.75)",
          opacity: dropOpacity,
          translate: `0px ${(1 - drop) * -420}px`,
          rotate: `${(1 - drop) * -4}deg`,
        }}
      >
        <div style={{ padding: "30px 34px", textAlign: "center" }}>
          <Img src={staticFile("sparrow-logo.svg")} style={{ width: 90, height: 72 }} />
          <div style={{ fontSize: 52, fontWeight: 600, letterSpacing: "-0.02em" }}>Sparrow</div>
          <div style={{ fontFamily: F.mono, fontSize: 18, opacity: 0.6, marginTop: 4 }}>
            webhook delivery service
          </div>
          <div
            style={{
              display: "flex",
              flexWrap: "wrap",
              justifyContent: "center",
              gap: 10,
              marginTop: 24,
            }}
          >
            {["queue", "retries", "signing", "health"].map((chip) => (
              <div
                key={chip}
                style={{
                  fontFamily: F.mono,
                  fontSize: 17,
                  padding: "8px 16px",
                  borderRadius: 999,
                  border: "1px solid rgba(11,15,20,0.18)",
                  background: `rgba(0,173,216,${0.08 + busy * 0.16})`,
                }}
              >
                {chip}
              </div>
            ))}
          </div>
          <div
            style={{
              marginTop: 22,
              fontFamily: F.mono,
              fontSize: 17,
              color: C.coralText,
              opacity: interpolate(frame, [186, 198], [0, 1], {
                extrapolateLeft: "clamp",
                extrapolateRight: "clamp",
              }),
            }}
          >
            order.created &#8594; fan-out &#215;{RECEIVERS.length}
          </div>
        </div>
      </div>

      {/* Consumers */}
      {RECEIVERS.map((r, i) => {
        const arrived = spring({ frame: frame - r.at, fps, config: { damping: 200 } });
        return (
          <div
            key={r.name}
            style={{
              ...card,
              left: RIGHT_X,
              top: RIGHT_TOP + i * (RIGHT_H + RIGHT_GAP),
              width: RIGHT_W,
              height: RIGHT_H,
              display: "flex",
              flexDirection: "column",
              justifyContent: "center",
              padding: "0 28px",
              ...pop(70 + i * 6),
            }}
          >
            <div
              style={{
                display: "flex",
                alignItems: "center",
                gap: 12,
                fontSize: 26,
                fontWeight: 500,
              }}
            >
              <Img src={staticFile(r.icon)} style={{ width: 26, height: 26 }} />
              {r.name}
            </div>
            <div
              style={{
                fontFamily: F.mono,
                fontSize: 17,
                marginTop: 6,
                color: r.status.includes("\u2713") ? "#0E7490" : C.coralText,
                opacity: arrived,
              }}
            >
              {r.status}
            </div>
            <div
              style={{
                position: "absolute",
                right: 22,
                top: RIGHT_H / 2 - 17,
                width: 34,
                height: 34,
                borderRadius: "50%",
                background: r.status.includes("\u2713") ? C.tealDeep : C.coral,
                color: "white",
                display: "flex",
                alignItems: "center",
                justifyContent: "center",
                fontSize: 19,
                fontWeight: 700,
                scale: String(
                  interpolate(frame, [r.at, r.at + 8], [0, 1], {
                    extrapolateLeft: "clamp",
                    extrapolateRight: "clamp",
                    easing: Easing.back(2.2),
                  }),
                ),
              }}
            >
              &#10003;
            </div>
          </div>
        );
      })}

      {/* Numbered steps — the repo's own architecture story */}
      <div
        style={{
          position: "absolute",
          top: RIGHT_TOP - 44,
          left: RIGHT_X + 2,
          fontFamily: F.mono,
          fontSize: 19,
          opacity: 0.55 * badge(70),
          color: C.ink,
        }}
      >
        destinations
      </div>
      <StepBadge x={1248} y={200} color="#0E7490" label={"1 \u00B7 subscribe"} in_={badge(100)} />
      <StepBadge x={560} y={330} color={C.coralText} label={"2 \u00B7 push"} in_={badge(150)} />
      <StepBadge x={1248} y={820} color="#0B0F14" label={"3 \u00B7 deliver"} in_={badge(195)} />

      </AbsoluteFill>

      <div
        style={{
          position: "absolute",
          bottom: 74,
          width: "100%",
          textAlign: "center",
          fontFamily: F.display,
          color: C.ink,
          opacity: caption,
          translate: `0px ${(1 - caption) * 18}px`,
          filter: `blur(${(1 - caption) * 6}px)`,
        }}
      >
        <div style={{ fontSize: 42, fontWeight: 500 }}>
          Register each destination once. Push events. Sparrow handles delivery.
        </div>
        <div style={{ fontFamily: F.mono, fontSize: 22, marginTop: 12, opacity: 0.6 }}>
          webhooks &#183; Slack &#183; email &#183; SMS &#183; push
        </div>
      </div>
    </AbsoluteFill>
  );
};

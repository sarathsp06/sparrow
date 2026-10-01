import React from "react";
import { AbsoluteFill, Img, Sequence, staticFile, useCurrentFrame } from "remotion";
import { Audio } from "@remotion/media";
import { C, F } from "../../theme";
import { cueFrame, wordFrame, useReveal, ease } from "../timing";
import { card, Chip, NightKicker, Terminal, TypeLine } from "../ui";
import { Wire, Comet, Ripple } from "../../components/Wire";

// Frame 8 — Shape the request → any HTTP API → recipes.
// Act 1 (c0–c2): one outgoing request is shaped: a Go template rewrites the
// body, your own headers + any method, so it can call any HTTP API.
// Act 2 (c3–c4): recipes are exactly those templates, pre-built per destination.

const ID = "08-recipes";
const c = (i: number) => cueFrame(ID, i);

const ENVELOPE = [
  ['"event_name"', '"item.shipped"'],
  ['"event_id"', '"evt_9f2…"'],
  ['"attempt"', "1"],
  ['"payload"', '{ "parcel": "pkg_88a1", "weight_kg": 2.4 }'],
];
const TEMPLATE = [
  "{",
  '  "summary": "{{ .payload.parcel }} shipped",',
  '  "weight_kg": {{ .payload.weight_kg }},',
  '  "source": "{{ .event_name }}"',
  "}",
];
const RENDERED = [
  ['"summary"', '"pkg_88a1 shipped"'],
  ['"weight_kg"', "2.4"],
  ['"source"', '"item.shipped"'],
];
const METHODS = ["POST", "PUT", "PATCH", "DELETE"];
const DESTS = [
  { m: "POST", url: "hooks.partner.example/in", y: 250 },
  { m: "PUT", url: "api.crm.example/orders/ord_123", y: 380 },
  { m: "PATCH", url: "billing.internal/invoices/inv_77", y: 510 },
];

const RECIPES = [
  { label: "PagerDuty", icon: "icons/pagerduty.svg", fmt: "Events API v2", at: wordFrame(ID, 3, "pagerduty") },
  { label: "Twilio", icon: "icons/twilio.svg", fmt: "Messages", at: wordFrame(ID, 3, "twilio") },
  { label: "SendGrid", icon: "icons/sendgrid.svg", fmt: "Mail Send v3", at: wordFrame(ID, 3, "sendgrid") },
  { label: "ClickHouse", icon: "icons/clickhouse.svg", fmt: "JSONEachRow", at: wordFrame(ID, 3, "clickhouse") },
  { label: "Discord", icon: "icons/discord.svg", fmt: "embeds", at: wordFrame(ID, 3, "more") },
  { label: "ntfy", icon: "icons/ntfy.svg", fmt: "topic", at: wordFrame(ID, 3, "more") + 4 },
  { label: "Slack", icon: "icons/slack.svg", fmt: "Block Kit", at: wordFrame(ID, 3, "more") + 8 },
];
const GRID = { x: 120, y: 190, w: 400, h: 110, gx: 22, gy: 22 };
const tilePos = (i: number) => {
  const row = i < 4 ? 0 : 1;
  const col = i < 4 ? i : i - 4;
  const off = row === 1 ? (GRID.w + GRID.gx) / 2 : 0;
  return { x: GRID.x + col * (GRID.w + GRID.gx) + off, y: GRID.y + row * (GRID.h + GRID.gy) };
};
const YAML = [
  ["webhook:", ""],
  ["  url:", " https://events.pagerduty.com/v2/enqueue"],
  ["  headers:", ""],
  ["    Authorization:", " Token token=${PAGERDUTY_KEY}"],
  ["subscription:", ""],
  ["  transform_template:", " |"],
  ["", '    "event_action": "trigger",'],
  ["", '    "dedup_key": {{ .event_id | json }},'],
  ["", '    "payload": { "summary": "{{ .payload.parcel }} shipped" }'],
];

export const Recipes: React.FC = () => {
  const frame = useCurrentFrame();
  const r = useReveal();

  // ── act 1 anchors ──
  const shapeAt = wordFrame(ID, 0, "shape");
  const templateAt = wordFrame(ID, 0, "template");
  const expectsAt = wordFrame(ID, 0, "whatever");
  const headersAt = wordFrame(ID, 1, "headers");
  const tokenAt = wordFrame(ID, 1, "token");
  const encAt = wordFrame(ID, 1, "encrypted");
  const methodAts = METHODS.map((m) => wordFrame(ID, 1, m.toLowerCase()));
  const anyAt = wordFrame(ID, 2, "almost");
  const notJustAt = wordFrame(ID, 2, "not");
  // ── act 2 anchors ──
  const recipesAt = c(3);
  const prebuiltAt = wordFrame(ID, 3, "pre-built");
  const yamlAt = c(3) + 4;
  const cmdAt = wordFrame(ID, 4, "sparrow");

  const cardIn = r(c(0));
  const envIn = ENVELOPE.map((_, i) => ease(frame, shapeAt + i * 4, shapeAt + 8 + i * 4));
  const tplIn = r(templateAt);
  const morph = ease(frame, expectsAt, expectsAt + 14);
  const hdrIn = r(headersAt);
  const tokIn = r(tokenAt);
  const encIn = r(encAt);
  const methodIdx = methodAts.reduce((acc, at, i) => (frame >= at ? i : acc), -1);
  const method = methodIdx < 0 ? "POST" : frame > methodAts[3] + 30 ? "PUT" : METHODS[methodIdx];
  const destIn = DESTS.map((_, i) => r(anyAt + i * 6));
  const notJust = r(notJustAt);

  // act 1 → act 2: request stage recedes, recipe stage lands
  const out = ease(frame, recipesAt - 6, recipesAt + 14);
  const act1 = 1 - out;
  const tplKeep = 1 - ease(frame, anyAt - 4, anyAt + 10); // template card leaves when endpoints arrive
  const yamlIn = r(yamlAt);
  const termIn = r(c(4));
  const pd = tilePos(0);
  const pdCenter = { x: pd.x + GRID.w / 2, y: pd.y + GRID.h / 2 };
  const termCenter = { x: 120 + 380, y: 700 };
  const hi = ease(frame, cmdAt + 24, cmdAt + 36);

  const CARD = { x: 120, y: 170, w: 780 };
  const reqRight = { x: CARD.x + CARD.w, y: CARD.y + 260 };

  return (
    <AbsoluteFill>
      <NightKicker label={out < 0.5 ? "SHAPE THE REQUEST · ANY HTTP API" : "RECIPES · ONE YAML EACH"} at={c(0)} />

      {/* ── Act 1: the request card ── */}
      <div
        style={{
          ...card,
          position: "absolute",
          left: CARD.x,
          top: CARD.y,
          width: CARD.w,
          padding: 0,
          overflow: "hidden",
          opacity: cardIn * act1,
          transform: `translateY(${(1 - cardIn) * 16}px) scale(${1 - out * 0.1})`,
          transformOrigin: "left top",
        }}
      >
        <div style={{ padding: "18px 26px", borderBottom: "1px solid rgba(11,15,20,0.08)", display: "flex", alignItems: "center", gap: 14, fontFamily: F.mono }}>
          <span style={{ color: "#fff", background: C.tealDeep, borderRadius: 8, padding: "6px 16px", fontSize: 26, fontWeight: 700, minWidth: 96, textAlign: "center" }}>{method}</span>
          <span style={{ fontSize: 22, color: C.ink }}>{methodIdx >= 1 ? "api.crm.example/orders/ord_123" : "hooks.partner.example/in"}</span>
        </div>
        <div style={{ padding: "14px 26px", borderBottom: "1px solid rgba(11,15,20,0.08)", fontFamily: F.mono, fontSize: 22, lineHeight: 1.65 }}>
          <div style={{ fontSize: 15, letterSpacing: "0.12em", color: "rgba(11,15,20,0.45)" }}>HEADERS</div>
          <div style={{ opacity: hdrIn }}><span style={{ color: C.tealDeep }}>content-type</span>: application/json</div>
          <div style={{ opacity: hdrIn }}><span style={{ color: C.tealDeep }}>x-tenant</span>: acme</div>
          <div style={{ opacity: tokIn, display: "flex", alignItems: "center", gap: 10 }}>
            <span><span style={{ color: C.tealDeep }}>authorization</span>: Bearer ••••••••••</span>
            <svg width={18} height={22} viewBox="0 0 16 20" style={{ opacity: encIn }}>
              <rect x={1} y={9} width={14} height={10} rx={2.5} fill={C.tealDeep} />
              <path d="M4 9 V6 a4 4 0 0 1 8 0 V9" stroke={C.tealDeep} strokeWidth={2} fill="none" />
            </svg>
            <span style={{ opacity: encIn }}><Chip tone="teal" size={15}>encrypted at rest · AES-256-GCM</Chip></span>
          </div>
        </div>
        <div style={{ padding: "14px 26px 20px", fontFamily: F.mono, fontSize: 22, lineHeight: 1.6 }}>
          <div style={{ fontSize: 15, letterSpacing: "0.12em", color: "rgba(11,15,20,0.45)", display: "flex", gap: 12, alignItems: "center" }}>
            BODY
            <span style={{ opacity: morph }}><Chip tone="coral" size={14}>rendered by your template</Chip></span>
          </div>
          {morph < 0.5 ? (
            <div>
              {"{"}
              {ENVELOPE.map(([k, v], i) => (
                <div key={k} style={{ paddingLeft: 22, opacity: envIn[i] }}>
                  <span style={{ color: C.tealDeep }}>{k}</span>: {v}{i < ENVELOPE.length - 1 ? "," : ""}
                </div>
              ))}
              {"}"}
            </div>
          ) : (
            <div style={{ opacity: morph }}>
              {"{"}
              {RENDERED.map(([k, v], i) => (
                <div key={k} style={{ paddingLeft: 22 }}>
                  <span style={{ color: C.coralText }}>{k}</span>: {v}{i < RENDERED.length - 1 ? "," : ""}
                </div>
              ))}
              {"}"}
            </div>
          )}
        </div>
      </div>

      {/* Go template card (right of the request) */}
      <div
        style={{
          position: "absolute",
          left: 960,
          top: 330,
          width: 840,
          background: C.navy,
          color: C.cream,
          borderRadius: 18,
          padding: "18px 26px",
          fontFamily: F.mono,
          fontSize: 22,
          lineHeight: 1.6,
          boxShadow: "0 22px 56px rgba(11,15,20,0.25)",
          opacity: tplIn * tplKeep * act1,
          transform: `translateX(${(1 - tplIn) * 24}px)`,
        }}
      >
        <div style={{ color: C.teal, fontSize: 17, marginBottom: 8 }}>subscription.transform_template · Go template</div>
        {TEMPLATE.map((l, i) => (
          <div key={i} style={{ opacity: ease(frame, templateAt + 4 + i * 3, templateAt + 12 + i * 3), whiteSpace: "pre" }}>
            {l.split(/(\{\{[^}]+\}\})/g).map((part, k) => (
              <span key={k} style={{ color: part.startsWith("{{") ? C.coral : "rgba(232,230,225,0.85)" }}>{part}</span>
            ))}
          </div>
        ))}
      </div>
      {/* template → body arrow */}
      <svg width={1920} height={1080} style={{ position: "absolute", inset: 0, pointerEvents: "none", overflow: "visible", opacity: tplIn * tplKeep * act1 }}>
        <Wire id="tpl" a={{ x: 960, y: 420 }} b={{ x: CARD.x + CARD.w, y: CARD.y + 420 }} frame={frame} drawStart={templateAt + 10} drawEnd={templateAt + 26} color="rgba(249,115,22,0.6)" width={3} />
        <Comet a={{ x: 960, y: 420 }} b={{ x: CARD.x + CARD.w, y: CARD.y + 420 }} frame={frame} start={expectsAt - 10} end={expectsAt + 2} color={C.coral} r={8} />
      </svg>

      {/* c2: any HTTP API — three generic destinations */}
      {DESTS.map((d, i) => (
        <div key={d.url} style={{ ...card, position: "absolute", left: 1240, top: d.y, width: 580, padding: "18px 22px", display: "flex", alignItems: "center", gap: 14, opacity: destIn[i] * act1, transform: `translateX(${(1 - destIn[i]) * 30}px)` }}>
          <span style={{ fontFamily: F.mono, fontSize: 20, fontWeight: 700, color: "#fff", background: C.tealDeep, borderRadius: 8, padding: "4px 12px" }}>{d.m}</span>
          <span style={{ fontFamily: F.mono, fontSize: 21, color: C.ink }}>{d.url}</span>
        </div>
      ))}
      <svg width={1920} height={1080} style={{ position: "absolute", inset: 0, pointerEvents: "none", overflow: "visible", opacity: act1 }}>
        {DESTS.map((d, i) => {
          const b = { x: 1240, y: d.y + 36 };
          return (
            <React.Fragment key={d.url}>
              <Wire id={`d-${i}`} a={reqRight} b={b} frame={frame} drawStart={anyAt + i * 5} drawEnd={anyAt + 18 + i * 5} color="rgba(0,173,216,0.45)" width={2.5} />
              <Comet a={reqRight} b={b} frame={frame} start={anyAt + 8 + i * 6} end={anyAt + 26 + i * 6} color={C.coral} r={8} />
              <Ripple x={b.x} y={b.y} frame={frame} at={anyAt + 26 + i * 6} color={C.teal} size={40} />
            </React.Fragment>
          );
        })}
      </svg>
      <div style={{ position: "absolute", left: 1240, top: 640, opacity: notJust * act1, transform: `translateY(${(1 - notJust) * 10}px)` }}>
        <Chip tone="coral" size={24}>any HTTP API · not just a webhook</Chip>
      </div>

      {/* ── Act 2: recipes ── */}
      {RECIPES.map((rc, i) => {
        const s = r(rc.at);
        const p = tilePos(i);
        const isPD = i === 0;
        return (
          <div key={rc.label} style={{ ...card, position: "absolute", left: p.x, top: p.y, width: GRID.w, height: GRID.h, display: "flex", alignItems: "center", gap: 18, padding: "0 24px", opacity: s, transform: `translateY(${(1 - s) * 18}px) scale(${0.92 + s * 0.08})`, borderColor: isPD && hi > 0 ? `rgba(0,173,216,${0.2 + hi * 0.6})` : undefined, boxShadow: isPD && hi > 0 ? `0 16px 44px rgba(11,15,20,0.09), 0 0 ${hi * 30}px rgba(0,173,216,0.4)` : undefined }}>
            <Img src={staticFile(rc.icon)} style={{ width: 44, height: 44 }} />
            <div>
              <div style={{ fontFamily: F.display, fontSize: 28, fontWeight: 700, color: C.ink }}>{rc.label}</div>
              <div style={{ fontFamily: F.mono, fontSize: 18, color: C.tealDeep }}>{rc.fmt}</div>
            </div>
          </div>
        );
      })}
      {frame >= prebuiltAt - 4 && (
        <div style={{ position: "absolute", left: 120, top: 455, opacity: r(prebuiltAt) * (1 - termIn), transform: `translateY(${(1 - r(prebuiltAt)) * 10}px)` }}>
          <Chip tone="coral" size={22}>same template + headers, pre-built per destination</Chip>
        </div>
      )}
      {frame >= yamlAt - 4 && (
        <div style={{ position: "absolute", left: 1000, top: 470, width: 800, background: C.navy, color: C.cream, borderRadius: 18, padding: "18px 26px", fontFamily: F.mono, fontSize: 20, lineHeight: 1.55, boxShadow: "0 22px 56px rgba(11,15,20,0.25)", opacity: yamlIn, transform: `translateY(${(1 - yamlIn) * 16}px)` }}>
          <div style={{ color: C.teal, fontSize: 17, marginBottom: 8 }}>satellites/recipes/pagerduty.yaml</div>
          {YAML.map(([k, v], i) => (
            <div key={i} style={{ opacity: ease(frame, yamlAt + 6 + i * 3, yamlAt + 14 + i * 3), whiteSpace: "pre" }}>
              <span style={{ color: C.teal }}>{k}</span>
              <span style={{ color: v.includes("{{") ? C.coral : "rgba(232,230,225,0.8)" }}>{v}</span>
            </div>
          ))}
        </div>
      )}
      {frame >= c(4) - 4 && (
        <div style={{ position: "absolute", left: 120, top: 470, width: 800, opacity: termIn, transform: `translateY(${(1 - termIn) * 16}px)` }}>
          <Terminal title="terminal" fontSize={26}>
            <div>
              <span style={{ color: C.teal }}>$ </span>
              <TypeLine text="sparrow use pagerduty --event item.shipped" from={cmdAt} to={cmdAt + 30} />
            </div>
            <div style={{ color: C.teal, opacity: ease(frame, cmdAt + 34, cmdAt + 42) }}>✓ webhook + subscription created from recipe</div>
          </Terminal>
        </div>
      )}
      <svg width={1920} height={1080} style={{ position: "absolute", inset: 0, pointerEvents: "none", overflow: "visible" }}>
        <Wire id="rec" a={termCenter} b={pdCenter} frame={frame} drawStart={cmdAt + 20} drawEnd={cmdAt + 36} color="rgba(0,173,216,0.45)" width={2.5} />
        <Comet a={termCenter} b={pdCenter} frame={frame} start={cmdAt + 24} end={cmdAt + 38} color={C.teal} r={8} />
        <Ripple x={pdCenter.x} y={pdCenter.y} frame={frame} at={cmdAt + 38} color={C.teal} size={50} />
      </svg>

      <Sequence from={expectsAt}><Audio src={staticFile("sfx/ping.mp3")} volume={0.14} /></Sequence>
      <Sequence from={anyAt + 26}><Audio src={staticFile("sfx/chime.mp3")} volume={0.14} /></Sequence>
      <Sequence from={cmdAt + 38}><Audio src={staticFile("sfx/ping.mp3")} volume={0.18} /></Sequence>
    </AbsoluteFill>
  );
};

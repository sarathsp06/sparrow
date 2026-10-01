import React from "react";
import { AbsoluteFill, Sequence, interpolate, staticFile, useCurrentFrame } from "remotion";
import { Audio } from "@remotion/media";
import { C, F } from "../../theme";
import { cueFrame, cueEndFrame, wordFrame, useReveal, ease } from "../timing";
import { card, Chip, Dock, NightKicker, Parcel, Terminal, TypeLine } from "../ui";
import { Wire, Comet, Ripple } from "../../components/Wire";

// Frame 4 — Night one: the happy path, on one stage.
// Three columns: the push terminal (left), the request being built (middle),
// the docks that subscribe (right). Everything reveals on its spoken cue.

const ID = "04-happy";
const c0 = cueFrame(ID, 0);
const c1 = cueFrame(ID, 1);
const c2 = cueFrame(ID, 2);
const c3 = cueFrame(ID, 3);
const c4 = cueFrame(ID, 4);

const COL_A = { x: 80, w: 600 };
const COL_B = { x: 720, w: 520 };
const COL_C = { x: 1290, w: 550 };
const TOP = 170;

const CMD1 = "curl -X POST /v1/consumers/acme/events?event=item.shipped";
const CMD2 = `-d '{"payload":{"parcel":"pkg_88a1",…},"idempotency_key":"pkg_88a1"}'`;

const SUBS = [
  { name: "billing-svc", sub: "item.shipped", ok: true },
  { name: "partner-dhl", sub: "item.shipped · region=eu", ok: true },
  { name: "crm-sync", sub: "item.shipped", ok: true },
  { name: "analytics-us", sub: "item.shipped · region=us", ok: false },
];
const DOCK_H = 96;
const DOCK_GAP = 18;
const dockY = (i: number) => TOP + 90 + i * (DOCK_H + DOCK_GAP);

export const Happy: React.FC = () => {
  const frame = useCurrentFrame();
  const r = useReveal();

  // c0: terminal types the POST as the VO says "one POST"
  const termIn = r(c0 + 6);
  // wide + big while it's the only thing on stage; shrinks into column A at fan-out
  const shrink = ease(frame, c3 - 8, c3 + 14);
  const termW = interpolate(shrink, [0, 1], [1160, COL_A.w]);
  const termFont = interpolate(shrink, [0, 1], [25, 20]);
  const t1a = c0 + 10;
  const t1b = t1a + 34;
  const t2a = t1b + 3;
  const t2b = t2a + 26;
  // parcel flies from the prompt into Sparrow (the hub is the terminal itself here)
  const resp1At = Math.max(c1, t2b + 2);
  const resp1 = r(resp1At);
  const idAt = wordFrame(ID, 1, "event");
  const idGlow = ease(frame, idAt, idAt + 12);

  // c2: ghost re-push + duplicate:true
  const g1a = c2 + 2;
  const g1b = c2 + 22;
  const resp2At = wordFrame(ID, 2, "same", 1);
  const resp2 = r(resp2At);
  const stampAt = wordFrame(ID, 2, "no");
  const stamp = r(stampAt);

  // c3: fan-out — docks appear, filters land on their words
  const docksAt = c3 + 4;
  const nameAt = wordFrame(ID, 3, "name");
  const consumerAt = wordFrame(ID, 3, "consumer");
  const labelsAt = wordFrame(ID, 3, "labels");
  const greyAt = wordFrame(ID, 3, "region");
  const grey = ease(frame, greyAt, greyAt + 14);

  // c4: request card builds
  const reqAt = c4;
  const reqIn = r(reqAt);
  const envAt = wordFrame(ID, 4, "envelope");
  const tplAt = wordFrame(ID, 4, "template");
  const hdrAt = wordFrame(ID, 4, "secret");
  const encAt = wordFrame(ID, 4, "encrypted");
  const methodAt = wordFrame(ID, 4, "any");
  const tplMorph = ease(frame, tplAt, tplAt + 14);
  const METHODS = ["POST", "PUT", "PATCH", "DELETE"];
  const methodIdx = Math.min(3, Math.max(0, Math.floor((frame - methodAt) / 9)));
  const method = frame < methodAt ? "POST" : frame > methodAt + 40 ? "POST" : METHODS[methodIdx];

  // c5: signed + out the door — comets to the three open docks
  const sigAt = wordFrame(ID, 5, "signed");
  const sig = r(sigAt);
  const outAt = wordFrame(ID, 5, "out");
  const ticksAt = wordFrame(ID, 5, "three", 2);
  const reqRight = { x: COL_B.x + COL_B.w, y: TOP + 250 };

  const c5End = cueEndFrame(ID, 5);

  return (
    <AbsoluteFill>
      <NightKicker label="NIGHT ONE" clock="22:04" at={c0} />

      {/* ── Column A: push terminal ── */}
      <div
        style={{
          position: "absolute",
          left: COL_A.x,
          top: TOP,
          width: termW,
          opacity: termIn,
          transform: `translateY(${(1 - termIn) * 16}px)`,
        }}
      >
        <Terminal title="warehouse-app" fontSize={termFont} style={{ minHeight: interpolate(shrink, [0, 1], [470, 560]) }}>
          <div style={{ whiteSpace: "pre-wrap", wordBreak: "break-all" }}>
            <span style={{ color: C.teal }}>$ </span>
            <TypeLine text={CMD1} from={t1a} to={t1b} />
          </div>
          <div style={{ whiteSpace: "pre-wrap", wordBreak: "break-all", paddingLeft: 22 }}>
            <TypeLine text={CMD2} from={t2a} to={t2b} />
          </div>
          {resp1 > 0.01 && (
            <div style={{ marginTop: 14, opacity: resp1, transform: `translateY(${(1 - resp1) * 8}px)` }}>
              <div style={{ color: C.teal, fontWeight: 700 }}>201 Created</div>
              <div style={{ color: "rgba(232,230,225,0.8)" }}>
                {"{ "}
                <span style={{ color: C.teal, textShadow: `0 0 ${idGlow * 14}px rgba(0,173,216,0.8)` }}>"event_id"</span>
                {': "evt_9f2…", "duplicate": '}
                <span style={{ color: C.teal }}>false</span>
                {" }"}
              </div>
            </div>
          )}
          {frame >= g1a && (
            <div style={{ marginTop: 16, opacity: ease(frame, g1a, g1a + 6), whiteSpace: "pre-wrap", wordBreak: "break-all" }}>
              <span style={{ color: C.teal }}>$ </span>
              <TypeLine text={CMD1} from={g1a} to={g1b} color="rgba(232,230,225,0.5)" />
              <div style={{ paddingLeft: 22, color: "rgba(232,230,225,0.5)" }}>
                <TypeLine text={`… "idempotency_key":"pkg_88a1"`} from={g1b - 10} to={g1b + 8} />
              </div>
            </div>
          )}
          {resp2 > 0.01 && (
            <div style={{ marginTop: 12, opacity: resp2 }}>
              <div style={{ color: C.teal, fontWeight: 700 }}>201 Created</div>
              <div style={{ color: "rgba(232,230,225,0.8)" }}>
                {'{ "event_id": "evt_9f2…", "duplicate": '}
                <span style={{ color: C.coral, fontWeight: 700, textShadow: "0 0 12px rgba(249,115,22,0.6)" }}>true</span>
                {" }"}
              </div>
            </div>
          )}
        </Terminal>
        {stamp > 0.01 && (
          <div
            style={{
              position: "absolute",
              right: -10,
              bottom: -26,
              opacity: stamp,
              transform: `scale(${0.85 + stamp * 0.15}) rotate(-4deg)`,
            }}
          >
            <Chip tone="coral" size={22}>
              ✕ same event · no duplicate
            </Chip>
          </div>
        )}
      </div>

      {/* ── Column C: subscriptions / docks ── */}
      {SUBS.map((s, i) => {
        const inS = r(docksAt + i * 6);
        const isGrey = !s.ok && grey > 0;
        return (
          <div
            key={s.name}
            style={{
              position: "absolute",
              left: COL_C.x,
              top: dockY(i),
              opacity: inS,
              transform: `translateX(${(1 - inS) * 30}px)`,
            }}
          >
            <Dock
              name={s.name}
              sub={s.sub}
              width={COL_C.w}
              state={isGrey ? "skipped" : frame >= ticksAt + i * 6 && s.ok ? "open" : "idle"}
              badge={
                isGrey ? (
                  <Chip tone="red" size={18}>✕</Chip>
                ) : frame >= ticksAt + i * 6 && s.ok ? (
                  <Chip tone="teal" size={20}>✓</Chip>
                ) : undefined
              }
            />
          </div>
        );
      })}
      {/* filter chips under the docks */}
      <div style={{ position: "absolute", left: COL_C.x, top: TOP, display: "flex", gap: 12 }}>
        {[
          { l: "event ✓", at: nameAt, tone: "teal" as const },
          { l: "consumer: acme ✓", at: consumerAt, tone: "teal" as const },
          { l: "labels · region=eu", at: labelsAt, tone: "coral" as const },
        ].map((f) => {
          const s = r(f.at);
          return (
            <div key={f.l} style={{ opacity: s, transform: `translateY(${(1 - s) * 10}px)` }}>
              <Chip tone={f.tone} size={19}>{f.l}</Chip>
            </div>
          );
        })}
      </div>

      {/* ── Column B: request card ── */}
      <div
        style={{
          ...card,
          position: "absolute",
          left: COL_B.x,
          top: TOP + 90,
          width: COL_B.w,
          padding: 0,
          overflow: "hidden",
          opacity: reqIn,
          transform: `translateY(${(1 - reqIn) * 16}px)`,
          borderColor: sig > 0.5 ? "rgba(0,173,216,0.6)" : undefined,
          boxShadow: sig > 0.5 ? `0 16px 44px rgba(11,15,20,0.09), 0 0 ${sig * 30}px rgba(0,173,216,0.35)` : undefined,
        }}
      >
        <div style={{ padding: "16px 24px", borderBottom: "1px solid rgba(11,15,20,0.08)", display: "flex", alignItems: "center", gap: 12, fontFamily: F.mono }}>
          <span style={{ color: "#fff", background: C.tealDeep, borderRadius: 8, padding: "4px 14px", fontSize: 22, fontWeight: 700, minWidth: 86, textAlign: "center" }}>
            {method}
          </span>
          <span style={{ fontSize: 19, color: C.ink, whiteSpace: "nowrap", overflow: "hidden", textOverflow: "ellipsis" }}>hooks.partner-dhl.example/in</span>
        </div>
        <div style={{ padding: "14px 24px", borderBottom: "1px solid rgba(11,15,20,0.08)", fontFamily: F.mono, fontSize: 19, lineHeight: 1.65 }}>
          <div style={{ fontSize: 14, letterSpacing: "0.12em", color: "rgba(11,15,20,0.45)" }}>HEADERS</div>
          <div style={{ opacity: ease(frame, hdrAt, hdrAt + 8) }}>
            <span style={{ color: C.tealDeep }}>content-type</span>: application/json
          </div>
          <div style={{ opacity: ease(frame, hdrAt + 6, hdrAt + 14), display: "flex", alignItems: "center", gap: 8 }}>
            <span><span style={{ color: C.tealDeep }}>authorization</span>: Bearer ••••••••</span>
            <svg width={16} height={20} viewBox="0 0 16 20" style={{ opacity: ease(frame, encAt, encAt + 8) }}>
              <rect x={1} y={9} width={14} height={10} rx={2.5} fill={C.tealDeep} />
              <path d="M4 9 V6 a4 4 0 0 1 8 0 V9" stroke={C.tealDeep} strokeWidth={2} fill="none" />
            </svg>
            <span style={{ fontSize: 15, color: C.tealDeep, opacity: ease(frame, encAt, encAt + 10) }}>AES-256-GCM at rest</span>
          </div>
          {sig > 0.01 && (
            <>
              <div style={{ opacity: ease(frame, sigAt, sigAt + 8), color: C.coralText }}>webhook-id: msg_9f2a…</div>
              <div style={{ opacity: ease(frame, sigAt + 4, sigAt + 12), color: C.coralText }}>webhook-timestamp: 1790799840</div>
              <div style={{ opacity: ease(frame, sigAt + 8, sigAt + 16), color: C.coralText }}>webhook-signature: v1,K5oQ7r…</div>
            </>
          )}
        </div>
        <div style={{ padding: "14px 24px 18px", fontFamily: F.mono, fontSize: 19, lineHeight: 1.6 }}>
          <div style={{ fontSize: 14, letterSpacing: "0.12em", color: "rgba(11,15,20,0.45)", display: "flex", gap: 10, alignItems: "center" }}>
            BODY
            <span style={{ opacity: tplMorph, color: C.coralText, background: "rgba(249,115,22,0.1)", borderRadius: 999, padding: "2px 10px", fontSize: 13 }}>
              {"{{ .payload.parcel }}"}
            </span>
          </div>
          {tplMorph < 0.5 ? (
            <div style={{ opacity: ease(frame, envAt, envAt + 10) }}>
              {"{"}
              <div style={{ paddingLeft: 18 }}><span style={{ color: C.tealDeep }}>"event_name"</span>: "item.shipped",</div>
              <div style={{ paddingLeft: 18, opacity: ease(frame, envAt + 4, envAt + 12) }}><span style={{ color: C.tealDeep }}>"event_id"</span>: "evt_9f2…",</div>
              <div style={{ paddingLeft: 18, opacity: ease(frame, envAt + 8, envAt + 16) }}><span style={{ color: C.tealDeep }}>"attempt"</span>: 1,</div>
              <div style={{ paddingLeft: 18, opacity: ease(frame, envAt + 12, envAt + 20) }}><span style={{ color: C.tealDeep }}>"payload"</span>: {"{ … }"}</div>
              {"}"}
            </div>
          ) : (
            <div style={{ opacity: tplMorph }}>
              {"{"}
              <div style={{ paddingLeft: 18 }}><span style={{ color: C.tealDeep }}>"tracking"</span>: "pkg_88a1",</div>
              <div style={{ paddingLeft: 18 }}><span style={{ color: C.tealDeep }}>"weight_kg"</span>: 2.4,</div>
              <div style={{ paddingLeft: 18 }}><span style={{ color: C.tealDeep }}>"status"</span>: "shipped"</div>
              {"}"}
            </div>
          )}
        </div>
        {sig > 0.01 && (
          <div style={{ position: "absolute", bottom: 14, right: 18, opacity: sig, transform: `scale(${0.8 + sig * 0.2}) rotate(-4deg)` }}>
            <Chip tone="teal" size={18}>✓ signed · HMAC / Ed25519</Chip>
          </div>
        )}
      </div>

      {/* ── Wires: request → docks, parcels out the door ── */}
      <svg width={1920} height={1080} style={{ position: "absolute", inset: 0, pointerEvents: "none", overflow: "visible" }}>
        {SUBS.map((s, i) => {
          if (!s.ok) return null;
          const b = { x: COL_C.x, y: dockY(i) + DOCK_H / 2 };
          const k = SUBS.slice(0, i).filter((x) => x.ok).length;
          return (
            <React.Fragment key={s.name}>
              <Wire id={`h-${i}`} a={reqRight} b={b} frame={frame} drawStart={outAt + k * 4} drawEnd={outAt + 18 + k * 4} color="rgba(0,173,216,0.45)" width={2.5} />
              <Comet a={reqRight} b={b} frame={frame} start={outAt + 6 + k * 6} end={ticksAt + k * 6} color={C.coral} r={9} />
              <Ripple x={b.x} y={b.y} frame={frame} at={ticksAt + k * 6} color={C.teal} size={44} />
            </React.Fragment>
          );
        })}
      </svg>
      {/* parcel icons riding the comets */}
      {SUBS.filter((s) => s.ok).map((s, k) => {
        const start = outAt + 6 + k * 6;
        const end = ticksAt + k * 6;
        if (frame < start || frame > end) return null;
        const t = interpolate(frame, [start, end], [0, 1], { extrapolateLeft: "clamp", extrapolateRight: "clamp" });
        const i = SUBS.indexOf(s);
        const x = reqRight.x + (COL_C.x - reqRight.x) * t;
        const y = reqRight.y + (dockY(i) + DOCK_H / 2 - reqRight.y) * t;
        return <Parcel key={s.name} size={40} style={{ position: "absolute", left: x - 20, top: y - 34 }} />;
      })}

      {/* Held beat: after the last tick, a quiet "3 / 3 delivered" */}
      <div
        style={{
          position: "absolute",
          left: COL_C.x,
          top: dockY(SUBS.length) + 6,
          opacity: r(Math.min(c5End - 10, ticksAt + 24)),
        }}
      >
        <Chip tone="teal" size={22}>3 / 3 delivered · night one</Chip>
      </div>

      <Sequence from={t1a}><Audio src={staticFile("sfx/typing.mp3")} volume={0.12} /></Sequence>
      <Sequence from={resp1At + 2}><Audio src={staticFile("sfx/ping.mp3")} volume={0.16} /></Sequence>
      <Sequence from={ticksAt}><Audio src={staticFile("sfx/chime.mp3")} volume={0.18} /></Sequence>
    </AbsoluteFill>
  );
};

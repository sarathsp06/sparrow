import React from "react";
import { AbsoluteFill, Img, Sequence, interpolate, staticFile, useCurrentFrame } from "remotion";
import { Audio } from "@remotion/media";
import { C, F } from "../../theme";
import { cueFrame, wordFrame, useReveal, ease } from "../timing";
import { card, Chip, Dock, NightKicker, Parcel, Terminal, TypeLine } from "../ui";
import { Comet, Ripple } from "../../components/Wire";

// Frame 6 — Night two: the partner goes dark at 03:12 and comes back at 06:40.
// Upper stage: Sparrow hub (left) → partner dock (right), a clock that moves.
// Lower stage: night = classify bins + retry timeline + health; dawn = the
// one-shot bulk retry terminal and the attempt log.

const ID = "06-night";
const c = (i: number) => cueFrame(ID, i);

const AMBER = "#F5A524";

const HUB = { x: 330, y: 300 };
const DOCK = { x: 1240, y: 250 };
const DOCK_W = 560;
const DOCK_MID = { x: DOCK.x, y: DOCK.y + 66 };

// Retry timeline geometry (night lower stage)
const TL = { x: 120, y: 690, w: 1160 };
const UNITS = 7;
const ux = (u: number) => TL.x + (u / UNITS) * TL.w;

export const Night: React.FC = () => {
  const frame = useCurrentFrame();
  const r = useReveal();

  // ── cue anchors ──
  const closedAt = wordFrame(ID, 0, "closed");
  const fiveAt = wordFrame(ID, 0, "503");
  const readsAt = wordFrame(ID, 1, "reads");
  const retryAt = wordFrame(ID, 1, "worth");
  const stopAt = wordFrame(ID, 1, "stops");
  const oneAt = wordFrame(ID, 2, "one");
  const twoAt = wordFrame(ID, 2, "two");
  const fourAt = wordFrame(ID, 2, "four");
  const cappedAt = wordFrame(ID, 2, "capped");
  const a429At = wordFrame(ID, 3, "429");
  const waitsAt = wordFrame(ID, 3, "waits");
  const notCountAt = wordFrame(ID, 3, "doesn't");
  const healthyAt = wordFrame(ID, 4, "healthy");
  const degradedAt = wordFrame(ID, 4, "degraded");
  const unhealthyAt = wordFrame(ID, 4, "unhealthy");
  const fiveRowAt = wordFrame(ID, 5, "five");
  const emailAt = wordFrame(ID, 5, "email");
  const asleepAt = wordFrame(ID, 5, "asleep");
  const dawnAt = c(6);
  const backAt = wordFrame(ID, 6, "back");
  const failedAt = wordFrame(ID, 6, "every");
  const snapshotAt = wordFrame(ID, 6, "snapshot");
  const retryOneAt = wordFrame(ID, 6, "retry");
  const liveAt = wordFrame(ID, 7, "live");
  const exactlyAt = wordFrame(ID, 7, "exactly");
  const recordAt = wordFrame(ID, 8, "record");
  const statusAt = wordFrame(ID, 8, "status");
  const timingAt = wordFrame(ID, 8, "timing");
  const categoryAt = wordFrame(ID, 8, "category");

  // ── phase ──
  const dawn = ease(frame, dawnAt - 6, dawnAt + 20); // 0 night → 1 dawn
  const clock = frame < oneAt ? "03:12" : frame < twoAt ? "03:13" : frame < fourAt ? "03:15" : frame < a429At ? "03:19" : frame < dawnAt - 6 ? "03:21" : "06:40";

  // ── upper stage ──
  const hubIn = r(c(0));
  const dockIn = r(c(0) + 4);
  const shut = ease(frame, closedAt, closedAt + 10);
  const cometBounce = frame >= c(0) + 6 && frame < closedAt + 30;
  const partnerState: "closed" | "open" | "idle" = frame >= backAt ? "open" : shut > 0.5 ? "closed" : "idle";

  // ── night lower stage ──
  const nightOp = 1 - dawn;
  const tlIn = ease(frame, c(2) - 4, c(2) + 16);
  const gapR = [ease(frame, oneAt - 2, oneAt + 10), ease(frame, twoAt - 2, twoAt + 10), ease(frame, fourAt - 2, fourAt + 10)];
  const attR = [r(c(2)), r(oneAt + 6), r(twoAt + 6)];
  const cap = r(cappedAt);
  const chip429 = r(a429At);
  const snooze = r(waitsAt);
  const counter = r(notCountAt);

  // health state (right column under the dock)
  const hState: "healthy" | "degraded" | "unhealthy" | null =
    frame >= unhealthyAt ? "unhealthy" : frame >= degradedAt ? "degraded" : frame >= healthyAt ? "healthy" : null;
  const hColor = hState === "healthy" ? C.teal : hState === "degraded" ? AMBER : C.red;
  const hPct = hState === "healthy" ? 96 : hState === "degraded" ? 86 : hState === "unhealthy" ? 71 : 0;
  const healthIn = r(c(4));
  const dots = Array.from({ length: 5 }, (_, i) => ease(frame, fiveRowAt + i * 4, fiveRowAt + i * 4 + 8));
  const emailIn = r(emailAt);
  const asleep = r(asleepAt);

  // ── dawn lower stage ──
  const termIn = r(backAt + 6);
  const q1a = failedAt;
  const q1b = q1a + 46;
  const r1 = r(Math.max(snapshotAt, q1b + 2));
  const q2a = retryOneAt;
  const q2b = q2a + 26;
  const streamStart = q2b + 4;
  const streamEnd = streamStart + 40;
  const count = Math.round(interpolate(frame, [streamStart, streamEnd], [0, 128], { extrapolateLeft: "clamp", extrapolateRight: "clamp" }));
  const snapIn = r(liveAt);
  const exactly = r(exactlyAt);
  const logIn = r(recordAt);

  const ROWS = [
    { n: "#1", t: "03:12", s: "503", cat: "server_error", ok: false },
    { n: "#2", t: "03:13", s: "503", cat: "server_error", ok: false },
    { n: "#3", t: "03:15", s: "503", cat: "server_error", ok: false },
    { n: "#4", t: "03:19", s: "429", cat: "rate_limited", ok: false },
    { n: "#5", t: "06:40", s: "200", cat: "success", ok: true },
  ];

  return (
    <AbsoluteFill>
      {/* Night tint that lifts at dawn */}
      <AbsoluteFill
        style={{
          background: "linear-gradient(180deg, rgba(20,18,56,0.86) 0%, rgba(20,18,56,0.74) 50%, rgba(20,18,56,0.62) 100%)",
          opacity: nightOp,
        }}
      />
      <AbsoluteFill
        style={{
          background: "linear-gradient(180deg, rgba(249,115,22,0.18) 0%, rgba(249,115,22,0.04) 50%, transparent 100%)",
          opacity: dawn,
        }}
      />
      {/* moon / sun */}
      <div
        style={{
          position: "absolute",
          right: 130,
          top: 60,
          width: 72,
          height: 72,
          borderRadius: "50%",
          background: dawn < 0.5 ? "#F4F2ED" : C.coral,
          boxShadow: dawn < 0.5 ? "0 0 40px rgba(244,242,237,0.6), inset -18px -8px 0 rgba(24,22,60,0.35)" : "0 0 60px rgba(249,115,22,0.6)",
          opacity: r(c(0)),
        }}
      />

      <NightKicker label={dawn < 0.5 ? "NIGHT TWO" : "NIGHT TWO · DAWN"} clock={clock} at={c(0)} tone={dawn < 0.5 ? "night" : "dawn"} />

      {/* ── Upper stage: hub → dock ── */}
      <div style={{ position: "absolute", left: HUB.x - 70, top: HUB.y - 70, opacity: hubIn, display: "flex", flexDirection: "column", alignItems: "center", gap: 8 }}>
        <div style={{ width: 140, height: 140, borderRadius: "50%", background: "rgba(0,173,216,0.12)", border: `3px solid ${C.teal}`, display: "grid", placeItems: "center", boxShadow: "0 0 30px rgba(0,173,216,0.3)" }}>
          <Img src={staticFile("sparrow-logo.svg")} style={{ width: 74, height: 60 }} />
        </div>
        <div style={{ fontFamily: F.display, fontSize: 24, fontWeight: 700, color: dawn < 0.5 ? C.cream : C.ink }}>Sparrow</div>
      </div>

      <svg width={1920} height={1080} style={{ position: "absolute", inset: 0, pointerEvents: "none", overflow: "visible" }}>
        <line x1={HUB.x + 74} y1={HUB.y} x2={DOCK.x - 10} y2={DOCK_MID.y} stroke={dawn < 0.5 ? "rgba(244,242,237,0.28)" : "rgba(11,15,20,0.22)"} strokeWidth={3} strokeDasharray="2 12" strokeLinecap="round" strokeDashoffset={-frame * 0.9} opacity={hubIn} />
        {cometBounce && (
          <>
            <Comet a={{ x: HUB.x + 74, y: HUB.y }} b={{ x: DOCK.x - 10, y: DOCK_MID.y }} frame={frame} start={c(0) + 6} end={closedAt} color={C.coral} r={10} />
            <Comet a={{ x: DOCK.x - 10, y: DOCK_MID.y }} b={{ x: HUB.x + 74, y: HUB.y }} frame={frame} start={closedAt + 2} end={closedAt + 26} color={C.red} r={8} />
          </>
        )}
        <Ripple x={DOCK.x - 10} y={DOCK_MID.y} frame={frame} at={closedAt} color={C.red} size={60} />
        {frame >= backAt && (
          <Ripple x={DOCK.x - 10} y={DOCK_MID.y} frame={frame} at={backAt} color={C.teal} size={70} />
        )}
        {/* dawn: a stream of 128 parcels */}
        {frame >= streamStart && frame <= streamEnd + 8 &&
          Array.from({ length: 7 }, (_, k) => (
            <Comet key={k} a={{ x: HUB.x + 74, y: HUB.y }} b={{ x: DOCK.x - 10, y: DOCK_MID.y }} frame={frame} start={streamStart + k * 5} end={streamStart + 22 + k * 5} color={C.coral} r={7} />
          ))}
      </svg>

      <div style={{ position: "absolute", left: DOCK.x, top: DOCK.y, opacity: dockIn, transform: `translateX(${(1 - dockIn) * 30}px)` }}>
        <Dock
          name="partner-dhl"
          sub="hooks.partner-dhl.example/in"
          width={DOCK_W}
          state={partnerState}
          badge={
            partnerState === "closed" ? (
              <Chip tone="red" size={24} style={{ opacity: ease(frame, fiveAt, fiveAt + 8) }}>503</Chip>
            ) : partnerState === "open" ? (
              <Chip tone="teal" size={22}>{count > 0 ? `${count} / 128 ✓` : "back online"}</Chip>
            ) : undefined
          }
        />
        {/* health under the dock (night) */}
        <div style={{ marginTop: 18, display: "flex", alignItems: "center", gap: 16, opacity: healthIn * nightOp }}>
          <div style={{ width: 300, height: 16, borderRadius: 8, background: "rgba(244,242,237,0.18)", overflow: "hidden" }}>
            <div style={{ width: `${hPct}%`, height: "100%", background: hColor }} />
          </div>
          {hState && (
            <Chip tone={hState === "healthy" ? "teal" : hState === "degraded" ? "coral" : "red"} size={20}>
              {hState} · {hPct}%
            </Chip>
          )}
        </div>
        <div style={{ marginTop: 14, display: "flex", gap: 10, alignItems: "center", opacity: nightOp }}>
          {dots.map((d, i) => (
            <div key={i} style={{ width: 30, height: 30, borderRadius: "50%", border: `2px solid ${C.red}`, background: `rgba(229,72,77,${d * 0.3})`, display: "grid", placeItems: "center", fontFamily: F.mono, fontWeight: 700, fontSize: 16, color: C.red, opacity: d, transform: `scale(${0.6 + d * 0.4})` }}>✕</div>
          ))}
          <span style={{ fontFamily: F.mono, fontSize: 18, color: C.cream, opacity: dots[4] }}>5 in a row → unhealthy</span>
        </div>
        {/* email card */}
        {frame >= emailAt - 4 && (
          <div style={{ ...card, marginTop: 16, width: DOCK_W, padding: "16px 22px", borderLeft: `5px solid ${C.red}`, opacity: emailIn * nightOp, transform: `translateX(${(1 - emailIn) * 30}px)` }}>
            <div style={{ fontFamily: F.mono, fontSize: 17, color: "rgba(11,15,20,0.5)" }}>✉ to oncall@acme.example</div>
            <div style={{ fontFamily: F.mono, fontSize: 20, fontWeight: 700, color: C.ink, marginTop: 4 }}>sparrow.webhook.health_changed</div>
            <div style={{ fontFamily: F.display, fontSize: 20, marginTop: 4 }}>
              partner-dhl: <span style={{ color: AMBER }}>degraded</span> → <span style={{ color: C.red }}>unhealthy</span>
            </div>
          </div>
        )}
      </div>

      {/* "You're asleep" */}
      {frame >= asleepAt - 4 && (
        <div style={{ position: "absolute", left: HUB.x - 40, top: HUB.y + 130, opacity: asleep * nightOp, fontFamily: F.serif, fontSize: 40, color: C.cream, fontStyle: "italic" }}>
          you: asleep <span style={{ fontFamily: F.mono, fontSize: 26, color: C.teal }}>z z z</span>
        </div>
      )}

      {/* ── Night lower stage: bins + timeline ── */}
      <div style={{ position: "absolute", left: 120, top: 460, display: "flex", gap: 20, opacity: nightOp }}>
        {[
          { l: "↻ retry", tone: "teal" as const, at: retryAt, items: ["5xx", "timeout", "refused", "network", "429"] },
          { l: "■ stop", tone: "red" as const, at: stopAt, items: ["4xx", "DNS", "TLS"] },
        ].map((bin) => {
          const s = r(bin.at);
          return (
            <div key={bin.l} style={{ ...card, padding: "14px 18px", opacity: s * r(readsAt), transform: `translateY(${(1 - s) * 14}px)`, borderColor: bin.tone === "teal" ? "rgba(0,173,216,0.5)" : "rgba(229,72,77,0.5)", display: "flex", alignItems: "center", gap: 12 }}>
              <span style={{ fontFamily: F.mono, fontSize: 22, fontWeight: 700, color: bin.tone === "teal" ? C.tealDeep : C.red }}>{bin.l}</span>
              {bin.items.map((it, i) => (
                <span key={it} style={{ opacity: ease(frame, bin.at + 4 + i * 4, bin.at + 12 + i * 4) }}>
                  <Chip tone={bin.tone} size={20}>{it}</Chip>
                </span>
              ))}
            </div>
          );
        })}
      </div>

      <svg width={1920} height={1080} style={{ position: "absolute", inset: 0, pointerEvents: "none", opacity: nightOp }}>
        <line x1={TL.x} y1={TL.y} x2={TL.x + tlIn * TL.w} y2={TL.y} stroke="rgba(244,242,237,0.6)" strokeWidth={4} strokeLinecap="round" />
        {[
          { from: 0, to: 1, l: "+1m" },
          { from: 1, to: 3, l: "+2m" },
          { from: 3, to: 7, l: "+4m" },
        ].map((g, i) => (
          <g key={g.l} opacity={gapR[i]}>
            <line x1={ux(g.from) + 18} y1={TL.y - 34} x2={ux(g.to) - 18} y2={TL.y - 34} stroke={C.coral} strokeWidth={3} />
            <text x={(ux(g.from) + ux(g.to)) / 2} y={TL.y - 46} textAnchor="middle" fontFamily={F.mono} fontSize={26} fontWeight={700} fill={C.coral}>{g.l}</text>
          </g>
        ))}
        {[0, 1, 3].map((u, i) => (
          <g key={u} opacity={attR[i]}>
            <circle cx={ux(u)} cy={TL.y} r={16} fill={C.red} />
            <text x={ux(u)} y={TL.y + 7} textAnchor="middle" fontFamily={F.mono} fontSize={18} fontWeight={700} fill="#fff">✕</text>
            <text x={ux(u)} y={TL.y + 48} textAnchor="middle" fontFamily={F.mono} fontSize={20} fill={C.cream}>#{i + 1}</text>
          </g>
        ))}
        {/* 429 snooze arc from #3 lane to #4 */}
        <g opacity={snooze}>
          <path d={`M ${ux(3) + 30} ${TL.y - 10} Q ${(ux(3) + ux(7)) / 2} ${TL.y - 150}, ${ux(7) - 30} ${TL.y - 10}`} fill="none" stroke={C.teal} strokeWidth={3} strokeDasharray="10 8" strokeLinecap="round" />
          <text x={(ux(3) + ux(7)) / 2} y={TL.y - 96} textAnchor="middle" fontFamily={F.mono} fontSize={22} fontWeight={700} fill={C.teal}>waits Retry-After · z z</text>
        </g>
        <g opacity={counter}>
          <circle cx={ux(7)} cy={TL.y} r={16} fill="none" stroke={C.teal} strokeWidth={3} strokeDasharray="4 4" />
          <text x={ux(7)} y={TL.y + 48} textAnchor="middle" fontFamily={F.mono} fontSize={20} fill={C.cream}>#4 · still 3 / 4 attempts</text>
        </g>
      </svg>
      {frame >= cappedAt - 4 && (
        <div style={{ position: "absolute", left: TL.x, top: TL.y + 70, opacity: cap * nightOp }}>
          <Chip tone="ink" size={22}>delay = 60s × 2ⁿ⁻¹ · max 24h</Chip>
        </div>
      )}
      {frame >= a429At - 4 && (
        <div style={{ position: "absolute", left: ux(5.6), top: TL.y - 175, opacity: chip429 * nightOp, transform: `translateY(${(1 - chip429) * 12}px)` }}>
          <Chip tone="coral" size={22}>429 Too Many Requests · Retry-After: 120</Chip>
        </div>
      )}

      {/* ── Dawn lower stage: bulk retry terminal + attempt log ── */}
      {dawn > 0.01 && (
        <>
          <div style={{ position: "absolute", left: 120, top: 430, width: 1000, opacity: termIn * dawn, transform: `translateY(${(1 - termIn) * 16}px)` }}>
            <Terminal title="one snapshot, one retry" fontSize={21} style={{ minHeight: 250 }}>
              <div style={{ whiteSpace: "pre-wrap", wordBreak: "break-all" }}>
                <span style={{ color: C.teal }}>$ </span>
                <TypeLine text="GET /v1/consumers/acme/deliveries?status=failed&created_after=2026-09-29&prepare_retry=true" from={q1a} to={q1b} />
              </div>
              {r1 > 0.01 && (
                <div style={{ opacity: r1, color: "rgba(232,230,225,0.85)" }}>
                  {'{ "retry_id": '}<span style={{ color: C.teal }}>"rj_3b1e…"</span>{', "total": '}<span style={{ color: C.coral, fontWeight: 700 }}>128</span>{" }"}
                </div>
              )}
              {frame >= q2a && (
                <div style={{ marginTop: 10, whiteSpace: "pre-wrap", wordBreak: "break-all" }}>
                  <span style={{ color: C.teal }}>$ </span>
                  <TypeLine text={`POST /v1/consumers/acme/deliveries:retry  {"retry_id":"rj_3b1e…"}`} from={q2a} to={q2b} />
                </div>
              )}
              {frame >= streamStart && (
                <div style={{ marginTop: 8, color: C.teal, fontWeight: 700 }}>
                  ↻ retrying {count} / 128 {count === 128 ? "✓" : ""}
                </div>
              )}
            </Terminal>
          </div>
          {/* snapshot card */}
          {frame >= liveAt - 4 && (
            <div style={{ ...card, position: "absolute", left: 1240, top: 430, width: DOCK_W, padding: "18px 24px", opacity: snapIn * dawn, transform: `translateX(${(1 - snapIn) * 24}px)`, borderColor: "rgba(0,173,216,0.5)" }}>
              <div style={{ fontFamily: F.mono, fontSize: 18, letterSpacing: "0.12em", color: C.tealDeep, fontWeight: 700 }}>SNAPSHOT · rj_3b1e…</div>
              <div style={{ fontFamily: F.display, fontSize: 24, marginTop: 6, color: C.ink }}>128 deliveries, frozen at 06:40</div>
              <div style={{ fontFamily: F.mono, fontSize: 18, marginTop: 8, color: "rgba(11,15,20,0.55)", opacity: exactly }}>
                new failures after 06:40 → <span style={{ color: C.coralText }}>not included</span>
              </div>
            </div>
          )}
          {/* attempt log */}
          {frame >= recordAt - 4 && (
            <div style={{ ...card, position: "absolute", left: 120, top: 700, width: 1680, padding: "10px 24px", opacity: logIn * dawn, transform: `translateY(${(1 - logIn) * 14}px)`, fontFamily: F.mono, fontSize: 20 }}>
              <div style={{ display: "flex", gap: 0, color: "rgba(11,15,20,0.5)", fontSize: 16, letterSpacing: "0.1em", paddingBottom: 6, borderBottom: "1px solid rgba(11,15,20,0.08)" }}>
                <span style={{ width: 90 }}>ATTEMPT</span>
                <span style={{ width: 150, color: `rgba(11,15,20,${0.5 + 0.5 * ease(frame, timingAt, timingAt + 8)})` }}>TIME</span>
                <span style={{ width: 150, color: `rgba(11,15,20,${0.5 + 0.5 * ease(frame, statusAt, statusAt + 8)})` }}>STATUS</span>
                <span style={{ color: `rgba(11,15,20,${0.5 + 0.5 * ease(frame, categoryAt, categoryAt + 8)})` }}>CATEGORY</span>
              </div>
              <div style={{ display: "flex", gap: 12, paddingTop: 8, fontSize: 18 }}>
                {ROWS.map((row, i) => {
                  const s = r(recordAt + 6 + i * 4);
                  return (
                    <div key={row.n} style={{ display: "flex", opacity: s, width: 326, whiteSpace: "nowrap" }}>
                      <span style={{ width: 60, color: "rgba(11,15,20,0.45)" }}>{row.n}</span>
                      <span style={{ width: 80 }}>{row.t}</span>
                      <span style={{ width: 60, color: row.ok ? C.teal : C.red, fontWeight: 700 }}>{row.s}</span>
                      <span style={{ color: row.ok ? C.tealDeep : C.coralText }}>{row.cat}</span>
                    </div>
                  );
                })}
              </div>
            </div>
          )}
        </>
      )}

      {/* parcel that never made it, sitting by the hub during the night */}
      {frame >= closedAt + 26 && dawn < 0.5 && (
        <Parcel size={54} color={C.coral} style={{ position: "absolute", left: HUB.x + 90, top: HUB.y - 20, opacity: nightOp }} />
      )}

      <Sequence from={closedAt}><Audio src={staticFile("sfx/impact-bass-1.mp3")} volume={0.22} /></Sequence>
      <Sequence from={emailAt}><Audio src={staticFile("sfx/ping.mp3")} volume={0.18} /></Sequence>
      <Sequence from={backAt}><Audio src={staticFile("sfx/chime.mp3")} volume={0.2} /></Sequence>
      <Sequence from={streamEnd}><Audio src={staticFile("sfx/ping.mp3")} volume={0.2} /></Sequence>
    </AbsoluteFill>
  );
};

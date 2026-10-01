import React from "react";
import { AbsoluteFill, Sequence, interpolate, staticFile, useCurrentFrame } from "remotion";
import { Audio } from "@remotion/media";
import { C, F } from "../../theme";
import { cueFrame, wordFrame, useReveal, ease } from "../timing";
import { card, Chip, Dock, NightKicker, Parcel, Terminal, TypeLine } from "../ui";

// Frame 7 — Night three: Tuesday's deploy put the wrong weight on every parcel.
// Fix, re-push the day, pause a partner mid-deploy, and let customers watch
// from the portal.

const ID = "07-tuesday";
const c = (i: number) => cueFrame(ID, i);

const PARCELS = ["pkg_88a1", "pkg_88a2", "pkg_88a3", "pkg_88a4", "pkg_88a5", "pkg_88a6"];

export const Tuesday: React.FC = () => {
  const frame = useCurrentFrame();
  const r = useReveal();

  const wrongAt = wordFrame(ID, 0, "wrong");
  const everyAt = wordFrame(ID, 0, "every");
  const fixAt = wordFrame(ID, 1, "fix");
  const repushAt = wordFrame(ID, 1, "re-push");
  const snapshotAt = wordFrame(ID, 1, "snapshot");
  const jobAt = wordFrame(ID, 1, "job");
  const pauseAt = wordFrame(ID, 2, "pause");
  const waitAt = wordFrame(ID, 2, "wait");
  const resumeAt = wordFrame(ID, 2, "resume");
  const portalAt = wordFrame(ID, 3, "portal");
  const linkAt = wordFrame(ID, 3, "link");

  const parcelsIn = PARCELS.map((_, i) => r(c(0) + i * 3));
  const bug = ease(frame, wrongAt, wrongAt + 10);
  const fixed = ease(frame, fixAt + 6, fixAt + 18);
  const diffIn = r(wrongAt);

  const termIn = r(repushAt);
  const q1a = repushAt + 4;
  const q1b = q1a + 44;
  const resp = r(Math.max(snapshotAt, q1b + 2));
  const q2a = Math.max(jobAt, q1b + 6);
  const q2b = q2a + 20;
  const prog = ease(frame, q2b + 2, q2b + 40);
  const total = 2417;
  const done = Math.round(prog * total);

  const pauseIn = r(pauseAt);
  const paused = frame >= pauseAt && frame < resumeAt;
  const queued = Math.min(37, Math.max(0, Math.floor((frame - waitAt) / 2)));
  const resumed = ease(frame, resumeAt, resumeAt + 20);

  const portalIn = r(portalAt);
  const linkIn = r(linkAt);

  // The parcel row moves up-left once the terminal comes in
  const shrink = ease(frame, repushAt - 4, repushAt + 16);

  return (
    <AbsoluteFill>
      <NightKicker label="NIGHT THREE · TUESDAY" clock="19:30" at={c(0)} />

      {/* Parcel row with weights */}
      <div
        style={{
          position: "absolute",
          left: 120,
          top: interpolate(shrink, [0, 1], [200, 160]),
          display: "flex",
          gap: 22,
          transform: `scale(${1 - shrink * 0.22})`,
          transformOrigin: "left top",
        }}
      >
        {PARCELS.map((p, i) => {
          const s = parcelsIn[i];
          const isBad = bug > 0.5 && fixed < 0.5;
          const isFixed = fixed > 0.5 && done > (i / PARCELS.length) * total;
          return (
            <div key={p} style={{ ...card, width: 250, padding: "16px 18px", display: "flex", alignItems: "center", gap: 14, opacity: s, transform: `translateY(${(1 - s) * 16}px)`, borderColor: isBad ? "rgba(229,72,77,0.55)" : isFixed ? "rgba(0,173,216,0.55)" : undefined }}>
              <Parcel size={48} color={isBad ? C.red : isFixed ? C.teal : C.coral} />
              <div>
                <div style={{ fontFamily: F.mono, fontSize: 19, fontWeight: 700, color: C.ink }}>{p}</div>
                <div style={{ fontFamily: F.mono, fontSize: 18, color: isBad ? C.red : isFixed ? C.tealDeep : "rgba(11,15,20,0.55)" }}>
                  weight_kg: {isBad ? "0" : isFixed ? "2.4" : "2.4"}
                </div>
              </div>
            </div>
          );
        })}
      </div>

      {/* Deploy tag + diff */}
      <div style={{ position: "absolute", left: 120, top: 330, display: "flex", alignItems: "center", gap: 18, opacity: diffIn * (1 - shrink * 0.4), transform: `translateY(${(1 - diffIn) * 12}px) scale(${1 - shrink * 0.15})`, transformOrigin: "left top" }}>
        <Chip tone="red" size={22}>deploy v2.3.1 · 19:30</Chip>
        <div style={{ fontFamily: F.mono, fontSize: 22, background: C.navy, color: C.cream, borderRadius: 12, padding: "10px 18px", lineHeight: 1.5 }}>
          <div style={{ color: "#ff8a80", opacity: 1 - fixed * 0.5 }}>- weight_kg: item.weight_g</div>
          <div style={{ color: "#8ce99a", opacity: 0.3 + fixed * 0.7 }}>+ weight_kg: item.weight_g / 1000</div>
        </div>
        <span style={{ opacity: ease(frame, everyAt, everyAt + 8), fontFamily: F.mono, fontSize: 20, color: C.red }}>× every parcel, all day</span>
        <span style={{ opacity: fixed }}><Chip tone="teal" size={22}>✓ fixed</Chip></span>
      </div>

      {/* Re-push terminal */}
      {frame >= repushAt - 4 && (
        <div style={{ position: "absolute", left: 120, top: 440, width: 1000, opacity: termIn, transform: `translateY(${(1 - termIn) * 16}px)` }}>
          <Terminal title="re-push Tuesday" fontSize={21} style={{ minHeight: 230 }}>
            <div style={{ whiteSpace: "pre-wrap", wordBreak: "break-all" }}>
              <span style={{ color: C.teal }}>$ </span>
              <TypeLine text="GET /v1/consumers/acme/events?event=item.shipped&created_after=2026-09-29&created_before=2026-09-29&prepare_repush=true" from={q1a} to={q1b} />
            </div>
            {resp > 0.01 && (
              <div style={{ opacity: resp, color: "rgba(232,230,225,0.85)" }}>
                {'{ "repush_id": '}<span style={{ color: C.teal }}>"rp_51c0…"</span>{', "total": '}<span style={{ color: C.coral, fontWeight: 700 }}>2417</span>{" }"}
              </div>
            )}
            {frame >= q2a && (
              <div style={{ marginTop: 8, whiteSpace: "pre-wrap", wordBreak: "break-all" }}>
                <span style={{ color: C.teal }}>$ </span>
                <TypeLine text={`POST /v1/consumers/acme/events:rePush  {"repush_id":"rp_51c0…"}`} from={q2a} to={q2b} />
              </div>
            )}
            {frame >= q2b + 2 && (
              <div style={{ marginTop: 10, display: "flex", alignItems: "center", gap: 16 }}>
                <div style={{ flex: 1, height: 14, borderRadius: 7, background: "rgba(244,242,237,0.15)", overflow: "hidden" }}>
                  <div style={{ width: `${prog * 100}%`, height: "100%", background: C.teal }} />
                </div>
                <span style={{ color: C.teal, fontWeight: 700, minWidth: 220 }}>event_repush · {done} / {total}{prog >= 1 ? " ✓" : ""}</span>
              </div>
            )}
          </Terminal>
        </div>
      )}

      {/* Pause / resume the partner */}
      {frame >= pauseAt - 4 && (
        <div style={{ position: "absolute", left: 1180, top: 440, opacity: pauseIn, transform: `translateX(${(1 - pauseIn) * 24}px)` }}>
          <Dock
            name="partner-dhl"
            sub={paused ? "mid-deploy · paused" : "resumed · draining"}
            width={620}
            state={paused ? "paused" : resumed > 0.5 ? "open" : "idle"}
            badge={paused ? <Chip tone="coral" size={22}>⏸ paused</Chip> : <Chip tone="teal" size={22}>▶ resumed</Chip>}
          />
          <div style={{ marginTop: 14, display: "flex", alignItems: "center", gap: 12, opacity: ease(frame, waitAt, waitAt + 8) }}>
            <div style={{ display: "flex", gap: 4 }}>
              {Array.from({ length: Math.round(queued * (1 - resumed)) }, (_, i) => (
                <div key={i} style={{ width: 10, height: 22, borderRadius: 3, background: C.coral, opacity: 0.8 }} />
              ))}
            </div>
            <span style={{ fontFamily: F.mono, fontSize: 20, color: C.ink }}>
              {resumed > 0.5 ? "0 waiting · delivered" : `${queued} deliveries waiting`}
            </span>
          </div>
          <div style={{ marginTop: 10, fontFamily: F.mono, fontSize: 17, color: "rgba(11,15,20,0.5)", opacity: ease(frame, pauseAt + 6, pauseAt + 14) }}>
            POST …/webhooks/{"{id}"}:pause  ·  :resume
          </div>
        </div>
      )}

      {/* Portal */}
      {frame >= portalAt - 4 && (
        <div style={{ ...card, position: "absolute", left: 1180, top: 660, width: 620, padding: 0, overflow: "hidden", opacity: portalIn, transform: `translateY(${(1 - portalIn) * 20}px)` }}>
          <div style={{ display: "flex", alignItems: "center", gap: 10, padding: "10px 16px", background: "#EDEBE6", borderBottom: "1px solid rgba(11,15,20,0.1)" }}>
            {["#FF5F57", "#FEBC2E", "#28C840"].map((cc) => <span key={cc} style={{ width: 10, height: 10, borderRadius: 99, background: cc }} />)}
            <span style={{ marginLeft: 10, fontFamily: F.mono, fontSize: 16, color: "rgba(11,15,20,0.6)" }}>sparrow.acme.example/portal#token=spt_…</span>
          </div>
          <div style={{ padding: "14px 20px", display: "flex", flexDirection: "column", gap: 8 }}>
            <div style={{ fontFamily: F.display, fontSize: 22, fontWeight: 700, color: C.ink }}>DHL · consumer portal</div>
            {[
              ["partner-dhl", "healthy", "128 retried · 2417 re-pushed"],
            ].map(([n, h, m]) => (
              <div key={n} style={{ display: "flex", alignItems: "center", gap: 12, fontFamily: F.mono, fontSize: 18 }}>
                <span style={{ fontWeight: 700 }}>{n}</span>
                <Chip tone="teal" size={15}>{h}</Chip>
                <span style={{ color: "rgba(11,15,20,0.55)" }}>{m}</span>
              </div>
            ))}
            <div style={{ opacity: linkIn, fontFamily: F.mono, fontSize: 16, color: C.tealDeep }}>one scoped, expiring link · they add endpoints, retry, inspect</div>
          </div>
        </div>
      )}

      <Sequence from={wrongAt}><Audio src={staticFile("sfx/impact-bass-1.mp3")} volume={0.16} /></Sequence>
      <Sequence from={q2b + 40}><Audio src={staticFile("sfx/chime.mp3")} volume={0.18} /></Sequence>
      <Sequence from={resumeAt}><Audio src={staticFile("sfx/ping.mp3")} volume={0.16} /></Sequence>
    </AbsoluteFill>
  );
};

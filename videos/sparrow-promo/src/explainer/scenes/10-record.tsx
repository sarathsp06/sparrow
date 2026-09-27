import React from "react";
import {
  AbsoluteFill,
  Sequence,
  interpolate,
  useCurrentFrame,
  staticFile,
} from "remotion";
import { Audio } from "@remotion/media";
import { C, F } from "../../theme";
import { cueFrame, useReveal, ease } from "../timing";
import { card } from "../ui";

const ID = "10-record";

// Delivery attempt rows — exactly as storyboard specifies.
const ROWS = [
  { n: 1, status: "503", time: "1.2s", category: "server_error", ok: false },
  { n: 2, status: "—", time: "30.0s", category: "timeout", ok: false },
  { n: 3, status: "connection refused", time: "—", category: "connection_refused", ok: false },
  { n: 4, status: "200", time: "84ms", category: "success", ok: true },
];

const COLS = ["#", "status", "time", "category"];
const COL_W = [60, 260, 120, 240];

const RETRY_CMD = "POST /v1/consumers/acme/deliveries/dlv_4e9…:retry";

export const Record: React.FC = () => {
  const frame = useCurrentFrame();
  const r = useReveal();

  const c0 = cueFrame(ID, 0);
  const c1 = cueFrame(ID, 1);
  const c2 = cueFrame(ID, 2);

  // Header / kicker
  const kickerIn = r(c0);
  const tableHeaderIn = r(c0 + 6);

  // Rows reveal one by one across c1 duration
  const rowDelay = 14; // frames between each row reveal
  const rowIn = (i: number) => r(c1 + i * rowDelay);

  // c2: retry command and fresh row
  const retryIn = r(c2);
  const retryRowIn = r(c2 + 22);

  // Table geometry
  const TABLE_X = 480;
  const TABLE_Y = 240;
  const TABLE_W = COL_W.reduce((a, b) => a + b, 0) + 80;
  const ROW_H = 62;
  const HEADER_H = 52;

  return (
    <AbsoluteFill>
      {/* Kicker */}
      <div
        style={{
          position: "absolute",
          top: 64,
          left: 72,
          display: "flex",
          alignItems: "center",
          gap: 14,
          fontFamily: F.mono,
          fontSize: 22,
          fontWeight: 500,
          letterSpacing: "0.16em",
          color: C.ink,
          whiteSpace: "pre",
          opacity: kickerIn,
          transform: `translateY(${(1 - kickerIn) * 12}px)`,
        }}
      >
        <span
          style={{
            width: 12,
            height: 12,
            borderRadius: 3,
            background: C.coral,
            boxShadow: "0 0 12px rgba(249,115,22,0.6)",
          }}
        />
        DELIVERY dlv_4e9…
      </div>

      {/* "Nothing is a mystery." hero text */}
      <div
        style={{
          position: "absolute",
          top: 140,
          left: 120,
          fontFamily: F.display,
          fontSize: 76,
          fontWeight: 700,
          letterSpacing: "-0.03em",
          lineHeight: 1.05,
          color: C.ink,
          opacity: kickerIn,
          transform: `translateY(${(1 - kickerIn) * 18}px)`,
          filter: `blur(${(1 - kickerIn) * 6}px)`,
        }}
      >
        Nothing is a mystery.
      </div>

      {/* Delivery attempts table */}
      <div
        style={{
          ...card,
          position: "absolute",
          left: TABLE_X,
          top: TABLE_Y,
          width: TABLE_W,
          padding: "0",
          overflow: "hidden",
          opacity: tableHeaderIn,
          transform: `translateY(${(1 - tableHeaderIn) * 14}px)`,
        }}
      >
        {/* Table header */}
        <div
          style={{
            display: "flex",
            alignItems: "center",
            height: HEADER_H,
            padding: "0 28px",
            borderBottom: "1px solid rgba(11,15,20,0.10)",
            fontFamily: F.mono,
            fontSize: 17,
            fontWeight: 700,
            letterSpacing: "0.08em",
            color: "rgba(11,15,20,0.45)",
          }}
        >
          {COLS.map((col, ci) => {
            // Highlight column headers as VO names them
            const highlight =
              ci >= 1
                ? ease(
                    frame,
                    c1 + 20 + (ci - 1) * 12,
                    c1 + 30 + (ci - 1) * 12,
                    0,
                    1,
                  )
                : 0;
            return (
              <div
                key={col}
                style={{
                  width: COL_W[ci],
                  color: interpolate(
                    highlight,
                    [0, 1],
                    [0.45, 1],
                    { extrapolateLeft: "clamp", extrapolateRight: "clamp" },
                  )
                    ? `rgba(11,15,20,${0.45 + highlight * 0.55})`
                    : "rgba(11,15,20,0.45)",
                }}
              >
                {col.toUpperCase()}
              </div>
            );
          })}
        </div>

        {/* Data rows */}
        {ROWS.map((row, i) => {
          const s = rowIn(i);
          return (
            <div
              key={row.n}
              style={{
                display: "flex",
                alignItems: "center",
                height: ROW_H,
                padding: "0 28px",
                borderBottom:
                  i < ROWS.length - 1
                    ? "1px solid rgba(11,15,20,0.06)"
                    : "none",
                fontFamily: F.mono,
                fontSize: 22,
                color: C.ink,
                opacity: s,
                transform: `translateY(${(1 - s) * 10}px)`,
              }}
            >
              <div style={{ width: COL_W[0], color: "rgba(11,15,20,0.4)" }}>
                #{row.n}
              </div>
              <div
                style={{
                  width: COL_W[1],
                  color: row.ok ? C.teal : C.red,
                  fontWeight: 600,
                }}
              >
                {row.status}
                {row.ok && (
                  <span style={{ marginLeft: 8, fontSize: 18 }}>✓</span>
                )}
              </div>
              <div
                style={{
                  width: COL_W[2],
                  color: "rgba(11,15,20,0.6)",
                }}
              >
                {row.time}
              </div>
              <div
                style={{
                  width: COL_W[3],
                  fontSize: 19,
                  color: row.ok
                    ? C.tealDeep
                    : C.coralText,
                  fontWeight: 500,
                }}
              >
                {row.category}
              </div>
            </div>
          );
        })}

        {/* Retry row #5 */}
        <div
          style={{
            display: "flex",
            alignItems: "center",
            height: ROW_H,
            padding: "0 28px",
            fontFamily: F.mono,
            fontSize: 22,
            color: C.ink,
            opacity: retryRowIn,
            transform: `translateY(${(1 - retryRowIn) * 10}px)`,
            borderTop: "1px solid rgba(11,15,20,0.06)",
            background: "rgba(0,173,216,0.04)",
          }}
        >
          <div style={{ width: COL_W[0], color: "rgba(11,15,20,0.4)" }}>
            #5
          </div>
          <div style={{ width: COL_W[1], color: C.teal, fontWeight: 600 }}>
            200 <span style={{ fontSize: 18 }}>✓</span>
          </div>
          <div style={{ width: COL_W[2], color: "rgba(11,15,20,0.6)" }}>
            62ms
          </div>
          <div
            style={{
              width: COL_W[3],
              fontSize: 19,
              color: C.tealDeep,
              fontWeight: 500,
            }}
          >
            success
          </div>
        </div>
      </div>

      {/* Retry command chip */}
      <div
        style={{
          position: "absolute",
          left: TABLE_X,
          top: TABLE_Y + HEADER_H + ROWS.length * ROW_H + ROW_H + 40,
          fontFamily: F.mono,
          fontSize: 20,
          color: C.ink,
          opacity: retryIn,
          transform: `translateY(${(1 - retryIn) * 14}px)`,
          display: "flex",
          alignItems: "center",
          gap: 14,
        }}
      >
        <div
          style={{
            padding: "14px 24px",
            borderRadius: 12,
            background: C.navy,
            color: "#E8E6E1",
            boxShadow: "0 12px 32px rgba(11,15,20,0.18)",
            display: "flex",
            alignItems: "center",
            gap: 12,
          }}
        >
          <span style={{ color: C.teal }}>$</span>
          {RETRY_CMD}
        </div>
        {/* Button press indicator */}
        <div
          style={{
            width: 42,
            height: 42,
            borderRadius: 10,
            background: C.teal,
            display: "grid",
            placeItems: "center",
            color: "#fff",
            fontSize: 22,
            fontWeight: 700,
            boxShadow: `0 6px 18px rgba(0,173,216,0.35)`,
            opacity: retryIn,
            transform: `scale(${0.85 + retryIn * 0.15})`,
          }}
        >
          ↻
        </div>
      </div>

      {/* SFX: ping on the success row (#4) landing */}
      <Sequence from={c1 + 3 * rowDelay}>
        <Audio src={staticFile("sfx/ping.mp3")} volume={0.2} />
      </Sequence>

      {/* SFX: ping on the retry row (#5) landing */}
      <Sequence from={c2 + 22}>
        <Audio src={staticFile("sfx/ping.mp3")} volume={0.22} />
      </Sequence>
    </AbsoluteFill>
  );
};

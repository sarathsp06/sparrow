import React from "react";
import {
  AbsoluteFill,
  Sequence,
  interpolate,
  useCurrentFrame,
  staticFile,
  Easing,
} from "remotion";
import { Audio } from "@remotion/media";
import { C, F } from "../../theme";
import { cueFrame, useReveal } from "../timing";
import { WordsIn } from "../ui";

const ID = "12a-open";

// -- Animated coral strike-through line over a ghost tile --
const StrikeThrough: React.FC<{
  frame: number;
  at: number;
  width: number;
}> = ({ frame, at, width }) => {
  const p = interpolate(frame, [at, at + 10], [0, 1], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
    easing: Easing.out(Easing.cubic),
  });
  if (p <= 0) return null;
  return (
    <svg
      width={width + 24}
      height={10}
      style={{
        position: "absolute",
        top: "50%",
        left: -12,
        marginTop: -5,
        pointerEvents: "none",
      }}
    >
      <line
        x1={0}
        y1={5}
        x2={(width + 24) * p}
        y2={5}
        stroke={C.coral}
        strokeWidth={3.5}
        strokeLinecap="round"
      />
    </svg>
  );
};

// -- A ghost tile: faded label that gets struck through --
const GhostTile: React.FC<{
  label: string;
  frame: number;
  appearAt: number;
  strikeAt: number;
  width: number;
  style?: React.CSSProperties;
}> = ({ label, frame, appearAt, strikeAt, width, style }) => {
  const r = useReveal();
  const showS = r(appearAt);
  const dimP = interpolate(frame, [strikeAt, strikeAt + 14], [1, 0.4], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
  });
  return (
    <div
      style={{
        position: "relative",
        padding: "14px 28px",
        borderRadius: 14,
        background: "rgba(255,255,255,0.48)",
        border: "1px solid rgba(11,15,20,0.10)",
        fontFamily: F.mono,
        fontSize: 24,
        fontWeight: 500,
        color: C.ink,
        whiteSpace: "nowrap",
        opacity: showS * dimP,
        transform: `translateY(${(1 - showS) * 14}px)`,
        width,
        textAlign: "center",
        ...style,
      }}
    >
      {label}
      <StrikeThrough frame={frame} at={strikeAt} width={width} />
    </div>
  );
};

// -- PostgreSQL database drum shape (cylinder) --
const PgDrum: React.FC<{
  growAt: number;
  queueAt: number;
}> = ({ growAt, queueAt }) => {
  const r = useReveal();
  const drumS = r(growAt, 26);
  const queueS = r(queueAt, 22);

  const W = 280;
  const H = 200;
  const ELLIPSE_RY = 28;
  const cx = W / 2;

  return (
    <div
      style={{
        position: "relative",
        width: W,
        height: H + ELLIPSE_RY,
        opacity: drumS,
        transform: `scale(${0.85 + drumS * 0.15}) translateY(${(1 - drumS) * 20}px)`,
      }}
    >
      <svg width={W} height={H + ELLIPSE_RY * 2} style={{ position: "absolute", top: 0, left: 0 }}>
        {/* Drum body */}
        <rect
          x={0}
          y={ELLIPSE_RY}
          width={W}
          height={H}
          fill="rgba(0,173,216,0.10)"
          stroke={C.teal}
          strokeWidth={2}
        />
        {/* Bottom ellipse */}
        <ellipse
          cx={cx}
          cy={ELLIPSE_RY + H}
          rx={cx}
          ry={ELLIPSE_RY}
          fill="rgba(0,173,216,0.08)"
          stroke={C.teal}
          strokeWidth={2}
        />
        {/* Top ellipse */}
        <ellipse
          cx={cx}
          cy={ELLIPSE_RY}
          rx={cx}
          ry={ELLIPSE_RY}
          fill="rgba(0,173,216,0.16)"
          stroke={C.teal}
          strokeWidth={2.5}
        />
        {/* Cover the stroke between body and top/bottom ellipses */}
        <rect
          x={1}
          y={ELLIPSE_RY}
          width={W - 2}
          height={H}
          fill="rgba(0,173,216,0.10)"
        />
      </svg>

      {/* PostgreSQL label */}
      <div
        style={{
          position: "absolute",
          top: ELLIPSE_RY + 36,
          width: "100%",
          textAlign: "center",
          fontFamily: F.mono,
          fontSize: 28,
          fontWeight: 700,
          color: C.tealDeep,
          letterSpacing: "0.02em",
        }}
      >
        PostgreSQL
      </div>

      {/* River queue inner layer */}
      <div
        style={{
          position: "absolute",
          top: ELLIPSE_RY + 90,
          left: 30,
          right: 30,
          height: 56,
          borderRadius: 10,
          background: `rgba(0,173,216,${0.08 + queueS * 0.10})`,
          border: `1.5px solid rgba(0,173,216,${queueS * 0.5})`,
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          fontFamily: F.mono,
          fontSize: 18,
          fontWeight: 600,
          color: C.tealDeep,
          opacity: queueS,
          transform: `translateY(${(1 - queueS) * 8}px)`,
        }}
      >
        River queue
      </div>
    </div>
  );
};

export const Open: React.FC = () => {
  const frame = useCurrentFrame();
  const r = useReveal();

  const c0 = cueFrame(ID, 0);
  const c1 = cueFrame(ID, 1);
  const c2 = cueFrame(ID, 2);

  // c1 word offsets (dur=4.696s at 30fps = ~141 frames)
  // "Just PostgreSQL." -> start of c1
  // "The queue lives there too." -> ~0.8s offset
  const queueOffset = 0.8;
  const queueAt = cueFrame(ID, 1, queueOffset);
  // "No Redis," -> ~2.4s offset
  const noRedisAt = cueFrame(ID, 1, 2.4);
  // "no broker." -> ~3.3s offset
  const noBrokerAt = cueFrame(ID, 1, 3.3);
  // Third ghost "worker fleet" struck on same beat as "no broker"
  const noWorkerAt = cueFrame(ID, 1, 3.6);

  // Ghost tiles appear with the drum (c1) but get struck at different beats
  const ghostAppearAt = c1 + 8;

  // c2 word offsets (dur=5.421s at 30fps = ~163 frames)
  // "And it's MIT licensed:" -> start of c2
  const mitAt = c2 + 4;
  // "free to run," -> ~1.4s
  const runAt = cueFrame(ID, 2, 1.4);
  // "fork" -> ~2.2s
  const forkAt = cueFrame(ID, 2, 2.2);
  // "and ship." -> ~2.8s
  const shipAt = cueFrame(ID, 2, 2.8);
  // "No per-message pricing." -> ~3.4s
  const pricingAt = cueFrame(ID, 2, 3.4);
  const pricingStrikeAt = cueFrame(ID, 2, 3.8);

  // MIT badge
  const mitS = r(mitAt, 24);

  // run / fork / ship verbs
  const runS = r(runAt);
  const forkS = r(forkAt);
  const shipS = r(shipAt);
  const verbs = [
    { text: "run", s: runS },
    { text: "fork", s: forkS },
    { text: "ship", s: shipS },
  ];

  return (
    <AbsoluteFill>
      {/* c0: Question text */}
      <div
        style={{
          position: "absolute",
          top: 130,
          width: "100%",
          textAlign: "center",
        }}
      >
        <WordsIn
          text="So what does it need to run?"
          at={c0}
          span={12}
          emphasis={["run"]}
          emphasisColor={C.coral}
          style={{
            fontFamily: F.display,
            fontSize: 88,
            fontWeight: 700,
            letterSpacing: "-0.03em",
            lineHeight: 1.1,
            color: C.ink,
          }}
        />
      </div>

      {/* c1: PostgreSQL drum — centre-left area */}
      <div
        style={{
          position: "absolute",
          left: 280,
          top: 340,
        }}
      >
        <PgDrum growAt={c1} queueAt={queueAt} />
      </div>

      {/* c1: Ghost tiles — right side, stacked */}
      <div
        style={{
          position: "absolute",
          right: 260,
          top: 360,
          display: "flex",
          flexDirection: "column",
          gap: 16,
          alignItems: "flex-end",
        }}
      >
        <GhostTile
          label="Redis"
          frame={frame}
          appearAt={ghostAppearAt}
          strikeAt={noRedisAt}
          width={160}
        />
        <GhostTile
          label="message broker"
          frame={frame}
          appearAt={ghostAppearAt + 4}
          strikeAt={noBrokerAt}
          width={240}
        />
        <GhostTile
          label="worker fleet"
          frame={frame}
          appearAt={ghostAppearAt + 8}
          strikeAt={noWorkerAt}
          width={200}
        />
      </div>

      {/* c2: MIT License badge — teal seal style */}
      <div
        style={{
          position: "absolute",
          left: 680,
          top: 400,
          display: "flex",
          flexDirection: "column",
          alignItems: "center",
          gap: 18,
          opacity: mitS,
          transform: `scale(${0.88 + mitS * 0.12})`,
        }}
      >
        <div
          style={{
            width: 160,
            height: 160,
            borderRadius: "50%",
            border: `3px solid ${C.teal}`,
            background: "rgba(0,173,216,0.08)",
            boxShadow: `0 0 ${24 * mitS}px rgba(0,173,216,0.25)`,
            display: "flex",
            flexDirection: "column",
            alignItems: "center",
            justifyContent: "center",
            fontFamily: F.display,
          }}
        >
          <div
            style={{
              fontSize: 42,
              fontWeight: 700,
              color: C.tealDeep,
              letterSpacing: "-0.02em",
              lineHeight: 1,
            }}
          >
            MIT
          </div>
          <div
            style={{
              fontSize: 16,
              fontWeight: 600,
              color: C.tealDeep,
              letterSpacing: "0.08em",
              marginTop: 4,
            }}
          >
            LICENSE
          </div>
        </div>

        {/* run / fork / ship */}
        <div
          style={{
            display: "flex",
            gap: 8,
            alignItems: "center",
            fontFamily: F.mono,
            fontSize: 24,
            fontWeight: 600,
            color: C.ink,
          }}
        >
          {verbs.map((v, i) => (
            <React.Fragment key={v.text}>
              {i > 0 && (
                <span
                  style={{
                    opacity: v.s * 0.5,
                    color: "rgba(11,15,20,0.35)",
                    fontSize: 20,
                  }}
                >
                  {"·"}
                </span>
              )}
              <span
                style={{
                  opacity: v.s,
                  transform: `translateY(${(1 - v.s) * 10}px)`,
                  display: "inline-block",
                }}
              >
                {v.text}
              </span>
            </React.Fragment>
          ))}
        </div>
      </div>

      {/* c2: "per-message pricing" chip, struck through */}
      <div
        style={{
          position: "absolute",
          right: 300,
          top: 680,
        }}
      >
        <GhostTile
          label="per-message pricing"
          frame={frame}
          appearAt={pricingAt > 0 ? pricingAt : c2}
          strikeAt={pricingStrikeAt}
          width={290}
          style={{
            fontSize: 22,
            padding: "12px 24px",
          }}
        />
      </div>

      {/* SFX: impact on MIT stamp */}
      <Sequence from={mitAt}>
        <Audio src={staticFile("sfx/impact-bass-1.mp3")} volume={0.22} />
      </Sequence>

      {/* SFX: whoosh on strike-throughs */}
      <Sequence from={noRedisAt}>
        <Audio src={staticFile("sfx/whoosh-short.mp3")} volume={0.2} />
      </Sequence>
    </AbsoluteFill>
  );
};

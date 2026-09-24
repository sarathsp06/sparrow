import React from "react";
import { interpolate, Easing } from "remotion";
import { C } from "../theme";

type Pt = { x: number; y: number };

// Horizontal S-curve between two points (cubic bezier, tangents flat at both ends).
const controls = (a: Pt, b: Pt) => {
  const dx = (b.x - a.x) * 0.5;
  return [a, { x: a.x + dx, y: a.y }, { x: b.x - dx, y: b.y }, b] as const;
};

export const pointOnWire = (a: Pt, b: Pt, t: number): Pt => {
  const [p0, p1, p2, p3] = controls(a, b);
  const u = 1 - t;
  return {
    x: u * u * u * p0.x + 3 * u * u * t * p1.x + 3 * u * t * t * p2.x + t * t * t * p3.x,
    y: u * u * u * p0.y + 3 * u * u * t * p1.y + 3 * u * t * t * p2.y + t * t * t * p3.y,
  };
};

const pathD = (a: Pt, b: Pt) => {
  const [p0, p1, p2, p3] = controls(a, b);
  return `M ${p0.x} ${p0.y} C ${p1.x} ${p1.y}, ${p2.x} ${p2.y}, ${p3.x} ${p3.y}`;
};

// Dotted connector that draws in from `a` toward `b`, with dashes that keep
// flowing in the direction of travel so the diagram never sits still.
export const Wire: React.FC<{
  a: Pt;
  b: Pt;
  frame: number;
  drawStart: number;
  drawEnd: number;
  color?: string;
  width?: number;
  id: string;
}> = ({ a, b, frame, drawStart, drawEnd, color = "rgba(11,15,20,0.28)", width = 3, id }) => {
  const draw = interpolate(frame, [drawStart, drawEnd], [0, 1], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
    easing: Easing.bezier(0.33, 1, 0.68, 1),
  });
  if (draw <= 0) return null;
  const minX = Math.min(a.x, b.x);
  const span = Math.abs(b.x - a.x);
  const clipX = a.x <= b.x ? minX - 10 : b.x + span * (1 - draw) - 10;
  return (
    <>
      <clipPath id={`clip-${id}`}>
        <rect x={clipX} y={0} width={span * draw + 20} height={1080} />
      </clipPath>
      <path
        d={pathD(a, b)}
        fill="none"
        stroke={color}
        strokeWidth={width}
        strokeDasharray="2 10"
        strokeDashoffset={-frame * 0.9}
        strokeLinecap="round"
        clipPath={`url(#clip-${id})`}
      />
    </>
  );
};

// A glowing packet with a fading tail, travelling along a Wire.
export const Comet: React.FC<{
  a: Pt;
  b: Pt;
  frame: number;
  start: number;
  end: number;
  color?: string;
  r?: number;
}> = ({ a, b, frame, start, end, color = C.coral, r = 9 }) => {
  if (frame < start || frame > end + 6) return null;
  const t = interpolate(frame, [start, end], [0, 1], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
    easing: Easing.bezier(0.45, 0, 0.55, 1),
  });
  const fade = interpolate(frame, [end, end + 6], [1, 0], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
  });
  const tail = [0, 0.035, 0.07, 0.105, 0.14, 0.175];
  return (
    <g opacity={fade} style={{ filter: `drop-shadow(0 0 8px ${color})` }}>
      {tail.map((d, i) => {
        const p = pointOnWire(a, b, Math.max(0, t - d));
        return (
          <circle
            key={i}
            cx={p.x}
            cy={p.y}
            r={r * (1 - i * 0.14)}
            fill={color}
            opacity={i === 0 ? 1 : 0.5 - i * 0.07}
          />
        );
      })}
    </g>
  );
};

// Expanding ring used when something lands (a delivery, a drop-in).
export const Ripple: React.FC<{
  x: number;
  y: number;
  frame: number;
  at: number;
  color?: string;
  size?: number;
}> = ({ x, y, frame, at, color = C.teal, size = 60 }) => {
  const p = interpolate(frame, [at, at + 20], [0, 1], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
    easing: Easing.out(Easing.cubic),
  });
  if (frame < at || p >= 1) return null;
  return (
    <circle cx={x} cy={y} r={8 + p * size} fill="none" stroke={color} strokeWidth={3 * (1 - p)} opacity={1 - p} />
  );
};

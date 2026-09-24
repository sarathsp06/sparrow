import React from "react";
import { linearTiming, TransitionSeries } from "@remotion/transitions";
import { slide } from "@remotion/transitions/slide";
import { Audio } from "@remotion/media";
import { AbsoluteFill, Sequence, staticFile } from "remotion";
import { Backdrop } from "./components/Backdrop";
import { blurFade } from "./components/blurFade";
import { FlyBy } from "./components/FlyBy";
import { Hook } from "./scenes/Hook";
import { OldWay } from "./scenes/OldWay";
import { Flow } from "./scenes/Flow";
import { Transform } from "./scenes/Transform";
import { Features } from "./scenes/Features";
import { Compare } from "./scenes/Compare";
import { Inside } from "./scenes/Inside";
import { Close } from "./scenes/Close";

// ~59.7s @ 30fps = 1790 frames.
// Sequence durations sum to 1790 + 7 transitions * 15 = 1895.
const T = 15;
const xfade = () => (
  <TransitionSeries.Transition
    presentation={blurFade()}
    timing={linearTiming({ durationInFrames: T })}
  />
);

export const DURATION = 1790;

// Scene transitions start at these absolute frames (sequence starts minus fade).
const WHOOSHES = [145, 370, 805, 1030, 1195, 1255, 1500];
// Flow deliveries land at flow start (370) + receiver `at` (200..290, step 18).
const PINGS = [570, 588, 606, 624, 642, 660];
// The sparrow darts across a few of the cuts, riding the whoosh.
const FLYBYS = [
  { at: 370, y: 180 },
  { at: 1030, y: 760, reverse: true },
  { at: 1500, y: 300 },
];

export const SparrowPromo: React.FC = () => (
  <AbsoluteFill>
    {/* One persistent paper backdrop; cream scenes are transparent over it. */}
    <Backdrop />
    <TransitionSeries>
      <TransitionSeries.Sequence durationInFrames={160} name="Hook">
        <Hook />
      </TransitionSeries.Sequence>
      <TransitionSeries.Transition
        presentation={slide({ direction: "from-bottom" })}
        timing={linearTiming({ durationInFrames: T })}
      />
      <TransitionSeries.Sequence durationInFrames={240} name="OldWay">
        <OldWay />
      </TransitionSeries.Sequence>
      {xfade()}
      <TransitionSeries.Sequence durationInFrames={450} name="Flow">
        <Flow />
      </TransitionSeries.Sequence>
      {xfade()}
      <TransitionSeries.Sequence durationInFrames={240} name="Transform">
        <Transform />
      </TransitionSeries.Sequence>
      {xfade()}
      <TransitionSeries.Sequence durationInFrames={180} name="Features">
        <Features />
      </TransitionSeries.Sequence>
      {xfade()}
      <TransitionSeries.Sequence durationInFrames={75} name="Inside">
        <Inside />
      </TransitionSeries.Sequence>
      {xfade()}
      <TransitionSeries.Sequence durationInFrames={260} name="Compare">
        <Compare />
      </TransitionSeries.Sequence>
      {xfade()}
      <TransitionSeries.Sequence durationInFrames={290} name="Close">
        <Close />
      </TransitionSeries.Sequence>
    </TransitionSeries>

    {FLYBYS.map((f) => (
      <Sequence key={f.at} from={f.at} durationInFrames={26} name="Sparrow fly-by">
        <FlyBy y={f.y} reverse={f.reverse} />
      </Sequence>
    ))}

    {/* Sound design (no music bed): subtle, low in the mix */}
    <Audio src={staticFile("sfx/impact-bass-1.mp3")} volume={0.5} name="Hook hit" />
    {WHOOSHES.map((at) => (
      <Sequence key={at} from={at} name="Whoosh">
        <Audio src={staticFile("sfx/whoosh-short.mp3")} volume={0.35} />
      </Sequence>
    ))}
    {PINGS.map((at) => (
      <Sequence key={at} from={at} name="Delivery ping">
        <Audio src={staticFile("sfx/ping.mp3")} volume={0.25} />
      </Sequence>
    ))}
    <Sequence from={428} name="Sparrow drop thud">
      <Audio src={staticFile("sfx/impact-bass-1.mp3")} volume={0.35} />
    </Sequence>
    <Sequence from={1650} name="Typing">
      <Audio src={staticFile("sfx/typing.mp3")} volume={0.4} />
    </Sequence>
    <Sequence from={1715} name="Close chime">
      <Audio src={staticFile("sfx/chime.mp3")} volume={0.45} />
    </Sequence>
  </AbsoluteFill>
);

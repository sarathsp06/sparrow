import "./index.css";
import { Composition, Folder } from "remotion";
import { SparrowPromo, DURATION } from "./SparrowPromo";
import { Hook } from "./scenes/Hook";
import { OldWay } from "./scenes/OldWay";
import { Flow } from "./scenes/Flow";
import { Transform } from "./scenes/Transform";
import { Features } from "./scenes/Features";
import { Inside } from "./scenes/Inside";
import { Compare } from "./scenes/Compare";
import { Close } from "./scenes/Close";
import { withBackdrop } from "./components/Backdrop";

const OldWayB = withBackdrop(OldWay);
const FlowB = withBackdrop(Flow);
const TransformB = withBackdrop(Transform);
const FeaturesB = withBackdrop(Features);
const InsideB = withBackdrop(Inside);
const CompareB = withBackdrop(Compare);
const CloseB = withBackdrop(Close);

export const RemotionRoot: React.FC = () => {
  return (
    <>
      <Composition
        id="SparrowPromo"
        component={SparrowPromo}
        durationInFrames={DURATION}
        fps={30}
        width={1920}
        height={1080}
      />
      <Folder name="Scenes">
        <Composition
          id="Hook"
          component={Hook}
          durationInFrames={160}
          fps={30}
          width={1920}
          height={1080}
        />
        <Composition
          id="OldWay"
          component={OldWayB}
          durationInFrames={240}
          fps={30}
          width={1920}
          height={1080}
        />
        <Composition
          id="Flow"
          component={FlowB}
          durationInFrames={450}
          fps={30}
          width={1920}
          height={1080}
        />
        <Composition
          id="Transform"
          component={TransformB}
          durationInFrames={240}
          fps={30}
          width={1920}
          height={1080}
        />
        <Composition
          id="Features"
          component={FeaturesB}
          durationInFrames={180}
          fps={30}
          width={1920}
          height={1080}
        />
        <Composition
          id="Inside"
          component={InsideB}
          durationInFrames={75}
          fps={30}
          width={1920}
          height={1080}
        />
        <Composition
          id="Compare"
          component={CompareB}
          durationInFrames={260}
          fps={30}
          width={1920}
          height={1080}
        />
        <Composition
          id="Close"
          component={CloseB}
          durationInFrames={290}
          fps={30}
          width={1920}
          height={1080}
        />
      </Folder>
    </>
  );
};

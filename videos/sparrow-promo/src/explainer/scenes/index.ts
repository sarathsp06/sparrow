import type React from "react";
import { Hook } from "./01-hook";
import { Pain } from "./02-pain";
import { Meet } from "./03-meet";
import { Push } from "./04-push";
import { FanOut } from "./05-fanout";
import { Sign } from "./06-sign";
import { Recipes } from "./06b-recipes";
import { Classify } from "./07-classify";
import { Backoff } from "./08-backoff";
import { Health } from "./09-health";
import { Record } from "./10-record";
import { Dashboard } from "./10b-dashboard";
import { Takeaway } from "./11-takeaway";
import { Open } from "./12a-open";
import { Compare } from "./12b-compare";
import { Cta } from "./12-cta";

// Order must match script.json / timing.json frames.
export const SCENES: React.FC[] = [
  Hook, Pain, Meet, Push, FanOut, Sign, Recipes, Classify, Backoff, Health, Record, Dashboard, Open, Compare, Takeaway, Cta,
];

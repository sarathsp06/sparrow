import type React from "react";
import { Hook } from "./01-hook";
import { Pain } from "./02-pain";
import { Three } from "./03-three";
import { Happy } from "./04-happy";
import { Verify } from "./05-verify";
import { Night } from "./06-night";
import { Tuesday } from "./07-tuesday";
import { Recipes } from "./08-recipes";
import { Seeing } from "./09-seeing";
import { Needs } from "./10-needs";
import { Compare } from "./11-compare";
import { Takeaway } from "./12-takeaway";
import { Cta } from "./13-cta";

// Order must match script.json / timing.json frames.
export const SCENES: React.FC[] = [Hook, Pain, Three, Happy, Verify, Night, Tuesday, Recipes, Seeing, Needs, Compare, Takeaway, Cta];

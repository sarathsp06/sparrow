// Shipped adapter recipes, bundled from satellites/recipes/*.yaml at build
// time. A recipe is pure pre-fill: destination (webhook) + transform template.
import { load } from 'js-yaml';

export interface RecipeParam {
  name: string;
  prompt?: string;
  required?: boolean;
}

export interface Recipe {
  version: number;
  name: string;
  description: string;
  params?: RecipeParam[];
  webhook: {
    url: string;
    headers?: Record<string, string>;
    secret_headers?: Record<string, string>;
  };
  subscription?: {
    transform_template?: string;
  };
}

const files = import.meta.glob('../../../satellites/recipes/*.yaml', {
  eager: true,
  query: '?raw',
  import: 'default',
}) as Record<string, string>;

export const recipes: Recipe[] = Object.values(files)
  .map((raw) => load(raw) as Recipe)
  .filter((r) => r?.version === 1 && !!r.name && !!r.webhook?.url)
  .sort((a, b) => a.name.localeCompare(b.name));

/** Replace {{param "name"}} placeholders — same substitution the CLI applies. */
export function substituteParams(s: string, params: Record<string, string>): string {
  return s.replace(/\{\{\s*param\s+"([^"]+)"\s*\}\}/g, (_, name: string) => params[name] ?? '');
}

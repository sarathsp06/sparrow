// Shipped adapter recipes are served by the Sparrow API. This file only keeps
// the small client-side type and param substitution helper.
export interface RecipeParam {
  name: string;
  /** Short label. */
  prompt?: string;
  required?: boolean;
  default?: string;
  secret?: boolean;
  /** One-line hint: what the value is and where to find it. */
  help?: string;
  /** Sample value, shown as the placeholder. */
  example?: string;
  /** The only accepted values; rendered as a select. */
  enum?: string[] | null;
  docs_url?: string;
  activation_required?: boolean;
  must_override_default?: boolean;
}
export interface Recipe {
  version: number;
  name: string;
  description: string;
  params?: RecipeParam[] | null;
  /** The consumer the recipe is meant for, when it matters. */
  consumer?: { name: string; note: string } | null;
  webhook: {
    url: string;
    headers?: Record<string, string>;
    secret_headers?: Record<string, string>;
    requires_transform?: boolean;
  };
  subscription?: {
    transform_template?: string;
  };
}

/** Replace {{param "name"}} placeholders — same substitution the CLI applies. */
export function substituteParams(s: string, params: Record<string, string>): string {
  return s.replace(/\{\{\s*param\s+"([^"]+)"\s*\}\}/g, (_, name: string) => params[name] ?? '');
}

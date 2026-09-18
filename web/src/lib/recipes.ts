// Shipped adapter recipes are served by the Sparrow API. This file only keeps
// the small client-side type and param substitution helper.
export interface RecipeParam {
  name: string;
  prompt?: string;
  required?: boolean;
  default?: string;
  secret?: boolean;
  activation_required?: boolean;
  must_override_default?: boolean;
}
export interface Recipe {
  version: number;
  name: string;
  description: string;
  params?: RecipeParam[] | null;
  webhook: {
    url: string;
    headers?: Record<string, string>;
    secret_headers?: Record<string, string>;
  };
  subscription?: {
    transform_template?: string;
  };
}

/** Replace {{param "name"}} placeholders — same substitution the CLI applies. */
export function substituteParams(s: string, params: Record<string, string>): string {
  return s.replace(/\{\{\s*param\s+"([^"]+)"\s*\}\}/g, (_, name: string) => params[name] ?? '');
}

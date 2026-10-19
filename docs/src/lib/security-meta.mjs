// GitHub Pages cannot send custom response headers, so the docs carry the two
// that have a <meta> equivalent: Content-Security-Policy and Referrer-Policy.
// X-Frame-Options, X-Content-Type-Options and HSTS have none (github.io is on
// the HSTS preload list anyway), and a <meta> CSP ignores frame-ancestors.
//
// Inline scripts stay allowed: Starlight and the custom pages ship them unhashed.
// 'wasm-unsafe-eval' is for Pagefind search. Images are any https source because
// pages embed badges and remote screenshots. Applied by Starlight's `head` and by
// SecurityMeta.astro on the pages outside Starlight. /reference/api (Scalar) is
// not covered: it renders its own document and loads from cdn.jsdelivr.net.
export const contentSecurityPolicy = [
  "default-src 'self'",
  "script-src 'self' 'unsafe-inline' 'wasm-unsafe-eval'",
  "style-src 'self' 'unsafe-inline' https://fonts.googleapis.com",
  "font-src 'self' data: https://fonts.gstatic.com",
  "img-src 'self' data: https:",
  "media-src 'self'",
  'frame-src https://www.youtube-nocookie.com data:',
  "connect-src 'self'",
  "object-src 'none'",
  "base-uri 'self'",
  "form-action 'self'",
  'upgrade-insecure-requests',
].join('; ');

export const referrerPolicy = 'strict-origin-when-cross-origin';

// Starlight `head` entries.
export const securityHead = [
  { tag: 'meta', attrs: { 'http-equiv': 'Content-Security-Policy', content: contentSecurityPolicy } },
  { tag: 'meta', attrs: { name: 'referrer', content: referrerPolicy } },
];

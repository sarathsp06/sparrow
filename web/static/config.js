// Sparrow UI runtime configuration.
//
// Only needed when the UI is deployed on its own (a static site on a different
// host than the Sparrow server). Edit this file on the static host — no rebuild
// required. When the Sparrow server serves the UI itself (SPARROW_SERVE_UI=true)
// leave it as is: the UI talks to the same origin and asks you to sign in.
//
//   apiUrl  Absolute URL of the Sparrow server, e.g. "https://sparrow-api.example.com".
//           Overrides PUBLIC_API_URL baked in at build time. Also add the UI's
//           origin to CORS_ALLOWED_ORIGINS on the server.
//   apiKey  SPARROW_API_KEY of the server. Optional: if omitted and the server
//           requires a key, the UI asks for it and remembers it in this browser.
//           Anything placed here is readable by everyone who can load the UI
//           (including consumer portal visitors), so prefer the prompt.
window.__SPARROW_CONFIG__ = window.__SPARROW_CONFIG__ || {
  // apiUrl: "https://sparrow-api.example.com",
  // apiKey: "",
};

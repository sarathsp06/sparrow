# Sparrow Web Dashboard

SvelteKit web dashboard for the Sparrow webhook delivery system. Builds to static files and is embedded into the Go server binary.

## Prerequisites

- Node.js 18+
- npm

## Local Development

```bash
# Install dependencies
npm install

# Start dev server (default: http://localhost:5173)
npm run dev
```

The dev server connects to the Sparrow Go backend API. Make sure the backend is running (`make run` from the project root).

## Build

```bash
npm run build
```

Build output goes to `../internal/ui/dist/` (the Go embed directory). The static adapter produces a fully self-contained SPA with `index.html` as the fallback for client-side routing.

## Embedding in the Go Binary

1. Build the frontend: `npm run build` (or `make build-ui` from the project root)
2. Build the server: `go build ./cmd/server` (or `make build-with-ui` for both steps)
3. Run with `SPARROW_SERVE_UI=true` -- the UI is served at `http://localhost:8080/`

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `PUBLIC_API_URL` | `http://localhost:8080` | Sparrow backend API URL (Connect-RPC HTTP/JSON) |
| `PUBLIC_AUTH_PROVIDER` | *(auto-detect)* | Auth provider: `clerk`, `none`, or unset for auto-detect |
| `PUBLIC_CLERK_PUBLISHABLE_KEY` | *(unset)* | Clerk publishable key (enables Clerk auth when set) |

Set these in `web/.env` for development. When embedded in the Go binary, the UI uses relative paths (`/`) to call the API on the same origin. See the [Configuration Reference](../README.md#configuration-reference) for the full list of backend and frontend env vars.

## Project Structure

```
web/
├── src/
│   ├── app.html              # HTML shell
│   ├── app.css               # Global styles (Tailwind CSS 4)
│   ├── app.d.ts              # TypeScript declarations
│   ├── lib/
│   │   ├── auth.ts           # Provider-agnostic token abstraction
│   │   ├── services.ts       # Connect-RPC API clients + Bearer token interceptor
│   │   ├── utils.ts          # Utility functions
│   │   ├── auth/
│   │   │   ├── types.ts      # AuthProviderType, AuthProviderConfig
│   │   │   ├── provider.ts   # Provider detection from env vars
│   │   │   ├── AuthShell.svelte   # Dispatches to active provider shell
│   │   │   └── providers/
│   │   │       ├── clerk/
│   │   │       │   └── ClerkAuthShell.svelte   # Clerk sign-in, user button, org switcher
│   │   │       └── none/
│   │   │           └── NoAuthShell.svelte      # No-auth fallback (open access)
│   │   ├── components/       # Reusable Svelte components
│   │   └── assets/           # Static assets (favicon, etc.)
│   └── routes/               # SvelteKit pages
│       ├── +layout.svelte    # Root layout (delegates to AuthShell)
│       ├── +layout.ts        # Prerender + SPA config
│       ├── +page.svelte      # Home
│       ├── webhooks/         # Webhook management
│       ├── events/           # Event management
│       ├── health/           # Health dashboard
│       └── deliveries/       # Delivery details
├── svelte.config.js          # SvelteKit config (static adapter)
├── vite.config.ts            # Vite config
└── package.json
```

## Auth Provider System

The frontend uses a pluggable auth provider architecture. The root layout delegates to `AuthShell.svelte`, which renders the correct provider shell based on environment variables.

**Flow:** `+layout.svelte` -> `AuthShell.svelte` -> `ClerkAuthShell.svelte` or `NoAuthShell.svelte`

Each provider shell:
- Accepts `header` and `children` snippets from the layout
- Controls when page content renders (e.g., only after sign-in)
- Provides its own sign-in/sign-out UI
- Calls `registerTokenProvider()` from `auth.ts` to inject tokens into API requests

The services layer (`services.ts`) calls `getSessionToken()` to get a Bearer token for each API request. It never imports any provider SDK directly.

**Adding a new provider:** See the [README](../README.md#web-ui-authentication-pluggable-providers) for step-by-step instructions.

## Tech Stack

- **SvelteKit 2** + **Svelte 5** (Runes)
- **Vite 7**
- **Tailwind CSS 4** (via Vite plugin)
- **Flowbite Svelte** (UI components)
- **Connect-RPC** (API client for Go backend)
- **Static adapter** (embedded in Go binary via `go:embed`)
- **svelte-clerk** (Clerk integration, optional)

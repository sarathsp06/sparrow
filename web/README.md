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

Set this in `web/.env` for development. When embedded in the Go binary, the UI uses relative paths (`/`) to call the API on the same origin. See [CONFIGURATION.md](../CONFIGURATION.md) for the full list of backend env vars.

## Project Structure

```
web/
├── src/
│   ├── app.html              # HTML shell
│   ├── app.css               # Global styles (Tailwind CSS 4)
│   ├── app.d.ts              # TypeScript declarations
│   ├── lib/
│   │   ├── services.ts       # Connect-RPC API clients
│   │   ├── utils.ts          # Utility functions
│   │   ├── components/       # Reusable Svelte components
│   │   └── assets/           # Static assets (favicon, etc.)
│   └── routes/               # SvelteKit pages
│       ├── +layout.svelte    # Root layout (navigation)
│       ├── +layout.ts        # Prerender + SPA config
│       ├── +page.svelte      # Home / landing page
│       ├── webhooks/         # Webhook management
│       ├── events/           # Event management
│       ├── namespaces/       # Namespace management
│       ├── health/           # Health dashboard
│       └── deliveries/       # Delivery details
├── svelte.config.js          # SvelteKit config (static adapter)
├── vite.config.ts            # Vite config
└── package.json
```

## Tech Stack

- **SvelteKit 2** + **Svelte 5** (Runes)
- **Vite 7**
- **Tailwind CSS 4** (via Vite plugin)
- **Flowbite Svelte** (UI components)
- **Connect-RPC** (API client for Go backend)
- **Static adapter** (embedded in Go binary via `go:embed`)

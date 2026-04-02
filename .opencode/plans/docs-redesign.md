# Docs Redesign Plan

## Task 1: GitHub Pages Deployment
- Create `.github/workflows/deploy-docs.yml` using `withastro/action@v3`
- Triggers on push to main (paths: docs/**) + workflow_dispatch

## Task 2: Scalar-Inspired API Reference Redesign

### Data layer
- Add `example?: string` to `Field` in types.ts
- Make `examples` optional on `Rpc`
- Rewrite all 5 service data files: remove hand-written curl/Go, add example values to fields

### Components
- `ApiServicePage.astro` — Two-column layout, auto-generate curl + response JSON
- `ApiEndpoint.astro` — Method badge, cleaner header
- `ApiRequest.astro` — Compact inline property list
- `ApiResponse.astro` — Compact inline property list
- `ErrorCodes.astro` — Collapsible styling

### Styles
- `custom.css` — Two-column grid, Scalar-inspired visual treatment

## Status: APPROVED — ready for implementation

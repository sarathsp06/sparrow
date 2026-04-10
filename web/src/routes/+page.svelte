<script lang="ts">
  import favicon from "$lib/assets/favicon.svg";

  const features = [
    {
      icon: "M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z",
      title: "Guaranteed Delivery",
      description: "Events are persisted in PostgreSQL before delivery begins. The River job queue provides at-least-once delivery with configurable retries (default: 3 attempts).",
    },
    {
      icon: "M1 4v6h6M23 20v-6h-6M20.49 9A9 9 0 0 0 5.64 5.64L1 10m22 4l-4.64 4.36A9 9 0 0 1 3.51 15",
      title: "Intelligent Retries",
      description: "Exponential backoff with configurable intervals. Errors are classified into retryable (5xx, timeout, connection refused, network) and non-retryable (4xx, DNS, TLS).",
    },
    {
      icon: "M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0 1 12 2.944a11.955 11.955 0 0 1-8.618 3.04A12.02 12.02 0 0 0 3 12c0 3.209 1.262 6.126 3.318 8.28A11.966 11.966 0 0 0 12 23.056a11.966 11.966 0 0 0 5.682-2.776A12.018 12.018 0 0 0 21 12c0-1.395-.236-2.735-.673-3.984",
      title: "Idempotent Event Delivery",
      description: "Deduplicate events with optional idempotency keys. Duplicate pushes are detected and safely returned without re-processing -- critical for financial, healthcare, and compliance-sensitive workflows.",
    },
    {
      icon: "M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z",
      title: "gRPC + Connect-RPC",
      description: "Native gRPC on :50051 and Connect-RPC (HTTP/JSON) on :8080. Use curl, gRPC clients, or the built-in web dashboard -- same API, same behavior.",
    },
    {
      icon: "M14.7 6.3a1 1 0 0 0 0 1.4l1.6 1.6a1 1 0 0 0 1.4 0l3.77-3.77a6 6 0 0 1-7.94 7.94l-6.91 6.91a2.12 2.12 0 0 1-3-3l6.91-6.91a6 6 0 0 1 7.94-7.94l-3.76 3.76z",
      title: "Complete Observability",
      description: "OpenTelemetry traces, metrics, and structured logs exported via OTLP. Per-webhook health tracking with error category breakdown (client, server, timeout, network).",
    },
    {
      icon: "M12 2L2 7l10 5 10-5-10-5zM2 17l10 5 10-5M2 12l10 5 10-5",
      title: "Payload Transformation",
      description: "Go templates transform event payloads per subscription. If a transform fails, Sparrow delivers the original payload as an envelope -- never drops a delivery.",
    },
  ];

  const securityFeatures = [
    {
      icon: "M21 2l-2 2m-7.61 7.61a5.5 5.5 0 1 1-7.778 7.778 5.5 5.5 0 0 1 7.777-7.777zm0 0L15.5 7.5m0 0l3 3L22 7l-3-3m-3.5 3.5L19 4",
      title: "Envelope Encryption (AES-256-GCM)",
      description: "Webhook secrets and sensitive headers are encrypted at rest using envelope encryption. Each record gets a unique data encryption key (DEK), wrapped by a master key (KEK). DEKs are zeroed from memory after use.",
      tags: ["AES-256-GCM", "Per-Record DEK", "Key Rotation"],
    },
    {
      icon: "M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z",
      title: "HMAC-SHA256 Webhook Signing",
      description: "Every delivery is signed with a timestamp-prefixed HMAC-SHA256 signature using the webhook's secret. Recipients can verify authenticity and detect replay attacks via the X-Sparrow-Signature-256 and X-Sparrow-Timestamp headers.",
      tags: ["HMAC-SHA256", "Replay Protection", "Per-Webhook Secret"],
    },
    {
      icon: "M12 9v2m0 4h.01M3.09 10.64A9.97 9.97 0 0 1 12 2a9.97 9.97 0 0 1 8.91 8.64M20.42 15A10 10 0 0 1 3.58 15M21 19a2 2 0 1 1-4 0 2 2 0 0 1 4 0zM7 19a2 2 0 1 1-4 0 2 2 0 0 1 4 0zM14 19a2 2 0 1 1-4 0 2 2 0 0 1 4 0z",
      title: "SSRF Protection",
      description: "Two-layer defense against Server-Side Request Forgery: URL validation blocks private networks, cloud metadata endpoints, and localhost at redirect time. Dial-time IP validation prevents DNS rebinding attacks.",
      tags: ["DNS Rebinding", "Metadata Block", "IP Validation"],
    },
    {
      icon: "M15 3h4a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2h-4M10 17l5-5-5-5M13.8 12H3",
      title: "API Key Authentication",
      description: "Optional shared-secret authentication via X-API-Key header. Keys are compared using constant-time comparison (crypto/subtle) to prevent timing attacks. When unset, endpoints are open for internal deployments behind a VPN.",
      tags: ["Constant-Time", "Header + Query", "gRPC + HTTP"],
    },
    {
      icon: "M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0 1 12 2.944a11.955 11.955 0 0 1-8.618 3.04A12.02 12.02 0 0 0 3 12c0 3.209 1.262 6.126 3.318 8.28A11.966 11.966 0 0 0 12 23.056a11.966 11.966 0 0 0 5.682-2.776A12.018 12.018 0 0 0 21 12c0-1.395-.236-2.735-.673-3.984",
      title: "Security Headers",
      description: "Every HTTP response includes X-Content-Type-Options (nosniff), X-Frame-Options (DENY), strict Referrer-Policy, and Permissions-Policy. CORS is locked to explicit origins in production.",
      tags: ["XSS", "Clickjacking", "CORS"],
    },
    {
      icon: "M9 17v-2m3 2v-4m3 4v-6m2 10H7a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5.586a1 1 0 0 1 .707.293l5.414 5.414a1 1 0 0 1 .293.707V19a2 2 0 0 1-2 2z",
      title: "Audit-Ready & Fully Open Source",
      description: "All security controls are implemented in auditable Go code -- no black boxes. Organizations with HIPAA, SOC 2, or PCI DSS requirements can review, verify, and extend every layer themselves. No paid tier required.",
      tags: ["Audit-Friendly", "Compliance-Ready", "Full Audit"],
    },
  ];

  const steps = [
    {
      num: "01",
      title: "Define an Event",
      description: "Events describe what happens in your system. Register a type once, then emit it as many times as you need.",
      code: `curl -s -X POST http://localhost:8080/webhook.EventService/RegisterEvent \\
  -H "Content-Type: application/json" \\
  -d '{
    "name": "order.created",
    "description": "Fired when a new order is placed",
    "active": true
  }'`,
    },
    {
      num: "02",
      title: "Register a Webhook",
      description: "Point your endpoint at one or more events inside a namespace. Sparrow creates a subscription for each event automatically.",
      code: `curl -s -X POST http://localhost:8080/webhook.WebhookService/RegisterWebhook \\
  -H "Content-Type: application/json" \\
  -d '{
    "namespace": "default",
    "url": "https://testhooks.sarathsadasivan.com/hooks",
    "events": ["order.created"],
    "active": true
  }'`,
    },
    {
      num: "03",
      title: "Push an Event",
      description: "Emit an event with a JSON payload, optional labels, and an optional idempotency key. Sparrow matches subscriptions, fans out deliveries, and retries failures automatically. Duplicate pushes with the same id are detected and safely deduplicated.",
      code: `curl -s -X POST http://localhost:8080/webhook.EventService/PushEvent \\
  -H "Content-Type: application/json" \\
  -d '{
    "namespace": "default",
    "event": "order.created",
    "payload": {
      "order_id": "ord_123",
      "amount": 99.99
    },
    "labels": { "region": "us-east", "priority": "high" },
    "id": "idempotency-key-123"
  }'`,
    },
  ];

  const techStack = ["Go", "PostgreSQL", "gRPC", "SvelteKit", "OpenTelemetry"];

  const comparisonServices = [
    { key: "sparrow", name: "Sparrow", highlight: true },
    { key: "svix", name: "Svix" },
    { key: "convoy", name: "Convoy" },
    { key: "hookdeck", name: "Hookdeck" },
    { key: "aws", name: "AWS EventBridge / SNS" },
    { key: "diy", name: "DIY" },
  ];

  const comparisonFeatures = [
    {
      feature: "Open Source",
      sparrow: true, svix: "Partial", convoy: true, hookdeck: false, aws: false, diy: "N/A",
    },
    {
      feature: "Self-Hosted",
      sparrow: true, svix: true, convoy: true, hookdeck: false, aws: false, diy: true,
    },
    {
      feature: "No Per-Message Pricing",
      sparrow: true, svix: false, convoy: true, hookdeck: false, aws: false, diy: true,
    },
    {
      feature: "Encryption at Rest (Secrets)",
      sparrow: true, svix: "Enterprise", convoy: false, hookdeck: "Managed", aws: "Managed", diy: false,
    },
    {
      feature: "SSRF Protection",
      sparrow: true, svix: true, convoy: "Partial", hookdeck: true, aws: "N/A", diy: false,
    },
    {
      feature: "Minimal Infra (Binary + Postgres)",
      sparrow: true, svix: "SaaS", convoy: false, hookdeck: "SaaS", aws: "Managed", diy: "Varies",
    },
    {
      feature: "gRPC / Connect-RPC Native",
      sparrow: true, svix: false, convoy: false, hookdeck: false, aws: false, diy: false,
    },
    {
      feature: "Payload Transformation",
      sparrow: true, svix: "Paid", convoy: true, hookdeck: true, aws: "Partial", diy: false,
    },
    {
      feature: "Guaranteed Delivery & Retries",
      sparrow: true, svix: true, convoy: true, hookdeck: true, aws: "Partial", diy: false,
    },
    {
      feature: "Webhook Health Tracking",
      sparrow: true, svix: true, convoy: true, hookdeck: true, aws: false, diy: false,
    },
    {
      feature: "OpenTelemetry Native",
      sparrow: true, svix: "Enterprise", convoy: false, hookdeck: false, aws: "X-Ray", diy: false,
    },
    {
      feature: "HMAC Signing",
      sparrow: true, svix: true, convoy: true, hookdeck: true, aws: false, diy: false,
    },
    {
      feature: "Fully Auditable Security Code",
      sparrow: true, svix: "Partial", convoy: true, hookdeck: false, aws: false, diy: true,
    },
    {
      feature: "Idempotent Event Ingestion",
      sparrow: true, svix: true, convoy: false, hookdeck: true, aws: false, diy: false,
    },
    {
      feature: "No Vendor Lock-In",
      sparrow: true, svix: "Partial", convoy: true, hookdeck: false, aws: false, diy: true,
    },
    {
      feature: "Multi-Language Client SDKs",
      sparrow: true, svix: true, convoy: true, hookdeck: true, aws: true, diy: false,
    },
  ];
</script>

<svelte:head>
  <title>Sparrow - Secure Event-Driven Webhook Delivery System</title>
  <meta name="description" content="Production-ready webhook delivery with envelope encryption at rest, HMAC signing, SSRF protection, guaranteed delivery, and complete observability. Fully open source." />
</svelte:head>

<div class="font-inter min-h-screen bg-white text-gray-900">
  <!-- ============================================================ -->
  <!-- HEADER -->
  <!-- ============================================================ -->
  <header class="fixed top-0 left-0 right-0 z-50 border-b border-gray-200/60 bg-white/80 backdrop-blur-xl">
    <div class="max-w-[1400px] mx-auto px-6 flex items-center justify-between h-16">
      <a href="/" class="flex items-center gap-3">
        <img src={favicon} alt="Sparrow" class="w-8 h-8" />
        <span class="text-xl font-bold font-fira text-gray-900">Sparrow</span>
      </a>
      <nav class="hidden md:flex items-center gap-8">
        <a href="#features" class="text-sm text-gray-500 hover:text-teal-600 transition-colors">Features</a>
        <a href="#security" class="text-sm text-gray-500 hover:text-teal-600 transition-colors">Security</a>
        <a href="#getting-started" class="text-sm text-gray-500 hover:text-teal-600 transition-colors">Getting Started</a>
        <a href="#comparison" class="text-sm text-gray-500 hover:text-teal-600 transition-colors">Comparison</a>
        <a href="https://sarathsp06.github.io/sparrow/guides/go/" target="_blank" class="text-sm text-gray-500 hover:text-teal-600 transition-colors">SDKs</a>
        <a href="https://sarathsp06.github.io/sparrow/" target="_blank" class="text-sm text-gray-500 hover:text-teal-600 transition-colors">Documentation</a>
      </nav>
      <div class="flex items-center gap-3">
        <a
          href="https://github.com/sarathsp06/sparrow"
          target="_blank"
          class="hidden md:inline-flex items-center gap-2 px-4 py-2 text-sm rounded-lg border border-gray-200 text-gray-600 hover:bg-gray-50 transition-all"
        >
          <svg class="w-4 h-4" viewBox="0 0 24 24" fill="currentColor"><path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0 0 24 12c0-6.63-5.37-12-12-12z"/></svg>
          GitHub
        </a>
        <a href="/webhooks" class="px-4 py-2 text-sm font-medium rounded-lg bg-teal-500 text-white hover:bg-teal-600 transition-all">
          Open Dashboard
        </a>
      </div>
    </div>
  </header>

  <!-- ============================================================ -->
  <!-- HERO SECTION -->
  <!-- ============================================================ -->
  <section class="relative min-h-screen flex items-center justify-center pt-16 overflow-hidden">
    <div class="absolute inset-0 bg-gradient-to-br from-teal-50/60 via-white to-sky-50/40"></div>
    <div class="absolute top-1/4 left-1/4 w-96 h-96 rounded-full blur-[80px] bg-teal-100/50 animate-pulse"></div>
    <div class="absolute bottom-1/4 right-1/4 w-64 h-64 rounded-full blur-[60px] bg-sky-100/40 animate-pulse" style="animation-delay: 1.5s;"></div>

    <div class="max-w-[1400px] mx-auto px-6 relative z-10">
      <div class="grid grid-cols-1 lg:grid-cols-2 gap-12 items-center">
        <div class="flex flex-col gap-8">
          <div class="inline-flex items-center gap-2 px-4 py-2 rounded-full w-fit text-sm bg-teal-50 border border-teal-200/60">
            <svg class="w-4 h-4 text-teal-500" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/></svg>
            <span class="text-teal-600 font-medium">Secure Webhook Delivery</span>
          </div>

          <h1 class="text-4xl md:text-5xl lg:text-6xl font-bold leading-tight font-fira">
            Modern <span class="bg-gradient-to-r from-teal-500 to-sky-500 bg-clip-text text-transparent">Event-Driven</span><br />
            Webhook System
          </h1>

          <p class="text-lg leading-relaxed max-w-xl text-gray-500">
            Production-ready webhook delivery with encryption at rest, HMAC signing, SSRF protection, guaranteed delivery, and complete observability. Fully open source -- every security layer is auditable.
          </p>

          <div class="flex flex-col sm:flex-row gap-4">
            <a href="/webhooks" class="inline-flex items-center justify-center gap-2 px-7 py-3 text-base font-medium rounded-xl bg-teal-500 text-white hover:bg-teal-600 transition-all shadow-lg shadow-teal-500/20">
              Open Dashboard
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2"><path d="M5 12h14M12 5l7 7-7 7"/></svg>
            </a>
            <a
              href="https://github.com/sarathsp06/sparrow"
              target="_blank"
              class="inline-flex items-center justify-center gap-2 px-7 py-3 text-base font-medium rounded-xl border border-gray-200 text-gray-700 hover:bg-gray-50 transition-all"
            >
              <svg class="w-5 h-5" viewBox="0 0 24 24" fill="currentColor"><path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0 0 24 12c0-6.63-5.37-12-12-12z"/></svg>
              View on GitHub
            </a>
          </div>

          <div class="pt-8 border-t border-gray-200">
            <p class="text-xs uppercase tracking-widest mb-3 text-gray-400">Built with</p>
            <div class="flex flex-wrap gap-3">
              {#each techStack as tech}
                <span class="px-3 py-1 text-xs font-fira rounded-md bg-gray-100 text-gray-500">{tech}</span>
              {/each}
            </div>
          </div>
        </div>

        <div class="flex justify-center lg:justify-end">
          <div class="relative">
            <div class="absolute inset-0 rounded-full blur-[60px] bg-gradient-to-br from-teal-200/40 via-teal-100/30 to-transparent" style="transform: scale(1.1);"></div>
            <div class="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-80 h-80 rounded-full blur-[40px] bg-teal-100/40"></div>
            <div class="relative p-8 rounded-3xl backdrop-blur-sm bg-gradient-to-br from-teal-50/60 via-white/40 to-transparent border border-teal-200/30">
              <img
                src={favicon}
                alt="Sparrow"
                class="relative w-72 h-72 md:w-80 md:h-80 drop-shadow-2xl"
                style="animation: float 6s ease-in-out infinite;"
              />
            </div>
          </div>
        </div>
      </div>
    </div>
  </section>

  <!-- ============================================================ -->
  <!-- FEATURES SECTION -->
  <!-- ============================================================ -->
  <section id="features" class="py-24 bg-white">
    <div class="max-w-[1400px] mx-auto px-6">
      <div class="text-center mb-16">
        <div class="inline-flex items-center gap-2 px-4 py-2 rounded-full text-sm mb-6 bg-teal-50 border border-teal-200/60">
          <span class="text-teal-600 font-medium">Core Features</span>
        </div>
        <h2 class="text-3xl md:text-4xl font-bold font-fira mb-4">
          Enterprise-Grade <span class="bg-gradient-to-r from-teal-500 to-sky-500 bg-clip-text text-transparent">Webhook Infrastructure</span>
        </h2>
        <p class="text-lg max-w-3xl mx-auto text-gray-500">Everything you need to build reliable event-driven systems at scale</p>
      </div>

      <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {#each features as feature}
          <div class="group p-8 rounded-xl border border-gray-200/80 bg-gray-50/50 transition-all duration-300 hover:border-teal-300/60 hover:shadow-lg hover:shadow-teal-500/5 hover:-translate-y-1">
            <div class="w-12 h-12 flex items-center justify-center rounded-xl mb-6 bg-teal-50">
              <svg class="w-5 h-5 text-teal-500" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d={feature.icon}/>
              </svg>
            </div>
            <h3 class="text-xl font-semibold mb-3 font-fira text-gray-900">{feature.title}</h3>
            <p class="leading-relaxed text-gray-500 text-[0.9375rem]">{feature.description}</p>
          </div>
        {/each}
      </div>
    </div>
  </section>

  <!-- ============================================================ -->
  <!-- SECURITY SECTION -->
  <!-- ============================================================ -->
  <section id="security" class="py-24 bg-gray-50/70">
    <div class="max-w-[1400px] mx-auto px-6">
      <div class="text-center mb-16">
        <div class="inline-flex items-center gap-2 px-4 py-2 rounded-full text-sm mb-6 bg-amber-50 border border-amber-200/60">
          <svg class="w-4 h-4 text-amber-500" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/></svg>
          <span class="text-amber-600 font-medium">Security</span>
        </div>
        <h2 class="text-3xl md:text-4xl font-bold font-fira mb-4">
          Security That's <span class="bg-gradient-to-r from-amber-500 to-orange-500 bg-clip-text text-transparent">Auditable, Not Proprietary</span>
        </h2>
        <p class="text-lg max-w-3xl mx-auto text-gray-500">
          Encryption at rest, SSRF protection, and HMAC signing are typically gated behind enterprise licenses.
          Sparrow ships them all as open-source Go code -- every line is reviewable.
          Organizations with HIPAA, SOC 2, or PCI DSS requirements can audit, verify, and extend the security controls themselves without licensing constraints.
        </p>
      </div>

      <!-- Encryption callout -->
      <div class="mb-8 p-8 rounded-xl border border-amber-200/60 bg-gradient-to-r from-amber-50/80 to-orange-50/40">
        <div class="flex flex-col md:flex-row gap-6 items-start">
          <div class="w-14 h-14 flex-shrink-0 flex items-center justify-center rounded-xl bg-amber-100">
            <svg class="w-7 h-7 text-amber-600" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <rect x="3" y="11" width="18" height="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/>
            </svg>
          </div>
          <div>
            <h3 class="text-xl font-semibold font-fira text-gray-900 mb-2">Encryption at Rest -- Open Source</h3>
            <p class="text-gray-600 text-[0.9375rem] leading-relaxed mb-3">
              Most webhook platforms store secrets in plaintext or gate encryption behind paid tiers.
              Sparrow encrypts webhook secrets and sensitive headers at rest using AES-256-GCM envelope encryption with per-record keys -- the same pattern used by AWS KMS and Google Cloud KMS.
              As an open-source project, Sparrow cannot grant compliance certifications, but it provides the technical controls that enable organizations to meet regulatory requirements through their own audit processes.
            </p>
            <div class="flex flex-wrap gap-2">
              <span class="px-2.5 py-1 text-xs font-fira rounded bg-amber-100 text-amber-700">AES-256-GCM</span>
              <span class="px-2.5 py-1 text-xs font-fira rounded bg-amber-100 text-amber-700">Envelope Encryption</span>
              <span class="px-2.5 py-1 text-xs font-fira rounded bg-amber-100 text-amber-700">Per-Record DEK</span>
              <span class="px-2.5 py-1 text-xs font-fira rounded bg-amber-100 text-amber-700">Key Rotation</span>
              <span class="px-2.5 py-1 text-xs font-fira rounded bg-amber-100 text-amber-700">DEK Memory Zeroing</span>
            </div>
          </div>
        </div>
      </div>

      <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {#each securityFeatures as feature}
          <div class="group p-8 rounded-xl border border-gray-200/80 bg-white transition-all duration-300 hover:border-amber-300/60 hover:shadow-lg hover:shadow-amber-500/5 hover:-translate-y-1">
            <div class="w-12 h-12 flex items-center justify-center rounded-xl mb-6 bg-amber-50">
              <svg class="w-5 h-5 text-amber-600" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d={feature.icon}/>
              </svg>
            </div>
            <h3 class="text-lg font-semibold mb-3 font-fira text-gray-900">{feature.title}</h3>
            <p class="leading-relaxed text-gray-500 text-[0.9375rem] mb-4">{feature.description}</p>
            <div class="flex flex-wrap gap-2">
              {#each feature.tags as tag}
                <span class="px-2 py-0.5 text-xs font-fira rounded bg-gray-100 text-gray-500">{tag}</span>
              {/each}
            </div>
          </div>
        {/each}
      </div>
    </div>
  </section>

  <!-- ============================================================ -->
  <!-- GETTING STARTED SECTION -->
  <!-- ============================================================ -->
  <section id="getting-started" class="py-24 bg-white">
    <div class="max-w-[1400px] mx-auto px-6">
      <div class="text-center mb-16">
        <div class="inline-flex items-center gap-2 px-4 py-2 rounded-full text-sm mb-6 bg-teal-50 border border-teal-200/60">
          <span class="text-teal-600 font-medium">Getting Started</span>
        </div>
        <h2 class="text-3xl md:text-4xl font-bold font-fira mb-4">
          Up and Running <span class="bg-gradient-to-r from-teal-500 to-sky-500 bg-clip-text text-transparent">in Minutes</span>
        </h2>
        <p class="text-lg max-w-3xl mx-auto text-gray-500">
          Start Sparrow with <code class="px-2 py-0.5 rounded bg-gray-200 text-gray-700 font-fira text-base">docker compose up -d</code> then run these three commands
        </p>
      </div>

      <div class="grid grid-cols-1 gap-6 max-w-4xl mx-auto">
        {#each steps as step}
          <div class="relative p-8 rounded-xl border border-gray-200/80 bg-gray-50/50">
            <span class="absolute top-6 right-6 text-5xl font-bold font-fira text-teal-100">{step.num}</span>
            <h3 class="text-xl font-semibold mb-2 text-teal-600 font-fira">{step.title}</h3>
            <p class="mb-5 text-gray-500 text-[0.9375rem]">{step.description}</p>
            <div class="rounded-lg p-4 overflow-x-auto bg-gray-900">
              <code class="text-sm text-teal-400 whitespace-pre font-fira">{step.code}</code>
            </div>
          </div>
        {/each}
      </div>

      <div class="mt-10 max-w-4xl mx-auto text-center">
        <p class="text-gray-500 text-[0.9375rem]">
          Need a test endpoint? Use
          <a href="https://testhooks.sarathsadasivan.com/" target="_blank" rel="noopener noreferrer" class="text-teal-600 hover:text-teal-700 font-medium">testhooks.sarathsadasivan.com</a>
          to inspect webhook payloads. For fine-grained routing, create subscriptions with
          <code class="px-1.5 py-0.5 rounded bg-gray-200 text-gray-700 font-fira text-sm">label_filters</code>
          via the SubscriptionService -- only events whose labels match will be delivered.
          Manage everything through the
          <a href="/webhooks" class="text-teal-600 hover:text-teal-700 font-medium">web dashboard</a>
          at <code class="px-1.5 py-0.5 rounded bg-gray-200 text-gray-700 font-fira text-sm">localhost:8080</code>.
        </p>
      </div>

      <div class="mt-8 max-w-4xl mx-auto">
        <div class="p-6 rounded-xl border border-teal-200/60 bg-teal-50/40">
          <div class="flex flex-col md:flex-row items-start md:items-center gap-4">
            <div class="w-10 h-10 flex-shrink-0 flex items-center justify-center rounded-lg bg-teal-100">
              <svg class="w-5 h-5 text-teal-600" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <polyline points="16 18 22 12 16 6"/><polyline points="8 6 2 12 8 18"/>
              </svg>
            </div>
            <div class="flex-1">
              <h4 class="font-semibold font-fira text-gray-900 mb-1">Prefer a client library?</h4>
              <p class="text-sm text-gray-500">Skip curl and use native SDKs for
                <a href="https://sarathsp06.github.io/sparrow/guides/go/" target="_blank" class="text-teal-600 hover:text-teal-700 font-medium">Go</a>,
                <a href="https://sarathsp06.github.io/sparrow/guides/python/" target="_blank" class="text-teal-600 hover:text-teal-700 font-medium">Python</a>, and
                <a href="https://sarathsp06.github.io/sparrow/guides/typescript/" target="_blank" class="text-teal-600 hover:text-teal-700 font-medium">TypeScript</a>.
                Connect-RPC (HTTP/JSON) and gRPC transports supported.
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>
  </section>

  <!-- ============================================================ -->
  <!-- COMPARISON SECTION -->
  <!-- ============================================================ -->
  <section id="comparison" class="py-24 bg-white">
    <div class="max-w-[1400px] mx-auto px-6">
      <div class="text-center mb-16">
        <div class="inline-flex items-center gap-2 px-4 py-2 rounded-full text-sm mb-6 bg-teal-50 border border-teal-200/60">
          <span class="text-teal-600 font-medium">Why Sparrow?</span>
        </div>
        <h2 class="text-3xl md:text-4xl font-bold font-fira mb-4">
          How Sparrow <span class="bg-gradient-to-r from-teal-500 to-sky-500 bg-clip-text text-transparent">Compares</span>
        </h2>
        <p class="text-lg max-w-3xl mx-auto text-gray-500">
          There are many ways to deliver webhooks. Here's why teams choose Sparrow.
        </p>
      </div>

      <div class="overflow-x-auto rounded-xl border border-gray-200/80 bg-white shadow-sm">
        <table class="w-full text-sm min-w-[900px]">
          <thead>
            <tr class="border-b border-gray-200">
              <th class="text-left py-4 px-5 font-medium text-gray-400 uppercase tracking-wider text-xs bg-gray-50/50 sticky left-0 z-10 min-w-[200px]">Feature</th>
              {#each comparisonServices as svc}
                <th class="py-4 px-4 text-center font-semibold font-fira text-xs uppercase tracking-wider min-w-[120px] {svc.highlight ? 'bg-teal-50/80 text-teal-700 border-x border-teal-200/50' : 'text-gray-500 bg-gray-50/50'}">
                  {#if svc.highlight}
                    <div class="flex flex-col items-center gap-1">
                      <span class="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full bg-teal-500 text-white text-[10px] font-bold uppercase tracking-widest">Recommended</span>
                      <span class="text-sm">{svc.name}</span>
                    </div>
                  {:else}
                    {svc.name}
                  {/if}
                </th>
              {/each}
            </tr>
          </thead>
          <tbody>
            {#each comparisonFeatures as row, i}
              <tr class="border-b border-gray-100 last:border-b-0 {i % 2 === 0 ? '' : 'bg-gray-50/30'}">
                <td class="py-3.5 px-5 font-medium text-gray-700 sticky left-0 z-10 {i % 2 === 0 ? 'bg-white' : 'bg-gray-50/30'}">{row.feature}</td>
                {#each comparisonServices as svc}
                  {@const val = row[svc.key]}
                  <td class="py-3.5 px-4 text-center {svc.highlight ? 'border-x border-teal-200/50' : ''} {svc.highlight && val === true ? 'bg-teal-50/60' : ''}">
                    {#if val === true}
                      <span class="inline-flex items-center justify-center w-6 h-6 rounded-full {svc.highlight ? 'bg-teal-500 text-white' : 'bg-green-100 text-green-600'}">
                        <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="3"><path d="M5 13l4 4L19 7"/></svg>
                      </span>
                    {:else if val === false}
                      <span class="inline-flex items-center justify-center w-6 h-6 rounded-full bg-gray-100 text-gray-300">
                        <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2.5"><path d="M6 18L18 6M6 6l12 12"/></svg>
                      </span>
                    {:else}
                      <span class="text-xs font-medium text-gray-400 px-2 py-0.5 rounded bg-gray-100/80">{val}</span>
                    {/if}
                  </td>
                {/each}
              </tr>
            {/each}
          </tbody>
        </table>
      </div>

      <div class="mt-12 p-8 rounded-xl border border-teal-200/60 bg-gradient-to-r from-teal-50/80 to-sky-50/60">
        <div class="grid grid-cols-2 md:grid-cols-4 gap-6 text-center">
          <div>
            <p class="text-2xl font-bold font-fira text-teal-600">$0</p>
            <p class="text-sm text-gray-500 mt-1">Per-message cost</p>
          </div>
          <div>
            <p class="text-2xl font-bold font-fira text-teal-600">6 Layers</p>
            <p class="text-sm text-gray-500 mt-1">Of security controls</p>
          </div>
          <div>
            <p class="text-2xl font-bold font-fira text-teal-600">100%</p>
            <p class="text-sm text-gray-500 mt-1">Open source, no gated features</p>
          </div>
          <div>
            <p class="text-2xl font-bold font-fira text-teal-600">Minutes</p>
            <p class="text-sm text-gray-500 mt-1">Docker Compose to production</p>
          </div>
        </div>
      </div>
    </div>
  </section>

  <!-- ============================================================ -->
  <!-- CTA SECTION -->
  <!-- ============================================================ -->
  <section class="py-24 bg-gradient-to-b from-white to-teal-50/30">
    <div class="max-w-3xl mx-auto px-6 text-center">
      <h2 class="text-3xl md:text-4xl font-bold font-fira mb-6">
        Ready to Build <span class="bg-gradient-to-r from-teal-500 to-sky-500 bg-clip-text text-transparent">Reliable Webhooks</span>?
      </h2>
      <p class="text-lg mb-8 text-gray-500">
        Join the developers using Sparrow to power their event-driven architectures. Open source, secure by default, and free to use.
      </p>
      <div class="flex flex-col sm:flex-row items-center justify-center gap-4">
        <a href="/webhooks" class="inline-flex items-center gap-2 px-7 py-3 text-base font-medium rounded-xl bg-teal-500 text-white hover:bg-teal-600 transition-all shadow-lg shadow-teal-500/20">
          Open Dashboard
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2"><path d="M5 12h14M12 5l7 7-7 7"/></svg>
        </a>
        <a
          href="https://github.com/sarathsp06/sparrow"
          target="_blank"
          class="inline-flex items-center gap-2 px-7 py-3 text-base font-medium rounded-xl border border-gray-200 text-gray-700 hover:bg-gray-50 transition-all"
        >
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2"><polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"/></svg>
          Star on GitHub
        </a>
      </div>
    </div>
  </section>

  <!-- ============================================================ -->
  <!-- FOOTER -->
  <!-- ============================================================ -->
  <footer class="py-12 border-t border-gray-200">
    <div class="max-w-[1400px] mx-auto px-6">
      <div class="flex flex-col md:flex-row items-center justify-between gap-6">
        <div class="flex items-center gap-2">
          <img src={favicon} alt="Sparrow" class="w-6 h-6" />
          <span class="font-fira font-semibold text-gray-900">Sparrow</span>
          {#if import.meta.env.VITE_APP_VERSION && import.meta.env.VITE_APP_VERSION !== 'dev'}
            <span class="text-xs text-gray-400 font-fira">v{import.meta.env.VITE_APP_VERSION}</span>
          {/if}
        </div>
        <p class="text-sm text-gray-400">Open source webhook delivery system</p>
        <div class="flex items-center gap-6">
          <a href="https://github.com/sarathsp06/sparrow" target="_blank" class="text-sm text-gray-500 hover:text-teal-600 transition-colors">GitHub</a>
          <a href="https://sarathsp06.github.io/sparrow/" target="_blank" class="text-sm text-gray-500 hover:text-teal-600 transition-colors">Docs</a>
        </div>
      </div>
    </div>
  </footer>
</div>

<style>
  @keyframes float {
    0%, 100% { transform: translateY(0); }
    50% { transform: translateY(-10px); }
  }
</style>

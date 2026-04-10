<script lang="ts">
  import favicon from "$lib/assets/favicon.svg";

  const features = [
    {
      icon: "M5 12H3l9-9 9 9h-2M5 12v7a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2v-7",
      title: "One Binary + PostgreSQL",
      description: "No Redis, no MongoDB, no message broker. Sparrow is a single Go binary backed by PostgreSQL. Deploy with docker compose or a Helm chart in minutes.",
    },
    {
      icon: "M21 2l-2 2m-7.61 7.61a5.5 5.5 0 1 1-7.778 7.778 5.5 5.5 0 0 1 7.777-7.777zm0 0L15.5 7.5m0 0l3 3L22 7l-3-3m-3.5 3.5L19 4",
      title: "Encryption at Rest",
      description: "Webhook secrets encrypted with AES-256-GCM envelope encryption and per-record data keys. Most competitors store secrets in plaintext or gate encryption behind paid tiers.",
    },
    {
      icon: "M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z",
      title: "gRPC + HTTP/JSON",
      description: "Native gRPC on :50051 and Connect-RPC (HTTP/JSON) on :8080. Use curl, generated SDKs for Go/Python/TypeScript, or the built-in web dashboard.",
    },
    {
      icon: "M1 4v6h6M23 20v-6h-6M20.49 9A9 9 0 0 0 5.64 5.64L1 10m22 4l-4.64 4.36A9 9 0 0 1 3.51 15",
      title: "Retries + Health Tracking",
      description: "Exponential backoff with 9-category error classification. Per-webhook health metrics track success rate, P95 response time, and consecutive failures.",
    },
  ];

  const steps = [
    {
      num: "01",
      title: "Start Sparrow",
      description: "One command. PostgreSQL is included.",
      code: `docker compose up -d`,
    },
    {
      num: "02",
      title: "Register a Webhook",
      description: "Point an endpoint at the events it should receive.",
      code: `curl -X POST http://localhost:8080/webhook.WebhookService/RegisterWebhook \\
  -H "Content-Type: application/json" \\
  -d '{
    "namespace": "default",
    "url": "https://example.com/webhook",
    "events": ["order.created"],
    "active": true
  }'`,
    },
    {
      num: "03",
      title: "Push an Event",
      description: "Sparrow matches subscriptions, fans out deliveries, and retries failures.",
      code: `curl -X POST http://localhost:8080/webhook.EventService/PushEvent \\
  -H "Content-Type: application/json" \\
  -d '{
    "namespace": "default",
    "event": "order.created",
    "payload": {"order_id": "ord_123", "amount": 99.99},
    "labels": {"region": "us-east"}
  }'`,
    },
  ];

  const comparisonServices = [
    { key: "sparrow", name: "Sparrow", highlight: true },
    { key: "svix", name: "Svix" },
    { key: "convoy", name: "Convoy" },
    { key: "hookdeck", name: "Hookdeck" },
    { key: "aws", name: "AWS SNS" },
    { key: "diy", name: "DIY" },
  ];

  type ComparisonRow = {
    feature: string;
    sparrow: boolean | string;
    svix: boolean | string;
    convoy: boolean | string;
    hookdeck: boolean | string;
    aws: boolean | string;
    diy: boolean | string;
  };

  const comparisonFeatures: ComparisonRow[] = [
    {
      feature: "Fully Open Source (MIT)",
      sparrow: true, svix: "Partial", convoy: true, hookdeck: false, aws: false, diy: true,
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
      feature: "Minimal Infra (Binary + Postgres)",
      sparrow: true, svix: false, convoy: false, hookdeck: "SaaS", aws: "Managed", diy: "Varies",
    },
    {
      feature: "Encryption at Rest",
      sparrow: true, svix: "Enterprise", convoy: false, hookdeck: "Managed", aws: "Managed", diy: false,
    },
    {
      feature: "gRPC / Connect-RPC Native",
      sparrow: true, svix: false, convoy: false, hookdeck: false, aws: false, diy: false,
    },
    {
      feature: "Payload Transformation",
      sparrow: true, svix: "Paid", convoy: true, hookdeck: true, aws: false, diy: false,
    },
    {
      feature: "Webhook Health Tracking",
      sparrow: true, svix: true, convoy: true, hookdeck: true, aws: false, diy: false,
    },
    {
      feature: "Client SDKs",
      sparrow: "3", svix: "10+", convoy: true, hookdeck: true, aws: true, diy: false,
    },
    {
      feature: "Consumer App Portal",
      sparrow: false, svix: true, convoy: false, hookdeck: false, aws: false, diy: false,
    },
    {
      feature: "Rate Limiting",
      sparrow: false, svix: true, convoy: true, hookdeck: true, aws: true, diy: false,
    },
  ];
</script>

<svelte:head>
  <title>Sparrow - Self-Hosted Webhook Delivery</title>
  <meta name="description" content="Self-hosted webhook delivery with encryption at rest, HMAC signing, guaranteed delivery, and health tracking. One binary + PostgreSQL. MIT licensed." />
</svelte:head>

<div class="font-inter min-h-screen bg-white text-gray-900">
  <!-- HEADER -->
  <header class="fixed top-0 left-0 right-0 z-50 border-b border-gray-200/60 bg-white/80 backdrop-blur-xl">
    <div class="max-w-[1400px] mx-auto px-6 flex items-center justify-between h-16">
      <a href="/" class="flex items-center gap-3">
        <img src={favicon} alt="Sparrow" class="w-8 h-8" />
        <span class="text-xl font-bold font-fira text-gray-900">Sparrow</span>
      </a>
      <nav class="hidden md:flex items-center gap-8">
        <a href="#features" class="text-sm text-gray-500 hover:text-teal-600 transition-colors">Features</a>
        <a href="#getting-started" class="text-sm text-gray-500 hover:text-teal-600 transition-colors">Getting Started</a>
        <a href="#comparison" class="text-sm text-gray-500 hover:text-teal-600 transition-colors">Comparison</a>
        <a href="https://sarathsp06.github.io/sparrow/" target="_blank" class="text-sm text-gray-500 hover:text-teal-600 transition-colors">Docs</a>
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
          Dashboard
        </a>
      </div>
    </div>
  </header>

  <!-- HERO -->
  <section class="relative min-h-[85vh] flex items-center pt-16 overflow-hidden">
    <div class="absolute inset-0 bg-gradient-to-br from-teal-50/60 via-white to-sky-50/40"></div>
    <div class="absolute top-1/4 left-1/4 w-96 h-96 rounded-full blur-[80px] bg-teal-100/50 animate-pulse"></div>
    <div class="absolute bottom-1/4 right-1/4 w-64 h-64 rounded-full blur-[60px] bg-sky-100/40 animate-pulse" style="animation-delay: 1.5s;"></div>

    <div class="max-w-[1400px] mx-auto px-6 relative z-10">
      <div class="grid grid-cols-1 lg:grid-cols-2 gap-12 items-center">
        <div class="flex flex-col gap-8">
          <div class="inline-flex items-center gap-2 px-4 py-2 rounded-full w-fit text-sm bg-teal-50 border border-teal-200/60">
            <span class="text-teal-600 font-medium">MIT Licensed &middot; Self-Hosted</span>
          </div>

          <h1 class="text-4xl md:text-5xl lg:text-6xl font-bold leading-tight font-fira">
            Webhook delivery <span class="bg-gradient-to-r from-teal-500 to-sky-500 bg-clip-text text-transparent">you own.</span>
          </h1>

          <p class="text-lg leading-relaxed max-w-xl text-gray-500">
            One binary + PostgreSQL. Async fan-out with retries, HMAC signing, encryption at rest, per-webhook health metrics, and a built-in dashboard. No SaaS, no per-message fees, no vendor lock-in.
          </p>

          <div class="flex flex-col sm:flex-row gap-4">
            <a href="https://sarathsp06.github.io/sparrow/getting-started/quickstart/" target="_blank" class="inline-flex items-center justify-center gap-2 px-7 py-3 text-base font-medium rounded-xl bg-teal-500 text-white hover:bg-teal-600 transition-all shadow-lg shadow-teal-500/20">
              Get Started
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
        </div>

        <div class="flex justify-center lg:justify-end">
          <div class="relative">
            <div class="absolute inset-0 rounded-full blur-[60px] bg-gradient-to-br from-teal-200/40 via-teal-100/30 to-transparent" style="transform: scale(1.1);"></div>
            <div class="relative p-8 rounded-3xl backdrop-blur-sm bg-gradient-to-br from-teal-50/60 via-white/40 to-transparent border border-teal-200/30">
              <img
                src={favicon}
                alt="Sparrow"
                class="relative w-64 h-64 md:w-72 md:h-72 drop-shadow-2xl"
                style="animation: float 6s ease-in-out infinite;"
              />
            </div>
          </div>
        </div>
      </div>
    </div>
  </section>

  <!-- FEATURES -->
  <section id="features" class="py-24 bg-white">
    <div class="max-w-[1400px] mx-auto px-6">
      <div class="text-center mb-16">
        <h2 class="text-3xl md:text-4xl font-bold font-fira mb-4">
          What <span class="bg-gradient-to-r from-teal-500 to-sky-500 bg-clip-text text-transparent">You Get</span>
        </h2>
      </div>

      <div class="grid grid-cols-1 md:grid-cols-2 gap-6 max-w-5xl mx-auto">
        {#each features as feature}
          <div class="group p-8 rounded-xl border border-gray-200/80 bg-gray-50/50 transition-all duration-300 hover:border-teal-300/60 hover:shadow-lg hover:shadow-teal-500/5">
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

      <!-- Also included -->
      <div class="mt-12 max-w-5xl mx-auto">
        <div class="p-6 rounded-xl border border-gray-200/60 bg-gray-50/30">
          <p class="text-sm font-medium text-gray-400 uppercase tracking-wider mb-4">Also included</p>
          <div class="grid grid-cols-2 md:grid-cols-4 gap-4">
            <div class="text-sm text-gray-600"><span class="text-teal-500 mr-2">&#10003;</span>HMAC-SHA256 signing</div>
            <div class="text-sm text-gray-600"><span class="text-teal-500 mr-2">&#10003;</span>SSRF protection</div>
            <div class="text-sm text-gray-600"><span class="text-teal-500 mr-2">&#10003;</span>Idempotent event delivery</div>
            <div class="text-sm text-gray-600"><span class="text-teal-500 mr-2">&#10003;</span>Label-based filtering</div>
            <div class="text-sm text-gray-600"><span class="text-teal-500 mr-2">&#10003;</span>OpenTelemetry native</div>
            <div class="text-sm text-gray-600"><span class="text-teal-500 mr-2">&#10003;</span>Go template transforms</div>
            <div class="text-sm text-gray-600"><span class="text-teal-500 mr-2">&#10003;</span>Batch re-push &amp; retry</div>
            <div class="text-sm text-gray-600"><span class="text-teal-500 mr-2">&#10003;</span>Namespace isolation</div>
          </div>
        </div>
      </div>
    </div>
  </section>

  <!-- GETTING STARTED -->
  <section id="getting-started" class="py-24 bg-gray-50/70">
    <div class="max-w-[1400px] mx-auto px-6">
      <div class="text-center mb-16">
        <h2 class="text-3xl md:text-4xl font-bold font-fira mb-4">
          Three Steps <span class="bg-gradient-to-r from-teal-500 to-sky-500 bg-clip-text text-transparent">to Delivery</span>
        </h2>
      </div>

      <div class="grid grid-cols-1 gap-6 max-w-4xl mx-auto">
        {#each steps as step}
          <div class="relative p-8 rounded-xl border border-gray-200/80 bg-white">
            <span class="absolute top-6 right-6 text-5xl font-bold font-fira text-teal-100">{step.num}</span>
            <h3 class="text-xl font-semibold mb-2 text-teal-600 font-fira">{step.title}</h3>
            <p class="mb-5 text-gray-500 text-[0.9375rem]">{step.description}</p>
            <div class="rounded-lg p-4 overflow-x-auto bg-gray-900">
              <code class="text-sm text-teal-400 whitespace-pre font-fira">{step.code}</code>
            </div>
          </div>
        {/each}
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
              <p class="text-sm text-gray-500">Use native SDKs for
                <a href="https://sarathsp06.github.io/sparrow/getting-started/go/" target="_blank" class="text-teal-600 hover:text-teal-700 font-medium">Go</a>,
                <a href="https://sarathsp06.github.io/sparrow/getting-started/python/" target="_blank" class="text-teal-600 hover:text-teal-700 font-medium">Python</a>, or
                <a href="https://sarathsp06.github.io/sparrow/getting-started/typescript/" target="_blank" class="text-teal-600 hover:text-teal-700 font-medium">TypeScript</a>.
                Connect-RPC and gRPC transports supported.
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>
  </section>

  <!-- COMPARISON -->
  <section id="comparison" class="py-24 bg-white">
    <div class="max-w-[1400px] mx-auto px-6">
      <div class="text-center mb-16">
        <h2 class="text-3xl md:text-4xl font-bold font-fira mb-4">
          Honest <span class="bg-gradient-to-r from-teal-500 to-sky-500 bg-clip-text text-transparent">Comparison</span>
        </h2>
        <p class="text-lg max-w-3xl mx-auto text-gray-500">
          Sparrow is not the right choice for everyone. Here's where it fits.
        </p>
      </div>

      <div class="overflow-x-auto rounded-xl border border-gray-200/80 bg-white shadow-sm">
        <table class="w-full text-sm min-w-[900px]">
          <thead>
            <tr class="border-b border-gray-200">
              <th class="text-left py-4 px-5 font-medium text-gray-400 uppercase tracking-wider text-xs bg-gray-50/50 sticky left-0 z-10 min-w-[220px]">Feature</th>
              {#each comparisonServices as svc}
                <th class="py-4 px-4 text-center font-semibold font-fira text-xs uppercase tracking-wider min-w-[110px] {svc.highlight ? 'bg-teal-50/80 text-teal-700 border-x border-teal-200/50' : 'text-gray-500 bg-gray-50/50'}">
                  {svc.name}
                </th>
              {/each}
            </tr>
          </thead>
          <tbody>
            {#each comparisonFeatures as row, i}
              <tr class="border-b border-gray-100 last:border-b-0 {i % 2 === 0 ? '' : 'bg-gray-50/30'}">
                <td class="py-3.5 px-5 font-medium text-gray-700 sticky left-0 z-10 {i % 2 === 0 ? 'bg-white' : 'bg-gray-50/30'}">{row.feature}</td>
                {#each comparisonServices as svc}
                  {@const val = row[svc.key as keyof ComparisonRow]}
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

      <div class="mt-8 max-w-4xl mx-auto">
        <div class="p-6 rounded-xl border border-gray-200/60 bg-gray-50/30">
          <p class="text-sm text-gray-500 leading-relaxed">
            <strong class="text-gray-700">When to use Sparrow:</strong> You need outbound webhook delivery, want to self-host, and don't want to pay per message.
            <strong class="text-gray-700">When not to:</strong> You need a consumer app portal, advanced rate limiting, or 10+ language SDKs today -- Svix is a better fit.
          </p>
        </div>
      </div>
    </div>
  </section>

  <!-- CTA -->
  <section class="py-24 bg-gradient-to-b from-white to-teal-50/30">
    <div class="max-w-3xl mx-auto px-6 text-center">
      <h2 class="text-3xl md:text-4xl font-bold font-fira mb-6">
        Ready to <span class="bg-gradient-to-r from-teal-500 to-sky-500 bg-clip-text text-transparent">get started</span>?
      </h2>
      <p class="text-lg mb-8 text-gray-500">
        MIT licensed. Deploy in minutes. Zero per-message fees.
      </p>
      <div class="flex flex-col sm:flex-row items-center justify-center gap-4">
        <a href="https://sarathsp06.github.io/sparrow/getting-started/quickstart/" target="_blank" class="inline-flex items-center gap-2 px-7 py-3 text-base font-medium rounded-xl bg-teal-500 text-white hover:bg-teal-600 transition-all shadow-lg shadow-teal-500/20">
          Read the Docs
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

  <!-- FOOTER -->
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
        <p class="text-sm text-gray-400">MIT licensed &middot; Go + PostgreSQL &middot; Open source webhook delivery</p>
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

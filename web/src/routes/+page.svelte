<script>
  import favicon from "$lib/assets/favicon.svg";
</script>

<svelte:head>
  <title>Sparrow - Webhook Queue</title>
</svelte:head>

<div class="min-h-screen bg-gray-50 flex flex-col">
  <main class="flex-1 flex flex-col items-center justify-center text-center px-4">
    <div class="max-w-2xl py-8">
      <div class="mb-6">
        <img
          src={favicon}
          alt="Sparrow logo"
          class="w-24 h-24 mx-auto mb-4"
        />
        <h1 class="text-4xl md:text-5xl font-extrabold text-gray-900 leading-tight">
          Sparrow Webhook Queue
        </h1>
        <p class="text-lg text-gray-500 mt-2 font-mono">
          Developer-Focused. Minimal. Reliable.
        </p>
      </div>

      <p class="text-gray-600 mb-8 max-w-lg mx-auto">
        Register, monitor, and deliver webhooks with speed and clarity.
        Open source, built for developers.
      </p>

      <div class="mb-8">
        <div class="flex flex-col sm:flex-row gap-3 justify-center items-center max-w-md mx-auto">
          <a
            href="/webhooks"
            class="w-full sm:w-auto bg-gray-900 text-white font-medium py-3 px-8 rounded-lg shadow hover:bg-gray-800 transition text-sm text-center"
          >
            View Webhooks
          </a>
          <a
            href="/webhooks/register"
            class="w-full sm:w-auto bg-white text-gray-900 font-medium py-3 px-8 rounded-lg shadow border border-gray-300 hover:bg-gray-50 transition text-sm text-center"
          >
            Register Webhook
          </a>
        </div>
        <div class="mt-4 flex items-center justify-center gap-4 text-sm">
          <a href="/events" class="text-gray-500 hover:text-gray-900 transition font-medium">
            View Events
          </a>
          <span class="text-gray-300">|</span>
          <a href="/health" class="text-gray-500 hover:text-gray-900 transition font-medium">
            Health Dashboard
          </a>
        </div>
      </div>
    </div>
  </main>

  <section class="py-16 px-4 border-t border-gray-200 bg-white">
    <div class="max-w-5xl mx-auto">
      <h2 class="text-2xl font-bold text-gray-900 text-center mb-10">Features</h2>
      <div class="grid grid-cols-1 sm:grid-cols-2 gap-6">
        <div class="bg-gray-50 rounded-lg p-6 border border-gray-100">
          <h3 class="text-base font-semibold text-gray-800 mb-2">Event-Driven Delivery</h3>
          <p class="text-sm text-gray-600">
            Push events and let Sparrow handle delivery, retries, and status tracking automatically.
          </p>
        </div>
        <div class="bg-gray-50 rounded-lg p-6 border border-gray-100">
          <h3 class="text-base font-semibold text-gray-800 mb-2">Observability Dashboard</h3>
          <p class="text-sm text-gray-600">
            Monitor webhook health, delivery attempts, and event flows in real time.
          </p>
        </div>
        <div class="bg-gray-50 rounded-lg p-6 border border-gray-100">
          <h3 class="text-base font-semibold text-gray-800 mb-2">Flexible API</h3>
          <p class="text-sm text-gray-600">
            gRPC, REST, and Connect support for easy integration with any stack.
          </p>
        </div>
        <div class="bg-gray-50 rounded-lg p-6 border border-gray-100">
          <h3 class="text-base font-semibold text-gray-800 mb-2">Security First</h3>
          <p class="text-sm text-gray-600">
            Built-in authentication, header management, and audit logging for peace of mind.
          </p>
        </div>
      </div>
    </div>
  </section>

  <footer class="py-6 text-center text-gray-400 text-xs font-mono border-t border-gray-100">
    Sparrow &copy; {new Date().getFullYear()} &mdash; Built for developers
  </footer>
</div>

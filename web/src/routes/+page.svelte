<script>
  // No logic needed for static landing page
</script>

<!-- Paper-inspired Light Theme Landing Page -->
<div class="min-h-screen bg-gray-50 font-mono flex flex-col relative overflow-hidden">


  <main class="flex-1 flex flex-col items-center justify-center text-center px-4 z-10">
    <div class="max-w-2xl py-4">
      <h1 class="text-5xl md:text-6xl font-extrabold text-gray-900 mb-6 leading-tight drop-shadow animate-fade-in">
        <span class="text-gray-800">Sparrow Webhook Queue</span><br>
        <span class="text-gray-500 font-mono text-2xl">Developer-Focused. Minimal. Reliable.</span>
      </h1>
      <p class="text-lg md:text-xl text-gray-600 mb-8 animate-fade-in delay-100">
        Register, monitor, and deliver webhooks with speed and clarity.<br>
        <span class="text-gray-700 font-semibold">Open source. Built for developers.</span>
      </p>
      <div class="flex flex-col md:flex-row gap-4 justify-center mb-10 animate-fade-in delay-200">
        <a href="/webhooks" class="bg-gray-900 text-white font-bold py-3 px-8 rounded-lg shadow hover:scale-105 hover:bg-gray-800 transition text-lg">Get Started</a>
        <a href="/events" class="bg-white border border-gray-900 text-gray-900 font-bold py-3 px-8 rounded-lg shadow hover:bg-gray-900 hover:text-white transition text-lg">View Events</a>
      </div>
    </div>
  </main>

  <section id="features" class="py-16 px-4 bg-white z-10">
    <div class="max-w-5xl mx-auto text-center">
      <h2 class="text-3xl font-bold mb-10 text-gray-900 drop-shadow">Features</h2>
      <div class="grid grid-cols-1 md:grid-cols-2 gap-8">
        <div class="paper-card bg-gray-100 rounded-xl p-6 shadow">
          <h3 class="text-xl font-semibold mb-2 text-gray-700">Event-Driven Delivery</h3>
          <p class="text-gray-600">Push events and let Sparrow handle delivery, retries, and status tracking automatically.</p>
        </div>
        <div class="paper-card bg-gray-100 rounded-xl p-6 shadow">
          <h3 class="text-xl font-semibold mb-2 text-gray-700">Observability Dashboard</h3>
          <p class="text-gray-600">Monitor webhook health, delivery attempts, and event flows in real time.</p>
        </div>
        <div class="paper-card bg-gray-100 rounded-xl p-6 shadow">
          <h3 class="text-xl font-semibold mb-2 text-gray-700">Flexible API</h3>
          <p class="text-gray-600">gRPC, REST, and Connect support for easy integration with any stack.</p>
        </div>
        <div class="paper-card bg-gray-100 rounded-xl p-6 shadow">
          <h3 class="text-xl font-semibold mb-2 text-gray-700">Security First</h3>
          <p class="text-gray-600">Built-in authentication, header management, and audit logging for peace of mind.</p>
        </div>
      </div>
    </div>
  </section>

  <footer class="py-8 text-center text-gray-400 z-10 font-mono text-sm">
    Sparrow &copy; {new Date().getFullYear()} &mdash; Built for developers
  </footer>

  <style>
    .paper-card {
      background: #f7f7f7;
      border: 1px solid #e5e7eb;
      box-shadow: 0 2px 8px 0 rgba(60, 60, 100, 0.06);
      transition: box-shadow 0.2s, transform 0.2s;
    }
    .paper-card:hover {
      box-shadow: 0 8px 24px 0 rgba(60, 60, 100, 0.10);
      transform: translateY(-2px) scale(1.02);
    }
    @keyframes fade-in {
      from { opacity: 0; transform: translateY(20px); }
      to { opacity: 1; transform: none; }
    }
    .animate-fade-in {
      animation: fade-in 0.8s cubic-bezier(.4,0,.2,1) both;
    }
    .delay-100 { animation-delay: 0.1s; }
    .delay-200 { animation-delay: 0.2s; }
    .delay-300 { animation-delay: 0.3s; }
  </style>
</div>

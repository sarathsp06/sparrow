<script>
  // No logic needed for static landing page
</script>

<!-- Paper-inspired Light Theme Landing Page -->
<div class="min-h-screen bg-gradient-to-br from-white via-gray-50 to-gray-100 font-display flex flex-col relative overflow-hidden">
  <header class="flex items-center justify-between px-8 py-6 z-10 bg-white/90 backdrop-blur-md border-b border-gray-200 shadow-sm">
    <div class="flex items-center gap-2">
      <span class="inline-block bg-primary rounded-full p-2 shadow-lg">
        <svg class="w-8 h-8 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" /></svg>
      </span>
      <span class="text-2xl font-bold text-primary drop-shadow">Sparrow</span>
    </div>
    <nav class="flex gap-8 text-lg font-medium">
      <a href="/webhooks" class="hover:text-primary transition">Webhooks</a>
      <a href="/events" class="hover:text-primary transition">Events</a>
      <a href="#features" class="hover:text-primary transition">Features</a>
    </nav>
  </header>

  <main class="flex-1 flex flex-col items-center justify-center text-center px-4 z-10">
    <div class="max-w-2xl">
      <h1 class="text-6xl md:text-7xl font-extrabold text-gray-900 mb-6 leading-tight drop-shadow-lg animate-fade-in">
        <span class="bg-gradient-to-r from-blue-400 via-pink-400 to-purple-400 bg-clip-text text-transparent">Effortless Webhook Management</span><br>
        <span class="text-primary">for Modern Apps</span>
      </h1>
      <p class="text-xl md:text-2xl text-gray-700 mb-8 animate-fade-in delay-100">
        Sparrow lets you register, monitor, and deliver webhooks with reliability, observability, and blazing speed.<br>
        <span class="text-blue-500 font-semibold">Built for developers, trusted by teams.</span>
      </p>
      <div class="flex flex-col md:flex-row gap-4 justify-center mb-10 animate-fade-in delay-200">
        <a href="/webhooks" class="bg-primary text-white font-bold py-3 px-8 rounded-xl shadow-xl hover:scale-105 hover:bg-primary/90 transition text-lg">Get Started</a>
        <a href="/events" class="bg-white border border-primary text-primary font-bold py-3 px-8 rounded-xl shadow-xl hover:bg-primary hover:text-white transition text-lg">View Events</a>
      </div>
      <div class="flex flex-wrap gap-8 justify-center items-center mt-8 animate-fade-in delay-300">
        <div class="paper-card flex flex-col items-center p-6 rounded-xl shadow-xl">
          <span class="material-symbols-outlined text-5xl text-blue-400 mb-2">bolt</span>
          <span class="font-semibold text-gray-700">Fast Delivery</span>
        </div>
        <div class="paper-card flex flex-col items-center p-6 rounded-xl shadow-xl">
          <span class="material-symbols-outlined text-5xl text-pink-400 mb-2">visibility</span>
          <span class="font-semibold text-gray-700">Full Observability</span>
        </div>
        <div class="paper-card flex flex-col items-center p-6 rounded-xl shadow-xl">
          <span class="material-symbols-outlined text-5xl text-purple-400 mb-2">security</span>
          <span class="font-semibold text-gray-700">Secure & Reliable</span>
        </div>
        <div class="paper-card flex flex-col items-center p-6 rounded-xl shadow-xl">
          <span class="material-symbols-outlined text-5xl text-blue-400 mb-2">settings</span>
          <span class="font-semibold text-gray-700">Easy Integration</span>
        </div>
      </div>
    </div>
  </main>

  <section id="features" class="py-20 px-4 bg-white z-10">
    <div class="max-w-5xl mx-auto text-center">
      <h2 class="text-4xl font-bold mb-10 text-gray-900 drop-shadow">Features</h2>
      <div class="grid grid-cols-1 md:grid-cols-2 gap-10">
        <div class="paper-card bg-blue-50 rounded-2xl p-8 shadow-xl">
          <h3 class="text-2xl font-semibold mb-3 text-blue-500">Event-Driven Delivery</h3>
          <p class="text-gray-700">Push events and let Sparrow handle delivery, retries, and status tracking automatically.</p>
        </div>
        <div class="paper-card bg-pink-50 rounded-2xl p-8 shadow-xl">
          <h3 class="text-2xl font-semibold mb-3 text-pink-500">Observability Dashboard</h3>
          <p class="text-gray-700">Monitor webhook health, delivery attempts, and event flows in real time.</p>
        </div>
        <div class="paper-card bg-purple-50 rounded-2xl p-8 shadow-xl">
          <h3 class="text-2xl font-semibold mb-3 text-purple-500">Flexible API</h3>
          <p class="text-gray-700">gRPC, REST, and Connect support for easy integration with any stack.</p>
        </div>
        <div class="paper-card bg-blue-50 rounded-2xl p-8 shadow-xl">
          <h3 class="text-2xl font-semibold mb-3 text-blue-500">Security First</h3>
          <p class="text-gray-700">Built-in authentication, header management, and audit logging for peace of mind.</p>
        </div>
      </div>
    </div>
  </section>

  <footer class="py-8 text-center text-gray-500 z-10">
    Sparrow &copy; {new Date().getFullYear()} &mdash; Built for developers
  </footer>

  <style>
    .paper-card {
      background: #fff;
      border: 1px solid #f3f3f3;
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
    .bg-primary { background-color: #2563eb; }
    .text-primary { color: #2563eb; }
  </style>
</div>

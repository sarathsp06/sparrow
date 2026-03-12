<!--
  ClerkAuthBridge — invisible component that bridges the Clerk context
  (available inside ClerkProvider's component tree) with the auth module
  (used by services.ts outside the component tree).
  
  It registers a token getter function so that Connect-RPC requests can
  automatically include the Clerk session JWT.
-->
<script lang="ts">
  import { useClerkContext } from "svelte-clerk/client";
  import { registerTokenProvider } from "$lib/auth";

  const ctx = useClerkContext();

  // Register the token getter so services.ts can access it
  registerTokenProvider(async () => {
    if (!ctx.session) return null;
    return await ctx.session.getToken();
  });
</script>

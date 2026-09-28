<script lang="ts">
  import { onMount } from "svelte";
  import { api, unwrap } from "$lib/services";
  import { formatAPIError, timeAgo } from "$lib/utils";
  import ConfirmDialog from "$lib/components/ConfirmDialog.svelte";
  import EmptyState from "$lib/components/EmptyState.svelte";
  import type { components } from "$lib/api-types";

  type Token = components["schemas"]["TokenOut"];
  type Invite = components["schemas"]["InviteOut"];
  type Who = components["schemas"]["WhoAmIOutputBody"];

  let who = $state<Who | null>(null);
  let tokens = $state<Token[]>([]);
  let invites = $state<Invite[]>([]);
  let showInactive = $state(false);
  let loading = $state(true);
  let error = $state("");

  async function load() {
    error = "";
    try {
      const [w, t, i] = await Promise.all([
        api.GET("/v1/whoami"),
        api.GET("/v1/tokens", { params: { query: { include_inactive: showInactive } } }),
        api.GET("/v1/invites"),
      ]);
      who = unwrap(w);
      tokens = unwrap(t).items ?? [];
      invites = unwrap(i).items ?? [];
    } catch (e) {
      error = formatAPIError(e, "Failed to load access");
    } finally {
      loading = false;
    }
  }
  onMount(load);

  const access = (consumer: string | null | undefined) => (consumer ? `Portal: ${consumer}` : "Full access");
  const when = (t: string | null | undefined) => (t ? new Date(t).toLocaleString() : "Never");

  // --- Create dialog (invite or token) ---
  type Kind = "invite" | "token";
  let dialog = $state<Kind | null>(null);
  let name = $state("");
  let scope = $state<"full" | "consumer">("full");
  let consumer = $state("");
  let inviteTTL = $state(24 * 3600);
  /** Select value for a token that never expires (sent as never_expires). */
  const NEVER = -1;
  let tokenTTL = $state(0);
  let creating = $state(false);
  let createError = $state("");
  let result = $state<{ label: string; value: string; note: string } | null>(null);
  let copied = $state(false);

  function openDialog(kind: Kind) {
    dialog = kind;
    name = "";
    scope = "full";
    consumer = "";
    inviteTTL = 24 * 3600;
    tokenTTL = 0;
    createError = "";
    result = null;
    copied = false;
  }

  // Consumer tokens always expire (7 days by default, 30 at most); tenant-wide
  // tokens use the server default (SPARROW_TOKEN_DEFAULT_TTL, 90 days unless
  // changed) unless asked. NEVER asks for a token that does not expire.
  const tokenTTLOptions = $derived(
    scope === "consumer"
      ? [
          [7 * 86400, "7 days"],
          [30 * 86400, "30 days"],
        ]
      : [
          [0, "Server default (90 days unless changed)"],
          [7 * 86400, "7 days"],
          [30 * 86400, "30 days"],
          [90 * 86400, "90 days"],
          [NEVER, "Never (revoke when no longer needed)"],
        ],
  );
  $effect(() => {
    if (scope === "consumer" && tokenTTL <= 0) tokenTTL = 7 * 86400;
  });

  async function create(e: SubmitEvent) {
    e.preventDefault();
    if (!name.trim() || (scope === "consumer" && !consumer.trim())) return;
    creating = true;
    createError = "";
    const c = scope === "consumer" ? consumer.trim() : undefined;
    try {
      if (dialog === "invite") {
        const res = unwrap(
          await api.POST("/v1/invites", {
            body: {
              name: name.trim(),
              consumer: c,
              ttl_seconds: inviteTTL,
              token_ttl_seconds: tokenTTL > 0 ? tokenTTL : undefined,
              token_never_expires: tokenTTL === NEVER || undefined,
            },
          }),
        );
        result = {
          label: "Invite link",
          value: location.origin + res.path,
          note: `Works once, until ${new Date(res.invite.expires_at).toLocaleString()}. ${c ? `It opens ${c}'s portal.` : "It signs the browser into this console."}`,
        };
      } else {
        const res = unwrap(
          await api.POST("/v1/tokens", {
            body: {
              name: name.trim(),
              consumer: c,
              ttl_seconds: tokenTTL > 0 ? tokenTTL : undefined,
              never_expires: tokenTTL === NEVER || undefined,
            },
          }),
        );
        result = res.portal_path
          ? {
              label: "Portal link",
              value: location.origin + res.portal_path,
              note: `Copy it now: it won't be shown again. It opens ${c}'s portal (and the token in it only works through the portal API, /portal/api/).`,
            }
          : {
              label: "Token",
              value: res.secret,
              note: `Copy it now: it won't be shown again. Send it as the X-API-Key header or "Authorization: Bearer".`,
            };
      }
      await load();
    } catch (err) {
      createError = formatAPIError(err, dialog === "invite" ? "Failed to create invite" : "Failed to create token");
    } finally {
      creating = false;
    }
  }

  async function copy() {
    if (!result) return;
    try {
      await navigator.clipboard.writeText(result.value);
      copied = true;
    } catch {
      copied = false;
    }
  }

  // --- Revoke / cancel ---
  let revoking = $state<Token | null>(null);
  let cancelling = $state<Invite | null>(null);

  async function revoke() {
    const t = revoking;
    revoking = null;
    if (!t) return;
    try {
      unwrap(await api.DELETE("/v1/tokens/{token_id}", { params: { path: { token_id: t.id } } }));
      await load();
    } catch (e) {
      error = formatAPIError(e, "Failed to revoke token");
    }
  }

  async function cancelInvite() {
    const inv = cancelling;
    cancelling = null;
    if (!inv) return;
    try {
      unwrap(await api.DELETE("/v1/invites/{invite_id}", { params: { path: { invite_id: inv.id } } }));
      await load();
    } catch (e) {
      error = formatAPIError(e, "Failed to cancel invite");
    }
  }

  const statusTone: Record<string, string> = { active: "ok", pending: "beacon", revoked: "bad", expired: "idle" };
</script>

<svelte:head><title>Access · Sparrow</title></svelte:head>

<main class="mx-auto max-w-7xl px-4 sm:px-8 py-8 pb-24">
  <div class="flex flex-col sm:flex-row sm:items-end sm:justify-between gap-4 mb-6">
    <div>
      <p class="eyebrow mb-1.5">Settings / Access</p>
      <h1 class="text-2xl">Access</h1>
      <p class="text-sm text-muted mt-1">Who can use this Sparrow server. Invite people with a one-time link; give machines a token.</p>
    </div>
    <div class="flex items-center gap-2">
      <button class="btn btn-ghost" onclick={() => openDialog("token")}>Create token</button>
      <button class="btn btn-beacon" onclick={() => openDialog("invite")}><span class="text-lg leading-none" aria-hidden="true">+</span> Invite</button>
    </div>
  </div>

  {#if who && !who.auth_enabled}
    <div class="panel p-4 mb-6" role="note" style="border-color:color-mix(in srgb,var(--color-warn) 40%,transparent);background:color-mix(in srgb,var(--color-warn) 8%,var(--color-panel))">
      <p class="text-sm"><strong>Authentication is off.</strong> <span class="mono">SPARROW_API_KEY</span> is not set, so anyone who can reach Sparrow has full access and tokens are not checked. Set it to require a key or token.</p>
    </div>
  {/if}

  {#if error}
    <div class="panel p-4 mb-6" role="alert" style="border-color:color-mix(in srgb,var(--color-bad) 40%,transparent);background:color-mix(in srgb,var(--color-bad) 8%,var(--color-panel))">
      <p class="text-sm" style="color:var(--color-bad)">{error}</p>
    </div>
  {/if}

  <section class="mb-8" aria-labelledby="tokens-heading">
    <div class="flex items-center justify-between mb-3">
      <h2 id="tokens-heading" class="text-lg">Tokens</h2>
      <label class="flex items-center gap-2 text-xs text-muted">
        <input type="checkbox" bind:checked={showInactive} onchange={load} /> Show revoked and expired
      </label>
    </div>
    {#if loading}
      <div class="panel h-28 animate-pulse"></div>
    {:else if tokens.length === 0}
      <div class="panel"><EmptyState icon="link" title="No tokens yet" description="Invite someone, or create a token for a script or CI job." /></div>
    {:else}
      <div class="panel overflow-hidden">
        <div class="overflow-x-auto">
          <table class="w-full text-left">
            <thead>
              <tr class="border-b border-line">
                <th class="th">Name</th>
                <th class="th">Access</th>
                <th class="th hidden md:table-cell">Created by</th>
                <th class="th hidden sm:table-cell">Last used</th>
                <th class="th hidden lg:table-cell">Expires</th>
                <th class="th">Status</th>
                <th class="th"></th>
              </tr>
            </thead>
            <tbody>
              {#each tokens as t (t.id)}
                <tr class="row-line" data-testid="token-row">
                  <td class="td font-medium text-text">
                    {t.name}
                    {#if who?.token_id === t.id}<span class="chip ml-2">this browser</span>{/if}
                  </td>
                  <td class="td text-muted">{access(t.consumer)}</td>
                  <td class="td text-muted hidden md:table-cell">{t.created_by}</td>
                  <td class="td text-muted hidden sm:table-cell" title={t.last_used_at ?? ""}>{t.last_used_at ? timeAgo(t.last_used_at) : "Never"}</td>
                  <td class="td text-muted hidden lg:table-cell">{when(t.expires_at)}</td>
                  <td class="td"><span class="chip" style="color:var(--color-{statusTone[t.status] ?? 'idle'})">{t.status}</span></td>
                  <td class="td text-right whitespace-nowrap">
                    {#if t.status === "active"}
                      <button class="btn btn-danger !px-3 !py-1.5 text-xs" onclick={() => (revoking = t)} aria-label="Revoke token {t.name}">Revoke</button>
                    {/if}
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      </div>
    {/if}
  </section>

  <section aria-labelledby="invites-heading">
    <h2 id="invites-heading" class="text-lg mb-3">Pending invites</h2>
    {#if !loading && invites.length === 0}
      <p class="text-sm text-muted">No invites waiting to be used.</p>
    {:else if !loading}
      <div class="panel overflow-hidden">
        <div class="overflow-x-auto">
          <table class="w-full text-left">
            <thead>
              <tr class="border-b border-line">
                <th class="th">For</th>
                <th class="th">Access</th>
                <th class="th hidden md:table-cell">Invited by</th>
                <th class="th">Expires</th>
                <th class="th"></th>
              </tr>
            </thead>
            <tbody>
              {#each invites as inv (inv.id)}
                <tr class="row-line" data-testid="invite-row">
                  <td class="td font-medium text-text">{inv.name}</td>
                  <td class="td text-muted">{access(inv.consumer)}</td>
                  <td class="td text-muted hidden md:table-cell">{inv.created_by}</td>
                  <td class="td text-muted">{new Date(inv.expires_at).toLocaleString()}</td>
                  <td class="td text-right"><button class="btn btn-ghost !px-3 !py-1.5 text-xs" onclick={() => (cancelling = inv)} aria-label="Cancel invite for {inv.name}">Cancel</button></td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      </div>
    {/if}
  </section>
</main>

{#if dialog}
  <div class="fixed inset-0 z-50 flex items-center justify-center p-4" role="dialog" aria-modal="true" aria-labelledby="access-dialog-title" tabindex="-1" onkeydown={(e) => e.key === "Escape" && (dialog = null)}>
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div class="fixed inset-0 bg-black/40 backdrop-blur-sm" role="presentation" onclick={() => (dialog = null)}></div>
    <div class="panel relative w-full max-w-lg p-6">
      <p class="eyebrow mb-1.5">Access</p>
      <h3 id="access-dialog-title" class="text-lg font-semibold text-text mb-4">{dialog === "invite" ? "Invite someone" : "Create a token"}</h3>

      {#if result}
        <label class="block mb-2">
          <span class="field-label">{result.label}</span>
          <div class="flex gap-2">
            <input class="input mono flex-1" readonly value={result.value} aria-label={result.label} onfocus={(e) => e.currentTarget.select()} />
            <button class="btn btn-beacon" type="button" onclick={copy}>{copied ? "Copied" : "Copy"}</button>
          </div>
        </label>
        <p class="text-sm text-muted mb-5 leading-relaxed">{result.note}</p>
        <div class="flex justify-end"><button class="btn btn-ghost" onclick={() => (dialog = null)}>Done</button></div>
      {:else}
        <form onsubmit={create} class="space-y-4">
          <label class="block">
            <span class="field-label">{dialog === "invite" ? "Who is it for?" : "Name"}</span>
            <input class="input" bind:value={name} required maxlength="200" placeholder={dialog === "invite" ? "e.g. alice" : "e.g. ci-deploy"} />
          </label>
          <fieldset>
            <legend class="field-label">Access</legend>
            <label class="flex items-center gap-2 text-sm mb-1"><input type="radio" bind:group={scope} value="full" /> Full access (same as the API key)</label>
            <label class="flex items-center gap-2 text-sm"><input type="radio" bind:group={scope} value="consumer" /> One consumer's portal only</label>
            {#if scope === "consumer"}
              <input class="input mt-2" bind:value={consumer} required placeholder="consumer, e.g. acme" aria-label="Consumer" />
            {/if}
          </fieldset>
          {#if dialog === "invite"}
            <label class="block">
              <span class="field-label">Link expires after</span>
              <select class="input" bind:value={inviteTTL}>
                <option value={3600}>1 hour</option>
                <option value={24 * 3600}>24 hours</option>
                <option value={7 * 24 * 3600}>7 days</option>
              </select>
            </label>
          {/if}
          <label class="block">
            <span class="field-label">{dialog === "invite" ? "Their access expires" : "Expires"}</span>
            <select class="input" bind:value={tokenTTL}>
              {#each tokenTTLOptions as [secs, label]}<option value={secs}>{label}</option>{/each}
            </select>
          </label>
          {#if createError}<p class="text-sm" role="alert" style="color:var(--color-bad)">{createError}</p>{/if}
          <div class="flex justify-end gap-3 pt-2">
            <button type="button" class="btn btn-ghost" onclick={() => (dialog = null)}>Cancel</button>
            <button type="submit" class="btn btn-beacon" disabled={creating}>{creating ? "Creating…" : dialog === "invite" ? "Create invite link" : "Create token"}</button>
          </div>
        </form>
      {/if}
    </div>
  </div>
{/if}

<ConfirmDialog
  open={revoking !== null}
  title="Revoke token"
  message={`"${revoking?.name}" stops working right away (within 30 seconds on every server). This can't be undone.${who?.token_id === revoking?.id ? " This is the token this browser uses: you will be signed out." : ""}`}
  confirmLabel="Revoke"
  variant="danger"
  onconfirm={revoke}
  oncancel={() => (revoking = null)}
/>

<ConfirmDialog
  open={cancelling !== null}
  title="Cancel invite"
  message={`The invite link for "${cancelling?.name}" will stop working.`}
  confirmLabel="Cancel invite"
  cancelLabel="Keep"
  variant="warning"
  onconfirm={cancelInvite}
  oncancel={() => (cancelling = null)}
/>

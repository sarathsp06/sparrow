<script lang="ts">
  import { page } from '$app/state';
  import { goto } from '$app/navigation';
  import { onMount } from 'svelte';
  import { namespaceClient, namespaceMembershipClient, teamClient } from '$lib/services';
  import type { NamespaceResource, NamespaceMembership, TeamMember } from '../../../../../proto/webhook_pb.js';
  import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';
  import { getIsTenantAdmin, canManageNamespace } from '$lib/stores/auth.svelte.js';

  const namespaceName = $derived(page.params.namespace);

  let ns = $state<NamespaceResource | null>(null);
  let members = $state<NamespaceMembership[]>([]);
  let loading = $state(true);
  let error = $state('');

  // Edit mode
  let editing = $state(false);
  let editDescription = $state('');
  let saving = $state(false);

  // Add member form
  let showAddMember = $state(false);
  let newSubjectId = $state('');
  let newRole = $state('namespace:member');
  let addingMember = $state(false);

  // Remove member confirmation
  let confirmRemove = $state(false);
  let memberToRemove = $state<NamespaceMembership | null>(null);

  // User profile enrichment (from TeamService, best-effort)
  let userProfiles = $state<Map<string, TeamMember>>(new Map());

  const canManage = $derived.by(() => getIsTenantAdmin() || canManageNamespace(namespaceName));

  const roleOptions = [
    { value: 'namespace:admin', label: 'Admin', description: 'Full namespace access' },
    { value: 'namespace:member', label: 'Member', description: 'Read + write' },
    { value: 'namespace:viewer', label: 'Viewer', description: 'Read-only' },
  ];

  async function fetchNamespace() {
    loading = true;
    error = '';
    try {
      const [nsResp, membersResp] = await Promise.all([
        namespaceClient.getNamespace({ name: namespaceName }),
        namespaceMembershipClient.listNamespaceMembers({ namespace: namespaceName }),
      ]);
      ns = nsResp.namespace ?? null;
      members = membersResp.members ?? [];

      // Best-effort: enrich member display with names/emails/avatars from TeamService.
      // If the identity provider doesn't support it, we fall back to subjectId display.
      loadUserProfiles();
    } catch (e: unknown) {
      error = e instanceof Error ? e.message : 'Failed to load namespace';
    } finally {
      loading = false;
    }
  }

  async function loadUserProfiles() {
    try {
      const resp = await teamClient.listMembers({ pagination: { limit: 200, offset: 0 } });
      const map = new Map<string, TeamMember>();
      for (const m of resp.members) {
        map.set(m.userId, m);
      }
      userProfiles = map;
    } catch {
      // Silently ignore — team management may not be available
    }
  }

  function memberDisplayName(subjectId: string): string {
    const profile = userProfiles.get(subjectId);
    if (profile) {
      if (profile.firstName || profile.lastName) {
        return [profile.firstName, profile.lastName].filter(Boolean).join(' ');
      }
      if (profile.email) return profile.email;
    }
    return subjectId;
  }

  function memberEmail(subjectId: string): string {
    return userProfiles.get(subjectId)?.email ?? '';
  }

  function memberImageUrl(subjectId: string): string {
    return userProfiles.get(subjectId)?.imageUrl ?? '';
  }

  function memberInitials(subjectId: string): string {
    const profile = userProfiles.get(subjectId);
    if (profile) {
      if (profile.firstName && profile.lastName) {
        return (profile.firstName[0] + profile.lastName[0]).toUpperCase();
      }
      if (profile.firstName) return profile.firstName.slice(0, 2).toUpperCase();
      if (profile.email) return profile.email.slice(0, 2).toUpperCase();
    }
    return subjectId.slice(0, 2).toUpperCase();
  }

  function startEdit() {
    editDescription = ns?.description ?? '';
    editing = true;
  }

  async function saveEdit() {
    if (!ns) return;
    saving = true;
    error = '';
    try {
      const resp = await namespaceClient.updateNamespace({
        id: ns.id,
        description: editDescription.trim(),
      });
      if (resp.namespace) {
        ns = resp.namespace;
      }
      editing = false;
    } catch (e: unknown) {
      error = e instanceof Error ? e.message : 'Failed to update namespace';
    } finally {
      saving = false;
    }
  }

  async function handleAddMember() {
    if (!newSubjectId.trim()) return;
    addingMember = true;
    error = '';
    try {
      const resp = await namespaceMembershipClient.assignNamespaceRole({
        subjectId: newSubjectId.trim(),
        namespace: namespaceName,
        role: newRole,
      });
      if (resp.membership) {
        // Replace existing or add new
        const idx = members.findIndex(m => m.subjectId === resp.membership!.subjectId);
        if (idx >= 0) {
          members = [...members.slice(0, idx), resp.membership, ...members.slice(idx + 1)];
        } else {
          members = [...members, resp.membership];
        }
        newSubjectId = '';
        newRole = 'namespace:member';
        showAddMember = false;
      }
    } catch (e: unknown) {
      error = e instanceof Error ? e.message : 'Failed to add member';
    } finally {
      addingMember = false;
    }
  }

  function promptRemoveMember(member: NamespaceMembership) {
    memberToRemove = member;
    confirmRemove = true;
  }

  async function handleRemoveMember() {
    if (!memberToRemove) return;
    try {
      await namespaceMembershipClient.removeNamespaceRole({
        subjectId: memberToRemove.subjectId,
        namespace: namespaceName,
      });
      members = members.filter(m => m.id !== memberToRemove!.id);
      confirmRemove = false;
      memberToRemove = null;
    } catch (e: unknown) {
      error = e instanceof Error ? e.message : 'Failed to remove member';
      confirmRemove = false;
    }
  }

  function formatDate(ts: { seconds?: bigint } | undefined): string {
    if (!ts?.seconds) return '—';
    return new Date(Number(ts.seconds) * 1000).toLocaleDateString('en-US', {
      year: 'numeric', month: 'short', day: 'numeric',
    });
  }

  function roleBadgeClass(role: string): string {
    switch (role) {
      case 'namespace:admin': return 'bg-purple-100 text-purple-700';
      case 'namespace:member': return 'bg-blue-100 text-blue-700';
      case 'namespace:viewer': return 'bg-gray-100 text-gray-600';
      default: return 'bg-gray-100 text-gray-600';
    }
  }

  function roleLabel(role: string): string {
    return role.replace('namespace:', '');
  }

  onMount(fetchNamespace);
</script>

<svelte:head>
  <title>{namespaceName} | Namespaces | Sparrow</title>
</svelte:head>

<div class="max-w-6xl mx-auto px-4 py-8 font-inter">
  <!-- Breadcrumb -->
  <nav class="text-sm text-gray-400 mb-6">
    <button onclick={() => goto('/namespaces')} class="hover:text-gray-600 transition">Namespaces</button>
    <span class="mx-2">/</span>
    <span class="text-gray-700 font-medium">{namespaceName}</span>
  </nav>

  {#if error}
    <div class="bg-red-50 text-red-700 px-4 py-3 rounded-lg mb-6 text-sm">{error}</div>
  {/if}

  {#if loading}
    <div class="text-center py-12 text-gray-400">Loading namespace...</div>
  {:else if !ns}
    <div class="text-center py-12 text-gray-500">Namespace not found.</div>
  {:else}
    <!-- Namespace details -->
    <div class="bg-white border border-gray-200 rounded-xl p-6 mb-8 shadow-sm">
      <div class="flex items-start justify-between mb-4">
        <div>
          <h1 class="text-2xl font-bold text-gray-900">{ns.name}</h1>
          <p class="text-xs text-gray-400 mt-1 font-fira">ID: {ns.id}</p>
        </div>
        {#if canManage && !editing}
          <button
            onclick={startEdit}
            class="px-3 py-1.5 text-sm bg-gray-100 text-gray-700 rounded-lg hover:bg-gray-200 transition"
          >
            Edit
          </button>
        {/if}
      </div>

      {#if editing}
        <form onsubmit={(e) => { e.preventDefault(); saveEdit(); }} class="space-y-3">
          <div>
            <label for="edit-desc" class="block text-sm font-medium text-gray-700 mb-1">Description</label>
            <input
              id="edit-desc"
              type="text"
              bind:value={editDescription}
              class="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-gray-900/20"
            />
          </div>
          <div class="flex gap-2">
            <button
              type="submit"
              disabled={saving}
              class="px-3 py-1.5 text-sm bg-gray-900 text-white rounded-lg hover:bg-gray-800 transition disabled:opacity-50"
            >
              {saving ? 'Saving...' : 'Save'}
            </button>
            <button
              type="button"
              onclick={() => (editing = false)}
              class="px-3 py-1.5 text-sm bg-gray-100 text-gray-700 rounded-lg hover:bg-gray-200 transition"
            >
              Cancel
            </button>
          </div>
        </form>
      {:else}
        <p class="text-gray-600 text-sm">{ns.description || 'No description.'}</p>
      {/if}

      <div class="flex gap-6 mt-4 text-xs text-gray-400">
        <span>Created {formatDate(ns.createdAt)}</span>
        <span>Updated {formatDate(ns.updatedAt)}</span>
      </div>
    </div>

    <!-- Members section -->
    <div class="bg-white border border-gray-200 rounded-xl p-6 shadow-sm">
      <div class="flex items-center justify-between mb-6">
        <div>
          <h2 class="text-xl font-semibold text-gray-900">Members</h2>
          <p class="text-sm text-gray-500 mt-1">Users with direct access to this namespace.</p>
        </div>
        {#if canManage}
          <button
            onclick={() => (showAddMember = !showAddMember)}
            class="px-3 py-1.5 text-sm bg-gray-900 text-white rounded-lg hover:bg-gray-800 transition font-medium"
          >
            {showAddMember ? 'Cancel' : '+ Add Member'}
          </button>
        {/if}
      </div>

      <!-- Add member form -->
      {#if showAddMember}
        <div class="bg-gray-50 rounded-lg p-4 mb-6 border border-gray-100">
          <form onsubmit={(e) => { e.preventDefault(); handleAddMember(); }} class="flex flex-col sm:flex-row gap-3">
            <div class="flex-1">
              <input
                type="text"
                bind:value={newSubjectId}
                placeholder="User subject ID (e.g., user_abc123)"
                class="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-gray-900/20"
                required
              />
            </div>
            <div>
              <select
                bind:value={newRole}
                class="px-3 py-2 border border-gray-300 rounded-lg text-sm bg-white focus:outline-none focus:ring-2 focus:ring-gray-900/20"
              >
                {#each roleOptions as opt}
                  <option value={opt.value}>{opt.label} — {opt.description}</option>
                {/each}
              </select>
            </div>
            <button
              type="submit"
              disabled={addingMember || !newSubjectId.trim()}
              class="px-4 py-2 bg-gray-900 text-white rounded-lg hover:bg-gray-800 transition text-sm font-medium disabled:opacity-50 whitespace-nowrap"
            >
              {addingMember ? 'Adding...' : 'Add'}
            </button>
          </form>
        </div>
      {/if}

      <!-- Members list -->
      {#if members.length === 0}
        <p class="text-gray-400 text-sm text-center py-8">
          No members assigned. Users with tenant-wide roles can still access this namespace.
        </p>
      {:else}
        <div class="divide-y divide-gray-100">
          {#each members as member}
            <div class="flex items-center justify-between py-3">
              <div class="flex items-center gap-3">
                {#if memberImageUrl(member.subjectId)}
                  <img
                    src={memberImageUrl(member.subjectId)}
                    alt=""
                    class="w-8 h-8 rounded-full object-cover"
                  />
                {:else}
                  <div class="w-8 h-8 rounded-full bg-gray-200 flex items-center justify-center text-xs font-medium text-gray-600">
                    {memberInitials(member.subjectId)}
                  </div>
                {/if}
                <div>
                  <p class="text-sm font-medium text-gray-900">{memberDisplayName(member.subjectId)}</p>
                  {#if memberEmail(member.subjectId) && memberDisplayName(member.subjectId) !== memberEmail(member.subjectId)}
                    <p class="text-xs text-gray-400">{memberEmail(member.subjectId)}</p>
                  {/if}
                  {#if memberDisplayName(member.subjectId) !== member.subjectId}
                    <p class="text-xs text-gray-400 font-fira">{member.subjectId}</p>
                  {/if}
                  <p class="text-xs text-gray-400">Added {formatDate(member.createdAt)}</p>
                </div>
              </div>
              <div class="flex items-center gap-3">
                <span class="px-2 py-0.5 text-xs font-medium rounded-full {roleBadgeClass(member.role)}">
                  {roleLabel(member.role)}
                </span>
                {#if canManage}
                  <button
                    onclick={() => promptRemoveMember(member)}
                    class="text-xs text-red-500 hover:text-red-700 transition"
                  >
                    Remove
                  </button>
                {/if}
              </div>
            </div>
          {/each}
        </div>
      {/if}
    </div>
  {/if}
</div>

<!-- Remove member confirmation -->
<ConfirmDialog
  open={confirmRemove}
  title="Remove Member"
  message={`Remove "${memberToRemove ? memberDisplayName(memberToRemove.subjectId) : ''}" from namespace "${namespaceName}"? They will lose direct access.`}
  confirmLabel="Remove"
  variant="danger"
  onconfirm={handleRemoveMember}
  oncancel={() => { confirmRemove = false; memberToRemove = null; }}
/>

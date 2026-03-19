<!--
  Team / Organisation Management page.

  Custom team management UI that uses the TeamService API to list members,
  invite new members by email, update roles, remove members, and manage
  pending invitations. Works with any identity provider that implements
  the TeamManager interface (currently Clerk).

  When no identity provider supports team management, shows a helpful message.
-->
<script lang="ts">
  import { onMount } from 'svelte';
  import { teamClient } from '$lib/services';
  import type { TeamMember, TeamInvitation } from '../../../../proto/webhook_pb.js';
  import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';
  import EmptyState from '$lib/components/EmptyState.svelte';
  import { getIsTenantAdmin } from '$lib/stores/auth.svelte.js';

  // Data state
  let membersList = $state<TeamMember[]>([]);
  let invitationsList = $state<TeamInvitation[]>([]);
  let totalMembers = $state(0);
  let totalInvitations = $state(0);
  let loading = $state(true);
  let error = $state('');
  let unsupported = $state(false);

  // Invite form
  let showInviteForm = $state(false);
  let inviteEmail = $state('');
  let inviteRole = $state('org:member');
  let inviting = $state(false);
  let inviteError = $state('');

  // Remove member confirmation
  let confirmRemove = $state(false);
  let memberToRemove = $state<TeamMember | null>(null);

  // Revoke invitation confirmation
  let confirmRevoke = $state(false);
  let invitationToRevoke = $state<TeamInvitation | null>(null);

  // Role editing
  let editingRoleUserId = $state<string | null>(null);
  let editingRoleValue = $state('');
  let updatingRole = $state(false);

  // Active tab
  let activeTab = $state<'members' | 'invitations'>('members');

  const isAdmin = $derived.by(() => getIsTenantAdmin());

  const roleOptions = [
    { value: 'org:admin', label: 'Admin', description: 'Full organization access' },
    { value: 'org:member', label: 'Member', description: 'Standard member access' },
  ];

  function roleBadgeClass(role: string): string {
    if (role.includes('admin')) return 'bg-purple-100 text-purple-700';
    return 'bg-blue-100 text-blue-700';
  }

  function roleLabel(role: string): string {
    return role.replace('org:', '');
  }

  function formatDate(ts: { seconds?: bigint } | undefined): string {
    if (!ts?.seconds) return '\u2014';
    return new Date(Number(ts.seconds) * 1000).toLocaleDateString('en-US', {
      year: 'numeric', month: 'short', day: 'numeric',
    });
  }

  function memberInitials(m: TeamMember): string {
    if (m.firstName && m.lastName) {
      return (m.firstName[0] + m.lastName[0]).toUpperCase();
    }
    if (m.firstName) return m.firstName.slice(0, 2).toUpperCase();
    if (m.email) return m.email.slice(0, 2).toUpperCase();
    return m.userId.slice(0, 2).toUpperCase();
  }

  function memberDisplayName(m: TeamMember): string {
    if (m.firstName || m.lastName) {
      return [m.firstName, m.lastName].filter(Boolean).join(' ');
    }
    return m.email || m.userId;
  }

  async function fetchData() {
    loading = true;
    error = '';
    unsupported = false;
    try {
      const [membersResp, invitationsResp] = await Promise.all([
        teamClient.listMembers({ pagination: { limit: 100, offset: 0 } }),
        teamClient.listInvitations({ status: 'pending', pagination: { limit: 100, offset: 0 } }),
      ]);
      membersList = membersResp.members ?? [];
      totalMembers = membersResp.totalCount;
      invitationsList = invitationsResp.invitations ?? [];
      totalInvitations = invitationsResp.totalCount;
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : String(e);
      // If the backend returns Unimplemented, team management is not available
      if (msg.includes('Unimplemented') || msg.includes('not available') || msg.includes('unimplemented')) {
        unsupported = true;
      } else {
        error = msg;
      }
    } finally {
      loading = false;
    }
  }

  async function handleInvite() {
    if (!inviteEmail.trim()) return;
    inviting = true;
    inviteError = '';
    try {
      const resp = await teamClient.inviteMember({
        email: inviteEmail.trim(),
        role: inviteRole,
      });
      if (resp.invitation) {
        invitationsList = [...invitationsList, resp.invitation];
        totalInvitations += 1;
      }
      inviteEmail = '';
      inviteRole = 'org:member';
      showInviteForm = false;
    } catch (e: unknown) {
      inviteError = e instanceof Error ? e.message : 'Failed to send invitation';
    } finally {
      inviting = false;
    }
  }

  function promptRemoveMember(member: TeamMember) {
    memberToRemove = member;
    confirmRemove = true;
  }

  async function handleRemoveMember() {
    if (!memberToRemove) return;
    try {
      await teamClient.removeMember({ userId: memberToRemove.userId });
      membersList = membersList.filter(m => m.userId !== memberToRemove!.userId);
      totalMembers -= 1;
      confirmRemove = false;
      memberToRemove = null;
    } catch (e: unknown) {
      error = e instanceof Error ? e.message : 'Failed to remove member';
      confirmRemove = false;
    }
  }

  function startEditRole(member: TeamMember) {
    editingRoleUserId = member.userId;
    editingRoleValue = member.role;
  }

  function cancelEditRole() {
    editingRoleUserId = null;
    editingRoleValue = '';
  }

  async function saveRole(member: TeamMember) {
    if (editingRoleValue === member.role) {
      cancelEditRole();
      return;
    }
    updatingRole = true;
    try {
      const resp = await teamClient.updateMemberRole({
        userId: member.userId,
        role: editingRoleValue,
      });
      if (resp.member) {
        const idx = membersList.findIndex(m => m.userId === member.userId);
        if (idx >= 0) {
          membersList = [...membersList.slice(0, idx), resp.member, ...membersList.slice(idx + 1)];
        }
      }
      cancelEditRole();
    } catch (e: unknown) {
      error = e instanceof Error ? e.message : 'Failed to update role';
    } finally {
      updatingRole = false;
    }
  }

  function promptRevokeInvitation(inv: TeamInvitation) {
    invitationToRevoke = inv;
    confirmRevoke = true;
  }

  async function handleRevokeInvitation() {
    if (!invitationToRevoke) return;
    try {
      await teamClient.revokeInvitation({ invitationId: invitationToRevoke.id });
      invitationsList = invitationsList.filter(i => i.id !== invitationToRevoke!.id);
      totalInvitations -= 1;
      confirmRevoke = false;
      invitationToRevoke = null;
    } catch (e: unknown) {
      error = e instanceof Error ? e.message : 'Failed to revoke invitation';
      confirmRevoke = false;
    }
  }

  onMount(fetchData);
</script>

<svelte:head>
  <title>Team | Sparrow</title>
</svelte:head>

<div class="max-w-6xl mx-auto px-4 py-8 font-inter">
  <!-- Header -->
  <div class="mb-8">
    <h1 class="text-2xl font-bold text-gray-900">Team</h1>
    <p class="text-sm text-gray-500 mt-1">Manage organization members and invitations.</p>
  </div>

  {#if error}
    <div class="bg-red-50 border border-red-200 rounded-lg p-4 mb-6">
      <div class="flex items-start gap-3">
        <span class="material-symbols-outlined text-red-500 text-xl mt-0.5">error</span>
        <div>
          <p class="text-sm font-medium text-red-800">{error}</p>
          <button onclick={fetchData} class="text-sm text-red-600 hover:text-red-800 underline mt-1">Retry</button>
        </div>
      </div>
    </div>
  {/if}

  {#if loading}
    <div class="text-center py-12 text-gray-400">Loading team...</div>
  {:else if unsupported}
    <div class="flex items-center justify-center min-h-[60vh]">
      <div class="text-center">
        <span class="material-symbols-outlined text-gray-300 text-5xl mb-4">group_off</span>
        <h2 class="text-xl font-bold text-gray-700 mb-2">Team Management Not Available</h2>
        <p class="text-gray-500 max-w-md mx-auto">
          Team management requires an identity provider that supports it.
          Configure Clerk or a compatible provider to enable member and invitation management.
        </p>
      </div>
    </div>
  {:else}
    <!-- Stats -->
    <div class="grid grid-cols-2 gap-4 mb-6">
      <div class="bg-white border border-gray-200 rounded-xl p-4 shadow-sm">
        <p class="text-xs font-medium text-gray-500 uppercase tracking-wider">Members</p>
        <p class="text-2xl font-bold text-gray-900 mt-1">{totalMembers}</p>
      </div>
      <div class="bg-white border border-gray-200 rounded-xl p-4 shadow-sm">
        <p class="text-xs font-medium text-gray-500 uppercase tracking-wider">Pending Invitations</p>
        <p class="text-2xl font-bold text-gray-900 mt-1">{totalInvitations}</p>
      </div>
    </div>

    <!-- Tabs -->
    <div class="flex items-center gap-1 bg-gray-100 rounded-lg p-1 mb-6 w-fit">
      <button
        onclick={() => { activeTab = 'members'; }}
        class="px-4 py-1.5 text-sm font-medium rounded-md transition {activeTab === 'members' ? 'bg-white text-gray-900 shadow-sm' : 'text-gray-500 hover:text-gray-700'}"
      >
        Members ({totalMembers})
      </button>
      <button
        onclick={() => { activeTab = 'invitations'; }}
        class="px-4 py-1.5 text-sm font-medium rounded-md transition {activeTab === 'invitations' ? 'bg-white text-gray-900 shadow-sm' : 'text-gray-500 hover:text-gray-700'}"
      >
        Invitations ({totalInvitations})
      </button>
    </div>

    <!-- Members Tab -->
    {#if activeTab === 'members'}
      <div class="bg-white border border-gray-200 rounded-xl shadow-sm">
        <div class="flex items-center justify-between px-6 py-4 border-b border-gray-100">
          <h2 class="text-lg font-semibold text-gray-900">Members</h2>
          {#if isAdmin}
            <button
              onclick={() => { showInviteForm = !showInviteForm; inviteError = ''; }}
              class="px-3 py-1.5 text-sm bg-gray-900 text-white rounded-lg hover:bg-gray-800 transition font-medium"
            >
              {showInviteForm ? 'Cancel' : '+ Invite Member'}
            </button>
          {/if}
        </div>

        <!-- Invite form -->
        {#if showInviteForm}
          <div class="bg-gray-50 px-6 py-4 border-b border-gray-100">
            {#if inviteError}
              <div class="bg-red-50 text-red-700 px-3 py-2 rounded-lg mb-3 text-sm">{inviteError}</div>
            {/if}
            <form onsubmit={(e) => { e.preventDefault(); handleInvite(); }} class="flex flex-col sm:flex-row gap-3">
              <div class="flex-1">
                <input
                  type="email"
                  bind:value={inviteEmail}
                  placeholder="Email address"
                  class="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-gray-900/20"
                  required
                />
              </div>
              <div>
                <select
                  bind:value={inviteRole}
                  class="px-3 py-2 border border-gray-300 rounded-lg text-sm bg-white focus:outline-none focus:ring-2 focus:ring-gray-900/20"
                >
                  {#each roleOptions as opt}
                    <option value={opt.value}>{opt.label} — {opt.description}</option>
                  {/each}
                </select>
              </div>
              <button
                type="submit"
                disabled={inviting || !inviteEmail.trim()}
                class="px-4 py-2 bg-gray-900 text-white rounded-lg hover:bg-gray-800 transition text-sm font-medium disabled:opacity-50 whitespace-nowrap"
              >
                {inviting ? 'Sending...' : 'Send Invite'}
              </button>
            </form>
          </div>
        {/if}

        <!-- Members list -->
        <div class="px-6">
          {#if membersList.length === 0}
            <div class="py-8">
              <EmptyState
                icon="group"
                title="No members"
                description="Invite members by email to get started."
              >
                {#snippet action()}
                  {#if isAdmin}
                    <button
                      onclick={() => { showInviteForm = true; }}
                      class="px-4 py-2 bg-gray-900 text-white rounded-lg hover:bg-gray-800 transition text-sm font-medium"
                    >
                      + Invite Member
                    </button>
                  {/if}
                {/snippet}
              </EmptyState>
            </div>
          {:else}
            <div class="divide-y divide-gray-100">
              {#each membersList as member}
                <div class="flex items-center justify-between py-4">
                  <div class="flex items-center gap-3">
                    {#if member.imageUrl}
                      <img
                        src={member.imageUrl}
                        alt=""
                        class="w-9 h-9 rounded-full object-cover"
                      />
                    {:else}
                      <div class="w-9 h-9 rounded-full bg-gray-200 flex items-center justify-center text-xs font-medium text-gray-600">
                        {memberInitials(member)}
                      </div>
                    {/if}
                    <div>
                      <p class="text-sm font-medium text-gray-900">{memberDisplayName(member)}</p>
                      {#if (member.firstName || member.lastName) && member.email}
                        <p class="text-xs text-gray-400">{member.email}</p>
                      {/if}
                      <p class="text-xs text-gray-400">Joined {formatDate(member.joinedAt)}</p>
                    </div>
                  </div>
                  <div class="flex items-center gap-3">
                    {#if editingRoleUserId === member.userId}
                      <div class="flex items-center gap-2">
                        <select
                          bind:value={editingRoleValue}
                          class="px-2 py-1 border border-gray-300 rounded-md text-xs bg-white focus:outline-none focus:ring-2 focus:ring-gray-900/20"
                        >
                          {#each roleOptions as opt}
                            <option value={opt.value}>{opt.label}</option>
                          {/each}
                        </select>
                        <button
                          onclick={() => saveRole(member)}
                          disabled={updatingRole}
                          class="text-xs text-gray-900 font-medium hover:text-gray-700 disabled:opacity-50"
                        >
                          {updatingRole ? '...' : 'Save'}
                        </button>
                        <button
                          onclick={cancelEditRole}
                          class="text-xs text-gray-400 hover:text-gray-600"
                        >
                          Cancel
                        </button>
                      </div>
                    {:else}
                      <span class="px-2 py-0.5 text-xs font-medium rounded-full {roleBadgeClass(member.role)}">
                        {roleLabel(member.role)}
                      </span>
                      {#if isAdmin}
                        <button
                          onclick={() => startEditRole(member)}
                          class="text-xs text-gray-400 hover:text-gray-600 transition"
                          title="Change role"
                        >
                          <span class="material-symbols-outlined text-base">edit</span>
                        </button>
                        <button
                          onclick={() => promptRemoveMember(member)}
                          class="text-xs text-red-500 hover:text-red-700 transition"
                          title="Remove member"
                        >
                          Remove
                        </button>
                      {/if}
                    {/if}
                  </div>
                </div>
              {/each}
            </div>
          {/if}
        </div>
      </div>
    {/if}

    <!-- Invitations Tab -->
    {#if activeTab === 'invitations'}
      <div class="bg-white border border-gray-200 rounded-xl shadow-sm">
        <div class="flex items-center justify-between px-6 py-4 border-b border-gray-100">
          <h2 class="text-lg font-semibold text-gray-900">Pending Invitations</h2>
          {#if isAdmin}
            <button
              onclick={() => { activeTab = 'members'; showInviteForm = true; inviteError = ''; }}
              class="px-3 py-1.5 text-sm bg-gray-900 text-white rounded-lg hover:bg-gray-800 transition font-medium"
            >
              + Invite Member
            </button>
          {/if}
        </div>

        <div class="px-6">
          {#if invitationsList.length === 0}
            <div class="py-8">
              <EmptyState
                icon="mail"
                title="No pending invitations"
                description="All invitations have been accepted or expired."
              />
            </div>
          {:else}
            <div class="divide-y divide-gray-100">
              {#each invitationsList as inv}
                <div class="flex items-center justify-between py-4">
                  <div class="flex items-center gap-3">
                    <div class="w-9 h-9 rounded-full bg-amber-50 flex items-center justify-center">
                      <span class="material-symbols-outlined text-amber-500 text-lg">mail</span>
                    </div>
                    <div>
                      <p class="text-sm font-medium text-gray-900">{inv.email}</p>
                      <p class="text-xs text-gray-400">Invited {formatDate(inv.createdAt)}</p>
                      {#if inv.expiresAt?.seconds}
                        <p class="text-xs text-gray-400">Expires {formatDate(inv.expiresAt)}</p>
                      {/if}
                    </div>
                  </div>
                  <div class="flex items-center gap-3">
                    <span class="px-2 py-0.5 text-xs font-medium rounded-full {roleBadgeClass(inv.role)}">
                      {roleLabel(inv.role)}
                    </span>
                    <span class="px-2 py-0.5 text-xs font-medium rounded-full bg-amber-50 text-amber-700">
                      {inv.status || 'pending'}
                    </span>
                    {#if isAdmin}
                      <button
                        onclick={() => promptRevokeInvitation(inv)}
                        class="text-xs text-red-500 hover:text-red-700 transition"
                      >
                        Revoke
                      </button>
                    {/if}
                  </div>
                </div>
              {/each}
            </div>
          {/if}
        </div>
      </div>
    {/if}
  {/if}
</div>

<!-- Remove member confirmation -->
<ConfirmDialog
  open={confirmRemove}
  title="Remove Member"
  message={`Remove ${memberToRemove ? memberDisplayName(memberToRemove) : ''} from the organization? They will lose all access.`}
  confirmLabel="Remove"
  variant="danger"
  onconfirm={handleRemoveMember}
  oncancel={() => { confirmRemove = false; memberToRemove = null; }}
/>

<!-- Revoke invitation confirmation -->
<ConfirmDialog
  open={confirmRevoke}
  title="Revoke Invitation"
  message={`Revoke the invitation for ${invitationToRevoke?.email ?? ''}? They will no longer be able to join.`}
  confirmLabel="Revoke"
  variant="danger"
  onconfirm={handleRevokeInvitation}
  oncancel={() => { confirmRevoke = false; invitationToRevoke = null; }}
/>

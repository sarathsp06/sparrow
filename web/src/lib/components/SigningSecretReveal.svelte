<script lang="ts">
  // Shows a webhook signing secret the server returns in plaintext exactly
  // once (at registration or rotation). Every later read is masked, so this is
  // the user's only chance to copy it.
  interface Props {
    secret: string;
    publicKey?: string;
  }

  let { secret, publicKey = '' }: Props = $props();
  let copied = $state('');

  async function copy(label: string, value: string) {
    try {
      await navigator.clipboard.writeText(value);
      copied = label;
      setTimeout(() => { if (copied === label) copied = ''; }, 1500);
    } catch {
      // Clipboard blocked: the value stays selectable on screen.
    }
  }
</script>

<div class="space-y-3">
  <div>
    <span class="field-label">Signing secret (HMAC)</span>
    <div class="flex items-center gap-2">
      <code class="text-xs mono panel-2 px-2 py-1.5 rounded break-all text-text flex-1 select-all" data-testid="signing-secret">{secret}</code>
      <button type="button" onclick={() => copy('secret', secret)} class="btn btn-ghost !px-2 !py-1 !text-xs shrink-0">{copied === 'secret' ? 'Copied' : 'Copy'}</button>
    </div>
  </div>
  {#if publicKey}
    <div>
      <span class="field-label">Ed25519 public key</span>
      <div class="flex items-center gap-2">
        <code class="text-xs mono panel-2 px-2 py-1.5 rounded break-all text-text flex-1 select-all">{publicKey}</code>
        <button type="button" onclick={() => copy('key', publicKey)} class="btn btn-ghost !px-2 !py-1 !text-xs shrink-0">{copied === 'key' ? 'Copied' : 'Copy'}</button>
      </div>
      <p class="text-[10px] text-faint mt-0.5">Verifies the v1a, signature. Not secret, and always shown on the webhook page.</p>
    </div>
  {/if}
  <p class="text-sm" style="color:var(--color-warn)">Copy the signing secret now and give it to the receiver. You won't see it again: Sparrow only stores it encrypted and masks it on every later read.</p>
</div>

<script lang="ts">
  import { WebhookHealth } from '../../../../proto/webhook_pb.js';

  interface Props {
    health: WebhookHealth;
    size?: 'sm' | 'md';
  }

  let { health, size = 'sm' }: Props = $props();

  const config: Record<WebhookHealth, { bg: string; text: string; dot: string; label: string }> = {
    [WebhookHealth.HEALTH_UNSPECIFIED]: { bg: 'bg-gray-100', text: 'text-gray-600', dot: 'bg-gray-400', label: 'Unknown' },
    [WebhookHealth.HEALTH_HEALTHY]:    { bg: 'bg-green-50', text: 'text-green-700', dot: 'bg-green-500', label: 'Healthy' },
    [WebhookHealth.HEALTH_DEGRADED]:   { bg: 'bg-yellow-50', text: 'text-yellow-700', dot: 'bg-yellow-500', label: 'Degraded' },
    [WebhookHealth.HEALTH_UNHEALTHY]:  { bg: 'bg-red-50', text: 'text-red-700', dot: 'bg-red-500', label: 'Unhealthy' },
  };

  const sizeClasses = {
    sm: 'px-2 py-0.5 text-xs',
    md: 'px-2.5 py-1 text-sm',
  };

  const dotSize = {
    sm: 'w-1.5 h-1.5',
    md: 'w-2 h-2',
  };
</script>

<span class="inline-flex items-center gap-1.5 rounded-full font-medium {config[health].bg} {config[health].text} {sizeClasses[size]}">
  <span class="rounded-full {config[health].dot} {dotSize[size]}"></span>
  {config[health].label}
</span>

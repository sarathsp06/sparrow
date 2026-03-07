<script lang="ts">
  import { WebhookDeliveryStatus } from '../../../../proto/webhook_pb.js';

  interface Props {
    status: WebhookDeliveryStatus;
  }

  let { status }: Props = $props();

  const config: Record<WebhookDeliveryStatus, { bg: string; text: string; label: string }> = {
    [WebhookDeliveryStatus.DELIVERY_UNSPECIFIED]: { bg: 'bg-gray-100', text: 'text-gray-600', label: 'Unknown' },
    [WebhookDeliveryStatus.DELIVERY_PENDING]:     { bg: 'bg-yellow-50', text: 'text-yellow-700', label: 'Pending' },
    [WebhookDeliveryStatus.DELIVERY_SENDING]:     { bg: 'bg-blue-50', text: 'text-blue-700', label: 'Sending' },
    [WebhookDeliveryStatus.DELIVERY_SUCCESS]:     { bg: 'bg-green-50', text: 'text-green-700', label: 'Success' },
    [WebhookDeliveryStatus.DELIVERY_FAILED]:      { bg: 'bg-red-50', text: 'text-red-700', label: 'Failed' },
    [WebhookDeliveryStatus.DELIVERY_RETRYING]:    { bg: 'bg-orange-50', text: 'text-orange-700', label: 'Retrying' },
    [WebhookDeliveryStatus.DELIVERY_EXPIRED]:     { bg: 'bg-gray-100', text: 'text-gray-500', label: 'Expired' },
  };
</script>

<span class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium {config[status].bg} {config[status].text}">
  {config[status].label}
</span>

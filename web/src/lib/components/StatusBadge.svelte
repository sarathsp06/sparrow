<script lang="ts">
  interface Props {
    status: string;
  }

  let { status }: Props = $props();

  const config: Record<string, { bg: string; text: string; label: string }> = {
    pending:   { bg: 'bg-yellow-50', text: 'text-yellow-700', label: 'Pending' },
    sending:   { bg: 'bg-blue-50', text: 'text-blue-700', label: 'Sending' },
    success:   { bg: 'bg-green-50', text: 'text-green-700', label: 'Success' },
    failed:    { bg: 'bg-red-50', text: 'text-red-700', label: 'Failed' },
    retrying:  { bg: 'bg-orange-50', text: 'text-orange-700', label: 'Retrying' },
    expired:   { bg: 'bg-gray-100', text: 'text-gray-500', label: 'Expired' },
  };

  const fallback = { bg: 'bg-gray-100', text: 'text-gray-600', label: 'Unknown' };
</script>

<span class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium {(config[status] ?? fallback).bg} {(config[status] ?? fallback).text}">
  {(config[status] ?? fallback).label}
</span>

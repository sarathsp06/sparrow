<script lang="ts">
  interface Props {
    health: string;
    size?: 'sm' | 'md';
  }

  let { health, size = 'sm' }: Props = $props();

  const config: Record<string, { bg: string; text: string; dot: string; label: string; tooltip?: string }> = {
    unknown:   { bg: 'bg-gray-100', text: 'text-gray-600', dot: 'bg-gray-400', label: '\u2014', tooltip: 'Health is calculated after the first deliveries' },
    healthy:   { bg: 'bg-green-50', text: 'text-green-700', dot: 'bg-green-500', label: 'Healthy' },
    degraded:  { bg: 'bg-yellow-50', text: 'text-yellow-700', dot: 'bg-yellow-500', label: 'Degraded' },
    unhealthy: { bg: 'bg-red-50', text: 'text-red-700', dot: 'bg-red-500', label: 'Unhealthy' },
  };

  const fallback = config.unknown;

  const sizeClasses = {
    sm: 'px-2 py-0.5 text-xs',
    md: 'px-2.5 py-1 text-sm',
  };

  const dotSize = {
    sm: 'w-1.5 h-1.5',
    md: 'w-2 h-2',
  };
</script>

<span class="inline-flex items-center gap-1.5 rounded-full font-medium {(config[health] ?? fallback).bg} {(config[health] ?? fallback).text} {sizeClasses[size]}" title={(config[health] ?? fallback).tooltip ?? ''}>
  <span class="rounded-full {(config[health] ?? fallback).dot} {dotSize[size]}"></span>
  {(config[health] ?? fallback).label}
</span>

<script lang="ts">
  import { toasts } from '../toast.svelte';
</script>

<div class="toast-region" role="status" aria-live="polite">
  {#each toasts as t (t.id)}
    <div class="toast" class:error={t.kind === 'error'}>
      <span class="toast-dot"></span>
      {t.message}
    </div>
  {/each}
</div>

<script lang="ts">
  import Icon from './Icon.svelte';

  interface PlanSummary {
    id: number;
    name: string;
    regionCount: number;
    createdAt: string;
    updatedAt: string;
  }

  interface Props {
    plans: PlanSummary[];
    loading: boolean;
    onSelect: (id: number) => void;
    onDelete: (id: number) => void;
    onClose: () => void;
  }

  let { plans, loading, onSelect, onDelete, onClose }: Props = $props();

  function formatDate(iso: string): string {
    try {
      return new Date(iso).toLocaleDateString('pt-BR', { day: '2-digit', month: '2-digit', year: 'numeric' });
    } catch {
      return iso;
    }
  }
</script>

<div class="plan-grid-overlay">
  <div class="plan-grid-panel">
    <div class="plan-grid-header">
      <h3>Planos de pintura</h3>
      <button class="icon-btn" onclick={onClose}>
        <Icon name="close" size={18} />
      </button>
    </div>

    {#if loading}
      <div class="plan-grid-empty">
        <p>Carregando planos...</p>
      </div>
    {:else if plans.length === 0}
      <div class="plan-grid-empty">
        <Icon name="image" size={48} />
        <p>Nenhum plano salvo</p>
        <p class="meta">Crie um plano e salve para aparecer aqui.</p>
      </div>
    {:else}
      <div class="plan-grid-list">
        {#each plans as plan (plan.id)}
          <div class="plan-card">
            <div class="plan-card-info">
              <span class="plan-card-name">{plan.name}</span>
              <span class="plan-card-meta">{plan.regionCount} regiões · {formatDate(plan.updatedAt)}</span>
            </div>
            <div class="plan-card-actions">
              <button class="btn-sm" onclick={() => onSelect(plan.id)}>
                <Icon name="upload" size={14} /> Abrir
              </button>
              <button class="btn-sm danger" onclick={() => onDelete(plan.id)}>
                <Icon name="trash" size={14} />
              </button>
            </div>
          </div>
        {/each}
      </div>
    {/if}
  </div>
</div>

<style>
  .plan-grid-overlay {
    position: fixed;
    inset: 0;
    background: rgba(0,0,0,0.3);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 50;
    animation: fade-in 0.2s ease;
  }

  @keyframes fade-in {
    from { opacity: 0; }
    to { opacity: 1; }
  }

  .plan-grid-panel {
    background: var(--papel);
    border: 1px solid var(--hairline);
    border-radius: var(--radius-surface);
    width: 90%;
    max-width: 560px;
    max-height: 80vh;
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  .plan-grid-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 12px 16px;
    border-bottom: 1px solid var(--hairline);
  }

  .plan-grid-header h3 {
    margin: 0;
    font-size: 15px;
    font-weight: 600;
  }

  .plan-grid-empty {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 48px 24px;
    gap: 8px;
    color: var(--grafite);
    opacity: 0.5;
  }

  .plan-grid-empty p { margin: 0; }
  .plan-grid-empty .meta { font-size: 12px; }

  .plan-grid-list {
    padding: 8px;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .plan-card {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 10px 12px;
    border: 1px solid var(--hairline);
    border-radius: var(--radius-surface);
    transition: border-color 0.15s;
  }

  .plan-card:hover { border-color: var(--grafite); }

  .plan-card-info {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .plan-card-name {
    font-size: 14px;
    font-weight: 600;
  }

  .plan-card-meta {
    font-size: 12px;
    color: var(--grafite);
    opacity: 0.6;
  }

  .plan-card-actions {
    display: flex;
    gap: 4px;
  }

  .btn-sm {
    display: inline-flex;
    align-items: center;
    gap: 3px;
    padding: 4px 10px;
    border: 1px solid var(--hairline);
    border-radius: var(--radius-surface);
    background: var(--papel);
    cursor: pointer;
    font-size: 12px;
    color: var(--grafite);
    white-space: nowrap;
  }

  .btn-sm:hover { background: var(--bancada); }
  .btn-sm.danger { color: var(--laca); }

  .icon-btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 32px;
    height: 32px;
    border: none;
    border-radius: var(--radius-surface);
    background: transparent;
    cursor: pointer;
    color: var(--grafite);
  }

  .icon-btn:hover { background: var(--bancada); }
</style>

<script lang="ts">
  import { onMount } from 'svelte';
  import Textfield from '@smui/textfield';
  import Icon from './Icon.svelte';
  import * as PaintService from '../../../bindings/paint-match-ai/paintservice';

  interface Manufacturer {
    id: number;
    name: string;
    country: string;
    website: string;
    logoPath: string;
    paintCount: number;
  }

  let manufacturers: Manufacturer[] = $state([]);
  let filtered: Manufacturer[] = $state([]);
  let loading = $state(true);
  let searchQuery = $state('');

  onMount(async () => {
    try {
      manufacturers = (await PaintService.GetManufacturers()) || [];
      filtered = manufacturers;
    } catch (e) {
      console.error('Erro carregando fabricantes:', e);
    } finally {
      loading = false;
    }
  });

  $effect(() => {
    if (!searchQuery) {
      filtered = manufacturers;
      return;
    }
    const q = searchQuery.toLowerCase();
    filtered = manufacturers.filter(m =>
      m.name.toLowerCase().includes(q) ||
      m.country.toLowerCase().includes(q)
    );
  });
</script>

<div class="page-container">
  <!-- Header -->
  <div class="page-header animate-rise">
    <h1 class="page-title">Fabricantes</h1>
    <p class="page-subtitle">{filtered.length} de {manufacturers.length} fabricantes cadastrados</p>
    <div class="page-divider"></div>
  </div>

  <!-- Search -->
  <div class="panel p-3 mb-6 animate-rise" style="animation-delay: 80ms; position: relative; z-index: 1;">
    <Textfield
      variant="outlined"
      bind:value={searchQuery}
      label="Buscar por nome ou país..."
      style="width: 100%;"
    >
      {#snippet leadingIcon()}
        <span class="mdc-text-field__icon mdc-text-field__icon--leading" style="color: var(--ink-500); display: flex;"><Icon name="search" size={17} /></span>
      {/snippet}
    </Textfield>
  </div>

  <!-- Grid -->
  {#if loading}
    <div class="grid-4">
      {#each Array(8) as _}
        <div class="panel p-4" style="display: flex; flex-direction: column; gap: 10px;">
          <div class="skeleton" style="width: 48px; height: 48px; border-radius: 50%;"></div>
          <div class="skeleton" style="height: 12px; width: 70%;"></div>
          <div class="skeleton" style="height: 10px; width: 40%;"></div>
        </div>
      {/each}
    </div>
  {:else}
    <div class="grid-4">
      {#each filtered as mfr, i (mfr.id)}
        <div class="panel mfr-card animate-rise" style="animation-delay: {Math.min(i * 25, 200)}ms;">
          <div class="mfr-head">
            <div class="mfr-logo">
              {#if mfr.logoPath}
                <img
                  src="file://{mfr.logoPath}"
                  alt={mfr.name}
                  loading="lazy"
                  onerror={(e) => (e.currentTarget as HTMLImageElement).style.display = 'none'}
                />
              {:else}
                <Icon name="building" size={22} />
              {/if}
            </div>
            <div class="mfr-id">
              <div class="mfr-name font-display">{mfr.name}</div>
              {#if mfr.country}
                <div class="mfr-country">{mfr.country}</div>
              {/if}
            </div>
          </div>
          <div class="mfr-footer">
            <span class="mfr-count font-mono">{mfr.paintCount} {mfr.paintCount === 1 ? 'tinta' : 'tintas'}</span>
            {#if mfr.website}
              <a class="mfr-link" href={mfr.website} target="_blank" rel="noopener noreferrer">site</a>
            {/if}
          </div>
        </div>
      {/each}
    </div>

    {#if filtered.length === 0}
      <div style="text-align: center; padding: 80px 0;">
        <div style="color: var(--ink-500); display: flex; justify-content: center; margin-bottom: 16px;"><Icon name="search-off" size={40} /></div>
        <p class="font-medium" style="color: var(--ink-500);">Nenhum fabricante encontrado</p>
      </div>
    {/if}
  {/if}
</div>

<style>
  /* Card composto: identidade (logo + nome/país) em cima, etiqueta embaixo */
  .mfr-card {
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  .mfr-head {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 16px;
    flex: 1;
  }

  .mfr-logo {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 46px;
    height: 46px;
    border-radius: var(--radius-control);
    background: var(--ink-800);
    color: var(--ink-500);
    overflow: hidden;
    flex-shrink: 0;
  }

  .mfr-logo img {
    width: 100%;
    height: 100%;
    object-fit: contain;
    padding: 6px;
  }

  .mfr-id {
    min-width: 0;
  }

  .mfr-name {
    font-weight: 640;
    font-size: 14.5px;
    color: var(--ink-100);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .mfr-country {
    font-size: 11.5px;
    color: var(--ink-500);
    margin-top: 1px;
  }

  .mfr-footer {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 9px 16px;
    border-top: 1px solid var(--ink-700);
    background: var(--ink-850);
  }

  /* Contagem como etiqueta mono, tipo linha de rótulo */
  .mfr-count {
    font-size: 10.5px;
    font-weight: 500;
    color: var(--ink-500);
  }

  .mfr-link {
    font-size: 11px;
    color: var(--lacquer-tint);
    text-decoration: none;
  }

  .mfr-link:hover {
    text-decoration: underline;
  }
</style>

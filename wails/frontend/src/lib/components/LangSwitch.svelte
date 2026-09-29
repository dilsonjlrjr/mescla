<script lang="ts">
  // Seletor de idioma do protótipo (US-18/NFR-09): pílulas de 44×40 dentro de
  // uma moldura de raio 8. Selecionado = fundo --color-accent-900 + texto
  // --color-accent-300; os demais em --color-neutral-500.
  import { i18n, LANGS, setLang } from '../i18n.svelte';
</script>

<div
  role="group"
  aria-label="Idioma"
  style="display: flex; gap: 3px; flex-shrink: 0; padding: 3px; border: 1px solid var(--color-neutral-800); border-radius: 8px;"
>
  {#each LANGS as l (l.code)}
    <button
      onclick={() => setLang(l.code)}
      aria-label={l.label}
      title={l.label}
      aria-pressed={i18n.lang === l.code}
      style="min-width: 44px; height: 40px; padding: 0 6px; border: none; border-radius: 6px; background: {i18n.lang ===
      l.code
        ? 'var(--color-accent-900)'
        : 'transparent'}; color: {i18n.lang === l.code
        ? 'var(--color-accent-300)'
        : 'var(--color-neutral-500)'}; font-family: inherit; font-size: 12.5px; font-weight: 500; letter-spacing: 0.04em; cursor: pointer;"
    >
      {l.code.toUpperCase()}
    </button>
  {/each}
</div>

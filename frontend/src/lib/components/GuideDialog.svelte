<script lang="ts">
  // Guia de uso — abre sozinho na primeira vez, e sempre pelo "?" da sidebar.
  import Dialog, { Content as DialogContent } from '@smui/dialog';
  import BrandMark from './BrandMark.svelte';

  interface Props {
    open: boolean;
    onClose: () => void;
  }

  let { open = $bindable(), onClose }: Props = $props();

  const steps = [
    {
      n: '1',
      title: 'Escolha a tinta que você conhece',
      desc: 'No Catálogo ou na busca da tela Equivalência: são 11 mil tintas de 35 marcas, com a cor real de cada frasco.',
    },
    {
      n: '2',
      title: 'Diga qual marca você tem em mãos',
      desc: 'A Mescla procura a melhor combinação usando só as tintas dessa marca — de 1 tinta pura até misturas de 3.',
    },
    {
      n: '3',
      title: 'Leia o resultado com honestidade',
      desc: 'O selo de proximidade diz se a cor fica idêntica, próxima ou impossível com esse catálogo — sem fingir precisão.',
    },
  ];
</script>

<Dialog bind:open onSMUIDialogClosed={onClose} surface$style="background: var(--ink-900); border: 1px solid var(--ink-700); border-radius: var(--radius-surface); max-width: 640px; width: 100%;">
  <DialogContent>
    <div style="padding: 12px 8px 8px;">
      <div class="flex items-center gap-3 mb-2">
        <BrandMark size={34} />
        <div>
          <div class="font-display font-bold" style="font-size: 22px; color: var(--paper); line-height: 1.1;">Mescla</div>
          <div style="font-size: 12px; color: var(--ink-500);">a cor certa, em qualquer marca</div>
        </div>
      </div>

      <p style="font-size: 13.5px; color: var(--ink-300); line-height: 1.6; margin: 14px 0 20px;">
        Você viu a cor perfeita numa tinta que não encontra por aqui — ou o pote acabou.
        A Mescla encontra a mesma cor nas marcas que você tem, medindo a diferença
        como o olho humano vê (ΔE2000). Tudo offline.
      </p>

      <div class="steps mb-5">
        {#each steps as s}
          <div class="step">
            <div class="step-n">{s.n}</div>
            <div class="step-title">{s.title}</div>
            <div class="step-desc">{s.desc}</div>
          </div>
        {/each}
      </div>

      <div class="panel p-4 mb-5" style="background: var(--ink-850);">
        <div class="font-mono" style="font-size: 10.5px; font-weight: 500; text-transform: uppercase; letter-spacing: 0.08em; color: var(--ink-500); margin-bottom: 8px;">Como ler o selo de proximidade</div>
        <div class="flex flex-wrap gap-2">
          <span class="delta-badge excellent" style="font-size: 11px;">Idêntica <span class="delta-value">ΔE &lt; 1</span></span>
          <span class="delta-badge excellent" style="font-size: 11px;">Muito próxima <span class="delta-value">ΔE &lt; 3</span></span>
          <span class="delta-badge good" style="font-size: 11px;">Próxima <span class="delta-value">ΔE &lt; 6</span></span>
          <span class="delta-badge fair" style="font-size: 11px;">Diferença visível <span class="delta-value">ΔE &lt; 12</span></span>
          <span class="delta-badge poor" style="font-size: 11px;">Cor diferente <span class="delta-value">ΔE ≥ 12</span></span>
        </div>
      </div>

      <button class="btn-primary" onclick={() => { open = false; onClose(); }}>
        Começar
      </button>
    </div>
  </DialogContent>
</Dialog>

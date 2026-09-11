<script module lang="ts">
  // rf-13 — funções puras extraídas para teste (vitest importa deste bloco de
  // módulo, sem montar o componente). Nada aqui depende de i18n/estado/DOM.

  /** RN10/CA10 — percentual × volume alvo, arredondado a 0,05 ml (uma gota). */
  export function pctToMl(percentage: number, targetMl: number): number {
    const raw = (percentage / 100) * targetMl;
    return Math.round(raw / 0.05) * 0.05;
  }

  /** CA19 — conta fabricantes DISTINTOS por manufacturerId, nunca por nome de tinta
   *  (correção do bug em PerguntaView.svelte:376-379 da versão anterior). */
  export function countManufacturers(ingredients: { manufacturerId: number }[]): number {
    return new Set(ingredients.map(i => i.manufacturerId)).size;
  }

  /** CA2/RG-13 — título da fórmula cross-brand: fabricantes de `manufacturers`,
   *  já ordenados e sem repetição pelo servidor, unidos por " + ". */
  export function crossBrandTitle(manufacturers: string[]): string {
    return manufacturers.join(' + ');
  }

  /** CAN7 — só aplica o resultado se o token da requisição ainda for o mais recente. */
  export function isCurrentRequest(token: number, latest: number): boolean {
    return token === latest;
  }

  // ── Gotas: menor proporção inteira (RG-12) ──
  function gcd(a: number, b: number): number {
    return b === 0 ? a : gcd(b, a % b);
  }

  /** RN10/CA10 — reparte `total` entre os percentuais em passos de `passo`, de
   *  modo que as parcelas exibidas somem exatamente `total`. Arredondar cada
   *  parcela por conta própria não fecha: 1/3 de 10 ml três vezes dá 10,05, e
   *  três linhas de 33% somam 99. O resto vai para as maiores frações. */
  export function repartir(percentuais: number[], total: number, passo: number): number[] {
    const soma = percentuais.reduce((a, b) => a + b, 0) || 1;
    const passos = Math.round(total / passo);
    const exatos = percentuais.map(pct => ((pct / soma) * total) / passo);
    const base = exatos.map(Math.floor);
    let sobra = passos - base.reduce((a, b) => a + b, 0);
    const porFracao = exatos
      .map((v, idx) => ({ idx, frac: v - Math.floor(v) }))
      .sort((a, b) => b.frac - a.frac);
    for (let k = 0; sobra > 0 && porFracao.length > 0; k++, sobra--) {
      base[porFracao[k % porFracao.length].idx]++;
    }
    return base.map(v => v * passo);
  }

  /** RG-12 — a menor razão inteira que representa os percentuais da fórmula. */
  export function computeDropsFromIngredients(ingredients: { percentage: number }[]): number[] {
    const raw = ingredients.map(i => i.percentage);
    const ints = raw.map(Math.floor);
    let left = 100 - ints.reduce((a, b) => a + b, 0);
    const byFrac = raw.map((v, idx) => ({ idx, frac: v - Math.floor(v) })).sort((a, b) => b.frac - a.frac);
    for (let k = 0; left > 0 && byFrac.length; k++, left--) ints[byFrac[k % byFrac.length].idx]++;
    const g = ints.filter(v => v > 0).reduce((acc, v) => gcd(acc, v), 0) || 1;
    const reduzido = ints.map(v => Math.max(1, Math.round(v / g)));
    // O stepper opera de 1 a 40 gotas (RG-12). Truncar cada componente em 40
    // isoladamente REESCREVE a proporção: 57/43 (coprimos, g=1) viraria 40/40,
    // isto é 50/50, com o ΔE00 do servidor pertencendo a outra mistura. Quando
    // o maior passa de 40, a razão inteira é reescalada por igual.
    const maior = Math.max(...reduzido);
    if (maior <= 40) return reduzido;
    const fator = maior / 40;
    return reduzido.map(v => Math.max(1, Math.round(v / fator)));
  }
</script>

<script lang="ts">
  // T1 — Pergunta e resposta (rf-04, reescrita rf-13). Dois passos: a busca
  // resolve numa FICHA da tinta (nada calculado); só o toque em "Buscar
  // equivalência" — ou trocar o universo — dispara o motor (M1). O universo é
  // um seletor único de três opções (M2), substituindo onlyHave + brandFilter +
  // pílulas por fabricante da versão anterior. Cross-brand é real agora: o
  // universo "Misturar marcas" chama o motor UMA vez, sem fabricante, e a
  // fórmula pode combinar potes de marcas diferentes (RG-13) — antes o código
  // rodava N chamadas (uma por fabricante) e ficava com a menor, uma
  // aproximação que este rf substitui.
  //
  // Motor de cor: esta tela não recalcula ΔE00 no cliente (fora de escopo,
  // rf-05). O stepper de gotas (RG-12) ajusta a PROPORÇÃO local sem re-chamar
  // o solver; o ΔE00 mostrado é sempre o último valor do servidor pra fórmula
  // base. Em ml, a quantidade por ingrediente é percentual × volume alvo,
  // arredondada a 0,05 ml (RN10) — ver pctToMl acima.
  import PaintBottle from '../components/PaintBottle.svelte';
  import Spinner from '../components/Spinner.svelte';
  import BrandMark from '../components/BrandMark.svelte';
  import LangSwitch from '../components/LangSwitch.svelte';
  import { allManufacturers, allPaints, paintById, searchPaints, hexOf, type Paint } from '../services/catalog';
  import { catalogRev } from '../services/catalogRev.svelte';
  import {
    findSimilar, suggestEquivalentRecipe, suggestRecipeForColor, recipeForColorInBrand, suggestFromStock,
    bestBrandsFor, ehUniversoVazio,
    type EquivalentRecipe, type BrandBest, type SearchResult,
  } from '../services/engine';
  import { shelf, inShelf, toggleShelf } from '../services/shelf.svelte';
  import { stock } from '../services/stock.svelte';
  import { saveRecipe } from '../services/recipes.svelte';
  import { switchTab, pushLayer } from '../nav.svelte';
  import { appState } from '../appState.svelte';
  import { recents, rememberPaint, rememberMescla } from '../recents.svelte';
  import { toast } from '../toast.svelte';
  import { hexToRgb } from '../color/theory';
  import { verdictKeys, deltaIsGood } from '../ui';
  import { t, decimal } from '../i18n.svelte';

  // ── Busca ──
  let query = $state('');
  let suggestOpen = $state(false);
  let inputEl: HTMLInputElement | undefined = $state();

  type Target = Paint | { r: number; g: number; b: number; hex: string };

  let sourcePaint: Paint | null = $state(null);
  let freeTarget: { r: number; g: number; b: number; hex: string } | null = $state(null);
  let target: Target | null = $derived(sourcePaint ?? freeTarget);

  let codeMode = $derived.by(() => {
    const q = query.trim();
    if (!q) return false;
    return /^[a-zA-Z0-9.-]+$/.test(q) && /\d/.test(q);
  });

  // ── Estados (M1) ──
  type Stage = 'vazio' | 'ficha' | 'calculando' | 'resposta' | 'nao-achei' | 'pool-vazio';
  let stage: Stage = $state('vazio');

  /** Seletor único de universo (M2) — um valor por vez (RG-07). */
  interface Universo {
    tipo: 'mix' | 'marca' | 'estoque';
    manufacturerId?: number;
  }
  let universo: Universo = $state({ tipo: 'mix' });

  let unit: 'drops' | 'ml' = $state('drops');
  let targetVolumeMl = $state(10);
  let forceMix = $state(false);

  // ── Resultado ──
  let readyPot: SearchResult | null = $state(null);
  let formula: EquivalentRecipe | null = $state(null);
  let otherBrands: BrandBest[] = $state([]);
  let manualDrops: number[] | null = $state(null);
  let poolVazioBody = $state('');

  // CAN7: token de sequência — resposta de requisição obsoleta é descartada.
  let requestSeq = 0;
  // CA17: camada de histórico da resposta — o back do Android fecha a
  // resposta antes de sair do app.
  let closeRespostaLayer: (() => void) | null = null;

  $effect(() => {
    if (appState.pendingTargetPaint) {
      const p = appState.pendingTargetPaint;
      appState.pendingTargetPaint = null;
      pickPaint(p);
    }
  });

  let suggestions = $derived.by(() => {
    const q = query.trim();
    if (!q || sourcePaint || freeTarget) return [];
    return searchPaints(q, { limit: 8 });
  });

  let sampleGrid = $derived.by(() => {
    const all = allPaints();
    if (all.length <= 24) return all;
    const step = Math.max(1, Math.floor(all.length / 24));
    const out: Paint[] = [];
    for (let i = 0; i < all.length && out.length < 24; i += step) out.push(all[i]);
    return out;
  });

  function looksLikeHex(q: string): string | null {
    const m = /^#?([0-9a-fA-F]{6})$/.exec(q.trim());
    return m ? `#${m[1].toUpperCase()}` : null;
  }

  function onQueryInput() {
    suggestOpen = query.trim().length > 0;
  }

  function closeSuggest() {
    suggestOpen = false;
  }

  function submitQuery() {
    const q = query.trim();
    if (!q) return;
    const exact = searchPaints(q, { limit: 1 })[0];
    const hex = looksLikeHex(q);
    if (exact && (!hex || exact.code.toLowerCase() === q.toLowerCase())) {
      pickPaint(exact);
      return;
    }
    if (hex) {
      const rgb = hexToRgb(hex)!;
      pickHex(rgb.r, rgb.g, rgb.b, hex);
      return;
    }
    if (exact) {
      pickPaint(exact);
      return;
    }
    closeRespostaLayer = null;
    sourcePaint = null;
    freeTarget = null;
    readyPot = null;
    formula = null;
    otherBrands = [];
    stage = 'nao-achei';
  }

  // A logo é o caminho de volta à página principal. Em T1 já estamos nela, então
  // "voltar" é limpar: descarta alvo, resposta e busca, e volta ao estado
  // inicial com a grade de amostras. Fecha a camada da resposta antes, senão o
  // back do Android encontraria uma camada órfã (CA17).
  function irParaInicio() {
    closeRespostaLayer?.();
    closeRespostaLayer = null;
    suggestOpen = false;
    query = '';
    sourcePaint = null;
    freeTarget = null;
    forceMix = false;
    manualDrops = null;
    readyPot = null;
    formula = null;
    otherBrands = [];
    outrasMarcasDe = null;
    universo = { tipo: 'mix' };
    stage = 'vazio';
  }

  function resetForNewTarget() {
    suggestOpen = false;
    query = '';
    forceMix = false;
    manualDrops = null;
    readyPot = null;
    formula = null;
    otherBrands = [];
    outrasMarcasDe = null;
    universo = { tipo: 'mix' };
    closeRespostaLayer = null;
    stage = 'ficha';
  }

  function pickPaint(p: Paint) {
    sourcePaint = p;
    freeTarget = null;
    resetForNewTarget();
    rememberPaint(p.id);
  }

  function pickHex(r: number, g: number, b: number, hex: string) {
    sourcePaint = null;
    freeTarget = { r, g, b, hex };
    resetForNewTarget();
  }

  function pickSample(p: Paint) {
    pickPaint(p);
  }

  function shootPiece() {
    switchTab('plano');
  }

  // "Fotografar o pote" (ler etiqueta por foto) não tem serviço de OCR ainda
  // (fora de escopo, US-19/US-20 Could) — leva o foco pro campo de busca como
  // retorno inofensivo. Só o botão do CABEÇALHO saiu nesta rodada (M3); a
  // saída de "não achei essa tinta" continua oferecendo o mesmo atalho.
  function shootPot() {
    inputEl?.focus();
  }

  // ── Camada de histórico da resposta (CA17) ──
  function abrirResposta() {
    stage = 'resposta';
    if (!closeRespostaLayer) {
      closeRespostaLayer = pushLayer(() => {
        closeRespostaLayer = null;
        if (stage === 'resposta') stage = 'ficha';
      });
    }
  }

  function voltarParaFicha() {
    closeRespostaLayer?.();
  }

  // ── Cálculo ──
  async function bestOverManufacturers<T extends { deltaE: number }>(
    ids: number[],
    call: (mfrId: number) => Promise<T>,
  ): Promise<T | null> {
    const settled = await Promise.allSettled(ids.map(call));
    const ok: T[] = [];
    for (const s of settled) {
      if (s.status === 'fulfilled') ok.push(s.value);
    }
    if (ok.length === 0) return null;
    ok.sort((a, b) => a.deltaE - b.deltaE);
    return ok[0];
  }

  // A lista "a mesma cor em outra marca" depende só da tinta de origem, não do
  // universo — buscar de novo a cada troca de universo é chamada à toa. E ela
  // NÃO pode derrubar a resposta já calculada: se a rota falhar, a fórmula
  // continua na tela, só sem a lista.
  let outrasMarcasDe: number | null = $state(null);
  async function carregarOutrasMarcas(seq: number) {
    const sp = sourcePaint;
    if (!sp) return;
    if (outrasMarcasDe === sp.id && otherBrands.length > 0) return;
    try {
      const brands = await bestBrandsFor(sp.id);
      if (!isCurrentRequest(seq, requestSeq)) return;
      otherBrands = brands.sort((a, b) => a.deltaE - b.deltaE);
      outrasMarcasDe = sp.id;
    } catch (e) {
      console.error('Não consegui listar as outras marcas:', e);
    }
  }

  async function compute() {
    const tg = target;
    if (!tg) return;
    const seq = ++requestSeq;
    stage = 'calculando';
    readyPot = null;
    formula = null;
    manualDrops = null;
    if (outrasMarcasDe !== (sourcePaint?.id ?? null)) otherBrands = [];

    try {
      // RG-10 item 1 — pote pronto vence, a menos que o pintor peça a mistura.
      // Só no universo "misturar marcas": `findSimilar` varre o catálogo
      // inteiro e ignora o universo escolhido, então em "uma marca" ou "meu
      // estoque" ele respondia com pote de OUTRA marca e a mistura pedida
      // nunca aparecia (D-012). Nesses universos quem decide pote x mistura é
      // a própria rota da receita: ela devolve `method: "single"` quando uma
      // tinta só resolve, dentro do universo.
      if (!forceMix && universo.tipo === 'mix') {
        // Pede 2: o primeiro resultado da busca por cor é a PRÓPRIA tinta de
        // origem (ΔE 0), e responder "use a tinta que você já escolheu" não é
        // equivalência (D-011). Alvo livre (hex) não tem origem a excluir.
        const similar = await findSimilar(tg.r, tg.g, tg.b, 30, 2);
        if (!isCurrentRequest(seq, requestSeq)) return;
        const best = similar.find(s => s.paintId !== sourcePaint?.id);
        if (best && best.deltaE < 2.5) {
          readyPot = best;
          abrirResposta();
          // CA14 vale para o pote pronto também: é justamente aí que o pintor
          // quer ver a mesma cor nas outras marcas.
          void carregarOutrasMarcas(seq);
          return;
        }
      }

      if (allManufacturers().length === 0) {
        poolVazioBody = t('noneRegHere');
        stage = 'pool-vazio';
        return;
      }

      let result: EquivalentRecipe | null = null;
      let motivoServidor = '';

      if (universo.tipo === 'estoque') {
        if (shelf.manufacturerIds.length === 0 && stock.paints.length === 0) {
          poolVazioBody = t('noneBodyStock');
          stage = 'pool-vazio';
          return;
        }
        if (sourcePaint && stock.paints.length > 0) {
          try {
            result = await suggestFromStock(sourcePaint.id, stock.paints);
          } catch {
            /* estoque não monta receita — segue pras marcas da estante */
          }
        }
        // id 0 nunca é fabricante: passá-lo adiante faria a chamada omitir
        // targetManufacturerId e o servidor responderia com o catálogo inteiro,
        // sugerindo tinta que o pintor não tem.
        const marcasDaEstante = shelf.manufacturerIds.filter(id => id > 0);
        if (!result && marcasDaEstante.length > 0) {
          result = await bestOverManufacturers(marcasDaEstante, id =>
            sourcePaint
              ? suggestEquivalentRecipe(sourcePaint.id, id, 3)
              : recipeForColorInBrand(tg.r, tg.g, tg.b, id, 3),
          );
        }
      } else if (universo.tipo === 'marca' && universo.manufacturerId) {
        result = sourcePaint
          ? await suggestEquivalentRecipe(sourcePaint.id, universo.manufacturerId, 3)
          : await recipeForColorInBrand(tg.r, tg.g, tg.b, universo.manufacturerId, 3);
      } else if (sourcePaint) {
        // Misturar marcas (M2 padrão) — RG-13: uma chamada só, sem fabricante,
        // pool = catálogo inteiro (rf-13, substitui o loop por marca do rf-04).
        result = await suggestEquivalentRecipe(sourcePaint.id, undefined, 3);
      } else {
        const resp = await suggestRecipeForColor(tg.r, tg.g, tg.b, { maxIngredients: 3 });
        if (ehUniversoVazio(resp)) {
          motivoServidor = resp.motivo;
        } else {
          result = resp;
        }
      }

      if (!isCurrentRequest(seq, requestSeq)) return;

      if (result && ehUniversoVazio(result)) {
        motivoServidor = result.motivo;
        result = null;
      }

      if (!result) {
        poolVazioBody = motivoServidor || t('noneRegHere');
        stage = 'pool-vazio';
        return;
      }

      formula = result;
      abrirResposta();

      void carregarOutrasMarcas(seq);

      rememberMescla({
        sourceId: sourcePaint?.id ?? -1,
        brandIds: Array.from(new Set(result.ingredients.map(i => i.manufacturerId))),
        sourceName: sourcePaint?.name ?? freeTarget?.hex ?? '',
        targetManufacturer: result.crossBrand ? crossBrandTitle(result.manufacturers ?? []) : result.targetManufacturer,
        deltaE: result.deltaE,
        sourceRGB: [result.sourceR, result.sourceG, result.sourceB],
        resultRGB: [result.resultR, result.resultG, result.resultB],
      });
    } catch (e) {
      console.error('Erro ao calcular:', e);
      if (isCurrentRequest(seq, requestSeq)) {
        toast(t('noneRegHere'), 'error');
        stage = 'ficha';
      }
    } finally {
      if (isCurrentRequest(seq, requestSeq) && stage === 'calculando') stage = 'ficha';
    }
  }

  function buscarEquivalencia() {
    if (target && !buscaBloqueada) void compute();
  }

  // RN15: escolher o universo é preparação, não busca. Trocar de opção ou de
  // fornecedor no combo muda só o que a próxima busca vai usar — quem calcula é
  // o botão. Sem fornecedor escolhido não há o que buscar.
  let buscaBloqueada = $derived(universo.tipo === 'marca' && !universo.manufacturerId);

  function onUniversoChange(u: Universo) {
    universo = u;
  }

  function useMixAnyway() {
    forceMix = true;
    void compute();
  }

  // Exceção declarada da RN15: a linha de "a mesma cor em outra marca" é ação
  // direta sobre um resultado ("mostre naquela marca"), então vale como o
  // próprio toque no botão e recalcula na hora.
  function pickOtherBrand(b: BrandBest) {
    universo = { tipo: 'marca', manufacturerId: b.manufacturerId };
    if (target) void compute();
  }

  let baseDrops = $derived.by(() => (formula ? computeDropsFromIngredients(formula.ingredients) : []));
  let drops = $derived(manualDrops ?? baseDrops);
  let totalDrops = $derived(drops.reduce((a, b) => a + b, 0));

  // As parcelas exibidas saem de `repartir`, para % somar 100 e ml somar o
  // volume alvo — em ml o passo é 0,05 (uma gota), como manda RN10.
  let pctExibido = $derived.by(() => {
    const f = formula;
    if (!f) return [];
    return repartir(f.ingredients.map((_, i) => pctOf(i)), 100, 1);
  });
  let mlExibido = $derived.by(() => {
    const f = formula;
    if (!f) return [];
    return repartir(f.ingredients.map((_, i) => pctOf(i)), targetVolumeMl, 0.05);
  });

  function pctOf(i: number): number {
    // Sem ajuste manual, o percentual é o que o servidor calculou — as gotas
    // são a menor razão inteira que o representa, e arredondá-las de volta
    // devolveria um percentual ligeiramente diferente do da fórmula (RN10).
    if (!manualDrops) return formula?.ingredients[i]?.percentage ?? 0;
    return (drops[i] / (totalDrops || 1)) * 100;
  }

  function adjustDrop(i: number, delta: number) {
    if (!formula) return;
    const cur = manualDrops ? [...manualDrops] : [...baseDrops];
    cur[i] = Math.max(1, Math.min(40, cur[i] + delta));
    manualDrops = cur;
  }

  function resetDrops() {
    manualDrops = null;
  }

  // Prévia de cor da mistura sob as gotas atuais — média em luz linear
  // (RG-03), só pra amostra "mistura" reagir ao stepper; o ΔE00 exibido segue
  // sendo o do servidor pra fórmula base.
  function srgbToLinear(v: number): number {
    const c = v / 255;
    return c <= 0.04045 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4;
  }
  function linearToSrgb(c: number): number {
    const v = c <= 0.0031308 ? c * 12.92 : 1.055 * c ** (1 / 2.4) - 0.055;
    return Math.max(0, Math.min(255, Math.round(v * 255)));
  }
  let mixPreview = $derived.by(() => {
    if (!formula) return null;
    const total = drops.reduce((a, b) => a + b, 0) || 1;
    let rl = 0, gl = 0, bl = 0;
    formula.ingredients.forEach((ing, i) => {
      const w = drops[i] / total;
      rl += srgbToLinear(ing.r) * w;
      gl += srgbToLinear(ing.g) * w;
      bl += srgbToLinear(ing.b) * w;
    });
    return { r: linearToSrgb(rl), g: linearToSrgb(gl), b: linearToSrgb(bl) };
  });

  // ── Rótulo/manchete (RG-05/RG-13, CA2/CA19) ──
  let manufacturerCount = $derived.by(() => (formula ? countManufacturers(formula.ingredients) : 0));

  let headline = $derived.by(() => {
    if (readyPot) return t('hPote');
    const f = formula;
    if (!f) return '';
    if (f.ingredients.length <= 1) return t('hPote');
    if (f.crossBrand) return t('hMixBrands', { n: f.ingredients.length, b: manufacturerCount });
    return t('hMix', { n: f.ingredients.length });
  });

  let formulaTitle = $derived.by(() => {
    const f = formula;
    if (!f) return '';
    return f.crossBrand ? crossBrandTitle(f.manufacturers ?? []) : f.targetManufacturer;
  });

  let vc = $derived.by(() => {
    if (formula) return verdictKeys(formula.deltaE);
    if (readyPot) return verdictKeys(readyPot.deltaE);
    return null;
  });
  let deltaValue = $derived.by(() => (readyPot ? readyPot.deltaE : (formula?.deltaE ?? 0)));
  let deltaColor = $derived(deltaIsGood(deltaValue) ? 'var(--color-accent-2)' : 'var(--color-text)');

  let mixSwatch = $derived.by(() => {
    if (readyPot) return { r: readyPot.r, g: readyPot.g, b: readyPot.b };
    if (mixPreview) return mixPreview;
    if (formula) return { r: formula.resultR, g: formula.resultG, b: formula.resultB };
    return null;
  });

  // ── Posse (services/shelf, por fabricante — RN12) ──
  let fichaHave = $derived.by(() => (sourcePaint ? inShelf(sourcePaint.manufacturerId) : false));
  function toggleFichaHave() {
    if (sourcePaint) toggleShelf(sourcePaint.manufacturerId);
  }
  function ingredientHave(mfrId: number): boolean {
    return inShelf(mfrId);
  }
  function toggleIngredientHave(mfrId: number) {
    // Ingrediente sem fabricante resolvido (id 0) não tem o que alternar —
    // gravar 0 na estante corromperia o universo "só o que eu tenho", que
    // trata cada id da estante como fabricante real.
    if (!mfrId) return;
    toggleShelf(mfrId);
  }

  // Fabricantes mudam em T4 com esta tela montada (rf-14): a lista se refaz
  // a cada recarga do catálogo.
  let mfrs = $derived.by(() => {
    void catalogRev.n;
    return allManufacturers();
  });

  let universoBrandName = $derived(
    universo.tipo === 'marca' ? (mfrs.find(m => m.id === universo.manufacturerId)?.name ?? '') : '',
  );

  let poolVazioTitle = $derived.by(() => {
    if (universo.tipo === 'estoque') return t('noneTitleStock');
    if (universo.tipo === 'marca') return t('noneTitleBrand', { brand: universoBrandName });
    return t('hNoneCross');
  });

  let canSaveRecipe = $derived.by(() => !!readyPot || (!!formula && !formula.crossBrand));

  function rerun(m: (typeof recents.mesclas)[number]) {
    if (m.sourceId >= 0) {
      const p = paintById(m.sourceId);
      if (p) {
        pickPaint(p);
        return;
      }
    }
    pickHex(m.sourceRGB[0], m.sourceRGB[1], m.sourceRGB[2], hexOf({ r: m.sourceRGB[0], g: m.sourceRGB[1], b: m.sourceRGB[2] }));
  }

  function doSaveRecipe() {
    if (!target || !canSaveRecipe) return;
    const mfrId = readyPot ? paintById(readyPot.paintId)?.manufacturerId : formula?.ingredients[0]?.manufacturerId;
    if (mfrId == null) return;
    saveRecipe({
      name: sourcePaint?.name ?? freeTarget?.hex ?? '',
      targetPaintId: sourcePaint?.id ?? null,
      targetR: target.r,
      targetG: target.g,
      targetB: target.b,
      manufacturerId: mfrId,
    });
    toast(t('recipeSavedToast'));
    switchTab('receitas');
  }
</script>

<div class="t1">
  <!-- Cabeçalho: marca, busca e idioma (M3) — o botão "fotografar o pote" saiu daqui. -->
  <div
    style="display: flex; align-items: center; flex-wrap: wrap; row-gap: 10px; column-gap: 18px; min-height: 76px; flex-shrink: 0; padding: 12px 20px; border-bottom: 1px solid var(--color-line);"
  >
    <button
      class="pressable"
      onclick={irParaInicio}
      aria-label={t('homeAria')}
      style="display: flex; align-items: center; gap: 10px; flex-shrink: 0; min-height: 44px; padding: 0 6px; border: none; background: transparent; color: inherit; font-family: inherit; cursor: pointer;"
    >
      <BrandMark size={30} />
      <span style="font-size: clamp(16px, 1.7cqi, 19px); font-weight: 500; letter-spacing: -0.02em;">Mescla</span>
    </button>

    <div style="position: relative; flex: 1 1 420px; min-width: 0; order: 3;">
      <div
        style="display: flex; align-items: center; gap: 12px; height: 54px; padding: 0 14px 0 16px; border: 1px solid var(--color-field-border); border-radius: 14px; background: var(--color-field); overflow: hidden;"
      >
        <i class="ph ph-magnifying-glass" style="font-size: 22px; color: var(--color-neutral-500); flex-shrink: 0;"></i>
        <input
          bind:this={inputEl}
          bind:value={query}
          oninput={onQueryInput}
          onkeydown={e => e.key === 'Enter' && submitQuery()}
          onfocus={() => (suggestOpen = query.trim().length > 0)}
          type="text"
          placeholder={t('phSearch')}
          aria-label={t('phSearch')}
          autocomplete="off"
          autocorrect="off"
          autocapitalize="off"
          spellcheck="false"
          enterkeyhint="search"
          style="flex: 1 1 auto; min-width: 60px; height: 50px; background: transparent; border: none; outline: none; color: var(--color-text); font-family: inherit; font-size: clamp(16px, 1.7cqi, 19px); font-weight: 500; letter-spacing: -0.01em;"
        />
        {#if codeMode}
          <span
            style="display: inline-flex; align-items: center; gap: 6px; flex-shrink: 0; height: 32px; padding: 0 10px; border-radius: 8px; background: var(--color-accent-900); color: var(--color-accent-400); font-size: 12px; font-weight: 500; letter-spacing: 0.04em; text-transform: uppercase; white-space: nowrap;"
          >
            <i class="ph-bold ph-keyboard" style="font-size: 14px;"></i>{t('kbdCode')}
          </span>
        {/if}
        <button
          class="pressable t1h-acc"
          onclick={shootPiece}
          aria-label={t('shootPiece')}
          title={t('shootPiece')}
          style="display: inline-flex; align-items: center; justify-content: center; flex-shrink: 0; width: 46px; height: 46px; border: 1px solid var(--color-accent-700); border-radius: 8px; background: transparent; color: var(--color-accent-400); font-family: inherit; cursor: pointer;"
        >
          <i class="ph ph-camera" style="font-size: 19px;"></i>
        </button>
      </div>

      {#if suggestOpen && suggestions.length > 0}
        <div
          class="t1-suggest-pop"
          role="listbox"
          aria-label={t('suggestTitle')}
          style="position: absolute; top: 62px; left: 0; right: 0; z-index: 40; padding: 16px; border: 1px solid var(--color-neutral-800); border-radius: 14px; background: var(--color-raised); box-shadow: 0 18px 48px rgba(0, 0, 0, 0.55);"
        >
          <div style="display: flex; align-items: center; justify-content: space-between; margin-bottom: 12px;">
            <span class="section-label">{t('suggestTitle')}</span>
            <button
              class="pressable"
              onclick={closeSuggest}
              style="height: 44px; padding: 0 12px; border: none; background: transparent; color: var(--color-neutral-500); font-family: inherit; font-size: 14px; cursor: pointer;"
              >{t('close')}</button
            >
          </div>
          <div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(210px, 1fr)); gap: 12px;">
            {#each suggestions as p (p.id)}
              <button
                class="pressable t1h-border"
                onclick={() => pickPaint(p)}
                role="option"
                aria-selected="false"
                style="display: flex; align-items: center; gap: 14px; padding: 14px; min-height: 96px; border: 1px solid var(--color-neutral-800); border-radius: 14px; background: var(--color-surface); text-align: left; cursor: pointer; font-family: inherit;"
              >
                <PaintBottle r={p.r} g={p.g} b={p.b} width={34} height={57} label={p.name} />
                <span style="display: flex; flex-direction: column; gap: 3px; min-width: 0;">
                  <span
                    style="font-size: 15px; font-weight: 500; color: var(--color-text); white-space: nowrap; overflow: hidden; text-overflow: ellipsis;"
                    >{p.name}</span
                  >
                  <span class="font-mono" style="font-size: 12px; color: var(--color-neutral-500);"
                    >{p.code} · {p.manufacturer}</span
                  >
                </span>
              </button>
            {/each}
          </div>
        </div>
      {/if}
    </div>

    <div style="display: flex; align-items: center; gap: 10px; flex-shrink: 0;">
      <LangSwitch />
    </div>
  </div>

  <!-- Miolo em 2 colunas (38% / 1fr) — pool-vazio e não-achei saem da coluna
       de resposta (M em tela cheia), o resto usa a grade normal. -->
  <div style="flex: 1; min-height: 0; display: grid; grid-template-columns: minmax(0, 38%) minmax(0, 1fr);">
    {#if stage === 'pool-vazio'}
      <div style="grid-column: 1 / -1; display: flex; flex-direction: column; align-items: flex-start; gap: 14px; padding: 40px 22px; overflow-y: auto;">
        <i class="ph ph-drop-half" style="font-size: 40px; color: var(--color-neutral-700);"></i>
        <p style="margin: 0; font-size: 24px; font-weight: 500; color: var(--color-text);">{poolVazioTitle}</p>
        <p style="margin: 0; font-size: 16px; color: var(--color-neutral-400); max-width: 420px; text-wrap: pretty;">{poolVazioBody}</p>
        <div style="display: flex; gap: 10px; margin-top: 8px;">
          <button
            class="pressable t1h-acc"
            onclick={() => switchTab('estante')}
            style="display: inline-flex; align-items: center; gap: 10px; height: 56px; padding: 0 20px; border: 1px solid var(--color-accent); border-radius: 8px; background: transparent; color: var(--color-accent-400); font-family: inherit; font-size: 16px; font-weight: 500; cursor: pointer;"
          >
            <i class="ph ph-plus" style="font-size: 18px;"></i>{t('addPaint')}
          </button>
          {#if universo.tipo !== 'mix'}
            <button
              class="pressable t1h-ghost"
              onclick={() => onUniversoChange({ tipo: 'mix' })}
              style="height: 56px; padding: 0 20px; border: 1px solid var(--color-neutral-800); border-radius: 8px; background: transparent; color: var(--color-neutral-400); font-family: inherit; font-size: 16px; font-weight: 500; cursor: pointer;"
              >{t('useAllBrands')}</button
            >
          {/if}
        </div>
      </div>
    {:else if stage === 'nao-achei'}
      <div style="grid-column: 1 / -1; display: flex; flex-direction: column; align-items: flex-start; gap: 14px; padding: 40px 22px; overflow-y: auto;">
        <i class="ph ph-magnifying-glass" style="font-size: 40px; color: var(--color-neutral-700);"></i>
        <p style="margin: 0; font-size: 24px; font-weight: 500; color: var(--color-text);">{t('noResultH')}</p>
        <p style="margin: 0; font-size: 16px; color: var(--color-neutral-400); max-width: 420px;">{t('noResultBody')}</p>
        <button
          class="pressable t1h-acc"
          onclick={shootPot}
          style="display: inline-flex; align-items: center; gap: 10px; height: 56px; margin-top: 8px; padding: 0 20px; border: 1px solid var(--color-accent-700); border-radius: 8px; background: transparent; color: var(--color-accent-400); font-family: inherit; font-size: 16px; font-weight: 500; cursor: pointer;"
        >
          <i class="ph ph-camera" style="font-size: 20px;"></i>{t('shootPot')}
        </button>
      </div>
    {:else if stage === 'vazio'}
      <div
        style="padding: 22px 22px 20px 20px; border-right: 1px solid var(--color-line); display: flex; flex-direction: column; gap: 20px; min-height: 0; overflow-y: auto;"
      >
        <div style="display: flex; flex-direction: column; gap: 14px;">
          <h1 class="font-display" style="margin: 0; font-size: clamp(22px, 2.9cqi, 36px); font-weight: 500; letter-spacing: -0.02em; line-height: 1.14; color: var(--color-text);">
            {t('emptyH')}
          </h1>
          <p style="margin: 0; font-size: 18px; line-height: 1.5; color: var(--color-neutral-400);">{t('emptyP')}</p>
          <button
            class="pressable t1h-acc"
            onclick={shootPiece}
            style="display: inline-flex; align-items: center; justify-content: center; gap: 10px; height: 72px; margin-top: 6px; border: 1px solid var(--color-accent-700); border-radius: 14px; background: transparent; color: var(--color-accent-400); font-family: inherit; font-size: 18px; font-weight: 500; cursor: pointer;"
          >
            <i class="ph ph-camera" style="font-size: 26px;"></i>{t('shootPiece')}
          </button>
        </div>
      </div>
      <div style="min-height: 0; overflow-y: auto; padding: 22px 20px 24px 22px;">
        <p class="section-label" style="margin: 0 0 14px;">{t('pickFinger')}</p>
        <div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(64px, 1fr)); gap: 8px;">
          {#each sampleGrid as p (p.id)}
            <button
              class="pressable t1h-swatch"
              onclick={() => pickSample(p)}
              title="{p.name} · {p.manufacturer}"
              style="height: 60px; border: 1px solid var(--color-rule); border-radius: 8px; background: rgb({p.r}, {p.g}, {p.b}); cursor: pointer;"
            ></button>
          {/each}
        </div>
        <p style="margin: 24px 0 0; font-size: 15px; color: var(--color-neutral-500);">{t('pickFingerNote')}</p>
      </div>
    {:else}
      <!-- ficha / calculando / resposta — a ficha fica sempre à esquerda. -->
      <div
        style="padding: 22px 22px 20px 20px; border-right: 1px solid var(--color-line); display: flex; flex-direction: column; gap: 20px; min-height: 0; overflow-y: auto;"
      >
        <div>
          <p class="section-label" style="margin: 0 0 10px;">{t('sheetTitle')}</p>
          <div style="display: flex; align-items: center; gap: 16px;">
            <PaintBottle r={target?.r ?? 0} g={target?.g ?? 0} b={target?.b ?? 0} width={54} height={90} label={sourcePaint?.name ?? freeTarget?.hex ?? ''} />
            <div style="display: flex; flex-direction: column; gap: 4px; min-width: 0;">
              <span class="font-display" style="font-size: clamp(19px, 2.3cqi, 28px); font-weight: 500; letter-spacing: -0.02em; color: var(--color-text); white-space: nowrap; overflow: hidden; text-overflow: ellipsis;">
                {sourcePaint ? sourcePaint.name : freeTarget?.hex}
              </span>
              {#if sourcePaint}
                <span class="font-mono" style="font-size: 14px; color: var(--color-neutral-500);">{sourcePaint.code} · {sourcePaint.manufacturer}</span>
              {/if}
              <span class="font-mono" style="font-size: 13px; color: var(--color-neutral-500);">
                {hexOf({ r: target?.r ?? 0, g: target?.g ?? 0, b: target?.b ?? 0 }).toUpperCase()} · {target?.r}, {target?.g}, {target?.b}
              </span>
            </div>
          </div>
          {#if sourcePaint}
            <button
              class="pressable"
              onclick={toggleFichaHave}
              aria-label={t('ariaHave')}
              style="display: inline-flex; align-items: center; gap: 8px; margin-top: 12px; height: 44px; padding: 0 12px 0 4px; border: none; background: transparent; color: {fichaHave ? 'var(--color-accent-400)' : 'var(--color-neutral-400)'}; font-family: inherit; font-size: 14px; font-weight: 500; cursor: pointer;"
            >
              <span
                style="display: inline-flex; align-items: center; justify-content: center; width: 24px; height: 24px; border: 1px solid {fichaHave ? 'var(--color-accent)' : 'var(--color-neutral-700)'}; border-radius: 4px; background: {fichaHave ? 'var(--color-accent)' : 'transparent'};"
              >
                <i class="ph-bold ph-check" style="font-size: 15px; color: {fichaHave ? 'var(--color-accent-100)' : 'transparent'};"></i>
              </span>
              {fichaHave ? t('have') : t('dontHave')}
            </button>
          {/if}
        </div>

        <div style="display: flex; flex-direction: column; gap: 10px;">
          <span class="section-label">{t('universeLabel')}</span>
          <div role="group" aria-label={t('universeLabel')} style="display: flex; flex-wrap: wrap; gap: 8px;">
            <button
              class="pressable t1h-border"
              aria-pressed={universo.tipo === 'mix'}
              disabled={stage === 'calculando'}
              onclick={() => onUniversoChange({ tipo: 'mix' })}
              style="min-height: 44px; height: 48px; padding: 0 16px; border: 1px solid {universo.tipo === 'mix' ? 'var(--color-accent)' : 'var(--color-neutral-800)'}; border-radius: 8px; background: {universo.tipo === 'mix' ? 'var(--color-accent)' : 'transparent'}; color: {universo.tipo === 'mix' ? 'var(--color-accent-100)' : 'var(--color-text)'}; font-family: inherit; font-size: 14px; font-weight: 500; cursor: pointer;"
              >{t('universoMix')}</button
            >
            <button
              class="pressable t1h-border"
              aria-pressed={universo.tipo === 'marca'}
              disabled={stage === 'calculando'}
              onclick={() => onUniversoChange({ tipo: 'marca', manufacturerId: universo.manufacturerId })}
              style="min-height: 44px; height: 48px; padding: 0 16px; border: 1px solid {universo.tipo === 'marca' ? 'var(--color-accent)' : 'var(--color-neutral-800)'}; border-radius: 8px; background: {universo.tipo === 'marca' ? 'var(--color-accent)' : 'transparent'}; color: {universo.tipo === 'marca' ? 'var(--color-accent-100)' : 'var(--color-text)'}; font-family: inherit; font-size: 14px; font-weight: 500; cursor: pointer;"
              >{t('universoBrand')}</button
            >
            <button
              class="pressable t1h-border"
              aria-pressed={universo.tipo === 'estoque'}
              disabled={stage === 'calculando'}
              onclick={() => onUniversoChange({ tipo: 'estoque' })}
              style="min-height: 44px; height: 48px; padding: 0 16px; border: 1px solid {universo.tipo === 'estoque' ? 'var(--color-accent)' : 'var(--color-neutral-800)'}; border-radius: 8px; background: {universo.tipo === 'estoque' ? 'var(--color-accent)' : 'transparent'}; color: {universo.tipo === 'estoque' ? 'var(--color-accent-100)' : 'var(--color-text)'}; font-family: inherit; font-size: 14px; font-weight: 500; cursor: pointer;"
              >{t('onlyStock')}</button
            >
          </div>
          {#if universo.tipo === 'marca'}
            <select
              aria-label={t('chooseBrandAria')}
              disabled={stage === 'calculando'}
              value={universo.manufacturerId ?? ''}
              onchange={e => onUniversoChange({ tipo: 'marca', manufacturerId: Number((e.currentTarget as HTMLSelectElement).value) || undefined })}
              style="height: 44px; padding: 0 12px; border: 1px solid var(--color-neutral-800); border-radius: 8px; background: var(--color-field); color: var(--color-text); font-family: inherit; font-size: 14px;"
            >
              <!-- RN1: nenhum fornecedor vem escolhido. O placeholder é a única
                   opção sem valor, e enquanto ele estiver ativo a busca fica
                   desabilitada (RN15). -->
              <option value="" disabled>{t('chooseSupplierPlaceholder')}</option>
              {#each mfrs as m (m.id)}
                <option value={m.id}>{m.name}</option>
              {/each}
            </select>
          {/if}
        </div>

        <button
          class="pressable t1h-acc"
          onclick={buscarEquivalencia}
          disabled={stage === 'calculando' || buscaBloqueada}
          style="display: inline-flex; align-items: center; justify-content: center; gap: 10px; height: 56px; border: 1px solid var(--color-accent); border-radius: 8px; background: var(--color-accent); color: var(--color-accent-100); font-family: inherit; font-size: 16px; font-weight: 500; cursor: {buscaBloqueada ? 'not-allowed' : 'pointer'}; opacity: {buscaBloqueada ? 0.5 : 1};"
        >
          {#if stage === 'calculando'}
            <Spinner size={18} label={t('calculating')} />
          {:else}
            <i class="ph ph-flask" style="font-size: 18px;"></i>
          {/if}
          {t('searchEquivalenceBtn')}
        </button>
      </div>

      <div style="min-height: 0; overflow-y: auto; padding: 22px 20px 24px 22px;">
        {#if stage === 'calculando'}
          <div style="display: flex; flex-direction: column; gap: 12px;">
            {#each Array(3) as _}
              <div style="display: flex; align-items: center; gap: 18px; height: 108px; padding: 0 18px; border: 1px solid var(--color-line); border-radius: 14px; background: var(--color-panel);">
                <div style="width: 44px; height: 74px; border-radius: 8px; background: var(--color-surface);"></div>
                <div style="flex: 1; display: flex; flex-direction: column; gap: 10px;">
                  <div style="height: 16px; width: 44%; border-radius: 4px; background: var(--color-surface);"></div>
                  <div style="height: 14px; width: 26%; border-radius: 4px; background: var(--color-surface);"></div>
                </div>
                <div style="width: 180px; height: 56px; border-radius: 8px; background: var(--color-surface);"></div>
              </div>
            {/each}
            <p style="display: flex; align-items: center; gap: 10px; margin: 0; font-size: 14px; color: var(--color-neutral-500);">
              <Spinner size={20} label={t('calculating')} />
              {t('calculating')}
            </p>
          </div>
        {:else if stage === 'resposta'}
          <div class="animate-rise" style="display: flex; flex-direction: column; gap: 18px; min-height: 0;">
            <button
              class="pressable t1h-link"
              onclick={voltarParaFicha}
              style="align-self: flex-start; display: inline-flex; align-items: center; gap: 8px; height: 44px; padding: 0 8px; border: none; background: transparent; color: var(--color-neutral-400); font-family: inherit; font-size: 14px; cursor: pointer;"
            >
              <i class="ph ph-arrow-left" style="font-size: 16px;"></i>{t('voltarBtn')}
            </button>

            {#if vc}
              <div>
                <p class="section-label" style="margin: 0 0 10px;">{t('answerKicker')}</p>
                <h1
                  class="font-display"
                  style="margin: 0; font-size: clamp(23px, 3.1cqi, 40px); font-weight: 500; letter-spacing: -0.025em; line-height: 1.12; color: var(--color-text); text-wrap: pretty;"
                >
                  {headline}
                </h1>
                <p style="margin: 12px 0 0; font-size: clamp(14px, 1.5cqi, 18px); line-height: 1.5; color: var(--color-neutral-400); text-wrap: pretty;">
                  {t(vc.c)}
                </p>
              </div>

              <div
                style="display: flex; flex-direction: column; gap: 12px; padding: 18px; border: 1px solid var(--color-rule); border-radius: 14px; background: var(--color-panel);"
              >
                <span class="section-label">{t('howClose')}</span>
                <div style="display: flex; align-items: stretch; flex-wrap: wrap; gap: 14px;">
                  <div style="display: flex; flex-direction: column; gap: 6px;">
                    <div style="display: flex; border-radius: 8px; overflow: hidden; border: 1px solid var(--color-neutral-800);">
                      <span style="width: 74px; height: 86px; background: rgb({target?.r}, {target?.g}, {target?.b});"></span>
                      <span
                        style="width: 74px; height: 86px; background: {mixSwatch
                          ? `rgb(${mixSwatch.r}, ${mixSwatch.g}, ${mixSwatch.b})`
                          : 'transparent'};"
                      ></span>
                    </div>
                    <div style="display: flex; gap: 14px; font-size: 11px; letter-spacing: 0.06em; text-transform: uppercase; color: var(--color-neutral-500);">
                      <span style="width: 74px;">{t('target')}</span>
                      <span style="width: 74px;">{t('mixture')}</span>
                    </div>
                  </div>
                  <div style="display: flex; flex-direction: column; justify-content: center; gap: 2px; min-width: 0;">
                    <span style="display: flex; align-items: baseline; gap: 8px;">
                      <span class="font-mono" style="font-size: clamp(32px, 4.4cqi, 54px); font-weight: 500; line-height: 1; letter-spacing: -0.03em; color: {deltaColor};"
                        >{decimal(deltaValue, 1)}</span
                      >
                      <span style="font-size: 14px; color: var(--color-neutral-500);">ΔE00</span>
                    </span>
                    <span style="font-size: clamp(14px, 1.45cqi, 17px); font-weight: 500; color: var(--color-text); text-wrap: pretty;">{t(vc.v)}</span>
                  </div>
                </div>
              </div>
            {/if}

            {#if readyPot}
              <div
                style="display: flex; align-items: center; gap: 22px; padding: 22px; border: 1px solid var(--color-accent-700); border-radius: 14px; background: var(--color-accent-panel);"
              >
                <PaintBottle r={readyPot.r} g={readyPot.g} b={readyPot.b} width={72} height={120} label={readyPot.name} />
                <div style="display: flex; flex-direction: column; gap: 6px; min-width: 0;">
                  <span style="font-size: 12px; font-weight: 500; letter-spacing: 0.12em; text-transform: uppercase; color: var(--color-accent-400);">{t('useDirect')}</span>
                  <span style="font-size: clamp(20px, 2.5cqi, 30px); font-weight: 500; letter-spacing: -0.02em; line-height: 1.12; color: var(--color-text);">{readyPot.name}</span>
                  <span class="font-mono" style="font-size: 16px; color: var(--color-neutral-400);">{readyPot.code} · {readyPot.manufacturer}</span>
                </div>
              </div>
              <p style="margin: 0; font-size: 16px; color: var(--color-neutral-400);">{t('mixWell')}</p>
              <button
                class="pressable t1h-ghost"
                onclick={useMixAnyway}
                style="align-self: flex-start; height: 52px; padding: 0 20px; border: 1px solid var(--color-neutral-800); border-radius: 8px; background: transparent; color: var(--color-neutral-400); font-family: inherit; font-size: 15px; font-weight: 500; cursor: pointer;"
                >{t('wantMix')}</button
              >
            {:else if formula}
              {#if !formula.reproducible}
                <div style="padding: 16px 18px; border: 1px solid var(--color-accent-700); border-radius: 14px; background: var(--color-accent-panel);">
                  <p style="margin: 0; font-size: 17px; font-weight: 500; color: var(--color-text);">{t('unreachT')}</p>
                  <p style="margin: 6px 0 0; font-size: 15px; color: var(--color-neutral-400);">
                    {t('unreachB', { pool: formula.crossBrand ? t('poolCross') : t('poolBrand', { brand: formulaTitle }), d: decimal(formula.deltaE, 1) })}
                  </p>
                </div>
              {/if}

              <div style="display: flex; align-items: flex-end; justify-content: space-between; flex-wrap: wrap; row-gap: 12px; gap: 16px;">
                <div>
                  <p class="section-label" style="margin: 0;">{t('formula')}</p>
                  <h2 class="font-display" style="margin: 4px 0 0; font-size: clamp(17px, 2.1cqi, 26px); font-weight: 500; letter-spacing: -0.015em; color: var(--color-text);">
                    {formulaTitle}
                  </h2>
                </div>
                <div style="display: flex; flex-direction: column; align-items: flex-end; gap: 6px;">
                  <button
                    class="pressable t1h-acc"
                    onclick={doSaveRecipe}
                    disabled={!canSaveRecipe}
                    style="display: inline-flex; align-items: center; gap: 10px; height: 56px; padding: 0 22px; border: 1px solid var(--color-accent); border-radius: 8px; background: transparent; color: var(--color-accent-400); font-family: inherit; font-size: 17px; font-weight: 500; cursor: pointer; opacity: {canSaveRecipe ? 1 : 0.5};"
                  >
                    <i class="ph ph-bookmark-simple" style="font-size: 21px;"></i>{t('saveRecipeBtn')}
                  </button>
                  {#if !canSaveRecipe}
                    <span style="font-size: 12px; color: var(--color-neutral-500); max-width: 240px; text-align: right;">{t('saveDisabledCrossBrand')}</span>
                  {/if}
                </div>
              </div>

              <div style="display: flex; height: 12px; border-radius: 4px; overflow: hidden; border: 1px solid var(--color-rule);">
                {#each formula.ingredients as ing, i (ing.paintId)}
                  <span style="height: 12px; width: {pctOf(i)}%; background: rgb({ing.r}, {ing.g}, {ing.b});"></span>
                {/each}
              </div>

              <div style="display: flex; align-items: center; gap: 12px; flex-wrap: wrap;">
                <div style="display: flex; border: 1px solid var(--color-neutral-800); border-radius: 8px; overflow: hidden;">
                  <button
                    class="pressable"
                    onclick={() => (unit = 'drops')}
                    style="height: 44px; padding: 0 14px; border: none; background: {unit === 'drops' ? 'var(--color-accent)' : 'transparent'}; color: {unit === 'drops' ? 'var(--color-accent-100)' : 'var(--color-text)'}; font-family: inherit; font-size: 14px; font-weight: 500; cursor: pointer;"
                    >{t('unitDrops')}</button
                  >
                  <button
                    class="pressable"
                    onclick={() => (unit = 'ml')}
                    style="height: 44px; padding: 0 14px; border: none; border-left: 1px solid var(--color-neutral-800); background: {unit === 'ml' ? 'var(--color-accent)' : 'transparent'}; color: {unit === 'ml' ? 'var(--color-accent-100)' : 'var(--color-text)'}; font-family: inherit; font-size: 14px; font-weight: 500; cursor: pointer;"
                    >{t('unitMl')}</button
                  >
                </div>
                {#if unit === 'ml'}
                  <div role="group" aria-label={t('targetVolumeLabel')} style="display: flex; gap: 6px;">
                    {#each [5, 10, 20] as v (v)}
                      <button
                        class="pressable t1h-border"
                        aria-pressed={targetVolumeMl === v}
                        onclick={() => (targetVolumeMl = v)}
                        style="min-width: 44px; height: 44px; padding: 0 12px; border: 1px solid {targetVolumeMl === v ? 'var(--color-accent)' : 'var(--color-neutral-800)'}; border-radius: 8px; background: {targetVolumeMl === v ? 'var(--color-accent)' : 'transparent'}; color: {targetVolumeMl === v ? 'var(--color-accent-100)' : 'var(--color-text)'}; font-family: inherit; font-size: 13px; font-weight: 500; cursor: pointer;"
                        >{v} ml</button
                      >
                    {/each}
                  </div>
                {/if}
              </div>

              <div style="display: flex; flex-direction: column; gap: 12px;">
                {#each formula.ingredients as ing, i (ing.paintId)}
                  <div
                    style="display: flex; align-items: center; flex-wrap: wrap; row-gap: 12px; gap: 16px; min-height: 100px; padding: 12px 16px; border: 1px solid var(--color-rule); border-radius: 14px; background: var(--color-panel);"
                  >
                    <PaintBottle r={ing.r} g={ing.g} b={ing.b} width={44} height={74} label={ing.name} />

                    <div style="display: flex; flex-direction: column; gap: 6px; flex: 1 1 150px; min-width: 0;">
                      <span style="font-size: clamp(15px, 1.7cqi, 20px); font-weight: 500; color: var(--color-text); white-space: nowrap; overflow: hidden; text-overflow: ellipsis;">{ing.name}</span>
                      <span class="font-mono" style="font-size: 13px; color: var(--color-neutral-500);">{ing.manufacturer}{ing.code ? ` · ${ing.code}` : ''}</span>
                      <button
                        class="pressable"
                        onclick={() => toggleIngredientHave(ing.manufacturerId)}
                        aria-label={t('ariaHave')}
                        style="display: inline-flex; align-items: center; gap: 8px; align-self: flex-start; height: 44px; padding: 0 12px 0 4px; border: none; background: transparent; color: {ingredientHave(ing.manufacturerId)
                          ? 'var(--color-accent-400)'
                          : 'var(--color-neutral-400)'}; font-family: inherit; font-size: 14px; font-weight: 500; cursor: pointer; white-space: nowrap;"
                      >
                        <span
                          style="display: inline-flex; align-items: center; justify-content: center; width: 24px; height: 24px; border: 1px solid {ingredientHave(ing.manufacturerId)
                            ? 'var(--color-accent)'
                            : 'var(--color-neutral-700)'}; border-radius: 4px; background: {ingredientHave(ing.manufacturerId)
                            ? 'var(--color-accent)'
                            : 'transparent'};"
                        >
                          <i class="ph-bold ph-check" style="font-size: 15px; color: {ingredientHave(ing.manufacturerId) ? 'var(--color-accent-100)' : 'transparent'};"></i>
                        </span>
                        {t('haveOnShelf')}
                      </button>
                    </div>

                    <div style="display: flex; align-items: center; gap: 10px; flex-shrink: 0;">
                      <button
                        class="pressable t1h-step"
                        onclick={() => adjustDrop(i, -1)}
                        aria-label={t('ariaMinus')}
                        style="width: 56px; height: 56px; border: 1px solid var(--color-field-border); border-radius: 8px; background: transparent; color: var(--color-text); font-family: inherit; font-size: 26px; line-height: 1; cursor: pointer;"
                        >−</button
                      >
                      <div
                        style="display: flex; flex-direction: column; align-items: center; justify-content: center; width: 96px; height: 56px; border: 1px solid var(--color-rule); border-radius: 8px; background: var(--color-surface);"
                      >
                        {#if unit === 'drops'}
                          <span class="font-mono" style="font-size: clamp(19px, 2cqi, 25px); font-weight: 500; line-height: 1; color: var(--color-text);">{drops[i]}</span>
                          <span style="font-size: 11px; letter-spacing: 0.06em; text-transform: uppercase; color: var(--color-neutral-500);">{drops[i] === 1 ? t('drop') : t('drops')}</span>
                        {:else}
                          <span class="font-mono" style="font-size: clamp(19px, 2cqi, 25px); font-weight: 500; line-height: 1; color: var(--color-text);">{decimal(mlExibido[i] ?? 0, 2)}</span>
                          <span style="font-size: 11px; letter-spacing: 0.06em; text-transform: uppercase; color: var(--color-neutral-500);">{t('unitMl')}</span>
                        {/if}
                      </div>
                      <button
                        class="pressable t1h-step"
                        onclick={() => adjustDrop(i, 1)}
                        aria-label={t('ariaPlus')}
                        style="width: 56px; height: 56px; border: 1px solid var(--color-field-border); border-radius: 8px; background: transparent; color: var(--color-text); font-family: inherit; font-size: 26px; line-height: 1; cursor: pointer;"
                        >+</button
                      >
                    </div>

                    <span class="font-mono" style="min-width: 54px; text-align: right; flex-shrink: 0; font-size: clamp(17px, 1.9cqi, 24px); font-weight: 500; color: var(--color-neutral-400);"
                      >{pctExibido[i] ?? 0}%</span
                    >
                  </div>
                {/each}
              </div>

              <div style="display: flex; align-items: center; gap: 18px;">
                <span class="font-mono" style="font-size: 15px; color: var(--color-neutral-500);">
                  {t('totalLabel', { v: unit === 'drops' ? `${totalDrops} ${t('drops')}` : `${decimal(targetVolumeMl, 2)} ml` })}
                </span>
                <span style="flex: 1;"></span>
                {#if manualDrops}
                  <button
                    class="pressable t1h-link"
                    onclick={resetDrops}
                    style="height: 44px; padding: 0 14px; border: none; background: transparent; color: var(--color-neutral-500); font-family: inherit; font-size: 14px; cursor: pointer;"
                    >{t('resetProp')}</button
                  >
                {/if}
              </div>

              <p style="margin: 0; font-size: 16px; color: var(--color-neutral-400);">{formula.tips?.[0] || t('startBiggest')}</p>

              {#if otherBrands.length > 0}
                <p class="section-label" style="margin: 12px 0 0;">{t('sameOtherBrand')}</p>
                <div style="display: flex; flex-direction: column;">
                  {#each otherBrands as b (b.manufacturerId)}
                    <button
                      class="pressable t1h-row"
                      onclick={() => pickOtherBrand(b)}
                      style="display: flex; align-items: center; gap: 16px; min-height: 64px; padding: 10px 4px; border: none; border-bottom: 1px solid var(--color-line); background: transparent; text-align: left; cursor: pointer; font-family: inherit;"
                    >
                      <span style="width: 44px; height: 36px; border-radius: 4px; border: 1px solid var(--color-neutral-800); background: rgb({b.r}, {b.g}, {b.b}); flex-shrink: 0;"></span>
                      <span style="display: flex; flex-direction: column; flex: 1; min-width: 0;">
                        <span style="font-size: 16px; font-weight: 500; color: var(--color-text);">{b.manufacturer}</span>
                        <span class="font-mono" style="font-size: 13px; color: var(--color-neutral-500); white-space: nowrap; overflow: hidden; text-overflow: ellipsis;"
                          >{b.name}{b.code ? ` · ${b.code}` : ''}</span
                        >
                      </span>
                      <span class="font-mono" style="font-size: 14px; color: {deltaIsGood(b.deltaE) ? 'var(--color-accent-2)' : 'var(--color-neutral-400)'}; flex-shrink: 0;"
                        >ΔE {decimal(b.deltaE, 1)}</span
                      >
                    </button>
                  {/each}
                </div>
              {/if}
            {/if}
          </div>
        {:else}
          <div style="display: flex; flex-direction: column; align-items: flex-start; gap: 10px; padding: 40px 0; color: var(--color-neutral-500);">
            <i class="ph ph-flask" style="font-size: 32px; color: var(--color-neutral-700);"></i>
            <p style="margin: 0; font-size: 16px;">{t('tapToCalc')}</p>
          </div>
        {/if}
      </div>
    {/if}
  </div>

  <!-- Rodapé: navegação (M4) acima do histórico "Na mesa hoje". -->
  <div style="flex-shrink: 0; border-top: 1px solid var(--color-line); background: var(--color-bar);">
    <div style="display: flex; align-items: center; gap: 10px; height: 64px; min-height: 64px; padding: 0 20px; border-bottom: 1px solid var(--color-line);">
      <button
        class="pressable t1h-acc"
        onclick={() => switchTab('plano')}
        style="display: inline-flex; align-items: center; gap: 8px; height: 48px; padding: 0 16px; border: 1px solid var(--color-accent-700); border-radius: 8px; background: transparent; color: var(--color-accent-400); font-family: inherit; font-size: 15px; font-weight: 500; cursor: pointer; flex-shrink: 0; white-space: nowrap;"
      >
        <i class="ph ph-crosshair" style="font-size: 18px;"></i>{t('navPlano')}
      </button>
      <button
        class="pressable t1h-nav"
        onclick={() => switchTab('receitas')}
        style="height: 48px; padding: 0 14px; border: none; border-radius: 8px; background: transparent; color: var(--color-neutral-400); font-family: inherit; font-size: 15px; cursor: pointer; flex-shrink: 0; white-space: nowrap;"
        >{t('navReceitas')}</button
      >
      <button
        class="pressable t1h-nav"
        onclick={() => switchTab('estante')}
        style="height: 48px; padding: 0 14px; border: none; border-radius: 8px; background: transparent; color: var(--color-neutral-400); font-family: inherit; font-size: 15px; cursor: pointer; flex-shrink: 0; white-space: nowrap;"
        >{t('navTintas')}</button
      >
    </div>
    <div style="height: 108px; min-height: 108px; padding: 0 20px; display: flex; align-items: center; gap: 18px;">
      <span style="flex-shrink: 0; width: 84px; font-size: 12px; font-weight: 500; letter-spacing: 0.12em; text-transform: uppercase; color: var(--color-neutral-500); line-height: 1.4;">{t('onTable')}</span>
      <div style="flex: 1; min-width: 0; display: flex; gap: 10px; overflow-x: auto; padding: 6px 0;">
        {#each recents.mesclas as m (m.sourceId + '|' + m.targetManufacturer)}
          <button
            class="pressable t1h-hist"
            onclick={() => rerun(m)}
            style="display: flex; flex-direction: column; align-items: center; gap: 4px; flex-shrink: 0; width: 80px; min-height: 86px; padding: 6px 4px; border: 1px solid transparent; border-radius: 8px; background: transparent; cursor: pointer; font-family: inherit;"
          >
            <PaintBottle r={m.sourceRGB[0]} g={m.sourceRGB[1]} b={m.sourceRGB[2]} width={28} height={47} label={m.sourceName} />
            <span style="font-size: 11.5px; color: var(--color-neutral-400); text-align: center; line-height: 1.25; max-width: 84px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;"
              >{m.sourceName || t('youAsked')}</span
            >
          </button>
        {/each}
      </div>
    </div>
  </div>
</div>

<style>
  .t1 {
    display: flex;
    flex-direction: column;
    height: 100%;
    min-height: 0;
  }

  .t1h-acc:hover {
    background: var(--color-accent-hover);
  }

  .t1h-ghost:hover {
    border-color: var(--color-accent-700);
    color: var(--color-accent-400);
  }

  .t1h-nav:hover {
    color: var(--color-text);
    background: var(--color-surface);
  }

  .t1h-border:hover {
    border-color: var(--color-accent-700);
  }

  .t1h-swatch:hover {
    border-color: var(--color-accent-400);
  }

  .t1h-step:hover {
    border-color: var(--color-accent);
    color: var(--color-accent-400);
  }

  .t1h-link:hover {
    color: var(--color-accent-400);
  }

  .t1h-row:hover {
    background: var(--color-panel);
  }

  .t1h-hist:hover {
    border-color: var(--color-neutral-800);
    background: var(--color-panel);
  }

  .t1-suggest-pop {
    animation: suggest-drop 140ms cubic-bezier(0.4, 0, 0.2, 1) both;
    transform-origin: top center;
  }

  @keyframes suggest-drop {
    from {
      opacity: 0;
      transform: translateY(-6px) scaleY(0.98);
    }
    to {
      opacity: 1;
      transform: translateY(0) scaleY(1);
    }
  }
</style>

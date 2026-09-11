// Revisão do catálogo em memória. services/catalog.ts não é reativo, e as
// quatro telas ficam montadas ao mesmo tempo (App.svelte só as esconde): quem
// lista fabricante lê `catalogRev.n` para se refazer depois de cada recarga
// (rf-14 — cadastrar, alterar ou excluir em T4 aparece em T1 e T3 na hora).
export const catalogRev = $state({ n: 0 });

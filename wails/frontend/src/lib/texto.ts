// Normalização de texto para busca: sem maiúscula e sem acento. Uma função só
// para o Combobox e a lista de projetos de T2 buscarem do mesmo jeito.
export function norm(s: string): string {
  return s.toLowerCase().normalize('NFD').replace(/[̀-ͯ]/g, '');
}

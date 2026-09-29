// Baixa um texto como arquivo pelo navegador (rf-23). Sem BOM: o CSV exportado
// precisa voltar na importação sem erro de cabeçalho.

export function baixarTexto(nome: string, texto: string, tipo = 'text/csv;charset=utf-8'): void {
  const url = URL.createObjectURL(new Blob([texto], { type: tipo }));
  const a = document.createElement('a');
  a.href = url;
  a.download = nome;
  a.style.display = 'none';
  document.body.appendChild(a);
  a.click();
  a.remove();
  // Revogar logo em seguida cancela o download no Safari do iPad.
  setTimeout(() => URL.revokeObjectURL(url), 60_000);
}

# Fonte dos dados de catálogo

Os arquivos `*.md` deste diretório (um por marca, tabela `Name | Code | Set | R | G | B | Hex`)
foram obtidos do dataset público **Miniature Painter Pro**:

- Repositório: https://github.com/Arcturus5404/miniature-paints
- App: https://miniaturepainterpro.app/

Importados para o banco pelo seed `db/seeds/007_catalog_importer.go`
(rodar com `go run ./cmd/seed -import-catalog`).

Os valores RGB são medições feitas pela comunidade (aproximações realistas dos
tons de tinta), não swatches oficiais dos fabricantes.

## Exceções (dados locais, fora do dataset MPP)

- **Corfix.md** — transcrito manualmente da cartela oficial Decorfix
  (Fosca 110 + Brilhante 24 + Metálica 15 + Fluo 8 = 157 cores), lendo o RGB de
  cada swatch impresso. Precisão limitada pela reprodução de cor da cartela.
- **Talento.md** — o fabricante não publica cartela com valores de cor. As 333
  cores (linhas Game, Hobby, Metálicas, Modelismo, Plus e Speed) foram extraídas
  das fotos de produto do site oficial (talentoartes.com.br): cor amostrada do
  corpo do frasco por análise de imagem, com revisão visual individual e
  correções manuais (~90 itens). Padrões militares (RLM/RAL/FS) usam valores
  canônicos. Precisão menor que as demais fontes — aproximações de foto.

## Licença (MIT)

MIT License — Copyright (c) 2022 Rick Fleuren

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.

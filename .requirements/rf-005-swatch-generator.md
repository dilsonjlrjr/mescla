# RF-005 — Swatch Generator

## Metadata
- **ID**: RF-005
- **Nome**: Swatch Generator
- **Status**: In Progress
- **Data**: 2026-06-30
- **Dependências**: RF-001, RF-004

## Objetivo
Gerar imagens de swatch (amostras de cor) para cada tinta cadastrada no banco de dados, representando visualmente a cor da tinta.

## Escopo
1. Criar função que gera imagem PNG de swatch 64x64px com a cor da tinta
2. Salvar swatches em `assets/swatches/`
3. Registrar path do swatch no banco de dados (campo `swatch_path` na tabela `paints`)
4. Suportar geração em lote para todas as tintas
5. Flag `--generate-swatches` no main.go

## Critérios de Aceite
- [ ] Cada tinta tem um swatch PNG 64x64px gerado
- [ ] Swatch é salvo em `assets/swatches/{paint_id}_{slug}.png`
- [ ] Path é registrado no campo `swatch_path` da tabela `paints`
- [ ] Geração em lote funciona para todas as 45 tintas
- [ ] Flag `--generate-swatches` executa a geração
- [ ] Código compila e roda sem erros

## Restrições
- Usar Go standard library para gerar imagens (image/color)
- Swatch deve ser sólido (sem gradientes)
- Formato PNG

## Não Incluído
- Paletas de mistura
- Efeitos visuais (textura, brilho)
- Comparação lado a lado

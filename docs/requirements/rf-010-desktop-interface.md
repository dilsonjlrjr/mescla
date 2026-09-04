# RF-010 — Desktop Interface

## ID
RF-010

## Título
Desktop Interface — Wails 3 + Svelte 5 + TypeScript + TailwindCSS

## Status
In Progress

## Objetivo
Criar a interface desktop completa do Paint Match AI utilizando Wails 3, Svelte 5, TypeScript e TailwindCSS.

## Motivação
O backend está pronto (RF-001 a RF-009). Precisamos da interface visual para que o usuário interaja com o sistema.

## Escopo
- Setup Wails 3 no projeto existente
- Configurar frontend Svelte 5 + TypeScript + TailwindCSS
- Criar Go services com bindings para o frontend
- Implementar telas: Home, Catálogo, Comparação, Mistura, Busca por Cor
- Design moderno, bonito, informativo

## O que faz
- Conecta frontend ao backend Go via Wails bindings
- Exibe catálogo de tintas com swatches, thumbnails, informações técnicas
- Permite busca por cor (similaridade)
- Permite comparação entre tintas
- Permite sugestão de misturas
- Exibe equivalências entre fabricantes

## O que não faz
- Não modifica banco de dados
- Não altera lógica de negócio existente
- Não implementa IA conversacional (RF-009 é retrieval apenas)

## Dependências
- RF-001 Database Schema ✓
- RF-002 Manufacturer Importer ✓
- RF-003 Asset Downloader ✓
- RF-004 Paint Importer ✓
- RF-005 Swatch Generator ✓
- RF-006 Color Converter ✓
- RF-007 Mix Engine ✓
- RF-008 Similarity Engine ✓
- RF-009 AI Retrieval ✓

## Pré-requisitos
- Wails 3 CLI instalado ✓
- Node.js + npm instalados ✓
- Todos os pacotes Go backend prontos ✓

## Gatilhos
- Usuário abre a aplicação desktop
- Usuário pesquisa tinta
- Usuário seleciona cor
- Usuário pede sugestão de mistura

## Fluxo esperado
1. App abre → Home com busca central
2. Usuário pesquisa → resultados com swatches
3. Usuário seleciona tinta → detalhes completos
4. Usuário pede equivalências → lista de tintas similares
5. Usuário pede mistura → receita com proporções

## Estrutura técnica
- Wails 3 application framework
- Go services com métodos exportados (bindings)
- Svelte 5 com runes ($state, $derived, $effect)
- TypeScript para type safety
- TailwindCSS para estilização
- Vite para bundling

## Arquivos envolvidos
- `main.go` — entry point Wails app
- `app.go` — PaintService com bindings
- `frontend/` — Svelte 5 project
- `frontend/src/App.svelte` — app principal
- `frontend/src/lib/` — componentes
- `frontend/package.json` — dependências
- `frontend/vite.config.ts` — config Vite
- `Taskfile.yml` — build tasks
- `build/` — build config

## Bibliotecas
- github.com/wailsapp/wails/v3
- @wailsio/runtime
- svelte 5
- typescript
- tailwindcss
- vite

## Banco de dados
- paint_knowledge.db (existente)

## Critérios de aceite
- [ ] App abre com janela desktop
- [ ] Home exibe busca
- [ ] Catálogo de tintas funciona
- [ ] Busca por similaridade funciona
- [ ] Comparação entre tintas funciona
- [ ] Sugestão de mistura funciona
- [ ] Swatches são exibidos corretamente
- [ ] Design é bonito e informativo

## Critérios de qualidade
- Código limpo e organizado
- Componentes reutilizáveis
- Responsivo
- Performance adequada
- Type safety completo

## Exemplo de uso
```bash
wails3 dev
# Abre janela desktop com Paint Match AI
```

## Próximos requisitos dependentes
- Nenhum (RF-010 é o último)

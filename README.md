# Mescla AI

Equivalência de tintas para pintores de miniaturas: dada uma cor ou um pote,
o Mescla encontra a tinta mais próxima em qualquer marca e monta a receita de
mistura com o que o pintor tem no estoque.

O produto sai em duas formas, **com a mesma interface**:

- **Web (PWA)**, em `front/`. Roda no navegador e fala com o `mescla-api` pela rede.
- **Desktop**, em `wails/`. É um app nativo (Wails v3) para macOS, Windows e
  Linux. Funciona offline, com o banco embutido.

A web é a referência de interface. O desktop mostra exatamente as mesmas telas.

## Estrutura

```
api/        Motor e dados em Go (módulo mescla-ai)
  service/    PaintService: catálogo, receitas, estoque, projetos, relatórios
  domain/     color, similarity, mix, equivalence, stock, ai
  db/         migrações e seeds
  httpapi/    rotas HTTP (fasthttp) que a web consome
  cmd/        apiserver (o mescla-api) e seed
front/      Interface web (Svelte 5 + Vite, PWA)
wails/      App desktop (Wails v3)
  frontend/   cópia da interface web + ponte para os bindings
  catalogo/   banco de catálogo que o desktop embute (versionado)
  cmd/        exportar-catalogo
  build/      ícones e empacotamento por plataforma
  scripts/    build-all.sh, exportar-catalogo.sh e sincronizar-com-web.sh
```

## Pré-requisitos

- Go 1.26 (ver `.tool-versions`)
- Node.js e npm
- Para o desktop: `wails3` (Wails v3 alpha). No Linux, os headers de GTK4 e WebKitGTK.
- Para o build Linux feito a partir do macOS: Docker

## Rodar em desenvolvimento

**API e web**

```bash
make api                          # mescla-api em :8080 (MESCLA_LISTEN_ADDR muda a porta)
cd front && npm install && npm run dev
```

Em dev, o Vite encaminha `/api` para `http://localhost:8080`.

**Desktop**

```bash
cd wails && wails3 task dev
```

Os bindings TypeScript (`wails/frontend/bindings/`) são gerados pelo build e
ficam fora do git. Para gerá-los à mão, rode
`cd wails && wails3 generate bindings -clean=true -ts -i`.

**Banco**

O `PaintService` usa o banco indicado em `MESCLA_DB_PATH`. Sem essa variável, ele
procura `data/paint_knowledge.db` e depois `paint_knowledge.db`. Se não achar
nenhum dos dois, cria o banco em `<config do usuário>/Mescla/` a partir do
catálogo embutido no binário. No desktop, esse catálogo é
`wails/catalogo/paint_knowledge.db`.

## Build e deploy

| Alvo | Comando | Saída |
|---|---|---|
| macOS (arm64, amd64, universal, .app, DMG) | `make macos` | `wails/dist/` |
| Windows amd64 (cross-compile, sem CGO) | `make windows` | `wails/dist/` |
| Linux amd64 (via Docker) | `make linux` | `wails/dist/` (.tar.gz com `install.sh`) |
| Web (PWA) | `make mobile` | `front/dist/` |
| Web + API no servidor | `docker compose up -d --build` | nginx na porta 50444 |

No compose, o `builder` gera o banco e o PWA, o `api` serve o `mescla-api` e o
`web` (nginx) serve o PWA e encaminha `/api/` para o `api`. Não compile todas as
imagens em paralelo: o build pode estourar a memória. Compile `api` e
`builder` um de cada vez e depois rode `up -d`.

## Catálogo do desktop

O desktop embute o catálogo como está cadastrado na web: fabricantes, linhas,
tintas, cores, tipos de tinta e a marca de ignorar na mistura. O estoque
(Minhas tintas), os projetos de pintura e as receitas salvas **não** entram.

Para atualizar o catálogo depois de cadastrar algo na web:

```bash
./wails/scripts/exportar-catalogo.sh            # lê o banco do container mescla-api-1
./wails/scripts/exportar-catalogo.sh <banco.db> # ou de um arquivo
```

Depois, faça commit de `wails/catalogo/paint_knowledge.db` e gere o build.
O `build-all.sh` para se o arquivo não existir.

O catálogo só é instalado no primeiro boot, quando não existe banco em
`<config do usuário>/Mescla/`. Um desktop já instalado continua com o banco
que tem. Para receber o catálogo novo, apague esse arquivo. Isso também apaga
o estoque, os projetos e as receitas guardados naquele desktop.

## Testes e checagens

```bash
go test ./api/...
cd front && npm test && npm run check
cd wails/frontend && npm run check && npm run build
```

## Desktop igual à web: como manter

`wails/frontend/src` é uma **cópia fiel** de `front/src`. Só três partes são
próprias do desktop:

- `src/main.ts` instala a ponte antes de montar a interface.
- `src/vite-env.d.ts` não traz os tipos do PWA.
- `src/desktop/` guarda o código que só o desktop tem:
  - `ponteApi.ts` troca o `fetch('/api/...')` da web por chamadas aos bindings
    do `PaintService`, dentro do processo e sem rede. Espelha rota, método,
    corpo, status e mensagem de erro de `api/httpapi`.
  - `downloadNativo.ts` faz o `<a download>` da web abrir o diálogo nativo de
    salvar (`DialogService.SaveFileAs`, em `wails/dialogs.go`).
  - `migrarReceitas.ts` converte as receitas salvas pela interface antiga do
    desktop.

Regras:

1. **Não edite telas dentro de `wails/frontend/src`.** Toda mudança de interface
   é feita em `front/`. Depois, rode:

   ```bash
   ./wails/scripts/sincronizar-com-web.sh
   ```

   O script copia `front/src` e `front/public` para o desktop e preserva
   `desktop/`, `main.ts` e `vite-env.d.ts`. No fim, ele lista o que ainda difere.
   O esperado é só esses três itens.

2. **Rota nova ou alterada em `api/httpapi` precisa do mesmo caso em
   `wails/frontend/src/desktop/ponteApi.ts`.** Sem isso, o desktop responde
   404 (`rota não encontrada`) onde a web funciona. O método do `PaintService`
   também precisa estar exposto no binding (`wails3 generate bindings`).

3. **Dependência nova em `front/package.json` entra também em
   `wails/frontend/package.json`.** O desktop usa as mesmas dependências da
   web, mais `@wailsio/runtime`.

# RF-003 — Asset Downloader

## Status

Done

---

## Objetivo

Criar o processo de download e armazenamento local de todos os recursos visuais da aplicação: logotipos de fabricantes, thumbnails de tintas e imagens oficiais (fotos dos frascos).

---

## Motivação

Recursos visuais são essenciais para a experiência da aplicação. Sem eles, a interface fica genérica e os usuários não conseguem identificar visualmente as tintas. Esta etapa prepara todos os assets antes da importação de dados.

---

## Escopo

- Download de logotipos oficiais dos fabricantes
- Download de thumbnails de tintas (miniaturas)
- Download de imagens oficiais (fotos dos frascos)
- Armazenamento local em diretórios organizados
- Atualização dos campos `logo_path`, `thumbnail_path`, `image_path` no banco
- Registro na tabela `resources`
- Tratamento de erros de download

---

## O que faz

- Cria estrutura de diretórios `assets/`
- Baixa logotipo de cada fabricante via URL oficial
- Baixa thumbnail de cada tinta (miniatura para listagens)
- Baixa imagem oficial de cada tinta (foto do frasco em alta resolução)
- Salva com nomenclatura padronizada
- Atualiza paths no banco (`manufacturers.logo_path`, `paints.thumbnail_path`, `paints.image_path`)
- Registra todos os downloads na tabela `resources`
- Retry em caso de falha de rede

---

## O que não faz

- Não processa ou redimensiona imagens
- Não gera swatches (RF-005)
- Não cria interface para visualização
- Não baixa catálogos completos

---

## Dependências

- **RF-002** — Manufacturer Importer (concluído)
- **RF-004** — Paint Importer (para thumbnails/imagens de tintas)

---

## Pré-requisitos

- Fabricantes cadastrados no banco
- Tintas cadastradas no banco (para thumbnails/imagens)
- Acesso à internet para download
- URLs de imagens válidas

---

## Gatilhos

- Executado após RF-002 para logotipos de fabricantes
- Executado após RF-004 para thumbnails/imagens de tintas
- Pode ser re-executado para atualizar assets

---

## Fluxo esperado

### Fase 1 — Logotipos de Fabricantes

1. Consultar fabricantes no banco
2. Para cada fabricante:
   a. Verificar se logotipo já existe localmente
   b. Se não existe, baixar da URL oficial
   c. Salvar em `assets/manufacturers/`
   d. Atualizar `logo_path` no banco
   e. Registrar em `resources`

### Fase 2 — Thumbnails de Tintas

1. Consultar tintas no banco
2. Para cada tinta:
   a. Verificar se thumbnail já existe localmente
   b. Se não existe, baixar da URL do catálogo
   c. Salvar em `assets/paints/thumbnails/`
   d. Atualizar `thumbnail_path` no banco
   e. Registrar em `resources`

### Fase 3 — Imagens Oficiais (Frascos)

1. Consultar tintas no banco
2. Para cada tinta:
   a. Verificar se imagem já existe localmente
   b. Se não existe, baixar da URL oficial
   c. Salvar em `assets/paints/images/`
   d. Atualizar `image_path` no banco
   e. Registrar em `resources`

---

## Estrutura técnica

### Diretório de assets

```
assets/
├── manufacturers/
│   ├── 1_ak_interactive.png
│   ├── 2_vallejo.png
│   └── ...
└── paints/
    ├── thumbnails/
    │   ├── 123_ak_3gen_001_thumb.png
    │   └── ...
    └── images/
        ├── 123_ak_3gen_001.png
        └── ...
```

### Nomenclatura de arquivos

| Tipo | Padrão | Exemplo |
|------|--------|---------|
| Logo fabricante | `{id}_{manufacturer_slug}.{ext}` | `1_ak_interactive.png` |
| Thumbnail tinta | `{id}_{manufacturer}_{line}_{code}_thumb.{ext}` | `123_ak_3gen_001_thumb.png` |
| Imagem tinta | `{id}_{manufacturer}_{line}_{code}.{ext}` | `123_ak_3gen_001.png` |

### Funções principais

```go
// DownloadManufacturerLogos baixa logotipos de todos os fabricantes
func DownloadManufacturerLogos(db *sql.DB, assetsDir string) error {
    // ...
}

// DownloadPaintThumbnails baixa thumbnails de todas as tintas
func DownloadPaintThumbnails(db *sql.DB, assetsDir string) error {
    // ...
}

// DownloadPaintImages baixa imagens oficiais de todas as tintas
func DownloadPaintImages(db *sql.DB, assetsDir string) error {
    // ...
}

// DownloadAllAssets executa todos os downloads
func DownloadAllAssets(db *sql.DB, assetsDir string) error {
    // ...
}
```

### SQL de atualização

```sql
-- Logotipo do fabricante
UPDATE manufacturers SET logo_path = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?;

-- Thumbnail da tinta
UPDATE paints SET thumbnail_path = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?;

-- Imagem da tinta
UPDATE paints SET image_path = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?;

-- Registro de recurso
INSERT INTO resources (paint_id, manufacturer_id, type, path, url, description)
VALUES (?, ?, ?, ?, ?, ?);
```

### HTTP Client

```go
type DownloadConfig struct {
    Timeout     time.Duration
    MaxRetries  int
    RetryDelay  time.Duration
    UserAgent   string
    Concurrency int
}
```

---

## Arquivos envolvidos

- `db/seeds/003_asset_downloader.go` — Lógica de download
- `assets/manufacturers/` — Logotipos de fabricantes
- `assets/paints/thumbnails/` — Thumbnails de tintas
- `assets/paints/images/` — Imagens oficiais de tintas

---

## Bibliotecas

- `net/http` (padrão Go)
- `io` (padrão Go)
- `os` (padrão Go)
- `path/filepath` (padrão Go)
- `sync` (padrão Go — downloads paralelos)
- `time` (padrão Go — retry/timeout)

---

## Banco de dados

- Tabelas: `manufacturers`, `paints`, `resources`
- Operações: UPDATE, INSERT, SELECT

---

## Critérios de aceite

1. Diretórios `assets/` são criados com estrutura correta
2. Logotipos são baixados para os 10 fabricantes
3. Thumbnails são baixados para todas as tintas cadastradas
4. Imagens oficiais são baixadas para todas as tintas cadastradas
5. Paths são atualizados no banco (`logo_path`, `thumbnail_path`, `image_path`)
6. Registros são inseridos em `resources`
7. Download falha graciosamente (log de erro, continua para próximo)
8. Re-execução não rebaixa assets existentes
9. Downloads paralelos funcionam sem race conditions

---

## Critérios de qualidade

- HTTP client com timeout configurável (30s padrão)
- Retry com backoff exponencial (3 tentativas)
- User-Agent adequado nas requisições
- Logs de progresso detalhados (por fabricante/tinta)
- Tratamento de diferentes Content-Types (png, jpg, webp, svg)
- Concurrency limitada (max 5 downloads simultâneos)
- Verificação de integridade (Content-Length vs bytes recebidos)

---

## Exemplo de uso

```bash
# Executar todos os downloads
go run main.go --download-assets

# Apenas logotipos de fabricantes
go run main.go --download-logos

# Apenas imagens de tintas
go run main.go --download-paint-images

# Verificar assets
ls -la assets/manufacturers/
ls -la assets/paints/thumbnails/
ls -la assets/paints/images/

# Verificar banco
sqlite3 paint_knowledge.db "SELECT id, name, logo_path FROM manufacturers;"
sqlite3 paint_knowledge.db "SELECT id, name, thumbnail_path, image_path FROM paints LIMIT 10;"
```

---

## Próximos requisitos dependentes

- **RF-005** — Swatch Generator (gera swatches visuais a partir das cores)
- **RF-010** — Desktop Interface (utiliza todos os assets visuais)

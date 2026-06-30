# RF-002 — Manufacturer Importer

## Status

Done

---

## Objetivo

Criar o processo de importação e população da tabela `manufacturers` com todos os fabricantes de tintas para modelismo definidos na base de conhecimento.

---

## Motivação

Fabricantes são a raiz da hierarquia de dados. Sem fabricantes cadastrados, não é possível criar linhas de produtos, importar tintas ou gerar equivalências. Esta é a primeira dependência da cadeia de dados.

---

## Escopo

- Definição da lista completa de fabricantes
- Cadastramento com nome, país, site oficial e caminho do logotipo
- Script Go para inserção idempotent (não duplica se já existe)
- Validação de dados antes da inserção

---

## O que faz

- Popula tabela `manufacturers` com fabricantes reais
- Inclui: AK Interactive, Vallejo, Citadel, Army Painter, Scale75, Tamiya, Mr Hobby, Acrilex, Corfix, Talento
- Armazena país de origem e website oficial
- Prepara espaço para logotipos (campo `logo_path`)
- Execução idempotent (verifica existência antes de inserir)

---

## O que não faz

- Não baixa logotipos (RF-003 cuida de recursos visuais)
- Não cria linhas de produtos (RF-004)
- Não cria tintas (RF-004)
- Não implementa interface gráfica

---

## Dependências

- **RF-001** — Database Schema (concluído)

---

## Pré-requisitos

- Banco `paint_knowledge.db` existente
- Tabela `manufacturers` criada

---

## Gatilhos

- Executado uma vez na configuração inicial
- Pode ser re-executado para adicionar novos fabricantes

---

## Fluxo esperado

1. Carregar lista de fabricantes (hardcoded ou arquivo JSON)
2. Para cada fabricante:
   a. Verificar se já existe pelo nome
   b. Se não existe, inserir com dados completos
   c. Se existe, atualizar dados se necessário
3. Retornar relatório de inserções/atualizações

---

## Estrutura técnica

### Dados por fabricante

```go
type ManufacturerSeed struct {
    Name    string
    Country string
    Website string
    LogoURL string // URL para download futuro
}
```

### Lista de fabricantes

| Fabricante | País | Website |
|------------|------|---------|
| AK Interactive | Spain | https://ak-interactive.com |
| Vallejo | Spain | https://acrylicosvallejo.com |
| Citadel | UK | https://www.games-workshop.com |
| Army Painter | Denmark | https://www.thearmypainter.com |
| Scale75 | Spain | https://scale75.com |
| Tamiya | Japan | https://www.tamiya.com |
| Mr Hobby | Japan | https://www.gsiCreos.co.jp |
| Acrilex | Brazil | https://www.acrilex.com.br |
| Corfix | Brazil | https://www.corfix.com.br |
| Talento | Brazil | https://www.talento.com.br |

### Função principal

```go
// SeedManufacturers popula fabricantes no banco
func SeedManufacturers(db *sql.DB) error {
    // ...
}
```

### SQL de inserção

```sql
INSERT INTO manufacturers (name, country, website)
VALUES (?, ?, ?)
ON CONFLICT(name) DO UPDATE SET
    country = excluded.country,
    website = excluded.website,
    updated_at = CURRENT_TIMESTAMP;
```

---

## Arquivos envolvidos

- `db/seeds/002_manufacturers.go` — Dados e lógica de inserção
- `db/seeds/seed.go` — Executor central de seeds (criado neste requisito)

---

## Bibliotecas

- `database/sql` (padrão Go)
- `modernc.org/sqlite` ou `github.com/mattn/go-sqlite3`

---

## Banco de dados

- Tabela: `manufacturers`
- Operação: INSERT com ON CONFLICT (upsert)
- Verificação: SELECT por nome antes de inserir

---

## Critérios de aceite

1. Todos os 10 fabricantes são inseridos no banco
2. Execução dupla não cria duplicatas
3. Dados de país e website estão corretos
4. Fabricantes podem ser consultados via SQL
5. Script executa sem erros

---

## Critérios de qualidade

- Código Go idiomático
- Tratamento de erros adequado
- Logs de progresso durante inserção
- Função reutilizável para futuros seeds
- Testes unitários para lógica de inserção

---

## Exemplo de uso

```bash
# Executar seed
go run db/seeds/002_manufacturers.go

# Verificar resultados
sqlite3 paint_knowledge.db "SELECT id, name, country, website FROM manufacturers;"
```

Resultado esperado:
```
1|AK Interactive|Spain|https://ak-interactive.com
2|Vallejo|Spain|https://acrylicosvallejo.com
3|Citadel|UK|https://www.games-workshop.com
4|Army Painter|Denmark|https://www.thearmypainter.com
5|Scale75|Spain|https://scale75.com
6|Tamiya|Japan|https://www.tamiya.com
7|Mr Hobby|Japan|https://www.gsiCreos.co.jp
8|Acrilex|Brazil|https://www.acrilex.com.br
9|Corfix|Brazil|https://www.corfix.com.br
10|Talento|Brazil|https://www.talento.com.br
```

---

## Próximos requisitos dependentes

- **RF-003** — Thumbnail Downloader (baixa logotipos dos fabricantes)
- **RF-004** — Paint Importer (cria linhas e tintas por fabricante)

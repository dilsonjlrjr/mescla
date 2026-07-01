package seeds

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// AssetDownloaderConfig configura o downloader
type AssetDownloaderConfig struct {
	AssetsDir   string
	Timeout     time.Duration
	MaxRetries  int
	RetryDelay  time.Duration
	UserAgent   string
	Concurrency int
}

// DefaultConfig retorna configuração padrão
func DefaultConfig() AssetDownloaderConfig {
	return AssetDownloaderConfig{
		AssetsDir:   "assets",
		Timeout:     30 * time.Second,
		MaxRetries:  3,
		RetryDelay:  2 * time.Second,
		UserAgent:   "PaintMatchAI/1.0",
		Concurrency: 5,
	}
}

// ManufacturerAsset representa um asset de fabricante para download
type ManufacturerAsset struct {
	ID      int
	Name    string
	LogoURL string
}

// manufacturerLogos contém URLs conhecidas de logotipos
var manufacturerLogos = map[string]string{
	"AK Interactive":   "https://ak-interactive.com/wp-content/uploads/2022/03/blanco_logo-02-copia.png",
	"Vallejo":          "https://cdn.worldvectorlogo.com/logos/vallejo.svg",
	"Citadel":          "https://www.games-workshop.com/resources/logo-citadel.png",
	"Army Painter":     "https://thearmypainter.com/cdn/shop/files/logo.svg",
	"Scale75":          "https://scale75.com/cdn/shop/files/00__Logo_Scale75_CMYK.jpg?v=1753709085",
	"Tamiya":           "https://www.tamiya.com/cms/images/new_blklogo.gif",
	"Mr Hobby":         "https://www.gsiCreos.co.jp/images/logo.png",
	"Acrilex":          "https://acrilex.com.br/wp-content/uploads/2024/01/logo-acrilex.png",
	"Corfix":           "https://www.corfix.com.br/wp-content/themes/bones/library/images/logotipo.png",
	"Green Stuff World": "https://www.greenstuffworld.com/img/logo-1699283611.jpg",
}

// AssetDownloader gerencia downloads de assets
type AssetDownloader struct {
	db     *sql.DB
	config AssetDownloaderConfig
	client *http.Client
	dbMu   sync.Mutex // serializa escritas concorrentes; SQLite só aceita um escritor por vez
}

// NewAssetDownloader cria um novo downloader
func NewAssetDownloader(db *sql.DB, config AssetDownloaderConfig) *AssetDownloader {
	return &AssetDownloader{
		db:     db,
		config: config,
		client: &http.Client{Timeout: config.Timeout},
	}
}

// DownloadManufacturerLogos baixa logotipos de todos os fabricantes
func (d *AssetDownloader) DownloadManufacturerLogos() error {
	dir := filepath.Join(d.config.AssetsDir, "manufacturers")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("criar diretório %s: %w", dir, err)
	}

	rows, err := d.db.Query("SELECT id, name FROM manufacturers ORDER BY id")
	if err != nil {
		return fmt.Errorf("consultar fabricantes: %w", err)
	}
	defer rows.Close()

	type task struct {
		id      int
		name    string
		logoURL string
	}

	var tasks []task
	for rows.Next() {
		var id int
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return fmt.Errorf("ler fabricante: %w", err)
		}
		if url, ok := manufacturerLogos[name]; ok {
			tasks = append(tasks, task{id: id, name: name, logoURL: url})
		} else {
			log.Printf("[asset] URL de logo não encontrada para: %s", name)
		}
	}

	return d.downloadConcurrent(len(tasks), func(i int) error {
		t := tasks[i]
		slug := slugify(t.name)
		filename := fmt.Sprintf("%d_%s.png", t.id, slug)
		destPath := filepath.Join(dir, filename)

		if err := d.downloadFile(t.logoURL, destPath); err != nil {
			log.Printf("[asset] Erro baixando logo %s: %v", t.name, err)
			return nil // Continua para próximo
		}

		// Atualizar banco (serializado: SQLite não aceita escritas concorrentes)
		relPath, _ := filepath.Rel(".", destPath)
		d.dbMu.Lock()
		defer d.dbMu.Unlock()

		_, err := d.db.Exec(
			"UPDATE manufacturers SET logo_path = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
			relPath, t.id,
		)
		if err != nil {
			return fmt.Errorf("atualizar logo_path para %s: %w", t.name, err)
		}

		// Registrar em resources
		_, err = d.db.Exec(
			"INSERT INTO resources (manufacturer_id, type, path, url, description) VALUES (?, ?, ?, ?, ?)",
			t.id, "logo", relPath, t.logoURL, fmt.Sprintf("Logo of %s", t.name),
		)
		if err != nil {
			return fmt.Errorf("registrar resource para %s: %w", t.name, err)
		}

		log.Printf("[asset] Logo baixado: %s -> %s", t.name, relPath)
		return nil
	})
}

// downloadConcurrent executa downloads em paralelo
func (d *AssetDownloader) downloadConcurrent(total int, fn func(i int) error) error {
	if total == 0 {
		return nil
	}

	sem := make(chan struct{}, d.config.Concurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errs []error

	for i := 0; i < total; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			if err := fn(idx); err != nil {
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()

	if len(errs) > 0 {
		return fmt.Errorf("%d erros no download: %v", len(errs), errs[0])
	}
	return nil
}

// downloadFile baixa um arquivo com retry
func (d *AssetDownloader) downloadFile(url, destPath string) error {
	// Verificar se já existe
	if _, err := os.Stat(destPath); err == nil {
		log.Printf("[asset] Já existe: %s", destPath)
		return nil
	}

	var lastErr error
	for attempt := 0; attempt < d.config.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := d.config.RetryDelay * time.Duration(1<<uint(attempt-1))
			log.Printf("[asset] Retry %d/%d em %v", attempt+1, d.config.MaxRetries, delay)
			time.Sleep(delay)
		}

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return fmt.Errorf("criar request: %w", err)
		}
		req.Header.Set("User-Agent", d.config.UserAgent)

		resp, err := d.client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request falhou: %w", err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("status %d: %s", resp.StatusCode, resp.Status)
			continue
		}

		// Criar diretório pai se necessário
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return fmt.Errorf("criar diretório: %w", err)
		}

		out, err := os.Create(destPath)
		if err != nil {
			return fmt.Errorf("criar arquivo: %w", err)
		}

		_, err = io.Copy(out, resp.Body)
		out.Close()
		if err != nil {
			os.Remove(destPath)
			lastErr = fmt.Errorf("salvar arquivo: %w", err)
			continue
		}

		return nil
	}

	return fmt.Errorf("falhou após %d tentativas: %w", d.config.MaxRetries, lastErr)
}

// slugify converte nome em slug para filename
func slugify(name string) string {
	slug := strings.ToLower(name)
	slug = strings.ReplaceAll(slug, " ", "_")
	slug = strings.ReplaceAll(slug, "-", "_")
	// Remover caracteres especiais
	var result strings.Builder
	for _, r := range slug {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// GetAssetDownloaderSeed retorna o Seed registro
func GetAssetDownloaderSeed() Seed {
	return Seed{
		Name: "003_asset_downloader",
		Fn: func(db *sql.DB) error {
			config := DefaultConfig()
			downloader := NewAssetDownloader(db, config)
			return downloader.DownloadManufacturerLogos()
		},
	}
}

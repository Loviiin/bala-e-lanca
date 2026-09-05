// Package obsidian fornece writers para persistir leituras da balança
// no vault do Obsidian — tanto no filesystem local quanto diretamente
// no CouchDB via protocolo Self-hosted LiveSync.
//
// O LiveSyncWriter implementa o formato de documentos que o plugin
// LiveSync espera: um documento de metadados (tipo "plain") com o
// caminho do arquivo e referências a chunks, e documentos "leaf"
// com o conteúdo de cada chunk. O ID de cada leaf é gerado com
// xxhash64 em base36, que é o algoritmo que o plugin valida.
package obsidian

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/cespare/xxhash/v2"
)

// LiveSyncConfig contém as credenciais e URL do CouchDB.
type LiveSyncConfig struct {
	URL      string // ex: "http://127.0.0.1:5984"
	Database string // ex: "obsidian-livesync"
	Username string
	Password string
}

// LiveSyncWriter escreve documentos no CouchDB no formato que o plugin
// Self-hosted LiveSync do Obsidian entende: um doc de metadados por
// arquivo e um doc "leaf" por chunk de conteúdo.
type LiveSyncWriter struct {
	cfg    LiveSyncConfig
	client *http.Client
}

// NewLiveSyncWriter cria um writer configurado para falar com o CouchDB.
func NewLiveSyncWriter(cfg LiveSyncConfig) *LiveSyncWriter {
	return &LiveSyncWriter{
		cfg: cfg,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ---- formato dos documentos LiveSync ----

// liveSyncMeta é o documento de metadados que o LiveSync cria pra cada
// arquivo no vault. O campo children lista os IDs dos chunks (leafs).
type liveSyncMeta struct {
	ID       string                 `json:"_id"`
	Rev      string                 `json:"_rev,omitempty"`
	Type     string                 `json:"type"`
	Path     string                 `json:"path"`
	Children []string               `json:"children"`
	CTime    int64                  `json:"ctime"`
	MTime    int64                  `json:"mtime"`
	Size     int                    `json:"size"`
	Eden     map[string]interface{} `json:"eden"`
}

// liveSyncLeaf é o documento de conteúdo (chunk). O _id é "h:" + hash.
type liveSyncLeaf struct {
	ID   string `json:"_id"`
	Rev  string `json:"_rev,omitempty"`
	Type string `json:"type"`
	Data string `json:"data"`
}

// couchResponse é a resposta padrão do CouchDB a um PUT/DELETE.
type couchResponse struct {
	OK  bool   `json:"ok"`
	ID  string `json:"id"`
	Rev string `json:"rev"`
}

// ---- lógica de hash ----

// ChunkID calcula o ID de um chunk no formato esperado pelo LiveSync:
// "h:" + xxhash64(content) em base36. É exportado pra facilitar testes.
func ChunkID(content string) string {
	h := xxhash.Sum64String(content)
	return "h:" + strconv.FormatUint(h, 36)
}

// ---- operações HTTP no CouchDB ----

// dbURL retorna a URL base do banco (ex: "http://host:5984/obsidian-livesync").
func (w *LiveSyncWriter) dbURL() string {
	return strings.TrimRight(w.cfg.URL, "/") + "/" + w.cfg.Database
}

// docURL retorna a URL de um documento específico no banco.
func (w *LiveSyncWriter) docURL(docID string) string {
	return w.dbURL() + "/" + url.PathEscape(docID)
}

// doRequest executa uma requisição HTTP autenticada no CouchDB.
func (w *LiveSyncWriter) doRequest(method, url string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(w.cfg.Username, w.cfg.Password)
	req.Header.Set("Content-Type", "application/json")
	return w.client.Do(req)
}

// getDoc busca um documento do CouchDB. Retorna nil, nil se não existir (404).
func (w *LiveSyncWriter) getDoc(docID string, target interface{}) error {
	resp, err := w.doRequest("GET", w.docURL(docID), nil)
	if err != nil {
		return fmt.Errorf("GET %s: %w", docID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil // doc não existe, target fica com zero-value
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("GET %s: status %d — %s", docID, resp.StatusCode, body)
	}

	return json.NewDecoder(resp.Body).Decode(target)
}

// putDoc cria ou atualiza um documento no CouchDB.
func (w *LiveSyncWriter) putDoc(doc interface{}) (*couchResponse, error) {
	data, err := json.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("marshal doc: %w", err)
	}

	// Extrair _id do JSON pra montar a URL
	var peek struct {
		ID string `json:"_id"`
	}
	if err := json.Unmarshal(data, &peek); err != nil {
		return nil, fmt.Errorf("extrair _id: %w", err)
	}

	resp, err := w.doRequest("PUT", w.docURL(peek.ID), bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("PUT %s: %w", peek.ID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("PUT %s: status %d — %s", peek.ID, resp.StatusCode, body)
	}

	var result couchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decodificar resposta PUT: %w", err)
	}
	return &result, nil
}

// Ping verifica se o banco de dados está acessível.
func (w *LiveSyncWriter) Ping() error {
	resp, err := w.doRequest("GET", w.dbURL(), nil)
	if err != nil {
		return fmt.Errorf("ping CouchDB: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ping CouchDB: status %d — %s", resp.StatusCode, body)
	}
	return nil
}

// ---- API pública ----

// WriteFile escreve (ou atualiza) um arquivo no CouchDB no formato
// LiveSync. Cria o leaf (chunk de conteúdo) e o documento de metadados.
//
// O docPath é o caminho relativo dentro do vault do Obsidian, exatamente
// como o LiveSync espera (ex: "Saude/Loviin/2026-09.md").
func (w *LiveSyncWriter) WriteFile(docPath, content string) error {
	now := time.Now().UnixMilli()

	// 1. Gerar ID do chunk com xxhash64 + base36
	leafID := ChunkID(content)

	// 2. Verificar se o leaf já existe (content-addressable: se o conteúdo
	//    é idêntico, o hash será o mesmo e não precisa reescrever)
	var existingLeaf liveSyncLeaf
	if err := w.getDoc(leafID, &existingLeaf); err != nil {
		return fmt.Errorf("verificar leaf existente: %w", err)
	}

	if existingLeaf.ID == "" {
		// Leaf não existe, criar
		leaf := liveSyncLeaf{
			ID:   leafID,
			Type: "leaf",
			Data: content,
		}
		if _, err := w.putDoc(&leaf); err != nil {
			return fmt.Errorf("criar leaf %s: %w", leafID, err)
		}
	}

	// 3. Verificar se já existe um documento de metadados (precisamos do
	//    _rev pra atualizar sem conflito)
	var existingMeta liveSyncMeta
	if err := w.getDoc(docPath, &existingMeta); err != nil {
		return fmt.Errorf("verificar meta existente: %w", err)
	}

	ctime := now
	if existingMeta.ID != "" {
		// Preservar ctime original em atualizações
		ctime = existingMeta.CTime
	}

	// 4. Criar/atualizar o documento de metadados
	meta := liveSyncMeta{
		ID:       docPath,
		Type:     "plain",
		Path:     docPath,
		Children: []string{leafID},
		CTime:    ctime,
		MTime:    now,
		Size:     len(content),
		Eden:     map[string]interface{}{},
	}

	if existingMeta.Rev != "" {
		meta.Rev = existingMeta.Rev
	}

	if _, err := w.putDoc(&meta); err != nil {
		return fmt.Errorf("criar/atualizar meta %s: %w", docPath, err)
	}

	return nil
}

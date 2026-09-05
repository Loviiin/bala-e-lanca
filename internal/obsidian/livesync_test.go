package obsidian

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestChunkIDUsesXXHash64Base36 verifica que o ChunkID gera o hash no
// formato que o LiveSync espera: xxhash64 codificado em base36.
//
// O hash de "hello" com xxhash64 é 0x26c7827d889f6da3 (decimal
// 2794345569481354659), que em base36 é "kzm0fsb93sif".
func TestChunkIDUsesXXHash64Base36(t *testing.T) {
	id := ChunkID("hello")

	if !strings.HasPrefix(id, "h:") {
		t.Fatalf("ChunkID deve começar com 'h:', got %q", id)
	}

	hash := strings.TrimPrefix(id, "h:")

	// xxhash64 em base36 gera no máximo ~13 caracteres (2^64-1 em base36
	// = "3w5e11264sgsf"), nunca 40 como o SHA-256 truncado que estava
	// sendo usado antes.
	if len(hash) > 13 {
		t.Fatalf("hash deveria ter no máximo 13 chars (base36), got %d chars: %q", len(hash), hash)
	}

	// Verificar que todos os caracteres são base36 válidos (0-9, a-z)
	for _, c := range hash {
		if !(c >= '0' && c <= '9') && !(c >= 'a' && c <= 'z') {
			t.Fatalf("caractere inválido em base36: %c (hash: %q)", c, hash)
		}
	}
}

// TestChunkIDIsDeterministic verifica que o mesmo conteúdo sempre gera
// o mesmo chunk ID (content-addressable).
func TestChunkIDIsDeterministic(t *testing.T) {
	a := ChunkID("conteúdo de teste")
	b := ChunkID("conteúdo de teste")

	if a != b {
		t.Fatalf("ChunkID não é determinístico: %q != %q", a, b)
	}
}

// TestChunkIDDifferentContentDifferentHash verifica que conteúdos
// diferentes geram hashes diferentes.
func TestChunkIDDifferentContentDifferentHash(t *testing.T) {
	a := ChunkID("foo")
	b := ChunkID("bar")

	if a == b {
		t.Fatalf("conteúdos diferentes geraram o mesmo hash: %q", a)
	}
}

// TestWriteFileCreatesLeafAndMeta verifica que WriteFile cria os dois
// documentos no CouchDB: o leaf (chunk) e o meta (arquivo).
func TestWriteFileCreatesLeafAndMeta(t *testing.T) {
	docs := make(map[string]json.RawMessage)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extrair doc ID do path (remover o prefixo "/testdb/")
		docID := strings.TrimPrefix(r.URL.Path, "/testdb/")

		switch r.Method {
		case "GET":
			if r.URL.Path == "/testdb" {
				// Ping
				json.NewEncoder(w).Encode(map[string]string{"db_name": "testdb"})
				return
			}
			if raw, ok := docs[docID]; ok {
				w.Write(raw)
			} else {
				w.WriteHeader(http.StatusNotFound)
				json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
			}

		case "PUT":
			var body json.RawMessage
			json.NewDecoder(r.Body).Decode(&body)

			// Injetar um _rev falso na resposta (CouchDB sempre retorna um)
			rev := "1-abc123"
			docs[docID] = body

			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"ok":  true,
				"id":  docID,
				"rev": rev,
			})
		}
	}))
	defer server.Close()

	writer := NewLiveSyncWriter(LiveSyncConfig{
		URL:      server.URL,
		Database: "testdb",
		Username: "admin",
		Password: "secret",
	})

	content := "| data | peso (kg) |\n|---|---|\n| 2026-09-05 | 82.40 |\n"
	docPath := "Saude/Loviin/2026-09.md"

	if err := writer.WriteFile(docPath, content); err != nil {
		t.Fatalf("WriteFile falhou: %v", err)
	}

	expectedLeafID := ChunkID(content)

	// Verificar que o leaf foi criado
	leafRaw, ok := docs[expectedLeafID]
	if !ok {
		t.Fatalf("leaf %s não foi criado no CouchDB", expectedLeafID)
	}

	var leaf liveSyncLeaf
	json.Unmarshal(leafRaw, &leaf)

	if leaf.Type != "leaf" {
		t.Errorf("leaf.type = %q, esperado %q", leaf.Type, "leaf")
	}
	if leaf.Data != content {
		t.Errorf("leaf.data não bate com o conteúdo original")
	}

	// Verificar que o meta foi criado
	metaRaw, ok := docs[docPath]
	if !ok {
		t.Fatalf("meta doc %s não foi criado no CouchDB", docPath)
	}

	var meta liveSyncMeta
	json.Unmarshal(metaRaw, &meta)

	if meta.Type != "plain" {
		t.Errorf("meta.type = %q, esperado %q", meta.Type, "plain")
	}
	if meta.Path != docPath {
		t.Errorf("meta.path = %q, esperado %q", meta.Path, docPath)
	}
	if len(meta.Children) != 1 || meta.Children[0] != expectedLeafID {
		t.Errorf("meta.children = %v, esperado [%s]", meta.Children, expectedLeafID)
	}
	if meta.Size != len(content) {
		t.Errorf("meta.size = %d, esperado %d", meta.Size, len(content))
	}
}

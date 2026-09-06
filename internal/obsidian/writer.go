// Package obsidian escreve as leituras no vault, um arquivo por mês por
// pessoa, em formato de tabela Markdown que o Dataview/Tracker conseguem
// ler. A escrita é atômica (escreve em tmp + rename) pra nunca corromper
// o arquivo se o Obsidian estiver com ele aberto ou sincronizando.
package obsidian

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Loviiin/okok-scale-logger/internal/bia"
)

type Writer struct {
	VaultDir string // ex: /data/vault (montado do host)
}

func NewWriter(vaultDir string) *Writer {
	return &Writer{VaultDir: vaultDir}
}

// PathFor monta o caminho tipo "<vault>/<pessoa>/2026-09.md" — um arquivo
// por pessoa por mês, evitando um único arquivo gigante que deixa o
// Dataview lento com o tempo.
func (w *Writer) PathFor(personName string, t time.Time) string {
	dir := filepath.Join(w.VaultDir, sanitize(personName))
	filename := t.Format("2006-01") + ".md"
	return filepath.Join(dir, filename)
}

func sanitize(name string) string {
	return strings.ReplaceAll(name, "/", "-")
}

func getHeader(personName string, t time.Time) string {
	return fmt.Sprintf(`---
type: pesagens
person: %s
month: %s
---
# Leituras de %s - %s

| data | peso (kg) | imc | %%gordura | %%água | massa magra (kg) | músculo esquelético (kg) | bmr (kcal) |
|---|---|---|---|---|---|---|---|
`, personName, t.Format("2006-01"), personName, t.Format("2006-01"))
}

// AppendReading adiciona uma linha na tabela do mês da pessoa, criando o
// arquivo (com header e frontmatter) se ainda não existir.
func (w *Writer) AppendReading(personName string, t time.Time, weightKg float64, m bia.Metrics, hasImpedance bool) error {
	path := w.PathFor(personName, t)

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("criar diretório: %w", err)
	}

	existing, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("ler arquivo existente: %w", err)
		}
		existing = []byte(getHeader(personName, t))
	}

	var row string
	if hasImpedance {
		row = fmt.Sprintf(
			"| %s | %.2f | %.1f | %.1f | %.1f | %.2f | %.2f | %.0f |\n",
			t.Format("2006-01-02 15:04"),
			weightKg,
			m.BMI,
			m.BodyFatPercent,
			m.BodyWaterPercent,
			m.FatFreeMassKg,
			m.SkeletalMuscleKg,
			m.BMRKcal,
		)
	} else {
		// Sem impedância, as estimativas de composição quebram (geram números absurdos
		// ou negativos). Mostra só o peso e métricas baseadas apenas em altura/peso.
		row = fmt.Sprintf(
			"| %s | %.2f | %.1f | - | - | - | - | %.0f |\n",
			t.Format("2006-01-02 15:04"),
			weightKg,
			m.BMI,
			m.BMRKcal,
		)
	}

	newContent := append(existing, []byte(row)...)

	return writeAtomic(path, newContent)
}

// writeAtomic escreve em um arquivo temporário no MESMO diretório (pra
// garantir que o rename seja atômico dentro do mesmo filesystem) e só
// depois substitui o arquivo final. Isso evita que o Obsidian veja um
// arquivo pela metade se ler no meio da escrita.
func writeAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return fmt.Errorf("criar arquivo temporário: %w", err)
	}
	tmpPath := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("escrever arquivo temporário: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("fechar arquivo temporário: %w", err)
	}
	if err := os.Chmod(tmpPath, 0o644); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("ajustar permissões do arquivo temporário: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("renomear arquivo temporário: %w", err)
	}

	return nil
}

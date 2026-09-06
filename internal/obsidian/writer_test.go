package obsidian

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Loviiin/okok-scale-logger/internal/bia"
)

func TestAppendReadingWritesAtomicallyAndAppends(t *testing.T) {
	vaultDir := t.TempDir()
	writer := NewWriter(vaultDir)
	firstTime := time.Date(2026, time.September, 5, 10, 30, 0, 0, time.UTC)
	secondTime := firstTime.Add(10 * time.Minute)
	metrics := bia.Metrics{BMI: 24.5, BodyFatPercent: 18.2, BodyWaterPercent: 58.1, FatFreeMassKg: 67.3, SkeletalMuscleKg: 31.4, BMRKcal: 1680}

	if err := writer.AppendReading("Pessoa/Teste", firstTime, 82.4, metrics, true); err != nil {
		t.Fatalf("primeira gravação: %v", err)
	}
	if err := writer.AppendReading("Pessoa/Teste", secondTime, 82.1, metrics, true); err != nil {
		t.Fatalf("segunda gravação: %v", err)
	}

	path := filepath.Join(vaultDir, "Pessoa-Teste", "2026-09.md")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ler arquivo final: %v", err)
	}

	if got := strings.Count(string(content), "| data | peso"); got != 1 {
		t.Fatalf("cabeçalho apareceu %d vezes, esperado 1", got)
	}
	if got := strings.Count(string(content), "| 2026-09-05"); got != 2 {
		t.Fatalf("arquivo contém %d leituras, esperado 2", got)
	}
	if !strings.Contains(string(content), "| 2026-09-05 10:30 | 82.40 |") {
		t.Errorf("primeira leitura não encontrada no conteúdo final:\n%s", content)
	}
	if !strings.Contains(string(content), "| 2026-09-05 10:40 | 82.10 |") {
		t.Errorf("segunda leitura não encontrada no conteúdo final:\n%s", content)
	}

	tmpFiles, err := filepath.Glob(filepath.Join(vaultDir, "Pessoa-Teste", ".tmp-*"))
	if err != nil {
		t.Fatalf("procurar temporários: %v", err)
	}
	if len(tmpFiles) != 0 {
		t.Fatalf("sobraram arquivos temporários: %v", tmpFiles)
	}
}

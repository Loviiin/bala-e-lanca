// Package profile guarda os perfis das pessoas da casa e decide, quando
// uma pesagem estabiliza, de quem provavelmente é aquele peso.
package profile

import (
	"fmt"
	"math"
	"os"

	"gopkg.in/yaml.v3"
)

// Sex é usado nas fórmulas de composição corporal (Deurenberg, Janssen,
// Mifflin-St Jeor todas diferenciam por sexo biológico).
type Sex string

const (
	Male   Sex = "M"
	Female Sex = "F"
)

// Profile representa uma pessoa da casa.
type Profile struct {
	Name   string  `yaml:"name"`
	Sex    Sex     `yaml:"sex"`
	Age    int     `yaml:"age"`
	Height float64 `yaml:"height_cm"`

	// ExpectedWeightKg é usado só pra identificação automática — é a
	// última leitura estável dessa pessoa, e vai sendo atualizado a
	// cada pesagem bem-sucedida.
	ExpectedWeightKg float64 `yaml:"expected_weight_kg"`

	// ToleranceKg é a margem pra bater com ExpectedWeightKg. Se a leitura
	// cair fora da tolerância de TODOS os perfis, vira "não identificado".
	ToleranceKg float64 `yaml:"tolerance_kg"`

	// Campos opcionais (nil = não preenchido) — só entram nas métricas
	// tier 2 (RCQ, risco cardiovascular). Não vêm da balança, são
	// medidos manualmente com fita métrica de vez em quando.
	WaistCm *float64 `yaml:"waist_cm,omitempty"`
	HipCm   *float64 `yaml:"hip_cm,omitempty"`
}

type Config struct {
	Profiles []Profile `yaml:"profiles"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("ler config %s: %w", path, err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsear yaml: %w", err)
	}
	return &cfg, nil
}

// Identify tenta achar de qual perfil é o peso informado, pegando o mais
// próximo dentro da tolerância de cada um. Retorna nil se nenhum perfil
// bater — nesse caso, grave a leitura como "não identificada" em vez de
// arriscar atribuir a pessoa errada.
func Identify(profiles []Profile, weightKg float64) *Profile {
	var best *Profile
	bestDiff := math.Inf(1)

	for i := range profiles {
		p := &profiles[i]
		diff := math.Abs(p.ExpectedWeightKg - weightKg)
		tolerance := p.ToleranceKg
		if tolerance == 0 {
			tolerance = 4.0 // default razoável se não configurado
		}
		if diff <= tolerance && diff < bestDiff {
			best = p
			bestDiff = diff
		}
	}

	return best
}

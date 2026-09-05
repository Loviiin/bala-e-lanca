package main

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/Loviiin/okok-scale-logger/internal/bia"
	"github.com/Loviiin/okok-scale-logger/internal/ble"
	"github.com/Loviiin/okok-scale-logger/internal/obsidian"
	"github.com/Loviiin/okok-scale-logger/internal/profile"
)

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	configPath := envOr("CONFIG_PATH", "/data/config.yaml")
	vaultDir := envOr("VAULT_DIR", "/data/vault")
	scaleMAC := os.Getenv("SCALE_MAC")
	stabilityCount, _ := strconv.Atoi(envOr("STABILITY_COUNT", "3"))

	if scaleMAC == "" {
		log.Fatal("variável SCALE_MAC não configurada (ex: A8:0B:6B:77:98:C7)")
	}

	cfg, err := profile.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("carregar config: %v", err)
	}

	writer := obsidian.NewWriter(vaultDir)
	tracker := newStabilityTracker(stabilityCount)

	scanner := ble.NewScanner(scaleMAC)

	err = scanner.Start(func(r ble.Reading) {
		stable, isNew := tracker.observe(r)
		if !stable || !isNew {
			return
		}

		log.Printf("leitura estável: %.2f kg, impedância %.1f ohm (tem impedância: %v)",
			r.WeightKg, r.ImpedanceOhm, r.HasImpedance)

		p := profile.Identify(cfg.Profiles, r.WeightKg)
		if p == nil {
			log.Printf("ATENÇÃO: nenhum perfil bateu com %.2f kg — gravando como 'nao-identificado'", r.WeightKg)
			p = &profile.Profile{Name: "nao-identificado", Height: 170, Age: 30, Sex: profile.Male}
		}

		impedance := r.ImpedanceOhm
		if !r.HasImpedance {
			impedance = 0
		}

		metrics := bia.Calculate(bia.Input{
			WeightKg:     r.WeightKg,
			ImpedanceOhm: impedance,
			Profile:      *p,
		})

		if err := writer.AppendReading(p.Name, time.Now(), r.WeightKg, metrics); err != nil {
			log.Printf("ERRO ao gravar no vault: %v", err)
		}
	})
	if err != nil {
		log.Fatalf("erro no scanner: %v", err)
	}
}

// stabilityTracker decide quando uma leitura "travou" (peso repetiu N
// vezes seguidas) e evita gravar a mesma pesagem várias vezes enquanto a
// pessoa ainda está em cima da balança.
type stabilityTracker struct {
	requiredCount int
	lastWeight    float64
	repeatCount   int
	alreadyLogged bool
	lastLoggedAt  time.Time
}

const minimumReadingInterval = time.Minute

func newStabilityTracker(requiredCount int) *stabilityTracker {
	if requiredCount < 1 {
		requiredCount = 3
	}
	return &stabilityTracker{requiredCount: requiredCount}
}

// observe retorna (estável, é-uma-leitura-nova-pra-gravar).
func (s *stabilityTracker) observe(r ble.Reading) (stable bool, isNew bool) {
	// Balança voltou pro idle (ninguém em cima) — libera pra gravar de
	// novo na próxima vez que alguém subir.
	if r.WeightKg == 0 {
		s.lastWeight = 0
		s.repeatCount = 0
		s.alreadyLogged = false
		return false, false
	}

	if r.WeightKg == s.lastWeight {
		s.repeatCount++
	} else {
		s.lastWeight = r.WeightKg
		s.repeatCount = 1
		s.alreadyLogged = false
	}

	isStable := s.repeatCount >= s.requiredCount
	if isStable && !s.alreadyLogged {
		if !s.lastLoggedAt.IsZero() && time.Since(s.lastLoggedAt) < minimumReadingInterval {
			return true, false
		}
		s.alreadyLogged = true
		s.lastLoggedAt = time.Now()
		return true, true
	}
	return isStable, false
}

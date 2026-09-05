package bia

import (
	"testing"

	"github.com/Loviiin/okok-scale-logger/internal/profile"
)

// Teste de sanidade, não de precisão clínica: garante que as fórmulas não
// devolvem números absurdos (negativos, >100%, etc) pra um caso realista.
//
// Ajuste os "wantAprox" comparando com o valor que o app oficial da OKOK
// mostrar pra você no mesmo dia, se quiser calibrar.
func TestCalculate_CasoRealista(t *testing.T) {
	p := profile.Profile{
		Name:   "Teste",
		Sex:    profile.Male,
		Age:    30,
		Height: 175, // cm
	}

	m := Calculate(Input{
		WeightKg:     99.40,
		ImpedanceOhm: 500.0,
		Profile:      p,
	})

	if m.BMI <= 0 || m.BMI > 80 {
		t.Errorf("BMI fora de uma faixa plausível: %.2f", m.BMI)
	}
	if m.BodyFatPercent <= 0 || m.BodyFatPercent > 70 {
		t.Errorf("BodyFatPercent fora de uma faixa plausível: %.2f", m.BodyFatPercent)
	}
	if m.FatFreeMassKg <= 0 || m.FatFreeMassKg >= 99.40 {
		t.Errorf("FatFreeMassKg deveria ser positivo e menor que o peso total: %.2f", m.FatFreeMassKg)
	}
	if m.BodyWaterPercent <= 20 || m.BodyWaterPercent >= 80 {
		t.Errorf("BodyWaterPercent fora de uma faixa humana plausível: %.2f", m.BodyWaterPercent)
	}
	if m.SkeletalMuscleKg <= 0 {
		t.Errorf("SkeletalMuscleKg deveria ser positivo quando há impedância: %.2f", m.SkeletalMuscleKg)
	}
	if m.BMRKcal < 500 || m.BMRKcal > 5000 {
		t.Errorf("BMRKcal fora de uma faixa plausível: %.2f", m.BMRKcal)
	}
}

func TestCalculate_SemImpedancia(t *testing.T) {
	p := profile.Profile{Sex: profile.Female, Age: 28, Height: 165}
	m := Calculate(Input{WeightKg: 60, ImpedanceOhm: 0, Profile: p})

	if m.SkeletalMuscleKg != 0 {
		t.Errorf("sem impedância, SkeletalMuscleKg deveria ficar 0 (não calculável), veio %.2f", m.SkeletalMuscleKg)
	}
	// BMI, %gordura, BMR não dependem de impedância — continuam válidos.
	if m.BMI <= 0 {
		t.Errorf("BMI deveria continuar sendo calculado mesmo sem impedância")
	}
}

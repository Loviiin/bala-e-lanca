// Package bia implementa as fórmulas de composição corporal (BIA =
// Bioelectrical Impedance Analysis) a partir de peso + impedância + perfil
// da pessoa (altura, sexo, idade).
//
// IMPORTANTE: o algoritmo exato que a Chipsea/OKOK usa (CsAlgoBuilder) é
// proprietário e fechado — não temos acesso a ele. As fórmulas aqui são
// equações científicas PÚBLICAS e amplamente citadas na literatura de
// bioimpedância (Deurenberg, Janssen, Mifflin-St Jeor, Kyle, Kushner).
// Elas dão uma estimativa correta na faixa certa, mas não vão bater 100%
// número-a-número com o que o app oficial da OKOK mostra. Se quiser mais
// precisão, pese-se no app oficial e nesse sistema no mesmo dia e ajuste
// os fatores de calibração em Calculate.
package bia

import "github.com/Loviiin/okok-scale-logger/internal/profile"

// Input é tudo que as fórmulas precisam.
type Input struct {
	WeightKg     float64
	ImpedanceOhm float64
	Profile      profile.Profile
}

// Metrics é o pacote completo de métricas calculadas.
type Metrics struct {
	BMI float64 // kg/m²

	BodyFatPercent float64
	FatMassKg      float64

	BodyWaterPercent float64
	BodyWaterKg      float64

	MuscleMassKg     float64
	SkeletalMuscleKg float64
	FatFreeMassKg    float64

	BMRKcal float64

	// Tier 2 — só preenchidos se WaistCm/HipCm estiverem configurados
	// no perfil (ponteiro != nil). Ficam zerados caso contrário.
	WaistToHipRatio    *float64
	WaistToHeightRatio *float64
}

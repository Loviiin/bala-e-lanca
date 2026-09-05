package bia

import "github.com/Loviiin/okok-scale-logger/internal/profile"

func sexFactor(s profile.Sex) float64 {
	if s == profile.Male {
		return 1
	}
	return 0
}

// bmi = peso / altura² (altura em metros)
func bmi(weightKg, heightCm float64) float64 {
	heightM := heightCm / 100.0
	return weightKg / (heightM * heightM)
}

// bodyFatPercent usa a fórmula de Deurenberg (1991), uma das mais citadas
// pra estimar %gordura a partir de IMC + idade + sexo. É a mesma base que
// o bioscale-okok-alternative usa como fallback.
//
// Não usa a impedância diretamente — é só IMC-based. Fica menos precisa
// pra atletas (subestima massa magra alta), mas é robusta e não depende
// de coeficientes proprietários que não temos.
func bodyFatPercent(weightKg, heightCm float64, age int, sex profile.Sex) float64 {
	bmiVal := bmi(weightKg, heightCm)
	return 1.20*bmiVal + 0.23*float64(age) - 10.8*sexFactor(sex) - 5.4
}

// skeletalMuscleKg usa a fórmula de Janssen et al. 2000 (J Appl Physiol),
// validada contra ressonância magnética. Essa SIM usa a impedância
// (resistência) diretamente — é o motivo de a balança medir isso.
func skeletalMuscleKg(heightCm, impedanceOhm float64, age int, sex profile.Sex) float64 {
	if impedanceOhm <= 0 {
		return 0 // sem impedância medida (sem contato nos eletrodos), não dá pra calcular
	}
	return ((heightCm*heightCm/impedanceOhm)*0.401 + sexFactor(sex)*3.825 + float64(age)*-0.071) + 5.102
}

// bmrKcal usa Mifflin-St Jeor, o padrão-ouro atual pra taxa metabólica
// basal (mais preciso que a antiga Harris-Benedict).
func bmrKcal(weightKg, heightCm float64, age int, sex profile.Sex) float64 {
	base := 10*weightKg + 6.25*heightCm - 5*float64(age)
	if sex == profile.Male {
		return base + 5
	}
	return base - 161
}

// impedanceIndex (H²/R) é o preditor clássico de Kyle et al. 2004 pra
// massa livre de gordura. Aqui só devolvemos o índice cru; se quiser
// evoluir pra regressão completa de Kyle, precisa também da reatância
// (Xc), que essa balança de eletrodo único não mede — só a resistência.
func impedanceIndex(heightCm, impedanceOhm float64) float64 {
	if impedanceOhm <= 0 {
		return 0
	}
	return (heightCm * heightCm) / impedanceOhm
}

// Calculate roda todas as fórmulas e monta o pacote de métricas.
func Calculate(in Input) Metrics {
	p := in.Profile
	weight := in.WeightKg
	height := p.Height

	bfPercent := bodyFatPercent(weight, height, p.Age, p.Sex)
	fatMass := weight * bfPercent / 100.0
	ffm := weight - fatMass // massa livre de gordura, por consistência com o BF% acima

	// Água corporal: Pace & Rathbun (1945) — ~73% da massa livre de
	// gordura. Simples e citado como referência padrão (é a mesma que
	// o bioscale-okok-alternative usa).
	waterKg := 0.73 * ffm
	waterPercent := 0.0
	if weight > 0 {
		waterPercent = (waterKg / weight) * 100.0
	}

	smm := skeletalMuscleKg(height, in.ImpedanceOhm, p.Age, p.Sex)

	m := Metrics{
		BMI:              bmi(weight, height),
		BodyFatPercent:   bfPercent,
		FatMassKg:        fatMass,
		BodyWaterPercent: waterPercent,
		BodyWaterKg:      waterKg,
		SkeletalMuscleKg: smm,
		// Estimativa grosseira: músculo esquelético costuma ser ~75%
		// da massa muscular total (o resto é liso + cardíaco). Ajuste
		// esse fator se algum dia comparar com um DEXA de verdade.
		MuscleMassKg:  smm / 0.75,
		FatFreeMassKg: ffm,
		BMRKcal:       bmrKcal(weight, height, p.Age, p.Sex),
	}

	if p.WaistCm != nil && p.HipCm != nil && *p.HipCm > 0 {
		ratio := *p.WaistCm / *p.HipCm
		m.WaistToHipRatio = &ratio
	}
	if p.WaistCm != nil && height > 0 {
		ratio := *p.WaistCm / height
		m.WaistToHeightRatio = &ratio
	}

	return m
}

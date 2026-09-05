package ble

import "testing"

// Casos de teste com os bytes REAIS capturados no vídeo de reverse
// engineering (nRF Connect, campo RAW). Servem como regressão: se algum
// dia você mexer no decoder, esses valores não podem mudar.
func TestDecode_ValoresReaisCapturados(t *testing.T) {
	cases := []struct {
		name         string
		payload      []byte
		wantWeight   float64
		wantImpedanc float64
		wantHasImp   bool
	}{
		{
			name:         "idle - ninguém na balança",
			payload:      []byte{0x00, 0x00, 0x17, 0x70, 0x0A, 0x01, 0x24, 0xA8, 0x0B, 0x6B, 0x77, 0x98, 0xC7},
			wantWeight:   0.0,
			wantImpedanc: 600.0,
			wantHasImp:   false,
		},
		{
			name:         "1a pesagem travada (99.40kg, bom contato)",
			payload:      []byte{0x26, 0xD4, 0x17, 0x70, 0x0A, 0x01, 0x25, 0xA8, 0x0B, 0x6B, 0x77, 0x98, 0xC7},
			wantWeight:   99.40,
			wantImpedanc: 600.0,
			wantHasImp:   false,
		},
		{
			name:         "2a pesagem travada (53.30kg, bom contato)",
			payload:      []byte{0x14, 0xD2, 0x17, 0x70, 0x0A, 0x01, 0x24, 0xA8, 0x0B, 0x6B, 0x77, 0x98, 0xC7},
			wantWeight:   53.30,
			wantImpedanc: 600.0,
			wantHasImp:   false,
		},
		{
			name:         "3a pesagem travada (42.10kg, SEM contato nos eletrodos)",
			payload:      []byte{0x10, 0x72, 0x00, 0x00, 0x0A, 0x01, 0x24, 0xA8, 0x0B, 0x6B, 0x77, 0x98, 0xC7},
			wantWeight:   42.10,
			wantImpedanc: 0.0,
			wantHasImp:   false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Decode(tc.payload)
			if err != nil {
				t.Fatalf("Decode retornou erro inesperado: %v", err)
			}
			if got.WeightKg != tc.wantWeight {
				t.Errorf("peso = %.2f, esperado %.2f", got.WeightKg, tc.wantWeight)
			}
			if got.ImpedanceOhm != tc.wantImpedanc {
				t.Errorf("impedância = %.1f, esperado %.1f", got.ImpedanceOhm, tc.wantImpedanc)
			}
			if got.HasImpedance != tc.wantHasImp {
				t.Errorf("hasImpedance = %v, esperado %v", got.HasImpedance, tc.wantHasImp)
			}
		})
	}
}

func TestDecode_TamanhoInvalido(t *testing.T) {
	_, err := Decode([]byte{0x01, 0x02})
	if err == nil {
		t.Fatal("esperava erro para payload curto demais, não retornou nenhum")
	}
}

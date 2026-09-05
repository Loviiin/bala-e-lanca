// Package ble cuida de escanear e decodificar os advertisements BLE da
// balança OKOK/Chipsea.
//
// Layout confirmado empiricamente (ver conversa de reverse engineering):
// o "ManufacturerData.Data" que a lib de bluetooth entrega já vem SEM os
// 2 bytes de "company ID" (esses 2 bytes são, na real, lixo/contador da
// própria balança, não um Company ID de verdade — por isso não são usados
// aqui pra nada além de já terem sido descartados pela lib).
//
// payload (13 bytes):
//
//	[0:2]  peso bruto, big-endian, kg * 100
//	[2:4]  impedância bruta, big-endian, ohms * 10 (0x0000 = não medida)
//	[4:6]  constante fixa de protocolo (0x0A, 0x01) — ignorado
//	[6]    contador/sequência — ignorado por enquanto
//	[7:13] MAC do próprio dispositivo, espelhado — ignorado
package ble

import (
	"encoding/binary"
	"errors"
)

// PayloadLen é o tamanho esperado do payload após remover os 2 bytes
// de "company id" falso. Se a sua balança usar uma revisão de firmware
// diferente, esse número (e os offsets abaixo) podem mudar — foi assim
// que descobrimos que a formatação "OKOK 2.0" documentada em outros
// projetos (19 bytes) não bate com essa balança (13 bytes).
const PayloadLen = 13

var ErrUnexpectedLength = errors.New("ble: payload com tamanho inesperado, não é um pacote OKOK reconhecido")

// Reading é uma leitura decodificada de um advertisement da balança.
type Reading struct {
	WeightKg     float64
	ImpedanceOhm float64
	HasImpedance bool
}

// Decode extrai peso e impedância de um payload de manufacturer data.
//
// weight = payload[0:2] / 100.0
// impedance = payload[2:4] / 10.0 (0 quando a balança não completou a
// medição de bioimpedância — normalmente por falta de contato nos
// eletrodos, ou porque a leitura ainda está instável)
func Decode(payload []byte) (Reading, error) {
	if len(payload) != PayloadLen {
		return Reading{}, ErrUnexpectedLength
	}

	weightRaw := binary.BigEndian.Uint16(payload[0:2])
	impedanceRaw := binary.BigEndian.Uint16(payload[2:4])

	return Reading{
		WeightKg:     float64(weightRaw) / 100.0,
		ImpedanceOhm: float64(impedanceRaw) / 10.0,
		HasImpedance: impedanceRaw != 0,
	}, nil
}

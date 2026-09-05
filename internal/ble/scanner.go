package ble

import (
	"fmt"
	"log"
	"sync"

	"tinygo.org/x/bluetooth"
)

// Scanner escaneia advertisements BLE e filtra só os da balança configurada.
type Scanner struct {
	adapter       *bluetooth.Adapter
	targetAddr    string // MAC da balança, ex: "A8:0B:6B:77:98:C7"
	duplicateOnce sync.Once
}

func NewScanner(targetAddr string) *Scanner {
	return &Scanner{
		adapter:    bluetooth.DefaultAdapter,
		targetAddr: targetAddr,
	}
}

// Start começa a escanear e chama onReading toda vez que decodifica um
// advertisement válido da balança configurada. É bloqueante — rode numa
// goroutine se precisar continuar fazendo outra coisa no main.
func (s *Scanner) Start(onReading func(Reading)) error {
	if err := s.adapter.Enable(); err != nil {
		return fmt.Errorf("habilitar stack BLE: %w", err)
	}

	log.Printf("escaneando, filtrando por %s...", s.targetAddr)

	return s.adapter.Scan(func(adapter *bluetooth.Adapter, device bluetooth.ScanResult) {
		s.duplicateOnce.Do(func() {
			if err := enableDuplicateData(); err != nil {
				log.Printf("não foi possível habilitar anúncios duplicados: %v", err)
			}
		})

		if device.Address.String() != s.targetAddr {
			return
		}

		for _, md := range device.AdvertisementPayload.ManufacturerData() {
			// Não filtramos pelo CompanyID aqui de propósito: nessa
			// balança esse campo não é um Company ID de verdade, é
			// lixo/contador que muda a cada pacote (ver decode.go).
			reading, err := Decode(md.Data)
			if err != nil {
				// Payload de tamanho diferente do esperado — pode ser
				// outro AD structure que não é o nosso, ignora.
				continue
			}
			onReading(reading)
		}
	})
}

func (s *Scanner) Stop() error {
	return s.adapter.StopScan()
}

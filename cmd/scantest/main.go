// Smoke test do adaptador Bluetooth: escaneia dispositivos BLE próximos e,
// opcionalmente, decodifica os advertisements da balança configurada.
package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/Loviiin/okok-scale-logger/internal/ble"
	"tinygo.org/x/bluetooth"
)

func main() {
	targetMAC := flag.String("mac", "", "MAC da balança (opcional) para decodificar peso/impedância")
	duration := flag.Duration("duration", 30*time.Second, "tempo de escaneamento")
	flag.Parse()

	target := strings.ToUpper(strings.TrimSpace(*targetMAC))
	adapter := bluetooth.DefaultAdapter

	fmt.Println("habilitando o adaptador Bluetooth...")
	if err := adapter.Enable(); err != nil {
		log.Fatalf("FALHOU ao habilitar o Bluetooth: %v\n\nConfira se o Bluetooth está ligado no Windows e se nenhum outro programa está usando o adaptador.", err)
	}

	fmt.Println("Bluetooth habilitado com sucesso! Escaneando por", *duration, "...")
	fmt.Println("(qualquer dispositivo BLE por perto vai aparecer abaixo)")
	fmt.Println()

	seen := make(map[string]bool)
	var seenMu sync.Mutex

	scanErr := make(chan error, 1)
	go func() {
		scanErr <- adapter.Scan(func(adapter *bluetooth.Adapter, device bluetooth.ScanResult) {
			addr := strings.ToUpper(device.Address.String())

			seenMu.Lock()
			isNew := !seen[addr]
			seen[addr] = true
			seenMu.Unlock()

			if isNew {
				name := device.LocalName()
				if name == "" {
					name = "(sem nome)"
				}
				fmt.Printf("[novo device] %s  RSSI=%d  nome=%s\n", addr, device.RSSI, name)
			}

			if target == "" || addr != target {
				return
			}

			for _, md := range device.AdvertisementPayload.ManufacturerData() {
				fmt.Printf("  -> manufacturer data bruto: %s\n", hex.EncodeToString(md.Data))
				reading, err := ble.Decode(md.Data)
				if err != nil {
					fmt.Printf("  -> não decodificou como pacote OKOK conhecido: %v\n", err)
					continue
				}
				fmt.Printf("  -> PESO: %.2f kg | IMPEDÂNCIA: %.1f ohm (medida: %v)\n",
					reading.WeightKg, reading.ImpedanceOhm, reading.HasImpedance)
			}
		})
	}()

	timer := time.NewTimer(*duration)
	<-timer.C
	if err := adapter.StopScan(); err != nil {
		log.Printf("erro ao parar o scan: %v", err)
	}
	if err := <-scanErr; err != nil {
		log.Printf("erro durante o scan: %v", err)
	}

	seenMu.Lock()
	count := len(seen)
	seenMu.Unlock()

	fmt.Println("\nfim do teste.")
	fmt.Println("total de dispositivos BLE únicos encontrados:", count)
	if count == 0 {
		fmt.Println("\nNENHUM dispositivo encontrado — confira o adaptador e o driver do Bluetooth no Windows.")
	}
}

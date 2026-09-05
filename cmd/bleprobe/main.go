// bleprobe registra os payloads BLE da balança agrupados por conteúdo.
// Use uma rodada por condição: descalço, chinelo e diferentes contatos.
package main

import (
	"encoding/binary"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Loviiin/okok-scale-logger/internal/ble"
	"tinygo.org/x/bluetooth"
)

type packetStats struct {
	count        int
	weight       float64
	rawImpedance uint16
	sequence     byte
}

func main() {
	targetMAC := flag.String("mac", "A8:0B:6B:77:98:C7", "MAC da balança")
	label := flag.String("label", "sem-label", "rótulo desta rodada")
	duration := flag.Duration("duration", 30*time.Second, "tempo de coleta")
	flag.Parse()

	target := strings.ToUpper(strings.TrimSpace(*targetMAC))
	adapter := bluetooth.DefaultAdapter
	if err := adapter.Enable(); err != nil {
		log.Fatalf("habilitar Bluetooth: %v", err)
	}

	packets := make(map[string]*packetStats)
	var packetsMu sync.Mutex
	scanErr := make(chan error, 1)

	fmt.Printf("RODADA=%s DURACAO=%s MAC=%s\n", *label, *duration, target)
	fmt.Println("Colete uma condição por vez e mantenha a balança transmitindo.")

	go func() {
		scanErr <- adapter.Scan(func(_ *bluetooth.Adapter, device bluetooth.ScanResult) {
			if strings.ToUpper(device.Address.String()) != target {
				return
			}
			for _, manufacturer := range device.AdvertisementPayload.ManufacturerData() {
				payload := manufacturer.Data
				if len(payload) != ble.PayloadLen {
					continue
				}
				key := hex.EncodeToString(payload)
				weightRaw := binary.BigEndian.Uint16(payload[0:2])
				rawImpedance := binary.BigEndian.Uint16(payload[2:4])

				packetsMu.Lock()
				stats := packets[key]
				if stats == nil {
					stats = &packetStats{
						weight:       float64(weightRaw) / 100,
						rawImpedance: rawImpedance,
						sequence:     payload[6],
					}
					packets[key] = stats
					reading, _ := ble.Decode(payload)
					fmt.Printf("NOVO raw=%s peso=%.2f raw_imp=0x%04x imp=%.1f tem_imp=%t seq=0x%02x\n",
						key, reading.WeightKg, rawImpedance, reading.ImpedanceOhm, reading.HasImpedance, payload[6])
				}
				stats.count++
				packetsMu.Unlock()
			}
		})
	}()

	<-time.After(*duration)
	if err := adapter.StopScan(); err != nil {
		log.Printf("parar scan: %v", err)
	}
	if err := <-scanErr; err != nil {
		log.Printf("scan: %v", err)
	}

	keys := make([]string, 0, len(packets))
	for key := range packets {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	fmt.Printf("\nRESUMO rodada=%s pacotes_distintos=%d\n", *label, len(keys))
	for _, key := range keys {
		stats := packets[key]
		reading, _ := ble.Decode(mustDecodeHex(key))
		fmt.Printf("count=%d raw=%s peso=%.2f raw_imp=0x%04x imp=%.1f tem_imp=%t seq=0x%02x\n",
			stats.count, key, reading.WeightKg, stats.rawImpedance, reading.ImpedanceOhm, reading.HasImpedance, stats.sequence)
	}
}

func mustDecodeHex(value string) []byte {
	payload, err := hex.DecodeString(value)
	if err != nil {
		panic(err)
	}
	return payload
}

module github.com/Loviiin/okok-scale-logger

go 1.23.0

require (
	github.com/cespare/xxhash/v2 v2.3.0
	gopkg.in/yaml.v3 v3.0.1
	tinygo.org/x/bluetooth v0.16.0
)

require (
	github.com/go-ole/go-ole v1.3.0 // indirect
	github.com/godbus/dbus/v5 v5.2.2 // indirect
	github.com/saltosystems/winrt-go v0.0.0-20260513072510-45f10383b2b8 // indirect
	github.com/sirupsen/logrus v1.10.2 // indirect
	github.com/soypat/cyw43439 v0.1.2-0.20260731160358-f2a6af121857 // indirect
	github.com/soypat/lneto v0.3.2 // indirect
	github.com/soypat/seqs v0.0.0-20260125140838-2c1c6b1bd69e // indirect
	github.com/tinygo-org/cbgo v0.0.4 // indirect
	github.com/tinygo-org/pio v0.3.0 // indirect
	golang.org/x/exp v0.0.0-20260824195058-e88cd73687aa // indirect
	golang.org/x/sys v0.47.0 // indirect
	tinygo.org/x/espradio v0.3.0 // indirect
)

// Rode `go mod tidy` na sua máquina (com internet liberada pro proxy do Go)
// pra resolver as versões exatas e gerar o go.sum. Este ambiente de setup
// não tem acesso ao proxy.golang.org, então os arquivos aqui são o
// esqueleto de código, não um build já validado.

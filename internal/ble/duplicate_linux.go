//go:build linux

package ble

import (
	"fmt"
	"strings"

	"github.com/godbus/dbus/v5"
)

// enableDuplicateData asks BlueZ to emit advertisements even when their
// payload is unchanged. tinygo/bluetooth sets only Transport=le internally.
func enableDuplicateData() error {
	bus, err := dbus.SystemBus()
	if err != nil {
		return err
	}
	defer bus.Close()

	root := bus.Object("org.bluez", dbus.ObjectPath("/"))
	var managed map[dbus.ObjectPath]map[string]map[string]dbus.Variant
	if err := root.Call("org.freedesktop.DBus.ObjectManager.GetManagedObjects", 0).Store(&managed); err != nil {
		return err
	}

	var adapterPath dbus.ObjectPath
	for path, interfaces := range managed {
		if _, ok := interfaces["org.bluez.Adapter1"]; !ok {
			continue
		}
		if strings.HasSuffix(string(path), "/hci0") {
			adapterPath = path
			break
		}
		if adapterPath == "" {
			adapterPath = path
		}
	}
	if adapterPath == "" {
		return fmt.Errorf("nenhum adaptador BlueZ encontrado")
	}

	adapter := bus.Object("org.bluez", adapterPath)
	return adapter.Call("org.bluez.Adapter1.SetDiscoveryFilter", 0, map[string]interface{}{
		"Transport":     "le",
		"DuplicateData": true,
	}).Err
}

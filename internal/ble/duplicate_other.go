//go:build !linux

package ble

func enableDuplicateData() error {
	return nil
}

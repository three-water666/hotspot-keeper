//go:build windows
// +build windows

package main

import (
	"golang.org/x/sys/windows/registry"
	"os"
)

func enableAutoStart() {
	exePath, _ := os.Executable()
	k, _, _ := registry.CreateKey(registry.CURRENT_USER,
		`Software\Microsoft\Windows\CurrentVersion\Run`,
		registry.SET_VALUE)
	k.SetStringValue("HotspotKeeper", exePath)
	k.Close()
}

func disableAutoStart() {
	k, err := registry.OpenKey(registry.CURRENT_USER,
		`Software\Microsoft\Windows\CurrentVersion\Run`,
		registry.SET_VALUE)
	if err == nil {
		k.DeleteValue("HotspotKeeper")
		k.Close()
	}
}

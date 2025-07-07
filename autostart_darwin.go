//go:build darwin
// +build darwin

package main

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
)

func enableAutoStart() {
	usr, _ := user.Current()
	plistPath := filepath.Join(usr.HomeDir, "Library/LaunchAgents/com.hotspot.keeper.plist")
	exePath, _ := os.Executable()

	plist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
 "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key><string>com.hotspot.keeper</string>
	<key>ProgramArguments</key>
	<array><string>%s</string></array>
	<key>RunAtLoad</key><true/>
	<key>KeepAlive</key><false/>
</dict>
</plist>`, exePath)

	os.WriteFile(plistPath, []byte(plist), 0644)
}

func disableAutoStart() {
	usr, _ := user.Current()
	plistPath := filepath.Join(usr.HomeDir, "Library/LaunchAgents/com.hotspot.keeper.plist")
	os.Remove(plistPath)
}

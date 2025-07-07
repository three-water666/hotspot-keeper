//go:build windows
// +build windows

package main

import _ "embed"

//go:embed icons/green.ico
var iconRunning []byte

//go:embed icons/gray.ico
var iconStopped []byte

//go:embed icons/yellow.ico
var iconNoNet []byte

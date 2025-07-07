//go:build darwin
// +build darwin

package main

import _ "embed"

//go:embed icons/green.png
var iconRunning []byte

//go:embed icons/gray.png
var iconStopped []byte

//go:embed icons/yellow.png
var iconNoNet []byte

// main.go
package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"os/user"
	"path/filepath"
	"time"

	"github.com/getlantern/systray"
	"golang.org/x/sys/windows/registry"
)

type Config struct {
	IntervalSec int `json:"interval_sec"`
	TotalBytes  int `json:"total_bytes"`
}

var config Config
var configPath string

func loadConfig() {
	usr, _ := user.Current()
	configPath = filepath.Join(usr.HomeDir, ".hotspot_keeper.json")
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		config = Config{IntervalSec: 5}
		saveConfig()
		return
	}
	json.Unmarshal(data, &config)
}

func saveConfig() {
	data, _ := json.MarshalIndent(config, "", "  ")
	ioutil.WriteFile(configPath, data, 0644)
}

func setAutoStart() {
	exePath, _ := os.Executable()
	k, _, _ := registry.CreateKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.SET_VALUE)
	k.SetStringValue("HotspotKeeper", exePath)
	k.Close()
}

func main() {
	loadConfig()
	setAutoStart()
	systray.Run(onReady, func() {})
}

func onReady() {
	systray.SetTitle("Hotspot Keeper")
	systray.SetTooltip("防止iPhone热点断开")

	interval5 := systray.AddMenuItemCheckbox("5秒", "", config.IntervalSec == 5)
	interval10 := systray.AddMenuItemCheckbox("10秒", "", config.IntervalSec == 10)
	interval30 := systray.AddMenuItemCheckbox("30秒", "", config.IntervalSec == 30)

	systray.AddSeparator()
	statsItem := systray.AddMenuItem(fmt.Sprintf("流量统计：%dKB", config.TotalBytes/1024), "")
	systray.AddSeparator()
	quit := systray.AddMenuItem("退出", "退出程序")

	ticker := time.NewTicker(time.Duration(config.IntervalSec) * time.Second)
	go func() {
		for range ticker.C {
			bytesUsed := doRequest()
			config.TotalBytes += bytesUsed
			saveConfig()
			statsItem.SetTitle(fmt.Sprintf("流量统计：%dKB", config.TotalBytes/1024))
		}
	}()

	go func() {
		for {
			select {
			case <-interval5.ClickedCh:
				config.IntervalSec = 5
				interval5.Check()
				interval10.Uncheck()
				interval30.Uncheck()
				ticker.Reset(5 * time.Second)
				saveConfig()
			case <-interval10.ClickedCh:
				config.IntervalSec = 10
				interval5.Uncheck()
				interval10.Check()
				interval30.Uncheck()
				ticker.Reset(10 * time.Second)
				saveConfig()
			case <-interval30.ClickedCh:
				config.IntervalSec = 30
				interval5.Uncheck()
				interval10.Uncheck()
				interval30.Check()
				ticker.Reset(30 * time.Second)
				saveConfig()
			case <-quit.ClickedCh:
				systray.Quit()
				os.Exit(0)
			}
		}
	}()
}

func doRequest() int {
	client := http.Client{Timeout: 5 * time.Second}
	req, _ := http.NewRequest("HEAD", "https://www.gstatic.com/generate_204", nil)
	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return 0
	}
	resp.Body.Close()
	// 粗略估算 HEAD 请求约 700 字节
	elapsed := time.Since(start)
	return int(elapsed.Milliseconds())*10 + 700
}

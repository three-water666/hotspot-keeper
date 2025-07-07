package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"time"

	"github.com/getlantern/systray"
	"golang.org/x/sys/windows/registry"
)

//go:embed icons/green.ico
var iconRunning []byte

//go:embed icons/gray.ico
var iconStopped []byte

//go:embed icons/yellow.ico
var iconNoNet []byte

type Config struct {
	IntervalSec int  `json:"interval_sec"`
	TotalBytes  int  `json:"total_bytes"`
	AutoStart   bool `json:"auto_start"`
	IsRunning   bool `json:"is_running"`
}

var (
	config           Config
	configPath       string
	statusItem       *systray.MenuItem
	statsItem        *systray.MenuItem
	interval5        *systray.MenuItem
	interval10       *systray.MenuItem
	interval30       *systray.MenuItem
	startMenu        *systray.MenuItem
	startItem        *systray.MenuItem
	stopItem         *systray.MenuItem
	autoStartMenu    *systray.MenuItem
	autoStartOnItem  *systray.MenuItem
	autoStartOffItem *systray.MenuItem
	ticker           *time.Ticker
	ctx              context.Context
	cancelFunc       context.CancelFunc
)

func main() {
	loadConfig()
	systray.Run(onReady, onExit)
}

func onReady() {
	systray.SetTooltip("防止iPhone热点断开")

	// 初始化图标
	if config.IsRunning {
		systray.SetIcon(iconRunning)
	} else {
		systray.SetIcon(iconStopped)
	}

	systray.SetTitle("Hotspot Keeper")

	statusItem = systray.AddMenuItem("状态：已停止", "")

	// 探测间隔
	systray.AddSeparator()
	intervalMenu := systray.AddMenuItem("探测间隔", "")
	interval5 = intervalMenu.AddSubMenuItemCheckbox("5秒", "", config.IntervalSec == 5)
	interval10 = intervalMenu.AddSubMenuItemCheckbox("10秒", "", config.IntervalSec == 10)
	interval30 = intervalMenu.AddSubMenuItemCheckbox("30秒", "", config.IntervalSec == 30)

	// 启动/停止探测
	systray.AddSeparator()
	startMenu = systray.AddMenuItem("探测控制", "")
	startItem = startMenu.AddSubMenuItemCheckbox("启动", "", config.IsRunning)
	stopItem = startMenu.AddSubMenuItemCheckbox("停止", "", !config.IsRunning)

	// 开机启动
	autoStartMenu = systray.AddMenuItem("开机启动", "")
	autoStartOnItem = autoStartMenu.AddSubMenuItemCheckbox("启用", "", config.AutoStart)
	autoStartOffItem = autoStartMenu.AddSubMenuItemCheckbox("禁用", "", !config.AutoStart)

	// 流量统计
	systray.AddSeparator()
	statsItem = systray.AddMenuItem(fmt.Sprintf("流量统计：%dKB", config.TotalBytes/1024), "")

	// 退出
	systray.AddSeparator()
	quitItem := systray.AddMenuItem("退出", "退出程序")

	if config.IsRunning {
		startProbe()
	}

	go handleMenuEvents(quitItem)
}

func handleMenuEvents(quitItem *systray.MenuItem) {
	for {
		select {
		case <-interval5.ClickedCh:
			setInterval(5)
		case <-interval10.ClickedCh:
			setInterval(10)
		case <-interval30.ClickedCh:
			setInterval(30)
		case <-startItem.ClickedCh:
			startItem.Check()
			stopItem.Uncheck()
			startProbe()
		case <-stopItem.ClickedCh:
			startItem.Uncheck()
			stopItem.Check()
			stopProbe()
		case <-autoStartOnItem.ClickedCh:
			autoStartOnItem.Check()
			autoStartOffItem.Uncheck()
			enableAutoStart()
			config.AutoStart = true
			saveConfig()
		case <-autoStartOffItem.ClickedCh:
			autoStartOnItem.Uncheck()
			autoStartOffItem.Check()
			disableAutoStart()
			config.AutoStart = false
			saveConfig()
		case <-quitItem.ClickedCh:
			stopProbe()
			systray.Quit()
			os.Exit(0)
		}
	}
}

func setInterval(sec int) {
	config.IntervalSec = sec
	saveConfig()

	interval5.Check()
	interval10.Uncheck()
	interval30.Uncheck()
	if sec == 10 {
		interval10.Check()
	} else if sec == 30 {
		interval30.Check()
	}

	if config.IsRunning {
		stopProbe()
		startProbe()
	}
}

func startProbe() {
	if ctx != nil {
		return
	}
	ctx, cancelFunc = context.WithCancel(context.Background())
	ticker = time.NewTicker(time.Duration(config.IntervalSec) * time.Second)

	statusItem.SetTitle("状态：探测中")
	systray.SetIcon(iconRunning)
	config.IsRunning = true
	saveConfig()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				bytesUsed, err := doRequest()
				if err != nil {
					statusItem.SetTitle("状态：无网络")
					systray.SetIcon(iconNoNet)
					continue
				}
				statusItem.SetTitle("状态：探测中")
				systray.SetIcon(iconRunning)
				config.TotalBytes += bytesUsed
				statsItem.SetTitle(fmt.Sprintf("流量统计：%dKB", config.TotalBytes/1024))
				saveConfig()
			}
		}
	}()
}

func stopProbe() {
	if cancelFunc != nil {
		cancelFunc()
		ctx = nil
		cancelFunc = nil
	}
	if ticker != nil {
		ticker.Stop()
		ticker = nil
	}
	statusItem.SetTitle("状态：已停止")
	systray.SetIcon(iconStopped)
	config.IsRunning = false
	saveConfig()
}

func doRequest() (int, error) {
	client := http.Client{Timeout: 5 * time.Second}
	if !isNetworkAvailable() {
		return 0, fmt.Errorf("no network")
	}
	req, _ := http.NewRequest("HEAD", "https://www.gstatic.com/generate_204", nil)
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	resp.Body.Close()
	return 700, nil
}

func isNetworkAvailable() bool {
	conn, err := net.DialTimeout("tcp", "8.8.8.8:53", 2*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

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

func loadConfig() {
	usr, _ := user.Current()
	configPath = filepath.Join(usr.HomeDir, ".hotspot_keeper.json")
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		config = Config{IntervalSec: 5, AutoStart: false, IsRunning: false}
		saveConfig()
		return
	}
	json.Unmarshal(data, &config)
}

func saveConfig() {
	data, _ := json.MarshalIndent(config, "", "  ")
	ioutil.WriteFile(configPath, data, 0644)
}

func onExit() {
	stopProbe()
}

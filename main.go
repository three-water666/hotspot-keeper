// main.go
package main

import (
	"context"
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

type Config struct {
	IntervalSec int  `json:"interval_sec"`
	TotalBytes  int  `json:"total_bytes"`
	AutoStart   bool `json:"auto_start"`
}

var (
	config        Config
	configPath    string
	statusItem    *systray.MenuItem
	statsItem     *systray.MenuItem
	startStopItem *systray.MenuItem
	autoStartItem *systray.MenuItem
	interval5     *systray.MenuItem
	interval10    *systray.MenuItem
	interval30    *systray.MenuItem
	ctx           context.Context
	cancelFunc    context.CancelFunc
	isRunning     bool
	ticker        *time.Ticker
)

func main() {
	loadConfig()
	systray.Run(onReady, onExit)
}

func onReady() {
	systray.SetTitle("Hotspot Keeper")
	systray.SetTooltip("防止iPhone热点断开")

	// 状态显示
	statusItem = systray.AddMenuItem("状态：已停止", "")

	// 探测间隔菜单
	systray.AddSeparator()
	intervalMenu := systray.AddMenuItem("探测间隔", "")
	interval5 = intervalMenu.AddSubMenuItemCheckbox("5秒", "", config.IntervalSec == 5)
	interval10 = intervalMenu.AddSubMenuItemCheckbox("10秒", "", config.IntervalSec == 10)
	interval30 = intervalMenu.AddSubMenuItemCheckbox("30秒", "", config.IntervalSec == 30)

	// 启动/停止
	systray.AddSeparator()
	startStopItem = systray.AddMenuItem("启动探测", "开始网络保持探测")

	// 开机启动
	autoStartItem = systray.AddMenuItemCheckbox("开机启动", "", config.AutoStart)
	autoStartItem.Check()

	// 流量统计
	systray.AddSeparator()
	statsItem = systray.AddMenuItem(fmt.Sprintf("流量统计：%dKB", config.TotalBytes/1024), "")

	// 退出按钮
	systray.AddSeparator()
	quitItem := systray.AddMenuItem("退出", "退出程序")

	// 事件处理
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
		case <-startStopItem.ClickedCh:
			if isRunning {
				stopProbe()
			} else {
				startProbe()
			}
		case <-autoStartItem.ClickedCh:
			toggleAutoStart()
		case <-quitItem.ClickedCh:
			stopProbe()
			systray.Quit()
			os.Exit(0)
		}
	}
}

func setInterval(sec int) {
	config.IntervalSec = sec
	interval5.Check()
	interval10.Uncheck()
	interval30.Uncheck()
	if sec == 10 {
		interval5.Uncheck()
		interval10.Check()
	}
	if sec == 30 {
		interval5.Uncheck()
		interval10.Uncheck()
		interval30.Check()
	}
	saveConfig()
	if isRunning {
		stopProbe()
		startProbe()
	}
}

func startProbe() {
	if isRunning {
		return
	}
	isRunning = true
	ctx, cancelFunc = context.WithCancel(context.Background())
	ticker = time.NewTicker(time.Duration(config.IntervalSec) * time.Second)

	startStopItem.SetTitle("停止探测")
	statusItem.SetTitle("状态：探测中")

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				bytesUsed, netErr := doRequest()
				if netErr != nil {
					statusItem.SetTitle("状态：无网络")
					continue
				}
				config.TotalBytes += bytesUsed
				statsItem.SetTitle(fmt.Sprintf("流量统计：%dKB", config.TotalBytes/1024))
				statusItem.SetTitle("状态：探测中")
				saveConfig()
			}
		}
	}()
}

func stopProbe() {
	if !isRunning {
		return
	}
	isRunning = false
	if cancelFunc != nil {
		cancelFunc()
	}
	if ticker != nil {
		ticker.Stop()
	}
	startStopItem.SetTitle("启动探测")
	statusItem.SetTitle("状态：已停止")
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
	return 700, nil // 估算 HEAD 请求大小
}

func isNetworkAvailable() bool {
	conn, err := net.DialTimeout("tcp", "8.8.8.8:53", 2*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func toggleAutoStart() {
	config.AutoStart = !config.AutoStart
	if config.AutoStart {
		enableAutoStart()
		autoStartItem.Check()
	} else {
		disableAutoStart()
		autoStartItem.Uncheck()
	}
	saveConfig()
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

func onExit() {
	stopProbe()
}

func loadConfig() {
	usr, _ := user.Current()
	configPath = filepath.Join(usr.HomeDir, ".hotspot_keeper.json")
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		config = Config{IntervalSec: 5, AutoStart: false}
		saveConfig()
		return
	}
	json.Unmarshal(data, &config)
}

func saveConfig() {
	data, _ := json.MarshalIndent(config, "", "  ")
	ioutil.WriteFile(configPath, data, 0644)
}

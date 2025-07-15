# HotspotKeeper

> **防止 iPhone 热点自动断开的小工具，支持 Windows 和 macOS。**

## 功能简介

- 托盘驻留，自动定时探测网络，防止 iPhone 热点因长时间无流量而断开。
- 支持自定义探测间隔（5秒/10秒/30秒）。
- 支持流量统计。
- 支持开机自启动（Windows/ macOS）。
- 支持一键启动/停止探测。
- 支持多种状态图标（运行/停止/无网络）。

## 截图

> ![托盘菜单示例](icons/green.png)

## 安装与编译

### Windows

```cmd
# 需安装 Go 环境
# 编译命令：
go build -ldflags="-s -w -H=windowsgui" -o HotspotKeeper.exe
```

### macOS

```zsh
# 需安装 Go 环境
# 编译命令：
go build -ldflags="-s -w" -o HotspotKeeper
```

## 依赖

- [github.com/getlantern/systray](https://github.com/getlantern/systray)
- [golang.org/x/sys](https://pkg.go.dev/golang.org/x/sys)

依赖已在 `go.mod` 中声明，首次编译会自动拉取。

## 使用说明

1. 启动程序后会在系统托盘显示图标。
2. 右键菜单可设置探测间隔、启动/停止探测、开机自启、查看流量统计等。
3. 探测期间会定时访问 Google 以保持热点活跃。
4. 配置文件默认保存在用户主目录下 `.hotspot_keeper.json`。

## 开机自启动说明

- **Windows**：通过注册表 `Run` 项实现。
- **macOS**：通过 `~/Library/LaunchAgents/com.hotspot.keeper.plist` 实现。

## 图标说明

- `icons/green.*`：探测中
- `icons/gray.*`：已停止
- `icons/yellow.*`：无网络

## License

MIT
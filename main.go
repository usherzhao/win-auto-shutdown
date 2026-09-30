package main

import (
	"errors"
	"flag"
	"os"

	"github.com/lxn/walk"
	"github.com/lxn/win"
	"golang.org/x/sys/windows"
)

func main() {
	trayFlag := flag.Bool("tray", false, "启动后静默最小化到系统托盘")
	flag.Parse()

	// 单实例检测：防止重复启动多个进程互相冲突
	mutexName := windows.StringToUTF16Ptr("Local\\WinAutoShutdown_App_Mutex")
	hMutex, err := windows.CreateMutex(nil, true, mutexName)
	if err != nil && errors.Is(err, windows.ERROR_ALREADY_EXISTS) {
		// 已经有实例在运行，尝试查找并唤醒现有窗口
		titlePtr := windows.StringToUTF16Ptr("⏰ Windows 定时关机助手")
		hwnd := win.FindWindow(nil, titlePtr)
		if hwnd != 0 {
			win.ShowWindow(hwnd, win.SW_RESTORE)
			win.SetForegroundWindow(hwnd)
		}
		os.Exit(0)
		return
	}
	if hMutex != 0 {
		defer windows.CloseHandle(hMutex)
	}

	// 加载配置
	cfg := LoadConfig()

	// 加载应用图标（优先从嵌入资源获取，回退到文件或系统图标）
	var appIcon *walk.Icon
	if ico, err := walk.NewIconFromResourceId(2); err == nil {
		appIcon = ico
	} else if ico, err := walk.NewIconFromFile("app.ico"); err == nil {
		appIcon = ico
	} else {
		appIcon = walk.IconInformation()
	}

	// 初始化调度器
	scheduler := NewScheduler()

	// 创建并启动主窗口程序
	app := NewMainWindowApp(scheduler, cfg, appIcon)
	if err := app.Run(*trayFlag); err != nil {
		walk.MsgBox(nil, "程序启动错误", err.Error(), walk.MsgBoxIconError)
	}
}

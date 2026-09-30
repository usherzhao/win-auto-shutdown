package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

const (
	regRunPath = `Software\Microsoft\Windows\CurrentVersion\Run`
	regAppName = "WinAutoShutdown"
)

// IsAutoStartEnabled 检查当前程序是否已加入开机自启注册表
func IsAutoStartEnabled() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, regRunPath, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()

	val, _, err := k.GetStringValue(regAppName)
	if err != nil || val == "" {
		return false
	}

	exePath, err := os.Executable()
	if err != nil {
		return false
	}
	exePath = filepath.Clean(exePath)

	// 只要包含当前可执行程序路径即可
	return strings.Contains(strings.ToLower(val), strings.ToLower(exePath))
}

// SetAutoStart 设置或移除开机自动启动
func SetAutoStart(enable bool) error {
	k, err := registry.OpenKey(registry.CURRENT_USER, regRunPath, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("无法打开注册表启动项: %w", err)
	}
	defer k.Close()

	if enable {
		exePath, err := os.Executable()
		if err != nil {
			return fmt.Errorf("获取程序路径失败: %w", err)
		}
		exePath, err = filepath.Abs(exePath)
		if err != nil {
			return fmt.Errorf("获取程序绝对路径失败: %w", err)
		}
		// 使用引号包围路径以应对空格，添加 -tray 参数在开机时启动到托盘
		cmdVal := fmt.Sprintf("\"%s\" -tray", exePath)
		if err := k.SetStringValue(regAppName, cmdVal); err != nil {
			return fmt.Errorf("写入注册表自启项失败: %w", err)
		}
	} else {
		// 删除自启项
		_ = k.DeleteValue(regAppName)
	}
	return nil
}

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config 存储定时关机小工具的配置项
type Config struct {
	AutoStart              bool   `json:"auto_start"`               // 是否开机自启
	Mode                   string `json:"mode"`                     // "time" (指定时刻) 或 "countdown" (倒计时)
	TargetHour             int    `json:"target_hour"`              // 指定关机小时 (0-23)
	TargetMinute           int    `json:"target_minute"`            // 指定关机分钟 (0-59)
	CountdownHours         int    `json:"countdown_hours"`          // 倒计时小时
	CountdownMinutes       int    `json:"countdown_minutes"`        // 倒计时分钟
	DailySchedule          bool   `json:"daily_schedule"`           // 每天固定时间关机
	AdvanceReminderMinutes int    `json:"advance_reminder_minutes"` // 提前提醒分钟数 (默认5分钟)
	PlaySound              bool   `json:"play_sound"`               // 提醒时播放提示音
	Action                 string `json:"action"`                   // 执行动作: "shutdown" (关机), "restart" (重启), "sleep" (睡眠)
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		AutoStart:              false,
		Mode:                   "time",
		TargetHour:             23,
		TargetMinute:           0,
		CountdownHours:         1,
		CountdownMinutes:       0,
		DailySchedule:          false,
		AdvanceReminderMinutes: 5,
		PlaySound:              true,
		Action:                 "shutdown",
	}
}

// getConfigPath 获取配置文件路径
func getConfigPath() string {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		appData = "."
	}
	dir := filepath.Join(appData, "WinAutoShutdown")
	_ = os.MkdirAll(dir, 0755)
	return filepath.Join(dir, "config.json")
}

// LoadConfig 从文件加载配置，若失败则返回默认配置
func LoadConfig() *Config {
	cfg := DefaultConfig()
	filePath := getConfigPath()
	data, err := os.ReadFile(filePath)
	if err != nil {
		return cfg
	}
	_ = json.Unmarshal(data, cfg)

	// 保证合法边界
	if cfg.AdvanceReminderMinutes <= 0 {
		cfg.AdvanceReminderMinutes = 5
	}
	if cfg.Action == "" {
		cfg.Action = "shutdown"
	}
	if cfg.Mode == "" {
		cfg.Mode = "time"
	}
	return cfg
}

// SaveConfig 保存配置到文件
func (c *Config) SaveConfig() error {
	filePath := getConfigPath()
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, data, 0644)
}

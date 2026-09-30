package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

type ScheduleStatus int

const (
	StatusStopped ScheduleStatus = iota
	StatusRunning
)

// Scheduler 负责关机任务的核心调度与状态管理
type Scheduler struct {
	mu                     sync.RWMutex
	status                 ScheduleStatus
	targetTime             time.Time
	advanceReminderMinutes int
	action                 string // "shutdown", "restart", "sleep"
	dailyRepeat            bool
	dailyHour              int
	dailyMinute            int
	reminderTriggered      bool

	// 回调函数，通知 UI 更新
	onTick              func(remaining time.Duration, status ScheduleStatus)
	onReminderTriggered func(remaining time.Duration)
	onActionExecuted    func(action string)
	onCancelled         func()
}

// NewScheduler 创建新的调度器
func NewScheduler() *Scheduler {
	return &Scheduler{
		status:                 StatusStopped,
		advanceReminderMinutes: 5,
		action:                 "shutdown",
	}
}

// SetCallbacks 绑定事件回调
func (s *Scheduler) SetCallbacks(
	onTick func(remaining time.Duration, status ScheduleStatus),
	onReminderTriggered func(remaining time.Duration),
	onActionExecuted func(action string),
	onCancelled func(),
) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onTick = onTick
	s.onReminderTriggered = onReminderTriggered
	s.onActionExecuted = onActionExecuted
	s.onCancelled = onCancelled
}

// StartScheduleTime 按指定具体时刻 (小时、分钟) 启动定时
func (s *Scheduler) StartScheduleTime(hour, minute int, advanceMinutes int, action string, daily bool) time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	target := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())

	// 如果设定的时间今天已过，则排到明天
	if !target.After(now) {
		target = target.Add(24 * time.Hour)
	}

	s.targetTime = target
	s.advanceReminderMinutes = advanceMinutes
	s.action = action
	s.dailyRepeat = daily
	s.dailyHour = hour
	s.dailyMinute = minute
	s.status = StatusRunning
	s.reminderTriggered = false

	return target
}

// StartCountdown 按倒计时时长启动
func (s *Scheduler) StartCountdown(duration time.Duration, advanceMinutes int, action string) time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()

	target := time.Now().Add(duration)
	s.targetTime = target
	s.advanceReminderMinutes = advanceMinutes
	s.action = action
	s.dailyRepeat = false
	s.status = StatusRunning
	s.reminderTriggered = false

	return target
}

// Postpone 延迟指定的分钟数
func (s *Scheduler) Postpone(minutes int) time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.status != StatusRunning {
		return time.Time{}
	}

	s.targetTime = s.targetTime.Add(time.Duration(minutes) * time.Minute)

	// 如果延迟后的剩余时间大于提前提醒时间，重置提醒触发标记，使得下次到点时依然会提醒
	now := time.Now()
	advanceDuration := time.Duration(s.advanceReminderMinutes) * time.Minute
	if s.targetTime.Sub(now) > advanceDuration {
		s.reminderTriggered = false
	}

	return s.targetTime
}

// Cancel 取消当前定时任务
func (s *Scheduler) Cancel() {
	s.mu.Lock()
	cb := s.onCancelled
	s.status = StatusStopped
	s.reminderTriggered = false
	s.mu.Unlock()

	// 尝试取消 Windows 原生关机队列
	cmd := exec.Command(getShutdownPath(), "-a")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	_ = cmd.Run()

	if cb != nil {
		cb()
	}
}

// Status 获取当前运行状态与关键信息
func (s *Scheduler) Status() (status ScheduleStatus, target time.Time, remaining time.Duration, action string, reminderTriggered bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now()
	var rem time.Duration
	if s.status == StatusRunning {
		rem = s.targetTime.Sub(now)
		if rem < 0 {
			rem = 0
		}
	}
	return s.status, s.targetTime, rem, s.action, s.reminderTriggered
}

// Tick 每秒由定时器调用一次，检查并触发提醒或关机动作
func (s *Scheduler) Tick() {
	var shouldExecute bool
	var shouldRemind bool
	var curAction string
	var rem time.Duration
	var curStatus ScheduleStatus
	var tickCb func(remaining time.Duration, status ScheduleStatus)
	var remindCb func(remaining time.Duration)
	var execCb func(action string)

	s.mu.Lock()
	curStatus = s.status
	tickCb = s.onTick

	if s.status == StatusRunning {
		now := time.Now()
		rem = s.targetTime.Sub(now)

		if rem <= 0 {
			shouldExecute = true
			curAction = s.action
			execCb = s.onActionExecuted

			if s.dailyRepeat {
				// 每天重复：重新计算明天该时间
				tomorrow := now.Add(24 * time.Hour)
				s.targetTime = time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), s.dailyHour, s.dailyMinute, 0, 0, now.Location())
				s.reminderTriggered = false
			} else {
				s.status = StatusStopped
			}
		} else {
			advanceDuration := time.Duration(s.advanceReminderMinutes) * time.Minute
			if rem <= advanceDuration && !s.reminderTriggered {
				s.reminderTriggered = true
				shouldRemind = true
				remindCb = s.onReminderTriggered
			}
		}
	}
	s.mu.Unlock()

	if tickCb != nil {
		tickCb(rem, curStatus)
	}
	if shouldRemind && remindCb != nil {
		remindCb(rem)
	}
	if shouldExecute {
		if execCb != nil {
			execCb(curAction)
		}
		s.ExecuteAction(curAction)
	}
}

// ExecuteNow 立即执行动作
func (s *Scheduler) ExecuteNow(action string) {
	s.mu.Lock()
	s.status = StatusStopped
	s.reminderTriggered = false
	s.mu.Unlock()

	s.ExecuteAction(action)
}

// getShutdownPath 获取 Windows shutdown.exe 的绝对路径，优先从 System32 读取以防环境变量污染
func getShutdownPath() string {
	sysRoot := os.Getenv("SystemRoot")
	if sysRoot != "" {
		p := filepath.Join(sysRoot, "System32", "shutdown.exe")
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return "shutdown.exe"
}

// ExecuteAction 执行 Windows 系统关机、重启或睡眠
// 明确包含 -f 强制关机参数，防止被未保存工作等后台程序弹窗打断
func (s *Scheduler) ExecuteAction(action string) {
	shutdownExe := getShutdownPath()

	switch action {
	case "restart":
		// 强制重启电脑 (-r 重启, -f 强制关闭所有应用程序防止被卡住, -t 0 立即执行)
		cmd := exec.Command(shutdownExe, "-r", "-f", "-t", "0")
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		_ = cmd.Run()
	case "sleep":
		// 睡眠/休眠
		cmd := exec.Command("rundll32.exe", "powrprof.dll,SetSuspendState", "0,1,0")
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		_ = cmd.Run()
	default:
		// 强制关机 (-s 关机, -f 强制关闭所有正在运行的程序防止被未保存提示打断, -t 0 立即关机)
		cmd := exec.Command(shutdownExe, "-s", "-f", "-t", "0")
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		_ = cmd.Run()
	}
}

// FormatDuration 格式化剩余时间为中文友好字符串
func FormatDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	sec := int(d.Seconds()) % 60

	if h > 0 {
		return fmt.Sprintf("%02d小时 %02d分 %02d秒", h, m, sec)
	}
	return fmt.Sprintf("%02d分 %02d秒", m, sec)
}

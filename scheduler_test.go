package main

import (
	"testing"
	"time"
)

func TestSchedulerScheduleTime(t *testing.T) {
	s := NewScheduler()

	now := time.Now()
	// 设定1小时后的时间
	futureTime := now.Add(1 * time.Hour)
	target := s.StartScheduleTime(futureTime.Hour(), futureTime.Minute(), 5, "shutdown", false)

	status, targetTime, remaining, action, _ := s.Status()
	if status != StatusRunning {
		t.Fatalf("expected StatusRunning, got %v", status)
	}
	if action != "shutdown" {
		t.Fatalf("expected action shutdown, got %v", action)
	}
	if remaining < 50*time.Minute || remaining > 65*time.Minute {
		t.Fatalf("unexpected remaining duration: %v", remaining)
	}
	if target != targetTime {
		t.Fatalf("target mismatch: %v vs %v", target, targetTime)
	}
}

func TestSchedulerCountdown(t *testing.T) {
	s := NewScheduler()

	dur := 15 * time.Minute
	target := s.StartCountdown(dur, 5, "restart")

	status, _, remaining, action, _ := s.Status()
	if status != StatusRunning {
		t.Fatalf("expected StatusRunning, got %v", status)
	}
	if action != "restart" {
		t.Fatalf("expected action restart, got %v", action)
	}
	if remaining < 14*time.Minute || remaining > 16*time.Minute {
		t.Fatalf("unexpected remaining: %v", remaining)
	}
	if time.Until(target) > 16*time.Minute {
		t.Fatalf("target too far: %v", target)
	}
}

func TestSchedulerPostpone(t *testing.T) {
	s := NewScheduler()

	s.StartCountdown(10*time.Minute, 5, "shutdown")
	_, oldTarget, _, _, _ := s.Status()

	// 延迟 5 分钟
	newTarget5 := s.Postpone(5)
	diff5 := newTarget5.Sub(oldTarget)
	if diff5 < 4*time.Minute+50*time.Second || diff5 > 5*time.Minute+10*time.Second {
		t.Fatalf("postpone 5 min incorrect, diff: %v", diff5)
	}

	// 延迟 10 分钟
	newTarget10 := s.Postpone(10)
	diff10 := newTarget10.Sub(newTarget5)
	if diff10 < 9*time.Minute+50*time.Second || diff10 > 10*time.Minute+10*time.Second {
		t.Fatalf("postpone 10 min incorrect, diff: %v", diff10)
	}
}

func TestSchedulerCancel(t *testing.T) {
	s := NewScheduler()
	cancelled := false
	s.SetCallbacks(nil, nil, nil, func() {
		cancelled = true
	})

	s.StartCountdown(10*time.Minute, 5, "shutdown")
	status, _, _, _, _ := s.Status()
	if status != StatusRunning {
		t.Fatalf("expected StatusRunning")
	}

	s.Cancel()
	status, _, _, _, _ = s.Status()
	if status != StatusStopped {
		t.Fatalf("expected StatusStopped after cancel")
	}
	if !cancelled {
		t.Fatalf("expected onCancelled callback to be called")
	}
}

func TestSchedulerReminderTrigger(t *testing.T) {
	s := NewScheduler()
	remindTriggered := false
	s.SetCallbacks(
		nil,
		func(remaining time.Duration) {
			remindTriggered = true
		},
		nil,
		nil,
	)

	// 启动一个在 4 秒后关机、提前 1 分钟提醒的任务（因为 4秒 <= 60秒，应当立即触发提醒）
	s.StartCountdown(4*time.Second, 1, "shutdown")
	s.Tick()

	if !remindTriggered {
		t.Fatalf("expected reminder to trigger when remaining <= advance duration")
	}
}

func TestConfigSaveAndLoad(t *testing.T) {
	cfg := LoadConfig()
	origHour := cfg.TargetHour
	cfg.TargetHour = 19
	cfg.AdvanceReminderMinutes = 8

	err := cfg.SaveConfig()
	if err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	loaded := LoadConfig()
	if loaded.TargetHour != 19 || loaded.AdvanceReminderMinutes != 8 {
		t.Fatalf("LoadConfig mismatch: %+v", loaded)
	}

	// 恢复原值
	cfg.TargetHour = origHour
	_ = cfg.SaveConfig()
}

func TestAutoStartRegistry(t *testing.T) {
	// 测试开机自启设置
	initialState := IsAutoStartEnabled()

	// 尝试开启
	err := SetAutoStart(true)
	if err != nil {
		t.Fatalf("SetAutoStart(true) failed: %v", err)
	}
	if !IsAutoStartEnabled() {
		t.Fatalf("expected AutoStart to be enabled")
	}

	// 尝试关闭
	err = SetAutoStart(false)
	if err != nil {
		t.Fatalf("SetAutoStart(false) failed: %v", err)
	}
	if IsAutoStartEnabled() {
		t.Fatalf("expected AutoStart to be disabled")
	}

	// 恢复初始状态
	_ = SetAutoStart(initialState)
}

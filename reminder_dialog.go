package main

import (
	"fmt"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"github.com/lxn/win"
)

// ReminderDialog 关机提醒弹出对话框
type ReminderDialog struct {
	dialog         *walk.Dialog
	countdownLabel *walk.Label
	infoLabel      *walk.Label
	scheduler      *Scheduler
	config         *Config
	notifyIcon     *walk.NotifyIcon
	appIcon        *walk.Icon
}

var currentReminderDialog *ReminderDialog

// ShowReminderDialog 弹出置顶的关机提醒窗口
func ShowReminderDialog(owner walk.Form, s *Scheduler, cfg *Config, ni *walk.NotifyIcon, icon *walk.Icon) {
	// 如果已有提醒窗口在展示，直接激活并置顶
	if currentReminderDialog != nil && currentReminderDialog.dialog != nil {
		win.SetWindowPos(currentReminderDialog.dialog.Handle(), win.HWND_TOPMOST, 0, 0, 0, 0, win.SWP_NOMOVE|win.SWP_NOSIZE|win.SWP_SHOWWINDOW)
		win.SetForegroundWindow(currentReminderDialog.dialog.Handle())
		if cfg.PlaySound {
			win.MessageBeep(win.MB_ICONEXCLAMATION)
		}
		return
	}

	rd := &ReminderDialog{
		scheduler:  s,
		config:     cfg,
		notifyIcon: ni,
		appIcon:    icon,
	}
	currentReminderDialog = rd

	_, target, rem, action, _ := s.Status()
	actionText := "关机"
	if action == "restart" {
		actionText = "重启"
	} else if action == "sleep" {
		actionText = "睡眠"
	}

	err := Dialog{
		AssignTo: &rd.dialog,
		Title:    "⏰ 关机提醒 - 计算机即将自动关机",
		Icon:     icon,
		MinSize:  Size{Width: 460, Height: 270},
		Size:     Size{Width: 480, Height: 285},
		Layout:   VBox{Margins: Margins{Left: 20, Top: 15, Right: 20, Bottom: 15}, Spacing: 10},
		Children: []Widget{
			Label{
				Text:      fmt.Sprintf("⚠️ 计算机将在倒计时结束后执行【%s】！", actionText),
				Font:      Font{Family: "Microsoft YaHei UI", PointSize: 11, Bold: true},
				TextColor: walk.RGB(180, 0, 0),
			},
			Label{
				Text: "请尽快保存正在编辑的文件和工作内容，以免数据丢失。",
			},
			Label{
				AssignTo:  &rd.countdownLabel,
				Text:      fmt.Sprintf("剩余时间: %s", FormatDuration(rem)),
				Font:      Font{Family: "Microsoft YaHei UI", PointSize: 16, Bold: true},
				TextColor: walk.RGB(200, 30, 30),
			},
			Label{
				AssignTo: &rd.infoLabel,
				Text:     fmt.Sprintf("计划执行时间: %s", target.Format("2006-01-02 15:04:05")),
			},
			VSpacer{Size: 5},
			// 第一排核心操作按钮 (用户明确要求的选项: 取消、延迟5分钟、延迟10分钟)
			Composite{
				Layout: HBox{MarginsZero: true, Spacing: 8},
				Children: []Widget{
					PushButton{
						Text: "🛑 取消关机",
						OnClicked: func() {
							rd.closeDialog()
							s.Cancel()
							if ni != nil {
								ni.ShowInfo("定时关机", "已取消本次关机任务")
							}
						},
					},
					PushButton{
						Text: "⏱️ 延迟 5 分钟",
						OnClicked: func() {
							newTarget := s.Postpone(5)
							rd.closeDialog()
							if ni != nil {
								ni.ShowInfo("关机已延迟", fmt.Sprintf("已顺延 5 分钟，将于 %s 执行关机", newTarget.Format("15:04:05")))
							}
						},
					},
					PushButton{
						Text: "⏱️ 延迟 10 分钟",
						OnClicked: func() {
							newTarget := s.Postpone(10)
							rd.closeDialog()
							if ni != nil {
								ni.ShowInfo("关机已延迟", fmt.Sprintf("已顺延 10 分钟，将于 %s 执行关机", newTarget.Format("15:04:05")))
							}
						},
					},
				},
			},
			// 第二排扩展选项
			Composite{
				Layout: HBox{MarginsZero: true, Spacing: 8},
				Children: []Widget{
					PushButton{
						Text: "延迟 30 分钟",
						OnClicked: func() {
							newTarget := s.Postpone(30)
							rd.closeDialog()
							if ni != nil {
								ni.ShowInfo("关机已延迟", fmt.Sprintf("已顺延 30 分钟，将于 %s 执行关机", newTarget.Format("15:04:05")))
							}
						},
					},
					HSpacer{},
					PushButton{
						Text: "⚡ 立即关机",
						OnClicked: func() {
							rd.closeDialog()
							s.ExecuteNow(action)
						},
					},
				},
			},
		},
	}.Create(owner)

	if err != nil {
		currentReminderDialog = nil
		return
	}

	// 播放声音提示
	if cfg.PlaySound {
		win.MessageBeep(win.MB_ICONEXCLAMATION)
	}

	// 强制置顶并激活到最前
	win.SetWindowPos(rd.dialog.Handle(), win.HWND_TOPMOST, 0, 0, 0, 0, win.SWP_NOMOVE|win.SWP_NOSIZE|win.SWP_SHOWWINDOW)
	win.SetForegroundWindow(rd.dialog.Handle())

	rd.dialog.Closing().Attach(func(canceled *bool, reason walk.CloseReason) {
		currentReminderDialog = nil
	})

	rd.dialog.Show()
}

// UpdateReminderDialogOnTick 在每秒心跳时同步刷新弹窗上的剩余时间和状态
func UpdateReminderDialogOnTick(s *Scheduler) {
	if currentReminderDialog == nil || currentReminderDialog.dialog == nil {
		return
	}
	status, curTarget, rem, curAct, _ := s.Status()
	if status != StatusRunning || rem <= 0 {
		currentReminderDialog.closeDialog()
		return
	}
	if currentReminderDialog.countdownLabel != nil {
		_ = currentReminderDialog.countdownLabel.SetText(fmt.Sprintf("剩余时间: %s", FormatDuration(rem)))
	}
	if currentReminderDialog.infoLabel != nil {
		actName := "关机"
		if curAct == "restart" {
			actName = "重启"
		} else if curAct == "sleep" {
			actName = "睡眠"
		}
		_ = currentReminderDialog.infoLabel.SetText(fmt.Sprintf("计划执行时间: %s (动作: %s)", curTarget.Format("2006-01-02 15:04:05"), actName))
	}
}

func (rd *ReminderDialog) closeDialog() {
	if rd.dialog != nil {
		rd.dialog.Close(0)
	}
	currentReminderDialog = nil
}

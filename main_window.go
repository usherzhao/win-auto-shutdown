package main

import (
	"fmt"
	"time"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"github.com/lxn/win"
)

type MainWindowApp struct {
	mainWindow       *walk.MainWindow
	notifyIcon       *walk.NotifyIcon
	appIcon          *walk.Icon
	scheduler        *Scheduler
	config           *Config
	stopTicker       chan struct{}

	// 控件引用
	rbTimeMode       *walk.RadioButton
	rbCountdownMode  *walk.RadioButton
	neTargetHour     *walk.NumberEdit
	neTargetMinute   *walk.NumberEdit
	cbDaily          *walk.CheckBox
	neCountHour      *walk.NumberEdit
	neCountMin       *walk.NumberEdit
	neAdvanceMin     *walk.NumberEdit
	cbAction         *walk.ComboBox
	cbPlaySound      *walk.CheckBox
	cbAutoStart      *walk.CheckBox
	lblStatus        *walk.Label
	lblRemaining     *walk.Label
	lblTarget        *walk.Label
	lblReminderInfo  *walk.Label
	btnStart         *walk.PushButton
	btnCancel        *walk.PushButton
	trayStatusAction *walk.Action
}

func NewMainWindowApp(s *Scheduler, cfg *Config, icon *walk.Icon) *MainWindowApp {
	return &MainWindowApp{
		scheduler:  s,
		config:     cfg,
		appIcon:    icon,
		stopTicker: make(chan struct{}),
	}
}

func (app *MainWindowApp) Run(startInTray bool) error {
	actionList := []string{"关机", "重启", "睡眠"}
	defaultActionIdx := 0
	if app.config.Action == "restart" {
		defaultActionIdx = 1
	} else if app.config.Action == "sleep" {
		defaultActionIdx = 2
	}

	titleFont := Font{Family: "Microsoft YaHei UI", PointSize: 10, Bold: true}
	statusFont := Font{Family: "Microsoft YaHei UI", PointSize: 11, Bold: true}
	remainFont := Font{Family: "Microsoft YaHei UI", PointSize: 14, Bold: true}

	err := MainWindow{
		AssignTo: &app.mainWindow,
		Title:    "⏰ Windows 定时关机助手",
		Icon:     app.appIcon,
		MinSize:  Size{Width: 490, Height: 580},
		Size:     Size{Width: 510, Height: 600},
		Layout:   VBox{Margins: Margins{Left: 15, Top: 15, Right: 15, Bottom: 15}, Spacing: 8},
		Children: []Widget{
			// 1. 关机模式与时间设置
			GroupBox{
				Title:  "关机模式与时刻设定",
				Font:   titleFont,
				Layout: VBox{Margins: Margins{Left: 10, Top: 10, Right: 10, Bottom: 10}, Spacing: 6},
				Children: []Widget{
					Composite{
						Layout: HBox{MarginsZero: true, Spacing: 15},
						Children: []Widget{
							RadioButton{
								AssignTo: &app.rbTimeMode,
								Text:     "指定时刻关机",
								OnClicked: func() {
									app.updateModeUI()
								},
							},
							RadioButton{
								AssignTo: &app.rbCountdownMode,
								Text:     "倒计时关机",
								OnClicked: func() {
									app.updateModeUI()
								},
							},
						},
					},
					// 指定时间行
					Composite{
						Layout: HBox{MarginsZero: true, Spacing: 5},
						Children: []Widget{
							Label{Text: "指定时间:"},
							NumberEdit{
								AssignTo: &app.neTargetHour,
								Value:    float64(app.config.TargetHour),
								MinValue: 0,
								MaxValue: 23,
							},
							Label{Text: "时"},
							NumberEdit{
								AssignTo: &app.neTargetMinute,
								Value:    float64(app.config.TargetMinute),
								MinValue: 0,
								MaxValue: 59,
							},
							Label{Text: "分"},
							HSpacer{},
							CheckBox{
								AssignTo: &app.cbDaily,
								Text:     "每天固定此时间关机",
								Checked:  app.config.DailySchedule,
							},
						},
					},
					// 快捷时刻
					Composite{
						Layout: HBox{MarginsZero: true, Spacing: 5},
						Children: []Widget{
							Label{Text: "快捷设定:"},
							PushButton{Text: "22:00", OnClicked: func() { app.setTime(22, 0) }},
							PushButton{Text: "22:30", OnClicked: func() { app.setTime(22, 30) }},
							PushButton{Text: "23:00", OnClicked: func() { app.setTime(23, 0) }},
							PushButton{Text: "23:30", OnClicked: func() { app.setTime(23, 30) }},
							PushButton{Text: "00:00", OnClicked: func() { app.setTime(0, 0) }},
						},
					},
					// 倒计时行
					Composite{
						Layout: HBox{MarginsZero: true, Spacing: 5},
						Children: []Widget{
							Label{Text: "倒计时间:"},
							NumberEdit{
								AssignTo: &app.neCountHour,
								Value:    float64(app.config.CountdownHours),
								MinValue: 0,
								MaxValue: 99,
							},
							Label{Text: "小时"},
							NumberEdit{
								AssignTo: &app.neCountMin,
								Value:    float64(app.config.CountdownMinutes),
								MinValue: 0,
								MaxValue: 59,
							},
							Label{Text: "分钟"},
							HSpacer{},
						},
					},
					// 倒计时快捷按钮
					Composite{
						Layout: HBox{MarginsZero: true, Spacing: 5},
						Children: []Widget{
							Label{Text: "快捷时长:"},
							PushButton{Text: "15分钟", OnClicked: func() { app.setCountdown(0, 15) }},
							PushButton{Text: "30分钟", OnClicked: func() { app.setCountdown(0, 30) }},
							PushButton{Text: "1小时", OnClicked: func() { app.setCountdown(1, 0) }},
							PushButton{Text: "2小时", OnClicked: func() { app.setCountdown(2, 0) }},
							PushButton{Text: "3小时", OnClicked: func() { app.setCountdown(3, 0) }},
						},
					},
				},
			},

			// 2. 提醒与常规设置
			GroupBox{
				Title:  "关机提醒与系统设置",
				Font:   titleFont,
				Layout: VBox{Margins: Margins{Left: 10, Top: 10, Right: 10, Bottom: 10}, Spacing: 6},
				Children: []Widget{
					Composite{
						Layout: HBox{MarginsZero: true, Spacing: 5},
						Children: []Widget{
							Label{Text: "提前提醒:"},
							NumberEdit{
								AssignTo: &app.neAdvanceMin,
								Value:    float64(app.config.AdvanceReminderMinutes),
								MinValue: 1,
								MaxValue: 120,
							},
							Label{Text: "分钟提醒用户 (可自定义)"},
							HSpacer{},
							Label{Text: "快捷:"},
							PushButton{Text: "3分钟", OnClicked: func() { _ = app.neAdvanceMin.SetValue(3) }},
							PushButton{Text: "5分钟", OnClicked: func() { _ = app.neAdvanceMin.SetValue(5) }},
							PushButton{Text: "10分钟", OnClicked: func() { _ = app.neAdvanceMin.SetValue(10) }},
						},
					},
					Composite{
						Layout: HBox{MarginsZero: true, Spacing: 15},
						Children: []Widget{
							Label{Text: "执行动作:"},
							ComboBox{
								AssignTo:     &app.cbAction,
								Model:        actionList,
								CurrentIndex: defaultActionIdx,
							},
							CheckBox{
								AssignTo: &app.cbPlaySound,
								Text:     "提醒时播放警示音",
								Checked:  app.config.PlaySound,
							},
							CheckBox{
								AssignTo: &app.cbAutoStart,
								Text:     "开机自动启动",
								Checked:  IsAutoStartEnabled(),
								OnClicked: func() {
									enable := app.cbAutoStart.Checked()
									if err := SetAutoStart(enable); err != nil {
										walk.MsgBox(app.mainWindow, "设置开机自启失败", err.Error(), walk.MsgBoxIconError)
										app.cbAutoStart.SetChecked(!enable)
									} else {
										app.config.AutoStart = enable
										_ = app.config.SaveConfig()
									}
								},
							},
						},
					},
				},
			},

			// 3. 当前运行状态与倒计时看板
			GroupBox{
				Title:  "当前运行状态",
				Font:   titleFont,
				Layout: VBox{Margins: Margins{Left: 10, Top: 10, Right: 10, Bottom: 10}, Spacing: 5},
				Children: []Widget{
					Composite{
						Layout: HBox{MarginsZero: true, Spacing: 10},
						Children: []Widget{
							Label{
								AssignTo:  &app.lblStatus,
								Text:      "状态: 未启动",
								Font:      statusFont,
								TextColor: walk.RGB(100, 100, 100),
							},
							HSpacer{},
							Label{
								AssignTo:  &app.lblRemaining,
								Text:      "剩余时间: --",
								Font:      remainFont,
								TextColor: walk.RGB(0, 120, 215),
							},
						},
					},
					Label{
						AssignTo: &app.lblTarget,
						Text:     "计划关机时间: --",
					},
					Label{
						AssignTo: &app.lblReminderInfo,
						Text:     "提醒时间: --",
					},
				},
			},

			// 4. 快捷延迟操作条
			Composite{
				Layout: HBox{MarginsZero: true, Spacing: 8},
				Children: []Widget{
					Label{Text: "快捷延迟:"},
					PushButton{Text: "+5 分钟", OnClicked: func() { app.postpone(5) }},
					PushButton{Text: "+10 分钟", OnClicked: func() { app.postpone(10) }},
					PushButton{Text: "+30 分钟", OnClicked: func() { app.postpone(30) }},
					PushButton{Text: "+60 分钟", OnClicked: func() { app.postpone(60) }},
				},
			},

			VSpacer{Size: 5},

			// 5. 底部核心控制按钮
			Composite{
				Layout: HBox{MarginsZero: true, Spacing: 10},
				Children: []Widget{
					PushButton{
						AssignTo: &app.btnStart,
						Text:     "🚀 启动定时关机",
						OnClicked: func() {
							app.startSchedule()
						},
					},
					PushButton{
						AssignTo: &app.btnCancel,
						Text:     "🛑 取消定时",
						Enabled:  false,
						OnClicked: func() {
							app.cancelSchedule()
						},
					},
					HSpacer{},
					PushButton{
						Text: "最小化到托盘",
						OnClicked: func() {
							app.mainWindow.SetVisible(false)
							if app.notifyIcon != nil {
								app.notifyIcon.ShowInfo("定时关机助手", "程序已最小化至托盘并在后台守护。")
							}
						},
					},
					PushButton{
						Text: "立即关机",
						OnClicked: func() {
							app.executeImmediate()
						},
					},
				},
			},
		},
	}.Create()

	if err != nil {
		return err
	}

	// 初始化模式单选框选中状态
	if app.config.Mode == "countdown" {
		app.rbCountdownMode.SetChecked(true)
	} else {
		app.rbTimeMode.SetChecked(true)
	}
	app.updateModeUI()

	// 初始化系统托盘图标
	if err := app.setupTrayIcon(); err != nil {
		return err
	}

	// 绑定窗口关闭事件：点击右上角X时最小化到系统托盘，不退出后台任务
	app.mainWindow.Closing().Attach(func(canceled *bool, reason walk.CloseReason) {
		*canceled = true
		app.mainWindow.SetVisible(false)
		if app.notifyIcon != nil {
			app.notifyIcon.ShowInfo("定时关机助手", "程序已最小化到托盘，后台持续运行。双击托盘图标可重新打开。")
		}
	})

	// 绑定调度器事件回调
	app.setupSchedulerCallbacks()

	// 启动后台每秒 Ticker，通过 Synchronize 触发 UI 与调度器更新
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if app.mainWindow != nil {
					app.mainWindow.Synchronize(func() {
						app.scheduler.Tick()
					})
				}
			case <-app.stopTicker:
				return
			}
		}
	}()

	// 若配置了开机自动恢复每日定时，则自动启动定时任务
	if app.config.DailySchedule {
		app.startScheduleQuietly()
	}

	// 确定启动时是否显示窗口
	if startInTray {
		app.mainWindow.SetVisible(false)
		if app.notifyIcon != nil {
			app.notifyIcon.ShowInfo("定时关机助手", "已在后台静默启动。")
		}
	} else {
		app.mainWindow.SetVisible(true)
	}

	app.mainWindow.Run()
	return nil
}

// setupTrayIcon 创建系统托盘图标及右键菜单
func (app *MainWindowApp) setupTrayIcon() error {
	ni, err := walk.NewNotifyIcon(app.mainWindow)
	if err != nil {
		return err
	}
	app.notifyIcon = ni

	if app.appIcon != nil {
		_ = ni.SetIcon(app.appIcon)
	} else {
		_ = ni.SetIcon(walk.IconInformation())
	}
	_ = ni.SetToolTip("Windows 定时关机助手")
	_ = ni.SetVisible(true)

	// 左键点击托盘图标恢复主窗口
	ni.MouseDown().Attach(func(x, y int, button walk.MouseButton) {
		if button == walk.LeftButton {
			app.mainWindow.SetVisible(true)
			win.SetForegroundWindow(app.mainWindow.Handle())
		}
	})

	// 右键菜单
	ctxMenu := ni.ContextMenu()

	// 1. 状态展示项 (只读)
	app.trayStatusAction = walk.NewAction()
	app.trayStatusAction.SetText("状态: 未启动")
	app.trayStatusAction.SetEnabled(false)
	ctxMenu.Actions().Add(app.trayStatusAction)

	ctxMenu.Actions().Add(walk.NewSeparatorAction())

	// 2. 显示主界面
	showAction := walk.NewAction()
	showAction.SetText("显示主界面")
	showAction.Triggered().Attach(func() {
		app.mainWindow.SetVisible(true)
		win.SetForegroundWindow(app.mainWindow.Handle())
	})
	ctxMenu.Actions().Add(showAction)

	// 3. 快捷延迟菜单
	delay5Action := walk.NewAction()
	delay5Action.SetText("⏱️ 延迟 5 分钟")
	delay5Action.Triggered().Attach(func() { app.postpone(5) })
	ctxMenu.Actions().Add(delay5Action)

	delay10Action := walk.NewAction()
	delay10Action.SetText("⏱️ 延迟 10 分钟")
	delay10Action.Triggered().Attach(func() { app.postpone(10) })
	ctxMenu.Actions().Add(delay10Action)

	delay30Action := walk.NewAction()
	delay30Action.SetText("⏱️ 延迟 30 分钟")
	delay30Action.Triggered().Attach(func() { app.postpone(30) })
	ctxMenu.Actions().Add(delay30Action)

	// 4. 取消定时关机
	cancelAction := walk.NewAction()
	cancelAction.SetText("🛑 取消定时关机")
	cancelAction.Triggered().Attach(func() { app.cancelSchedule() })
	ctxMenu.Actions().Add(cancelAction)

	ctxMenu.Actions().Add(walk.NewSeparatorAction())

	// 5. 立即关机
	shutAction := walk.NewAction()
	shutAction.SetText("⚡ 立即关机")
	shutAction.Triggered().Attach(func() { app.executeImmediate() })
	ctxMenu.Actions().Add(shutAction)

	// 6. 退出程序
	exitAction := walk.NewAction()
	exitAction.SetText("🚪 彻底退出")
	exitAction.Triggered().Attach(func() {
		app.cleanupAndExit()
	})
	ctxMenu.Actions().Add(exitAction)

	return nil
}

// setupSchedulerCallbacks 绑定调度器事件
func (app *MainWindowApp) setupSchedulerCallbacks() {
	app.scheduler.SetCallbacks(
		// onTick
		func(remaining time.Duration, status ScheduleStatus) {
			if app.mainWindow == nil {
				return
			}
			// 同步刷新可能正在弹出的提醒对话框
			UpdateReminderDialogOnTick(app.scheduler)

			if status == StatusRunning {
				_ = app.lblStatus.SetText("状态: ⏰ 定时运行中")
				app.lblStatus.SetTextColor(walk.RGB(0, 150, 0))
				_ = app.lblRemaining.SetText(fmt.Sprintf("剩余: %s", FormatDuration(remaining)))

				_, target, _, _, _ := app.scheduler.Status()
				_ = app.lblTarget.SetText(fmt.Sprintf("计划关机时间: %s", target.Format("2006-01-02 15:04:05")))

				adv := int(app.neAdvanceMin.Value())
				remindTime := target.Add(-time.Duration(adv) * time.Minute)
				_ = app.lblReminderInfo.SetText(fmt.Sprintf("预计提醒时间: %s (提前 %d 分钟)", remindTime.Format("15:04:05"), adv))

				if app.btnStart != nil {
					app.btnStart.SetEnabled(false)
				}
				if app.btnCancel != nil {
					app.btnCancel.SetEnabled(true)
				}

				if app.notifyIcon != nil {
					tooltip := fmt.Sprintf("定时关机: 剩余 %s", FormatDuration(remaining))
					if len(tooltip) > 63 {
						tooltip = tooltip[:63]
					}
					_ = app.notifyIcon.SetToolTip(tooltip)
				}
				if app.trayStatusAction != nil {
					_ = app.trayStatusAction.SetText(fmt.Sprintf("已定时: %s (剩余 %s)", target.Format("15:04"), FormatDuration(remaining)))
				}
			} else {
				_ = app.lblStatus.SetText("状态: 未启动")
				app.lblStatus.SetTextColor(walk.RGB(100, 100, 100))
				_ = app.lblRemaining.SetText("剩余时间: --")
				_ = app.lblTarget.SetText("计划关机时间: --")
				_ = app.lblReminderInfo.SetText("提醒时间: --")

				if app.btnStart != nil {
					app.btnStart.SetEnabled(true)
				}
				if app.btnCancel != nil {
					app.btnCancel.SetEnabled(false)
				}

				if app.notifyIcon != nil {
					_ = app.notifyIcon.SetToolTip("Windows 定时关机助手 (未启动)")
				}
				if app.trayStatusAction != nil {
					_ = app.trayStatusAction.SetText("状态: 未启动")
				}
			}
		},
		// onReminderTriggered: 弹窗提醒用户！
		func(remaining time.Duration) {
			// 获取最新配置
			app.saveCurrentConfig()
			ShowReminderDialog(app.mainWindow, app.scheduler, app.config, app.notifyIcon, app.appIcon)
		},
		// onActionExecuted: 关机/重启指令发出后，安全退出自身，避免拦截或阻挡系统关机
		func(action string) {
			if app.notifyIcon != nil {
				app.notifyIcon.ShowInfo("定时关机", "强制关机指令已发出...")
			}
			if action == "shutdown" || action == "restart" {
				go func() {
					time.Sleep(500 * time.Millisecond)
					if app.mainWindow != nil {
						app.mainWindow.Synchronize(func() {
							app.cleanupAndExit()
						})
					}
				}()
			}
		},
		// onCancelled
		func() {
			if app.notifyIcon != nil {
				app.notifyIcon.ShowInfo("定时关机", "定时关机任务已取消")
			}
		},
	)
}

func (app *MainWindowApp) updateModeUI() {
	isTimeMode := app.rbTimeMode.Checked()
	app.neTargetHour.SetEnabled(isTimeMode)
	app.neTargetMinute.SetEnabled(isTimeMode)
	app.cbDaily.SetEnabled(isTimeMode)

	app.neCountHour.SetEnabled(!isTimeMode)
	app.neCountMin.SetEnabled(!isTimeMode)
}

func (app *MainWindowApp) setTime(hour, minute int) {
	app.rbTimeMode.SetChecked(true)
	app.updateModeUI()
	_ = app.neTargetHour.SetValue(float64(hour))
	_ = app.neTargetMinute.SetValue(float64(minute))
}

func (app *MainWindowApp) setCountdown(hour, minute int) {
	app.rbCountdownMode.SetChecked(true)
	app.updateModeUI()
	_ = app.neCountHour.SetValue(float64(hour))
	_ = app.neCountMin.SetValue(float64(minute))
}

func (app *MainWindowApp) postpone(minutes int) {
	status, _, _, _, _ := app.scheduler.Status()
	if status != StatusRunning {
		walk.MsgBox(app.mainWindow, "提示", "当前没有正在运行的定时关机任务", walk.MsgBoxIconInformation)
		return
	}
	newTarget := app.scheduler.Postpone(minutes)
	if app.notifyIcon != nil {
		app.notifyIcon.ShowInfo("关机已延迟", fmt.Sprintf("已成功延迟 %d 分钟，新关机时间: %s", minutes, newTarget.Format("15:04:05")))
	}
}

func (app *MainWindowApp) startSchedule() {
	app.saveCurrentConfig()

	actionMap := []string{"shutdown", "restart", "sleep"}
	actionIdx := app.cbAction.CurrentIndex()
	action := "shutdown"
	if actionIdx >= 0 && actionIdx < len(actionMap) {
		action = actionMap[actionIdx]
	}

	advMin := int(app.neAdvanceMin.Value())
	if advMin <= 0 {
		advMin = 5
	}

	if app.rbTimeMode.Checked() {
		hour := int(app.neTargetHour.Value())
		min := int(app.neTargetMinute.Value())
		daily := app.cbDaily.Checked()

		target := app.scheduler.StartScheduleTime(hour, min, advMin, action, daily)
		if app.notifyIcon != nil {
			app.notifyIcon.ShowInfo("定时关机已启动", fmt.Sprintf("将在 %s 执行关机，提前 %d 分钟提醒", target.Format("2006-01-02 15:04:05"), advMin))
		}
	} else {
		hour := int(app.neCountHour.Value())
		min := int(app.neCountMin.Value())
		totalDur := time.Duration(hour)*time.Hour + time.Duration(min)*time.Minute
		if totalDur <= 0 {
			walk.MsgBox(app.mainWindow, "提示", "请设置大于0的倒计时时长", walk.MsgBoxIconWarning)
			return
		}

		target := app.scheduler.StartCountdown(totalDur, advMin, action)
		if app.notifyIcon != nil {
			app.notifyIcon.ShowInfo("倒计时关机已启动", fmt.Sprintf("将在 %s 执行关机，提前 %d 分钟提醒", target.Format("2006-01-02 15:04:05"), advMin))
		}
	}
}

func (app *MainWindowApp) startScheduleQuietly() {
	action := app.config.Action
	advMin := app.config.AdvanceReminderMinutes
	if advMin <= 0 {
		advMin = 5
	}
	target := app.scheduler.StartScheduleTime(app.config.TargetHour, app.config.TargetMinute, advMin, action, true)
	if app.notifyIcon != nil {
		app.notifyIcon.ShowInfo("每日定时关机已恢复", fmt.Sprintf("计划将于 %s 关机 (提前 %d 分钟提醒)", target.Format("15:04:05"), advMin))
	}
}

func (app *MainWindowApp) cancelSchedule() {
	status, _, _, _, _ := app.scheduler.Status()
	if status != StatusRunning {
		return
	}
	app.scheduler.Cancel()
}

func (app *MainWindowApp) executeImmediate() {
	if walk.MsgBox(app.mainWindow, "确认立即关机", "您确定要立即关闭计算机吗？未保存的数据可能会丢失。", walk.MsgBoxIconQuestion|walk.MsgBoxYesNo) == win.IDYES {
		actionMap := []string{"shutdown", "restart", "sleep"}
		actionIdx := app.cbAction.CurrentIndex()
		action := "shutdown"
		if actionIdx >= 0 && actionIdx < len(actionMap) {
			action = actionMap[actionIdx]
		}
		app.scheduler.ExecuteNow(action)
	}
}

func (app *MainWindowApp) saveCurrentConfig() {
	if app.rbTimeMode == nil {
		return
	}
	if app.rbTimeMode.Checked() {
		app.config.Mode = "time"
	} else {
		app.config.Mode = "countdown"
	}
	app.config.TargetHour = int(app.neTargetHour.Value())
	app.config.TargetMinute = int(app.neTargetMinute.Value())
	app.config.DailySchedule = app.cbDaily.Checked()
	app.config.CountdownHours = int(app.neCountHour.Value())
	app.config.CountdownMinutes = int(app.neCountMin.Value())
	app.config.AdvanceReminderMinutes = int(app.neAdvanceMin.Value())
	app.config.PlaySound = app.cbPlaySound.Checked()
	app.config.AutoStart = app.cbAutoStart.Checked()

	actionMap := []string{"shutdown", "restart", "sleep"}
	actionIdx := app.cbAction.CurrentIndex()
	if actionIdx >= 0 && actionIdx < len(actionMap) {
		app.config.Action = actionMap[actionIdx]
	}

	_ = app.config.SaveConfig()
}

func (app *MainWindowApp) cleanupAndExit() {
	app.saveCurrentConfig()
	close(app.stopTicker)
	if app.notifyIcon != nil {
		app.notifyIcon.Dispose()
	}
	if app.mainWindow != nil {
		app.mainWindow.Dispose()
	}
	walk.App().Exit(0)
}

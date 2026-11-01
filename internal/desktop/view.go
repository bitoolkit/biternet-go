package desktop

import (
	"fmt"
	"math"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func (a *app) view(c *ui.Context) {
	dark := c.Theme().Dark
	if a.settings.Theme != "system" {
		dark = a.settings.Theme == "dark"
	}

	theme := appTheme(dark)
	c.SetTheme(theme)

	ui.Column(c).Fill().
		Background(theme.Background).
		BorderWidth(1).BorderColor(windowBorder(theme)).Radius(10).Clip().
		Font(theme.Font).TextColor(theme.Text).
		Children(func() {
			a.header(c)

			ui.Column(c).Grow(1).MinHeight(0).FillWidth().Padding(10, 18, 16).Children(func() {
				ui.Column(c.Key(a.page)).Grow(1).MinHeight(0).FillWidth().Children(func() {
					switch a.page {
					case "update":
						a.updateView(c)
					case "settings":
						a.preferences(c)
					default:
						if a.state.Online() {
							a.online(c)
						} else {
							a.loginForm(c)
						}
					}
				})

				a.feedback(c)

				if a.page == "" {
					a.sessionAction(c)
				}
			})

			ui.Row(c).FillWidth().Height(34).Shrink(0).PaddingX(18).
				BorderWidth(1, 0, 0, 0).BorderColor(hairline(theme)).
				Children(func() {
					if textButton(c, "BITERNET v"+Version()).Label("打开项目主页").Tooltip("打开项目主页").Clicked() {
						a.openLink(repositoryURL)
					}

					ui.Box(c).Grow(1)

					if a.update != nil && textButton(c, "有更新 ↗").Clicked() {
						a.page = "update"
						a.message = ""
						a.redraw()
					}
				})
		})

	c.OnShortcut(0, ui.KeyEscape, func() {
		if a.page != "" {
			a.page = ""
			a.redraw()
		} else {
			a.win.Close()
		}
	})

	c.OnShortcut(ui.Cmd, ui.KeyQ, mygo.App.Quit)

	c.OnShortcut(ui.Cmd, ui.KeyComma, func() {
		if a.page == "settings" {
			a.page = ""
		} else {
			a.page = "settings"
		}
		a.redraw()
	})
}

func (a *app) header(c *ui.Context) {
	theme := c.Theme()

	if a.page != "" {
		title := "偏好设置"
		if a.page == "update" {
			title = "版本更新"
		}

		ui.Row(c.Key("page-header")).FillWidth().Height(36).Shrink(0).PaddingX(8).Gap(8).
			Background(theme.Surface).BorderWidth(0, 0, 1, 0).BorderColor(hairline(theme)).
			DragWindow().
			Children(func() {
				back := ui.ButtonBase(c.Key("back")).Label("返回").
					Height(28).PaddingX(6).Gap(4).Radius(6).TextColor(theme.TextMuted)
				if back.Hovered() {
					back.Background(theme.SurfaceHover).TextColor(theme.Text)
				}
				if back.Pressed() {
					back.Background(theme.SurfacePressed)
				}

				back.Children(func() {
					ui.Icon(c, backIcon).Size(14, 14)
					ui.Text(c, "返回").FontSize(11)
				})
				if back.Clicked() {
					a.page = ""
					c.Invalidate()
				}

				ui.Text(c, title).FontSize(13).FontWeight(600).Grow(1)

				if iconButton(c, closeIcon, "关闭窗口").Clicked() {
					a.win.Close()
				}
			})

		return
	}

	ui.Row(c).FillWidth().Height(36).Shrink(0).Padding(0, 8, 0, 16).Gap(8).DragWindow().Children(func() {
		ui.Image(c, brandImage).Size(19, 19)
		ui.Text(c, "BITERNET").FontSize(11).FontWeight(700).LetterSpacing(.6).Grow(1)

		if iconButton(c, settingsIcon, "偏好设置").Clicked() {
			a.page = "settings"
			c.Invalidate()
		}

		if iconButton(c, closeIcon, "关闭窗口").Clicked() {
			a.win.Close()
		}
	})
}

func iconButton(c *ui.Context, svg *ui.SVG, label string) ui.Element {
	theme := c.Theme()
	button := ui.ButtonBase(c.Key(label)).Label(label).Size(28, 28).Center().Radius(6).TextColor(theme.TextMuted)

	if button.Hovered() {
		button.Background(theme.SurfaceHover).TextColor(theme.Text)
	}
	if button.Pressed() {
		button.Background(theme.SurfacePressed)
	}

	button.Children(func() { ui.Icon(c, svg).Size(15, 15) })

	return button
}

func (a *app) loginForm(c *ui.Context) {
	theme := c.Theme()
	disabled := a.busy != ""

	// Empty inputs and CJK placeholders need the same line height to stay centered.
	ui.Field(c, "用户名", func() {
		input := ui.TextInput(c.Key("username"), &a.username).
			Label("用户名").Placeholder("学号或用户名").
			FillWidth().Height(36).Padding(0, 10).FontSize(13).FixedLineHeight(20).Disabled(disabled)

		if input.Submitted() {
			a.login()
		}
	}).FontSize(11).TextColor(theme.TextMuted).Gap(5).Margin(0, 0, 10)

	ui.Field(c, "密码", func() {
		ui.Box(c).FillWidth().Height(36).Children(func() {
			input := ui.TextInput(c.Key("password"), &a.password).
				Label("密码").Placeholder("统一认证密码").
				FillWidth().Height(36).Padding(0, 36, 0, 10).FontSize(13).FixedLineHeight(20).Disabled(disabled)

			if !a.showPassword {
				input.Password()
			}
			if input.Submitted() {
				a.login()
			}

			svg, label := eyeIcon, "显示密码"
			if a.showPassword {
				svg, label = eyeOffIcon, "隐藏密码"
			}
			if iconButton(c, svg, label).Absolute().Right(4).Top(4).Disabled(disabled).Clicked() {
				a.showPassword = !a.showPassword
			}
		})
	}).FontSize(11).TextColor(theme.TextMuted).Gap(5)

	ui.Row(c).FillWidth().Justify(ui.SpaceBetween).Margin(10, 0, 8).Children(func() {
		if ui.Checkbox(c.Key("remember"), &a.settings.RememberPassword, "记住密码").
			FontSize(11).Disabled(disabled).Changed() {
			if !a.settings.RememberPassword {
				a.settings.AutoLogin = false
			}

			a.saveSettings()
		}

		if ui.Checkbox(c.Key("auto"), &a.settings.AutoLogin, "自动登录").
			FontSize(11).Disabled(disabled).Changed() {
			if a.settings.AutoLogin {
				a.settings.RememberPassword = true
				a.autoPaused = false
			}

			a.saveSettings()
		}
	})
}

func (a *app) sessionAction(c *ui.Context) {
	if a.state.Online() {
		label := "注销"
		if a.busy == "logout" {
			label = "正在注销…"
		}

		if ui.Button(c.Key("logout"), label).Label("注销").
			FillWidth().Height(34).Shrink(0).Margin(8, 0, 0).FontSize(12).
			Disabled(a.busy != "").Clicked() {
			a.logout()
		}

		return
	}

	label := "登录"
	if a.initializing {
		label = "正在检查…"
	} else if a.busy == "login" {
		label = "正在登录…"
	}

	if ui.PrimaryButton(c.Key("login"), label).Label("登录").
		FillWidth().Height(34).Shrink(0).Margin(8, 0, 0).FontSize(12).
		Disabled(a.busy != "" || a.initializing).Clicked() {
		a.login()
	}
}

func (a *app) feedback(c *ui.Context) {
	message := a.message
	if message == "" {
		message = a.backgroundError
	}

	// Reserve two text lines and their top spacing even when there is no error.
	ui.Box(c.Key("feedback")).FillWidth().Height(40).Shrink(0).Padding(6, 0, 0).Children(func() {
		if message != "" {
			ui.Text(c, message).FontSize(11).LineHeight(1.5).TextColor(c.Theme().Danger).
				MaxLines(2).Ellipsis("…").Tooltip(message)
		}
	})
}

func (a *app) online(c *ui.Context) {
	theme := c.Theme()

	ui.Row(c).FillWidth().Justify(ui.SpaceBetween).Children(func() {
		ui.Column(c).Gap(4).Grow(1).Children(func() {
			title := "已连接"
			if a.backgroundError != "" {
				title = "状态待确认"
			}
			ui.Text(c, title).FontSize(18).FontWeight(600)

			name := a.state.Username
			if name == "" {
				name = "未知用户"
			}
			ui.Text(c, name).FontSize(11).TextColor(theme.TextMuted).MaxLines(1).Ellipsis("…")
		})

		if a.backgroundError == "" {
			ui.Box(c).Size(32, 32).Radius(16).Background(theme.SurfaceHover).Center().Children(func() {
				ui.Icon(c, checkIcon).Size(18, 18).TextColor(theme.Text)
			})
		}
	})

	flux, fluxUnit := formatBytes(a.state.UsedBytes)
	duration, durationUnit := formatDuration(a.state.UsedSeconds)

	ui.Row(c).FillWidth().Gap(15).Margin(12, 0, 10).Children(func() {
		metric(c, flux, fluxUnit, "已用流量")
		metric(c, duration, durationUnit, "已用时长")
	})

	ui.Column(c).FillWidth().
		BorderWidth(1, 0, 0, 0).BorderColor(hairline(theme)).
		Padding(8, 0, 0).Gap(6).
		Children(func() {
			detail(c, "IP 地址", a.state.IP)

			balance := "—"
			if a.state.Balance != nil {
				balance = fmt.Sprintf("%.2f 元", *a.state.Balance)
			}
			detail(c, "账户余额", balance)
		})
}

func metric(c *ui.Context, value, unit, label string) {
	ui.Column(c.Key(label)).Grow(1).Basis(0).Gap(5).Children(func() {
		ui.Row(c).Gap(3).AlignItems(ui.Baseline).Children(func() {
			ui.Text(c, value).FontSize(26).FontWeight(500)
			ui.Text(c, unit).FontSize(11).TextColor(c.Theme().TextMuted)
		})

		ui.Text(c, label).FontSize(11).TextColor(c.Theme().TextMuted)
	})
}

func detail(c *ui.Context, label, value string) {
	ui.Row(c.Key(label)).FillWidth().Justify(ui.SpaceBetween).Gap(8).Children(func() {
		ui.Text(c, label).FontSize(11).TextColor(c.Theme().TextMuted)
		ui.Text(c, value).FontSize(11).Grow(1).MinWidth(0).TextAlign(ui.End).
			MaxLines(1).Ellipsis("…").Tooltip(value).Selectable()
	})
}

func (a *app) preferences(c *ui.Context) {
	startup := a.startupSettings()
	managed := a.managedStartup != nil

	settingRow(c, "开机启动", func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			if managed {
				ui.Text(c, "由 Nix 管理").FontSize(10).TextColor(c.Theme().TextMuted)
			}

			if ui.Switch(c.Key("startup"), &startup.AutoStart).
				Label("开机启动").Disabled(managed).Changed() {
				if err := mygo.App.SetOpenAtLogin(startup.AutoStart); err != nil {
					a.message = "无法修改开机启动：" + err.Error()
				} else {
					a.autoStart = startup.AutoStart
					if !a.autoStart {
						a.settings.SilentStart = false
					}

					a.message = ""
					a.saveSettings()
				}

				a.redraw()
			}
		})
	})

	settingRow(c, "启动时隐藏窗口", func() {
		if ui.Switch(c.Key("silent"), &startup.SilentStart).
			Label("启动时隐藏窗口").Disabled(managed || !startup.AutoStart).Changed() {
			a.settings.SilentStart = startup.SilentStart
			a.saveSettings()
		}
	})

	settingRow(c, "外观", func() {
		selected := map[string]string{
			"system": "跟随系统",
			"light":  "浅色",
			"dark":   "深色",
		}[a.settings.Theme]

		if ui.Select(c, &selected, []string{"跟随系统", "浅色", "深色"}).
			Label("外观").Width(104).FontSize(11).Changed() {
			a.settings.Theme = map[string]string{
				"跟随系统": "system",
				"浅色":   "light",
				"深色":   "dark",
			}[selected]
			mygo.Theme.SetSource(mygo.ThemeSource(a.settings.Theme))

			a.saveSettings()
			a.redraw()
		}
	})
}

func settingRow(c *ui.Context, label string, control func()) {
	ui.Row(c.Key(label)).FillWidth().Height(46).Justify(ui.SpaceBetween).
		BorderWidth(0, 0, 1, 0).BorderColor(hairline(c.Theme())).
		Children(func() {
			ui.Text(c, label).FontSize(12)
			control()
		})
}

func textButton(c *ui.Context, label string) ui.Element {
	button := ui.ButtonBase(c.Key(label)).Label(label).Height(28).
		TextColor(c.Theme().TextMuted).Cursor(ui.CursorPointer)
	if button.Hovered() {
		button.TextColor(c.Theme().Text)
	}

	button.Children(func() { ui.Text(c, label).FontSize(11) })

	return button
}

func (a *app) updateView(c *ui.Context) {
	theme := c.Theme()

	ui.Text(c, "发现新版本").FontSize(18).FontWeight(600).Margin(0, 0, 12)
	ui.Text(c, "v"+a.update.Version).FontSize(26).FontWeight(500).MaxLines(1).Ellipsis("…").Tooltip(a.update.Version)
	ui.Text(c, "当前版本 v"+Version()).FontSize(11).TextColor(theme.TextMuted).Margin(4, 0, 12)
	ui.Text(c, "前往 GitHub 下载并安装更新。").FontSize(12).TextColor(theme.TextMuted)

	ui.Box(c).Grow(1)

	ui.Row(c).FillWidth().Gap(8).Children(func() {
		if ui.Button(c, "稍后").Width(66).Height(34).FontSize(12).Clicked() {
			a.page = ""
			a.redraw()
		}

		if ui.PrimaryButton(c, "前往下载").Grow(1).Height(34).FontSize(12).Clicked() {
			a.openLink(a.update.URL)
		}
	})
}

func formatBytes(value *uint64) (string, string) {
	if value == nil {
		return "—", ""
	}

	amount := float64(*value)
	units := []string{"B", "KB", "MB", "GB", "TB"}
	unitIndex := 0
	for amount >= 1024 && unitIndex < len(units)-1 {
		amount /= 1024
		unitIndex++
	}

	if unitIndex == 0 {
		return fmt.Sprint(*value), units[unitIndex]
	}

	return fmt.Sprintf("%.1f", amount), units[unitIndex]
}

func formatDuration(value *int64) (string, string) {
	if value == nil {
		return "—", ""
	}

	seconds := max(*value, 0)
	if seconds >= 3600 {
		return fmt.Sprintf("%.1f", math.Floor(float64(seconds)/360)/10), "小时"
	}
	if seconds >= 60 {
		return fmt.Sprint(seconds / 60), "分钟"
	}

	return fmt.Sprint(seconds), "秒"
}

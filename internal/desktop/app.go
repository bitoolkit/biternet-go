package desktop

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bitoolkit/biternet-go/client"
	"github.com/bitoolkit/biternet-go/internal/config"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func Version() string {
	if version := mygo.App.Version(); version != "" {
		return version
	}
	return "0.0.0"
}

const (
	windowWidth  = 280
	windowHeight = 360
)

type request struct {
	action   string
	username string
	password string
}

type app struct {
	win  *mygo.Window
	tray *mygo.Tray

	client   *client.Client
	ctx      context.Context
	cancel   context.CancelFunc
	requests chan request

	settings   config.Settings
	configPath string
	username   string
	password   string

	state         client.State
	initializing  bool
	busy          string
	autoPaused    bool
	nextAutoLogin time.Time

	page           string
	showPassword   bool
	autoStart      bool
	managedStartup *config.Startup
	quitting       bool
	update         *release

	message         string
	backgroundError string
}

func Run(silent bool) error {
	mygo.App.SetName("BITERNET")

	if !mygo.App.RequestSingleInstanceLock() {
		return nil
	}
	if err := registerFonts(); err != nil {
		return err
	}

	path, err := config.Path()
	if err != nil {
		return err
	}
	settings, err := config.Load(path)
	if err != nil {
		return err
	}
	startup, err := config.NixStartup()
	if err != nil {
		return err
	}

	gateway, _ := client.New(client.Config{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	a := &app{
		client:         gateway,
		ctx:            ctx,
		cancel:         cancel,
		requests:       make(chan request, 1),
		settings:       settings,
		configPath:     path,
		username:       settings.Username,
		password:       settings.Password,
		initializing:   true,
		managedStartup: startup,
	}

	mygo.App.OnSecondInstance(func(_ []string, _ string) { a.show() })
	mygo.App.OnActivate(func(bool) { a.show() })
	mygo.App.OnBeforeQuit(func(*mygo.QuitEvent) {
		a.quitting = true
		a.cancel()
	})
	mygo.App.OnWillQuit(func(*mygo.QuitEvent) {
		if a.tray != nil {
			a.tray.Destroy()
		}
	})

	mygo.App.WhenReady(func() {
		mygo.Theme.SetSource(mygo.ThemeSource(settings.Theme))
		if a.managedStartup == nil {
			a.autoStart = mygo.App.OpenAtLogin()
		}

		a.win = mygo.NewWindow(mygo.WindowOptions{
			Title:             "BITERNET",
			Width:             windowWidth,
			Height:            windowHeight,
			MinWidth:          windowWidth,
			MinHeight:         windowHeight,
			UseContentSize:    true,
			Hidden:            true,
			TitleBarStyle:     mygo.TitleBarHidden,
			TitleBarHeight:    36,
			HideWindowButtons: true,
			DisableResize:     true,
			DisableMaximize:   true,
			DisableFullScreen: true,
			CornerRadius:      10,
			BackgroundColor:   "light-dark(#fafafa, #222222)",
			Content:           ui.View(a.view),
		})
		a.win.OnClose(func(e *mygo.CloseEvent) {
			if !a.quitting && a.tray != nil {
				e.PreventDefault()
				a.win.Hide()
			}
		})

		if a.managedStartup == nil && a.autoStart {
			if err := mygo.App.SetOpenAtLogin(true); err != nil {
				a.message = "无法更新开机启动项，请在偏好设置中重新启用"
			}
		}

		menu := mygo.NewMenu([]*mygo.MenuItem{
			{Label: "显示窗口", Click: func(*mygo.MenuItem, *mygo.Window) { a.show() }},
			{
				Label: "偏好设置",
				Click: func(*mygo.MenuItem, *mygo.Window) {
					a.page = "settings"
					a.show()
					a.redraw()
				},
			},
			mygo.Separator(),
			{Label: "退出程序", Click: func(*mygo.MenuItem, *mygo.Window) { mygo.App.Quit() }},
		})
		tray, err := mygo.NewTray(mygo.TrayOptions{
			ID:      "biternet-go",
			Icon:    trayIcon,
			ToolTip: "BITERNET",
			Menu:    menu,
		})
		if err != nil {
			a.message = "系统托盘不可用，关闭窗口将退出程序"
		} else {
			a.tray = tray
			a.tray.OnClick(a.show)
		}

		hidden := (silent || (mygo.App.WasOpenedAtLogin() && a.startupSettings().SilentStart)) && a.tray != nil
		a.redraw()
		if !hidden {
			a.show()
		}

		go a.runNetworkWorker()
		go a.checkUpdates()
	})

	return mygo.App.Run()
}

func (a *app) startupSettings() config.Startup {
	if a.managedStartup != nil {
		return *a.managedStartup
	}
	return config.Startup{AutoStart: a.autoStart, SilentStart: a.settings.SilentStart}
}

func (a *app) show() {
	if a.win != nil {
		a.win.Show()
		a.win.Restore()
		a.win.Focus()
	}
}

func (a *app) redraw() {
	if a.win == nil {
		return
	}

	a.win.Invalidate()
}

func (a *app) saveSettings() bool {
	a.settings.Username = a.username
	a.settings.Password = a.password

	if err := config.Save(a.configPath, a.settings); err != nil {
		a.message = err.Error()
		a.redraw()
		return false
	}

	return true
}

func (a *app) login() {
	if a.busy != "" || a.initializing {
		return
	}

	if a.username == "" || a.password == "" {
		a.message = "请填写用户名和密码"
		a.redraw()
		return
	}

	if !a.saveSettings() {
		return
	}

	a.autoPaused = false
	a.message = ""
	a.backgroundError = ""
	a.busy = "login"
	a.requests <- request{action: "login", username: a.username, password: a.password}
	a.redraw()
}

func (a *app) logout() {
	if a.busy != "" {
		return
	}

	a.autoPaused = true
	a.message = ""
	a.backgroundError = ""
	a.busy = "logout"
	a.requests <- request{action: "logout", username: a.state.Username}
	a.redraw()
}

// One worker owns all gateway I/O. Poll results cannot overwrite an in-flight
// user action; UI changes are posted back onto MyGo's main thread.
func (a *app) runNetworkWorker() {
	check := func() {
		state, err := a.client.Status(a.ctx)

		a.win.Update(func() { a.statusChecked(state, err) })
	}
	check()

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-a.ctx.Done():
			return

		case <-ticker.C:
			if len(a.requests) == 0 {
				check()
			}

		case req := <-a.requests:
			ctx, cancel := context.WithTimeout(a.ctx, 20*time.Second)
			state, err := changeSession(ctx, a.client, req)
			cancel()

			a.win.Update(func() { a.sessionChanged(req.action, state, err) })
		}
	}
}

func (a *app) statusChecked(state client.State, err error) {
	if a.quitting || a.busy != "" {
		return
	}

	a.initializing = false
	if err != nil {
		a.backgroundError = "无法刷新校园网状态，请检查网络连接"
		if a.state.Online() {
			a.backgroundError = "无法刷新校园网状态，当前显示上次数据"
		}
	} else {
		a.backgroundError = ""
		a.state = state

		if !state.Online() && a.settings.AutoLogin && !a.autoPaused &&
			a.username != "" && a.password != "" &&
			!time.Now().Before(a.nextAutoLogin) {
			a.login()
		}
	}

	a.redraw()
}

func (a *app) sessionChanged(action string, state client.State, err error) {
	if a.quitting {
		return
	}

	a.busy = ""
	if err != nil {
		a.message = actionError(action, err)
		a.nextAutoLogin = time.Now().Add(30 * time.Second)

		var gatewayError *client.GatewayError
		if errors.As(err, &gatewayError) || errors.Is(err, client.ErrAlreadyOnline) {
			a.autoPaused = true
		}
	} else {
		a.state = state
		a.message = ""
		a.backgroundError = ""
	}

	a.redraw()
}

func actionError(action string, err error) string {
	prefix := "登录失败"
	if action == "logout" {
		prefix = "注销失败"
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return prefix + "：请求超时，请重试"
	}
	if errors.Is(err, client.ErrAlreadyOnline) {
		return "已有其他账户在线，请先注销"
	}

	var gateway *client.GatewayError
	if errors.As(err, &gateway) {
		return prefix + "：" + gateway.Error()
	}

	return fmt.Sprintf("%s，请检查校园网连接后重试", prefix)
}

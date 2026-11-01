package desktop

import (
	"context"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bitoolkit/biternet-go/client"
	"github.com/bitoolkit/biternet-go/internal/config"
	"github.com/egoist/mygo/ui"
)

func testApp(t *testing.T) *app {
	t.Helper()
	if err := registerFonts(); err != nil {
		t.Fatal(err)
	}
	return &app{settings: config.Settings{Theme: "system"}, configPath: filepath.Join(t.TempDir(), "settings.json"), requests: make(chan request, 1)}
}

func click(t *testing.T, tt *ui.Tester, label string) {
	t.Helper()
	if err := tt.Click(label); err != nil {
		t.Fatal(err)
	}
}

func snapshot(t *testing.T, tt *ui.Tester, name string) {
	t.Helper()
	dir := os.Getenv("BITERNET_SCREENSHOTS")
	if dir == "" {
		return
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	file, err := os.Create(filepath.Join(dir, name+".png"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if err := png.Encode(file, tt.Image()); err != nil {
		t.Fatal(err)
	}
}

func TestLoginKeyboardAndRemember(t *testing.T) {
	a := testApp(t)
	tt := ui.NewTester(a.view, windowWidth, windowHeight)
	click(t, tt, "登录")
	if !tt.HasText("请填写用户名和密码") {
		t.Fatal("missing credential error")
	}
	click(t, tt, "用户名")
	tt.Type("20230001")
	click(t, tt, "密码")
	tt.Type(" secret ")
	click(t, tt, "自动登录")
	if !a.settings.RememberPassword || !a.settings.AutoLogin {
		t.Fatal("automatic login must remember credentials")
	}
	click(t, tt, "密码")
	tt.Key(0, ui.KeyEnter)
	select {
	case req := <-a.requests:
		if req.action != "login" || req.username != "20230001" || req.password != " secret " {
			t.Fatalf("altered credentials: %#v", req)
		}
	default:
		t.Fatal("Enter did not request login")
	}
	if !tt.HasText("正在登录…") {
		t.Fatal("missing busy state")
	}
	click(t, tt, "登录")
	if len(a.requests) != 0 {
		t.Fatal("submitted duplicate login while busy")
	}
	stored, err := config.Load(a.configPath)
	if err != nil || stored.Password != " secret " {
		t.Fatalf("credentials were not saved: %v", err)
	}
}

func TestNixStartupSettingsAreReadOnly(t *testing.T) {
	for _, theme := range []string{"light", "dark"} {
		for _, enabled := range []bool{false, true} {
			name := theme + "-nix-disabled"
			if enabled {
				name = theme + "-nix-enabled"
			}
			t.Run(name, func(t *testing.T) {
				a := testApp(t)
				a.settings.Theme = theme
				a.page = "settings"
				a.autoStart = !enabled
				a.settings.SilentStart = !enabled
				a.managedStartup = &config.Startup{AutoStart: enabled, SilentStart: enabled}
				want := *a.managedStartup

				tt := ui.NewTester(a.view, windowWidth, windowHeight)
				if !tt.HasText("由 Nix 管理") || a.startupSettings() != want {
					t.Fatal("Nix startup settings were not shown")
				}

				click(t, tt, "开机启动")
				click(t, tt, "启动时隐藏窗口")
				if a.startupSettings() != want || a.autoStart != !enabled || a.settings.SilentStart != !enabled {
					t.Fatal("clicking a managed switch changed startup settings")
				}
				if _, err := os.Stat(a.configPath); !os.IsNotExist(err) {
					t.Fatalf("managed switches wrote local settings: %v", err)
				}
				if a.message != "" {
					t.Fatalf("managed switches attempted an update: %s", a.message)
				}
				snapshot(t, tt, name)
			})
		}
	}
}

func TestThemesAndViews(t *testing.T) {
	for _, theme := range []string{"light", "dark"} {
		t.Run(theme, func(t *testing.T) {
			a := testApp(t)
			a.settings.Theme = theme
			a.username = "2024000000"
			a.password = "demo-password"
			a.settings.RememberPassword = true
			a.settings.AutoLogin = true
			tt := ui.NewTester(a.view, windowWidth, windowHeight)
			if !tt.HasText("BITERNET v0.0.0") {
				t.Fatal("missing version")
			}
			if tt.HasText("网络诊断") {
				t.Fatal("diagnostics should not be present")
			}
			for _, label := range []string{"用户名", "密码", "登录", "偏好设置", "关闭窗口"} {
				r, ok := tt.Find(label)
				if !ok || r.X < 0 || r.Y < 0 || r.X+r.W > windowWidth+.1 || r.Y+r.H > windowHeight+.1 {
					t.Fatalf("%s is outside window: %+v", label, r)
				}
			}
			snapshot(t, tt, theme+"-login")
			click(t, tt, "显示密码")
			if !a.showPassword {
				t.Fatal("password toggle did not change")
			}
			click(t, tt, "记住密码")
			if a.settings.RememberPassword || a.settings.AutoLogin {
				t.Fatal("forget must disable auto login")
			}
			click(t, tt, "偏好设置")
			if !tt.HasText("启动时隐藏窗口") {
				t.Fatal("settings did not open")
			}
			snapshot(t, tt, theme+"-settings")
			for _, label := range []string{"返回", "关闭窗口"} {
				r, ok := tt.Find(label)
				if !ok || r.Y < 0 || r.Y+r.H > 36.1 {
					t.Fatalf("%s missing from page header: %+v", label, r)
				}
			}
			click(t, tt, "返回")
			if a.page != "" || !tt.HasText("用户名") {
				t.Fatal("header back did not immediately return to login")
			}
			click(t, tt, "偏好设置")
			tt.Key(0, ui.KeyEscape)
			if a.page != "" {
				t.Fatal("Escape did not return to login")
			}
			bytes := uint64(13743895347)
			seconds := int64(129600)
			balance := 48.5
			a.state = client.State{Error: "ok", Username: "2024000000", IP: "10.42.16.128", UsedBytes: &bytes, UsedSeconds: &seconds, Balance: &balance}
			tt.Frame()
			for _, text := range []string{"已连接", "10.42.16.128", "48.50 元", "BITERNET v0.0.0"} {
				if !tt.HasText(text) {
					t.Fatalf("missing online value %s", text)
				}
			}
			snapshot(t, tt, theme+"-online")
			footer, _ := tt.Find("打开项目主页")
			click(t, tt, "偏好设置")
			if !tt.HasText("外观") || tt.HasText("已连接") {
				t.Fatal("settings did not replace online view immediately")
			}
			settingsFooter, _ := tt.Find("打开项目主页")
			if footer != settingsFooter {
				t.Fatalf("footer moved entering settings: %+v -> %+v", footer, settingsFooter)
			}
			click(t, tt, "返回")
			if !tt.HasText("已连接") {
				t.Fatal("back lost online view")
			}
			a.update = &release{Version: "1.2.0", URL: repositoryURL + "/releases/tag/v1.2.0"}
			tt.Frame()
			click(t, tt, "有更新 ↗")
			if !tt.HasText("发现新版本") || !tt.HasText("v1.2.0") {
				t.Fatal("missing update details")
			}
			for _, label := range []string{"稍后", "前往下载", "打开项目主页"} {
				r, ok := tt.Find(label)
				if !ok || r.X < 0 || r.Y < 0 || r.X+r.W > windowWidth+.1 || r.Y+r.H > windowHeight+.1 {
					t.Fatalf("%s outside update window: %+v", label, r)
				}
			}
			snapshot(t, tt, theme+"-update")
			click(t, tt, "稍后")
			if a.page != "" || !a.state.Online() {
				t.Fatal("dismissing update changed connection")
			}
			tt.Key(ui.Cmd, ui.KeyComma)
			if a.page != "settings" {
				t.Fatal("settings shortcut did not open settings")
			}
			tt.Key(0, ui.KeyEscape)
			click(t, tt, "注销")
			if !a.autoPaused {
				t.Fatal("manual logout must pause automatic login")
			}
			select {
			case req := <-a.requests:
				if req.action != "logout" {
					t.Fatal("wrong logout request")
				}
			default:
				t.Fatal("missing logout request")
			}
		})
	}
}

func TestConnectionLifecycle(t *testing.T) {
	a := testApp(t)
	a.settings.Theme = "dark"
	a.initializing = true
	a.username, a.password = "student", "secret"
	a.settings.AutoLogin, a.settings.RememberPassword = true, true
	tt := ui.NewTester(a.view, windowWidth, windowHeight)
	if !tt.HasText("正在检查…") {
		t.Fatal("missing initial status check")
	}
	click(t, tt, "登录")
	if len(a.requests) != 0 {
		t.Fatal("login during initial status check")
	}
	offline := client.State{Error: "not_online_error", IP: "10.1.2.3"}
	online := client.State{Error: "ok", IP: "10.1.2.3", Username: "student"}
	a.statusChecked(client.State{}, context.DeadlineExceeded)
	if a.initializing || a.backgroundError == "" || len(a.requests) != 0 {
		t.Fatal("query failure triggered login or stayed initializing")
	}
	a.statusChecked(offline, nil)
	if len(a.requests) != 1 {
		t.Fatal("network recovery did not trigger automatic login")
	}
	<-a.requests
	a.statusChecked(online, nil)
	if a.state.Online() {
		t.Fatal("in-flight action was overwritten by polling")
	}
	a.sessionChanged("login", client.State{}, context.DeadlineExceeded)
	a.statusChecked(offline, nil)
	if a.autoPaused || len(a.requests) != 0 {
		t.Fatal("transient failure bypassed retry delay or permanently paused")
	}
	a.nextAutoLogin = time.Time{}
	a.statusChecked(offline, nil)
	if len(a.requests) != 1 {
		t.Fatal("transient failure did not recover after delay")
	}
	<-a.requests
	a.sessionChanged("login", client.State{}, &client.GatewayError{Code: "E2531", Message: "账号或密码错误"})
	a.nextAutoLogin = time.Time{}
	a.statusChecked(offline, nil)
	if !a.autoPaused || len(a.requests) != 0 {
		t.Fatal("rejected credentials retried automatically")
	}
	tt.Frame()
	click(t, tt, "登录")
	if a.autoPaused || len(a.requests) != 1 {
		t.Fatal("manual retry did not resume login")
	}
	<-a.requests
	a.sessionChanged("login", online, nil)
	a.statusChecked(client.State{}, context.DeadlineExceeded)
	tt.Frame()
	if !a.state.Online() || !tt.HasText("状态待确认") || tt.HasText("已连接") {
		t.Fatal("failed status query misrepresented connection")
	}
	footerBefore, _ := tt.Find("打开项目主页")
	statusBefore, _ := tt.Find("状态待确认")
	ipBefore, _ := tt.Find("IP 地址")
	logoutBefore, ok := tt.Find("注销")
	if !ok || logoutBefore.Y+logoutBefore.H > footerBefore.Y {
		t.Fatal("error hid the logout button")
	}
	tt.Scroll(140, 170, 0, 400)
	footerAfter, _ := tt.Find("打开项目主页")
	statusAfter, _ := tt.Find("状态待确认")
	ipAfter, _ := tt.Find("IP 地址")
	logoutAfter, _ := tt.Find("注销")
	if footerBefore != footerAfter || logoutBefore != logoutAfter || statusBefore != statusAfter || ipBefore != ipAfter {
		t.Fatal("mouse wheel moved the window contents")
	}
	errorBounds, ok := tt.Find(a.backgroundError)
	if !ok || errorBounds.Y < 36 || errorBounds.Y+errorBounds.H > logoutAfter.Y {
		t.Fatalf("error is not reachable inside the fixed window: %+v", errorBounds)
	}
	snapshot(t, tt, "dark-status-error")
	a.statusChecked(online, nil)
	tt.Frame()
	click(t, tt, "注销")
	if len(a.requests) != 1 {
		t.Fatal("logout not requested")
	}
	<-a.requests
	a.sessionChanged("logout", offline, nil)
	a.statusChecked(offline, nil)
	if len(a.requests) != 0 || !a.autoPaused {
		t.Fatal("manual logout immediately logged back in")
	}
}

func TestPagesDoNotScroll(t *testing.T) {
	for _, theme := range []string{"light", "dark"} {
		for _, page := range []string{"login", "online", "settings", "update"} {
			for _, withError := range []bool{false, true} {
				name := theme + "-" + page
				if withError {
					name += "-error"
				}

				t.Run(name, func(t *testing.T) {
					a := testApp(t)
					a.settings.Theme = theme
					a.username = "2024000000"
					if page == "settings" || page == "update" {
						a.page = page
					}
					a.update = &release{Version: "1.2.0", URL: repositoryURL + "/releases/tag/v1.2.0"}

					if page == "online" {
						usedBytes, usedSeconds, balance := uint64(5_000_000_000), int64(138600), 10.0
						a.state = client.State{
							Error: "ok", Username: a.username, IP: "10.170.25.175",
							UsedBytes: &usedBytes, UsedSeconds: &usedSeconds, Balance: &balance,
						}
					}
					tt := ui.NewTester(a.view, windowWidth, windowHeight)
					labels := map[string][]string{
						"login":    {"用户名", "密码", "记住密码", "登录"},
						"online":   {"已连接", "已用流量", "IP 地址", "账户余额", "注销"},
						"settings": {"返回", "开机启动", "启动时隐藏窗口", "外观"},
						"update":   {"返回", "发现新版本", "稍后", "前往下载"},
					}[page]
					labels = append(labels, "打开项目主页")

					normalBounds := make(map[string]ui.Rect)
					for _, label := range labels {
						bounds, ok := tt.Find(label)
						if !ok {
							t.Fatalf("missing control: %s", label)
						}
						normalBounds[label] = bounds
					}

					if withError {
						a.message = strings.Repeat("操作未完成，请检查网络连接后重试。", 8)
						tt.Frame()
						for label, before := range normalBounds {
							after, _ := tt.Find(label)
							if before != after {
								t.Fatalf("error moved %s: %+v -> %+v", label, before, after)
							}
						}
						labels = append(labels, a.message)
					}

					for _, label := range labels {
						before, ok := tt.Find(label)
						if !ok || before.X < 0 || before.Y < 0 || before.X+before.W > windowWidth+.1 || before.Y+before.H > windowHeight-34+.1 && label != "打开项目主页" {
							t.Fatalf("%s outside the content area: %+v", label, before)
						}

						tt.Scroll(140, 170, 0, 400)
						after, _ := tt.Find(label)
						if before != after {
							t.Fatalf("wheel moved %s: %+v -> %+v", label, before, after)
						}
						tt.Scroll(140, 170, 0, -400)
					}

					if withError {
						feedback, _ := tt.Find(a.message)
						lastContent := map[string]string{"login": "记住密码", "online": "账户余额", "settings": "外观", "update": "前往下载"}[page]
						content, _ := tt.Find(lastContent)
						if content.Y+content.H > feedback.Y {
							t.Fatalf("error overlaps %s: content=%+v feedback=%+v", lastContent, content, feedback)
						}
					}

					snapshot(t, tt, name)

					if withError {
						a.message = ""
						tt.Frame()
						for label, before := range normalBounds {
							after, _ := tt.Find(label)
							if before != after {
								t.Fatalf("clearing error moved %s: %+v -> %+v", label, before, after)
							}
						}
					}
				})
			}
		}
	}
}

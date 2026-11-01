package desktop

import "github.com/egoist/mygo/ui"

func appTheme(dark bool) *ui.Theme {
	t := ui.LightTheme()
	t.Background = ui.Hex("#fafafa")
	t.Surface = ui.Hex("#ffffff")
	t.SurfaceHover = ui.Hex("#eeeeee")
	t.SurfacePressed = ui.Hex("#e4e4e4")
	t.Border = ui.Hex("#cccccc")
	t.Text = ui.Hex("#292929")
	t.TextMuted = ui.Hex("#6b6b6b")
	t.Accent = ui.Hex("#343434")
	t.AccentHover = ui.Hex("#454545")
	t.AccentPressed = ui.Hex("#242424")
	t.AccentText = ui.Hex("#ffffff")
	t.Danger = ui.Hex("#a84636")
	t.Focus = ui.Hex("#707070")
	t.Selection = ui.Hex("#343434").Alpha(.18)
	if dark {
		t.Dark = true
		t.Background = ui.Hex("#222222")
		t.Surface = ui.Hex("#1b1b1b")
		t.SurfaceHover = ui.Hex("#333333")
		t.SurfacePressed = ui.Hex("#3c3c3c")
		t.Border = ui.Hex("#4a4a4a")
		t.Text = ui.Hex("#e3e3e3")
		t.TextMuted = ui.Hex("#ababab")
		t.Accent = ui.Hex("#bfbfbf")
		t.AccentHover = ui.Hex("#d0d0d0")
		t.AccentPressed = ui.Hex("#ababab")
		t.AccentText = ui.Hex("#222222")
		t.Danger = ui.Hex("#e9a18f")
		t.Focus = ui.Hex("#ababab")
		t.Selection = ui.Hex("#bfbfbf").Alpha(.24)
	}
	t.Radius = 6
	t.Spacing = 3
	t.FontSize = 13
	t.Font = "Manrope, Noto Sans SC"
	return t
}

func windowBorder(t *ui.Theme) ui.Color {
	if t.Dark {
		return ui.Hex("#606060")
	}
	return ui.Hex("#b8b8b8")
}

func hairline(t *ui.Theme) ui.Color {
	if t.Dark {
		return ui.Hex("#383838")
	}
	return ui.Hex("#dedede")
}

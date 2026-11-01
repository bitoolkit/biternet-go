package desktop

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"fmt"
	"io"
	"sync"

	"github.com/egoist/mygo/ui"
)

//go:generate python ../../scripts/fonts.py

//go:embed fonts/Manrope.ttf.gz
var manropeFont []byte

// Noto Sans SC retains GB2312, application text, and all variable weights.
// Other characters use the platform's font fallback.
//
//go:embed fonts/NotoSansSC.ttf.gz
var notoSansSCFont []byte

//go:embed tray.png
var trayIcon []byte

var brandImage = func() *ui.Bitmap {
	img, err := ui.DecodeBitmap(trayIcon)
	if err != nil {
		panic(err)
	}
	return img
}()

var registerFonts = sync.OnceValue(func() error {
	for _, font := range []struct {
		compressed []byte
		family     string
	}{
		{compressed: manropeFont, family: "Manrope"},
		{compressed: notoSansSCFont, family: "Noto Sans SC"},
	} {
		reader, err := gzip.NewReader(bytes.NewReader(font.compressed))
		if err != nil {
			panic(err)
		}
		data, err := io.ReadAll(reader)
		reader.Close()
		if err != nil {
			panic(err)
		}

		if err := ui.RegisterFont(data, font.family); err != nil {
			return fmt.Errorf("加载字体 %s: %w", font.family, err)
		}
	}

	return nil
})

func icon(path string) *ui.SVG {
	return ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.65" stroke-linecap="round" stroke-linejoin="round">` + path + `</svg>`))
}

var (
	closeIcon    = icon(`<path d="m18 6-12 12M6 6l12 12"/>`)
	settingsIcon = icon(`<path d="M20 7h-9m-4 0H4m16 10h-3m-4 0H4"/><circle cx="9" cy="7" r="2"/><circle cx="15" cy="17" r="2"/>`)
	eyeIcon      = icon(`<path d="M2 12s3.5-7 10-7 10 7 10 7-3.5 7-10 7S2 12 2 12Z"/><circle cx="12" cy="12" r="3"/>`)
	eyeOffIcon   = icon(`<path d="m3 3 18 18M10.6 5.1A11 11 0 0 1 12 5c6.5 0 10 7 10 7a17 17 0 0 1-3 4M6 6c-2.6 2-4 6-4 6s3.5 7 10 7a12 12 0 0 0 5-1M10 10a3 3 0 0 0 4 4"/>`)
	checkIcon    = icon(`<path d="m5 12 4 4L19 6"/>`)
	backIcon     = icon(`<path d="m12 19-7-7 7-7m-7 7h14"/>`)
)

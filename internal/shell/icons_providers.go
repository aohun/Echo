//go:build nogui && echo

package shell

import (
	"encoding/base64"
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/egoist/mygo/ui"
	"github.com/yetone/magpie/internal/gui"
	"github.com/yetone/magpie/internal/provider"
)

// A provider row's picture, and the small glyphs beside the rows'.

// pngIcons are the built-in icons that are raster pictures, not SVG
// (magpie's own set, app.js's pngIcons).
var pngIcons = map[string]bool{"crush": true, "zcode": true, "alma": true, "hanako": true, "cindy": true, "typesafe": true, "atomcode": true}

// iconCache memoizes the parsed pictures: an icon drawn again (the window
// moving, a hover) is not read and parsed again.
var iconCache sync.Map // name -> ui.ImageSource

// iconPictureFor is a row icon's picture for any kind of name: the generic
// mark for "" and "generic", a user-stored picture for "file:<name>", the
// built-in's otherwise. nil names the generic mark too: a name the assets
// have none of draws as generic, as magpie's own icon() falls back.
func iconPictureFor(name string) ui.ImageSource {
	if name == "" || name == "generic" {
		return genericIcon()
	}
	if pic, ok := iconCache.Load(name); ok {
		if pic == nil {
			return genericIcon()
		}
		return pic.(ui.ImageSource)
	}
	pic := loadIconPicture(name)
	iconCache.Store(name, pic)
	if pic == nil {
		return genericIcon()
	}
	return pic
}

// loadIconPicture reads and parses one icon, of any of its kinds.
func loadIconPicture(name string) ui.ImageSource {
	if strings.HasPrefix(name, "file:") {
		return storedIconPicture(strings.TrimPrefix(name, "file:"))
	}
	if strings.HasPrefix(name, "data:image/") {
		return dataURIIconPicture(name)
	}
	// the few raster ones end in a name magpie's page asks a .png for
	if pngIcons[name] {
		if bm, ok := readBuiltin(name, ".png"); ok {
			return bm
		}
	}
	if s, ok := readBuiltin(name, ".svg"); ok {
		return s
	}
	if bm, ok := readBuiltin(name, ".png"); ok {
		return bm
	}
	return nil
}

// readBuiltin reads a built-in icon out of the gui package's embedded
// assets and parses it.
func readBuiltin(name, ext string) (ui.ImageSource, bool) {
	f := gui.IconsFS()
	if f == nil {
		return nil, false
	}
	b, err := f.(interface {
		ReadFile(name string) ([]byte, error)
	}).ReadFile(name + ext)
	if err != nil {
		return nil, false
	}
	return parsePicture(name+ext, b)
}

// storedIconPicture loads a user-stored picture (icon "file:<name>") from
// the icons folder beside providers.json, as the editor's /api/icons
// serves it. An icon's own extension is kept in its name.
func storedIconPicture(name string) ui.ImageSource {
	f := provider.IconFile(name)
	if f == "" {
		return nil
	}
	b, err := os.ReadFile(f)
	if err != nil {
		return nil
	}
	pic, ok := parsePicture(name, b)
	if !ok {
		return nil
	}
	return pic
}

// dataURIIconPicture decodes a data:image/... URI (SVG or raster).
func dataURIIconPicture(raw string) ui.ImageSource {
	comma := strings.IndexByte(raw, ',')
	if comma < 0 {
		return nil
	}
	meta := raw[:comma]
	payload := raw[comma+1:]
	var data []byte
	if strings.Contains(meta, ";base64") {
		var err error
		data, err = base64.StdEncoding.DecodeString(payload)
		if err != nil {
			return nil
		}
	} else {
		unescaped, err := url.QueryUnescape(payload)
		if err != nil {
			data = []byte(payload)
		} else {
			data = []byte(unescaped)
		}
	}
	if len(data) == 0 {
		return nil
	}
	if strings.Contains(meta, "svg") {
		if s, err := ui.ParseSVG(data); err == nil {
			return s
		}
	} else {
		if bm, err := ui.DecodeBitmap(data); err == nil {
			return bm
		}
	}
	return nil
}

// parsePicture parses a picture's bytes as what Image draws: an SVG for a
// .svg name, a decoded bitmap for anything else (PNG, JPEG, GIF, WebP, ICO).
func parsePicture(name string, b []byte) (ui.ImageSource, bool) {
	if strings.HasSuffix(name, ".svg") {
		s, err := ui.ParseSVG(b)
		if err != nil {
			return nil, false
		}
		return s, true
	}
	bm, err := ui.DecodeBitmap(b)
	if err != nil {
		return nil, false
	}
	return bm, true
}

// genericIcon is the mark a provider without a picture draws (magpie's own
// generic glyph, a traced cube).
func genericIcon() *ui.SVG {
	genericOnce.Do(func() {
		genericPic = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" width="16" height="16"><g fill="none" stroke="currentColor" stroke-width="1.4" stroke-linejoin="round"><path d="M8 2.2 13.2 5.1v5.8L8 13.8 2.8 10.9V5.1Z"/><path d="M8 8v5.8M2.8 5.1 8 8l5.2-2.9"/></g></svg>`))
	})
	return genericPic
}

var (
	genericOnce sync.Once
	genericPic  *ui.SVG

	chevOnce sync.Once
	chevPic  *ui.SVG
)

// chevRight is the row's trailing arrow (magpie's CHEV_R).
func chevRight() *ui.SVG {
	chevOnce.Do(func() {
		chevPic = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" width="11" height="11"><path d="M6 3.5 10.5 8 6 12.5" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round"/></svg>`))
	})
	return chevPic
}

// chevDown is the same arrow rotated to point down, marking an open,
// expanded section (the Connect and Models folds on the Gateway page).
func chevDown() *ui.SVG {
	chevDownOnce.Do(func() {
		chevDownPic = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" width="11" height="11"><path d="M3.5 6 8 10.5 12.5 6" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round"/></svg>`))
	})
	return chevDownPic
}

var (
	chevDownOnce sync.Once
	chevDownPic  *ui.SVG
)

// closeIcon is the cross in a dialog's corner, which puts the dialog away.
func closeIcon() *ui.SVG {
	closeOnce.Do(func() {
		closePic = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" width="11" height="11"><path d="M4.5 4.5 11.5 11.5M11.5 4.5 4.5 11.5" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round"/></svg>`))
	})
	return closePic
}

var (
	closeOnce sync.Once
	closePic  *ui.SVG
)

// copyIcon is the clipboard glyph beside a value that can be copied.
func copyIcon() *ui.SVG {
	copyOnce.Do(func() {
		copyPic = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" width="12" height="12"><rect x="5" y="5" width="9" height="9" rx="1.5" fill="none" stroke="currentColor" stroke-width="1.5"/><path d="M11 5V3.5A1.5 1.5 0 0 0 9.5 2H3.5A1.5 1.5 0 0 0 2 3.5v6A1.5 1.5 0 0 0 3.5 11H5" fill="none" stroke="currentColor" stroke-width="1.5"/></svg>`))
	})
	return copyPic
}

var (
	copyOnce sync.Once
	copyPic  *ui.SVG
)

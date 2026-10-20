//go:build nogui && echo

package shell

import (
	"sync"

	"github.com/egoist/mygo/ui"
)

// The page's own pictures, drawn as the page needs them: MyGo's icons are
// SVG, so each is parsed once and kept.

var (
	plugOnce  sync.Once
	plugPics  map[string]*ui.SVG
	pluginSVG = map[string]string{
		// what kind of plugin one is
		"provider":   `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" width="12" height="12"><path d="M6 2.5a1.5 1.5 0 0 1 3 0V4h2.2a.8.8 0 0 1 .8.8v2.2h-1.5a1.5 1.5 0 0 0 0 3H12v2.2a.8.8 0 0 1-.8.8H8.7a1.5 1.5 0 0 0-2.8 0H3.6a.8.8 0 0 1-.8-.8V3.4a.8.8 0 0 1 .8-.8H6z" fill="none" stroke="currentColor" stroke-width="1.3"/></svg>`,
		"middleware": `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" width="12" height="12"><path d="M10.6 2.2a3.4 3.4 0 0 0 4 4.5l-6 6a3.4 3.4 0 0 0-4.4-4.4l5.7-5.6z" fill="none" stroke="currentColor" stroke-width="1.3" stroke-linejoin="round"/></svg>`,
		"agent":      `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" width="12" height="12"><rect x="3" y="5.5" width="10" height="7" rx="2" fill="none" stroke="currentColor" stroke-width="1.3"/><path d="M8 3v2.5" fill="none" stroke="currentColor" stroke-width="1.3"/><circle cx="6" cy="9" r="1" fill="currentColor"/><circle cx="10" cy="9" r="1" fill="currentColor"/></svg>`,
		"tools":      `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" width="12" height="12"><path d="M3 3h4v4H3zM9 3h4v4H9zM3 9h4v4H3zM9 9h4v4H9z" fill="none" stroke="currentColor" stroke-width="1.2"/></svg>`,
		"page":       `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" width="12" height="12"><rect x="2.5" y="2.5" width="11" height="11" rx="2" fill="none" stroke="currentColor" stroke-width="1.3"/><path d="M2.5 6h11" fill="none" stroke="currentColor" stroke-width="1.3"/></svg>`,
		"star":       `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" width="12" height="12"><path d="M8 2.2l1.7 3.6 3.9.5-2.9 2.7.8 3.9L8 11.1l-3.5 1.8.8-3.9-2.9-2.7 3.9-.5z" fill="none" stroke="currentColor" stroke-width="1.2" stroke-linejoin="round"/></svg>`,
		"shield":     `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" width="12" height="12"><path d="M8 1.8l5 1.9v4.1c0 3-2.1 5-5 6.4-2.9-1.4-5-3.4-5-6.4V3.7z" fill="none" stroke="currentColor" stroke-width="1.2"/></svg>`,
		"out":        `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" width="11" height="11"><path d="M6 3H4a1 1 0 0 0-1 1v8a1 1 0 0 0 1 1h8a1 1 0 0 0 1-1V6" fill="none" stroke="currentColor" stroke-width="1.3" stroke-linecap="round"/><path d="M9.5 2.5H13.5V6.5M13.5 2.5L8 8" fill="none" stroke="currentColor" stroke-width="1.3" stroke-linecap="round"/></svg>`,
	}
)

func plugIcon(name string) *ui.SVG {
	plugOnce.Do(func() {
		plugPics = make(map[string]*ui.SVG, len(pluginSVG))
		for k, v := range pluginSVG {
			plugPics[k] = ui.MustParseSVG([]byte(v))
		}
	})
	return plugPics[name]
}

// plugKindIcon is the glyph of a plugin's kind.
func plugKindIcon(kind string) *ui.SVG {
	switch plugKindOf(kind) {
	case "middleware", "gateway":
		return plugIcon("middleware")
	case "agent":
		return plugIcon("agent")
	case "tools":
		return plugIcon("tools")
	case "page", "pn":
		return plugIcon("page")
	}
	return plugIcon("provider")
}

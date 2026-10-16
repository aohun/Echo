//go:build nogui && echo

package shell

import (
	"sync"

	"github.com/egoist/mygo/ui"
)

var (
	svgOnce sync.Once

	iconBrand         *ui.SVG
	iconSidebarToggle *ui.SVG
	iconAgents        *ui.SVG
	iconSessions      *ui.SVG
	iconLibrary       *ui.SVG
	iconProviders     *ui.SVG
	iconRouting       *ui.SVG
	iconGateway       *ui.SVG
	iconUsage         *ui.SVG
	iconPlugins       *ui.SVG
)

func initIcons() {
	svgOnce.Do(func() {
		iconBrand = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" width="16" height="16"><g fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M3 10v.01M6.5 6.5v7M10 3.5v13M13.5 6.5v7M17 10v.01"/></g></svg>`))
		iconSidebarToggle = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" width="15" height="15"><g fill="none" stroke="currentColor" stroke-width="1.25" stroke-linecap="round" stroke-linejoin="round"><rect x="1.75" y="2.75" width="12.5" height="10.5" rx="2.25"/><path d="M6 2.75v10.5"/></g></svg>`))
		iconAgents = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" width="16" height="16"><g fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"><path d="M4 6l4 4-4 4M10 14h6"/></g></svg>`))
		iconSessions = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" width="16" height="16"><g fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"><path d="M4 5h12v8H9l-4 3v-3H4z"/></g></svg>`))
		iconLibrary = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" width="16" height="16"><g fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"><path d="M5 3.5h9v13H5a1.5 1.5 0 0 1 0-3h9"/></g></svg>`))
		iconProviders = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" width="16" height="16"><g fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"><circle cx="7" cy="10" r="3"/><path d="M10 10h7M14 10v3M16.5 10v2"/></g></svg>`))
		iconRouting = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" width="16" height="16"><g fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"><path d="M3 10h5l4-5h5M8 10l4 5h5M14.5 3l2.5 2-2.5 2M14.5 13l2.5 2-2.5 2"/></g></svg>`))
		iconGateway = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" width="16" height="16"><g fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="4" width="14" height="5" rx="1.5"/><rect x="3" y="11" width="14" height="5" rx="1.5"/><path d="M6 6.5h.01M6 13.5h.01"/></g></svg>`))
		iconUsage = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" width="16" height="16"><g fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"><path d="M4 16V9M8 16V5M12 16v-5M16 16V7"/></g></svg>`))
		iconPlugins = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" width="16" height="16"><g fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"><path d="M8 3.5h4v2.5a1.5 1.5 0 0 0 3 0V3.5h1.5v13H4v-4h2.5a1.5 1.5 0 0 0 0-3H4v-6z"/></g></svg>`))
	})
}

func brandIcon() *ui.SVG {
	initIcons()
	return iconBrand
}

func toggleIcon() *ui.SVG {
	initIcons()
	return iconSidebarToggle
}

func pageIcon(p Page) *ui.SVG {
	initIcons()
	switch p {
	case PageAgents:
		return iconAgents
	case PageSessions:
		return iconSessions
	case PageLibrary:
		return iconLibrary
	case PageProviders:
		return iconProviders
	case PageRouting:
		return iconRouting
	case PageGateway:
		return iconGateway
	case PageUsage:
		return iconUsage
	case PagePlugins:
		return iconPlugins
	}
	return nil
}

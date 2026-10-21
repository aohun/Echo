package provider

import (
	"net/url"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/yetone/magpie/internal/settings"
)

// magpie's own gateway is on the port Settings has (Magic_zero on
// Discord): an app's entry there is magpie's, not one to import, and one
// on the default port is another program's once magpie has moved off it.
func TestGatewayURLOnSettingsPort(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("MAGPIE_ADDR", "")
	def := strconv.Itoa(settings.DefaultPort)
	is := func(u string) bool { p, _ := url.Parse(u); return gatewayURL(p) }
	if !is("http://127.0.0.1:"+def+"/v1") || is("http://127.0.0.1:3591/v1") {
		t.Fatal("the default port")
	}
	s := settings.Load()
	s.Port = 3591
	if err := settings.Save(s); err != nil {
		t.Fatal(err)
	}
	if !is("http://127.0.0.1:3591/v1") || !is("http://localhost:3591") || is("http://127.0.0.1:"+def+"/v1") || is("http://10.0.0.2:3591") {
		t.Fatal("Settings' port")
	}
	t.Setenv("MAGPIE_ADDR", "127.0.0.1:3592")
	if !is("http://127.0.0.1:3592") {
		t.Fatal("MAGPIE_ADDR's port")
	}
}

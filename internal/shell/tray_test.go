//go:build nogui && echo

package shell

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestTrayIconPictures checks the pictures the menu bar icon is made of:
// the template macOS tints, and the frames of the echo pulse it plays when
// the panel opens. A picture that is not a picture leaves the icon blank,
// and a blank icon in the menu bar is the app's own face.
func TestTrayIconPictures(t *testing.T) {
	if _, err := ui.DecodeBitmap(trayTemplatePNG); err != nil {
		t.Errorf("the menu bar template is not a picture: %v", err)
	}
	if _, err := ui.DecodeBitmap(trayPNG); err != nil {
		t.Errorf("the tray's own icon is not a picture: %v", err)
	}
	frames := trayFlapFrames()
	if len(frames) == 0 {
		t.Fatal("the echo pulse has no frames")
	}
	for i, f := range frames {
		if _, err := ui.DecodeBitmap(f); err != nil {
			t.Errorf("frame %d of the pulse is not a picture: %v", i+1, err)
		}
	}
}

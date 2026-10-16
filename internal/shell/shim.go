//go:build nogui

package shell

// dragShim makes magpie's header drag the window under MyGo.
//
// The page's header is marked `style="--wails-draggable:drag"`, which is how
// Wails knows to drag the window from there; MyGo drags from the CSS
// `--app-region: drag`. The shim maps one to the other before the page's own
// scripts run, so no page file has to change.
//
// This is not one of the three gaps the A0 probe enumerated; without it the
// window simply cannot be dragged by its header.
const dragShim = `(function () {
  var style = document.createElement("style");
  style.textContent =
    '[style*="--wails-draggable:drag"], [style*="--wails-draggable: drag"] { --app-region: drag; }' +
    '[style*="--wails-draggable:no-drag"], [style*="--wails-draggable: no-drag"] { --app-region: no-drag; }';
  (document.head || document.documentElement).appendChild(style);
})();`

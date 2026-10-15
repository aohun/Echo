# MyGo assessment

MyGo 0.4.5, MIT license, <https://mygo.egoist.dev/>. Read from the
published docs on 2026-10-12, for version 0.4.5. **Nothing here was built
or executed.** This page records what the docs state.

## What MyGo is

A Go desktop app framework. **Pure Go, no cgo**: native APIs are called at
runtime through `purego` (`dlopen` + assembly trampolines) on macOS and
Linux, and through the `syscall` package on Windows — never
`import "C"`. Consequence: any platform can cross-compile for any other.
`mygo build` defaults to `CGO_ENABLED=0`.

One app can hold windows of **two kinds**:

| | Web frontend | Native UI (`ui` package) |
| --- | --- | --- |
| Rendering | The OS webview: WKWebView (macOS), WebKitGTK 4.1 (4.0 fallback) on Linux, WebView2 on Windows. No bundled browser. | Drawn by MyGo on the GPU: Metal, Direct3D 11, OpenGL, or CPU. |
| Authoring | HTML/CSS/JS with any toolchain | Go only; `ui.Column(c).Fill().Center().Gap(12).Children(...)` |
| Reaching Go | `mygo.Bind(svc)`; `mygo generate` emits a TypeScript client | Direct function calls; no binding needed |
| Streaming | Typed events and channels | — |

Reported footprints (docs): hello-world ~7 MB binary, ~35 MB physical on
macOS, ~62 MB including WKWebView's helper processes, 0% idle CPU. A
native-UI window starts no webview process; the counter example is
65–67 MB on Linux tiled to half a 4K display. Nothing polls.

## The pieces that matter for an Echo build

### Bound services and a generated TypeScript client

`mygo.Bind(v)` before `App.Run` makes every exported method of `v`
callable from the page, named after the type (`Notes` → `Notes.add(...)`,
`Notes.list()`). `mygo.BindAs("Store", v)` renames.

Documented method rules:

- Any number of JSON-encodable arguments, including a variadic one.
- Returns may include an `error`.
- Channels let a method stream values back.
- `Bind` **panics at startup** when a name is taken twice or a method uses
  a type JSON cannot encode (a channel or a function).

`mygo generate` turns the bound surface into a TypeScript client carrying
the types of the Go structs and the documentation comments of the methods,
so calls, channels and events are compiler-checked on both sides.

**This is the strongest reason to choose MyGo over a Wails-style stack.**
magpie ships 45,201 lines of hand-written frontend with no generator and
no bundler, and pays for it in hand-maintained invariants
(`revalidated`/`versionedPage` content-hash naming; every string needing
zh/ja/de with matching placeholders). A generated client removes a class
of that work.

### Page loading, and the origin change

Windows load pages with scheme-less URLs (`/`, `/settings?tab=general`),
resolved against:

- during `mygo dev`, the dev server at `devUrl` in `mygo.config.ts`
  (`devCommand` starts it), so hot reload works;
- in builds, `mygo://localhost/`, serving the files of `frontendDist`
  that `mygo build` embeds after running `buildCommand`.

Without `devUrl`, `mygo dev` serves `frontendDist` from disk. Paths
without a file extension that match no file serve `index.html`, so
client-side routers work.

The docs state the origin is `mygo://localhost` on macOS and Linux and
`http://mygo.localhost` on Windows (WebView2 serves custom schemes under
`http://<scheme>.localhost`). **The docs also state that storage belongs
to an origin, so data a page stores under `mygo dev` is not what the built
app sees, and advise keeping durable state in Go** (for example a file in
`App.Path(mygo.PathUserData)`) handed to pages through bound methods.
This is a design constraint to plan for, not a defect: see O4 in
[06](06-open-questions-and-risks.md).

### Desktop APIs

`mygo.Dialog` (`Open`/`Save`/`Message`/`Error`, with `Parent` attaching as
a sheet on macOS); `mygo.NewNotification`; clipboard and drag data; the
shell; displays; dark mode; power; **global shortcuts**. Dialogs block
until answered and are meant to be called from bound methods or a
goroutine, since event listeners and menu items should return quickly.

### Menus, tray, distribution

Menus and the tray are covered by their own guide. `mygo build` produces,
per platform in `build/<os>-<arch>/`:

- macOS: `My App.app` signed, plus `My App 0.1.0 arm64.dmg`
  (`darwin/universal` combines arm64 and amd64 in one app);
- Windows: `My App.exe` and an NSIS `My App Setup … .exe`;
- Linux: the executable, a `.desktop` entry, a `.png`, an XML for declared
  file types, a `.tar.gz` with `install.sh`, and a `.deb`.

`-platform` takes GOOS/GOARCH targets. **Signing and DMG creation happen
on macOS only**; Windows installers are made elsewhere only where NSIS is
installed. `-skip-dmg`, `-skip-notarize` and `-debug` (keeps the
inspector) exist. Auto-updates, GitHub Actions, URL schemes and file types
each have a guide.

### Native UI widget coverage

Actions (Button, Link, Menu button, Context menu, Toggle, Segmented
control, Toolbar); Choices (Checkbox, Switch, Radio, Select, Slider, Range
slider, Stepper, Rating); Text input (Text, Number, Search, Combobox,
Autocomplete, Token field, Editable text, Find bar); Dates and colors
(Calendar, Date, Time, Color picker); Collections (List, Table, Tree,
Outline, Grid view); Containers (Scroll view, Grid, Split view,
Collapsible, Accordion, Form, **Web view**); Navigation (Tabs, Sidebar,
Breadcrumbs, Back/forward); Dialogs and messages (Dialog, Alert, Popover,
Tooltip, Toast); Status (Progress bar, Spinner, Meter, Badge); Charts;
Images (Image, Icon, Avatar).

Also documented: flexbox and grid layout, stylers, accessibility and
screen reader support, overlays, transitions and motion, custom widgets,
drag and drop, pages and navigation, an inspector, and a **browser
target** (compiled to WebAssembly, canvases drawn with WebGL 2).

### Official plugins

Fetch, WebSocket, SQLite, Updater, Terminal, Glass.

## Wails v3 → MyGo mapping

If Echo reuses magpie's shell design, this is the mapping to plan around.

| magpie's Wails-based piece | MyGo equivalent | Note |
| --- | --- | --- |
| `wailsapp/wails/v3` app lifecycle, windows, tray, menus | `mygo.App`, `mygo.NewWindow`, `mygo.WindowOptions`, Menus and the tray | Direct conceptual match |
| Hand-written `/api/*` routes on `Handler` + `//go:embed assets` | `mygo.Bind(svc)` + `mygo generate` + `mygo://localhost` serving `frontendDist` | **Strictly better**, but changes the contract shape |
| Wails dialogs, notifications | `mygo.Dialog`, `mygo.NewNotification` | Direct match |
| Wails clipboard, shell, displays | Clipboard and drag data, the shell, displays | Direct match |
| Wails global shortcuts | Global shortcuts | Direct match |
| Wails build/package | `mygo build` | Different output layout; NSIS on Windows |
| Wails update | Updater plugin | Compare against magpie's `internal/update` (2,111 lines) first |
| `magpie tui` (`-tags nogui`) | none | MyGo has no TUI |
| macOS/Windows shell specifics | partly covered by "Platform differences" | **Unverified.** O2. |

## What MyGo does not provide

- **No TUI.** A TUI, if Echo wants one, is Echo's own (magpie uses
  `bubbletea`/`bubbles`/`lipgloss`).
- **No domain logic.** MyGo is a shell. It does not make API translation,
  agent config editing or credential handling simpler.
- **No cgo guarantee for third-party dependencies.** The docs allow
  `CGO_ENABLED=1` for `mygo dev`, `mygo build` and `mygo generate` when
  dependencies need it, requiring a C compiler (or cross compiler) — which
  gives up the advertised cross-compile convenience.
- **No bundled database, HTTP router or config format.** Official plugins
  cover SQLite, Fetch and WebSocket, but the docs describe these as
  plugins, not as a batteries-included stack.

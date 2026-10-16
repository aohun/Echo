//go:build nogui

// Command echoprobe starts the Echo shell, walks magpie's eight views in the
// real window, and reports what each one rendered.
//
// It is the acceptance check for the shell: magpie's page is served unmodified,
// so the question is whether it boots and switches views with no page errors.
//
// It never touches a real agent config: the caller runs it under an isolated
// home (see the Makefile's probe target), which is the rule magpie's own tests
// follow (risk O12 in docs/06-open-questions-and-risks.md).
//
//	echoprobe -out /tmp/echo-probe.json
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/egoist/mygo"
	"github.com/yetone/magpie/internal/shell"
)

// views are magpie's pages, in the order its nav lists them.
var views = []string{
	"agents", "providers", "gateway", "routing",
	"usage", "sessions", "library", "plugins",
}

// viewResult is what one view reported.
type viewResult struct {
	View    string   `json:"view"`
	Present bool     `json:"present"`
	Visible bool     `json:"visible"`
	Current bool     `json:"current"`
	Nodes   int      `json:"nodes"`
	Errors  []string `json:"errors,omitempty"`
}

// report is the probe's outcome.
type report struct {
	Title      string       `json:"title"`
	Origin     string       `json:"origin"`
	BootPrefs  string       `json:"bootPrefs"`
	NavButtons int          `json:"navButtons"`
	BodyKids   int          `json:"bodyKids"`
	Views      []viewResult `json:"views"`
	OK         bool         `json:"ok"`
	Note       string       `json:"note,omitempty"`
}

// rep is filled on the main thread and printed once the app has quit.
var rep report

func main() {
	out := flag.String("out", "", "write the JSON report here, as well as stdout")
	settle := flag.Duration("settle", 3*time.Second, "how long to wait for the page to load")
	flag.Parse()

	opts := shell.Options{
		ExtraPreload: errorRecorder,
		Ready: func(w *mygo.Window) {
			// Let the page load and run its boot sequence. MyGo has no "page
			// loaded" callback that this shell subscribes to yet, so the wait
			// is time-based.
			time.AfterFunc(*settle, func() {
				rep = probe(w)
				mygo.App.Quit()
			})
		},
	}

	// App.Run must be called from the main goroutine, so the probe runs inside
	// the app and quits it when done.
	if err := shell.Run(opts); err != nil {
		log.Fatal(err)
	}

	b, _ := json.MarshalIndent(rep, "", "  ")
	fmt.Println(string(b))
	if *out != "" {
		if err := os.WriteFile(*out, append(b, '\n'), 0o644); err != nil {
			log.Fatal(err)
		}
	}
	if !rep.OK {
		os.Exit(1)
	}
}

// probe asks the running page what it rendered. It runs on the main thread,
// where MyGo's page calls are allowed.
func probe(w *mygo.Window) report {
	var r report
	r.Title = evalString(w, "document.title")
	r.Origin = evalString(w, "location.origin")
	r.BootPrefs = evalString(w, "JSON.stringify(window.bootPrefs || null)")
	r.NavButtons = evalInt(w, "document.querySelectorAll('#nav button').length")
	r.BodyKids = evalInt(w, "document.body.children.length")

	for _, v := range views {
		r.Views = append(r.Views, probeView(w, v))
	}

	r.OK = r.NavButtons >= 8
	for _, vr := range r.Views {
		if !vr.Present || len(vr.Errors) > 0 {
			r.OK = false
		}
	}
	if !r.OK {
		r.Note = "page errors are collected from window.__echoErrors; see the per-view errors"
	}
	return r
}

// probeView switches to one view and reports what it shows.
func probeView(w *mygo.Window, view string) viewResult {
	res := viewResult{View: view}
	// magpie's own router: show(view) is what a nav click calls.
	eval(w, fmt.Sprintf("window.show && window.show(%q)", view))
	// Let the view's own load settle; a view fetches when it is switched to.
	time.Sleep(900 * time.Millisecond)

	sel := fmt.Sprintf("#view-%s", view)
	res.Present = evalBool(w, fmt.Sprintf("!!document.querySelector(%q)", sel))
	res.Visible = evalBool(w, fmt.Sprintf("(() => { const e = document.querySelector(%q); return !!e && !e.hidden; })()", sel))
	res.Nodes = evalInt(w, fmt.Sprintf("(() => { const e = document.querySelector(%q); return e ? e.querySelectorAll('*').length : 0; })()", sel))
	res.Current = evalBool(w, fmt.Sprintf("!!document.querySelector('#nav button[data-view=%q][aria-current]')", view))
	res.Errors = evalStrings(w, "window.__echoErrors || []")
	return res
}

// errorRecorder collects page errors, so the probe can report whether the page
// booted cleanly. It is injected before the page's own scripts run.
const errorRecorder = `(function () {
  var errors = [];
  window.__echoErrors = errors;
  function record(what, detail) {
    errors.push(what + ": " + detail);
  }
  window.addEventListener("error", function (e) {
    record("error", (e.message || "?") + " @ " + (e.filename || "?") + ":" + (e.lineno || 0));
  });
  window.addEventListener("unhandledrejection", function (e) {
    var r = e.reason;
    record("unhandledrejection", (r && (r.stack || r.message)) || String(r));
  });
  var origError = console.error;
  console.error = function () {
    record("console.error", Array.prototype.map.call(arguments, String).join(" "));
    return origError.apply(console, arguments);
  };
})();`

func eval(w *mygo.Window, code string) {
	if _, err := w.Page().Eval(code); err != nil {
		log.Printf("eval %q: %v", code, err)
	}
}

func evalString(w *mygo.Window, code string) string {
	s, err := mygo.EvalAs[string](w.Page(), code)
	if err != nil {
		log.Printf("eval %q: %v", code, err)
	}
	return s
}

func evalInt(w *mygo.Window, code string) int {
	n, err := mygo.EvalAs[int](w.Page(), code)
	if err != nil {
		log.Printf("eval %q: %v", code, err)
	}
	return n
}

func evalBool(w *mygo.Window, code string) bool {
	b, err := mygo.EvalAs[bool](w.Page(), code)
	if err != nil {
		log.Printf("eval %q: %v", code, err)
	}
	return b
}

func evalStrings(w *mygo.Window, code string) []string {
	s, err := mygo.EvalAs[[]string](w.Page(), code)
	if err != nil {
		log.Printf("eval %q: %v", code, err)
	}
	return s
}

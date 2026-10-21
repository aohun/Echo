// Package magpieclient speaks to a running `magpie web` instance over
// loopback HTTP: the JSON API magpie's own page drives (docs/08,
// docs/09). One client serves one instance; New builds it, Probe asks
// the first question, and every write answers with the state to adopt.
//
// The client holds the instance's run key and sends it the way the page
// keeps it — as the cookie the key's link boots into. The key is a
// secret: it never goes to a log, a file or anywhere but the instance.
package magpieclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)// DefaultPort is the port `magpie web` listens on unless told another.
const DefaultPort = "3430"

// Client is one magpie web instance Echo talks to. Safe for concurrent
// use; the http.Client it carries is.
type Client struct {
	base string // the instance's origin, http://127.0.0.1:<port>
	key  string // the instance's run key
	http *http.Client
}

// New builds a client for the instance at base (an origin with an
// explicit port — the cookie's name carries it) holding the run key key.
func New(base, key string) *Client {
	return &Client{
		base: strings.TrimRight(base, "/"),
		key:  key,
		http: &http.Client{Timeout: 30 * time.Second},
	}
}

// APIError is a non-2xx answer, with the status and the body to read.
type APIError struct {
	Status int
	Body   string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("magpie web: %d %s", e.Status, truncate(e.Body, 200))
}

// State is what /api/state and every write answer. Only the fields Echo
// reads are typed; the rest stays raw, so an upstream addition cannot
// break the decode.
type State struct {
	Agents   []Agent        `json:"agents"`
	Catalog  string         `json:"catalog"`
	Clients  []any          `json:"clients"`
	FX       map[string]any `json:"fx"`
	Profiles []any         `json:"profiles"`
	Settings Settings       `json:"settings"`
}

// Agent is one detected agent as the Agents page reads it.
type Agent struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	Icon   string  `json:"icon"`
	Path   string  `json:"path"`
	Fields []Field `json:"fields"`
}

// Field is one agent setting: a model, a toggle, a choice.
type Field struct {
	Key     string   `json:"key"`
	Label   string   `json:"label"`
	Value   string   `json:"value"`
	Options []Option `json:"options"`
}

// Option is one choice a field offers.
type Option struct {
	Value  string `json:"value"`
	Note   string `json:"note"`
	Icon   string `json:"icon"`
	Group  string `json:"group"`
	Direct string `json:"direct"`
}

// Settings is the settings document's fields Echo reads today.
type Settings struct {
	Version  string         `json:"version"`
	Gateway  string         `json:"gateway"`
	Dir      string         `json:"dir"`
	Theme    string         `json:"theme"`
	Lang     string         `json:"lang"`
	Web      bool           `json:"web"`
	GatewayOn bool          `json:"gatewayOn,omitempty"`
	Raw      map[string]any `json:"-"`
}

// UnmarshalJSON keeps what Echo names and holds the rest in Raw, so a
// settings write can send the document back the way the page does.
func (s *Settings) UnmarshalJSON(b []byte) error {
	type plain Settings
	if err := json.Unmarshal(b, (*plain)(s)); err != nil {
		return err
	}
	return json.Unmarshal(b, &s.Raw)
}

// Info is what Probe reads: the instance is alive, is who Echo expects,
// and says where its gateway is.
type Info struct {
	Version string
	Gateway string
	Dir     string
}

// Probe asks the instance to say itself, and so doubles as both the
// health check and the version gate: a magpie that grew past the
// contract this client speaks fails here, visibly, at connect time.
func (c *Client) Probe(ctx context.Context) (Info, error) {
	var s Settings
	if err := c.do(ctx, http.MethodGet, "/api/settings", nil, &s); err != nil {
		return Info{}, err
	}
	return Info{Version: s.Version, Gateway: s.Gateway, Dir: s.Dir}, nil
}

// State reads the instance's full state document.
func (c *Client) State(ctx context.Context) (*State, error) {
	s := &State{}
	if err := c.do(ctx, http.MethodGet, "/api/state", nil, s); err != nil {
		return nil, err
	}
	return s, nil
}

// Set writes one agent field and answers with the new state to adopt.
func (c *Client) Set(ctx context.Context, agent, field, value string) (*State, error) {
	s := &State{}
	if err := c.do(ctx, http.MethodPost, "/api/set", map[string]string{
		"agent": agent, "field": field, "value": value,
	}, s); err != nil {
		return nil, err
	}
	return s, nil
}

// Rescan asks the instance to look for agents on this machine again, and
// answers with the state that found.
func (c *Client) Rescan(ctx context.Context) (*State, error) {
	s := &State{}
	if err := c.do(ctx, http.MethodPost, "/api/agents/rescan", nil, s); err != nil {
		return nil, err
	}
	return s, nil
}

// Drift reads what changed behind magpie's back, per agent: an empty map
// when nothing did. The page asks this while its window is up; Echo's
// views ask it on the same kind of timer.
func (c *Client) Drift(ctx context.Context) (map[string]json.RawMessage, error) {
	out := map[string]json.RawMessage{}
	if err := c.do(ctx, http.MethodGet, "/api/drift", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// Do runs any recorded call of the surface (docs/09): the escape hatch
// the views adopt one page at a time, until a typed method earns its
// place. A nil body sends none; a non-nil out decodes the answer.
func (c *Client) Do(ctx context.Context, method, path string, body, out any) error {
	return c.do(ctx, method, path, body, out)
}

// do runs one call: the run key's cookie on it, JSON bodies encoded, and
// a non-2xx answered as *APIError — never as a half-decoded success.
func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	var rdr *bytes.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("magpieclient: encode %s %s: %w", method, path, err)
		}
		rdr = bytes.NewReader(b)
	} else {
		rdr = bytes.NewReader(nil)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, rdr)
	if err != nil {
		return fmt.Errorf("magpieclient: %s %s: %w", method, path, err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.key != "" {
		req.AddCookie(&http.Cookie{Name: c.cookieName(), Value: c.key})
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("magpieclient: %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		var buf bytes.Buffer
		buf.ReadFrom(io.LimitReader(resp.Body, 4096))
		return &APIError{Status: resp.StatusCode, Body: buf.String()}
	}
	if out == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("magpieclient: decode %s %s: %w", method, path, err)
	}
	return nil
}

// cookieName is the instance's cookie: webGuard names it after the port
// it answers on, so the same key wears a different name on another port.
func (c *Client) cookieName() string {
	u, err := url.Parse(c.base)
	if err != nil {
		return "magpie_web_" + DefaultPort
	}
	if p := u.Port(); p != "" {
		return "magpie_web_" + p
	}
	return "magpie_web_" + DefaultPort
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

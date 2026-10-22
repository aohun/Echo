package magpieclient

// The providers page's write actions over HTTP (docs/08 M4), tested with
// a fake magpie (Go server).

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestProviderSwitchSendsOffAndAdoptsState(t *testing.T) {
	var (
		gotMethod string
		gotPath   string
		gotBody   string
	)
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath, gotBody = r.Method, r.URL.Path, readAllBody(r)
		rw.Header().Set("Content-Type", "application/json")
		rw.Write([]byte(`{"providers":[],"gateway":{"url":"http://127.0.0.1:3497","running":true}}`))
	}))
	defer srv.Close()

	c := New(srv.URL, "k")
	out, err := c.ProviderSwitch(context.Background(), "openai", true)
	if err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodPost || gotPath != "/api/provider/off" {
		t.Errorf("switch asked %s %s, want POST /api/provider/off", gotMethod, gotPath)
	}
	var sent map[string]string
	if err := json.Unmarshal([]byte(gotBody), &sent); err != nil {
		t.Fatalf("switch body %q: %v", gotBody, err)
	}
	if sent["id"] != "openai" {
		t.Errorf("switch body id %q", sent["id"])
	}
	if out.Gateway.URL == "" {
		t.Error("switch did not answer with the providers document")
	}
}

func TestProvidersArrangeSendsTheWholeOrder(t *testing.T) {
	var (
		gotMethod string
		gotPath   string
		gotBody   string
	)
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath, gotBody = r.Method, r.URL.Path, readAllBody(r)
		rw.Header().Set("Content-Type", "application/json")
		rw.Write([]byte(`{"providers":[],"gateway":{"url":"http://127.0.0.1:3497","running":true}}`))
	}))
	defer srv.Close()

	c := New(srv.URL, "k")
	_, err := c.ProvidersArrange(context.Background(), []string{"openai", "deepseek"})
	if err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodPost || gotPath != "/api/providers/arrange" {
		t.Errorf("arrange asked %s %s, want POST /api/providers/arrange", gotMethod, gotPath)
	}
	var sent struct {
		Order []string `json:"order"`
	}
	if err := json.Unmarshal([]byte(gotBody), &sent); err != nil {
		t.Fatalf("arrange body %q: %v", gotBody, err)
	}
	if len(sent.Order) != 2 || sent.Order[0] != "openai" || sent.Order[1] != "deepseek" {
		t.Errorf("arrange body order %v", sent.Order)
	}
}

func readAllBody(r *http.Request) string {
	var buf strings.Builder
	b := make([]byte, 4096)
	for {
		n, err := r.Body.Read(b)
		buf.Write(b[:n])
		if err != nil {
			break
		}
	}
	return buf.String()
}

package lib

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRedirectRecording(t *testing.T) {
	ts := NewHTTPRedirectTestServer(false)

	cfg := DefaultConfig()
	cfg.Job.Seeds = []string{ts.URL}
	cfg.Job.StopURL = ts.URL + "/e"
	cfg.ResourceHandlers = []*ResourceHandlerConfig{
		{Type: "MEM"},
	}

	walk, stop, err := NewWalkJob(func(c *Config) { *c = *cfg })
	if err != nil {
		t.Fatal(err.Error())
	}

	if err := walk.Start(stop); err != nil {
		t.Error(err)
	}

	rsc := walk.ResourceHandlers()[0].(*MemResourceHandler).Resources

	if rsc[len(rsc)-1].RedirectFrom == "" {
		t.Errorf("final resource %d missing redirectFrom", len(rsc)-1)
	}

	for i, r := range rsc[:len(rsc)-1] {
		if r.RedirectTo == "" {
			t.Errorf("resource %d missing redirectTo", i)
			t.Log(r)
		}
	}

	data, _ := json.MarshalIndent(rsc, "", " ")
	t.Log(string(data))

}

// Create a test server that redirects a bunch
func NewHTTPRedirectTestServer(withTooMany bool) *httptest.Server {
	redirectTo := func(path string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, path, http.StatusMovedPermanently)
		}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", redirectTo("/a"))
	mux.HandleFunc("/a", redirectTo("/b"))
	mux.HandleFunc("/b", redirectTo("/c"))
	mux.HandleFunc("/c", redirectTo("/d"))
	mux.HandleFunc("/d", redirectTo("/e"))
	mux.HandleFunc("/e", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html><body><h1>Hello World!</h1></body></html>`))
	})
	mux.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	if withTooMany {
		mux.HandleFunc("/toomany/01", redirectTo("/toomany/02"))
		mux.HandleFunc("/toomany/02", redirectTo("/toomany/03"))
		mux.HandleFunc("/toomany/03", redirectTo("/toomany/04"))
		mux.HandleFunc("/toomany/04", redirectTo("/toomany/05"))
		mux.HandleFunc("/toomany/05", redirectTo("/toomany/06"))
		mux.HandleFunc("/toomany/06", redirectTo("/toomany/07"))
		mux.HandleFunc("/toomany/07", redirectTo("/toomany/08"))
		mux.HandleFunc("/toomany/08", redirectTo("/toomany/09"))
		mux.HandleFunc("/toomany/09", redirectTo("/toomany/10"))
		mux.HandleFunc("/toomany/10", redirectTo("/toomany/11"))
		mux.HandleFunc("/toomany/11", redirectTo("/toomany/12"))
		mux.HandleFunc("/toomany/12", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`<html><body><h1>Hello World!</h1></body></html>`))
		})
	}

	return httptest.NewServer(mux)
}

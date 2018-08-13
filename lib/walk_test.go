package lib

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewWalk(t *testing.T) {
	s := httptest.NewServer(http.FileServer(http.Dir("testdata/qri_io")))

	crawl, stop, err := NewWalk(
		JSONConfigFromFilepath("testdata/qri_io.config.json"),
		ServerJSONConfig(s),
	)
	if err != nil {
		t.Fatal(err.Error())
	}

	crawl.Start(stop)
}

// ServerJSONConfig creates a configuration from a file, replacing seed & domain
// coordinator values with testserver urls
func ServerJSONConfig(s *httptest.Server) func(c *Config) {
	return func(c *Config) {
		c.Coordinator.Domains = []string{s.URL}
		c.Coordinator.Seeds = []string{s.URL}
	}
}

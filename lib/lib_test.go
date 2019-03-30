package lib

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// HTTPDirTestCase is a simulation of a domain for crawling, constructed from a directory with a
// standardized structure ()
type HTTPDirTestCase struct {
	Name    string
	DirPath string
	t       *testing.T
}

const (
	// standard test case walk configuration file
	configFilename = "config.json"
	// standard test files that should be turned into a local server for testing
	siteDirName = "site"
)

// NewHTTPDirTestCase creates a test case from a given filepath & testing struct
func NewHTTPDirTestCase(t *testing.T, path string) *HTTPDirTestCase {
	return &HTTPDirTestCase{
		Name:    filepath.Base(path),
		DirPath: path,
		t:       t,
	}
}

// Server generates a test server from the test case siteDirName directory
func (t *HTTPDirTestCase) Server() *httptest.Server {
	path := filepath.Join(t.DirPath, siteDirName)

	if fi, err := os.Stat(path); err != nil {
		t.t.Fatal(err.Error())
		return nil
	} else if !fi.IsDir() {
		t.t.Fatalf("cannot create test server. %s is not a directory", path)
		return nil
	}

	dir := http.Dir(path)
	return httptest.NewServer(http.FileServer(dir))
}

// Config generates the associated test case, with domains configured
// for the passed-in test server
func (t *HTTPDirTestCase) Config(s *httptest.Server) func(c *Config) {
	return func(c *Config) {
		JSONConfigFromFilepath(filepath.Join(t.DirPath, configFilename))(c)
		c.Job.Domains = append(c.Job.Domains, s.URL)
		c.Job.Seeds = append(c.Job.Seeds, s.URL)
	}
}

// ServerJSONConfig creates a configuration from a file, replacing seed & domain
// coordinator values with testserver urls
func ServerJSONConfig(s *httptest.Server) func(c *Config) {
	return func(c *Config) {
		c.Job.Domains = []string{s.URL}
		c.Job.Seeds = []string{s.URL}
	}
}

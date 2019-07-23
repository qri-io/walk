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
	// standard test case coordinator configuration file
	coordConfigFilename = "coordinator.config.json"
	// standard test job configuration file
	jobsConfigFilename = "job.config.json"
	// standard test files that should be turned into a local server for testing
	siteDirName = "site"
)

// MustCoordinator forces a test to fail if a coordinator can't be created
func MustCoordinator(t *testing.T, newFunc func() (Coordinator, error)) Coordinator {
	c, err := newFunc()
	if err != nil {
		t.Fatal(err)
	}
	return c
}

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

// CoordinatorConfig generates the associated test case, with domains configured
// for the passed-in test server
func (t *HTTPDirTestCase) CoordinatorConfig() func(c *CoordinatorConfig) {
	return func(c *CoordinatorConfig) {
		JSONCoordinatorConfigFromFilepath(filepath.Join(t.DirPath, coordConfigFilename))(c)
	}
}

func (t *HTTPDirTestCase) Coordinator() (Coordinator, error) {
	return NewCoordinator(t.CoordinatorConfig())
}

// JobConfig grabs the job configuration at testdir/job.config.json
func (t *HTTPDirTestCase) JobConfig(s *httptest.Server) *JobConfig {
	cfg := &JobConfig{}
	JSONJobConfigFromFilepath(filepath.Join(t.DirPath, jobsConfigFilename))(cfg)
	cfg.Domains = append(cfg.Domains, s.URL)
	cfg.Seeds = append(cfg.Seeds, s.URL)
	return cfg
}

// // ServerJSONConfig creates a configuration from a file, replacing seed & domain
// // coordinator values with testserver urls
// func ServerJSONConfig(s *httptest.Server) func(c *Config) {
// 	return func(c *Config) {
// 		c.Job.Domains = []string{s.URL}
// 		c.Job.Seeds = []string{s.URL}
// 	}
// }

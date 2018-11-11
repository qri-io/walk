package lib

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/dgraph-io/badger"
)

// Config is the global configuration for all components of a walk
type Config struct {
	Badger           *BadgerConfig
	Coordinator      *CoordinatorConfig
	RequestStore     *RequestStoreConfig
	Queue            *QueueConfig
	Workers          []*WorkerConfig
	ResourceHandlers []*ResourceHandlerConfig
}

// ApplyConfigs takes zero or more configuration functions to produce
// a single configuration
func ApplyConfigs(configs ...func(c *Config)) *Config {
	// combine configurations with default
	cfg := DefaultConfig()
	for _, o := range configs {
		o(cfg)
	}
	return cfg
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Coordinator: &CoordinatorConfig{
			Crawl:            true,
			Domains:          []string{"https://datatogether.org"},
			Seeds:            []string{"https://datatogether.org"},
			MaxAttempts:      3,
			StopAfterEntries: 5,
			DoneScanMilli:    30000,
		},
		Workers: []*WorkerConfig{
			&WorkerConfig{
				Type:                  "local",
				Parallelism:           2,
				DelayMilli:            500,
				Polite:                true,
				RecordResponseHeaders: false,
				RecordRedirects:       true,
			},
		},
		ResourceHandlers: []*ResourceHandlerConfig{},
		Badger:           NewBadgerConfig(),
		// DestPath: "sitemap.json",
	}
}

// ErrNoBadgerConfig is the result of attempting to connect to a badgerDB
// without one configured
var ErrNoBadgerConfig = fmt.Errorf("badger is not configured")

// BadgerDB returns the badger DB connection, creating a default-configured
// badger store if one doesn't exist
func (cfg *Config) BadgerDB() (*badger.DB, error) {
	if cfg.Badger == nil {
		return nil, ErrNoBadgerConfig
	}
	return cfg.Badger.DB()
}

// JSONConfigFromFilepath returns a func that reads a json-encoded
// config if the file specified by filepath exists, failing silently if no
// file is present. If a file is present but not valid json, the program panics
func JSONConfigFromFilepath(path string) func(*Config) {
	return func(c *Config) {
		if data, err := ioutil.ReadFile(path); err == nil {
			cfg := Config{}
			log.Infof("using config file: %s", path)
			if err := json.Unmarshal(data, &cfg); err != nil {
				err = fmt.Errorf("error parsing configuration file at path: %s: %s", path, err.Error())
				log.Errorf(err.Error())
				panic(err)
			}
			*c = cfg
		}
	}
}

// CoordinatorConfig holds all Coordinator configuration details
type CoordinatorConfig struct {
	// Seeds is a list of urls to seed the crawler with
	Seeds []string
	// If true, links from completed resources returned to the coordinator will
	// be added to the queue (aka, crawling). Only links within the domains list
	// that don't match ignore patterns will be crawled
	Crawl bool
	// Domains is the list of domains to crawl. Only domains listed
	// in this list will be crawled
	Domains []string
	// Ignore is a list of url patterns to ignore
	IgnorePatterns []string
	// DelayMilli determines how long to wait between fetches for a given worker
	DelayMilli int
	// StopAfterEntries kills the crawler after a specified number of urls have
	// been visited. default of 0 don't limit the number of entries
	StopAfterEntries int
	// StopUrl will stop the crawler after crawling a given URL
	StopURL string
	// UnfetchedScanFreqMilliseconds sets how often the crawler should scan the list of fetched
	// urls, checking links for unfetched urls. this "rehydrates" the crawler with urls that
	// might be missed while avoiding duplicate fetching. default value of 0 disables the check
	UnfetchedScanFreqMilliseconds int
	// BackoffResponseCodes is a list of response codes that when encountered will add
	// half the value of of CrawlDelayMilliseconds per request, slowing the crawl in response
	// every minute
	BackoffResponseCodes []int
	// MaxAttempts is the maximum number of times to try a url before giving up
	MaxAttempts int
	// How frequently to check to see if crawl is done, in milliseconds
	DoneScanMilli int
}

// RequestStoreConfig holds configuration details for a request store
type RequestStoreConfig struct {
	Type string
}

// QueueConfig holds configuration details for a Queue
type QueueConfig struct {
	Type string
}

// WorkerConfig holds configuration details for a request store
type WorkerConfig struct {
	Parallelism int
	Type        string
	DelayMilli  int
	// Polite is weather or not to respect robots.txt
	Polite bool
	// RecordResponseHeaders sets weather or not to keep a map of response headers
	RecordResponseHeaders bool
	// RecordRedirects specifies weather redirects should be recorded as redirects
	RecordRedirects bool
	UserAgent       string
}

// ResourceHandlerConfig holds configuration details for a resource handler
type ResourceHandlerConfig struct {
	Type string
	// SrcPath is the path to an input site file from a previous crawl
	SrcPath string
	// DestPath is the path to the output site file
	DestPath string
	// Prefix implements any namespacing for this config
	// not used by all ResourceHandlers
	Prefix string
}

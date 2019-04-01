package lib

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

// CoordinatorConfig is the global configuration for all components of a walk
type CoordinatorConfig struct {
	Badger       *BadgerConfig
	RequestStore *RequestStoreConfig
	Queue        *QueueConfig
	Collection   *CollectionConfig
	// UnfetchedScanFreqMilliseconds sets how often the crawler should scan the list of fetched
	// urls, checking links for unfetched urls. this "rehydrates" the crawler with urls that
	// might be missed while avoiding duplicate fetching. default value of 0 disables the check
	UnfetchedScanFreqMilliseconds int
}

// RequestStoreConfig holds configuration details for a request store
type RequestStoreConfig struct {
	Type string
}

// QueueConfig holds configuration details for a Queue
type QueueConfig struct {
	Type string
}

// CollectionConfig configures the on-disk collection. There can be at most
// one collection per walk process
type CollectionConfig struct {
	// LocalDirs is a slice of locations on disk to check for walks
	LocalDirs []string
}

// ApplyCoordinatorConfigs takes zero or more configuration functions to produce
// a single configuration
func ApplyCoordinatorConfigs(configs ...func(c *CoordinatorConfig)) *CoordinatorConfig {
	// combine configurations with default
	cfg := DefaultCoordinatorConfig()
	for _, o := range configs {
		o(cfg)
	}
	return cfg
}

// DefaultCoordinatorConfig returns the default configuration for a worker
func DefaultCoordinatorConfig() *CoordinatorConfig {
	return &CoordinatorConfig{
		Collection: &CollectionConfig{
			// default to checking local directory for collections
			LocalDirs: []string{"."},
		},
		Badger: NewBadgerConfig(),
	}
}

// JSONCoordinatorConfigFromFilepath returns a func that reads a json-encoded
// config if the file specified by filepath exists, failing silently if no
// file is present. If a file is present but not valid json, the program panics
func JSONCoordinatorConfigFromFilepath(path string) func(*CoordinatorConfig) {
	return func(c *CoordinatorConfig) {
		cfg := &CoordinatorConfig{}
		if err := readJSONFile(path, cfg); err != nil {
			panic(err)
		}
		log.Infof("using config file: %s", path)
		*c = *cfg
	}
}

// readJSONFile reads a configuration JSON file
func readJSONFile(path string, dest interface{}) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return fmt.Errorf("error reading json file at path: %s: %s", path, err.Error())
	}

	if err := json.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("error parsing json file at path: %s: %s", path, err.Error())
	}

	return nil
}

// JobConfig holds all Job configuration details
type JobConfig struct {
	// Seeds is a list of urls to seed the crawler with
	Seeds []string
	// SeedsPath is a filepath or URL to a newline-delimited list of seed URL strings
	SeedsPath string
	// If true, links from completed resources returned to the job will
	// be added to the queue (aka, crawling). Only links within the domains list
	// that don't match ignore patterns will be crawled
	Crawl bool
	// Domains is the list of domains to crawl. Only domains listed
	// in this list will be crawled
	Domains []string
	// Ignore is a list of url patterns to ignore
	IgnorePatterns []string
	// How frequently to check to see if a job is done, in milliseconds
	DoneScanMilli int
	// DelayMilli determines how long to wait between fetches for a given worker
	DelayMilli int
	// StopAfterEntries kills the crawler after a specified number of urls have
	// been visited. a value of 0 (the default) doesn't limit the number of entries
	StopAfterEntries int
	// StopUrl will stop the crawler after crawling a given URL
	StopURL string
	// BackoffResponseCodes is a list of response codes that when encountered will add
	// half the value of of CrawlDelayMilliseconds per request, slowing the crawl in response
	// every minute
	BackoffResponseCodes []int
	// MaxAttempts is the maximum number of times to try a url before giving up
	MaxAttempts int

	// Workers specifies configuration details for workers this job would like to
	// be sent to. The coordinator that orchestrates this job will take care of
	// worker allocation
	Workers []*WorkerConfig
	// ResourceHandler specifies where the results of completed requests should
	// be routed to. The coordinator that orchestarates this job will take care
	// of ResourceHandler allocation & routing
	ResourceHandlers []*ResourceHandlerConfig
}

// DefaultJobConfig creates a job configuration
func DefaultJobConfig() *JobConfig {
	return &JobConfig{
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
	}
}

// JSONJobConfigFromFilepath returns a func that reads a json-encoded
// config if the file specified by filepath exists, failing silently if no
// file is present. If a file is present but not valid json, the program panics
func JSONJobConfigFromFilepath(path string) func(*JobConfig) {
	return func(c *JobConfig) {
		cfg := &JobConfig{}
		if err := readJSONFile(path, cfg); err != nil {
			panic(err)
		}
		log.Infof("using config file: %s", path)
		*c = *cfg
	}
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

package lib

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// NewJob creates a Job
func NewJob(cfg *JobConfig, coord Coordinator) *Job {
	c := &Job{
		ID:         newJobID(),
		cfg:        cfg,
		coord:      coord,
		crawlDelay: time.Duration(cfg.DelayMilli) * time.Millisecond,
	}

	c.domains = make([]*url.URL, len(cfg.Domains))
	for i, rawurl := range cfg.Domains {
		u, err := url.Parse(rawurl)
		if err != nil {
			err = fmt.Errorf("error parsing configured domain: %s", err.Error())
			log.Error(err.Error())
			return c
		}
		c.domains[i] = u
	}

	return c
}

// Job is the central reporting hub for a crawl. It's in charge of populating
// the queue & keeping up-to-date records in the fetch request store. workers post their
// completed work back to the Job, which sends the created resources to any
// registered resource handlers
type Job struct {
	// id for this crawl
	ID string
	// Current job execution state, managed by coordinator
	status JobStatus
	// If execution errors, it's value should be set here
	err error
	// time crawler started
	start time.Time
	// finished is a count of the total number of urls finished
	finished int
	// cfg embeds this crawl's configuration
	cfg *JobConfig
	// domains is a list of domains to fetch from
	domains []*url.URL
	// crawlDelay is the current delay between requests on fetchbots
	// if Backoff is enabled this can get higher than cfg.CrawlDelayMilliseconds
	crawlDelay time.Duration
	// coordinator that owns this job
	coord Coordinator
	// cahnnel to halt job
	stop chan bool
}

// JobStatus tracks the state of a job
type JobStatus uint8

const (
	// JobStatusNew indicates a newly-created job
	JobStatusNew JobStatus = iota
	// JobStatusRunning indicates a job is running
	JobStatusRunning
	// JobStatusPaused indicates a job is paused
	JobStatusPaused
	// JobStatusComplete indicates a job is finished
	JobStatusComplete
	// JobStatusErrored indicates a job is errored
	JobStatusErrored
)

// String implements the stringer interface for job Status
func (js JobStatus) String() string {
	switch js {
	case JobStatusNew:
		return "new"
	case JobStatusRunning:
		return "running"
	case JobStatusPaused:
		return "paused"
	case JobStatusComplete:
		return "complete"
	case JobStatusErrored:
		return "errored"
	}
	return "unknown"
}

func newJobID() string {
	return strconv.Itoa(1)
}

// Config exposes the Job configuration
func (c *Job) Config() *JobConfig {
	return c.cfg
}

// Start kicks off coordinated fetching, seeding the queue & store & awaiting responses
// start will block until a signal is received on the stop channel, keep in mind
// a number of conditions can stop the crawler depending on configuration
func (c *Job) Start() (err error) {
	var (
		backoffT *time.Ticker
		// seedr                           io.Reader
		// unfetchedT, doneScanT *time.Ticker
		// finalizerErrs                   []error
		// wg sync.WaitGroup
	)
	c.stop = make(chan bool)

	if len(c.cfg.BackoffResponseCodes) > 0 {
		backoffT = time.NewTicker(time.Minute)
		go func() {
			for range backoffT.C {
				if c.crawlDelay > time.Duration(c.cfg.DelayMilli)*time.Millisecond {
					log.Infof("speeding up crawler")
					c.setCrawlDelay(c.crawlDelay - (time.Duration(c.cfg.DelayMilli)*time.Millisecond)/2)
				}
			}
		}()
	}

	c.start = time.Now()

	// block until receive on stop
	<-c.stop

	// log.Infof("%d urls remain in que for checking and processing", len(c.next))
	// if unfetchedT != nil {
	// 	unfetchedT.Stop()
	// }
	if backoffT != nil {
		backoffT.Stop()
	}

	// for _, rh := range c.ResourceHandlers() {
	// 	if finalizer, ok := rh.(ResourceFinalizer); ok {
	// 		wg.Add(1)
	// 		go func(t string, f ResourceFinalizer, errs *[]error) {
	// 			if err := f.FinalizeResources(); err != nil {
	// 				*errs = append(*errs, fmt.Errorf("%s: %s", t, err))
	// 			}
	// 			wg.Done()
	// 		}(rh.Type(), finalizer, &finalizerErrs)
	// 	}
	// }
	// wg.Wait()

	// if len(finalizerErrs) > 0 {
	// 	msg := "errors occured during finalization:\n"
	// 	for _, e := range finalizerErrs {
	// 		msg += fmt.Sprintf("%s\n", e.Error())
	// 	}
	// 	return fmt.Errorf(msg)
	// }

	return nil
}

// Errored sets the current job state to errored & retains the error
func (c *Job) Errored(err error) {
	c.status = JobStatusErrored
	c.err = err
}

// Complete marks the job as finished
func (c *Job) Complete() {
	c.stop <- true
	c.status = JobStatusComplete
}

// Seeds produces a channel of seed URLS to enqueue
func (c *Job) Seeds() (seeds chan string, err error) {
	seeds = make(chan string)

	var seedr io.Reader
	if seedr, err = c.enqueSeedsPath(); err != nil {
		return nil, fmt.Errorf("getting SeedsPath: %s", err.Error())
	}

	go func(c *Job, seedr io.Reader) {
		for _, url := range c.cfg.Seeds {
			seeds <- url
		}

		if seedr != nil {
			s := bufio.NewScanner(seedr)
			for s.Scan() {
				seeds <- s.Text()
			}
		}

		close(seeds)
	}(c, seedr)

	return
}

func (c *Job) enqueSeedsPath() (r io.Reader, err error) {
	if c.cfg.SeedsPath == "" {
		return nil, nil
	}

	if _, err := url.ParseRequestURI(c.cfg.SeedsPath); err == nil {
		log.Info("fetching SeedsPath URL")
		res, err := http.Get(c.cfg.SeedsPath)
		if err != nil {
			return nil, err
		}
		data, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return nil, err
		}
		res.Body.Close()
		return bytes.NewBuffer(data), nil
	}

	log.Info("reading SeedsPath file")
	data, err := ioutil.ReadFile(c.cfg.SeedsPath)
	if err != nil {
		return nil, err
	}
	return bytes.NewBuffer(data), nil
}

func (c *Job) setCrawlDelay(d time.Duration) {
	c.crawlDelay = d
	// for _, c := range c.workers {
	// 	c.SetDelay(d)
	// }
	log.Infof("crawler delay is now: %f seconds", c.crawlDelay.Seconds())
	// log.Infof("crawler request frequency: ~%f req/minute", float64(len(c.workers))/c.crawlDelay.Minutes())
}

// urlStringIsCandidate scans the slice of crawlingURLS to see if we should GET
// the passed-in url
// TODO - this is slated to eventually not be a simple list of ignored URLs,
// but a list of regexes or some other special pattern.
func (c *Job) urlStringIsCandidate(rawurl string) bool {
	for _, ignore := range c.cfg.IgnorePatterns {
		if strings.Contains(rawurl, ignore) {
			return false
		}
	}

	u, err := url.Parse(rawurl)
	if err != nil {
		return false
	}
	for _, d := range c.domains {
		if d.Host != u.Host {
			continue
		} else if u.Path != "" && !strings.HasPrefix(u.Path, d.Path) {
			return false
		}

		return true
	}
	return false
}

func (c *Job) okResponseStatus(s int) bool {
	return s >= http.StatusOK && s <= http.StatusPermanentRedirect
}

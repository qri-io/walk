package lib

import (
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"
)

// WorkCoordinator can coordinate workers. Workers pull from the output Requests
// channel and post finished resources using the completed method. This is the
// minimum interface a worker should need to turn Requests into Resources
type WorkCoordinator interface {
	// Queue returns a channel of Requests, which contain urls that need
	// to be fetched & turned into one or more resources
	Queue() (chan *Request, error)
	// Completed work is submitted to the coordinator by submitting one or more
	// constructed resources
	Completed(rsc ...*Resource) error
}

// Coordinator is the central reporting hub for a crawl. It's in charge of populating
// the queue & keeping up-to-date records in the fetch request store. workers post their
// completed work back to the coordinator, which sends the created resources to any
// registered resource handlers
type Coordinator struct {
	// time crawler started
	start time.Time
	// how many urls have been fetched and written to urls
	urlsWritten int

	// cfg embeds this crawl's configuration
	cfg *CoordinatorConfig

	// domains is a list of domains to fetch from
	domains []*url.URL

	queue    Queue
	frs      RequestStore
	handlers []ResourceHandler
	workers  []Worker

	// crawlDelay is the current delay between requests on fetchbots
	// if Backoff is enabled this can get higher than cfg.CrawlDelayMilliseconds
	crawlDelay time.Duration

	// flag indicating crawler is stopping
	stopping bool
	// finished is a count of the total number of urls finished
	finished int
}

// NewWalkJob creates a new walk write process from a given set of configurations
// if no configuration is provided, the default is used
// start the walk by calling Start on the returned coordinator
// halt the walk by sending a value on the returned stop channel
func NewWalkJob(configs ...func(*Config)) (coord *Coordinator, stop chan bool, err error) {
	// combine configurations with default
	cfg := ApplyConfigs(configs...)

	// create queue, store, workers, and handlers
	// TODO - needs to leverage config
	queue := NewMemQueue()
	// TODO - needs to leverage config
	frs := NewMemRequestStore()
	ws, err := NewWorkers(cfg.Workers)
	if err != nil {
		return
	}
	hs, err := NewResourceHandlers(cfg)
	if err != nil {
		return
	}

	// create coodinator
	coord = NewCoordinator(cfg.Coordinator, queue, frs, hs)
	stop = make(chan bool)

	// start workers
	for _, w := range ws {
		w.Start(coord)
	}

	return
}

// Config exposes the coordinator configuration
func (c *Coordinator) Config() *CoordinatorConfig {
	return c.cfg
}

// NewCoordinator creates a Coordinator
func NewCoordinator(cfg *CoordinatorConfig, q Queue, frs RequestStore, rh []ResourceHandler) *Coordinator {
	c := &Coordinator{
		cfg:        cfg,
		queue:      q,
		frs:        frs,
		handlers:   rh,
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

// SetHandlers configures the coordinator's resource handlers
func (c *Coordinator) SetHandlers(rh []ResourceHandler) error {
	if !c.start.IsZero() {
		return fmt.Errorf("crawl already started")
	}
	c.handlers = rh
	return nil
}

// ResourceHandlers exposes the coordinator's ResourceHandlers
func (c *Coordinator) ResourceHandlers() []ResourceHandler {
	return c.handlers
}

// Start kicks off coordinated fetching, seeding the queue & store & awaiting responses
// start will block until a signal is received on the stop channel, keep in mind
// a number of conditions can stop the crawler depending on configuration
func (c *Coordinator) Start(stop chan bool) error {
	var (
		unfetchedT, backoffT, doneScanT *time.Ticker
		finalizerErrs                   []error
		wg                              sync.WaitGroup
	)

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

	if c.cfg.DoneScanMilli > 0 {
		doneScanT = time.NewTicker(time.Millisecond * time.Duration(c.cfg.DoneScanMilli))
		log.Debugf("performing done scan checks every %d secs.", c.cfg.DoneScanMilli/1000)
		go func() {
			for range doneScanT.C {
				l, err := c.queue.Len()
				if err != nil {
					log.Errorf("error getting queue length: %s", err.Error())
					continue
				}
				if l == 0 {
					reqs, err := c.frs.List(-1, 0)
					if err != nil {
						log.Errorf("error reading: %s", err.Error())
						continue
					}
					for _, r := range reqs {
						if !(r.Status == RequestStatusDone || r.Status == RequestStatusFailed) {
							continue
						}
					}
					log.Info("no urls remain for checking, nothing left in queue, we done")
					stop <- true
					return
				}
			}
		}()
	}

	c.start = time.Now()
	for _, url := range c.cfg.Seeds {
		c.enqueue(&Request{URL: url})
	}

	// block until receive on stop
	<-stop

	// TODO - send stop signal to workers

	// log.Infof("%d urls remain in que for checking and processing", len(c.next))
	if unfetchedT != nil {
		unfetchedT.Stop()
	}
	if backoffT != nil {
		backoffT.Stop()
	}

	for _, rh := range c.ResourceHandlers() {
		if finalizer, ok := rh.(ResourceFinalizer); ok {
			wg.Add(1)
			go func(t string, f ResourceFinalizer, errs *[]error) {
				if err := f.FinalizeResources(); err != nil {
					*errs = append(*errs, fmt.Errorf("%s: %s", t, err))
				}
				wg.Done()
			}(rh.Type(), finalizer, &finalizerErrs)
		}
	}
	wg.Wait()

	if len(finalizerErrs) > 0 {
		msg := "errors occured during finalization:\n"
		for _, e := range finalizerErrs {
			msg += fmt.Sprintf("%s\n", e.Error())
		}
		return fmt.Errorf(msg)
	}

	return nil
}

// Queue gives access to the underlying queue as a channel of Fetch Requests
func (c *Coordinator) Queue() (chan *Request, error) {
	return c.queue.Chan()
}

// Completed sends one or more constructed resources to the coordinator
func (c *Coordinator) Completed(rsc ...*Resource) error {

	// handle any global state changes that may result from completed work
	// TODO - finish
	go func() {
		// for _, resc := range c.cfg.BackoffResponseCodes {
		// 	if res.StatusCode == resc {
		// 		log.Infof("encountered %d response. backing off", resc)
		// 		c.setCrawlDelay(c.crawlDelay + ((time.Duration(c.cfg.CrawlDelayMilliseconds) * time.Millisecond) / 2))
		// 	}
		// }
		// if c.finished == c.cfg.StopAfterEntries {
		// 	stop <- true
		// }
	}()

	// handle resources and create a deduplicated map
	// of unique candidate urls from all responses
	links := map[string]bool{}
	for _, r := range rsc {
		if err := c.dequeue(r); err != nil {
			log.Debugf("error dequing url: %s: %s", r.URL, err.Error())
		}

		if c.cfg.Crawl {
			for _, l := range r.Links {
				if c.urlStringIsCandidate(l) {
					links[l] = true
				}
			}
		}
	}

	for url := range links {
		r, err := c.frs.Get(url)
		if err != nil {
			log.Debugf("err getting url: %s: %s", url, err.Error())
		}
		if r == nil {
			c.enqueue(&Request{URL: url})
		}
	}

	return nil
}

func (c *Coordinator) enqueue(rs ...*Request) {
	for _, r := range rs {
		if c.stopping {
			r.Status = RequestStatusFailed
			c.frs.Put(r)
			continue
		}

		log.Debugf("enqueue: %s", r.URL)
		r.Status = RequestStatusQueued
		c.frs.Put(r)
		c.queue.Push(r)
	}
}

func (c *Coordinator) dequeue(rsc *Resource) error {
	fr, err := c.frs.Get(rsc.URL)
	if err == ErrNotFound {
		fr = &Request{URL: rsc.URL}
	} else if err != nil {
		log.Debugf("err getting url: %s: %s", rsc.URL, err.Error())
		return err
	}

	fr.PrevResStatus = rsc.Status
	fr.AttemptsMade++

	if c.okResponseStatus(fr.PrevResStatus) {
		log.Debugf("dequeue: %s", fr.URL)
		c.finished++
		c.urlsWritten++
		fr.Status = RequestStatusDone
		// send completed records to each handler
		for _, h := range c.handlers {
			go h.HandleResource(rsc)
		}
		return nil
	}

	if fr.AttemptsMade <= c.cfg.MaxAttempts {
		c.enqueue(fr)
		return nil
	}

	fr.Status = RequestStatusFailed
	return c.frs.Put(fr)
}

func (c *Coordinator) setCrawlDelay(d time.Duration) {
	c.crawlDelay = d
	for _, c := range c.workers {
		c.SetDelay(d)
	}
	log.Infof("crawler delay is now: %f seconds", c.crawlDelay.Seconds())
	log.Infof("crawler request frequency: ~%f req/minute", float64(len(c.workers))/c.crawlDelay.Minutes())
}

// urlStringIsCandidate scans the slice of crawlingURLS to see if we should GET
// the passed-in url
// TODO - this is slated to eventually not be a simple list of ignored URLs,
// but a list of regexes or some other special pattern.
func (c *Coordinator) urlStringIsCandidate(rawurl string) bool {
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

func (c *Coordinator) okResponseStatus(s int) bool {
	return s == 200
}

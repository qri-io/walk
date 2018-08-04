package lib

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/fetchbot"
)

const queueBufferSize = 5000

// Crawl holds state of a general crawl, including any number of
// "fetchbots", set by config.Parallism
type Crawl struct {
	// time crawler started
	start time.Time

	// cfg embeds this crawl's configuration
	cfg Config

	// domains is a list of domains to fetch from
	domains []*url.URL
	// urlLock protects access to urls domains map
	urlLock sync.Mutex
	// crawled is the list of stuff that's been crawled
	urls map[string]*URL

	// crawlers is a slice of all the executing crawlers
	crawlers []*fetchbot.Fetcher
	// queues holds each crawler's que
	queues []*fetchbot.Queue
	// queLock protects access to the queued map
	queLock sync.Mutex
	// queued keeps track of urls that are currently queued
	queued map[string]bool

	// next is a channel of candidate urls the crawlers seed from
	next chan string

	// crawlDelay is the current delay between requests on fetchbots
	// if Backoff is enabled this can get higher than cfg.CrawlDelayMilliseconds
	crawlDelay time.Duration

	// flag indicating crawler is stopping
	stopping bool
	// finished is a count of the total number of urls finished
	finished int
	// how long should pass before we re-visit a url, 0 means
	// urls are never stale (only visit once)
	staleDuration time.Duration
	// how many batches have been written
	batchCount int
	// how many urls have been fetched and written to urls
	urlsWritten int
}

// NewCrawl creates a Crawl struct
func NewCrawl(options ...func(*Config)) *Crawl {
	cfg := DefaultConfig()
	for _, opt := range options {
		opt(&cfg)
	}

	c := &Crawl{
		urls:          map[string]*URL{},
		queued:        map[string]bool{},
		next:          make(chan string, queueBufferSize),
		cfg:           cfg,
		crawlDelay:    time.Duration(cfg.CrawlDelayMilliseconds) * time.Millisecond,
		staleDuration: time.Duration(cfg.StaleDurationHours) * time.Hour,
	}

	c.LoadSitemapFile(c.cfg.SrcPath)

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

// FilterStrings filters a slice of strings using a match function
func FilterStrings(strings []string, match func(string) bool) []string {
	if strings == nil || len(strings) == 0 {
		return nil
	}

	i := 0
	res := make([]string, len(strings))
	for _, l := range strings {
		if match(l) {
			res[i] = l
			i++
		}
	}
	return res[:i]
}

// Start initiates the crawler
func (c *Crawl) Start(stop chan bool) error {
	var (
		mux                  fetchbot.Handler
		httpcli              *http.Client
		unfetchedT, backoffT *time.Ticker
	)

	mux = newMux(c, stop)
	httpcli = newClient(c)

	if c.cfg.StopURL != "" {
		mux = stopHandler(c.cfg.StopURL, stop, mux)
	}

	c.crawlers = make([]*fetchbot.Fetcher, c.cfg.Parallelism)
	for i := 0; i < c.cfg.Parallelism; i++ {
		fb := newFetchbot(i, c, mux, httpcli)
		c.crawlers[i] = fb
	}

	start := time.Now()
	qs, stopCrawling, done := c.startCrawling(c.crawlers)
	c.queues = qs

	if c.cfg.UnfetchedScanFreqMilliseconds > 0 {
		d := time.Millisecond * time.Duration(c.cfg.UnfetchedScanFreqMilliseconds)
		log.Infof("checking for unfetched urls every %f seconds", d.Seconds())
		unfetchedT = time.NewTicker(d)
		go func() {
			for range unfetchedT.C {
				if !c.stopping {
					if len(c.next) == queueBufferSize {
						log.Infof("next queue is full, skipping scan")
						continue
					}

					log.Infof("scanning for unfetched links")
					ufl := c.gatherUnfetchedLinks(250, stop)
					for linkurl := range ufl {
						c.next <- linkurl
					}
					log.Infof("seeded %d unfetched links from interval", len(ufl))
				}
			}
		}()
	}

	if len(c.cfg.BackoffResponseCodes) > 0 {
		backoffT = time.NewTicker(time.Minute)
		go func() {
			for range backoffT.C {
				if c.crawlDelay > time.Duration(c.cfg.CrawlDelayMilliseconds)*time.Millisecond {
					log.Infof("speeding up crawler")
					c.setCrawlDelay(c.crawlDelay - (time.Duration(c.cfg.CrawlDelayMilliseconds)*time.Millisecond)/2)
				}
			}
		}()
	}

	go func() {
		<-stop

		if c.cfg.BackupWriteInterval > 0 {
			path := fmt.Sprintf("%s.backup", c.cfg.DestPath)
			log.Infof("writing backup sitemap: %s", path)
			if err := c.WriteJSON(path); err != nil {
				log.Errorf("error writing backup sitemap: %s", err.Error())
			}
		}

		log.Infof("%d urls remain in que for checking and processing", len(c.next))
		if unfetchedT != nil {
			unfetchedT.Stop()
		}
		if backoffT != nil {
			backoffT.Stop()
		}

		stopCrawling <- true
	}()

	go func(qs []*fetchbot.Queue) {
		// make sure each fetchbot pulls at least one url to get it started
		for _, q := range qs {
			c.queNextURL(q)
		}
	}(qs)

	log.Infof("crawl started at %s", start)
	<-done
	return nil
}

func (c *Crawl) setCrawlDelay(d time.Duration) {
	c.crawlDelay = d
	for _, c := range c.crawlers {
		c.CrawlDelay = d
	}
	log.Infof("crawler delay is now: %f seconds", c.crawlDelay.Seconds())
	log.Infof("crawler request frequency: ~%f req/minute", float64(len(c.crawlers))/c.crawlDelay.Minutes())
}

// startCrawling initializes the crawler, queue, stopCrawler channel, and
// crawlingURLS slice
func (c *Crawl) startCrawling(crawlers []*fetchbot.Fetcher) (qs []*fetchbot.Queue, stopCrawling, done chan bool) {
	log.Infof("starting crawl with %d crawlers for %d domains", len(crawlers), len(c.domains))
	freq := time.Duration(c.cfg.CrawlDelayMilliseconds) * time.Millisecond
	log.Infof("crawler request frequency: ~%f req/minute", float64(len(crawlers))/freq.Minutes())
	stopCrawling = make(chan bool)
	done = make(chan bool)
	qs = make([]*fetchbot.Queue, len(crawlers))

	wg := sync.WaitGroup{}

	for i, fetcher := range crawlers {
		// Start processing
		qs[i] = fetcher.Start()
		time.Sleep(time.Second)
		wg.Add(1)

		go func(i int) {
			qs[i].Block()
			log.Infof("finished crawler: %d", i)
			wg.Done()
		}(i)
	}

	go func() {
		<-stopCrawling
		c.stopping = true
		log.Info("stopping crawlers")
		for _, q := range qs {
			q.Close()
		}
	}()

	go func() {
		wg.Wait()
		log.Info("done")
		done <- true
	}()

	seeds := FilterStrings(c.cfg.Seeds, c.urlStringIsCandidate)
	log.Infof("adding %d/%d seed urls", len(seeds), len(c.cfg.Seeds))

	for i, s := range seeds {
		if i < len(qs) {
			c.addURLToQue(qs[i], s)
			continue
		}
		c.next <- s
	}
	return
}

func (c *Crawl) unqueURLs(remove ...string) {
	if len(remove) > 0 {
		c.queLock.Lock()
		defer c.queLock.Unlock()
		for _, rm := range remove {
			c.queued[rm] = false
		}
	}
}

func (c *Crawl) queNextURL(q *fetchbot.Queue) {
	for {
		rawurl := <-c.next
		if err := c.addURLToQue(q, rawurl); err == nil {
			break
		}
	}
}

var (
	errAlreadyFetched  = fmt.Errorf("already fetched")
	errInvalidFetchURL = fmt.Errorf("invalid url for fetching")
	errInQueue         = fmt.Errorf("url is already queued for fetching")
)

func (c *Crawl) addURLToQue(q *fetchbot.Queue, rawurl string) error {
	u, err := url.Parse(rawurl)
	if err != nil {
		return err
	}

	if err := isWebpageURL(u); err != nil {
		return errInvalidFetchURL
	}

	normurl := NormalizeURLString(u)

	c.urlLock.Lock()
	if c.urls[normurl] != nil {
		c.urlLock.Unlock()
		return errAlreadyFetched
	}
	c.urlLock.Unlock()

	c.queLock.Lock()
	defer c.queLock.Unlock()

	if c.queued[normurl] {
		return errInQueue
	}

	c.queued[normurl] = true
	tg, err := NewTimedGet(normurl)
	if err != nil {
		return err
	}
	if err := q.Send(tg); err != nil {
		return err
	}
	return nil
}

func (c *Crawl) gatherUnfetchedLinks(max int, stop chan bool) map[string]bool {
	c.urlLock.Lock()
	defer c.urlLock.Unlock()

	links := make(map[string]bool, max)
	i := 0

	for _, u := range c.urls {
		if u != nil && u.Status == 200 {
			for _, linkurl := range u.Links {
				if c.urlStringIsCandidate(linkurl) {
					linku := c.urls[linkurl]
					if (linku == nil || linku.Status != http.StatusOK) && links[linkurl] == false {
						links[linkurl] = true
						i++
						if i == max {
							log.Infof("found max of %d unfetched links", max)
							return links
						}
					}
				}
			}
		}
	}

	if i > 0 {
		log.Infof("found %d unfetched links", i)
	} else if stop != nil {
		log.Infof("no unfetched links found. sending stop")
		stop <- true
	}

	return links
}

// urlStringIsCandidate scans the slice of crawlingURLS to see if we should GET
// the passed-in url
func (c *Crawl) urlStringIsCandidate(rawurl string) bool {
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
		if u.Path != "" && !strings.Contains(u.Path, d.Path) {
			return false
		}
		if d.Host == u.Host {
			return true
		}
	}
	return false
}

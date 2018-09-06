package lib

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/PuerkitoBio/fetchbot"
)

// Worker is the interface turning Requests into Resources
// by performing fetches
type Worker interface {
	SetDelay(time.Duration)
	Start(coord WorkCoordinator) error
	Stop() error
}

// NewWorkers creates a slice of Workers from a slice of Worker configs
func NewWorkers(wc []*WorkerConfig) (ws []Worker, err error) {
	for _, cfg := range wc {
		w, err := NewWorker(cfg)
		if err != nil {
			return nil, err
		}
		ws = append(ws, w)
	}
	return
}

// NewWorker creates a new worker for a given configuration
func NewWorker(cfg *WorkerConfig) (w Worker, err error) {
	switch cfg.Type {
	case "local":
		return NewLocalWorker(cfg), nil
	default:
		return nil, fmt.Errorf("unrecognized worker type: %s", cfg.Type)
	}
}

// LocalWorker is an in-process implementation of worker
// TODO - finish parallelism implementation
type LocalWorker struct {
	coord    WorkCoordinator
	stop     chan bool
	cfg      *WorkerConfig
	fetchers []*fetchbot.Fetcher
	queues   []*fetchbot.Queue
}

// NewLocalWorker creates a LocalWorker with crawl configuration settings
func NewLocalWorker(cfg *WorkerConfig) *LocalWorker {
	if cfg.Parallelism == 0 {
		cfg.Parallelism = 1
	}
	return &LocalWorker{
		stop: make(chan bool),
		cfg:  cfg,
	}
}

// SetDelay configures the delay between requests
func (w *LocalWorker) SetDelay(d time.Duration) {
	// TODO!
}

// Start the local worker reporting results to the given coordinator
func (w *LocalWorker) Start(coord WorkCoordinator) error {
	w.coord = coord
	cfg := w.cfg
	w.fetchers = make([]*fetchbot.Fetcher, cfg.Parallelism)
	w.queues = make([]*fetchbot.Queue, cfg.Parallelism)

	ch, err := w.coord.Queue()
	if err != nil {
		return err
	}

	for i := 0; i < cfg.Parallelism; i++ {
		f := fetchbot.New(newMux(coord, cfg.RecordRedirects, cfg.RecordResponseHeaders))
		f.DisablePoliteness = !cfg.Polite
		f.CrawlDelay = time.Duration(cfg.DelayMilli) * time.Millisecond
		f.UserAgent = cfg.UserAgent
		if cfg.RecordRedirects {
			f.HttpClient = NewRecordRedirectClient(coord)
		}

		w.fetchers[i] = f
		w.queues[i] = f.Start()
	}

	go func() {
		i := 0
		for {
			select {
			case fr := <-ch:
				tg, err := NewTimedGet(fr.URL)
				if err != nil {
					log.Error(err.Error())
					continue
				}
				if err := w.queues[i].Send(tg); err != nil {
					log.Error(err.Error())
					continue
				}
				i++
				if i == len(w.queues) {
					i = 0
				}
			case <-w.stop:
				for _, q := range w.queues {
					q.Close()
				}
				break
			}
		}
	}()

	return nil
}

// Stop the worker
func (w *LocalWorker) Stop() error {
	w.stop <- true
	return nil
}

// newMux creates a muxer (response multiplexer) for a fetchbot
func newMux(coord WorkCoordinator, recordRedirects, recordHeaders bool) *fetchbot.Mux {
	// Create the muxer
	mux := fetchbot.NewMux()

	// Handle all errors the same
	mux.HandleErrors(fetchbot.HandlerFunc(func(ctx *fetchbot.Context, res *http.Response, err error) {
		log.Infof("[ERR] %s %s - %s", ctx.Cmd.Method(), ctx.Cmd.URL(), err.Error())
		coord.Completed(&Resource{Error: err.Error()})
		return
	}))

	// Handle GET requests for html responses, to parse the body and enqueue all links as HEAD requests.
	mux.Response().Method("GET").Handler(fetchbot.HandlerFunc(
		func(ctx *fetchbot.Context, res *http.Response, err error) {

			r := &Resource{URL: ctx.Cmd.URL().String()}

			// TODO - huh? why this here? figure out & make a note
			if recordRedirects {
				r = &Resource{URL: NormalizeURL(res.Request.URL)}
			}

			log.Infof("[%d] %s %s", res.StatusCode, ctx.Cmd.Method(), r.URL)

			var st time.Time
			if timedCmd, ok := ctx.Cmd.(*TimedCmd); ok {
				st = timedCmd.Started
			}

			if err := r.HandleResponse(st, res, recordHeaders); err != nil {
				log.Debugf("error handling get response: %s - %s", ctx.Cmd.URL().String(), err.Error())
				return
			}

			if err := coord.Completed(r); err != nil {
				log.Errorf("[ERR] coordinator: %s", err.Error())
			}
		}))

	return mux
}

// stopHandler stops the fetcher if the stopurl is reached. Otherwise it dispatches
// the call to the wrapped Handler.
func stopHandler(stopurl string, stop chan bool, wrapped fetchbot.Handler) fetchbot.Handler {
	return fetchbot.HandlerFunc(func(ctx *fetchbot.Context, res *http.Response, err error) {
		if ctx.Cmd.URL().String() == stopurl {
			log.Infof(">>>>> STOP URL %s\n", ctx.Cmd.URL())
			// generally not a good idea to stop/block from a handler goroutine
			// so do it in a separate goroutine
			go func() {
				stop <- true
			}()
			return
		}
		wrapped.Handle(ctx, res, err)
	})
}

// NewRecordRedirectClient creates a http client with a custom checkRedirect function that
// creates records of Redirects & sends them to the coordinator
func NewRecordRedirectClient(wc WorkCoordinator) *http.Client {
	return &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {

			prev := via[len(via)-1]
			prevurl := NormalizeURL(prev.URL)

			r, _ := url.Parse(req.URL.String())
			canurlstr := NormalizeURL(r)

			if prevurl != canurlstr {
				log.Infof("[%d] %s %s -> %s", req.Response.StatusCode, prev.Method, prevurl, canurlstr)

				wc.Completed(&Resource{
					URL:        prevurl,
					Timestamp:  time.Now(),
					Status:     req.Response.StatusCode,
					RedirectTo: canurlstr,
				})
			}

			if len(via) >= 10 {
				return fmt.Errorf("stopped after 10 redirects")
			}
			return nil
		},
	}
}

// TimedCmd defines a Command implementation that sets an internal timestamp
// whenever it's URL method is called
type TimedCmd struct {
	U       *url.URL
	M       string
	Started time.Time
}

// NewTimedGet creates a new GET command with an internal Timer
func NewTimedGet(rawurl string) (*TimedCmd, error) {
	u, err := url.Parse(rawurl)
	if err != nil {
		return nil, err
	}
	return &TimedCmd{
		U: u,
		M: "GET",
	}, nil
}

// URL returns the resource targeted by this command.
func (c *TimedCmd) URL() *url.URL {
	if c.Started.IsZero() {
		c.Started = time.Now()
	}
	return c.U
}

// Method returns the HTTP verb to use to process this command (i.e. "GET", "HEAD", etc.).
func (c *TimedCmd) Method() string {
	return c.M
}

// Start initiates the crawler
// func (c *Coordinator) Start(stop chan bool) error {
//  var (
//    mux                  fetchbot.Handler
//    httpcli              *http.Client
//    unfetchedT, backoffT *time.Ticker
//  )

//  mux = newMux(c, stop)
//  httpcli = newHTTPClient(c)

//  if c.cfg.StopURL != "" {
//    mux = stopHandler(c.cfg.StopURL, stop, mux)
//  }

//  c.crawlers = make([]*fetchbot.Fetcher, c.cfg.Parallelism)
//  for i := 0; i < c.cfg.Parallelism; i++ {
//    fb := newFetchbot(i, c, mux, httpcli)
//    c.crawlers[i] = fb
//  }

//  start := time.Now()
//  qs, stopCrawling, done := c.startCrawling(c.crawlers)
//  c.queues = qs

//  if c.cfg.UnfetchedScanFreqMilliseconds > 0 {
//    d := time.Millisecond * time.Duration(c.cfg.UnfetchedScanFreqMilliseconds)
//    log.Infof("checking for unfetched urls every %f seconds", d.Seconds())
//    unfetchedT = time.NewTicker(d)
//    go func() {
//      for range unfetchedT.C {
//        if !c.stopping {
//          if len(c.next) == queueBufferSize {
//            log.Infof("next queue is full, skipping scan")
//            continue
//          }

//          log.Infof("scanning for unfetched links")
//          ufl := c.gatherUnfetchedLinks(250, stop)
//          for linkurl := range ufl {
//            c.next <- linkurl
//          }
//          log.Infof("seeded %d unfetched links from interval", len(ufl))
//        }
//      }
//    }()
//  }

//  go func(qs []*fetchbot.Queue) {
//    // make sure each fetchbot pulls at least one url to get it started
//    for _, q := range qs {
//      c.queNextURL(q)
//    }
//  }(qs)

//  log.Infof("crawl started at %s", start)
//  <-done
//  return nil
// }

// startCrawling initializes the crawler, queue, stopCrawler channel, and crawlingURLS slice
// func (c *Coordinator) startCrawling(crawlers []*fetchbot.Fetcher) (qs []*fetchbot.Queue, stopCrawling, done chan bool) {
//  log.Infof("starting crawl with %d crawlers for %d domains", len(crawlers), len(c.domains))
//  freq := time.Duration(c.cfg.CrawlDelayMilliseconds) * time.Millisecond
//  log.Infof("crawler request frequency: ~%f req/minute", float64(len(crawlers))/freq.Minutes())
//  stopCrawling = make(chan bool)
//  done = make(chan bool)
//  qs = make([]*fetchbot.Queue, len(crawlers))

//  wg := sync.WaitGroup{}

//  for i, fetcher := range crawlers {
//    // Start processing
//    qs[i] = fetcher.Start()
//    time.Sleep(time.Second)
//    wg.Add(1)

//    go func(i int) {
//      qs[i].Block()
//      log.Infof("finished crawler: %d", i)
//      wg.Done()
//    }(i)
//  }

//  go func() {
//    <-stopCrawling
//    c.stopping = true
//    log.Info("stopping crawlers")
//    for _, q := range qs {
//      q.Close()
//    }
//  }()

//  go func() {
//    wg.Wait()
//    log.Info("done")
//    done <- true
//  }()

//  seeds := FilterStrings(c.cfg.Seeds, c.urlStringIsCandidate)
//  log.Infof("adding %d/%d seed urls", len(seeds), len(c.cfg.Seeds))

//  for i, s := range seeds {
//    if i < len(qs) {
//      c.addURLToQue(qs[i], s)
//      continue
//    }
//    c.next <- s
//  }
//  return
// }

// func (c *Coordinator) unqueURLs(remove ...string) {
//  if len(remove) > 0 {
//    c.queLock.Lock()
//    defer c.queLock.Unlock()
//    for _, rm := range remove {
//      c.queued[rm] = false
//    }
//  }
// }

// func (c *Coordinator) queNextURL(q *fetchbot.Queue) {
//  for {
//    rawurl := <-c.next
//    if err := c.addURLToQue(q, rawurl); err == nil {
//      break
//    }
//  }
// }

// func (c *Coordinator) addURLToQue(q *fetchbot.Queue, rawurl string) error {
// 	u, err := url.Parse(rawurl)
// 	if err != nil {
// 		return err
// 	}

// 	if err := isWebpageURL(u); err != nil {
// 		return errInvalidFetchURL
// 	}

// 	normurl := NormalizeURLString(u)

// 	c.urlLock.Lock()
// 	if c.urls[normurl] != nil {
// 		c.urlLock.Unlock()
// 		return errAlreadyFetched
// 	}
// 	c.urlLock.Unlock()

// 	c.queLock.Lock()
// 	defer c.queLock.Unlock()

// 	if c.queued[normurl] {
// 		return errInQueue
// 	}

// 	return nil
// }

// func (c *Coordinator) gatherUnfetchedLinks(max int, stop chan bool) map[string]bool {
// 	c.urlLock.Lock()
// 	defer c.urlLock.Unlock()

// 	links := make(map[string]bool, max)
// 	i := 0

// 	for _, u := range c.urls {
// 		if u != nil && u.Status == 200 {
// 			for _, linkurl := range u.Links {
// 				if c.urlStringIsCandidate(linkurl) {
// 					linku := c.urls[linkurl]
// 					if (linku == nil || linku.Status != http.StatusOK) && links[linkurl] == false {
// 						links[linkurl] = true
// 						i++
// 						if i == max {
// 							log.Infof("found max of %d unfetched links", max)
// 							return links
// 						}
// 					}
// 				}
// 			}
// 		}
// 	}

// 	if i > 0 {
// 		log.Infof("found %d unfetched links", i)
// 	} else if stop != nil {
// 		log.Infof("no unfetched links found. sending stop")
// 		stop <- true
// 	}

// 	return links
// }

package lib

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/PuerkitoBio/fetchbot"
)

// Worker is the interface turning Requests into Resources by performing fetches
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
				i = (i + 1) % len(w.queues)
				time.Sleep(time.Duration(w.cfg.DelayMilli) * time.Millisecond)
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

			canurlstr, _ := NormalizeURLString(req.URL.String())

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

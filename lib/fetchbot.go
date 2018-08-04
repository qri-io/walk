package lib

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/fetchbot"
)

// newFetchbot creates a new fetchbot with crawl configuration settings
func newFetchbot(id int, c *Crawl, mux fetchbot.Handler, httpcli fetchbot.Doer) *fetchbot.Fetcher {
	// f := fetchbot.New(logHandler(id, mux))
	f := fetchbot.New(mux)
	f.DisablePoliteness = !c.cfg.Polite
	f.CrawlDelay = time.Duration(c.cfg.CrawlDelayMilliseconds) * time.Millisecond
	f.UserAgent = c.cfg.UserAgent
	f.HttpClient = httpcli
	return f
}

// muxer creates a new muxer (response multiplexer) for a crawler
func newMux(c *Crawl, stop chan bool) *fetchbot.Mux {
	// Create the muxer
	mux := fetchbot.NewMux()

	// Handle all errors the same
	mux.HandleErrors(fetchbot.HandlerFunc(func(ctx *fetchbot.Context, res *http.Response, err error) {
		if !strings.Contains(err.Error(), errAlreadyFetched.Error()) {
			log.Infof("[ERR] %s %s - %s", ctx.Cmd.Method(), ctx.Cmd.URL(), err.Error())
			c.urlLock.Lock()
			c.urls[ctx.Cmd.URL().String()] = &URL{Error: err.Error()}
			c.urlLock.Unlock()
		}

		c.unqueURLs(ctx.Cmd.URL().String())
		go c.queNextURL(ctx.Q)
	}))

	// Handle GET requests for html responses, to parse the body and enqueue all links as HEAD requests.
	mux.Response().Method("GET").Handler(fetchbot.HandlerFunc(
		func(ctx *fetchbot.Context, res *http.Response, err error) {

			u := &URL{URL: ctx.Cmd.URL().String()}

			if c.cfg.RecordRedirects {
				u = &URL{URL: NormalizeURLString(res.Request.URL)}
			}

			log.Infof("[%d] %s %s", res.StatusCode, ctx.Cmd.Method(), u.URL)

			var st time.Time
			if timedCmd, ok := ctx.Cmd.(*TimedCmd); ok {
				st = timedCmd.Started
			}

			if err := u.HandleGetResponse(st, res, c.cfg.RecordResponseHeaders); err != nil {
				log.Debugf("error handling get response: %s - %s", ctx.Cmd.URL().String(), err.Error())
				return
			}

			links := FilterStrings(u.Links, c.urlStringIsCandidate)
			unwritten := make([]string, len(links))

			c.urlLock.Lock()
			c.urls[u.URL] = u
			c.finished++
			c.urlsWritten++

			i := 0
			for _, l := range links {
				if c.urls[l] == nil {
					unwritten[i] = l
					i++
				}
			}
			unwritten = unwritten[:i]

			c.urlLock.Unlock()

			for _, resc := range c.cfg.BackoffResponseCodes {
				if res.StatusCode == resc {
					log.Infof("encountered %d response. backing off", resc)
					c.setCrawlDelay(c.crawlDelay + ((time.Duration(c.cfg.CrawlDelayMilliseconds) * time.Millisecond) / 2))
				}
			}

			if !c.stopping && len(c.next) < queueBufferSize {
				go func() {
					for _, l := range unwritten {
						c.next <- l
					}
					log.Infof("seeded %d/%d links for source: %s", len(unwritten), len(u.Links), u.URL)
				}()
			}

			if c.finished == c.cfg.StopAfterEntries {
				stop <- true
			}

			if c.cfg.BackupWriteInterval > 0 && (c.urlsWritten%c.cfg.BackupWriteInterval == 0) {
				go func() {
					path := fmt.Sprintf("%s.backup", c.cfg.DestPath)
					log.Infof("writing backup sitemap: %s", path)
					if err := c.WriteJSON(path); err != nil {
						log.Errorf("error writing backup sitemap: %s", err.Error())
					}
					c.batchCount++
				}()
			}

			c.unqueURLs(u.URL)
			go c.queNextURL(ctx.Q)

		}))

	return mux
}

// logHandler prints the fetch information and dispatches the call to the wrapped Handler.
// func logHandler(crawlerId int, wrapped fetchbot.Handler) fetchbot.Handler {
// 	return fetchbot.HandlerFunc(func(ctx *fetchbot.Context, res *http.Response, err error) {
// 		if err == nil {
// 			log.Infof("[%d] %s %d %s", res.StatusCode, ctx.Cmd.Method(), crawlerId, ctx.Cmd.URL())
// 		}
// 		wrapped.Handle(ctx, res, err)
// 	})
// }

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

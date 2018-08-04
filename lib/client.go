package lib

import (
	"fmt"
	"net/http"
	"net/url"
	"time"
)

func newClient(c *Crawl) *http.Client {
	if !c.cfg.RecordRedirects {
		return http.DefaultClient
	}

	return &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {

			prev := via[len(via)-1]
			prevurl := NormalizeURLString(prev.URL)

			r, _ := url.Parse(req.URL.String())
			canurlstr := NormalizeURLString(r)

			c.urlLock.Lock()
			defer c.urlLock.Unlock()

			if prevurl != canurlstr {
				log.Infof("[%d] %s %s -> %s", req.Response.StatusCode, prev.Method, prevurl, canurlstr)
				c.urls[prevurl] = &URL{
					URL:        prevurl,
					Timestamp:  time.Now(),
					Status:     req.Response.StatusCode,
					RedirectTo: canurlstr,
				}
				c.finished++
				c.urlsWritten++
			}

			if c.urls[canurlstr] != nil {
				return errAlreadyFetched
			}

			if len(via) >= 10 {
				return fmt.Errorf("stopped after 10 redirects")
			}
			return nil
		},
	}
}

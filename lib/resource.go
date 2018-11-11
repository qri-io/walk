package lib

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/PuerkitoBio/purell"
	"github.com/datatogether/ffi"
	"github.com/multiformats/go-multihash"
)

// Resource is data associated with a given URL at a point in time
type Resource struct {
	// A Url is uniquely identified by URI string without
	// any normalization. Url strings must always be absolute.
	URL string `json:"url"`
	// Timestamp of completed request
	Timestamp time.Time `json:"timestamp,omitempty"`
	// RequestDuration is the time remote server took to transfer content
	RequestDuration time.Duration `json:"duration,omitempty"`
	// Returned HTTP status code
	Status int `json:"status,omitempty"`
	// Returned HTTP 'Content-Type' header
	ContentType string `json:"contentType,omitempty"`
	// Result of mime sniffing to GET response body, as detailed at https://mimesniff.spec.whatwg.org
	ContentSniff string `json:"contentSniff,omitempty"`
	// ContentLength in bytes, will be the header value if only a HEAD request has been issued
	// After a valid GET response, it will be set to the length of the returned response
	ContentLength int64 `json:"contentLength,omitempty"`
	// HTML Title tag attribute
	Title string `json:"title,omitempty"`
	// key-value slice of returned headers from most recent HEAD or GET request
	// stored in the form [key,value,key,value...]
	Headers []string `json:"headers,omitempty"`
	// Hash is a base58 encoded multihash of res.Body
	Hash string `json:"hash,omitempty"`
	// Links
	Links []string `json:"links,omitempty"`
	// RedirectTo speficies where this url redirects to, cannonicalized
	RedirectTo string `json:"redirectTo,omitempty"`
	// Error contains any fetching error string
	Error string `json:"error,omitempty"`
	// contents of response body
	Body []byte `json:"body,omitempty"`
}

// HeadersMap formats u.Headers (a string slice) as a map[header]value
func (u *Resource) HeadersMap() (headers map[string]string) {
	headers = map[string]string{}
	for i, s := range u.Headers {
		if i%2 == 0 {
			headers[s] = u.Headers[i+1]
		}
	}
	return
}

// HandleResponse populates a resource based on an HTTP response
func (u *Resource) HandleResponse(started time.Time, res *http.Response, recordHeaders bool) (err error) {
	var doc *goquery.Document

	defer res.Body.Close()
	u.Body, err = ioutil.ReadAll(res.Body)
	if err != nil {
		return
	}

	u.Status = res.StatusCode
	u.ContentLength = int64(len(u.Body))
	u.ContentType = res.Header.Get("Content-Type")
	u.ContentSniff = http.DetectContentType(u.Body)
	u.Timestamp = time.Now()

	if recordHeaders {
		u.Headers = rawHeadersSlice(res)
	}

	if !started.IsZero() {
		u.RequestDuration = u.Timestamp.Sub(started)
		log.Debugf("%s took %s", u.URL, u.RequestDuration.String())
	}
	if mh, e := multihash.Sum(u.Body, multihash.SHA2_256, -1); e == nil {
		u.Hash = mh.String()
	}

	// additional processing for html documents.
	// sometimes xhtml documents can come back as text/plain, thus the text/plain addition
	if u.ContentSniff == "text/html; charset=utf-8" || u.ContentSniff == "text/plain; charset=utf-8" {
		// Process the body to find links
		doc, err = goquery.NewDocumentFromReader(bytes.NewBuffer(u.Body))
		if err != nil {
			return
		}

		u.Title = doc.Find("title").Text()
		err = u.ExtractDocLinks(doc)
		if err != nil {
			return
		}
	}

	return
}

// NormalizeURLString canonicalizes a URL
func NormalizeURLString(urlstr string) (string, error) {
	u, err := url.Parse(urlstr)
	if err != nil {
		return "", err
	}
	return NormalizeURL(u), nil
}

// NormalizeURL canonicalizes a URL
func NormalizeURL(u *url.URL) string {
	return purell.NormalizeURL(u, purell.FlagsUnsafeGreedy)
}

// construct a slice of [key,val,key,val,...] listing all response headers
func rawHeadersSlice(res *http.Response) (headers []string) {
	for key, val := range res.Header {
		headers = append(headers, []string{key, strings.Join(val, ",")}...)
	}
	return
}

// ExtractDocLinks extracts & stores a page's linked documents
// by selecting all a[href] links from a given qoquery document, using
// the receiver *Url as the base
func (u *Resource) ExtractDocLinks(doc *goquery.Document) error {
	pURL, err := url.Parse(u.URL)
	if err != nil {
		return err
	}

	// generate a list of normalized links
	doc.Find("[href]").Each(func(i int, s *goquery.Selection) {
		val, _ := s.Attr("href")

		// Resolve destination address to source url
		address, err := pURL.Parse(val)
		if err != nil {
			return
		}

		str := NormalizeURL(address)
		// deduplicate links
		for _, l := range u.Links {
			if str == l {
				return
			}
		}

		u.Links = append(u.Links, str)
	})

	return nil
}

var htmlMimeTypes = map[string]bool{
	"text/html":                 true,
	"text/html; charset=utf-8":  true,
	"text/plain; charset=utf-8": true,
	"text/xml; charset=utf-8":   true,
}

// htmlExtensions is a dictionary of "file extensions" that normally contain
// html content
var htmlExtensions = map[string]bool{
	".asp":   true,
	".aspx":  true,
	".cfm":   true,
	".html":  true,
	".net":   true,
	".php":   true,
	".xhtml": true,
	".":      true,
	"":       true,
}

var invalidSchemes = map[string]bool{
	"data":   true,
	"mailto": true,
	"ftp":    true,
}

func isWebpageURL(u *url.URL) error {
	if invalidSchemes[u.Scheme] {
		return fmt.Errorf("invalid scheme: %s", u.Scheme)
	}

	filename, err := ffi.FilenameFromUrlString(u.String())
	if err != nil {
		return fmt.Errorf("ffi err: %s", err.Error())
	}

	ext := filepath.Ext(filename)
	if !htmlExtensions[ext] {
		return fmt.Errorf("non-webpage extension: %s", ext)
	}

	return nil
}

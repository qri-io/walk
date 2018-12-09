package lib

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/datatogether/cdxj"
	"github.com/ugorji/go/codec"
)

// Walk are read-only operations on the result of a walk
type Walk interface {
	// Len returns the number of resources in the Walk
	Len() int
	// ID is an identifier for this Walk
	ID() string
	// SortedIndex returns an abbreviated list of records, assumes
	// the values are sorted by SURT url
	SortedIndex(limit, offset int) ([]*Resource, error)
	// FindIndex returns the porition within the index of a given url string, -1
	// if it doesn't exist
	FindIndex(url string) int
	// Get returns a resource for a given URL
	Get(url string, t time.Time) (*Resource, error)
	// Timespan returns the earliest & latest dates this Walk contains
	Timespan() (start, stop time.Time)
}

// CBORResourceFileReader is an implementation of Walk that reads from CBOR archives,
// implements the Walk interface
type CBORResourceFileReader struct {
	base        string
	index       []*cdxj.Record
	handle      codec.Handle
	start, stop time.Time
}

// NewCBORResourceFileReader creates a reader from a path to a walk, loading the cdxj index
func NewCBORResourceFileReader(path string) (*CBORResourceFileReader, error) {
	r := &CBORResourceFileReader{
		base:   path,
		handle: &codec.CborHandle{},
	}
	if err := r.loadIndex(); err != nil {
		return r, err
	}
	r.calcTimespan()
	return r, nil
}

func (cr *CBORResourceFileReader) loadIndex() (err error) {
	var f *os.File
	if f, err = os.Open(filepath.Join(cr.base, "index.cdxj")); err != nil {
		return
	}
	defer f.Close()

	r := cdxj.NewReader(f)
	cr.index, err = r.ReadAll()
	return err
}

func (cr *CBORResourceFileReader) calcTimespan() {
	cr.start, cr.stop = time.Now(), time.Time{}
	for _, r := range cr.index {
		if r.Timestamp.Before(cr.start) {
			cr.start = r.Timestamp
		}
		if r.Timestamp.After(cr.stop) {
			cr.stop = r.Timestamp
		}
	}
}

// Len returns the number of resources listed in the Walk
func (cr *CBORResourceFileReader) Len() int {
	return len(cr.index)
}

// ID gives an identifier for this Walk. not garunteed to be unique
func (cr *CBORResourceFileReader) ID() string {
	return filepath.Base(cr.base)
}

// Index gives a limit & Offset
func (cr *CBORResourceFileReader) Index(limit, offset int) (rsc []*Resource, err error) {
	for _, rec := range cr.index {
		if offset > 0 {
			offset--
			continue
		}

		rsc = append(rsc, &Resource{
			URL:       rec.URI,
			Timestamp: rec.Timestamp,
			Hash:      rec.JSON["hash"].(string),
		})

		limit--
		if limit == 0 {
			break
		}
	}

	return
}

// FindIndex returns the index position of a given url, -1 if not found
func (cr *CBORResourceFileReader) FindIndex(url string) int {
	surl, err := cdxj.SurtURL(url)
	if err != nil {
		return -1
	}
	url, err = cdxj.UnSurtURL(surl)
	if err != nil {
		return -1
	}

	for i, rec := range cr.index {
		if url == rec.URI {
			return i
		}
	}
	return -1
}

// SortedIndex returns an abbreviated list of records, assumes
// the values are sorted by SURT url
func (cr *CBORResourceFileReader) SortedIndex(limit, offset int) (rsc []*Resource, err error) {
	for _, rec := range cr.index {
		if offset > 0 {
			offset--
			continue
		}

		rsc = append(rsc, &Resource{
			URL:       rec.URI,
			Timestamp: rec.Timestamp,
			Hash:      rec.JSON["hash"].(string),
		})

		limit--
		if limit == 0 {
			break
		}
	}

	return rsc, nil
}

// Get grabs an individual resource from the Walk
func (cr *CBORResourceFileReader) Get(url string, t time.Time) (*Resource, error) {
	idx := cr.FindIndex(url)
	if idx == -1 {
		return nil, fmt.Errorf("not found")
	}

	md := cr.index[idx].JSON
	if md == nil || md["url"] == nil {
		return nil, fmt.Errorf("index is missing url metadata")
	}

	url, ok := md["url"].(string)
	if !ok {
		return nil, fmt.Errorf("expected meta 'url' field to be a string")
	}

	fname := base64.StdEncoding.EncodeToString([]byte(url))
	path := filepath.Join(cr.base, fmt.Sprintf("%s.cbor", fname))

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	rsc := &Resource{}
	err = codec.NewDecoder(f, cr.handle).Decode(rsc)
	return rsc, err
}

// Timespan gives the earliest & latest times this Walk covers
func (cr *CBORResourceFileReader) Timespan() (start, stop time.Time) {
	return cr.start, cr.stop
}

package lib

import (
	"fmt"
	"time"
)

// Collection defines operations on a group of Walks
type Collection interface {
	// Collections must implement the Walk interface, aggregating across all
	// Walks
	Walk
	// Walks gives access to the list of individual Walks
	Walks() ([]Walk, error)
}

// collection implements the exported interface
type collection struct {
	walks []Walk
}

// NewCollectionFromConfig creates a collection from a collection configuration
// currently it only supports creating CBOR walk readers from exact directories
// in the future it's functionality should be expanded
func NewCollectionFromConfig(cfg *CollectionConfig) (Collection, error) {
	var walks []Walk
	for _, path := range cfg.LocalDirs {
		rdr, err := NewCBORResourceFileReader(path)
		if err != nil {
			return nil, err
		}
		walks = append(walks, rdr)
	}
	return NewCollection(walks...), nil
}

// NewCollection creates a new collection from any number of walks
func NewCollection(walks ...Walk) Collection {
	return collection{
		walks: walks,
	}
}

func (c collection) Walks() ([]Walk, error) {
	return c.walks, nil
}

// Len returns the number of resources in the Walk
func (c collection) Len() (len int) {
	for _, w := range c.walks {
		len += w.Len()
	}
	return
}

// ID is an identifier for this Walk
func (c collection) ID() string {
	return "collection"
}

// SortedIndex returns an abbreviated list of records, assumes
// the values are sorted by SURT url
func (c collection) SortedIndex(limit, offset int) ([]*Resource, error) {
	return nil, fmt.Errorf("not finished")
}

// FindIndex returns the porition within the index of a given url string, -1
// if it doesn't exist
func (c collection) FindIndex(url string) int {
	return -1
}

// Get returns a resource for a given URL
func (c collection) Get(url string, t time.Time) (r *Resource, err error) {
	var rsc *Resource
	for _, w := range c.walks {
		rsc, err = w.Get(url, t)
		if err != nil && err != ErrNotFound {
			return
		}
		if r == nil || rsc.Timestamp.After(r.Timestamp) {
			r = rsc
		}
	}

	if r == nil {
		err = ErrNotFound
	}

	return
}

// Timespan returns the earliest & latest dates this Walk contains
func (c collection) Timespan() (start, stop time.Time) {
	return time.Time{}, time.Now()
}

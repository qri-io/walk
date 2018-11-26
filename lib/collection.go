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
func (c collection) Get(url string) (*Resource, error) {
	return nil, fmt.Errorf("not finished")
}

// Timespan returns the earliest & latest dates this Walk contains
func (c collection) Timespan() (start, stop time.Time) {
	return time.Time{}, time.Now()
}

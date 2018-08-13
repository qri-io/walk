package lib

import (
	"sync"
)

// RequestStore is the interface for storing requests by their URL string
type RequestStore interface {
	Put(*Request) error
	Get(URL string) (*Request, error)
	List(limit, offset int) ([]*Request, error)
}

// MemRequestStore is an in-memory implementation of a request store
type MemRequestStore struct {
	// lock protects access to urls domains map
	sync.Mutex
	// crawled is the list of stuff that's been crawled
	reqs map[string]*Request
}

// NewMemRequestStore creates a new
func NewMemRequestStore() *MemRequestStore {
	return &MemRequestStore{
		reqs: map[string]*Request{},
	}
}

// Put a request in the store
func (m *MemRequestStore) Put(r *Request) error {
	m.Lock()
	defer m.Unlock()
	m.reqs[r.URL] = r
	return nil
}

// Get a request from the store by URL string
func (m *MemRequestStore) Get(urlstr string) (*Request, error) {
	m.Lock()
	defer m.Unlock()
	r := m.reqs[urlstr]
	if r == nil {
		return nil, ErrNotFound
	}

	return r, nil
}

// List requests in the store
// TODO - THIS WILL ONLY WORK IF LIST EVERYTHING. MUST FIX
// listing should happen by lexographical URL order
func (m *MemRequestStore) List(limit, offset int) (frc []*Request, err error) {
	m.Lock()
	defer m.Unlock()

	i := 0
	for _, fr := range m.reqs {
		if i < offset {
			continue
		}
		frc = append(frc, fr)
		i++
		if i == limit {
			break
		}
	}

	return frc, nil
}

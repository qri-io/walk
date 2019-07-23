package lib

import (
	"bytes"
	"sync"

	"github.com/dgraph-io/badger"
	"github.com/ugorji/go/codec"
)

// RequestStore is the interface for storing requests by their URL string
type RequestStore interface {
	PutRequest(*Request) error
	GetRequest(URL string) (*Request, error)
	ListRequests(limit, offset int) ([]*Request, error)
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

// PutRequest in the store
func (m *MemRequestStore) PutRequest(r *Request) error {
	m.Lock()
	defer m.Unlock()
	m.reqs[r.URL] = r
	return nil
}

// GetRequest from the store by URL string
func (m *MemRequestStore) GetRequest(urlstr string) (*Request, error) {
	m.Lock()
	defer m.Unlock()
	r := m.reqs[urlstr]
	if r == nil {
		return nil, ErrNotFound
	}

	return r, nil
}

// ListRequests shows requests in the store
// TODO - THIS WILL ONLY WORK IF LIST EVERYTHING. MUST FIX
// listing should happen by lexographical URL order
func (m *MemRequestStore) ListRequests(limit, offset int) (frc []*Request, err error) {
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

// NewBadgerRequestStore creates a RequestStore from a badger.Db connection
func NewBadgerRequestStore(db *badger.DB) BadgerRequestStore {
	return BadgerRequestStore{
		db:     db,
		handle: &codec.CborHandle{},
	}
}

// BadgerRequestStore implements the request store interface in badger
type BadgerRequestStore struct {
	db     *badger.DB
	handle codec.Handle
}

func (rs BadgerRequestStore) prefixBytes() []byte {
	return []byte("rs.")
}

func (rs BadgerRequestStore) key(url string) []byte {
	return append(rs.prefixBytes(), []byte(url)...)
}

// PutRequest in the store
func (rs BadgerRequestStore) PutRequest(r *Request) (err error) {
	buf := &bytes.Buffer{}
	if err = codec.NewEncoder(buf, rs.handle).Encode(r); err != nil {
		return
	}

	// codec.NewEncoder()
	err = rs.db.Update(func(txn *badger.Txn) error {
		if err := txn.Set(rs.key(r.URL), buf.Bytes()); err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		log.Debugf("error adding map entry to badger: %s", err.Error())
	}
	return err
}

// GetRequest from the store by URL string
func (rs BadgerRequestStore) GetRequest(urlstr string) (req *Request, err error) {
	err = rs.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(rs.key(urlstr))
		if err != nil {
			return err
		}

		err = item.Value(func(val []byte) error {
			req = &Request{}
			return codec.NewDecoder(bytes.NewBuffer(val), rs.handle).Decode(req)
		})

		return nil
	})

	return
}

// ListRequests shows requests in the store
// TODO - THIS WILL ONLY WORK IF LIST EVERYTHING. MUST FIX
// listing should happen by lexographical URL order
func (rs BadgerRequestStore) ListRequests(limit, offset int) (frc []*Request, err error) {
	err = rs.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		prefix := rs.prefixBytes()
		defer it.Close()
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			// k := item.Key()
			r := &Request{}

			err := item.Value(func(val []byte) error {
				return codec.NewDecoder(bytes.NewBuffer(val), rs.handle).Decode(r)
			})
			if err != nil {
				return err
			}

			frc = append(frc, r)
		}
		return nil
	})

	return
}

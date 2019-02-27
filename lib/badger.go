package lib

import (
	"github.com/dgraph-io/badger"
)

// db is a single, shared badger connection
var db *badger.DB

// BadgerConfig configures the badger instance for walk
type BadgerConfig struct {
	// TODO - make badger configurable :/
	badger.Options
}

// NewBadgerConfig creates a badger configuration with default options
func NewBadgerConfig() *BadgerConfig {
	cfg := &BadgerConfig{
		Options: badger.DefaultOptions,
	}
	cfg.Dir = "badger"
	cfg.ValueDir = "badger"
	return cfg
}

// DB returns a connection to Badger
func (c *BadgerConfig) DB() (conn *badger.DB, err error) {
	if db == nil {
		// Open the Badger database at the location specified in c.Options
		// It will be created if it doesn't exist.
		if db, err = badger.Open(c.Options); err != nil {
			log.Error(err.Error())
			return nil, err
		}
	}
	return db, nil
}

// BadgerRequestStore records requests in a  badgerDB key/value store
// type BadgerRequestStore struct {
// 	prefix string
// 	db     *badger.DB
// }

// // NewBadgerRequestStore creates a BadgerRequestStore from a given prefix & badger.DB connection
// func NewBadgerRequestStore(prefix string, db *badger.DB) *BadgerRequestStore {
// 	return &BadgerRequestStore{
// 		prefix: prefix,
// 		db:     db,
// 	}
// }

// // Put a request in the Badger store
// func (s *BadgerRequestStore) Put(*Request) error {
// 	value, err := json.Marshal(me)
// 	if err != nil {
// 		log.Debugf("error encoding map entry: %s", err.Error())
// 		return
// 	}

// 	err = g.db.Update(func(txn *badger.Txn) error {
// 		if err := txn.Set(key, value); err != nil {
// 			return err
// 		}
// 		return nil
// 	})

// 	if err != nil {
// 		log.Debug("error adding map entry to badger: %s", err.Error())
// 	}

// 	return
// }
// func (s *BadgerRequestStore) Get(URL string) (*Request, error) {
// 	return
// }
// func (s *BadgerRequestStore) List(limit, offset int) ([]*Request, error) {
// 	sm := Sitemap{}
// 		err := g.db.View(func(txn *badger.Txn) error {
// 			it := txn.NewIterator(badger.DefaultIteratorOptions)
// 			prefix := g.prefixBytes()
// 			defer it.Close()
// 			for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
// 				item := it.Item()
// 				k := item.Key()
// 				v, err := item.Value()
// 				if err != nil {
// 					return err
// 				}
// 				e := &Entry{}
// 				if err := json.Unmarshal(v, e); err != nil {
// 					return err
// 				}
// 				sm[string(k[len(prefix):])] = e
// 				// fmt.Printf("key=%s, value=%s\n", k, v)
// 			}
// 			return nil
// 		})
// 		if err != nil {
// 			return err
// 		}

// 		data, err := json.MarshalIndent(sm, "", "  ")
// 		if err != nil {
// 			return err
// 		}

// 		return ioutil.WriteFile(path, data, os.ModePerm)
// 	}
// 	return
// }

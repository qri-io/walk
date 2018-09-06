package lib

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"time"

	"github.com/dgraph-io/badger"
)

// SitemapGenerator records resource reponses in a badgerDB key/value store
// and can create JSON output of the desired
type SitemapGenerator struct {
	prefix  string
	db      *badger.DB
	dstPath string
}

// NewSitemapGenerator creates a Sitemapgenerator from a given prefix & badger.DB connection
func NewSitemapGenerator(prefix, dstPath string, db *badger.DB) *SitemapGenerator {
	return &SitemapGenerator{
		prefix:  prefix,
		db:      db,
		dstPath: dstPath,
	}
}

// HandleResource implements ResourceHandler to add sitemap
func (g *SitemapGenerator) HandleResource(r *Resource) {
	me := NewEntryFromResource(r)

	key, err := g.key(me)
	if err != nil {
		log.Debugf("error getting resource key: %s", err.Error())
		return
	}

	value, err := json.Marshal(me)
	if err != nil {
		log.Debugf("error encoding map entry: %s", err.Error())
		return
	}

	err = g.db.Update(func(txn *badger.Txn) error {
		if err := txn.Set(key, value); err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		log.Debug("error adding map entry to badger: %s", err.Error())
	}
}

// FinalizeResources writes a json sitemap file to outpath
func (g *SitemapGenerator) FinalizeResources() error {
	if err := g.Generate(g.dstPath); err != nil {
		return err
	}
	return nil
}

// key creates the canonicalized key for a Entry
func (g *SitemapGenerator) key(me *Entry) ([]byte, error) {
	url, err := NormalizeURLString(me.URL)
	if err != nil {
		return nil, err
	}
	return append(g.prefixBytes(), []byte(url)...), nil
}

func (g *SitemapGenerator) prefixBytes() []byte {
	return []byte(g.prefix + ":")
}

// Generate creates a json sitemap file at the specified path
func (g *SitemapGenerator) Generate(path string) error {
	sm := Sitemap{}
	err := g.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		prefix := g.prefixBytes()
		defer it.Close()
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			k := item.Key()
			v, err := item.Value()
			if err != nil {
				return err
			}
			e := &Entry{}
			if err := json.Unmarshal(v, e); err != nil {
				return err
			}
			sm[string(k[len(prefix):])] = e
			// fmt.Printf("key=%s, value=%s\n", k, v)
		}
		return nil
	})
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(sm, "", "  ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(path, data, os.ModePerm)
}

// Sitemap is a list of entries
type Sitemap map[string]*Entry

// Entry is a subset of a resource relevant to a sitemap
type Entry struct {
	URL       string    `json:"url"`
	Title     string    `json:"title"`
	Timestamp time.Time `json:"timestamp"`
	Status    int       `json:"status"`
	Redirects []string  `json:"redirects"`
	Resources []string  `json:"resources"`
	Links     []string  `json:"links"`
}

// NewEntryFromResource pulls releveant values from a resource
// to create a Entry
func NewEntryFromResource(r *Resource) *Entry {
	return &Entry{
		URL:       r.URL,
		Title:     r.Title,
		Timestamp: r.Timestamp,
		Status:    r.Status,
		Links:     r.Links,
	}
}

// if c.cfg.BackupWriteInterval > 0 {
//  path := fmt.Sprintf("%s.backup", c.cfg.DestPath)
//  log.Infof("writing backup sitemap: %s", path)
//  if err := c.WriteJSON(path); err != nil {
//    log.Errorf("error writing backup sitemap: %s", err.Error())
//  }
// }

// LoadSitemapFile loads a sitemap.json file
// func (h *SiteMapHandler) LoadSitemapFile(path string) error {
// 	if filepath.Ext(path) == ".json" {
// 		if f, err := os.Open(path); err == nil {
// 			log.Infof("loading previous sitemap: %s", path)
// 			urls := make(map[string]*URL)
// 			if err := json.NewDecoder(f).Decode(&urls); err != nil {
// 				return nil
// 			}
// 			c.urlLock.Lock()
// 			defer c.urlLock.Unlock()

// 			added := 0
// 			for urlstr, u := range urls {
// 				c.urls[urlstr] = u
// 				added++
// 			}
// 			log.Info("********************")
// 			log.Infof("added: %d prior urls", added)
// 			log.Info("********************")
// 		}
// 	}
// 	return nil
// }

// WriteJSON writes a sitemap.json file
// func (h *SitemapHandler) WriteJSON(path string) error {
// 	if path == "" {
// 		path = c.cfg.DestPath
// 	}

// 	log.Infof("writing json index file to path: %s", path)

// 	c.urlLock.Lock()
// 	defer func() {
// 		log.Infof("done writing json index file: %s", path)
// 		c.urlLock.Unlock()
// 	}()

// 	f, err := os.Create(path)
// 	if err != nil {
// 		return err
// 	}

// 	us := make(map[string]*URL)
// 	i := 0
// 	for key, u := range c.urls {
// 		if !u.Timestamp.IsZero() {
// 			us[key] = u
// 			i++
// 		}
// 	}

// 	enc := json.NewEncoder(f)
// 	enc.SetIndent("", "  ")
// 	return enc.Encode(us)
// }

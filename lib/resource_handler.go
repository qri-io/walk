package lib

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ugorji/go/codec"
)

// ResourceHandler is the interface for doing stuff with a resource,
// usually just after it's been created
type ResourceHandler interface {
	HandleResource(*Resource)
}

// NewResourceHandlers creates a slice of ResourceHandlers from a slice of ResourceHandler configs
func NewResourceHandlers(cfgs []*ResourceHandlerConfig) (rhs []ResourceHandler, err error) {
	for _, c := range cfgs {
		rh, err := NewResourceHandler(c)
		if err != nil {
			return nil, err
		}
		rhs = append(rhs, rh)
	}

	return rhs, nil
}

// NewResourceHandler creates a ResourceHandler from a config
func NewResourceHandler(cfg *ResourceHandlerConfig) (ResourceHandler, error) {
	switch cfg.Type {
	case "CBOR":
		return &CBORResourceFileWriter{BasePath: cfg.SrcPath}, nil
	default:
		return nil, fmt.Errorf("unrecognized resource handler type: %s", cfg.Type)
	}
}

// CBORResourceFileWriter creates [multhash].cbor in a folder specified by BasePath
// TODO - this needs more thought, given that many of our use cases are time-dependant
type CBORResourceFileWriter struct {
	BasePath string
}

// HandleResource implements the ResourceHandler interface
func (rh *CBORResourceFileWriter) HandleResource(rsc *Resource) {
	if rsc.Hash == "" {
		log.Info("skipping resource, can only record resources with a hash field")
		return
	}

	f, err := os.Create(filepath.Join(rh.BasePath, rsc.Hash+".cbor"))
	defer f.Close()
	if err != nil {
		log.Error(err.Error())
		return
	}

	h := &codec.CborHandle{TimeRFC3339: true}
	h.Canonical = true
	enc := codec.NewEncoder(f, h)

	if err := enc.Encode(rsc); err != nil {
		log.Error(err.Error())
	}
}

// TODO @b5 - I'm hoping to implement the sitemap tricks from before as a resource handler
// c.LoadSitemapFile(c.cfg.SrcPath)

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

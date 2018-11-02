package lib

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ugorji/go/codec"
)

// ResourceHandler is the interface for doing stuff with a resource,
// usually just after it's been created
type ResourceHandler interface {
	Type() string
	HandleResource(*Resource)
}

// ResourceFinalizer is an opt-in interface for ResourceHandler
// Finalize is called when a crawl is concluded, giving handlers a chance
// to clean up, write files, etc.
type ResourceFinalizer interface {
	FinalizeResources() error
}

// NewResourceHandlers creates a slice of ResourceHandlers from a config
func NewResourceHandlers(cfg *Config) (rhs []ResourceHandler, err error) {
	for _, c := range cfg.ResourceHandlers {
		rh, err := NewResourceHandler(cfg, c)
		if err != nil {
			return nil, err
		}
		rhs = append(rhs, rh)
	}

	return rhs, nil
}

// NewResourceHandler creates a ResourceHandler from a config
func NewResourceHandler(c *Config, cfg *ResourceHandlerConfig) (ResourceHandler, error) {
	switch strings.ToUpper(cfg.Type) {
	case "CBOR":
		return &CBORResourceFileWriter{BasePath: cfg.DestPath}, nil
	case "SITEMAP":
		db, err := c.BadgerDB()
		if err != nil {
			return nil, err
		}
		return NewSitemapGenerator(cfg.Prefix, cfg.DestPath, db), nil
	default:
		return nil, fmt.Errorf("unrecognized resource handler type: %s", cfg.Type)
	}
}

// CBORResourceFileWriter creates [multhash].cbor in a folder specified by BasePath
// TODO - this needs more thought, given that many of our use cases are time-dependant
type CBORResourceFileWriter struct {
	BasePath string
}

// Type implements ResourceHandler, distinguishing this RH as "CBOR" type
func (rh *CBORResourceFileWriter) Type() string { return "CBOR" }

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

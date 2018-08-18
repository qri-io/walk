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

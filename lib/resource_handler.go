package lib

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/datatogether/cdxj"
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
		return NewCBORResourceFileWriter(cfg.DestPath)
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

// CBORResourceFileWriter creates [multhash].cbor in a folder specified by basePath
// the file writer also writes a .cdxj index of the urls it recorded to basePath/index.cdxj
type CBORResourceFileWriter struct {
	basePath  string
	indexFile *os.File
	handle    *codec.CborHandle
	index     *cdxj.Writer
}

// NewCBORResourceFileWriter writes
func NewCBORResourceFileWriter(dir string) (*CBORResourceFileWriter, error) {
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return nil, err
	}

	f, err := os.Create(filepath.Join(dir, "index.cdxj"))
	if err != nil {
		return nil, err
	}

	h := &codec.CborHandle{TimeRFC3339: true}
	h.Canonical = true

	return &CBORResourceFileWriter{
		basePath:  dir,
		handle:    h,
		indexFile: f,
		index:     cdxj.NewWriter(f),
	}, nil
}

// Type implements ResourceHandler, distinguishing this RH as "CBOR" type
func (rh *CBORResourceFileWriter) Type() string { return "CBOR" }

// HandleResource implements the ResourceHandler interface
func (rh *CBORResourceFileWriter) HandleResource(rsc *Resource) {
	if rsc.URL == "" {
		log.Info("skipping resource, can only record resources with a URL field")
		return
	}

	fname := base64.StdEncoding.EncodeToString([]byte(rsc.URL))

	f, err := os.Create(filepath.Join(rh.basePath, fname+".cbor"))
	defer f.Close()
	if err != nil {
		log.Error(err.Error())
		return
	}

	enc := codec.NewEncoder(f, rh.handle)
	if err := enc.Encode(rsc); err != nil {
		log.Error(err.Error())
	}

	meta := map[string]interface{}{
		"hash": rsc.Hash,
		"size": len(rsc.Body),
		"url":  rsc.URL,
	}

	if rsc.RedirectTo != "" {
		meta["redirectTo"] = rsc.RedirectTo
	}

	rec := cdxj.NewResponseRecord(rsc.URL, rsc.Timestamp, meta)
	if err := rh.index.Write(rec); err != nil {
		log.Error(err.Error())
	}
}

// FinalizeResources writes the index to it's destination writer
func (rh *CBORResourceFileWriter) FinalizeResources() error {
	return rh.index.Close()
}

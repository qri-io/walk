package api

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/datatogether/api/apiutil"
	"github.com/qri-io/walk/lib"
)

// CollectionHandlers defines HTTP handlers for interacting with a collection
type CollectionHandlers struct {
	collection lib.Collection
}

// HandleListWalks lists the walks connected to a collection
func (h *CollectionHandlers) HandleListWalks(w http.ResponseWriter, r *http.Request) {
	var res []string
	walks, err := h.collection.Walks()
	if err != nil {
		apiutil.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	for _, walk := range walks {
		res = append(res, walk.ID())
	}
	w.Header().Set("content-type", "application/json")
	apiutil.WriteResponse(w, res)
}

func (h *CollectionHandlers) getWalk(id string, w http.ResponseWriter, r *http.Request) lib.Walk {
	walks, err := h.collection.Walks()
	if err != nil {
		apiutil.WriteErrResponse(w, http.StatusInternalServerError, err)
		return nil
	}

	for _, walk := range walks {
		if walk.ID() == id {
			return walk
		}
	}

	apiutil.NotFoundHandler(w, r)
	return nil
}

// HandleWalkIndex lists walks contained in the collection
func (h *CollectionHandlers) HandleWalkIndex(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/collection/"):]
	w.Header().Set("content-type", "application/json")

	if walk := h.getWalk(id, w, r); walk != nil {
		rsc, err := walk.SortedIndex(10000000, 0)
		if err != nil {
			apiutil.WriteErrResponse(w, http.StatusInternalServerError, err)
			return
		}
		apiutil.WriteResponse(w, rsc)
	}
}

// HandleRawResourceMeta gives raw meta information for a capture
func (h *CollectionHandlers) HandleRawResourceMeta(w http.ResponseWriter, r *http.Request) {
	t, url, err := pathTimestampURL("/captures/meta/raw/", r.URL.Path)
	if err != nil {
		apiutil.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}
	w.Header().Set("content-type", "application/json")

	rsc, err := h.collection.Get(url, t)
	if err != nil {
		apiutil.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	apiutil.WriteResponse(w, rsc.Meta())
}

// HandleResolvedResourceMeta gives resolved meta information
func (h *CollectionHandlers) HandleResolvedResourceMeta(w http.ResponseWriter, r *http.Request) {
	t, url, err := pathTimestampURL("/captures/meta/resolved/", r.URL.Path)
	if err != nil {
		apiutil.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}
	w.Header().Set("content-type", "application/json")

	rsc, err := h.resolvedResource(t, url)
	if err != nil {
		apiutil.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	apiutil.WriteResponse(w, rsc.Meta())
}

// HandleRawResource returns the raw response for a given URL
func (h *CollectionHandlers) HandleRawResource(w http.ResponseWriter, r *http.Request) {
	t, url, err := pathTimestampURL("/captures/raw/", r.URL.Path)
	if err != nil {
		apiutil.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}

	rsc, err := h.collection.Get(url, t)
	if err != nil {
		apiutil.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	w.Write(rsc.Body)
}

// HandleResolvedResource fetches a resource, following any redirects
func (h *CollectionHandlers) HandleResolvedResource(w http.ResponseWriter, r *http.Request) {
	t, url, err := pathTimestampURL("/captures/resolved/", r.URL.Path)
	if err != nil {
		apiutil.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}

	rsc, err := h.resolvedResource(t, url)
	if err != nil {
		apiutil.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	w.Write(rsc.Body)
}

// maximum number of redirects a resource will follow when resolving
// TODO (b5): make this configurable
const maxRedirects = 20

func (h *CollectionHandlers) resolvedResource(t time.Time, url string) (rsc *lib.Resource, err error) {
	redirects := 0
	for {
		if rsc, err = h.collection.Get(url, t); err != nil {
			return
		}
		if rsc.RedirectTo != "" {
			url = rsc.RedirectTo
			redirects++
			if redirects == maxRedirects {
				err = fmt.Errorf("max %d redirects exceeded", maxRedirects)
				return
			}
			continue
		}
		break
	}

	return
}

func pathTimestampURL(prefix, path string) (t time.Time, url string, err error) {
	p := strings.TrimPrefix(path, prefix)
	split := strings.SplitN(p, "/", 2)
	if len(split) != 2 {
		err = fmt.Errorf("invalid {timestamp}/{url} combination")
		return
	}

	switch split[0] {
	case "now":
		t = time.Now()
	case "zero":
		t = time.Time{}
	default:
		if t, err = time.Parse(time.RFC3339, split[0]); err != nil {
			return
		}
	}

	url = split[1]
	return
}

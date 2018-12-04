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

func (h *CollectionHandlers) HandleWalkIndex(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/walks/"):]
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

func (h *CollectionHandlers) HandleListMeta(w http.ResponseWriter, r *http.Request) {
	// id := r.URL.Path[len("/meta/"):]
	// TODO (b5): FINISH
	_, _, err := pathTimestampURL("/captures/meta/", r.URL.Path)
	if err != nil {
		apiutil.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}

	w.Header().Set("content-type", "application/json")

	// if walk := h.getWalk(id, w, r); walk != nil {
	// 	page := apiutil.PageFromRequest(r)
	// 	rscs, err := walk.SortedIndex(page.Limit(), page.Offset())
	// 	if err != nil {
	// 		apiutil.WriteErrResponse(w, http.StatusInternalServerError, err)
	// 		return
	// 	}

	// 	var res []*lib.Resource
	// 	for _, c := range rscs {
	// 		rsc, err := walk.Get(c.URL, time.Time{})
	// 		if err != nil {
	// 			panic(err)
	// 		}
	// 		rsc.Body = nil
	// 		res = append(res, rsc)
	// 	}

	// 	apiutil.WritePageResponse(w, res, r, page)
	// 	return
	// }
}

func (h *CollectionHandlers) HandleResolvedResource(w http.ResponseWriter, r *http.Request) {
	t, url, err := pathTimestampURL("/captures/resolved/", r.URL.Path)
	if err != nil {
		apiutil.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}

	walks, err := h.collection.Walks()
	if err != nil {
		apiutil.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	rsc, err := walks[0].Get(url, t)
	if err != nil {
		apiutil.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	w.Write(rsc.Body)
}

func pathTimestampURL(prefix, path string) (t time.Time, url string, err error) {
	p := strings.TrimPrefix(path, prefix)
	split := strings.SplitN(p, "/", 2)
	if len(split) != 2 {
		err = fmt.Errorf("invalid {timestamp}/{url} combination")
		return
	}
	if t, err = time.Parse(time.RFC3339, split[0]); err != nil {
		return
	}

	url = split[1]
	return
}

package api

import (
	"net/http"

	"github.com/qri-io/walk/lib"
)

// CollectionHandlers defines HTTP handlers for interacting with a collection
type CollectionHandlers lib.Collection

// HandleListWalks lists the walks connected to a collection
func (h *CollectionHandlers) HandleListWalks(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("content-type", "application/json")
	w.Write([]byte("hello"))
}

func (h *CollectionHandlers) HandleWalk(w http.ResponseWriter, r *http.Request) {

}

func (h *CollectionHandlers) HandleResource(w http.ResponseWriter, r *http.Request) {

}

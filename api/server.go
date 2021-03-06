package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/qri-io/walk/lib"
)

// Server wraps a collection, turning it into an API server
type Server struct {
	Coordinator lib.Coordinator
	Collection  lib.Collection
	server      *http.Server
}

// Serve Blocks
func (s *Server) Serve(port string) (err error) {
	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: &bug21955Workaround{handler: NewServerRoutes(s)},
	}
	log.Infof("serving on port %s", s.server.Addr)
	return s.server.ListenAndServe()
}

// HealthCheckHandler is a basic ok response for load balancers & co
// returns the version of qri this node is running, pulled from the lib package
func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"meta":{"code":200,"status":"ok","version":"` + lib.VersionNumber + `"},"data":[]}`))
}

// NotFoundHandler indicates a route wasn't found
func NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	writeNotFound(w)
}

func writeNotFound(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte(`{"meta":{"code":404,"status":"not found","version":"` + lib.VersionNumber + `" },"data":[]}`))
}

// NewServerRoutes returns a Muxer that has all API routes
func NewServerRoutes(s *Server) *http.ServeMux {
	m := http.NewServeMux()

	m.Handle("/", s.middleware(NotFoundHandler))
	m.Handle("/status", s.middleware(HealthCheckHandler))

	ch := CollectionHandlers{collection: s.Collection}
	m.Handle("/collection", s.middleware(ch.HandleListWalks))
	m.Handle("/collection/", s.middleware(ch.HandleWalkIndex))
	m.Handle("/captures", s.middleware(ch.HandleCollectionIndex))
	m.Handle("/captures/", s.middleware(ch.HandleCollectionIndex))
	m.Handle("/captures/meta/raw/", s.middleware(ch.HandleRawResourceMeta))
	m.Handle("/captures/meta/resolved/", s.middleware(ch.HandleResolvedResourceMeta))
	m.Handle("/captures/raw/", s.middleware(ch.HandleRawResource))
	m.Handle("/captures/resolved/", s.middleware(ch.HandleResolvedResource))

	jh := JobHandlers{coord: s.Coordinator}
	m.Handle("/jobs", s.middleware(jh.HandleListJobs))
	m.Handle("/jobs/", s.middleware(jh.HandleJobs))

	return m
}

// Workaround for https://github.com/golang/go/issues/21955 to support escaped URLs in URL path
// by stripping redirected protocols and redirecting one more time
type bug21955Workaround struct {
	handler http.Handler
}

func (bf *bug21955Workaround) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if i := strings.Index(r.URL.Path, "http:/"); i != -1 {
		path := r.URL.Path[:i] + r.URL.Path[i+len("http:/"):]
		http.Redirect(w, r, path, http.StatusMovedPermanently)
		return
	}
	if i := strings.Index(r.URL.Path, "https:/"); i != -1 {
		path := r.URL.Path[:i] + r.URL.Path[i+len("https:/"):]
		http.Redirect(w, r, path, http.StatusMovedPermanently)
		return
	}

	bf.handler.ServeHTTP(w, r)
}

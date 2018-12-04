package api

import (
	"fmt"
	"net/http"

	"github.com/qri-io/walk/lib"
)

// Server wraps an
type Server struct {
	collection lib.Collection
	server     *http.Server
}

// NewServer creates a new server
func NewServer(col lib.Collection) *Server {
	return &Server{
		collection: col,
	}
}

// Serve Blocks
func (s *Server) Serve(port string) (err error) {
	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: NewServerRoutes(s),
	}
	return s.server.ListenAndServe()
}

// HealthCheckHandler is a basic ok response for load balancers & co
// returns the version of qri this node is running, pulled from the lib package
func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{ "meta": { "code": 200, "status": "ok", "version":"` + lib.VersionNumber + `" }, "data": [] }`))
}

// NewServerRoutes returns a Muxer that has all API routes
func NewServerRoutes(s *Server) *http.ServeMux {
	m := http.NewServeMux()

	m.Handle("/", s.middleware(HealthCheckHandler))
	m.Handle("/status", s.middleware(HealthCheckHandler))

	ch := CollectionHandlers{collection: s.collection}
	m.Handle("/walks", s.middleware(ch.HandleListWalks))
	m.Handle("/walks/", s.middleware(ch.HandleWalkIndex))
	m.Handle("/captures/meta/", s.middleware(ch.HandleListMeta))
	m.Handle("/captures/resolved/", s.middleware(ch.HandleResolvedResource))

	return m
}

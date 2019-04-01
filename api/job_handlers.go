package api

import (
	"net/http"

	"github.com/datatogether/api/apiutil"
	"github.com/qri-io/walk/lib"
)

// JobHandlers defines HTTP handlers for interacting with a collection
type JobHandlers struct {
	coord lib.Coordinator
}

func (h *JobHandlers) HandleJobs(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		h.HandleListJobs(w, r)
	case "POST":

	}
}

//
func (h *JobHandlers) HandleCreateJob(w http.ResponseWriter, r *http.Request) {

}

// HandleListJobs lists the jobs connected to a collection
func (h *JobHandlers) HandleListJobs(w http.ResponseWriter, r *http.Request) {

	p := apiutil.PageFromRequest(r)
	res := make([]*Job, p.Size)
	idx := 0
	for i, job := range jobs {
		if i < p.Offset() {
			continue
		}
		res[idx] = job
		idx++
		if idx == p.Size {
			break
		}
	}
	res = res[:idx]

	w.Header().Set("Content-Type", "application/json")
	apiutil.WriteResponse(w, res)
}

func (h *JobHandlers) getJob(id string) lib.Job {
	for _, walk := range h.Jobs {
		if walk.ID() == id {
			return walk
		}
	}

	return nil
}

// HandleJob gets a job in a collection
func (h *JobHandlers) HandleJob(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/jobs/"):]
	w.Header().Set("Content-Type", "application/json")
	page := apiutil.PageFromRequest(r)

	if job := h.getJob(id); job != nil {
		apiutil.WriteResponse(w, job)
		return
	}

	writeNotFound(w)
}

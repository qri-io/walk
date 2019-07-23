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
	jobs, err := h.coord.Jobs()
	if err != nil {
		apiutil.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	p := apiutil.PageFromRequest(r)
	res := make([]*lib.Job, p.Size)
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

// HandleJob gets a job in a collection
func (h *JobHandlers) HandleJob(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/jobs/"):]
	w.Header().Set("Content-Type", "application/json")

	job, err := h.coord.Job(id)
	if err != nil {
		apiutil.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	if err := apiutil.WriteResponse(w, job); err != nil {
		log.Error(err)
	}
}

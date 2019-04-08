package lib

import (
	"context"
	"fmt"
	"time"

	"github.com/dgraph-io/badger"
)

// ErrNoBadgerConfig is the result of attempting to connect to a badgerDB
// without one configured
var ErrNoBadgerConfig = fmt.Errorf("badger is not configured")

// Coordinator can coordinate workers. Workers pull from the output Requests
// channel and post finished resources using the completed method. This is the
// minimum interface a worker should need to turn Requests into Resources
type Coordinator interface {
	// NewJob creates a new job on this coordinator
	NewJob(confg *JobConfig) (*Job, error)
	// Jobs provides a list of jobs this Coordinator owns
	Jobs() ([]*Job, error)
	// Job fetches a single job from the coordinator
	Job(id string) (*Job, error)
	// StartJob Begins job execution
	StartJob(id string) error
	// Queue returns a channel of Requests, which contain urls that need
	// to be fetched & turned into one or more resources
	Queue() Queue
	// Coordinators must store a set of requests they've made
	RequestStore() RequestStore
	// Completed work is submitted to the Job by submitting one or more
	// constructed resources
	CompletedResources(rsc ...*Resource) error
	// Shutdown stopts the coordinator, closing any jobs it owns. this can take
	// some time (possibly minutes) to drain existing job queues & gracefully
	// terminate
	Shutdown() error
}

// NewCoordinator creates a coordinator
func NewCoordinator(configs ...func(*CoordinatorConfig)) (coord Coordinator, err error) {
	// combine configurations with default
	cfg := ApplyCoordinatorConfigs(configs...)

	// create queue, store, workers, and handlers
	// TODO - needs to leverage config
	queue := NewMemQueue()

	var db *badger.DB
	if cfg.Badger != nil {
		if db, err = cfg.Badger.DB(); err != nil {
			return
		}
	}

	// TODO - needs to leverage config
	frs := NewBadgerRequestStore(db)

	ctx, cancel := context.WithCancel(context.Background())

	coord = &coordinator{
		ctx:    ctx,
		cancel: cancel,

		queue:  queue,
		frs:    frs,
		badger: db,

		jobHandlers: map[string][]ResourceHandler{},
		jobWorkers:  map[string][]Worker{},

		processingChanges: make(chan int, 8),

		shutdown: make(chan bool),
	}

	go func(coord *coordinator) {
		for {
			select {
			case add := <-coord.processingChanges:
				coord.processing += add
			case <-coord.ctx.Done():
				return
			}
		}
	}(coord.(*coordinator))

	return
}

// coordinator implements the Coordinator interface
type coordinator struct {
	ctx    context.Context // execution context
	queue  Queue           // queue of resources to fetch
	frs    RequestStore    // store of request history
	badger *badger.DB      // coordinator requires a badger db connection

	jobs        []*Job                       // jobs owned by the coordinator
	jobHandlers map[string][]ResourceHandler // mapping of a job's handlers
	jobWorkers  map[string][]Worker          // mapping of a job's workers

	processing        int      // number of urls currently being processed
	processingChanges chan int // channel to modify processing value

	cancel   func()    // context cancel funce
	shutdown chan bool // channel to trigger coordinator shutdown
	stopping bool      // flag indicating coordinator is stopping
}

// Jobs lists all jobs being coordinated
func (coord *coordinator) Jobs() ([]*Job, error) {
	return coord.jobs, nil
}

// Job gets a coordinated job by ID
func (coord *coordinator) Job(id string) (*Job, error) {
	for _, job := range coord.jobs {
		if job.ID == id {
			return job, nil
		}
	}

	return nil, fmt.Errorf("coord: job not found")
}

// NewJob creates and starts a job
func (coord *coordinator) NewJob(cfg *JobConfig) (*Job, error) {
	job := newJob(cfg, coord)
	coord.jobs = append(coord.jobs, job)

	ws, err := NewWorkers(cfg.Workers)
	if err != nil {
		job.Errored(err)
		return nil, err
	}
	coord.jobWorkers[job.ID] = ws

	rhs, err := NewResourceHandlers(coord.badger, cfg.ResourceHandlers)
	if err != nil {
		job.Errored(err)
		return nil, err
	}

	coord.jobHandlers[job.ID] = rhs

	return job, nil
}

// StartJob begins executing a job
func (coord *coordinator) StartJob(id string) error {
	job, err := coord.Job(id)
	if err != nil {
		return err
	}

	// start workers
	for _, w := range coord.jobWorkers[job.ID] {
		if err := w.Start(coord); err != nil {
			return err
		}
	}

	if job.status == JobStatusNew {
		// setup channel of seed urls
		seeds, err := job.Seeds()
		if err != nil {
			err = fmt.Errorf("reading seeds: %s", err.Error())
			job.Errored(err)
			return err
		}

		// read seeds into the coordinator queue
		go func() {
			for url := range seeds {
				coord.enqueue(&Request{JobID: job.ID, URL: url})
			}
		}()
	}

	// start scanning for completion
	if job.cfg.DoneScanMilli > 0 {
		// TODO (b5): This is checking the _entire_ queue & frs. won't work when multiple
		// jobs are running (will require both to completely finish before "done" is ever triggered)
		doneScanT := time.NewTicker(time.Millisecond * time.Duration(job.cfg.DoneScanMilli))
		log.Debugf("coord: performing done scan checks every %d secs.", job.cfg.DoneScanMilli/1000)
		go func() {
			for range doneScanT.C {
				l, err := coord.queue.Len()
				if err != nil {
					log.Errorf("error getting queue length: %s", err.Error())
					continue
				}
				if l == 0 {
					reqs, err := coord.frs.ListRequests(-1, 0)
					if err != nil {
						log.Errorf("error reading: %s", err.Error())
						continue
					}
					for _, r := range reqs {
						if !(r.Status == RequestStatusDone || r.Status == RequestStatusFailed) {
							continue
						}
					}
					if coord.processing > 0 {
						continue
					}
					log.Info("no urls remain for checking, nothing left in queue, we done")
					// c.stop <- true
					// job.Complete()
					coord.Shutdown()
					return
				}
			}
		}()
	}

	log.Infof("coord: starting job: %s", id)
	if err := job.Start(); err != nil {
		job.Errored(err)
		return err
	}

	return nil
}

// Shutdown halts the coordinator
func (coord *coordinator) Shutdown() error {
	log.Debug("coord: shutting down")
	coord.stopping = true

	// wait if there are still urls in the queue
	// TODO (b5): drain existing queue into badger
	if leng, err := coord.queue.Len(); err == nil && leng > 0 {
		coord.shutdown <- true
	}
	coord.cancel()

	for _, rhs := range coord.jobHandlers {
		for _, rh := range rhs {
			if finalizer, ok := rh.(ResourceFinalizer); ok {
				log.Infof("finalizing: %s", rh.Type())
				finalizer.FinalizeResources()
			}
		}
	}
	return nil
}

// Queue gives access to the underlying queue as a channel of Fetch Requests
func (coord *coordinator) Queue() Queue {
	return coord.queue
}

// RequestStore exposes the coordinator's Fetch Request Store
func (coord *coordinator) RequestStore() RequestStore {
	return coord.frs
}

// CompletedResources sends one or more constructed resources to the coordinator
func (coord *coordinator) CompletedResources(rsc ...*Resource) error {

	// handle any global state changes that may result from completed work
	// TODO - finish
	// go func() {
	// for _, resc := range c.cfg.BackoffResponseCodes {
	// 	if res.StatusCode == resc {
	// 		log.Infof("encountered %d response. backing off", resc)
	// 		c.setCrawlDelay(c.crawlDelay + ((time.Duration(c.cfg.CrawlDelayMilliseconds) * time.Millisecond) / 2))
	// 	}
	// }
	// if c.finished == c.cfg.StopAfterEntries {
	// 	stop <- true
	// }
	// }()

	// handle resources and create a deduplicated map
	// of unique candidate urls from all responses
	links := map[string]string{}
	linkCount := 0
	for _, r := range rsc {
		job, err := coord.Job(r.JobID)
		if err != nil {
			log.Errorf("couldn't find job for completed resource: %s", r.URL)
			continue
		}
		if err := coord.dequeue(job, r); err != nil {
			log.Debugf("coord: error dequing url: %s: %s", r.URL, err.Error())
		}
		if job.cfg.Crawl {
			for _, l := range r.Links {
				linkCount++
				if job.urlStringIsCandidate(l) {
					links[l] = r.JobID
				}
			}
		}
	}

	log.Debugf("coord: completed %d resources with %d/%d links", len(rsc), len(links), linkCount)
	for url, jobID := range links {
		r, err := coord.frs.GetRequest(url)
		if err != nil && err != ErrNotFound {
			log.Debugf("coord: err getting url: %s: %s", url, err.Error())
		}
		if r == nil {
			coord.enqueue(&Request{URL: url, JobID: jobID})
		}
	}

	return nil
}

func (coord *coordinator) enqueue(rs ...*Request) {
	for _, r := range rs {
		if coord.stopping {
			r.Status = RequestStatusFailed
			coord.frs.PutRequest(r)
			continue
		}

		log.Debugf("coord: enqueue: %s", r.URL)
		r.Status = RequestStatusQueued
		coord.frs.PutRequest(r)
		coord.queue.Push(r)
		coord.processingChanges <- 1
	}
}

func (coord *coordinator) dequeue(job *Job, rsc *Resource) error {
	fr, err := coord.frs.GetRequest(rsc.URL)
	if err == ErrNotFound {
		fr = &Request{URL: rsc.URL}
	} else if err != nil {
		log.Debugf("coord: err getting url: %s: %s", rsc.URL, err.Error())
		return err
	}

	fr.PrevResStatus = rsc.Status
	fr.AttemptsMade++

	// job, err := coord.Job(rsc.JobID)
	// if err != nil {
	// 	log.Errorf("finding job for dequed url: %s", err)
	// 	return nil
	// }

	defer func() {
		coord.processingChanges <- -1

		if coord.stopping {
			if leng, err := coord.queue.Len(); leng == 0 && err == nil {
				// read off shutdown channel
				<-coord.shutdown
			}
		}

		if job.cfg.StopURL == fr.URL {
			log.Infof("coord: stop url encountered, stopping")
			// TODO (b5): horrible hack to make sure local tests pass b/c too much parallelism
			// should cleanup & use a channel to wait for handle resources goroutines above
			// to finish
			// TODO - this will now break all sorts of stuff, clean this up
			time.Sleep(time.Millisecond * 100)
			coord.Shutdown()
		}
	}()

	if job.okResponseStatus(fr.PrevResStatus) {
		log.Debugf("coord: dequeue: %s", fr.URL)

		job.finished++
		fr.Status = RequestStatusDone
		// send completed records to each handler
		for _, h := range coord.jobHandlers[job.ID] {
			go h.HandleResource(rsc)
		}
		return nil
	}

	if fr.AttemptsMade <= job.cfg.MaxAttempts {
		coord.enqueue(fr)
		return nil
	}

	fr.Status = RequestStatusFailed
	return coord.frs.PutRequest(fr)
}

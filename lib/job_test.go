package lib

import (
	"testing"
)

func TestBasicJob(t *testing.T) {
	tc := NewHTTPDirTestCase(t, "testdata/qri_io")
	s := tc.Server()

	coord := MustCoordinator(t, tc.Coordinator)

	job, err := coord.NewJob(tc.JobConfig(s))
	if err != nil {
		t.Fatal(err.Error())
	}

	coord.StartJob(job.ID)
}

// test that a given URL won’t get queued more than once in the same crawl
// func TestJobNoRequeue(t *testing.T) {
// 	reqs := map[string]int{}
// 	tc := NewHTTPDirTestCase(t, "testdata/self_linking")
// 	s := tc.Server()
// 	cfg := ApplyConfigs(tc.Config(s))

// 	jobID := newJobID()

// 	queue := NewMemQueue()
// 	queue.OnPush = func(r *Request) {
// 		reqs[r.URL]++
// 		if reqs[r.URL] > 1 {
// 			t.Errorf("multiple requests pushed to queue for URL: %s", r.URL)
// 		}
// 	}

// 	frs := NewMemRequestStore()
// 	ws, err := NewWorkers(cfg.Workers)
// 	if err != nil {
// 		t.Fatal(err.Error())
// 	}
// 	coord := NewJob(jobID, cfg.Job, queue, frs, nil)
// 	stop := make(chan bool)

// 	// start workers
// 	for _, w := range ws {
// 		w.Start(coord)
// 	}

// 	coord.Start(stop)
// 	t.Log(coord.urlsWritten)
// }

// func TestJobNoCrawl(t *testing.T) {
// 	reqs := map[string]int{}
// 	tc := NewHTTPDirTestCase(t, "testdata/self_linking")
// 	s := tc.Server()
// 	cfg := ApplyConfigs(tc.Config(s), func(c *Config) {
// 		c.Job.Crawl = false
// 	})

// 	jobID := newJobID()

// 	queue := NewMemQueue()
// 	queue.OnPush = func(r *Request) {
// 		reqs[r.URL]++
// 		if reqs[r.URL] > 1 {
// 			t.Errorf("multiple requests pushed to queue for URL: %s", r.URL)
// 		}
// 	}

// 	frs := NewMemRequestStore()
// 	ws, err := NewWorkers(cfg.Workers)
// 	if err != nil {
// 		t.Fatal(err.Error())
// 	}
// 	coord := NewJob(jobID, cfg.Job, queue, frs, nil)
// 	stop := make(chan bool)

// 	// start workers
// 	for _, w := range ws {
// 		w.Start(coord)
// 	}

// 	coord.Start(stop)
// 	t.Log(coord.urlsWritten)
// }

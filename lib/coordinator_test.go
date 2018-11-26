package lib

import (
	"testing"
)

func TestNewWalkJob(t *testing.T) {
	tc := NewHTTPDirTestCase(t, "testdata/qri_io")
	s := tc.Server()

	walk, stop, err := NewWalkJob(tc.Config(s))
	if err != nil {
		t.Fatal(err.Error())
	}

	walk.Start(stop)
}

// test that a given URL won’t get queued more than once in the same crawl
func TestCoordinatorNoRequeue(t *testing.T) {
	reqs := map[string]int{}
	tc := NewHTTPDirTestCase(t, "testdata/self_linking")
	s := tc.Server()
	cfg := ApplyConfigs(tc.Config(s))

	queue := NewMemQueue()
	queue.OnPush = func(r *Request) {
		reqs[r.URL]++
		if reqs[r.URL] > 1 {
			t.Errorf("multiple requests pushed to queue for URL: %s", r.URL)
		}
	}

	frs := NewMemRequestStore()
	ws, err := NewWorkers(cfg.Workers)
	if err != nil {
		t.Fatal(err.Error())
	}
	coord := NewCoordinator(cfg.Coordinator, queue, frs, nil)
	stop := make(chan bool)

	// start workers
	for _, w := range ws {
		w.Start(coord)
	}

	coord.Start(stop)
	t.Log(coord.urlsWritten)
}

func TestCoordinatorNoCrawl(t *testing.T) {
	reqs := map[string]int{}
	tc := NewHTTPDirTestCase(t, "testdata/self_linking")
	s := tc.Server()
	cfg := ApplyConfigs(tc.Config(s), func(c *Config) {
		c.Coordinator.Crawl = false
	})

	queue := NewMemQueue()
	queue.OnPush = func(r *Request) {
		reqs[r.URL]++
		if reqs[r.URL] > 1 {
			t.Errorf("multiple requests pushed to queue for URL: %s", r.URL)
		}
	}

	frs := NewMemRequestStore()
	ws, err := NewWorkers(cfg.Workers)
	if err != nil {
		t.Fatal(err.Error())
	}
	coord := NewCoordinator(cfg.Coordinator, queue, frs, nil)
	stop := make(chan bool)

	// start workers
	for _, w := range ws {
		w.Start(coord)
	}

	coord.Start(stop)
	t.Log(coord.urlsWritten)
}

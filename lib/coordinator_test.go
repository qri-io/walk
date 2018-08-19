package lib

import (
	"testing"
)

func TestCoordinatorRequeue(t *testing.T) {
	reqs := map[string]int{}
	tc := NewTestCase(t, "testdata/self_linking")
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
}

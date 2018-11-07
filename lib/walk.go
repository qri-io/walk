package lib

// NewWalk creates a new site walk from a given set of configurations
// if no configuration is provided, the default is used
// start the walk by calling Start on the returned coordinator
// halt the walk by sending a value on the returned stop channel
func NewWalk(configs ...func(*Config)) (coord *Coordinator, stop chan bool, err error) {
	// combine configurations with default
	cfg := ApplyConfigs(configs...)

	// create queue, store, workers, and handlers
	// TODO - needs to leverage config
	queue := NewMemQueue()
	// TODO - needs to leverage config
	frs := NewMemRequestStore()
	ws, err := NewWorkers(cfg.Workers)
	if err != nil {
		return
	}
	hs, err := NewResourceHandlers(cfg)
	if err != nil {
		return
	}

	// create coodinator
	coord = NewCoordinator(cfg.Coordinator, queue, frs, hs)
	stop = make(chan bool)

	// start workers
	for _, w := range ws {
		w.Start(coord)
	}

	return
}

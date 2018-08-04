package lib

import (
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/sirupsen/logrus"
)

var (
	// logger
	log       = logrus.New()
	cfgPath   string
	sigKilled bool
)

func stopOnSigKill(stop chan bool) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)

	for {
		<-ch
		if sigKilled == true {
			os.Exit(1)
		}
		sigKilled = true

		go func() {
			log.Infof(strings.Repeat("*", 72))
			log.Infof("  received kill signal. stopping & writing file. this'll take a second")
			log.Infof("  press ^C again to exit")
			log.Infof(strings.Repeat("*", 72))
			stop <- true
		}()
	}
}

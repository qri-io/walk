package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/qri-io/walk/lib"
	"github.com/spf13/cobra"
)

var (
	sigKilled bool
)

// StartCmd runs a crawl
var StartCmd = &cobra.Command{
	Use:   "start",
	Short: "generate a sitemap by crawling",
	Run: func(cmd *cobra.Command, args []string) {
		cfgPath, err := cmd.Flags().GetString("config")
		if err != nil {
			fmt.Printf("error getting config: %s", err.Error())
		}

		coord, stop, err := lib.NewWalkJob(lib.JSONConfigFromFilepath(cfgPath))
		if err != nil {
			fmt.Print(err.Error())
			return
		}

		go stopOnSigKill(stop)
		if err := coord.Start(stop); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		// log.Infof("crawl took: %f hours. wrote %d urls", time.Since(crawl.start).Hours(), crawl.urlsWritten)
	},
}

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

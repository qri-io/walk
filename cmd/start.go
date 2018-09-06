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

		coord, stop, err := lib.NewWalk(lib.JSONConfigFromFilepath(cfgPath))
		if err != nil {
			fmt.Print(err.Error())
			return
		}

		go stopOnSigKill(stop)
		if err := coord.Start(stop); err != nil {
			fmt.Printf("crawl failed: %s", err.Error())
		}

		for _, rh := range coord.ResourceHandlers() {
			if finalizer, ok := rh.(lib.ResourceFinalizer); ok {
				if err := finalizer.FinalizeResources(); err != nil {
					fmt.Printf("error finalizing resources: %s", err.Error())
				}
			}
		}

		// log.Infof("crawl took: %f hours. wrote %d urls", time.Since(crawl.start).Hours(), crawl.urlsWritten)
	},
}

func init() {
	StartCmd.Flags().StringP("config", "c", "config.json", "path to configuration json file")
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

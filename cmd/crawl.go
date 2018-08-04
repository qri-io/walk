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

// CrawlCmd runs a crawl
var CrawlCmd = &cobra.Command{
	Use:   "crawl",
	Short: "generate a sitemap by crawling",
	Run: func(cmd *cobra.Command, args []string) {
		cfgPath, err := cmd.Flags().GetString("config")
		if err != nil {
			fmt.Printf("error getting config: %s", err.Error())
		}

		crawl := lib.NewCrawl(lib.JSONConfigFromFilepath(cfgPath))

		stop := make(chan bool)
		go stopOnSigKill(stop)

		if err := crawl.Start(stop); err != nil {
			fmt.Printf("crawl failed: %s", err.Error())
		}

		if err := crawl.WriteJSON(""); err != nil {
			fmt.Printf("error writing file: %s", err.Error())
		}

		// log.Infof("crawl took: %f hours. wrote %d urls", time.Since(crawl.start).Hours(), crawl.urlsWritten)
	},
}

func init() {
	CrawlCmd.Flags().StringP("config", "c", "sitemap.config.json", "path to configuration json file")
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

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
	jobPath   string
)

// StartCmd runs a crawl
var StartCmd = &cobra.Command{
	Use:   "start",
	Short: "generate a sitemap by crawling",
	Run: func(cmd *cobra.Command, args []string) {
		coord, err := getCoordinator(cmd)
		if err != nil {
			fmt.Fprintf(streams.ErrOut, "getting coordinator: %s", err)
			os.Exit(1)
		}

		if jobPath == "" {
			fmt.Fprintf(streams.ErrOut, "getting coordinator: %s", err)
			os.Exit(1)
		}

		cfg, err := lib.JSONJobConfigFromFilepath(jobPath)
		if err != nil {
			fmt.Fprintf(streams.ErrOut, "reading job file: %s", err)
			os.Exit(1)
		}

		job, err := coord.NewJob(cfg)
		if err != nil {
			fmt.Fprintf(streams.ErrOut, "reading job file: %s", err)
			os.Exit(1)
		}

		go stopOnSigKill(coord)
		if err := coord.StartJob(job.ID); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		// log.Infof("crawl took: %f hours. wrote %d urls", time.Since(crawl.start).Hours(), crawl.urlsWritten)
	},
}

func init() {
	StartCmd.Flags().StringVarP(&jobPath, "job", "j", "", "path to a job file")
	cobra.MarkFlagRequired(StartCmd.Flags(), "job")
}

func stopOnSigKill(coord lib.Coordinator) {
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
			if err := coord.Shutdown(); err != nil {
				panic(err)
			}
		}()
	}
}

package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/dgraph-io/badger"
	"github.com/qri-io/walk/lib"
	"github.com/qri-io/walk/lib/sitemap"
	"github.com/spf13/cobra"
)

// SitemapCmd runs a crawl
var SitemapCmd = &cobra.Command{
	Use:   "sitemap",
	Short: "generate a sitemap by crawling",
	Run: func(cmd *cobra.Command, args []string) {
		wd, err := os.Getwd()
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		cfgPath, err := cmd.Flags().GetString("config")
		if err != nil {
			fmt.Printf("error getting config: %s", err.Error())
			return
		}

		// Open the Badger database located in the /tmp/badger directory.
		// It will be created if it doesn't exist.
		opts := badger.DefaultOptions
		opts.Dir = filepath.Join(wd, "badger")
		opts.ValueDir = filepath.Join(wd, "badger")
		db, err := badger.Open(opts)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer db.Close()

		gen := sitemap.NewGenerator("test", db)

		coord, stop, err := lib.NewWalk(lib.JSONConfigFromFilepath(cfgPath))
		if err != nil {
			fmt.Print(err.Error())
			return
		}

		if err := coord.SetHandlers([]lib.ResourceHandler{gen}); err != nil {
			fmt.Println(err)
			return
		}
		path := filepath.Join(wd, "sitemap.json")
		go stopOnSigKill(stop)
		if err := coord.Start(stop); err != nil {
			fmt.Printf("crawl failed: %s", err.Error())
		}

		if err := gen.Generate(path); err != nil {
			fmt.Printf(err.Error())
		}

		// log.Infof("crawl took: %f hours. wrote %d urls", time.Since(crawl.start).Hours(), crawl.urlsWritten)
	},
}

func init() {
	SitemapCmd.Flags().StringP("config", "c", "config.json", "path to configuration json file")
}

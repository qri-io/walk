package cmd

import (
	"fmt"

	"github.com/qri-io/walk/api"
	"github.com/qri-io/walk/lib"
	"github.com/spf13/cobra"
)

// ServerCmd runs a crawl
var ServerCmd = &cobra.Command{
	Use:   "server",
	Short: "start an api server",
	Run: func(cmd *cobra.Command, args []string) {
		cfgPath, err := cmd.Flags().GetString("config")
		if err != nil {
			fmt.Printf("error getting config: %s", err.Error())
		}

		cfg, err := lib.ReadJSONConfigFile(cfgPath)
		if err != nil {
			panic(err)
		}

		collection, err := lib.NewCollectionFromConfig(cfg.Collection)
		if err != nil {
			panic(err)
		}

		s := api.NewServer(collection)
		if err := s.Serve("3000"); err != nil {
			panic(err)
		}
	},
}

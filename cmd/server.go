package cmd

import (
	"github.com/qri-io/walk/api"
	"github.com/spf13/cobra"
)

// ServerCmd runs a crawl
var ServerCmd = &cobra.Command{
	Use:   "server",
	Short: "start an api server",
	Run: func(cmd *cobra.Command, args []string) {
		coord, err := getCoordinator(cmd)
		if err != nil {
			panic(err)
		}

		// TODO (b5): restore collection support for api server
		// collection, err := lib.NewCollectionFromConfig(coord)
		// if err != nil {
		// 	panic(err)
		// }

		s := api.Server{Coordinator: coord}
		if err := s.Serve("3000"); err != nil {
			panic(err)
		}
	},
}

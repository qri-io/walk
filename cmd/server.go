package cmd

import (
	"github.com/spf13/cobra"
)

// ServerCmd runs a crawl
var ServerCmd = &cobra.Command{
	Use:   "server",
	Short: "start an api server",
	Run: func(cmd *cobra.Command, args []string) {
		// cfgPath, err := cmd.Flags().GetString("config")
		// if err != nil {
		// 	fmt.Printf("error getting config: %s", err.Error())
		// }

		// TODO - finish
		// cfg, err := lib.ReadJSONConfigFile(cfgPath)
		// if err != nil {
		// 	panic(err)
		// }

		// api.NewServer()
	},
}

package cmd

import (
	"fmt"
	"os"

	"github.com/qri-io/ioes"
	"github.com/qri-io/walk/lib"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// logger
var (
	log     = logrus.New()
	streams = ioes.NewStdIOStreams()
)

// RootCmd is the walk command
var RootCmd = &cobra.Command{
	Short: "CLI tool for building sitemaps",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if debug, err := cmd.Flags().GetBool("debug"); err == nil && debug {
			lib.SetLogLevel("debug")
		}
	},
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err.Error())
		os.Exit(-1)
	}
}

func init() {
	RootCmd.PersistentFlags().StringP("config", "c", "config.json", "path to configuration json file")
	RootCmd.PersistentFlags().Bool("debug", false, "show debug output")
	RootCmd.AddCommand(
		StartCmd,
		NormalizeURLCmd,
		ConfigCmd,
		ServerCmd,
		JobCmd,
	)
}

func getCoordinator(cmd *cobra.Command) (lib.Coordinator, error) {
	cfgPath, err := cmd.Flags().GetString("config")
	if err != nil {
		fmt.Printf("error getting config: %s", err.Error())
	}

	return lib.NewCoordinator(lib.JSONCoordinatorConfigFromFilepath(cfgPath))
}

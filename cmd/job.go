package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/qri-io/walk/lib"
	"github.com/spf13/cobra"
)

// JobCmd contains subcommands for creating job files
var JobCmd = &cobra.Command{
	Use:   "job",
	Short: "work with walk jobs",
}

// NewJobCmd creates an empty job file with starting defaults
var NewJobCmd = &cobra.Command{
	Use:   "new",
	Short: "create a new blank job file",
	Run: func(cmd *cobra.Command, args []string) {
		data, err := json.MarshalIndent(lib.DefaultJobConfig(), "", "  ")
		if err != nil {
			panic(err)
		}
		fmt.Fprintln(streams.Out, string(data))
	},
}

func init() {
	JobCmd.AddCommand(
		NewJobCmd,
	)
}

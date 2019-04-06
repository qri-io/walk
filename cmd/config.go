package cmd

import (
	"github.com/spf13/cobra"
)

// ConfigCmd prints present configuration info
var ConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "detect & print current config",
	Long: `config helps you figure out what your current configuration looks like,
if no configuration exists, config prints the default configuration`,
	Run: func(cmd *cobra.Command, args []string) {
		// cfgPath, err := cmd.Flags().GetString("config")
		// if err != nil {
		// 	fmt.Printf("error getting config: %s", err.Error())
		// 	os.Exit(1)
		// }

		// cfg := &lib.CoordinatorConfig{}
		// if data, err := ioutil.ReadFile(cfgPath); err != nil {
		// 	cfg = lib.DefaultConfig()
		// } else {
		// 	if err := json.Unmarshal(data, cfg); err != nil {
		// 		cfg = lib.DefaultConfig()
		// 	}
		// }

		// data, err := json.MarshalIndent(cfg, "", "  ")
		// if err != nil {
		// 	fmt.Printf("error marshaling configuration: %s", err.Error())
		// 	os.Exit(1)
		// }

		// fmt.Print(string(data))
		// // log.Infof("crawl took: %f hours. wrote %d urls", time.Since(crawl.start).Hours(), crawl.urlsWritten)
	},
}

func init() {
	ConfigCmd.Flags().StringP("export", "e", "config.json", "path to configuration json file")
}

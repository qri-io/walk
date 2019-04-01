package main

import (
	"flag"
	"os"

	"github.com/qri-io/registry/regserver/handlers"
	"github.com/qri-io/walk/api"
	"github.com/sirupsen/logrus"
)

var (
	// logger
	log                        = logrus.New()
	printHelp                  bool
	port, adminKey, configPath string
)

// strEnvFlags maps configuration values to flags and environment variables
// each variable defaults to "defaultVal", can be set by an environment variable
// specified by the entry keym, or via a command-line flag. command-line flags
// override all enviornment variables
var strEnvFlags = map[string]struct {
	val        *string
	flag       string
	defaultVal string
	usage      string
}{
	"ADMIN_KEY":   {&adminKey, "admin-key", handlers.NewAdminKey(), "key to use for admin access"},
	"WALK_CONFIG": {&configPath, "config", "", "path to walk configuration file"},
	"PORT":        {&port, "port", "3000", "port to listen on"},
}

func init() {
	// configure flag package
	for _, fl := range strEnvFlags {
		flag.StringVar(fl.val, fl.flag, fl.defaultVal, fl.usage)
	}
}

func main() {
	parseFlags()
	if printHelp {
		return
	}

	// cfgPath := "config.json"

	// cfgPath, err := cmd.Flags().GetString("config")
	// if err != nil {
	// 	fmt.Printf("error getting config: %s", err.Error())
	// }

	// cfg, err := lib.ReadJSONConfigFile(cfgPath)
	// if err != nil {
	// 	panic(err)
	// }

	// collection, err := lib.NewCollectionFromConfig(cfg.Collection)
	// if err != nil {
	// 	panic(err)
	// }

	s := api.NewServer(nil)
	if err := s.Serve(port); err != nil {
		panic(err)
	}
}

func parseFlags() {
	flag.Parse()

	// check to see if flags are default and environment is set, overriding if so
	for key, def := range strEnvFlags {
		env := os.Getenv(key)
		if env != "" && *def.val == def.defaultVal {
			*def.val = env
		}
	}
}

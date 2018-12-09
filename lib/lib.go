package lib

import (
	"github.com/sirupsen/logrus"
)

var log = logrus.New()

// SetLogLevel sets the amount of logging output the library produces
func SetLogLevel(level string) {
	lvl, err := logrus.ParseLevel(level)
	if err != nil {
		panic(err)
	}
	log.SetLevel(lvl)
}

// VersionNumber is the current semver of the walk package
var VersionNumber = "0.1.0-dev"

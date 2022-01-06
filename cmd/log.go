package cmd

import (
	"github.com/apex/log"
)

func logLevel(verbosity int) log.Level {
	if verbosity >= 4 {
		return log.DebugLevel
	}

	switch verbosity {
	case 3:
		return log.InfoLevel
	case 2:
		return log.WarnLevel
	case 1:
		return log.ErrorLevel
	}

	return log.FatalLevel
}

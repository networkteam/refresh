package cmd

import (
	"os"
	dbg "runtime/debug"

	"github.com/apex/log"
	"github.com/spf13/cobra"

	"github.com/markbates/refresh/cmd/loghandler"
)

var cfgFile string
var debug bool
var verbosity int

var RootCmd = &cobra.Command{
	Use:   "refresh",
	Short: "Refresh is a command line tool that builds and (re)starts your Go application everytime you save a Go or template file.",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		log.SetLevel(logLevel(verbosity))
		h := loghandler.New(os.Stderr)
		h.Prefix = "⎯⎯⎯⎯⎯⎯ ⚡️refresh ⎯⎯⎯⎯⎯ "
		log.SetHandler(h)

		buildInfo, ok := dbg.ReadBuildInfo()
		if ok {
			log.Debugf("Version %s", buildInfo.Main.Version)
		}
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return Run(cfgFile)
	},
}

func Execute() {
	if err := RootCmd.Execute(); err != nil {
		os.Exit(-1)
	}
}

func init() {
	RootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "use delve to debug the app")
	RootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "path to configuration file")
	RootCmd.PersistentFlags().IntVarP(&verbosity, "verbosity", "v", 3, "verbosity of log output: 0=fatal, 1=error, 2=warn, 3=info, 4=debug")
}

package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/networkteam/refresh/refresh"
)

func init() {
	RootCmd.AddCommand(initCmd)
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "generates a default configuration file for you.",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Do not report errors as wrong usage
		cmd.SilenceUsage = true

		c := refresh.Configuration{
			AppRoot:            ".",
			IgnoredFolders:     []string{"vendor", "log", "logs", "tmp", "node_modules", "bin", "templates"},
			IncludedExtensions: []string{".go"},
			BuildTargetPath:    "",
			BuildPath:          os.TempDir(),
			BuildDelay:         100 * time.Millisecond,
			BinaryName:         "refresh-build",
			CommandFlags:       []string{},
			CommandEnv:         []string{},
			EnableColors:       true,
		}

		if cfgFile == "" {
			cfgFile = "refresh.yml"
		}

		_, err := os.Stat(cfgFile)
		if !os.IsNotExist(err) {
			return fmt.Errorf("config file %q already exists, skipping init", cfgFile)
		}

		return c.Dump(cfgFile)
	},
}

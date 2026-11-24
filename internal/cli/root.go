// Package cli implements the command-line interface
package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/291-Group/LAN-Orangutan/internal/config"
)

var (
	cfgFile string
	cfg     *config.Config
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "orangutan",
	Short: "Scout - Network Discovery Tool",
	Long: `Scout scans your local network and shows the devices on it
in a web dashboard.`,

	// Execute prints errors itself; without this cobra prints them a second
	// time.
	SilenceErrors: true,

	// Flags have parsed by the time this runs, so anything that fails from here
	// is a runtime problem rather than a usage mistake. Silencing usage now
	// keeps a real error visible instead of burying it under the flag listing,
	// while a genuine flag mistake still gets the full help.
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		cmd.SilenceUsage = true
	},
}

// Execute adds all child commands and runs the CLI
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", config.GetDefaultConfigFile(), "config file path")

	// Add subcommands
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(versionCmd)
}

func initConfig() {
	var err error
	cfg, err = config.Load(cfgFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Environment variables override the file, so containers can be configured
	// without mounting one.
	cfg.ApplyEnv()
}

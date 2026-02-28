package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	Execute()
}

var (
	configFile string
)

var rootCmd = &cobra.Command{
	Use:           "seristack",
	Short:         "Seristack - A modern task automation tool",
	SilenceErrors: true,
	SilenceUsage:  true,
	Long: `
Visit https://seristack.getsaas.in/ for more information.

See our work on https://github.com/TechXploreLabs/seristack.

Apache 2.0 License.`,
	Version: "0.1.1",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "config.yaml", "config file")
}

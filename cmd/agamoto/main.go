package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "agamoto",
	Short: "Network reconnaissance tool",
	Long: `A single-binary network reconnaissance tool — scans targets,
fingerprints services, searches the web, and produces AI-summarized
risk reports using any OpenAI-compatible provider.`,
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

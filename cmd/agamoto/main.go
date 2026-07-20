package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "agamoto",
	Short: "Network reconnaissance tool",
	Long: `A single-binary network reconnaissance tool — wraps nmap,
parses scan output, and prints readable reports.`,
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

package main

import (
	"fmt"
	"os"

	"github.com/BobbyNooby/agamoto/internal/nmap"
	"github.com/BobbyNooby/agamoto/internal/report"
	"github.com/spf13/cobra"
)

var (
	scanOutput     string
	scanNoDeep     bool
	scanNoResearch bool
)

var scanCmd = &cobra.Command{
	Use:   "scan <target> [-- <nmap-args>]",
	Short: "Scan a target with nmap",
	Long: `Scan a target using nmap and print a readable table.

Anything after "--" is passed directly to nmap. For example:
  agamoto scan localhost -- -p 22,80 -sV -v
  agamoto scan 10.0.0.1 -o report.txt -- -p 1-65535 -sV`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := args[0]
		nmapArgs := args[1:]

		if scanNoDeep {
			fmt.Fprintln(os.Stderr, "[agamoto] --no-deep-research set (not implemented yet)")
		}
		if scanNoResearch {
			fmt.Fprintln(os.Stderr, "[agamoto] --no-web-search set (not implemented yet)")
		}

		fmt.Fprintf(os.Stderr, "[agamoto] Starting scan of %s\n", target)
		if len(nmapArgs) > 0 {
			fmt.Fprintf(os.Stderr, "[agamoto] Passing args to nmap: %v\n", nmapArgs)
		}

		rawXML, err := nmap.Run(target, nmapArgs)
		if err != nil {
			return fmt.Errorf("scan failed: %w", err)
		}

		fmt.Fprintf(os.Stderr, "[agamoto] nmap finished, parsing XML output...\n")
		nmapRun, err := nmap.ParseXML(rawXML)
		if err != nil {
			return fmt.Errorf("parse failed: %w", err)
		}

		totalPorts := 0
		for _, host := range nmapRun.Hosts {
			totalPorts += len(host.Ports)
		}
		fmt.Fprintf(os.Stderr, "[agamoto] Parsed %d port(s) across %d host(s)\n", totalPorts, len(nmapRun.Hosts))

		fmt.Fprintf(os.Stderr, "[agamoto] Generating report...\n")
		output := report.FormatTable(nmapRun, false)

		if scanOutput != "" {
			fmt.Fprintf(os.Stderr, "[agamoto] Writing results to %s\n", scanOutput)
			return os.WriteFile(scanOutput, []byte(output), 0644)
		}

		fmt.Fprintf(os.Stderr, "[agamoto] Done.\n\n")
		fmt.Print(output)
		return nil
	},
}

func init() {
	scanCmd.Flags().StringVarP(&scanOutput, "output", "o", "", "Write results to file")
	scanCmd.Flags().BoolVar(&scanNoDeep, "no-deep-research", false, "Skip fetching full articles")
	scanCmd.Flags().BoolVar(&scanNoResearch, "no-web-search", false, "Skip web research")
	rootCmd.AddCommand(scanCmd)
}

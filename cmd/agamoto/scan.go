package main

import (
	"fmt"
	"os"

	"github.com/BobbyNooby/agamoto/internal/nmap"
	"github.com/BobbyNooby/agamoto/internal/report"
	"github.com/spf13/cobra"
)

var (
	scanPorts   string
	scanVerbose bool
	scanOutput  string
)

var scanCmd = &cobra.Command{
	Use:   "scan <target>",
	Short: "Scan a target with nmap",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := args[0]

		fmt.Fprintf(os.Stderr, "[agamoto] Starting scan of %s\n", target)
		fmt.Fprintf(os.Stderr, "[agamoto] Ports: %s\n", scanPorts)

		rawXML, err := nmap.Run(target, scanPorts)
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
		output := report.FormatTable(nmapRun, scanVerbose)

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
	scanCmd.Flags().StringVarP(&scanPorts, "ports", "p", "21-23,25,53,80,443,8080", "Port range")
	scanCmd.Flags().BoolVarP(&scanVerbose, "verbose", "v", false, "Include closed/refused ports")
	scanCmd.Flags().StringVarP(&scanOutput, "output", "o", "", "Write results to file")
	rootCmd.AddCommand(scanCmd)
}

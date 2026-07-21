package main

import (
	"fmt"
	"os"
	"time"

	"github.com/BobbyNooby/agamoto/internal/ai"
	"github.com/BobbyNooby/agamoto/internal/config"
	"github.com/BobbyNooby/agamoto/internal/nmap"
	"github.com/BobbyNooby/agamoto/internal/report"
	"github.com/spf13/cobra"
)

var (
	scanOutput     string
	scanNoDeep     bool
	scanNoResearch bool
)

func spinner(stop chan bool) {
	chars := []rune(`-\|`)
	i := 0
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			os.Stderr.WriteString("\r\033[K")
			return
		case <-ticker.C:
			os.Stderr.WriteString(fmt.Sprintf("\r[agamoto] Awaiting AI response... %c", chars[i%len(chars)]))
			i++
		}
	}
}

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

		// Load config for AI
		cfg := config.Merge(config.Merge(config.Defaults(), func() config.Config {
			c, err := config.Load()
			if err != nil {
				return config.Config{}
			}
			return c
		}()), config.FromEnv())

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
		tableOutput := report.FormatTable(nmapRun, false)

		var outputFile *os.File
		if scanOutput != "" {
			outputFile, err = os.OpenFile(scanOutput, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
			if err != nil {
				return fmt.Errorf("open output file: %w", err)
			}
			defer outputFile.Close()
		}

		// Print table to stdout
		fmt.Print(tableOutput)
		if outputFile != nil {
			fmt.Fprint(outputFile, tableOutput)
		}

		// AI analysis
		if cfg.APIKey != "" {
			fmt.Fprintf(os.Stderr, "[agamoto] Verifying API key...\n")
			client := ai.NewClient(cfg.APIBase, cfg.APIKey, cfg.Model, 120*time.Second)
			if err := client.Ping(); err != nil {
				fmt.Fprintf(os.Stderr, "[agamoto] API key validation failed: %v\n", err)
				fmt.Fprintf(os.Stderr, "[agamoto] AI analysis skipped.\n")
				if outputFile != nil {
					fmt.Fprintf(os.Stderr, "[agamoto] Writing results to %s\n", scanOutput)
				}
				fmt.Fprintf(os.Stderr, "[agamoto] Done.\n")
				return nil
			}

			// Build AI prompt
			prompt := fmt.Sprintf(ai.ScanPrompt, string(rawXML))

			// Spinner
			stopSpinner := make(chan bool)
			go spinner(stopSpinner)

			var firstToken bool
			aiResponse, err := client.ChatStream(
				"You are a cybersecurity analyst. Be concise.",
				prompt,
				func(token string) {
					if !firstToken {
						firstToken = true
						close(stopSpinner)
						fmt.Println("\n=== AI Analysis ===")
						if outputFile != nil {
							fmt.Fprintln(outputFile, "\n=== AI Analysis ===")
						}
					}
					fmt.Print(token)
					if outputFile != nil {
						fmt.Fprint(outputFile, token)
					}
				},
			)
			if err != nil {
				if !firstToken {
					close(stopSpinner)
				}
				fmt.Fprintf(os.Stderr, "\n[agamoto] AI analysis error: %v\n", err)
			} else {
				if !firstToken {
					close(stopSpinner)
					fmt.Println("\n=== AI Analysis ===")
					if outputFile != nil {
						fmt.Fprintln(outputFile, "\n=== AI Analysis ===")
					}
					fmt.Println(aiResponse)
					if outputFile != nil {
						fmt.Fprintln(outputFile, aiResponse)
					}
				}
				fmt.Println()
				if outputFile != nil {
					fmt.Fprintln(outputFile)
				}
			}
		} else {
			fmt.Fprintf(os.Stderr, "[agamoto] No API key set. Set with: agamoto config --api-key <key>\n")
		}

		if outputFile != nil {
			fmt.Fprintf(os.Stderr, "[agamoto] Writing results to %s\n", scanOutput)
		}
		fmt.Fprintf(os.Stderr, "[agamoto] Done.\n")
		return nil
	},
}

func init() {
	scanCmd.Flags().StringVarP(&scanOutput, "output", "o", "", "Write results to file")
	scanCmd.Flags().BoolVar(&scanNoDeep, "no-deep-research", false, "Skip fetching full articles")
	scanCmd.Flags().BoolVar(&scanNoResearch, "no-web-search", false, "Skip web research")
	rootCmd.AddCommand(scanCmd)
}

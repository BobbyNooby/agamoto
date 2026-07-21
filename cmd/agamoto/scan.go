package main

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/BobbyNooby/agamoto/internal/ai"
	"github.com/BobbyNooby/agamoto/internal/config"
	"github.com/BobbyNooby/agamoto/internal/nmap"
	"github.com/BobbyNooby/agamoto/internal/render"
	"github.com/BobbyNooby/agamoto/internal/report"
	"github.com/BobbyNooby/agamoto/internal/research"
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

func extractFingerprints(nmapRun *nmap.NmapRun) []research.ServiceFingerprint {
	seen := make(map[string]bool)
	var fps []research.ServiceFingerprint
	for _, host := range nmapRun.Hosts {
		for _, port := range host.Ports {
			if port.State.State != "open" {
				continue
			}
			if port.Service.Product == "" {
				continue
			}
			key := port.Service.Product + "|" + port.Service.Version
			if seen[key] {
				continue
			}
			seen[key] = true
			fps = append(fps, research.ServiceFingerprint{
				Product: port.Service.Product,
				Version: port.Service.Version,
			})
		}
	}
	return fps
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

			fingerprints := extractFingerprints(nmapRun)
			corpus := research.NewCorpus()

			if len(fingerprints) > 0 && !scanNoResearch {
				fmt.Fprintf(os.Stderr, "[agamoto] Researching CVE intelligence...\n")

				nvdClient := research.NewNVDClient(cfg.NVDAPIKey)
				kevLoader := research.NewKEVLoader()

				var cves []research.CVE
				var kevCatalog *research.KEVCatalog
				var kevErr error

				var wg sync.WaitGroup
				wg.Add(2)
				go func() {
					defer wg.Done()
					var err error
					cves, err = nvdClient.BatchQueryServices(fingerprints)
					if err != nil {
						fmt.Fprintf(os.Stderr, "[agamoto] NVD query failed: %v\n", err)
					}
				}()
				go func() {
					defer wg.Done()
					kevCatalog, kevErr = kevLoader.Load()
					if kevErr != nil {
						fmt.Fprintf(os.Stderr, "[agamoto] CISA KEV load failed: %v\n", kevErr)
					}
				}()
				wg.Wait()

				if len(cves) > 0 {
					corpus.AddCVEs(cves)
				}

				if kevCatalog != nil {
					seenCVEs := make(map[string]bool)
					var kevMatches []research.KEVEntry
					for _, cve := range cves {
						seenCVEs[cve.ID] = true
						if entry := kevCatalog.FindByCVE(cve.ID); entry != nil {
							kevMatches = append(kevMatches, *entry)
						}
					}
					// Fallback fuzzy match by product
					for _, fp := range fingerprints {
						for _, entry := range kevCatalog.FindByProduct("", fp.Product) {
							if !seenCVEs[entry.CveID] {
								kevMatches = append(kevMatches, entry)
								seenCVEs[entry.CveID] = true
							}
						}
					}
					corpus.AddKEVs(kevMatches)
				}

				fmt.Fprintf(os.Stderr, "[agamoto] Found %d CVE(s) and %d KEV match(es)\n", len(cves), len(corpus.KEVs))

				if scanNoDeep {
					fmt.Fprintf(os.Stderr, "[agamoto] Running basic web search...\n")
					webCorpus, err := research.BasicResearch(target, fingerprints)
					if err == nil && webCorpus != nil {
						corpus.WebResults = webCorpus.WebResults
					}
				} else {
					passes := cfg.MaxResearchPasses
					if passes <= 0 {
						passes = config.DefaultMaxResearchPasses
					}
					urls := cfg.MaxURLsPerQuery
					if urls <= 0 {
						urls = config.DefaultMaxURLsPerQuery
					}
					fmt.Fprintf(os.Stderr, "[agamoto] Running deep web research (%d passes, %d urls/query)...\n", passes, urls)
					webCorpus, err := research.RunDeepResearch(research.DeepResearchOptions{
						Target:          target,
						Services:        fingerprints,
						MaxPasses:       passes,
						MaxURLsPerQuery: urls,
					})
					if err == nil && webCorpus != nil {
						corpus.WebResults = webCorpus.WebResults
						corpus.Articles = webCorpus.Articles
					}
				}
			} else if scanNoResearch {
				fmt.Fprintf(os.Stderr, "[agamoto] Web research disabled by --no-web-search\n")
			}

			// Build AI prompt
			researchText := corpus.Format()
			prompt := fmt.Sprintf(ai.ScanPrompt, string(rawXML), researchText)

			// Spinner
			stopSpinner := make(chan bool)
			go spinner(stopSpinner)

			var firstToken bool
			md := render.NewStreamFormatter()

			var aiResponse string
			aiResponse, err = client.ChatStream(
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
					// Render for terminal
					rendered := md.Write(token)
					if rendered != "" {
						fmt.Print(rendered)
					}
					// Raw to file
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
				if firstToken {
					rest := md.Flush()
					if rest != "" {
						fmt.Print(rest)
					}
				} else {
					close(stopSpinner)
					fmt.Println("\n=== AI Analysis ===")
					if outputFile != nil {
						fmt.Fprintln(outputFile, "\n=== AI Analysis ===")
					}
					fmt.Print(render.Render(aiResponse))
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

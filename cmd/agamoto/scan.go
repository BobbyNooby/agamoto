package main

import (
	"fmt"
	"os"
	"strings"
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
	scanNoResearch bool
	scanDebug      bool
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

func logf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format, args...)
}

func debugf(format string, args ...interface{}) {
	if scanDebug {
		fmt.Fprintf(os.Stderr, "[debug] "+format, args...)
	}
}

var scanCmd = &cobra.Command{
	Use:     "scan <target> [-- <nmap-args>]",
	Aliases: []string{"s"},
	Short:   "Scan a target with nmap",
	Long: `Scan a target using nmap and print a readable table.

Anything after "--" is passed directly to nmap. For example:
  agamoto scan localhost -- -p 22,80,443 -sV -v
  agamoto scan 10.0.0.1 -o report.txt -- -p 1-65535 -sV`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := args[0]
		nmapArgs := args[1:]

		fileCfg, err := config.Load()
		if err != nil {
			logf("[agamoto] Warning: failed to load config: %v\n", err)
		}
		cfg := config.Merge(fileCfg, config.FromEnv())

		logf("[agamoto] Target: %s\n", target)
		if len(nmapArgs) > 0 {
			logf("[agamoto] nmap flags: %v\n", nmapArgs)
		}

		rawXML, err := nmap.Run(target, nmapArgs)
		if err != nil {
			return fmt.Errorf("scan failed: %w", err)
		}

		if scanDebug {
			xmlPreview := string(rawXML)
			if len(xmlPreview) > 4000 {
				xmlPreview = xmlPreview[:4000] + "..."
			}
			debugf("Raw nmap XML (%d bytes):\n%s\n\n", len(rawXML), xmlPreview)
		}

		logf("[agamoto] nmap scan complete, parsing results...\n")
		nmapRun, err := nmap.ParseXML(rawXML)
		if err != nil {
			return fmt.Errorf("parse failed: %w", err)
		}

		totalPorts := 0
		for _, host := range nmapRun.Hosts {
			totalPorts += len(host.Ports)
		}
		logf("[agamoto] Parsed %d port(s) across %d host(s)\n", totalPorts, len(nmapRun.Hosts))

		logf("[agamoto] Generating table report...\n")
		tableOutput := report.FormatTable(nmapRun)

		var outputFile *os.File
		if scanOutput != "" {
			outputFile, err = os.OpenFile(scanOutput, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
			if err != nil {
				return fmt.Errorf("open output file: %w", err)
			}
			defer outputFile.Close()
		}

		fmt.Print(tableOutput)
		if outputFile != nil {
			fmt.Fprint(outputFile, tableOutput)
		}

		if cfg.APIKey == "" {
			logf("[agamoto] No API key set. Run: agamoto config --api-key <key>\n")
			if outputFile != nil {
				logf("[agamoto] Results written to %s\n", scanOutput)
			}
			logf("[agamoto] Done.\n")
			return nil
		}

		logf("[agamoto] Connecting to %s (model: %s)...\n", cfg.APIBase, cfg.Model)
		client := ai.NewClient(cfg.APIBase, cfg.APIKey, cfg.Model, 120*time.Second)
		client.Debug = scanDebug
		if err := client.Ping(); err != nil {
			logf("[agamoto] API key validation failed: %v\n", err)
			if outputFile != nil {
				logf("[agamoto] Results written to %s\n", scanOutput)
			}
			logf("[agamoto] Done.\n")
			return nil
		}
		logf("[agamoto] API key valid\n")

		if !scanNoResearch {
			client.WebSearchMaxResults = cfg.WebSearchMaxResults
			if cfg.WebSearchMaxResults > 0 {
				logf("[agamoto] Web search enabled (max %d results)\n", cfg.WebSearchMaxResults)
			}
		}

		// CVE intelligence (NVD + CISA KEV)
		fingerprints := research.FingerprintsFromNmapRun(nmapRun)
		var cves []research.CVE
		var kevCatalog *research.KEVCatalog

		if len(fingerprints) > 0 {
			if cfg.NVDAPIKey != "" {
				logf("[agamoto] NVD API key: set (higher rate limits)\n")
			} else {
				logf("[agamoto] NVD API key: not set (default 5 requests/30s rate limit)\n")
			}

			nvdClient := research.NewNVDClient(cfg.NVDAPIKey)
			nvdClient.Logf = logf
			kevLoader := research.NewKEVLoader()

			var wg sync.WaitGroup
			wg.Add(2)

			go func() {
				defer wg.Done()
				var err error
				cves, err = nvdClient.BatchQueryServices(fingerprints)
				if err != nil {
					logf("[agamoto]   NVD batch query failed: %v\n", err)
				}
			}()

			go func() {
				defer wg.Done()
				logf("[agamoto]   Loading CISA KEV catalog from cisa.gov...\n")
				var err error
				kevCatalog, err = kevLoader.Load()
				if err != nil {
					logf("[agamoto]   CISA KEV load failed: %v\n", err)
				} else {
					logf("[agamoto]   CISA KEV catalog loaded (%d entries, version %s)\n", kevCatalog.Count, kevCatalog.CatalogVersion)
				}
			}()

			wg.Wait()
		}

		// Build CVE + KEV text for the prompt
		var researchParts []string
		var kevMatches []research.KEVEntry
		if len(cves) > 0 {
			researchParts = append(researchParts, research.FormatCVEsForPrompt(cves))
		}
		if kevCatalog != nil {
			seenCVEs := make(map[string]bool)
			for _, cve := range cves {
				seenCVEs[cve.ID] = true
				if entry := kevCatalog.FindByCVE(cve.ID); entry != nil {
					logf("[agamoto]   CVE %s is actively exploited (CISA KEV)\n", cve.ID)
					kevMatches = append(kevMatches, *entry)
				}
			}
			for _, fp := range fingerprints {
				for _, entry := range kevCatalog.FindByProduct("", fp.Product) {
					if !seenCVEs[entry.CveID] {
						logf("[agamoto]   CVE %s (%s %s) is actively exploited (CISA KEV)\n", entry.CveID, entry.VendorProject, entry.Product)
						kevMatches = append(kevMatches, entry)
						seenCVEs[entry.CveID] = true
					}
				}
			}
			kevText := research.FormatKEVForPrompt(kevMatches)
			if kevText != "" {
				researchParts = append(researchParts, kevText)
			}
		}

		logf("[agamoto] CVE intelligence: %d CVE(s), %d KEV match(es)\n", len(cves), len(kevMatches))

		researchText := "No CVE or KEV data available."
		if len(researchParts) > 0 {
			researchText = strings.Join(researchParts, "\n")
		}

		if scanDebug {
			debugf("Research context (%d chars):\n%s\n\n", len(researchText), researchText)
		}

		// Build the exact command string for the AI prompt.
		cmdStr := fmt.Sprintf("agamoto scan %s", target)
		if scanOutput != "" {
			cmdStr += fmt.Sprintf(" -o %s", scanOutput)
		}
		if scanNoResearch {
			cmdStr += " --no-web-search"
		}
		if scanDebug {
			cmdStr += " --debug"
		}
		if len(nmapArgs) > 0 {
			cmdStr += fmt.Sprintf(" -- %s", strings.Join(nmapArgs, " "))
		}

		// AI analysis
		prompt := fmt.Sprintf(ai.ScanTask, cmdStr, string(rawXML), researchText)
		if scanDebug {
			debugf("Full prompt (%d chars):\n%s\n\n", len(prompt), prompt)
		}

		stopSpinner := make(chan bool)
		go spinner(stopSpinner)

		var streamStarted bool
		md := render.NewStreamRenderer()
		aiStart := time.Now()

		var aiResponse string
		aiResponse, err = client.ChatStream(
			ai.SystemMessage,
			prompt,
			func(token string) {
				if !streamStarted {
					streamStarted = true
					close(stopSpinner)
					fmt.Println("\n=== AI Analysis ===")
					if outputFile != nil {
						fmt.Fprintln(outputFile, "\n=== AI Analysis ===")
					}
				}
				md.Write(token, os.Stdout, outputFile)
			},
		)
		if err != nil {
			if !streamStarted {
				close(stopSpinner)
			}
			logf("\n[agamoto] AI analysis error: %v\n", err)
		} else {
			aiDuration := time.Since(aiStart)
			debugf("AI response: %d chars in %s\n", len(aiResponse), aiDuration.Round(time.Millisecond))
			if scanDebug {
				preview := aiResponse
				if len(preview) > 3000 {
					preview = preview[:3000] + "..."
				}
				debugf("Raw AI response:\n%s\n\n", preview)
			}
			if streamStarted {
				md.Flush(os.Stdout)
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

		if outputFile != nil {
			logf("[agamoto] Results written to %s\n", scanOutput)
		}
		logf("[agamoto] Done.\n")
		return nil
	},
}

func init() {
	scanCmd.Flags().StringVarP(&scanOutput, "output", "o", "", "Write results to file")
	scanCmd.Flags().BoolVarP(&scanNoResearch, "no-web-search", "n", false, "Disable web search (NVD + CISA KEV still run)")
	scanCmd.Flags().BoolVarP(&scanDebug, "debug", "d", false, "Debug mode: show raw nmap XML, full AI prompt, and response metadata")
	rootCmd.AddCommand(scanCmd)
}

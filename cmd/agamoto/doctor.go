package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/BobbyNooby/agamoto/internal/config"
	"github.com/spf13/cobra"
)

var doctorInstall bool

type checkResult struct {
	Name   string
	Status string
	Fix    string
}

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check dependencies and configuration",
	Long: `Verify that agamoto's runtime dependencies are installed and the
configuration is usable. Use --install to attempt to install missing dependencies.`,
	SilenceErrors: true,
	SilenceUsage:  true,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		results := []checkResult{}

		// nmap
		nmapPath, nmapErr := exec.LookPath("nmap")
		if nmapErr == nil {
			results = append(results, checkResult{Name: "nmap", Status: "ok: " + nmapPath})
		} else {
			results = append(results, checkResult{Name: "nmap", Status: "missing", Fix: installHint("nmap")})
			if doctorInstall {
				fmt.Fprintln(os.Stderr, "[doctor] Installing nmap...")
				if err := installNmap(ctx); err != nil {
					results = append(results, checkResult{Name: "nmap install", Status: "failed: " + err.Error()})
				} else {
					nmapPath, _ = exec.LookPath("nmap")
					results = append(results, checkResult{Name: "nmap", Status: "ok: " + nmapPath})
				}
			}
		}

		// Go (only needed for source installs via go install)
		if goPath, err := exec.LookPath("go"); err == nil {
			results = append(results, checkResult{Name: "go", Status: "ok: " + goPath})
		} else {
			results = append(results, checkResult{Name: "go", Status: "missing", Fix: "install Go from https://go.dev/dl/"})
		}

		// Config
		cfg, err := config.Load()
		if err != nil {
			results = append(results, checkResult{Name: "config", Status: "error: " + err.Error(), Fix: "rm ~/.config/agamoto/config.json"})
		} else {
			results = append(results, checkResult{Name: "config", Status: "ok"})
		}

		// API key (file + env)
		merged := config.Merge(cfg, config.FromEnv())
		if merged.APIKey == "" {
			results = append(results, checkResult{Name: "api key", Status: "missing", Fix: "agamoto config --api-key <key>"})
		} else {
			results = append(results, checkResult{Name: "api key", Status: "ok"})
		}

		// Print results
		fmt.Println()
		for _, r := range results {
			if strings.HasPrefix(r.Status, "ok") {
				fmt.Printf("  \u2714 %s: %s\n", r.Name, strings.TrimPrefix(r.Status, "ok: "))
			} else {
				fmt.Printf("  \u2716 %s: %s\n", r.Name, r.Status)
			}
			if r.Fix != "" {
				fmt.Printf("    \u2192 %s\n", r.Fix)
			}
		}
		fmt.Println()

		for _, r := range results {
			if !strings.HasPrefix(r.Status, "ok") {
				os.Exit(1)
			}
		}
		return nil
	},
}

func installHint(pkg string) string {
	switch runtime.GOOS {
	case "darwin":
		return fmt.Sprintf("brew install %s", pkg)
	case "linux":
		switch {
		case commandExists("apt"):
			return fmt.Sprintf("sudo apt install -y %s", pkg)
		case commandExists("dnf"):
			return fmt.Sprintf("sudo dnf install -y %s", pkg)
		case commandExists("pacman"):
			return fmt.Sprintf("sudo pacman -S --noconfirm %s", pkg)
		}
	}
	return fmt.Sprintf("install %s using your package manager", pkg)
}

func installNmap(ctx context.Context) error {
	switch runtime.GOOS {
	case "darwin":
		return run(ctx, "brew", "install", "nmap")
	case "linux":
		switch {
		case commandExists("apt"):
			return run(ctx, "sudo", "apt", "install", "-y", "nmap")
		case commandExists("dnf"):
			return run(ctx, "sudo", "dnf", "install", "-y", "nmap")
		case commandExists("pacman"):
			return run(ctx, "sudo", "pacman", "-S", "--noconfirm", "nmap")
		}
	}
	return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
}

func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func run(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func init() {
	doctorCmd.Flags().BoolVarP(&doctorInstall, "install", "i", false, "Install missing dependencies")
	rootCmd.AddCommand(doctorCmd)
}

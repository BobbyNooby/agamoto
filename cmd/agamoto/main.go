package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"

	"github.com/spf13/cobra"
)

var version = "dev"

var rootCmd = &cobra.Command{
	Use:     "agamoto",
	Short:   "Network reconnaissance tool",
	Version: version,
	Long: `A single-binary network reconnaissance tool — wraps nmap,
parses scan output, and prints readable reports.`,
}

var defaultHelp func(*cobra.Command, []string)

func init() {
	defaultHelp = rootCmd.HelpFunc()
	rootCmd.SetHelpFunc(treeHelp)
}

func main() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	go func() {
		<-sigCh
		fmt.Fprintln(os.Stderr, "\nInterrupted.")
		os.Exit(130)
	}()

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// treeHelp prints a long tree-style help view for the root command,
// including every subcommand and its flags. Subcommands fall back to the
// default help.
func treeHelp(cmd *cobra.Command, args []string) {
	if cmd != rootCmd {
		defaultHelp(cmd, args)
		return
	}

	fmt.Println(cmd.Long)
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  agamoto [command]")

	if cmd.HasAvailableLocalFlags() {
		fmt.Println()
		fmt.Println("Global Flags:")
		fmt.Print(indentLines(cmd.LocalFlags().FlagUsages(), "  "))
	}

	subs := make([]*cobra.Command, 0)
	for _, c := range cmd.Commands() {
		if c.IsAvailableCommand() && c.Name() != "help" && c.Name() != "completion" {
			subs = append(subs, c)
		}
	}

	if len(subs) == 0 {
		return
	}

	fmt.Println()
	fmt.Println("Commands:")
	for i, c := range subs {
		isLast := i == len(subs)-1
		branch := "├── "
		indent := "│   "
		if isLast {
			branch = "└── "
			indent = "    "
		}

		use := strings.TrimPrefix(c.UseLine(), cmd.Name()+" ")
		if len(c.Aliases) > 0 {
			use = fmt.Sprintf("%s (alias: %s)", use, strings.Join(c.Aliases, ", "))
		}
		fmt.Printf("%s%s\n", branch, use)

		if c.Short != "" {
			fmt.Printf("%s%s\n", indent, c.Short)
		}

		if c.HasAvailableLocalFlags() {
			fmt.Printf("%sFlags:\n", indent)
			fmt.Print(indentLines(c.LocalFlags().FlagUsages(), indent+"  "))
		}

		if !isLast {
			fmt.Println()
		}
	}
}

func indentLines(text, indent string) string {
	if text == "" {
		return ""
	}
	lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
	var out strings.Builder
	for _, line := range lines {
		out.WriteString(indent)
		out.WriteString(line)
		out.WriteByte('\n')
	}
	return out.String()
}

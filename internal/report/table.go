package report

import (
	"fmt"
	"strings"

	"github.com/BobbyNooby/agamoto/internal/nmap"
)

func FormatTable(run *nmap.NmapRun, verbose bool) string {
	var b strings.Builder

	for _, host := range run.Hosts {
		addr := host.Address.Addr
		if addr == "" {
			addr = "(unknown)"
		}

		b.WriteString(fmt.Sprintf("Host: %s (%s)\n", addr, host.Status.State))
		if len(host.OS) > 0 {
			b.WriteString(fmt.Sprintf("OS: %s\n", host.OS[0].Name))
		}

		if len(host.Ports) == 0 {
			b.WriteString("  No open ports found\n")
			continue
		}

		b.WriteString(fmt.Sprintf("%-8s %-6s %-20s %s\n", "PORT", "STATE", "SERVICE", "VERSION"))
		b.WriteString(strings.Repeat("-", 60) + "\n")

		for _, port := range host.Ports {
			if port.State.State == "open" || verbose {
				svc := port.Service.Name
				if port.Service.Product != "" {
					svc = port.Service.Product
				}
				ver := port.Service.Version
				if ver != "" {
					svc += " " + ver
				}
				b.WriteString(fmt.Sprintf("%-8s %-6s %-20s %s\n",
					fmt.Sprintf("%d/%s", port.PortID, port.Protocol),
					port.State.State,
					svc,
					port.Service.Version,
				))
			}
		}
		b.WriteString("\n")
	}

	return b.String()
}

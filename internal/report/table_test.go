package report

import (
	"strings"
	"testing"

	"github.com/BobbyNooby/agamoto/internal/nmap"
)

func TestFormatTable(t *testing.T) {
	run := &nmap.NmapRun{
		Hosts: []nmap.Host{
			{
				Status:  nmap.Status{State: "up"},
				Address: nmap.Address{Addr: "192.168.1.1", AddrType: "ipv4"},
				Ports: []nmap.Port{
					{
						Protocol: "tcp",
						PortID:   22,
						State:    nmap.State{State: "open"},
						Service:  nmap.Service{Name: "ssh", Product: "OpenSSH", Version: "8.0"},
					},
				},
				OS: []nmap.OS{{Name: "Linux"}},
			},
		},
	}

	out := FormatTable(run)

	if !strings.Contains(out, "192.168.1.1") {
		t.Error("expected host IP in output")
	}
	if !strings.Contains(out, "22/tcp") {
		t.Error("expected port 22 in output")
	}
	if !strings.Contains(out, "OpenSSH") {
		t.Error("expected OpenSSH in output")
	}
	// Version should only appear in the VERSION column, not duplicated in SERVICE.
	if strings.Contains(out, "OpenSSH 8.0") {
		t.Error("expected version not duplicated in SERVICE column")
	}
}

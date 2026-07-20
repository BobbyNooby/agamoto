package nmap

import (
	"os"
	"testing"
)

func TestParseXML(t *testing.T) {
	data, err := os.ReadFile("../../testdata/nmap_result.xml")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	run, err := ParseXML(data)
	if err != nil {
		t.Fatalf("ParseXML: %v", err)
	}

	if len(run.Hosts) != 1 {
		t.Fatalf("expected 1 host, got %d", len(run.Hosts))
	}

	host := run.Hosts[0]
	if host.Address.Addr != "45.33.32.156" {
		t.Errorf("expected addr 45.33.32.156, got %s", host.Address.Addr)
	}

	if len(host.Ports) != 3 {
		t.Fatalf("expected 3 ports, got %d", len(host.Ports))
	}

	port := host.Ports[0]
	if port.PortID != 22 {
		t.Errorf("expected port 22, got %d", port.PortID)
	}
	if port.Protocol != "tcp" {
		t.Errorf("expected tcp, got %s", port.Protocol)
	}
	if port.State.State != "open" {
		t.Errorf("expected open, got %s", port.State.State)
	}
	if port.Service.Name != "ssh" {
		t.Errorf("expected ssh, got %s", port.Service.Name)
	}
	if port.Service.Product != "OpenSSH" {
		t.Errorf("expected OpenSSH, got %s", port.Service.Product)
	}
	if port.Service.Version != "6.6.1p1" {
		t.Errorf("expected 6.6.1p1, got %s", port.Service.Version)
	}

	if len(host.OS) == 0 {
		t.Fatal("expected OS match")
	}
	if host.OS[0].Name != "Linux 3.x" {
		t.Errorf("expected Linux 3.x, got %s", host.OS[0].Name)
	}
}

func TestParseXMLInvalid(t *testing.T) {
	_, err := ParseXML([]byte("invalid"))
	if err == nil {
		t.Fatal("expected error for invalid XML")
	}
}

package nmap

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

func Run(target string, extraArgs []string) ([]byte, error) {
	args := []string{"-oX", "-", "--stats-every", "5s"}
	args = append(args, extraArgs...)
	args = append(args, target)

	fmt.Fprintf(os.Stderr, "  → Running nmap %s\n", strings.Join(args, " "))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := exec.Command("nmap", args...)
	cmd.Stdout = &stdout
	cmd.Stderr = io.MultiWriter(os.Stderr, &stderr)

	if err := cmd.Run(); err != nil {
		if len(stderr.Bytes()) > 0 {
			return nil, fmt.Errorf("nmap: %s: %w", strings.TrimSpace(stderr.String()), err)
		}
		return nil, fmt.Errorf("nmap: %w", err)
	}
	return stdout.Bytes(), nil
}

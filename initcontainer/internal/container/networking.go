package container

import (
	"fmt"
	"os"
	"strings"
	"syscall"
)

var (
	dnsServers = []string{
		"1.1.1.1",
		"1.0.0.1",
		"2606:4700:4700::1111",
		"2606:4700:4700::1001",
	}
	defaultHosts = []string{
		"127.0.0.1 localhost.localdomain localhost",
		"::1 ip6-localhost ip6-loopback",
		"fe00::0 ip6-localnet",
		"ff00::0 ip6-mcastprefix",
	}
)

func (c *Container) setupNetworking() error {
	if err := syscall.Sethostname([]byte(c.config.Name)); err != nil {
		return fmt.Errorf("failed to set hostname: %w", err)
	}

	if err := os.WriteFile("/etc/hostname", []byte(c.config.Name+"\n"), 0x0755); err != nil {
		return fmt.Errorf("failed to write /etc/hostname: %w", err)
	}

	resolvEntries := make([]string, len(dnsServers))
	for index, server := range dnsServers {
		resolvEntries[index] = fmt.Sprintf("nameserver %v", server)
	}

	if err := os.WriteFile("/etc/resolv.conf", []byte(strings.Join(resolvEntries, "\n")+"\n"), 0x0755); err != nil {
		return fmt.Errorf("failed to write /etc/resolv.conf: %w", err)
	}

	if err := os.WriteFile("/etc/hosts", []byte(strings.Join(defaultHosts, "\n")+"\n"), 0x0755); err != nil {
		return fmt.Errorf("failed to write /etc/hosts: %w", err)
	}

	return nil
}

package parser

import (
	"fmt"
	"net"
	"regexp"
	"strings"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// InlineInventoryDetector provides utilities for detecting and parsing inline inventory specifications
type InlineInventoryDetector struct {
	logger interface {
		Debug(format string, args ...interface{})
		Info(format string, args ...interface{})
		Warn(format string, args ...interface{})
	}
}

// NewInlineInventoryDetector creates a new inline inventory detector
func NewInlineInventoryDetector(logger interface {
	Debug(format string, args ...interface{})
	Info(format string, args ...interface{})
	Warn(format string, args ...interface{})
}) *InlineInventoryDetector {
	return &InlineInventoryDetector{
		logger: logger,
	}
}

// IsInlineInventory checks if the provided string is an inline inventory specification
// rather than a file path. Returns true if it appears to be hosts, false if it looks like a file path.
func (d *InlineInventoryDetector) IsInlineInventory(input string) bool {
	input = strings.TrimSpace(input)

	// If empty, it's not inline inventory
	if input == "" {
		return false
	}

	// Check if it's clearly a file path
	if d.isFilePath(input) {
		return false
	}

	// Check if it looks like a host specification
	return d.looksLikeHostSpec(input)
}

// isFilePath checks if the string looks like a file path
func (d *InlineInventoryDetector) isFilePath(input string) bool {
	// Contains path separators
	if strings.Contains(input, "/") || strings.Contains(input, "\\") {
		return true
	}

	// Starts with ./ or ../
	if strings.HasPrefix(input, "./") || strings.HasPrefix(input, "../") {
		return true
	}

	// Starts with ~ (home directory)
	if strings.HasPrefix(input, "~") {
		return true
	}

	// Contains file extension patterns (case-insensitive)
	lowerInput := strings.ToLower(input)
	if strings.Contains(lowerInput, ".yml") || strings.Contains(lowerInput, ".yaml") ||
		strings.Contains(lowerInput, ".json") || strings.Contains(lowerInput, ".toml") ||
		strings.Contains(lowerInput, ".ini") || strings.Contains(lowerInput, ".txt") {
		return true
	}

	// Check if it contains spaces (unlikely for single hostname, but could be file path)
	// Only treat as file path if it has obvious file extensions
	return false
}

// looksLikeHostSpec checks if the string looks like a host specification
func (d *InlineInventoryDetector) looksLikeHostSpec(input string) bool {
	// Remove trailing comma (Ansible-style)
	hosts := strings.TrimSuffix(input, ",")

	// Split by comma to handle multiple hosts
	hostList := strings.Split(hosts, ",")

	// Check each host in the list
	for _, host := range hostList {
		host = strings.TrimSpace(host)
		if host == "" {
			continue
		}

		if !d.isValidHostSpecification(host) {
			// If any host doesn't look valid, the whole thing might not be inline inventory
			return false
		}
	}

	// If we got here, it looks like valid host specs
	return len(hostList) > 0
}

// isValidHostSpecification checks if a single host specification looks valid
func (d *InlineInventoryDetector) isValidHostSpecification(hostSpec string) bool {
	// Can be: [user@]host[:port]
	// where host can be IP address or hostname

	// Extract user@ if present
	var hostPart string
	if strings.Contains(hostSpec, "@") {
		parts := strings.SplitN(hostSpec, "@", 2)
		if len(parts) != 2 {
			return false
		}
		// Validate user part
		user := parts[0]
		if !d.isValidUsername(user) {
			return false
		}
		hostPart = parts[1]
	} else {
		hostPart = hostSpec
	}

	// Extract port if present
	if strings.Contains(hostPart, ":") && !strings.Contains(hostPart, "://") {
		parts := strings.SplitN(hostPart, ":", 2)
		if len(parts) != 2 {
			return false
		}
		hostPart = parts[0]
		port := parts[1]
		// Validate port
		if !d.isValidPort(port) {
			return false
		}
	}

	// Check if remaining part is valid IP or hostname
	return d.isValidIPOrHostname(hostPart)
}

// isValidUsername checks if a string is a valid username
func (d *InlineInventoryDetector) isValidUsername(user string) bool {
	if user == "" {
		return false
	}
	// Username pattern: alphanumeric, underscore, hyphen, dot
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9._\-]+$`, user)
	return matched
}

// isValidPort checks if a string is a valid port number
func (d *InlineInventoryDetector) isValidPort(port string) bool {
	if port == "" {
		return false
	}
	// Port must be numeric
	matched, _ := regexp.MatchString(`^\d+$`, port)
	return matched
}

// isValidIPOrHostname checks if a string is a valid IP address or hostname
func (d *InlineInventoryDetector) isValidIPOrHostname(host string) bool {
	if host == "" {
		return false
	}

	// Try to parse as IP address
	if net.ParseIP(host) != nil {
		return true
	}

	// Reject strings that look like invalid IP addresses (all numeric parts separated by dots)
	// e.g., "999.999.999.999", "256.1.1.1"
	if d.looksLikeInvalidIP(host) {
		return false
	}

	// Check if it's a valid hostname
	// Hostname pattern: labels separated by dots, each label alphanumeric with hyphens
	matched, _ := regexp.MatchString(
		`^([a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?\.)*[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?$`,
		host,
	)
	return matched
}

// looksLikeInvalidIP checks if a string looks like an IP address but is invalid
func (d *InlineInventoryDetector) looksLikeInvalidIP(host string) bool {
	// Check if it matches the numeric pattern of an IP but has invalid octets
	parts := strings.Split(host, ".")
	if len(parts) == 4 {
		// Check if all parts are numeric (looks like an IP attempt)
		allNumeric := true
		for _, part := range parts {
			if part == "" {
				allNumeric = false
				break
			}
			for _, ch := range part {
				if ch < '0' || ch > '9' {
					allNumeric = false
					break
				}
			}
			if !allNumeric {
				break
			}
		}
		return allNumeric // If all parts are numeric, it looks like an invalid IP
	}
	return false
}

// ParseInlineInventory converts an inline inventory specification to an Inventory object
// Supports formats like:
// - Single host: "192.168.1.1" or "host.example.com"
// - Multiple hosts: "host1,host2,host3"
// - Ansible-style: "192.168.1.1," or "host1,host2,"
// - With ports: "host1:2222,user@host2:2222"
// - With users: "user@host1,root@host2"
func (d *InlineInventoryDetector) ParseInlineInventory(input string) (*types.Inventory, error) {
	input = strings.TrimSpace(input)

	if input == "" {
		return nil, fmt.Errorf("empty inventory specification")
	}

	// Remove trailing comma (Ansible-style)
	input = strings.TrimSuffix(input, ",")

	// Split by comma to handle multiple hosts
	hostSpecs := strings.Split(input, ",")

	inventory := &types.Inventory{
		Hosts:  make([]types.Host, 0),
		Groups: make(map[string]*types.Group),
	}

	// Create a default "all" group
	allGroup := &types.Group{
		Name:  "all",
		Hosts: make(map[string]*types.Host),
		Vars:  make(map[string]interface{}),
	}

	for idx, hostSpec := range hostSpecs {
		hostSpec = strings.TrimSpace(hostSpec)
		if hostSpec == "" {
			continue
		}

		host := d.parseHostSpecification(hostSpec, idx+1)
		if host == nil {
			return nil, fmt.Errorf("failed to parse host specification: %s", hostSpec)
		}

		inventory.Hosts = append(inventory.Hosts, *host)
		allGroup.Hosts[host.Name] = host
	}

	if len(inventory.Hosts) == 0 {
		return nil, fmt.Errorf("no valid hosts in inventory specification")
	}

	inventory.Groups["all"] = allGroup

	d.logger.Info("Parsed inline inventory: %d hosts", len(inventory.Hosts))
	return inventory, nil
}

// parseHostSpecification parses a single host specification
// Format: [user@]host[:port]
func (d *InlineInventoryDetector) parseHostSpecification(hostSpec string, index int) *types.Host {
	var user, address string
	port := 22

	// Check for user@host format
	if strings.Contains(hostSpec, "@") {
		parts := strings.SplitN(hostSpec, "@", 2)
		user = parts[0]
		hostSpec = parts[1]
	}

	// Check for host:port format
	if strings.Contains(hostSpec, ":") && !strings.Contains(hostSpec, "://") {
		parts := strings.SplitN(hostSpec, ":", 2)
		address = parts[0]
		if portNum, err := d.parsePort(parts[1]); err == nil {
			port = portNum
		} else {
			d.logger.Warn("Invalid port in host specification: %s, using default 22", hostSpec)
			port = 22
		}
	} else {
		address = hostSpec
	}

	// Generate host name
	var name string
	if user != "" {
		name = fmt.Sprintf("%s@%s", user, address)
	} else {
		name = address
	}

	// Set default user if not specified
	if user == "" {
		user = "root"
	}

	host := &types.Host{
		Name:    name,
		Address: address,
		Port:    port,
		User:    user,
		Vars:    make(map[string]interface{}),
	}

	d.logger.Debug("Parsed inline host: name=%s, address=%s, port=%d, user=%s", name, address, port, user)
	return host
}

// parsePort parses a port string to integer
func (d *InlineInventoryDetector) parsePort(portStr string) (int, error) {
	portStr = strings.TrimSpace(portStr)
	var port int
	_, err := fmt.Sscanf(portStr, "%d", &port)
	if err != nil {
		return 0, fmt.Errorf("invalid port: %s", portStr)
	}
	if port < 1 || port > 65535 {
		return 0, fmt.Errorf("port out of range: %d", port)
	}
	return port, nil
}

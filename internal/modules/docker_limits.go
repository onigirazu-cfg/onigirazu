package modules

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// dockerLimits are the cpu and memory limits of a container; zero means not set
type dockerLimits struct {
	nanoCPUs int64
	memory   int64
}

// containerLimits reads cpus and memory from docker_container args
func containerLimits(args map[string]interface{}) (dockerLimits, error) {
	var l dockerLimits
	if v, ok := args["cpus"]; ok && v != nil && fmt.Sprint(v) != "" {
		cpus, err := strconv.ParseFloat(strings.TrimSpace(fmt.Sprint(v)), 64)
		if err != nil || cpus < 0 || cpus > 1e6 {
			return l, fmt.Errorf("cpus must be a number, got %v", v)
		}
		l.nanoCPUs = int64(math.Round(cpus * 1e9))
	}
	if v, ok := args["memory"]; ok && v != nil && fmt.Sprint(v) != "" {
		mem, err := parseDockerBytes(fmt.Sprint(v))
		if err != nil {
			return l, fmt.Errorf("memory: %w", err)
		}
		l.memory = mem
	}
	return l, nil
}

// parseDockerBytes parses sizes like 512m, 1G, 1.5g or plain bytes (binary units)
func parseDockerBytes(s string) (int64, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.TrimSuffix(s, "b")
	mult := 1.0
	if s != "" {
		switch s[len(s)-1] {
		case 'k':
			mult = 1 << 10
		case 'm':
			mult = 1 << 20
		case 'g':
			mult = 1 << 30
		case 't':
			mult = 1 << 40
		}
		if mult != 1 {
			s = s[:len(s)-1]
		}
	}
	n, err := strconv.ParseFloat(s, 64)
	if err != nil || n < 0 || n*mult > math.MaxInt64/2 {
		return 0, fmt.Errorf("invalid size %q", s)
	}
	return int64(n * mult), nil
}

// runArgs are the docker run flags for the limits
func (l dockerLimits) runArgs() []string {
	var out []string
	if l.nanoCPUs > 0 {
		out = append(out, "--cpus", formatCPUs(l.nanoCPUs))
	}
	if l.memory > 0 {
		out = append(out, "--memory", strconv.FormatInt(l.memory, 10))
	}
	return out
}

// updateArgs are the docker update flags for the limits that differ from the container
func (l dockerLimits) updateArgs(c *ContainerState) []string {
	if c == nil {
		return nil
	}
	var out []string
	if l.nanoCPUs > 0 && l.nanoCPUs != c.NanoCPUs {
		out = append(out, "--cpus", formatCPUs(l.nanoCPUs))
	}
	if l.memory > 0 && l.memory != c.Memory {
		out = append(out, "--memory", strconv.FormatInt(l.memory, 10))
		// docker refuses a memory update without swap, so swap keeps being
		// unlimited or gets the default docker gives a new container
		swap := "-1"
		if c.MemorySwap != -1 {
			swap = strconv.FormatInt(2*l.memory, 10)
		}
		out = append(out, "--memory-swap", swap)
	}
	return out
}

func formatCPUs(nano int64) string {
	return strconv.FormatFloat(float64(nano)/1e9, 'f', -1, 64)
}

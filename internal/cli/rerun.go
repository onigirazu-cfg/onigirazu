package cli

import "os"

// rerunLimit holds the failed hosts the dashboard asked to run again
var rerunLimit string

// RerunArgs are the arguments of the same apply limited to the failed hosts
// the dashboard asked to run again (X), or nil; the last --limit wins
func RerunArgs() []string {
	if rerunLimit == "" {
		return nil
	}
	return append(append([]string{}, os.Args[1:]...), "--limit", rerunLimit)
}

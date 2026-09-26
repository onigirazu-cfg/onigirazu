package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/onigirazu-cfg/onigirazu/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		var exit *cli.ExitError
		if errors.As(err, &exit) {
			if exit.Message != "" {
				fmt.Fprintln(os.Stderr, exit.Message)
			}
			os.Exit(exit.Code)
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

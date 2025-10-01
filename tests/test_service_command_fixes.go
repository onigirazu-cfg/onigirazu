package tests

import (
	"context"
	"fmt"
	"log"

	"github.com/onigirazu-cfg/onigirazu/internal/modules"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func test_service_command_fixesMain() {
	fmt.Println("Testing Service and Command Module Fixes...")

	// Test host configuration (localhost for testing)
	host := types.Host{
		Name:    "localhost",
		Address: "localhost",
		Port:    22,
		User:    "testuser",
	}

	ctx := context.Background()

	// Test Command Module Fixed
	fmt.Println("\n=== Testing Command Module Fixed ===")
	cmdModule := modules.NewCommandModuleFixed()

	cmdArgs := map[string]interface{}{
		"name":    "test_command",
		"command": "echo 'Hello from fixed command module'",
		"shell":   true,
	}

	result, err := cmdModule.Execute(ctx, host, cmdArgs)
	if err != nil {
		log.Printf("Command module error: %v", err)
	} else {
		fmt.Printf("Command Result: Success=%t, Changed=%t, Output=%v\n",
			result.Success, result.Changed, result.Output)
	}

	// Test Service Module Fixed
	fmt.Println("\n=== Testing Service Module Fixed ===")
	serviceModule := modules.NewServiceModuleFixed()

	serviceArgs := map[string]interface{}{
		"name":  "ssh",
		"state": "started",
	}

	result, err = serviceModule.Execute(ctx, host, serviceArgs)
	if err != nil {
		log.Printf("Service module error: %v", err)
	} else {
		fmt.Printf("Service Result: Success=%t, Changed=%t, Output=%v\n",
			result.Success, result.Changed, result.Output)
	}

	fmt.Println("\nTesting completed!")
}

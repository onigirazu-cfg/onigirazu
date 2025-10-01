package tests

import (
	"context"
	"fmt"
	"log"

	"github.com/onigirazu-cfg/onigirazu/internal/modules"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func test_package_fixMain() {
	fmt.Println("Testing Package Module Fixes...")

	// Create test host
	host := types.Host{
		Name:    "cs_server",
		Address: "cs.rastiegaiev.com",
		Port:    22,
		User:    "usx",
		KeyFile: "/Users/denys.rastiegaiev/.ssh/id_rsa",
	}

	// Create package module
	packageModule := modules.NewPackageModule()

	// Test package installation
	args := map[string]interface{}{
		"name":  "tree",
		"state": "present",
	}

	fmt.Printf("Testing package installation with args: %+v\n", args)

	result, err := packageModule.Execute(context.Background(), host, args)
	if err != nil {
		log.Printf("Error executing package module: %v", err)
	} else {
		fmt.Printf("Result: Success=%v, Changed=%v, Output=%+v\n",
			result.Success, result.Changed, result.Output)
	}

	// Test package removal
	args["state"] = "absent"
	args["name"] = "nano"

	fmt.Printf("Testing package removal with args: %+v\n", args)

	result, err = packageModule.Execute(context.Background(), host, args)
	if err != nil {
		log.Printf("Error executing package module: %v", err)
	} else {
		fmt.Printf("Result: Success=%v, Changed=%v, Output=%+v\n",
			result.Success, result.Changed, result.Output)
	}
}

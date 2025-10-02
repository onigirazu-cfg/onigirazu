package tests

import (
	"fmt"

	"gopkg.in/yaml.v2"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func test_nested_syntaxMain() {
	yamlContent := `
name: "Test task"
module:
  type: "command"
  command: "uname -a"
`

	// First, let's see what YAML parses this as
	var rawMap map[string]interface{}
	err := yaml.Unmarshal([]byte(yamlContent), &rawMap)
	if err != nil {
		fmt.Printf("Raw YAML Error: %v\n", err)
		return
	}

	fmt.Printf("Raw YAML: %+v\n", rawMap)
	fmt.Printf("Module field type: %T\n", rawMap["module"])
	fmt.Printf("Module field value: %+v\n", rawMap["module"])

	var task types.Task
	err = yaml.Unmarshal([]byte(yamlContent), &task)
	if err != nil {
		fmt.Printf("Task Error: %v\n", err)
		return
	}

	fmt.Printf("Task Name: %s\n", task.Name)
	fmt.Printf("Task Module: %s\n", task.Module)
	fmt.Printf("Task Args: %+v\n", task.Args)
}

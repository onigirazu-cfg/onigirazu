package tests

import (
	"fmt"

	"gopkg.in/yaml.v2"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func debug_parse_v2Main() {
	yamlContent := `---
plays:
  - name: "Debug Test"
    hosts: all
    tasks:
      - name: "Test task"
        module:
          type: "command"
          cmd: "echo"
          args: ["hello"]`

	var playbook types.Playbook
	err := yaml.Unmarshal([]byte(yamlContent), &playbook)
	if err != nil {
		fmt.Printf("YAML parse error: %v\n", err)
		return
	}

	fmt.Printf("Playbook parsed successfully\n")
	fmt.Printf("Plays: %d\n", len(playbook.Plays))
	if len(playbook.Plays) > 0 {
		play := playbook.Plays[0]
		fmt.Printf("Play name: %s\n", play.Name)
		fmt.Printf("Tasks: %d\n", len(play.Tasks))
		if len(play.Tasks) > 0 {
			task := play.Tasks[0]
			fmt.Printf("Task name: %s\n", task.Name)
			fmt.Printf("Task module: '%s'\n", task.Module)
			fmt.Printf("Task args: %+v\n", task.Args)
		}
	}
}

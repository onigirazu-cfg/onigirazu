package tests

import (
	"context"
	"fmt"
	"os"

	"github.com/onigirazu-cfg/onigirazu/internal/logger"
	"github.com/onigirazu-cfg/onigirazu/internal/parser"
	"github.com/onigirazu-cfg/onigirazu/internal/template"
)

func debugEnhancedParser() {
	// Create logger and template engine
	log := logger.NewEnhanced("debug", logger.FormatText, os.Stdout)
	templateEngine := template.NewEngine()

	// Create enhanced parser
	enhancedParser := parser.NewEnhancedParser(templateEngine, log)

	// Parse our test playbook
	ctx := context.Background()
	playbook, err := enhancedParser.ParsePlaybook(ctx, "/Users/denys.rastiegaiev/work/go_teransible/examples/debug-package.yml")
	if err != nil {
		fmt.Printf("Enhanced parser error: %v\n", err)
		return
	}

	fmt.Printf("Enhanced parser success!\n")
	fmt.Printf("Playbook: %s\n", playbook.Name)
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

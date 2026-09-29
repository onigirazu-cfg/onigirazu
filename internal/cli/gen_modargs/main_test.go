package main

import (
	"os"
	"testing"
)

// lint_module_args.go must match the modules: run go generate ./internal/cli
func TestGeneratedTableIsCurrent(t *testing.T) {
	table, err := collect("../../modules")
	if err != nil {
		t.Fatal(err)
	}
	want := render(table)
	got, err := os.ReadFile("../lint_module_args.go")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Error("internal/cli/lint_module_args.go is out of date: run go generate ./internal/cli")
	}
}

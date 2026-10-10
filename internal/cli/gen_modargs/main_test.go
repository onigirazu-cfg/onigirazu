package main

import (
	"os"
	"testing"
)

// module_args.go must match the modules: run go generate ./internal/modules
func TestGeneratedTableIsCurrent(t *testing.T) {
	table, err := collect("../../modules")
	if err != nil {
		t.Fatal(err)
	}
	want := render(table, "modules")
	got, err := os.ReadFile("../../modules/module_args.go")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Error("internal/modules/module_args.go is out of date: run go generate ./internal/modules")
	}
}

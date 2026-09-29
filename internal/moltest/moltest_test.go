package moltest

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestExpandEnv(t *testing.T) {
	env := map[string]string{"SET": "v", "EMPTY": ""}
	lookup := func(k string) (string, bool) { v, ok := env[k]; return v, ok }
	for in, want := range map[string]string{
		"${SET}":            "v",
		"$SET/x":            "v/x",
		"${UNSET:-def}":     "def",
		"${EMPTY:-def}":     "def",
		"${EMPTY-def}":      "",
		"${UNSET-def}":      "def",
		"${SET:-def}":       "v",
		"$$SET":             "$SET",
		"img:${UNSET:-a:b}": "img:a:b",
	} {
		if got := expandEnv(in, lookup); got != want {
			t.Errorf("expandEnv(%q) = %q, want %q", in, got, want)
		}
	}
}

const moleculeYML = `
dependency:
  name: galaxy
  options:
    requirements-file: ../../requirements.yml
driver:
  name: docker
platforms:
  - name: instance
    image: ${MT_IMAGE:-example/ubuntu:24.04}
    privileged: true
    cgroupns_mode: host
    override_command: false
    volumes: [/sys/fs/cgroup:/sys/fs/cgroup:rw]
    groups: [web]
  - name: second
    image: example/debian:12
    env: {B: "2", A: "1"}
provisioner:
  name: ansible
  env:
    ANSIBLE_ROLES_PATH: ../../..
    OTHER: plain
  inventory:
    group_vars:
      all: {x: 1}
      web: {y: 2}
    host_vars:
      instance: {z: 3}
`

func scenario(t *testing.T) *Scenario {
	t.Helper()
	project := filepath.Join(t.TempDir(), "myrole")
	dir := filepath.Join(project, "molecule", "default")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{"molecule.yml": moleculeYML, "converge.yml": "- hosts: all\n"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	s, err := Load(project, "default")
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestLoadAndRunArgs(t *testing.T) {
	s := scenario(t)
	if s.Platforms[0].Image != "example/ubuntu:24.04" {
		t.Errorf("image = %q", s.Platforms[0].Image)
	}
	if got := Scenarios(s.ProjectDir); !reflect.DeepEqual(got, []string{"default"}) {
		t.Errorf("Scenarios = %v", got)
	}
	r := &Runner{S: s}
	if c := r.Container(s.Platforms[0]); c != "molecule-myrole-default-instance" {
		t.Errorf("Container = %q", c)
	}
	args := strings.Join(r.runArgs(s.Platforms[0]), " ")
	for _, want := range []string{"--privileged", "--cgroupns=host", "-v /sys/fs/cgroup:/sys/fs/cgroup:rw", "--hostname instance"} {
		if !strings.Contains(args, want) {
			t.Errorf("run args %q lack %q", args, want)
		}
	}
	if !strings.HasSuffix(args, "example/ubuntu:24.04") {
		t.Errorf("override_command false keeps the image's command: %q", args)
	}
	second := strings.Join(r.runArgs(s.Platforms[1]), " ")
	if !strings.Contains(second, "-e A=1 -e B=2 example/debian:12 sh -c while true") {
		t.Errorf("second run args %q", second)
	}
	if s.playbook("converge") == "" || s.playbook("verify") != "" {
		t.Errorf("playbooks: converge %q verify %q", s.playbook("converge"), s.playbook("verify"))
	}
}

func TestInventoryAndEnv(t *testing.T) {
	s := scenario(t)
	r := &Runner{S: s}
	all := r.Inventory()["all"].(map[string]interface{})
	inst := all["hosts"].(map[string]interface{})["instance"].(map[string]interface{})
	if inst["ansible_connection"] != "docker" || inst["ansible_host"] != "molecule-myrole-default-instance" || inst["z"] != 3 {
		t.Errorf("instance vars = %v", inst)
	}
	web := all["children"].(map[string]interface{})["web"].(map[string]interface{})
	if _, ok := web["hosts"].(map[string]interface{})["instance"]; !ok || web["vars"].(map[string]interface{})["y"] != 2 {
		t.Errorf("web group = %v", web)
	}
	if all["vars"].(map[string]interface{})["x"] != 1 {
		t.Errorf("all vars = %v", all["vars"])
	}

	t.Setenv("ANSIBLE_ROLES_PATH", "")
	env := map[string]string{}
	for _, kv := range r.env() {
		k, v, _ := strings.Cut(kv, "=")
		env[k] = v
	}
	roles := filepath.SplitList(env["ANSIBLE_ROLES_PATH"])
	if len(roles) != 3 || roles[0] != filepath.Dir(s.ProjectDir) || roles[2] != filepath.Dir(s.ProjectDir) {
		t.Errorf("ANSIBLE_ROLES_PATH = %v", roles)
	}
	if env["OTHER"] != "plain" || env["MOLECULE_SCENARIO_NAME"] != "default" || env["ONIGIRAZU_MANAGED_STATE_DIR"] != r.runDir() {
		t.Errorf("env = %v", env)
	}
}

func TestChangedTasks(t *testing.T) {
	out := []byte(`{"tasks":[{"name":"a","changed":0},{"name":"b","changed":2,"host_results":{"h2":{"status":"changed"},"h1":{"status":"changed"},"h3":{"status":"success"}}}]}`)
	got, err := changedTasks(out)
	if err != nil || !reflect.DeepEqual(got, []string{"b (h1, h2)"}) {
		t.Errorf("changedTasks = %v, %v", got, err)
	}
	if _, err := changedTasks([]byte("not json")); err == nil {
		t.Error("bad JSON is an error")
	}
}

func TestLoadRejects(t *testing.T) {
	s := scenario(t)
	for _, body := range []string{
		"driver: {name: vagrant}\nplatforms: [{name: a, image: b}]\n",
		"platforms: []\n",
		"platforms: [{name: a}]\n",
		"platforms: [{name: a, image: b, pre_build_image: false}]\n",
	} {
		if err := os.WriteFile(filepath.Join(s.Dir, "molecule.yml"), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(s.ProjectDir, "default"); err == nil {
			t.Errorf("Load accepted %q", body)
		}
	}
	var buf bytes.Buffer
	r := &Runner{S: s, Out: &buf}
	if err := r.Run(t.Context(), "nosuch"); err == nil {
		t.Error("unknown step accepted")
	}
}

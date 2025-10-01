package formatter

import (
	"strings"
	"testing"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func TestYAMLFormatter_FormatTask(t *testing.T) {
	formatter := NewYAMLFormatter()

	tests := []struct {
		name     string
		task     *types.Task
		expected string
	}{
		{
			name: "Simple task with module parameters",
			task: &types.Task{
				Name:   "Install package",
				Module: "package_enhanced",
				Args: map[string]interface{}{
					"package_name": "git",
					"state":        "present",
					"update_cache": true,
				},
			},
			expected: `- name: "Install package"
  module: "package_enhanced"
  # Module parameters
  package_name: "git"
  state: "present"
  update_cache: true`,
		},
		{
			name: "Task with module and task parameters",
			task: &types.Task{
				Name:     "Install package with conditions",
				Module:   "package_enhanced",
				When:     "ansible_os_family == 'Debian'",
				Register: "install_result",
				Tags:     []string{"packages", "essential"},
				Args: map[string]interface{}{
					"package_name": "curl",
					"state":        "present",
					"version":      "latest",
				},
			},
			expected: `- name: "Install package with conditions"
  module: "package_enhanced"
  # Module parameters
  package_name: "curl"
  state: "present"
  version: "latest"
  # Task parameters
  when: "ansible_os_family == 'Debian'"
  register: "install_result"
  tags:
    - "packages"
    - "essential"`,
		},
		{
			name: "Complex task with all parameter types",
			task: &types.Task{
				Name:         "Complex package operation",
				Module:       "package_enhanced",
				When:         "install_nginx | default(true)",
				Register:     "nginx_result",
				IgnoreErrors: false,
				Timeout:      30 * time.Second,
				Retries:      3,
				Tags:         []string{"webserver", "nginx"},
				Notify:       []string{"restart nginx", "reload systemd"},
				Args: map[string]interface{}{
					"package_name": "nginx",
					"state":        "present",
					"version":      "1.18.*",
					"update_cache": true,
					"extra_packages": []interface{}{
						"nginx-extras",
						"nginx-doc",
					},
					"repository": map[string]interface{}{
						"name": "nginx-stable",
						"url":  "http://nginx.org/packages/ubuntu/",
						"key":  "ABF5BD827BD9BF62",
					},
				},
			},
			expected: `- name: "Complex package operation"
  module: "package_enhanced"
  # Module parameters
  extra_packages:
    - "nginx-extras"
    - "nginx-doc"
  package_name: "nginx"
  repository:
    key: "ABF5BD827BD9BF62"
    name: "nginx-stable"
    url: "http://nginx.org/packages/ubuntu/"
  state: "present"
  update_cache: true
  version: "1.18.*"
  # Task parameters
  when: "install_nginx | default(true)"
  register: "nginx_result"
  timeout: "30s"
  retries: 3
  tags:
    - "webserver"
    - "nginx"
  notify:
    - "restart nginx"
    - "reload systemd"`,
		},
		{
			name: "Task with loop",
			task: &types.Task{
				Name:   "Install multiple packages",
				Module: "package_enhanced",
				Loop: &types.Loop{
					Items: []interface{}{
						map[string]interface{}{
							"name":  "vim",
							"state": "present",
						},
						map[string]interface{}{
							"name":  "emacs",
							"state": "present",
						},
					},
					Variable: "item",
				},
				Args: map[string]interface{}{
					"package_name": "{{ item.name }}",
					"state":        "{{ item.state }}",
				},
			},
			expected: `- name: "Install multiple packages"
  module: "package_enhanced"
  # Module parameters
  package_name: "{{ item.name }}"
  state: "{{ item.state }}"
  # Task parameters
  loop:
    items:
      -
        name: "vim"
        state: "present"
      -
        name: "emacs"
        state: "present"
    variable: "item"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := formatter.FormatTask(tt.task, 0)
			if err != nil {
				t.Fatalf("FormatTask() error = %v", err)
			}

			// Normalize whitespace for comparison
			expected := strings.TrimSpace(tt.expected)
			actual := strings.TrimSpace(result)

			if actual != expected {
				t.Errorf("FormatTask() result mismatch:\nExpected:\n%s\n\nActual:\n%s", expected, actual)
			}
		})
	}
}

func TestYAMLFormatter_FormatPlaybook(t *testing.T) {
	formatter := NewYAMLFormatter()

	playbook := &types.Playbook{
		Name: "Test Playbook",
		Vars: map[string]interface{}{
			"install_packages": true,
			"package_list": []interface{}{
				"git",
				"curl",
				"vim",
			},
		},
		Plays: []types.Play{
			{
				Name:  "Install packages",
				Hosts: "localhost",
				Tasks: []types.Task{
					{
						Name:   "Install git",
						Module: "package_enhanced",
						Args: map[string]interface{}{
							"package_name": "git",
							"state":        "present",
						},
						When: "install_packages",
					},
				},
			},
		},
	}

	result, err := formatter.FormatPlaybook(playbook)
	if err != nil {
		t.Fatalf("FormatPlaybook() error = %v", err)
	}

	expected := `name: "Test Playbook"

vars:
  install_packages: true
  package_list:
    - "git"
    - "curl"
    - "vim"

plays:
  - name: "Install packages"
    hosts: "localhost"

    tasks:
      - name: "Install git"
        module: "package_enhanced"
        # Module parameters
        package_name: "git"
        state: "present"
        # Task parameters
        when: "install_packages"`

	// Normalize whitespace for comparison
	expectedNorm := strings.TrimSpace(expected)
	actualNorm := strings.TrimSpace(result)

	if actualNorm != expectedNorm {
		t.Errorf("FormatPlaybook() result mismatch:\nExpected:\n%s\n\nActual:\n%s", expectedNorm, actualNorm)
	}
}

func TestYAMLFormatter_needsQuoting(t *testing.T) {
	formatter := NewYAMLFormatter()

	tests := []struct {
		input    string
		expected bool
	}{
		{"simple", false},
		{"true", true},
		{"false", true},
		{"yes", true},
		{"no", true},
		{"null", true},
		{"123", true},
		{"0644", true},
		{"1.0", true},
		{"hello world", false},
		{"hello:world", true},
		{"hello{world}", true},
		{"hello[world]", true},
		{" leading", true},
		{"trailing ", true},
		{"", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := formatter.needsQuoting(tt.input)
			if result != tt.expected {
				t.Errorf("needsQuoting(%q) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

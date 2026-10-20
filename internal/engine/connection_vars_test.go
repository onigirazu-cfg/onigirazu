package engine

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/onigirazu-cfg/onigirazu/internal/interfaces"
	"github.com/onigirazu-cfg/onigirazu/internal/inventory"
	"github.com/onigirazu-cfg/onigirazu/internal/logger"
	"github.com/onigirazu-cfg/onigirazu/internal/parser"
	"github.com/stretchr/testify/mock"

	"github.com/onigirazu-cfg/onigirazu/internal/template"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
	"github.com/stretchr/testify/require"
)

func TestResolveConnectionExtraVars(t *testing.T) {
	e, _, _, _, _, _ := createTestEngine()
	e.templateEngine = template.NewEngine()
	e.extraVars = map[string]interface{}{"flag": true}
	h := types.Host{Name: "dtl", Address: "192.0.2.10", Vars: map[string]interface{}{
		"flag": false, "ansible_ssh_common_args": "{{ '' if flag | default(false) else '-o ProxyJump=bastion' }}",
	}}
	got, err := e.resolveConnection(context.Background(), h, nil)
	require.NoError(t, err)
	require.Empty(t, got.SSHArgs)
}

func TestResolveConnection(t *testing.T) {
	tests := []struct {
		name                       string
		vars, play, extra, runtime map[string]interface{}
		host                       types.Host
		want                       types.Host
		err                        string
	}{
		{name: "inventory flag", vars: map[string]interface{}{"flag": false, "ansible_ssh_common_args": "{{ '' if flag | default(false) else '-o ProxyJump=bastion' }}"}, want: types.Host{SSHArgs: "-o ProxyJump=bastion"}},
		{name: "plain", vars: map[string]interface{}{"ansible_host": "192.0.2.10", "ansible_user": "deploy", "ansible_port": 2222, "ansible_ssh_private_key_file": "/tmp/example-key", "ansible_password": "example-password", "ansible_become_pass": "example-become"}, want: types.Host{Address: "192.0.2.10", User: "deploy", Port: 2222, KeyFile: "/tmp/example-key", Password: "example-password", BecomePassword: "example-become"}},
		{name: "host templates", vars: map[string]interface{}{"ansible_host": "{{ address }}", "address": "192.0.2.20", "ansible_user": "{{ login }}", "login": "deploy", "ansible_port": "{{ ssh_port }}", "ssh_port": 2222}, want: types.Host{Address: "192.0.2.20", User: "deploy", Port: 2222}},
		{name: "play context", vars: map[string]interface{}{"ansible_user": "{{ login }}"}, play: map[string]interface{}{"login": "play-user"}, want: types.Host{User: "play-user"}},
		{name: "extra alias beats inventory alias", vars: map[string]interface{}{"onigirazu_user": "inventory-user"}, extra: map[string]interface{}{"ansible_user": "extra-user"}, want: types.Host{User: "extra-user"}},
		{name: "chained templates", vars: map[string]interface{}{"ansible_user": "{{ login }}", "login": "{{ deploy_user }}", "deploy_user": "deploy"}, want: types.Host{User: "deploy"}},
		{name: "extra user", vars: map[string]interface{}{"ansible_user": "inventory-user"}, extra: map[string]interface{}{"ansible_user": "extra-user"}, want: types.Host{User: "extra-user"}},
		{name: "statement template", vars: map[string]interface{}{"ansible_user": "{% if flag %}deploy{% else %}fallback{% endif %}"}, play: map[string]interface{}{"flag": true}, want: types.Host{User: "deploy"}},
		{name: "plain user ignored", play: map[string]interface{}{"user": "play-user"}},
		{name: "runtime from earlier play", vars: map[string]interface{}{"ansible_user": "inventory-user"}, runtime: map[string]interface{}{"ansible_user": "runtime-user"}, want: types.Host{User: "runtime-user"}},
		{name: "first alias even empty", vars: map[string]interface{}{"onigirazu_user": "", "ansible_user": "ignored"}},
		{name: "render failure", vars: map[string]interface{}{"ansible_ssh_common_args": "{{ undefined_var.x }}"}, err: "host dtl: ansible_ssh_common_args:"},
		{name: "bad port", vars: map[string]interface{}{"ansible_port": "x"}, err: `ansible_port "x" is not a port`},
		{name: "zero port", vars: map[string]interface{}{"ansible_port": 0}, err: `ansible_port "0" is not a port`},
		{name: "strict checking disabled", vars: map[string]interface{}{"ansible_ssh_common_args": "-o StrictHostKeyChecking=no"}, want: types.Host{SSHArgs: "-o StrictHostKeyChecking=no", InsecureIgnoreHostKey: true}},
		{name: "raw args reset", host: types.Host{SSHArgs: "-o StrictHostKeyChecking=no", InsecureIgnoreHostKey: true}, vars: map[string]interface{}{"ansible_ssh_common_args": "{{ '' }}"}},
		{name: "explicit insecure kept", host: types.Host{SSHArgs: "-o StrictHostKeyChecking=no", InsecureIgnoreHostKey: true}, vars: map[string]interface{}{"ansible_ssh_common_args": "{{ '' }}", "ansible_ssh_host_key_checking": false}, want: types.Host{InsecureIgnoreHostKey: true}},
		{name: "insecure var kept", host: types.Host{SSHArgs: "-o StrictHostKeyChecking=no", InsecureIgnoreHostKey: true}, vars: map[string]interface{}{"ansible_ssh_common_args": "{{ '' }}", "insecure_ignore_host_key": true}, want: types.Host{InsecureIgnoreHostKey: true}},
		{name: "independent insecure kept", host: types.Host{InsecureIgnoreHostKey: true}, vars: map[string]interface{}{"ansible_ssh_common_args": "{{ '' }}"}, want: types.Host{InsecureIgnoreHostKey: true}},
		{name: "join args", vars: map[string]interface{}{"ansible_ssh_common_args": "  -J bastion  ", "ansible_ssh_extra_args": " -o BatchMode=yes "}, want: types.Host{SSHArgs: "-J bastion -o BatchMode=yes"}},
		{name: "local resolved", vars: map[string]interface{}{"ansible_connection": "local", "ansible_user": "{{ 'local-user' }}"}, want: types.Host{User: "local-user"}},
		{name: "password failure redacted", vars: map[string]interface{}{"ansible_password": "{{ 'secret-value' | missing_filter }}"}, err: "host dtl: ansible_password: template rendering failed"},
		{name: "become failure redacted", vars: map[string]interface{}{"ansible_become_password": "{{ 'secret-value' | missing_filter }}"}, err: "host dtl: ansible_become_password: template rendering failed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e, _, _, _, _, _ := createTestEngine()
			e.templateEngine = template.NewEngine()
			e.extraVars = tt.extra
			e.hostVars["dtl"] = tt.runtime
			e.facts["dtl"] = map[string]interface{}{"ansible_user": "fact-user", "onigirazu_host": "192.0.2.99"}
			h := tt.host
			h.Name, h.Vars = "dtl", tt.vars
			got, err := e.resolveConnection(context.Background(), h, tt.play)
			if tt.err != "" {
				require.ErrorContains(t, err, tt.err)
				require.NotContains(t, err.Error(), "secret-value")
				return
			}
			require.NoError(t, err)
			got.Name, got.Vars = "", nil
			require.Equal(t, tt.want, got)
		})
	}
}

func TestConnectionFailureExcludesOnlyFailedHost(t *testing.T) {
	e, _, calls := perHostEngine(t, types.TaskResult{Success: true})
	// Keep task rendering mocked while using the real renderer for connection variables.
	renderer := template.NewEngine()
	original := e.templateEngine
	e.templateEngine = &connectionTestTemplate{TemplateEngine: original, render: renderer.Render}
	hosts := []types.Host{
		{Name: "bad", Address: "192.0.2.10", Vars: map[string]interface{}{"ansible_ssh_common_args": "{{ undefined_var.x }}"}},
		{Name: "good", Address: "192.0.2.11"},
	}
	result, err := e.executePlayOn(context.Background(), &types.Play{Name: "connection test", Tasks: []types.Task{{Name: "continues", Module: "debug"}}}, hosts)
	require.NoError(t, err)
	require.False(t, result.Success)
	require.Len(t, *calls, 1)
	require.Equal(t, "good", (*calls)[0]["host"])
	require.Equal(t, "bad", result.Hosts[0].Host)
	require.Contains(t, result.Hosts[0].Tasks[0].Error, "connection variables:")
}

type connectionTestTemplate struct {
	interfaces.TemplateEngine
	render func(context.Context, string, map[string]interface{}) (string, error)
}

func (t *connectionTestTemplate) Render(ctx context.Context, s string, v map[string]interface{}) (string, error) {
	return t.render(ctx, s, v)
}

func TestDelegateConnectionUsesDelegateVars(t *testing.T) {
	e, config, _, inv, registry, mockTemplate := createTestEngine()
	config.On("GetDryRun").Return(false)
	mockTemplate.On("RenderTaskArgs", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
	e.templateEngine = &connectionTestTemplate{TemplateEngine: mockTemplate, render: template.NewEngine().Render}
	inv.hosts = []types.Host{{Name: "delegate", Address: "delegate", Vars: map[string]interface{}{"ansible_host": "{{ endpoint }}", "endpoint": "192.0.2.20"}}}
	inv.On("GetHosts", mock.Anything).Return(nil, nil)
	var target types.Host
	registry.On("ExecuteTask", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Run(func(args mock.Arguments) { target = args.Get(2).(types.Host) }).Return(types.TaskResult{Success: true}, nil)
	host := types.Host{Name: "source", Address: "192.0.2.10", Vars: map[string]interface{}{"endpoint": "192.0.2.30"}}
	err := e.executeTaskOnHost(context.Background(), &types.Task{Name: "delegated", Module: "command", DelegateTo: "delegate"}, &host, nil, &types.PlayResult{})
	require.NoError(t, err)
	require.Equal(t, "192.0.2.20", target.Address)
}

func TestConnectionPreparationFatal(t *testing.T) {
	for _, tt := range []struct {
		name           string
		fatal, onlyBad bool
	}{
		{name: "any errors fatal", fatal: true}, {name: "all hosts failed", onlyBad: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			e, _, calls := perHostEngine(t, types.TaskResult{Success: true})
			e.templateEngine = &connectionTestTemplate{TemplateEngine: e.templateEngine, render: template.NewEngine().Render}
			hosts := []types.Host{{Name: "bad", Address: "192.0.2.10", Vars: map[string]interface{}{"ansible_port": "not-a-port"}}}
			if !tt.onlyBad {
				hosts = append(hosts, types.Host{Name: "good", Address: "192.0.2.11"})
			}
			result, err := e.executePlayOn(context.Background(), &types.Play{Name: "connection test", AnyErrorsFatal: tt.fatal, GatherFacts: true, Tasks: []types.Task{{Name: "must not run", Module: "debug"}}}, hosts)
			require.ErrorContains(t, err, "is not a port")
			require.False(t, result.Success)
			require.Empty(t, *calls)
			require.Equal(t, "Connection variables", result.Hosts[0].Tasks[0].TaskName)
		})
	}
}

func TestResolveConnectionAbsentFields(t *testing.T) {
	e, _, _, _, _, _ := createTestEngine()
	h := types.Host{Name: "dtl", Address: "192.0.2.10", User: "deploy", Port: 2222, KeyFile: "/tmp/example-key", SSHArgs: "-J bastion", Password: "example-password", BecomePassword: "example-become"}
	got, err := e.resolveConnection(context.Background(), h, nil)
	require.NoError(t, err)
	require.Equal(t, h, got)
}

func TestConnectionIsFrozenAtPlayStart(t *testing.T) {
	e, config, _, _, registry, mockTemplate := createTestEngine()
	config.On("GetDryRun").Return(false)
	mockTemplate.On("RenderTaskArgs", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
	e.templateEngine = &connectionTestTemplate{TemplateEngine: mockTemplate, render: template.NewEngine().Render}
	var targets []string
	registry.On("ExecuteTask", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		targets = append(targets, args.Get(2).(types.Host).User)
		e.setHostVar("dtl", "ansible_user", "later-user")
	}).Return(types.TaskResult{Success: true}, nil)
	host := types.Host{Name: "dtl", Address: "192.0.2.10", Vars: map[string]interface{}{"ansible_user": "{{ login }}", "login": "initial-user"}}
	_, err := e.executePlayOn(context.Background(), &types.Play{Name: "frozen", Tasks: []types.Task{
		{Name: "first", Module: "debug", Vars: map[string]interface{}{"ansible_user": "task-user"}},
		{Name: "second", Module: "debug"},
	}}, []types.Host{host})
	require.NoError(t, err)
	require.Equal(t, []string{"initial-user", "initial-user"}, targets)
}

func TestResolveConnectionKeyHomeExpansion(t *testing.T) {
	home, err := os.UserHomeDir()
	require.NoError(t, err)
	for _, key := range []string{"~/.ssh/example-key", "{{ '~/.ssh/example-key' }}"} {
		e, _, _, _, _, _ := createTestEngine()
		e.templateEngine = template.NewEngine()
		got, err := e.resolveConnection(context.Background(), types.Host{Name: "dtl", KeyFile: filepath.Join(home, ".ssh/example-key"), Vars: map[string]interface{}{"onigirazu_ssh_private_key_file": key}}, nil)
		require.NoError(t, err)
		require.Equal(t, filepath.Join(home, ".ssh/example-key"), got.KeyFile)
	}
}

func TestConnectionLiteralHostSecretsBeatGroupSecrets(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "inventory.yml")
	require.NoError(t, os.WriteFile(path, []byte(`all:
  vars:
    ansible_password: group-password
    ansible_become_password: group-become
  hosts:
    dtl:
      ansible_host: 192.0.2.10
      ansible_password: host-password
      ansible_become_password: host-become
`), 0600))
	p := parser.NewEnhancedParser(nil, logger.New(false))
	inv, err := p.ParseInventory(context.Background(), path)
	require.NoError(t, err)
	m := inventory.NewManager(p, logger.New(false), nil)
	require.NoError(t, m.SetInventory(inv))
	hosts, err := m.GetHosts("dtl")
	require.NoError(t, err)
	require.Len(t, hosts, 1)
	for _, extra := range []bool{false, true} {
		e, _, _, _, _, _ := createTestEngine()
		if extra {
			e.extraVars = map[string]interface{}{"ansible_password": "extra-password", "ansible_become_password": "extra-become"}
		}
		got, err := e.resolveConnection(context.Background(), hosts[0], nil)
		require.NoError(t, err)
		if extra {
			require.Equal(t, "extra-password", got.Password)
			require.Equal(t, "extra-become", got.BecomePassword)
		} else {
			require.Equal(t, "host-password", got.Password)
			require.Equal(t, "host-become", got.BecomePassword)
		}
	}
}

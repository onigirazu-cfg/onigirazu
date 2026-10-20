package cli

import (
	"bytes"
	"os"
	"os/user"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestServeUnitReference(t *testing.T) {
	got, err := renderServeUnit(serveUnitOptions{
		Exec: "/usr/bin/onigirazu", GlobalConfig: "/srv/patchctl/red-patching/onigirazu.yml",
		ServeConfig: "/srv/patchctl/red-patching/serve.yml", Addr: "127.0.0.1:8086",
		User: "patchctl", Group: "patchctl", WorkingDirectory: "/srv/patchctl/red-patching",
		EnvironmentFile: "/etc/onigirazu/serve.env",
		Env:             []string{"PATH=/srv/patchctl/red-patching/.venv/bin:/usr/local/bin:/usr/bin:/bin"},
		ReadWritePaths:  []string{"/var/lib/onigirazu", "/srv/patchctl", "/var/lib/onigirazu"},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := `[Unit]
Description=onigirazu serve: fleet server
After=network-online.target
Wants=network-online.target

[Service]
User=patchctl
Group=patchctl
WorkingDirectory=/srv/patchctl/red-patching
EnvironmentFile=/etc/onigirazu/serve.env
Environment=PATH=/srv/patchctl/red-patching/.venv/bin:/usr/local/bin:/usr/bin:/bin
ExecStart=/usr/bin/onigirazu -c /srv/patchctl/red-patching/onigirazu.yml serve --addr 127.0.0.1:8086 --config-file /srv/patchctl/red-patching/serve.yml
Restart=on-failure
RestartSec=5s
NoNewPrivileges=yes
ProtectSystem=strict
ReadWritePaths=/var/lib/onigirazu /srv/patchctl
PrivateTmp=yes

[Install]
WantedBy=multi-user.target
`
	if got != want {
		t.Fatalf("unit mismatch\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestServeUnitMinimal(t *testing.T) {
	got, err := renderServeUnit(serveUnitOptions{Exec: "/usr/bin/onigirazu", ServeConfig: "/etc/onigirazu/serve.yml"})
	if err != nil {
		t.Fatal(err)
	}
	for _, omitted := range []string{"\nUser=", "\nGroup=", "\nWorkingDirectory=", "\nEnvironmentFile=", "\nEnvironment=", " -c "} {
		if strings.Contains(got, omitted) {
			t.Errorf("unexpected %q in unit", omitted)
		}
	}
	for _, required := range []string{"ExecStart=/usr/bin/onigirazu serve --config-file /etc/onigirazu/serve.yml\n", "NoNewPrivileges=yes\n", "ProtectSystem=strict\n", "ReadWritePaths=\n", "PrivateTmp=yes\n"} {
		if !strings.Contains(got, required) {
			t.Errorf("missing %q in unit", required)
		}
	}
}

func TestServeUnitQuoting(t *testing.T) {
	got, err := renderServeUnit(serveUnitOptions{
		Exec: `/opt/with space/onigirazu`, GlobalConfig: `/srv/%app/$config.yml`,
		ServeConfig: `/srv/a "quote"\file.yml`, WorkingDirectory: `/srv/my %app`,
		EnvironmentFile: `/etc/my %app.env`, Env: []string{`MESSAGE=hello "world"\ %app $HOME`, "SECOND=kept"},
		ReadWritePaths: []string{`/srv/with space/%app`, `/srv/%app`},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`ExecStart="/opt/with space/onigirazu" -c /srv/%%app/$$config.yml serve --config-file "/srv/a \"quote\"\\file.yml"`,
		`Environment="MESSAGE=hello \"world\"\\ %%app $HOME"`,
		"Environment=SECOND=kept\n",
		"WorkingDirectory=/srv/my %%app\n", "EnvironmentFile=/etc/my %%app.env\n",
		`ReadWritePaths="/srv/with space/%%app" /srv/%%app`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in unit:\n%s", want, got)
		}
	}
}

func TestServeUnitRejectsNewlines(t *testing.T) {
	for _, tc := range []struct {
		name string
		set  func(*serveUnitOptions)
	}{
		{"exec", func(o *serveUnitOptions) { o.Exec += "\nx" }},
		{"global config", func(o *serveUnitOptions) { o.GlobalConfig = "x\nx" }},
		{"serve config", func(o *serveUnitOptions) { o.ServeConfig += "\nx" }},
		{"addr", func(o *serveUnitOptions) { o.Addr = "x\nx" }},
		{"user", func(o *serveUnitOptions) { o.User = "x\nx" }},
		{"group", func(o *serveUnitOptions) { o.Group = "x\nx" }},
		{"working directory", func(o *serveUnitOptions) { o.WorkingDirectory = "x\nx" }},
		{"environment file", func(o *serveUnitOptions) { o.EnvironmentFile = "x\nx" }},
		{"env", func(o *serveUnitOptions) { o.Env = []string{"KEY=x\nx"} }},
		{"writable path", func(o *serveUnitOptions) { o.ReadWritePaths = []string{"x\nx"} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			o := serveUnitOptions{Exec: "/usr/bin/onigirazu", ServeConfig: "/etc/serve.yml"}
			tc.set(&o)
			if _, err := renderServeUnit(o); err == nil {
				t.Fatal("accepted newline")
			}
		})
	}
}

func TestServeEnvFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "serve.env")
	content := "\n # comment\n; another comment\nPLAIN=value\nSINGLE='hello world'\nDOUBLE=\"two words\"\nESC=\"a\\\"b\\\\c\\$d\\x\"\nRAW='a\\\"b'\nBARE=a\\ b\nEMPTY=\nEQUALS=a=b\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := parseServeEnvFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"PLAIN=value", "SINGLE=hello world", "DOUBLE=two words", `ESC=a"b\c$d\x`, `RAW=a\"b`, "BARE=a b", "EMPTY=", "EQUALS=a=b"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestServeEnvFileErrorsDoNotExposeValues(t *testing.T) {
	for _, line := range []string{"export TOKEN=private-value", "TOKEN-private-value", "1TOKEN=private-value", "TOKEN=\"private-value"} {
		t.Run(line, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "serve.env")
			if err := os.WriteFile(path, []byte("# first line\n"+line+"\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			_, err := parseServeEnvFile(path)
			if err == nil {
				t.Fatal("accepted invalid environment line")
			}
			if !strings.Contains(err.Error(), "2") || strings.Contains(err.Error(), "private-value") {
				t.Fatalf("error must identify line 2 without its value: %v", err)
			}
		})
	}
}

func TestServeInstallPrint(t *testing.T) {
	current, err := user.Current()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	servePath, envPath := filepath.Join(dir, "serve.yml"), filepath.Join(dir, "serve.env")
	globalPath := filepath.Join(dir, "onigirazu.yml")
	const hook = "ONIGIRAZU_INSTALL_TEST_HOOK"
	t.Setenv(hook, "")
	if err := os.WriteFile(servePath, []byte("notify:\n  - url: ${"+hook+"}\ndata_dir: results\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(envPath, []byte(hook+"=https://hooks.example.com/install-test\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(globalPath, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	oldConfig := configPath
	t.Cleanup(func() { configPath = oldConfig })
	root := NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"-c", globalPath, "serve", "install", "-f", servePath, "--user", current.Username, "--environment-file", envPath, "--print"})
	if err := root.Execute(); err != nil {
		t.Fatalf("install --print: %v\n%s", err, out.String())
	}
	got := out.String()
	for _, want := range []string{" -c " + globalPath + " serve --config-file " + servePath, "WorkingDirectory=" + dir + "\n", "EnvironmentFile=" + envPath + "\n", "User=" + current.Username + "\n", "ReadWritePaths=" + filepath.Join(dir, "results") + " " + dir + " " + current.HomeDir + "\n"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in output:\n%s", want, got)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "results")); !os.IsNotExist(err) {
		t.Fatalf("--print created data directory: %v", err)
	}
}

func TestServeInstallEnvOverrideAndDefaultHome(t *testing.T) {
	current, err := user.Current()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	servePath, envPath := filepath.Join(dir, "serve.yml"), filepath.Join(dir, "serve.env")
	const hook = "ONIGIRAZU_INSTALL_OVERRIDE_HOOK"
	t.Setenv(hook, "")
	if err := os.WriteFile(servePath, []byte("notify:\n  - url: ${"+hook+"}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(envPath, []byte(hook+"=https://hooks.example.com/from-file\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	oldConfig := configPath
	t.Cleanup(func() { configPath = oldConfig })
	configPath = ""
	cmd := newServeCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"install", "-f", servePath, "--user", current.Username, "--environment-file", envPath, "--env", hook + "=", "--print"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("environment file must override --env for validation: %v", err)
	}
	want := "ReadWritePaths=" + filepath.Join(current.HomeDir, ".onigirazu", "serve") + " " + dir + " " + current.HomeDir + "\n"
	if !strings.Contains(out.String(), want) {
		t.Fatalf("missing service user's default data directory %q:\n%s", want, out.String())
	}
}

func TestServeInstallRelativeDataDirectory(t *testing.T) {
	current, err := user.Current()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	working := filepath.Join(dir, "work")
	if err := os.Mkdir(working, 0o700); err != nil {
		t.Fatal(err)
	}
	servePath := filepath.Join(dir, "serve.yml")
	if err := os.WriteFile(servePath, []byte("data_dir: results\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	oldConfig := configPath
	t.Cleanup(func() { configPath = oldConfig })
	configPath = ""
	cmd := newServeCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"install", "-f", servePath, "--user", current.Username, "--working-directory", working, "--read-write-path", working, "--print"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	want := "ReadWritePaths=" + filepath.Join(working, "results") + " " + working + " " + current.HomeDir + "\n"
	if !strings.Contains(out.String(), want) {
		t.Fatalf("relative data_dir must use service working directory and duplicate paths must be removed; missing %q:\n%s", want, out.String())
	}
	if _, err := os.Stat(filepath.Join(working, "results")); !os.IsNotExist(err) {
		t.Fatalf("--print created data directory: %v", err)
	}
}

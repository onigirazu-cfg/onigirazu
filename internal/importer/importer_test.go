package importer

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func b64(s string) string { return base64.StdEncoding.EncodeToString([]byte(s)) }

var sample = strings.Join([]string{
	"OS\tdebian\tubuntu\t24.04",
	"SYSTEMD\tyes",
	"PKG\tnginx",
	"PKG\tcurl",
	"UNIT\tnginx.service\tenabled\tenabled",
	"UNIT\tmyapp.service\tenabled\tdisabled",
	"UNIT\tcups.service\tdisabled\tenabled",
	"UNIT\tgetty@.service\tenabled\tenabled",
	"ACTIVE\tmyapp.service",
	"PASSWD\troot:x:0:0:root:/root:/bin/bash",
	"PASSWD\tbob:x:1001:1001:Bob:/home/bob:/bin/bash",
	"GROUP\tbob:x:1001:",
	"GROUP\tdocker:x:999:bob",
	"FSTAB\t/dev/sdb1 /data ext4 defaults 0 2",
	"FSTAB\tUUID=x / ext4 defaults 0 1",
	"TZ\t/usr/share/zoneinfo/Europe/Madrid",
	"TREE\t/opt/bigapp",
	"FILE\t/etc/apt/sources.list.d/local.list\t644\troot\troot\t0\t0",
	"DATA\t" + b64("deb [trusted=yes] file:/srv/repo ./\n"),
	"DIR\t/srv/repo\t755\troot\troot\t0\t0",
	"FILE\t/srv/repo/Packages\t644\troot\troot\t0\t0",
	"DATA\t",
	"FILE\t/etc/myapp.conf\t640\troot\tUNKNOWN\t0\t4242",
	"DATA\t" + b64("port=80\n"),
	"FILE\t/etc/ssl/private/k.pem\t600\troot\troot\t0\t0",
	"DATA\t" + b64("-----BEGIN PRIVATE KEY-----\nx\n"),
	"LINK\t/etc/myapp.d\t/opt/myapp/conf",
	"END",
}, "\n")

func TestParse(t *testing.T) {
	s, err := Parse("web1", sample)
	require.NoError(t, err)
	assert.Equal(t, "debian", s.OS)
	assert.True(t, s.Systemd)
	assert.Equal(t, []string{"curl", "nginx"}, s.Packages)
	assert.Equal(t, "Europe/Madrid", s.Timezone)
	assert.Len(t, s.Files, 6)
	for _, f := range s.Files {
		if f.Path == "/etc/ssl/private/k.pem" {
			assert.Equal(t, "private key", f.Secret)
		} else {
			assert.Empty(t, f.Secret, f.Path)
		}
	}
	_, err = Parse("web1", "OS\tdebian\n")
	assert.Error(t, err, "a cut off collector run")
}

func TestGenerate(t *testing.T) {
	s, err := Parse("web1", sample)
	require.NoError(t, err)
	dir := t.TempDir()
	rep, err := Generate([]*Snapshot{s}, dir)
	require.NoError(t, err)
	assert.Equal(t, 2, rep.Counts["packages"])
	assert.Equal(t, 2, rep.Counts["services"])
	assert.Equal(t, 1, rep.Counts["users"])
	assert.Equal(t, 1, rep.Counts["mounts"])
	require.Len(t, rep.Secrets, 1)

	tasks := roleTasks(t, dir, "host_web1")
	var names []string
	for _, task := range tasks {
		names = append(names, task["name"].(string))
	}
	idx := func(name string) int {
		for i, n := range names {
			if n == name {
				return i
			}
		}
		t.Fatalf("no task %q in %v", name, names)
		return -1
	}
	// the local repository is in place before the packages
	assert.Less(t, idx("File /srv/repo/Packages"), idx("Packages"))
	assert.Less(t, idx("User bob"), idx("Packages"))
	assert.Less(t, idx("Packages"), idx("File /etc/myapp.conf"))
	assert.NotContains(t, names, "File /etc/ssl/private/k.pem")
	assert.NotContains(t, names, "Service nginx", "enabled as the preset says")
	assert.NotContains(t, names, "Mount /")

	conf := tasks[idx("File /etc/myapp.conf")]["copy"].(map[string]interface{})
	assert.Equal(t, "4242", conf["group"], "a group without a name is kept by number")
	assert.Equal(t, "0640", conf["mode"])
	svc := tasks[idx("Service myapp")]["service"].(map[string]interface{})
	assert.Equal(t, "started", svc["state"])
	assert.Equal(t, false, tasks[idx("Service cups")]["service"].(map[string]interface{})["enabled"])
	user := tasks[idx("User bob")]["user"].(map[string]interface{})
	assert.Equal(t, "docker", user["groups"])
	assert.Equal(t, 1001, user["uid"])

	content, err := os.ReadFile(filepath.Join(dir, "roles", "host_web1", "files", "etc", "myapp.conf"))
	require.NoError(t, err)
	assert.Equal(t, "port=80\n", string(content))
	site, err := os.ReadFile(filepath.Join(dir, "site.yml"))
	require.NoError(t, err)
	assert.Contains(t, string(site), "hosts: web1")
	assert.Contains(t, string(site), "name: host_web1")
	assert.Contains(t, string(site), "tasks_from: packages")

	require.NoError(t, WriteReport(filepath.Join(dir, "IMPORT_REPORT.md"), rep))
	report, _ := os.ReadFile(filepath.Join(dir, "IMPORT_REPORT.md"))
	assert.Contains(t, string(report), "/etc/ssl/private/k.pem")
	assert.Contains(t, string(report), "/opt/bigapp")
	assert.NotContains(t, string(report), "BEGIN PRIVATE KEY")
}

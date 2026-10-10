package modules

import (
	"encoding/json"
	"reflect"
	"testing"
)

const inspectNginx = `{
  "Image": "sha256:aaa",
  "Config": {"Env": ["PATH=/usr/bin", "A=1"], "Cmd": ["nginx", "-g", "daemon off;"]},
  "HostConfig": {
    "PortBindings": {"80/tcp": [{"HostIp": "", "HostPort": "8080"}]},
    "Binds": ["/srv/www:/usr/share/nginx/html:ro"],
    "RestartPolicy": {"Name": "unless-stopped"}
  },
  "NetworkSettings": {"Networks": {"bridge": {}, "web": {}}}
}`

func TestContainerDiffs(t *testing.T) {
	var inspect map[string]interface{}
	if err := json.Unmarshal([]byte(inspectNginx), &inspect); err != nil {
		t.Fatal(err)
	}
	same := map[string]interface{}{
		"image":          "nginx:1.27",
		"env":            map[string]interface{}{"A": 1},
		"ports":          []interface{}{"8080:80"},
		"volumes":        []interface{}{"/srv/www:/usr/share/nginx/html:ro"},
		"command":        `nginx -g "daemon off;"`,
		"restart_policy": "unless-stopped",
		"networks":       []interface{}{map[string]interface{}{"name": "web"}},
	}
	if d := containerDiffs(inspect, same, "sha256:aaa"); len(d) != 0 {
		t.Fatalf("same options: got diffs %v", d)
	}
	changed := map[string]interface{}{
		"image":          "nginx:1.28",
		"env":            map[string]interface{}{"A": 2},
		"ports":          []interface{}{"127.0.0.1:8080:80"},
		"volumes":        []interface{}{"/srv/www:/usr/share/nginx/html"},
		"command":        []interface{}{"nginx"},
		"restart_policy": "always",
		"networks":       []interface{}{"other"},
	}
	want := []string{"command", "env", "image", "networks", "ports", "restart_policy", "volumes"}
	if d := containerDiffs(inspect, changed, "sha256:bbb"); !reflect.DeepEqual(d, want) {
		t.Fatalf("got %v, want %v", d, want)
	}
	// comparisons: ignore
	changed["comparisons"] = map[string]interface{}{"image": "ignore", "env": "ignore"}
	want = []string{"command", "networks", "ports", "restart_policy", "volumes"}
	if d := containerDiffs(inspect, changed, "sha256:bbb"); !reflect.DeepEqual(d, want) {
		t.Fatalf("ignore: got %v, want %v", d, want)
	}
	changed["comparisons"] = map[string]interface{}{"*": "ignore"}
	if d := containerDiffs(inspect, changed, "sha256:bbb"); len(d) != 0 {
		t.Fatalf("* ignore: got diffs %v", d)
	}
	// options the task does not set are not compared
	if d := containerDiffs(inspect, map[string]interface{}{}, ""); len(d) != 0 {
		t.Fatalf("no options: got diffs %v", d)
	}
}

func TestNormalizePort(t *testing.T) {
	for in, want := range map[string]string{
		"80":                "0.0.0.0::80/tcp",
		"8080:80":           "0.0.0.0:8080:80/tcp",
		"127.0.0.1:8080:80": "127.0.0.1:8080:80/tcp",
		"53:53/udp":         "0.0.0.0:53:53/udp",
		":8080:80":          "0.0.0.0:8080:80/tcp",
	} {
		if got := normalizePort(in); got != want {
			t.Errorf("%s: got %s, want %s", in, got, want)
		}
	}
}

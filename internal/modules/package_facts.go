package modules

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// PackageFactsModule lists the installed packages as ansible_facts.packages
// (Ansible's package_facts): name -> [{name, version, arch, source}], from
// dpkg, rpm or apk (manager: auto, apt, rpm, apk)
type PackageFactsModule struct {
	*BaseModule
}

// NewPackageFactsModule creates the module
func NewPackageFactsModule() *PackageFactsModule {
	return &PackageFactsModule{BaseModule: NewBaseModule("package_facts")}
}

func (m *PackageFactsModule) GetDescription() string { return "Installed packages as facts" }

// packageFactsScript prints "name\tversion\tarch\tsource" per package
const packageFactsScript = `m=%s
if [ "$m" = auto ]; then
  if command -v dpkg-query >/dev/null 2>&1; then m=apt; elif command -v rpm >/dev/null 2>&1; then m=rpm; elif command -v apk >/dev/null 2>&1; then m=apk; else echo "no package manager found" >&2; exit 3; fi
fi
case "$m" in
  apt) dpkg-query -W -f='${binary:Package}\t${Version}\t${Architecture}\t${db:Status-Status}\n' 2>/dev/null | awk -F'\t' '$4=="installed"{print $1"\t"$2"\t"$3"\tapt"}' ;;
  rpm) rpm -qa --qf '%%{NAME}\t%%{EPOCH}:%%{VERSION}-%%{RELEASE}\t%%{ARCH}\trpm\n' 2>/dev/null | sed 's/\t(none):/\t/' ;;
  apk) apk list -I 2>/dev/null | awk '{split($1,a,"-"); n=$1; sub(/-[0-9][^-]*-r[0-9]+$/,"",n); v=substr($1,length(n)+2); print n"\t"v"\t"$2"\tapk"}' ;;
  *) echo "unsupported manager $m" >&2; exit 3 ;;
esac`

// parsePackageFacts turns the script's lines into Ansible's packages map
func parsePackageFacts(out string) map[string]interface{} {
	packages := map[string]interface{}{}
	for _, line := range strings.Split(out, "\n") {
		f := strings.Split(strings.TrimSpace(line), "\t")
		if len(f) < 4 || f[0] == "" {
			continue
		}
		name := f[0]
		if i := strings.IndexByte(name, ':'); i > 0 && f[3] == "apt" {
			name = name[:i] // binary:Package carries :arch for foreign packages
		}
		entry := map[string]interface{}{"name": name, "version": f[1], "arch": f[2], "source": f[3]}
		list, _ := packages[name].([]interface{})
		packages[name] = append(list, entry)
	}
	return packages
}

func (m *PackageFactsModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	start := time.Now()
	result := types.TaskResult{TaskName: taskName(args), Host: host.Name, Module: m.name, Timestamp: start, Success: true, Output: map[string]interface{}{}}
	manager := strings.ToLower(getStringArg(args, "manager", "auto"))
	switch manager {
	case "auto", "apt", "rpm", "apk":
	default:
		result.Success, result.Error = false, fmt.Sprintf("package_facts: manager %q (auto, apt, rpm, apk)", manager)
		return result, nil
	}
	out, err := runShellOnHost(ctx, host, args, fmt.Sprintf(packageFactsScript, shellQuote(manager)))
	if err != nil {
		result.Success, result.Error = false, fmt.Sprintf("package_facts: %v", err)
		return result, nil
	}
	packages := parsePackageFacts(out)
	result.Output["ansible_facts"] = map[string]interface{}{"packages": packages}
	result.Output["msg"] = fmt.Sprintf("%d packages", len(packages))
	result.Duration = time.Since(start)
	return result, nil
}

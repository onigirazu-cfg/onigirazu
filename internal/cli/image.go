package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// image build: a container image from a playbook. A container starts from
// the base image, the playbook runs inside it over the docker/podman
// connection, the container is committed as the new image. No Dockerfile,
// the same roles as for the servers.

type imageOptions struct {
	from, tag, runtime, name string
	cmd, entrypoint          string
	labels, extraVars        []string
	keep, push, become       bool
	limitTags                string
}

func newImageCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "image",
		Short: "Container images built by playbooks",
	}
	o := &imageOptions{}
	build := &cobra.Command{
		Use:   "build PLAYBOOK",
		Short: "Run a playbook in a container from a base image and commit it as a new image",
		Example: `  onigirazu image build app.yml --from ubuntu:24.04 --tag registry.example.com/app:1.4
  onigirazu image build app.yml --from debian:12 --tag app:dev --cmd '["/usr/sbin/nginx","-g","daemon off;"]' --runtime podman --push`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if o.from == "" || o.tag == "" {
				return fmt.Errorf("--from and --tag are required")
			}
			return runImageBuild(cmd.Context(), args[0], o, cmd.OutOrStdout())
		},
	}
	f := build.Flags()
	f.StringVar(&o.from, "from", "", "Base image (required)")
	f.StringVar(&o.tag, "tag", "", "Name and tag of the image to make (required)")
	f.StringVar(&o.runtime, "runtime", "", "docker or podman (default: whichever is installed, docker first)")
	f.StringVar(&o.name, "name", "", "Name of the build container (default: onigirazu-build-<time>)")
	f.StringVar(&o.cmd, "cmd", "", "CMD of the image, a JSON array or a shell line")
	f.StringVar(&o.entrypoint, "entrypoint", "", "ENTRYPOINT of the image, a JSON array")
	f.StringArrayVar(&o.labels, "label", nil, "Label on the image, key=value (repeatable)")
	f.StringArrayVarP(&o.extraVars, "extra-vars", "e", nil, "Extra variables, as for apply (repeatable)")
	f.StringVar(&o.limitTags, "tags", "", "Only tasks with these tags")
	f.BoolVar(&o.keep, "keep", false, "Leave the build container running after the commit (or after a failure)")
	f.BoolVar(&o.push, "push", false, "Push the image after the build")
	f.BoolVar(&o.become, "become", false, "Use become in the playbook (containers usually run as root already)")
	cmd.AddCommand(build)
	return cmd
}

// imageRuntime picks docker or podman
func imageRuntime(want string) (string, error) {
	if want != "" {
		if _, err := exec.LookPath(want); err != nil {
			return "", fmt.Errorf("%s: not found", want)
		}
		return want, nil
	}
	for _, r := range []string{"docker", "podman"} {
		if _, err := exec.LookPath(r); err == nil {
			return r, nil
		}
	}
	return "", fmt.Errorf("neither docker nor podman is installed")
}

// commitChanges are the --change arguments of commit
func commitChanges(o *imageOptions, playbook string) []string {
	var args []string
	if o.cmd != "" {
		args = append(args, "--change", "CMD "+o.cmd)
	}
	if o.entrypoint != "" {
		args = append(args, "--change", "ENTRYPOINT "+o.entrypoint)
	}
	for _, l := range o.labels {
		args = append(args, "--change", "LABEL "+l)
	}
	args = append(args, "--change", "LABEL onigirazu.playbook="+filepath.Base(playbook), "--change", "LABEL onigirazu.built="+time.Now().UTC().Format(time.RFC3339))
	return args
}

func runImageBuild(ctx context.Context, playbook string, o *imageOptions, out io.Writer) error {
	runtime, err := imageRuntime(o.runtime)
	if err != nil {
		return err
	}
	name := o.name
	if name == "" {
		name = "onigirazu-build-" + time.Now().UTC().Format("20060102-150405")
	}
	run := func(args ...string) (string, error) {
		c := exec.CommandContext(ctx, runtime, args...) // #nosec G204 -- the runtime's own commands
		b, err := c.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("%s %s: %v: %s", runtime, strings.Join(args[:min(2, len(args))], " "), err, strings.TrimSpace(string(b)))
		}
		return strings.TrimSpace(string(b)), nil
	}
	fmt.Fprintf(out, "image: starting %s from %s with %s\n", name, o.from, runtime)
	// a sleeping container; busybox/alpine have sleep, so does coreutils
	if _, err := run("run", "-d", "--name", name, "--label", "onigirazu.build=1", o.from, "sh", "-c", "trap 'exit 0' TERM; while :; do sleep 3600 & wait $!; done"); err != nil {
		return err
	}
	cleanup := func() {
		if o.keep {
			fmt.Fprintf(out, "image: build container %s kept\n", name)
			return
		}
		_, _ = run("rm", "-f", name)
	}
	// the playbook, over the container connection
	dir, err := os.MkdirTemp("", "onigirazu-image-")
	if err != nil {
		cleanup()
		return err
	}
	defer os.RemoveAll(dir)
	inv, _ := yaml.Marshal(map[string]interface{}{"all": map[string]interface{}{"hosts": map[string]interface{}{name: map[string]interface{}{
		"ansible_connection": runtime, "ansible_host": name, "onigirazu_image_build": true}}}})
	invPath := filepath.Join(dir, "inventory.yml")
	if err := os.WriteFile(invPath, inv, 0o600); err != nil {
		cleanup()
		return err
	}
	inventoryPaths = []string{invPath}
	statePath = filepath.Join(dir, ".onigirazu-state")
	args := []string{playbook}
	for _, e := range o.extraVars {
		args = append(args, "-e", e)
	}
	if o.limitTags != "" {
		args = append(args, "--tags", o.limitTags)
	}
	if o.become {
		args = append(args, "--become")
	}
	result, err := runPlaybook(args)
	if err != nil || imageFailed(result) {
		cleanup()
		if err == nil {
			err = fmt.Errorf("the playbook failed in the container")
		}
		return err
	}
	fmt.Fprintf(out, "image: committing %s as %s\n", name, o.tag)
	commit := append([]string{"commit"}, commitChanges(o, playbook)...)
	commit = append(commit, name, o.tag)
	id, err := run(commit...)
	if err != nil {
		cleanup()
		return err
	}
	cleanup()
	fmt.Fprintf(out, "image: %s (%s)\n", o.tag, strings.TrimPrefix(id, "sha256:")[:min(12, len(strings.TrimPrefix(id, "sha256:")))])
	if o.push {
		fmt.Fprintf(out, "image: pushing %s\n", o.tag)
		if _, err := run("push", o.tag); err != nil {
			return err
		}
	}
	return nil
}

func imageFailed(result *types.PlaybookResult) bool {
	if result == nil {
		return true
	}
	for _, play := range result.Plays {
		for _, h := range play.Hosts {
			for _, t := range h.Tasks {
				if t.Failed && !t.Ignored {
					return true
				}
			}
		}
	}
	return false
}

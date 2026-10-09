# Plan in pull requests

A change to playbooks or inventory shows what it would do to the hosts before it is merged: a
workflow runs `onigirazu plan` for the pull request and posts the result as a comment, replaced on
every push (the way Atlantis comments a terraform plan).

```yaml
# .github/workflows/plan.yml
name: Plan
on:
  pull_request:
    paths: ['**.yml', '**.yaml', 'inventory/**', 'roles/**', 'group_vars/**', 'host_vars/**']
permissions:
  contents: read
  pull-requests: write
jobs:
  plan:
    runs-on: [self-hosted, infra]      # a runner that reaches the hosts
    steps:
      - uses: actions/checkout@v4
      - name: onigirazu
        run: |
          curl -fsSL https://github.com/onigirazu-cfg/onigirazu/releases/latest/download/onigirazu_Linux_x86_64.tar.gz | tar -xz
          sudo mv onigirazu /usr/local/bin/
      - name: Plan
        env:
          GITHUB_TOKEN: ${{ github.token }}
          ONIGIRAZU_SSH_KNOWN_HOSTS_FILE: ${{ runner.temp }}/known_hosts
        run: onigirazu plan site.yml -i inventory/hosts.yml --github-comment
```

`--github-comment` posts the Markdown report on the pull request of the run (`GITHUB_TOKEN`,
`GITHUB_REPOSITORY`, the event payload) and replaces its own earlier comment, found by the marker
line `<!-- onigirazu-plan: <playbook> -->`; one comment per playbook. The job's exit code is the
plan's: 0 (changes or none), 1 when a task could not be checked, so a host that is down fails the
check without hiding the plan.

The comment:

> ### Plan: 2 task(s) would change on 1 of 3 host(s) — `site.yml`
>
> **web1**
>
> - `nginx.conf` (template) — diff folded under *details*
> - `Restart nginx` (service): would be restarted
>
> Unchanged: web2, db1

`--format markdown` prints the same to stdout for other systems (GitLab, Gitea: post it with their
API in the job). `drift --github-comment` and `drift --format markdown` work the same way for a
scheduled check that reports into an issue.

The runner needs SSH access to the hosts (a read-only user is enough for check mode, `--become`
for files root owns) and the repository's secrets stay in the workflow, not in the comment: plans
show diffs of files, so a template that renders a secret shows it — keep those in `no_log` tasks
or out of plan's scope with `--skip-tags`.

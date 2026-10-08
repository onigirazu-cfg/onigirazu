# E2E tests

Runs every case in `cases/` on disposable vSphere VMs cloned from the current
`[latest]` Ubuntu golden images, then deletes the VMs.

- Workflow `E2E`: on pull requests from branches of this repository (not forks,
  not Dependabot), by hand (optionally a subset of cases, or keep the VMs) and
  nightly at 05:00 UTC (all cases); only on self-hosted runners labelled `vsphere-e2e`.
  A new push to a pull request cancels its running e2e.
- A pull request runs only the cases its changes need (`select-cases.sh`): docs,
  packaging and unit tests need none; `internal/modules/<name>.go` needs the cases
  using module `<name>` (all when no case uses it); a changed case needs itself;
  anything else needs all.
- Shards: one per ~150 s of cases, at most four (`E2E_SHARD=i/n`), each on its
  own VMs, balanced by `case-seconds.tsv` (longest case first onto the least loaded
  shard; times measured in a run). Add a new slow case there.
- Base templates: `image/build.sh` (workflow `E2E base image`: nightly at 03:30 UTC,
  on changes to `image/`, `images.sh`, `setup-lib.sh`, and by hand) clones each golden
  `[latest]` image, preinstalls what the cases install (`image/prepare.sh`: packages,
  Docker and the container images), seals it like the golden image (`image/seal.sh`)
  and keeps it as `e2e-base-<os>-<golden item>-<time>` in the e2e folder (two per OS),
  a template with a snapshot `base`: test VMs are linked clones of it (a delta disk,
  ~50 s for a pair). The golden images (nightly run) are cloned in full. The vCenter
  role needs `VirtualMachine.State.CreateSnapshot` for the snapshot.
  Pull requests and manual runs clone the base of the current golden item when there
  is one (`E2E_BASE=1`), otherwise the golden image; the nightly run always tests the
  golden images. A new install in a case's `setup.sh` belongs in `prepare.sh` too.
- VMs are named `tmp-e2e-onigirazu-<run>-<os>`, live in a dedicated folder and
  carry a "TEMPORARY" note with the run link and expiry time.
- `janitor.sh` runs hourly and deletes e2e VMs older than 3 hours from that
  folder, covering cancelled runs. `purge_vms` removes all of them at once;
  VMs of runs still in progress are kept.
- Access: each run generates an SSH key and passes it as
  `guestinfo.e2e_authorized_key`, the host name as `guestinfo.e2e_hostname`; the
  image's first-boot unit creates user `e2e` and sets the name (no guest
  customization). The base images bring their own DHCP netplan.
- Cleanup powers the run's VMs off hard before `terraform destroy` (no clean guest
  shutdown to wait for).

## A case

`cases/<NN-name>/`:
- `playbook.yml` — applied to all VMs (`hosts: all`);
- `setup.sh` — optional; runs on each VM as root before the first apply;
- `verify.sh` — runs on each VM as root after the apply and must exit 0;
- `verify-local.sh` — optional; runs on the runner in the case directory with `HOST` set
  (for results that land on the control machine, e.g. fetch);
- `EXPECT_FAIL` — optional; the apply must fail (no verify, no second apply);
- `NOT_IDEMPOTENT` — optional; skips the second apply that must change nothing.
- `EXPECTED_FAILED_TASKS` — optional; names of tasks that fail on purpose (handled by
  `rescue` or `ignore_errors`), one per line.

With `keep_vms` the run key and inventory stay on the runner in
`~/.cache/onigirazu-e2e/<run>`.

The playbook runs from a copy of its case directory, so relative paths work.

## Configuration

Repository secrets: `VSPHERE_SERVER`, `VSPHERE_USER`, `VSPHERE_PASSWORD`,
`E2E_DATACENTER`, `E2E_CLUSTER`, `E2E_HOST`, `E2E_DATASTORE`, `E2E_NETWORK`,
`E2E_FOLDER`, `E2E_LIBRARY`. Secrets rather than variables: Actions logs of a
public repository are public.

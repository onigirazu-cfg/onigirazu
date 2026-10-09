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
- `verify-local.sh` — optional; runs on the runner in the case directory with `HOST`, `HOST_IP`, `KEY` set
  (for results that land on the control machine, e.g. fetch); its `note: ...` lines are
  shown in the log when it passes;
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

The hourly janitor removes the VMs of finished runs (a cancelled run gets no cleanup) at once, VMs kept
with `keep_vms` and VMs it cannot match to a run after 3 h.

When a run fails, the log shows each VM's power state, guest state, guest IP, boot time and last vCenter
events, and the console screenshots are uploaded as the `e2e-diag-<shard>` artifact (7 days).

Before the hard power-off each VM releases its DHCP lease (`dhcp-release.sh`): the network's pool is
small and a powered-off VM keeps its lease until it expires.

### DHCP leases

The e2e network's DHCP pool is small and the VMs are powered off hard, so nothing gives their
leases back. `e2e/dhcp-leases.sh` removes them on the MikroTik through its REST API: a run removes
the leases of its own VMs by MAC after destroying them (e2e, bench, image build), and the janitor
sweeps dynamic leases of VMware MACs that belong to no VM in vCenter. Without the secrets the
scripts say so and the leases are left to expire.

Secrets: `MIKROTIK_API_URL` (`https://<router>`), `MIKROTIK_API_USER`, `MIKROTIK_API_PASSWORD`,
`MIKROTIK_API_INSECURE` (`1` for a self-signed certificate). On the router, a user that may only
come from the runner (`write` in RouterOS covers everything, so the address restriction matters):

```
/user group add name=dhcp-leases policy=read,write,api,rest-api,!local,!telnet,!ssh,!ftp,!reboot,!policy,!test,!winbox,!password,!web,!sniff,!sensitive,!romon
/user add name=onigirazu-e2e group=dhcp-leases address=<runner IP>/32 password=<from the password manager>
/ip service enable www-ssl
```

### Pre-warmed VMs

`e2e/pool.sh` keeps `POOL_SIZE` (2) linked clones per image key powered on and waiting
(`pool-e2e-<id>-<key>`, tagged with their template). A run claims one per key by renaming it to its
own name — atomic in vCenter, so parallel shards never take the same VM — and sets its key through
guestinfo; the base image's `e2e-access-refresh` timer applies it within seconds. What the pool lacks
is created with terraform as before. The pool is refilled after every run and by the janitor; the
image build purges VMs of the previous template. `E2E_POOL=0` skips the pool; the nightly run on
golden images never uses it. Pool VMs hold DHCP leases while they wait.

